# Анализ проекта Focus (актуализирован)

**Дата анализа:** 25 марта 2026 г.  
**Статус:** 🟡 В разработке (последний релиз: `v0.5.19`)

---

## Фактическое состояние

### Backend (Go)

- Реализованы auth/admin/rooms/messages/calendar/webhooks/bots.
- WebSocket защищен и проверяет room-level access.
- Введена наблюдаемость webhook и bot ошибок через admin endpoints.
- Базовые unit/integration тесты проходят (`go test ./...`).

### Frontend (messenger)

- Реальная интеграция с API по комнатам и сообщениям.
- Единый API client + обработка ошибок/loading/retry.
- Realtime UX на WebSocket: subscribe, live updates, reconnect + token refresh.

### Frontend-admin

- Реальные users/stats/ban-unban и управление конференциями.
- Добавлен раздел наблюдаемости webhook/bot ошибок.
- Ошибки операций отображаются в UI и store-state.

### DevOps / локальный контур

- `docker-compose` расширен до полного dev-стека (`api`, `frontend`, `frontend-admin`, `postgres`, `redis`, `keycloak`).
- Введены единые env-конвенции (`.env.example`) и one-command startup (`scripts/dev-up.ps1`).
- Добавлены CI quality/security gates + e2e/load smoke и отдельный pipeline для `jitsi-fork`.

---

## Что еще не завершено

### Ключевые незакрытые области

- `Этап 1`: корпоративная схема AD/Keycloak/Exchange (identity brokering, policy mapping, security hardening).
- `Этап 4`: форк `jitsi-meet-master` и бренд-кастомизация UI на основе `pics`.
- `Этап 6.2`: актуализация k8s stage/prod манифестов (ingress/TLS/policies/secrets/rotation).
- `Этап 7`: полноценные e2e/load/security/UAT/go-live контуры.

### По документации

- Источник правды по текущему execution-backlog: `docs/Roadmap_v2.md`.
- `docs/Roadmap.md` рассматривается как исторический план v1 и не отражает текущий прогресс.

---

## Риски

- Зависимость от корпоративных AD/Exchange политик и сроков выдачи доступов.
- Сложность сопровождения форка `jitsi-meet-master` при регулярных upstream-обновлениях.
- Требуется конвертация `pics/hdphoto1.wdp` в web-формат (`png`/`webp`) для production UI.
