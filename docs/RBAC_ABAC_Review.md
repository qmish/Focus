# RBAC/ABAC Review (AD roles mapping)

Дата: 25 марта 2026 г.

## Цель проверки

Проверить, как роли из claims применяются в API, и зафиксировать текущие ограничения по ABAC.

## Текущее состояние

- В session JWT используются роли пользователя (`roles` claims).
- На уровне маршрутов `/api/v1/admin/*` добавлен централизованный `RequireRole("admin")`.
- В admin handlers сохранены локальные role checks как дополнительная защита (defense-in-depth).
- Для обычных user-сценариев применяется проверка аутентификации и контекстных claims.

## Матрица (минимум)

- `admin`:
  - доступ к `admin users/stats/conferences/webhooks/bots`.
- `user`:
  - доступ к room/message/calendar пользовательским операциям.
- `moderator`:
  - отдельные room-level права на уровне бизнес-логики комнат.

## Вывод

- RBAC-проверки в API присутствуют и применяются централизованно для admin маршрутов.
- Полноценный ABAC (policy engine на основе атрибутов ресурса/контекста) еще не внедрен.

## Следующие шаги

1. Формализовать ABAC policy matrix (ресурс, действие, субъект, условия).
2. Добавить policy-evaluation слой (middleware/service) и покрыть тестами.
3. Синхронизировать AD group mapping -> roles/scopes -> ABAC attributes.
