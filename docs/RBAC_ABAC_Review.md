# RBAC/ABAC Review (AD roles mapping)

Дата: 25 марта 2026 г.

## Цель проверки

Проверить, как роли из claims применяются в API, и зафиксировать текущие ограничения по ABAC.

## Текущее состояние

- В session JWT используются роли пользователя (`roles` claims).
- На уровне маршрутов `/api/v1/admin/*` используется централизованный `RequireAccess` (roles/scopes).
- В admin handlers сохранены локальные role checks как дополнительная защита (defense-in-depth).
- Для обычных user-сценариев применяется проверка аутентификации и контекстных claims.
- Для критичных admin-операций включен ABAC слой:
  - `user.ban`,
  - `user.unban`,
  - `conference.end`.

## Матрица (минимум)

- `admin`:
  - доступ к `admin users/stats/conferences/webhooks/bots`.
- `user`:
  - доступ к room/message/calendar пользовательским операциям.
- `moderator`:
  - отдельные room-level права на уровне бизнес-логики комнат.

## Вывод

- RBAC-проверки в API присутствуют и применяются централизованно.
- Внедрен базовый ABAC policy engine (resource/action/context) для критичных операций.
- ABAC внедрен итеративно и требует дальнейшего расширения матрицы действий/ресурсов.

## Следующие шаги

1. Расширить ABAC policy matrix на webhook/bot/admin write-операции.
2. Добавить атрибуты ресурсов (owner/tenant/sensitivity) в policy evaluation.
3. Синхронизировать AD group mapping -> roles/scopes -> ABAC attributes в едином governance-процессе.
