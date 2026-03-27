# Webhooks

**Версия:** 1.0  
**Дата:** 27 марта 2026 г.

---

## 1. Контуры webhook

- **Inbound**: `POST /api/v1/webhooks/jitsi` — события конференций из Jitsi.
- **Outbound**: доставка событий Focus во внешние системы через `WebhookDispatcher`.

---

## 2. Inbound Jitsi webhook

### 2.1 Безопасность

- Заголовок подписи: `X-Jitsi-Signature`.
- Поддерживаемые форматы подписи:
  - base64 HMAC-SHA256
  - hex (`sha256=` префикс поддерживается)
- Идемпотентность:
  - `X-Idempotency-Key`, либо fallback на hash payload.

### 2.2 Обрабатываемые события

- `conference.created`
- `conference.ended`
- `participant.joined`
- `participant.left`

### 2.3 Эффекты

- обновление активности комнаты
- добавление/удаление участника комнаты (если есть user_id в payload)
- запись inbound event в store для дедупликации и трассировки

---

## 3. Outbound webhook dispatch

`WebhookDispatcher`:

1. Получает активные подписки по `eventType`.
2. Формирует JSON payload.
3. Добавляет подпись `X-Webhook-Signature` + timestamp/event headers.
4. Отправляет HTTP POST.
5. При ошибках применяет retry/backoff.
6. Записывает delivery result (успех/ошибка/dead-letter).

---

## 4. Операционные заметки

- Следите за ростом таблиц inbound/delivery (ротация/TTL/архив).
- При пиковых нагрузках увеличивайте timeout/retry и выносите dispatch в очередь.
- Для production используйте отдельные webhook secrets per consumer.
