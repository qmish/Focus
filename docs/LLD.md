# LLD (Low-Level Design)

**Версия:** 1.0  
**Дата:** 27 марта 2026 г.  
**Статус:** Актуально для текущего `main`

---

## 1. Структура backend (`API_Go`)

- `cmd/server` — composition root: DI, роутинг, middleware, запуск background workers.
- `internal/api/handlers` — HTTP-слой (валидация входа/выхода, коды ответов, audit hooks).
- `internal/repository` — доступ к БД (GORM), инкапсуляция SQL/транзакций.
- `internal/auth` — OIDC/session JWT, ABAC/RBAC проверки, session revocation.
- `internal/websocket` — hub, доставка realtime-событий, контроль доступа в комнаты.
- `internal/jitsi` — генерация токенов Jitsi и ссылки на конференции.
- `internal/exchange` — on-prem EWS клиент, sync worker, календарные модели.
- `internal/webhooks` — inbound/outbound webhook обработка, подписи, идемпотентность.
- `internal/bots` — BotEngine, регистрация команд, доставка ответа в чат.

---

## 2. HTTP слой и контракты

### 2.1 Routing

- Public: `/health`, `/ready`, `/openapi.yaml`, `/swagger/*`, `/api/v1/auth/*`, `/api/v1/branding/jitsi`, `/api/v1/webhooks/jitsi`.
- Protected (`authMiddleware`): `/api/v1/rooms`, `/api/v1/messages`, `/api/v1/files`, `/api/v1/calendar/*`, `/api/v1/admin/*`.
- Calendar endpoints включаются только при доступном `calendarService` (EWS клиент успешно инициализирован).

### 2.2 Handler conventions

- Валидация request body/query в handler.
- Ошибки в виде `http.Error(...)` + корректный HTTP status.
- Извлечение identity из `auth.GetUserClaimsFromContext`.
- Опциональная запись аудита:
  - auth: `AuthAuditRepository`
  - calendar: `CalendarAuditRepository`
- Для важных операций используются идемпотентность/дедупликация:
  - inbound Jitsi webhook: `X-Idempotency-Key` + hash fallback
  - calendar create: `Idempotency-Key`

---

## 3. Календарный контур (EWS)

### 3.1 Основные сущности

- `internal/exchange/ews.go`:
  - интерфейс `CalendarService`
  - клиент `EWSClient` (`GetEvents`, `CreateEvent`, `GetEvent`, `UpdateEvent`, `DeleteEvent`)
- `internal/models/meeting_link.go`:
  - связка `room_id <-> exchange_event_id`
- `internal/models/calendar_idempotency_key.go`:
  - хранение завершённых `POST /calendar/events` по ключу

### 3.2 Репозитории

- `MeetingLinkRepository`:
  - upsert/get/update mapping состояния встречи.
- `CalendarIdempotencyRepository`:
  - pending/completed состояние idempotency ключа и cached response payload.

### 3.3 Auth modes для EWS

- `basic`
- `ntlm`
- `kerberos` (SPNEGO, keytab/password, `krb5.conf`)

---

## 4. Sync worker

`internal/exchange/sync_worker.go`:

- interval polling (по `EXCHANGE_SYNC_INTERVAL`)
- окно синхронизации (`lookback`, `lookahead`)
- логика:
  1. получить активных пользователей
  2. для каждого пользователя загрузить EWS события в окне
  3. upsert в `meeting_links`
  4. отсутствующие в EWS пометить как `cancelled`

---

## 5. Webhooks (inbound/outbound)

### 5.1 Inbound Jitsi

- Проверка подписи (`X-Jitsi-Signature`, поддержка base64/hex).
- Идемпотентность через `IncomingEventStore`.
- Нормализация событий:
  - `conference.created`
  - `conference.ended`
  - `participant.joined`
  - `participant.left`

### 5.2 Outbound custom webhooks

- `WebhookDispatcher`:
  - lookup активных подписок по типу события
  - HMAC подпись (`X-Webhook-Signature`)
  - retry/backoff
  - запись delivery результата

---

## 6. BotEngine

### 6.1 Контракт

- Точка входа: `HandleMessage(ctx, roomID, userID, content)`.
- Команда определяется по префиксу `/`.
- Встроенные команды:
  - `/create`, `/schedule`
  - `/help`, `/status`
  - `/members`, `/whoami`, `/dice`, `/find`

### 6.2 Delivery path

1. Парсинг и rate-limit per-user.
2. Проверка доступа к комнате.
3. Вызов зарегистрированного handler.
4. Сохранение сообщения от bot-user в БД.
5. Broadcast в websocket hub.
6. Запись command event в `bot_command_events`.

---

## 7. Frontend интеграция (кратко)

- API layer: `frontend/src/lib/apiClient*`.
- Основная бизнес-страница: `frontend/src/pages/MessengerPage.tsx`.
- Realtime: websocket + optimistic UI.
- Встречи:
  - fetch `/api/v1/calendar/events`
  - create `/api/v1/calendar/events` с `Idempotency-Key`

---

## 8. Точки расширения

- Добавление нового внешнего календаря: реализовать `CalendarService`.
- Добавление новых bot-команд: регистрация в `registerBuiltinBots`/custom handler map.
- Добавление новых webhook-source: расширить `WebhookType` + store/dispatcher contracts.
