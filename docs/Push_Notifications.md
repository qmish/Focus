# Push-уведомления

Документ описывает архитектуру push-инфраструктуры Focus: backend, frontend
и подключение к мобильным провайдерам.

## Поддерживаемые платформы

| Платформа | Канал | Статус |
|-----------|-------|--------|
| Web (PWA в браузере) | Web Push (VAPID) | ✅ работает |
| Android (Tauri Mobile) | Web Push в WebView | ✅ работает |
| Android (нативный FCM) | FCM HTTP v1 | 🔧 каркас (`internal/push/fcm.go`) |
| iOS (PWA через Safari) | Web Push (iOS 16.4+) | ✅ работает |
| iOS (нативный APNs) | APNs HTTP/2 | 🔧 каркас (`internal/push/apns.go`) |

## Архитектура backend

```
HTTP API
└─ /api/v1/push/vapid-public-key  → handlers.PushHandler
   /api/v1/push/register           → upsert PushToken
   /api/v1/push/unregister         → удаление
                              │
                              ▼
                  repository.PushTokenRepository
                              │
                              ▼
                          PushToken (DB)
                              │
       ┌──────────────────────┼──────────────────────┐
       ▼                      ▼                      ▼
   WebPushSender         FCMSender              APNSSender
   (libwebpush)          (заглушка)             (заглушка)
       │                      │                      │
       └──────────────┬───────┴──────────────────────┘
                      ▼
                push.Service.NotifyUsers()
                      │
                      ▼
        вызывается из MessageHandler.notifyOffline
        (после CreateMessage):
          • офлайн-участники комнаты → новое сообщение
          • упомянутые пользователи  → mention-push (даже онлайн)
        Gone-подписки (HTTP 410) автоматически удаляются.
```

## Конфигурация

| ENV | Назначение | Дефолт |
|-----|------------|--------|
| `PUSH_ENABLED`        | Главный включатель push-сервиса | `false` |
| `VAPID_PUBLIC_KEY`    | base64url ECDSA P-256 public  | — |
| `VAPID_PRIVATE_KEY`   | base64url ECDSA P-256 private | — |
| `VAPID_SUBJECT`       | mailto:… для VAPID JWT        | `mailto:admin@focus.local` |
| `PUSH_WEBPUSH_TTL`    | TTL push-сообщения (Go duration) | `24h` |
| `PUSH_WEBPUSH_TIMEOUT`| HTTP-таймаут push-эндпоинта   | `10s` |
| `PUSH_FCM_ENABLED`    | Включить FCM-канал            | `false` |
| `PUSH_APNS_ENABLED`   | Включить APNs-канал           | `false` |

### Генерация VAPID

В репозитории есть утилита:

```bash
cd API_Go
go run ./cmd/vapidgen
# VAPID_PUBLIC_KEY=BMu...
# VAPID_PRIVATE_KEY=igL...
```

Положить ключи в Kubernetes secret `focus-secrets` под именами
`vapid-public-key` / `vapid-private-key`. Пример:

```bash
kubectl -n messenger-stage patch secret focus-secrets \
  --type merge \
  -p="$(jq -n --arg pub "$(echo -n "$VAPID_PUBLIC_KEY" | base64 -w0)" \
              --arg priv "$(echo -n "$VAPID_PRIVATE_KEY" | base64 -w0)" \
              '{data:{"vapid-public-key":$pub,"vapid-private-key":$priv}}')"
```

После этого Argo CD/rolling-update перекатит api-go.

## Frontend

`frontend/src/lib/pushSubscribe.ts`:
* `isPushSupported()` — проверка возможностей браузера/WebView;
* `subscribePush({ skipIfAlreadyRegistered: true })` — идемпотентная
  подписка: получает VAPID public, регистрирует service worker,
  запрашивает разрешение, создаёт `PushSubscription` и шлёт на
  `POST /api/v1/push/register`;
* `unsubscribePush()` — отписка локально + `POST /api/v1/push/unregister`.

Service worker — `frontend/src/sw-push.ts`:
* регистрируется при загрузке `MessengerPage`;
* обрабатывает `push` event и показывает Notification;
* `notificationclick` — открывает или фокусирует существующую вкладку с URL
  из payload.

## Модель данных

```sql
CREATE TABLE push_tokens (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform     VARCHAR(16) NOT NULL,  -- 'web' | 'fcm' | 'apns'
    endpoint     TEXT NOT NULL UNIQUE,
    p256dh_key   TEXT,                  -- только для web
    auth_key     TEXT,                  -- только для web
    user_agent   TEXT,
    locale       VARCHAR(16),
    last_used_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_push_tokens_user ON push_tokens(user_id);
```

При повторной регистрации того же endpoint — обновляются keys и user_id
(сценарий «один и тот же браузер у двух пользователей»).

## Триггеры push

`MessageHandler.notifyOffline()` запускается в фоне после `CreateMessage`:

1. Загружает участников комнаты (`RoomRepository.ListParticipants`).
2. Делит на:
   * **mentions** (из `msg.Metadata.Mentions`, кроме автора) — push с тегом
     `mention-<msgID>` и заголовком «<автор> упомянул вас в <комнате>».
   * **offline** — те, кого нет в `wsHub.IsUserOnline()` и кто не в mentions.
3. Для каждой группы вызывает `pushService.NotifyUsers()`. Под капотом —
   параллельная отправка всем токенам пользователя; ошибки HTTP 404/410
   возвращают `SendError{IsGone: true}` и токен удаляется автоматически.

## Превью текста уведомления

В `messageHandler.go::notifyOffline` для разных типов сообщений готовится
короткий preview:

| Тип сообщения      | Тело уведомления |
|--------------------|------------------|
| `text`             | первые ~100 символов |
| `image`            | 📷 Изображение |
| `file`             | 📎 Файл |
| `voice`            | 🎤 Голосовое сообщение |
| `video`            | 🎬 Видео |
| `system`           | системное сообщение |

## Что осталось

* Реализовать FCM и APNs провайдеры (заменить заглушки) — для нативных
  Tauri-приложений.
* Rate limiting на пользователя (≤ 1 push в 5 секунд) — пока рассчитываем
  на естественную дедупликацию через `tag`.
* Метрики Prometheus: `push_sent_total{platform,status}`.
