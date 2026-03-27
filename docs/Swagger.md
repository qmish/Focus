# Swagger / OpenAPI

**Версия:** 1.0  
**Дата:** 27 марта 2026 г.

---

## 1. Что реализовано

- OpenAPI спецификация: `API_Go/docs/openapi.yaml`
- Swagger UI в backend:
  - `GET /swagger/index.html`
  - `GET /openapi.yaml`

Swagger UI подключён в `cmd/server/main.go`.

---

## 2. Что описывает спецификация

- Health/readiness endpoints
- Auth endpoints (OIDC + local auth)
- Rooms/messages/files
- Calendar CRUD (включая `Idempotency-Key`)
- Webhook endpoint (`X-Jitsi-Signature`)
- Admin endpoints (расширенный набор):
  - users CRUD + roles + ban/unban
  - invites API
  - bot settings API
  - Exchange settings + test-connection

---

## 3. Security schemes

- `bearerAuth` — session JWT (`Authorization: Bearer`)
- `webhookSignature` — `X-Jitsi-Signature` для inbound webhook

---

## 4. Процесс обновления

1. Изменить/добавить endpoint в backend.
2. Обновить `API_Go/docs/openapi.yaml`.
3. Проверить запуск backend и открытие `/swagger/index.html`.
4. Проверить примерный сценарий через Swagger UI и smoke-тесты.

---

## 5. Примечание

Спецификация хранится как исходный `yaml`, чтобы избежать рассинхронизации между runtime-роутами и автогенерацией.
