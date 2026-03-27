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
