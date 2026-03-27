# Frontend Architecture

**Версия:** 1.0  
**Дата:** 27 марта 2026 г.

---

## 1. Технологии

- React 18 + TypeScript
- Vite
- Zustand stores
- API abstraction: `src/lib/apiClient.ts`
- Jitsi embed: `@jitsi/react-sdk` + `JitsiMeeting` компонент

---

## 2. Ключевые модули

- `src/pages/MessengerPage.tsx`
  - список комнат/сообщений
  - отправка текстов и файлов
  - планирование встреч через календарный API
- `src/components/JitsiMeeting.tsx`
  - подключение к конференции по room/token
  - обработка состояния call UI
- `src/lib/apiClientCore.ts`
  - базовый `fetch` wrapper
  - retry для GET
  - прокидывание auth/token/custom headers
- `frontend-admin/src/providers/AdminUiProvider.tsx`
  - `BrandingProvider + ThemeProvider` поведение в одном контексте
  - localStorage persistence (`focus_admin_branding`, `focus_admin_theme`)
- `frontend-admin/src/lib/adminApi.ts`
  - единый API-клиент для admin endpoint'ов (users/invites/bots/exchange)
- `frontend-admin/src/pages/BotsPage.tsx`
  - управление persisted настройками ботов
- `frontend-admin/src/pages/IntegrationsPage.tsx`
  - настройки Exchange EWS + test connection

---

## 3. Потоки данных

### 3.1 Чат

1. `GET /api/v1/rooms` -> render sidebar.
2. `GET /api/v1/messages?room_id=...` -> render timeline.
3. `POST /api/v1/messages` -> optimistic update + websocket reconcile.

### 3.2 Встречи (Exchange EWS)

1. `GET /api/v1/calendar/events` -> панель "Запланированные".
2. `POST /api/v1/calendar/events`:
   - формирование payload из формы
   - добавление `Idempotency-Key`
   - переход в созданную комнату при `room_id` в ответе.

### 3.3 Файлы

1. `POST /api/v1/files/upload` multipart.
2. Сохранение метаданных вложения в сообщение.
3. `GET /api/v1/files/{fileId}` для скачивания.

### 3.4 Admin-панель

1. CRUD пользователей: `GET/POST/PATCH/DELETE /api/v1/admin/users`.
2. Роли и блокировки: `PUT /users/:id/roles`, `POST /users/:id/ban|unban`.
3. Инвайты: `GET/POST /admin/invites`, `POST /admin/invites/:id/resend`.
4. Боты: `GET/POST/PATCH /admin/bots`, `POST /admin/bots/:id/enable|disable`.
5. Exchange: `GET/PUT /admin/exchange/settings`, `POST /admin/exchange/test-connection`.

---

## 4. Ошибки и устойчивость

- Сетевые ошибки GET автоматически повторяются (`retry=1`).
- Ошибки POST/PUT/DELETE сразу отображаются пользователю.
- Для календаря применяется серверная идемпотентность, frontend передаёт уникальный ключ.
- Если календарь недоступен, чатовый UX не блокируется.

---

## 5. Безопасность

- Session JWT в `Authorization: Bearer ...`.
- UI не хранит сервисные секреты.
- Передача токена в Jitsi выполняется только через backend-сгенерированные ссылки/токены.

---

## 6. Важные файлы

- `frontend/src/pages/MessengerPage.tsx`
- `frontend/src/components/JitsiMeeting.tsx`
- `frontend/src/lib/apiClient.ts`
- `frontend/src/lib/apiClientCore.ts`
- `frontend/src/index.css`
