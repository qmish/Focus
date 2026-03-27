# Release Notes

## v0.6.0 — Документация + Swagger + EWS hardening (27 марта 2026)

**Последний релиз** ✅

### 🎯 Цель
Привести документацию к фактической кодовой базе (EWS-only, sync worker, Kerberos, идемпотентность) и добавить встроенный Swagger/OpenAPI для API.

### ✨ Новое
- **Swagger/OpenAPI**
  - `API_Go/docs/openapi.yaml` — базовая OpenAPI 3.0 спецификация.
  - `GET /openapi.yaml` — отдача спецификации из backend.
  - `GET /swagger/index.html` — Swagger UI.
  - `API_Go/cmd/server/main.go` — подключение swagger endpoints.
  - `API_Go/Dockerfile` — включение `docs/openapi.yaml` в runtime image.

- **Документация проекта**
  - Добавлены новые документы:
    - `docs/LLD.md`
    - `docs/Frontend.md`
    - `docs/Bots.md`
    - `docs/Webhooks.md`
    - `docs/Swagger.md`
    - `docs/Exchange_OnPrem_EWS.md`
  - Обновлены:
    - `README.md`
    - `docs/README.md`
    - `docs/Architecture.md`
    - `docs/DataFlow.md`
    - `docs/Database.md`
    - `docs/Integration.md`

### 🔐 Технические акценты
- Зафиксирован контур Exchange как **on-prem EWS/OWA**.
- Документированы:
  - `meeting_links` и `calendar_idempotency_keys`
  - sync worker polling-процесс
  - Kerberos mode и k8s-манифест `k8s/exchange-kerberos-example.yaml`.

### 📊 Проверка
- `go test ./...` — успешно.
- `npm run build` (frontend) — успешно.

---

## v0.5.0 — Интеграция с MS Exchange Calendar (24 марта 2026)

**Последний релиз** ✅

### 🎯 Цель
Интеграция с календарями MS Exchange для синхронизации встреч.

### ✨ Новое
- **Microsoft Graph API Integration**
  - `internal/exchange/graph.go` — Graph API клиент
  - `CreateEvent()` — создание встреч в Exchange
  - `GetEvents()` — получение событий календаря
  - `UpdateEvent()` / `DeleteEvent()` — управление событиями

- **Calendar REST API**
  - `GET /api/v1/calendar/events` — список событий
  - `POST /api/v1/calendar/events` — создание встречи с Jitsi комнатой
  - `PUT /api/v1/calendar/events/:id` — обновление
  - `DELETE /api/v1/calendar/events/:id` — отмена

- **Функционал**
  - Автоматическое создание Jitsi комнаты при создании встречи
  - Генерация уникального room name
  - Сохранение связи meeting ↔ room
  - Вставка Jitsi URL в описание события
  - Отправка приглашений участникам через Exchange

### 📊 Тесты
- 86 тестов (все проходят)
- 14 тестов для exchange модуля

### 📦 Зависимости
- `github.com/microsoftgraph/msgraph-sdk-go` v1.96.0
- `github.com/Azure/azure-sdk-for-go/sdk/azidentity` v1.13.1

### 🔗 Ссылки
- Релиз: https://github.com/qmish/Focus/releases/tag/v0.5.0
- Документация: `docs/Integration.md`

---

## v0.4.0 — React фронтенд MVP (24 марта 2026)

### 🎯 Цель
Полноценный React фронтенд с интеграцией Keycloak и Jitsi.

### ✨ Новое
- **Frontend App**
  - LoginPage — вход через Keycloak
  - RoomsPage — список комнат, создание/удаление
  - RoomPage — чат + видеозвонок
  - ProfilePage — профиль пользователя

- **Компоненты**
  - Layout — sidebar навигация
  - JitsiMeeting — @jitsi/react-sdk интеграция

- **Store**
  - authStore — Keycloak аутентификация
  - roomsStore — управление комнатами

### 📊 Тесты
- 111 тестов (11 frontend)

---

## v0.3.0 — WebSocket и Real-time (24 марта 2026)

### 🎯 Цель
Real-time сообщения через WebSocket.

### ✨ Новое
- **WebSocket Hub**
  - Подписка/отписка от комнат
  - Рассылка сообщений
  - Typing indicators
  - События: user_joined, user_left

- **Message Repository**
  - Полный CRUD для сообщений
  - Реакции на сообщения
  - Поиск по содержимому

- **Join Room Handler**
  - `POST /api/v1/rooms/:id/join`
  - Генерация Jitsi JWT
  - Проверка прав доступа

### 📊 Тесты
- 111 тестов (все проходят)

---

## v0.2.0 — Keycloak OIDC и REST API (24 марта 2026)

### 🎯 Цель
Аутентификация через Keycloak и полный REST API.

### ✨ Новое
- **OIDC Authentication**
  - `internal/auth/oidc.go` — Keycloak OIDC провайдер
  - Login/Callback handlers
  - Session JWT генерация
  - Auth middleware

- **REST API**
  - Rooms: CRUD endpoints
  - Messages: CRUD endpoints
  - Auth: login, callback, refresh, logout, me

### 📊 Тесты
- 100+ тестов (13 auth)

---

## v0.1.0 — Базовая структура (24 марта 2026)

### 🎯 Цель
Инициализация проекта и документация.

### ✨ Новое
- **Документация (11 файлов)**
  - Architecture.md — C4 модель
  - HLD.md — High-Level Design
  - Infrastructure.md — Kubernetes развёртывание
  - DataFlow.md — DFD диаграммы
  - NetworkTopology.md — сетевая топология
  - Security.md — модель угроз
  - API.md — REST API спецификация
  - Database.md — ER-диаграмма
  - Integration.md — интеграции
  - Roadmap.md — план реализации

- **Структура проекта**
  - API_Go/ — Go бэкенд
  - docker-compose.yml — локальное окружение
  - .github/workflows/ci-cd.yml — CI/CD pipeline

- **Модели данных**
  - User, Room, Message, RoomParticipant
  - GORM миграции
  - Репозитории

### 📊 Тесты
- 39 тестов (27 models + 12 jitsi)

---

## 📈 Итоговая статистика проекта

| Метрика | Значение |
|---------|----------|
| **Релизов** | 5 |
| **Недель разработки** | 10 |
| **Файлов** | 75+ |
| **Строк кода** | 8,000+ |
| **Тестов** | 86 |
| **Документов** | 11 |

### Прогресс по этапам

| Этап | Статус | Релиз |
|------|--------|-------|
| Этап 0: Инициализация | ✅ 100% | v0.1.0 |
| Этап 1: Инфраструктура | ✅ 100% | v0.1.0 |
| Этап 2: Аутентификация и API | ✅ 100% | v0.2.0, v0.3.0 |
| Этап 3: Фронтенд | ✅ 100% | v0.4.0 |
| Этап 4: Интеграция с MS Exchange | ✅ 100% | v0.5.0 |
| Этап 5: Вебхуки и чат-боты | ⏳ В плане | — |
| Этап 6: Админка и масштабирование | ⏳ В плане | — |
| Этап 7: Тестирование и внедрение | ⏳ В плане | — |

**5 из 8 этапов завершены (62.5% проекта)** ✅
