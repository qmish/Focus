# Security Review: auth / websocket / webhook / token flows

Дата: 25 марта 2026 г.

## Объем review

- API auth flow (`/api/v1/auth/*` + middleware).
- WebSocket auth (`/api/v1/ws`).
- Inbound webhook signature validation.
- Session/Jitsi token secret separation.

## Что проверено

- Session JWT для API и WebSocket подписывается отдельным `SESSION_SECRET`.
- `SESSION_SECRET` и `JITSI_APP_SECRET` валидируются на различие на старте сервиса.
- Для non-development окружений запрещены слабые значения `SESSION_SECRET`.
- WebSocket возвращает `token_expired` / `session_revoked` при ошибках токена.
- Inbound webhook использует подпись `JITSI_APP_SECRET`.

## Автоматизация

- Добавлены unit tests для security-валидации конфигурации:
  - `API_Go/internal/config/config_test.go`
- CI smoke-шаги запуска API теперь передают отдельные секреты:
  - `SESSION_SECRET=ci-session-secret`
  - `JITSI_APP_SECRET=ci-jitsi-secret`

## Findings

1. Ранее session токены API/WS использовали `JITSI_APP_SECRET` (риск key reuse).
2. Исправлено: API/WS переведены на `SESSION_SECRET`.

## Остаточные риски

- RBAC/ABAC mapping по AD группам еще не завершен (roadmap этап 1 / 7.2).
- Полный цикл внешнего pentest/ZAP scan требуется в отдельном прогоне stage/prod.
