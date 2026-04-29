# Roadmap v3: Закрытие критических разрывов

Версия документа: 3.0  
Дата создания: 25 апреля 2026  
Статус: В работе  
Источник: `docs/Competitive_Analysis.md` — gap-анализ по 25+ корпоративным мессенджерам

Цель: закрыть 5 критических функциональных разрывов, выявленных при сравнительном анализе с Compass, eXpress, Slack, MS Teams, Mattermost и другими корпоративными мессенджерами.

## Правила ведения backlog

- [ ] Обновлять статус задач минимум 2 раза в неделю.
- [ ] Для каждой закрытой задачи фиксировать ссылку на PR/commit и краткий итог.
- [ ] Не закрывать этап без прохождения критериев готовности этапа.
- [ ] Любые блокеры фиксировать в разделе «Блокеры и риски».

## Текущее состояние кодовой базы (baseline)

### Что уже есть

| Компонент | Состояние |
|-----------|-----------|
| `Message.ReplyToID` | Поле есть, связь `ReplyTo *Message` настроена. Используется для цитирования |
| `MessageReaction` модель | GORM-модель, таблица `message_reactions` (id, message_id, user_id, emoji) |
| `MessageRepository.AddReaction/RemoveReaction/GetReactions` | Реализованы, **не зарегистрированы** в HTTP-роутере |
| `MessageRepository.Search` | ILIKE-поиск по содержимому, **не выставлен** через HTTP API |
| `MessageHandler.UpdateMessage` | Ставит `Metadata.Edited = true`, **не заполняет** `EditedAt`/`EditedBy`, **нет** WS-broadcast |
| `MessageHandler.DeleteMessage` | Софт-делит `is_deleted = true`, **нет** WS-broadcast |
| `Metadata` | Содержит `Edited`, `EditedAt`, `EditedBy`, `Reactions []Reaction` |
| WS-типы | `message`, `typing`, `user_joined`, `user_left`, `error` |
| Frontend `Message` интерфейс | Нет `reply_to_id`, `reactions`, `thread_count`, `thread_root_id`, `mentions` |
| Десктоп Tauri 2 | `#[cfg_attr(mobile, tauri::mobile_entry_point)]` — подготовка под мобильную точку входа |

### Чего нет

- Тредов (ветки обсуждений) — нет `thread_root_id`, нет API для получения ветки
- @Упоминаний — нет парсинга, нет уведомлений
- HTTP API для реакций — репозиторий готов, эндпоинты не зарегистрированы
- WS-событий `message_updated`, `message_deleted`, `reaction_added`, `reaction_removed`
- Мобильных приложений (iOS/Android)

---

## Этап 1. Треды (ветки обсуждений)

Оценка: 2–3 недели  
Приоритет: критический  
Зависимости: нет

### 1.1 Backend: модель и миграция

- [x] Добавить поле `ThreadRootID *uuid.UUID` в `Message` (`gorm:"type:uuid;index:idx_thread_root"`)
  - `thread_root_id` — ссылка на корневое сообщение треда (первое сообщение, на которое отвечают)
  - `reply_to_id` сохраняется для цитирования конкретного сообщения внутри треда
  - Файл: `API_Go/internal/models/message.go`
- [x] Добавить связь `ThreadRoot *Message` (`gorm:"foreignKey:ThreadRootID"`)
- [x] GORM AutoMigrate создаст колонку и индекс автоматически (dev-среда)
- [ ] Для prod: подготовить SQL-миграцию:
  ```sql
  ALTER TABLE messages ADD COLUMN thread_root_id UUID REFERENCES messages(id);
  CREATE INDEX idx_messages_thread ON messages(thread_root_id);
  ```

### 1.2 Backend: API эндпоинты

- [x] Расширить `CreateMessage`: принимать `thread_root_id` в теле запроса, устанавливать `message.ThreadRootID`
  - Файл: `API_Go/internal/api/handlers/message_handler.go`
- [x] Новый эндпоинт `GET /api/v1/messages/{id}/thread`
  - Query: `limit` (default 50), `offset` (default 0)
  - Возвращает сообщения с `thread_root_id = {id}`, отсортированные по `created_at ASC`
  - Файл: `API_Go/internal/api/handlers/message_handler.go`
- [x] Новый метод репозитория `GetThreadMessages(ctx, rootID, limit, offset) ([]*Message, error)`
  - Файл: `API_Go/internal/repository/message_repository.go`
- [x] Новый метод репозитория `CountThreadReplies(ctx, rootID) (int64, error)`
  - Файл: `API_Go/internal/repository/message_repository.go`
- [x] В `ListMessages`: добавить `thread_count` к каждому корневому сообщению
  - Подзапрос: `SELECT COUNT(*) FROM messages WHERE thread_root_id = msg.id AND is_deleted = false`
  - Сообщения с `thread_root_id != NULL` исключаются из основной ленты (они видны только внутри треда)
- [x] Зарегистрировать маршрут в `main.go`:
  ```
  r.Route("/messages/{id}", func(r chi.Router) {
      ...
      r.Get("/thread", messageHandler.GetThread)
  })
  ```

### 1.3 Backend: WebSocket

- [x] Добавить WS-тип `thread_reply` в `API_Go/internal/websocket/hub.go`
- [x] При создании сообщения с `thread_root_id`: broadcast `thread_reply` в комнату с payload `{root_message_id, thread_count, message}`
  - Файл: `API_Go/internal/api/handlers/message_handler.go`

### 1.4 Frontend: типы и store

- [x] Расширить интерфейс `Message` в `frontend/src/store/roomsStore.ts`:
  ```typescript
  thread_root_id?: string
  thread_count?: number
  reply_to?: { id: string; content: string; user?: { name: string } }
  ```
- [x] Добавить состояние для открытого треда в `MessengerPage`:
  ```typescript
  const [activeThread, setActiveThread] = useState<Message | null>(null)
  const [threadMessages, setThreadMessages] = useState<Message[]>([])
  ```

### 1.5 Frontend: UI компоненты

- [x] Извлечь рендер сообщения из `MessengerPage.tsx` в компонент `MessageBubble`
  - Файл: `frontend/src/components/MessageBubble.tsx` (новый)
  - Props: `message`, `isMine`, `onReplyInThread`, `formatTime`, `formatFileSize`, `getInitials`
- [x] Кнопка «Ответить в треде» при hover на сообщении (иконка thread/reply)
- [x] Боковая панель треда `ThreadPanel` (slide-in справа):
  - Файл: `frontend/src/components/ThreadPanel.tsx` (новый)
  - Заголовок с корневым сообщением
  - Список сообщений треда (загрузка через `GET /api/v1/messages/{id}/thread`)
  - Форма ввода для ответа в треде
  - Закрытие панели
- [x] В основной ленте: отображение `thread_count` под корневым сообщением («N ответов»), клик открывает `ThreadPanel`
- [x] Скрывать сообщения с `thread_root_id` из основной ленты (backend фильтрация в `GetByRoomID`)

### 1.6 Frontend: WebSocket

- [x] Обработка WS-события `thread_reply`:
  - Обновить `thread_count` на корневом сообщении в основной ленте
  - Если `ThreadPanel` открыта для этого треда — добавить сообщение в список

### 1.7 Тестирование

- [x] Backend: unit-тест модели `TestMessageWithThreadRootID`, `TestMessageWithoutThreadRootID`, `TestMessageThreadAndReplyToCombined`
- [x] Backend: handler unit-тесты: валидация запросов, маршрутизация GetThread
- [x] Backend: integration-тесты (при наличии test DB): создание thread-reply, фильтрация основной ленты, thread_count, GetThread
- [x] Frontend: компоненты компилируются, build проходит

### Критерии готовности этапа 1

- [x] Пользователь может ответить в треде на любое сообщение
- [x] Тред-ответы не засоряют основную ленту
- [x] `thread_count` обновляется в реальном времени через WS
- [x] Панель треда загружает историю и позволяет писать ответы

---

## Этап 2. @Упоминания пользователей

Оценка: 1 неделя  
Приоритет: критический  
Зависимости: Этап 1 (расширение модели Message)

### 2.1 Backend: модель и парсинг

- [x] Добавить поле `Mentions []string` в `Metadata` (`json:"mentions,omitempty"`)
  - Массив UUID упомянутых пользователей
  - Файл: `API_Go/internal/models/message.go`
- [x] Реализовать парсер упоминаний в `CreateMessage`:
  - Regex: `@(\w+)` — извлечь имена пользователей из `Content`
  - Резолвить имена через `UserRepository` (новый метод `FindByNames(ctx, names []string) ([]*User, error)`)
  - Заполнить `message.Metadata.Mentions` массивом UUID
  - Файл: `API_Go/internal/api/handlers/message_handler.go`

### 2.2 Backend: API

- [x] Новый эндпоинт `GET /api/v1/users/search?q=...&room_id=...`
  - Возвращает пользователей комнаты, имена которых содержат `q` (ILIKE)
  - Лимит: 10 результатов
  - Используется фронтендом для автокомплита при вводе `@`
  - Файл: `API_Go/internal/api/handlers/user_handler.go` (новый)
- [x] Зарегистрировать маршрут: `GET /api/v1/users/search`
  - Файл: `API_Go/cmd/server/main.go`

### 2.3 Backend: WebSocket уведомления

- [x] Добавить WS-тип `mention` в `hub.go`
- [x] При `CreateMessage` с непустым `Mentions`:
  - Для каждого упомянутого `user_id` — отправить персональное WS-событие `mention`:
    ```json
    {"type": "mention", "payload": {"room_id": "...", "message_id": "...", "mentioned_by": "...", "content_preview": "..."}}
    ```
  - Реализован `Hub.SendToUser(userID, msg)` — итерация по всем WS-сессиям пользователя

### 2.4 Frontend: автокомплит

- [x] Компонент `MentionPopup`:
  - Файл: `frontend/src/components/MentionPopup.tsx` (новый)
  - Триггер: ввод `@` в поле сообщения
  - Debounced запрос к `GET /api/v1/users/search?q=...&room_id=...`
  - Список участников с аватарами, выбор клавиатурой (стрелки + Enter) или мышью
  - При выборе: вставка `@Username` в текст ввода

### 2.5 Frontend: рендер и уведомления

- [x] В `MessageBubble`: парсить `@Username` в тексте и рендерить как `<span class="mention">@Username</span>`
  - Подсветка синим цветом, курсор pointer
- [x] Обработка WS-события `mention`:
  - Показать нативное уведомление (Notification API)

### 2.6 Тестирование

- [x] Backend: unit-тест парсинга `@username` из текста (TestMentionRegex)
- [x] Backend: unit-тесты SearchUsers (пустой query, невалидный room_id)
- [x] Backend: тесты Metadata.Mentions сериализации
- [x] Frontend: сборка проходит без ошибок

### Критерии готовности этапа 2

- [x] При вводе `@` появляется popup с участниками комнаты
- [x] Упоминания подсвечиваются в тексте сообщения
- [x] Упомянутый пользователь получает WS-уведомление и нативное уведомление (десктоп)
- [x] `metadata.mentions` содержит UUID упомянутых

---

## Этап 3. Реакции на сообщения

Оценка: 1 неделя  
Приоритет: критический  
Зависимости: Этап 1 (компонент `MessageBubble`)

### 3.1 Backend: HTTP-эндпоинты

- [x] Создать `ReactionHandler` с зависимостью от `MessageRepository` и `Hub`
  - Файл: `API_Go/internal/api/handlers/reaction_handler.go` (новый)
  - Методы:
    - `AddReaction(w, r)` — `POST /api/v1/messages/{id}/reactions`
      - Body: `{"emoji": "👍"}`
      - Вызывает `msgRepo.AddReaction`, broadcast WS `reaction_added`
    - `RemoveReaction(w, r)` — `DELETE /api/v1/messages/{id}/reactions/{emoji}`
      - Вызывает `msgRepo.RemoveReaction`, broadcast WS `reaction_removed`
    - `ListReactions(w, r)` — `GET /api/v1/messages/{id}/reactions`
      - Вызывает `msgRepo.GetReactions`, агрегирует в `[{emoji, count, user_ids}]`

### 3.2 Backend: регистрация маршрутов

- [x] Зарегистрировать в `main.go`:
  ```
  r.Route("/messages/{id}", func(r chi.Router) {
      ...
      r.Post("/reactions", reactionHandler.AddReaction)
      r.Delete("/reactions/{emoji}", reactionHandler.RemoveReaction)
      r.Get("/reactions", reactionHandler.ListReactions)
  })
  ```

### 3.3 Backend: обогащение ListMessages

- [x] В `GetByRoomID`: добавить `Preload("Reactions")` с джойном по `MessageReaction`
  - Файл: `API_Go/internal/repository/message_repository.go`
- [x] В `ListMessages` response: агрегировать `message.Reactions` в `reactions_summary`

### 3.4 Backend: WebSocket

- [x] Добавить WS-типы `reaction_added`, `reaction_removed` в `hub.go`
- [x] Payload:
  ```json
  {"message_id": "...", "room_id": "...", "user_id": "...", "emoji": "👍"}
  ```

### 3.5 Frontend: типы

- [x] Расширить `Message` интерфейс в `roomsStore.ts`:
  ```typescript
  reactions_summary?: { emoji: string; count: number; user_ids: string[] }[]
  ```

### 3.6 Frontend: UI компоненты

- [x] Emoji picker при hover на сообщении:
  - Файл: `frontend/src/components/EmojiPicker.tsx` (новый)
  - Быстрые реакции: 👍 ❤️ 😂 😮 🔥 👎
  - Показывается при hover на `MessageBubble` (иконка emoji)
- [x] Строка реакций под `MessageBubble`:
  - Файл: `frontend/src/components/ReactionsBar.tsx` (новый)
  - Каждая реакция: `[emoji count]`, подсветка если текущий пользователь в `user_ids`
  - Клик по реакции: toggle (добавить/убрать свою)

### 3.7 Frontend: WebSocket

- [x] Обработка `reaction_added` / `reaction_removed`:
  - Обновить `reactions_summary` на соответствующем сообщении в state
  - Не перезагружать весь список сообщений

### 3.8 Тестирование

- [x] Backend: unit-тесты ReactionHandler (6 тестов: invalid/missing/unauthorized для add/remove/list)
- [x] Backend: unit-тесты AggregateReactions (3 теста: multiple, empty, single)
- [x] Backend: integration-тест add -> list -> remove -> verify (при наличии DB)
- [x] Frontend: сборка проходит без ошибок

### Критерии готовности этапа 3

- [x] Пользователь может добавить/убрать реакцию на любое сообщение
- [x] Реакции отображаются под сообщением с агрегированными счётчиками
- [x] Реакции обновляются в реальном времени через WS
- [x] В ListMessages реакции приходят сразу (preload)

---

## Этап 4. Редактирование и удаление сообщений

Оценка: 1 неделя  
Приоритет: критический  
Зависимости: Этап 1 (компонент `MessageBubble`)

### 4.1 Backend: доработка UpdateMessage

- [x] В `UpdateMessage`: заполнять `EditedAt` и `EditedBy`
  ```go
  now := time.Now()
  userID, _ := uuid.Parse(claims.UserID)
  message.Metadata.EditedAt = &now
  message.Metadata.EditedBy = &userID
  ```
  - Файл: `API_Go/internal/api/handlers/message_handler.go`
- [x] После `Update`: broadcast WS `message_updated` с полным обновлённым сообщением
- [x] Окно редактирования 24 часа (`MESSAGE_EDIT_WINDOW_HOURS` ENV, default 24, 0 = unlimited): возврат `410 Gone` если истекло
- [x] Валидация: пустой/слишком длинный (>10000) контент → `400`; не автор → `403`; уже удалено → `410`

### 4.2 Backend: доработка DeleteMessage

- [x] После `Delete`: broadcast WS `message_deleted` с `{message_id, room_id, thread_root_id, deleted_by}`
- [x] Гибрид-авторизация удаления чужого сообщения:
  - Автор сообщения — всегда
  - Глобальный admin (`claims.Roles` содержит `admin`) — всегда
  - Иначе: `RoomParticipant.Role IN (moderator, admin)` для конкретной комнаты через `RoomRepository.GetParticipant`
- [x] Идемпотентность: повторное удаление уже удалённого → `204 No Content`
- [x] Файл: `API_Go/internal/api/handlers/message_handler.go`

### 4.3 Backend: WebSocket типы

- [x] Добавлены `MessageTypeMessageUpdated = "message_updated"` и `MessageTypeMessageDeleted = "message_deleted"` в `hub.go`
  - Файл: `API_Go/internal/websocket/hub.go`

### 4.4 Frontend: типы и store

- [x] Расширен `Message` интерфейс:
  ```typescript
  metadata?: {
    ...
    edited?: boolean
    edited_at?: string
    edited_by?: string
  }
  is_deleted?: boolean
  ```
  - Файл: `frontend/src/store/roomsStore.ts`

### 4.5 Frontend: контекстное меню

- [x] Компонент `MessageContextMenu`:
  - Файл: `frontend/src/components/MessageContextMenu.tsx` (новый)
  - Триггер: кнопка `…` при hover на `MessageBubble`
  - Пункты: «Редактировать» (только автор и не истёк edit-window), «Удалить», «Ответить в треде»
  - Для глобального admin: «Удалить» доступно и для чужих сообщений (room-moderator проверяется backend-ом, при 403 показывается toast)
  - Закрытие по клику вне и Escape

### 4.6 Frontend: режим редактирования

- [x] При нажатии «Редактировать»:
  - Заполняется поле ввода текстом сообщения
  - Показывается индикатор «Редактирование сообщения» над полем ввода с кнопкой «Отмена»
  - При отправке: `PUT /api/v1/messages/{id}` вместо `POST`
  - После успешного обновления: выход из режима редактирования
- [x] Метка «(ред.)» рядом со временем сообщения, если `metadata.edited === true`
- [x] Обработка `410 Gone` на edit: показ ошибки «Истёк срок редактирования сообщения (24 часа)»

### 4.7 Frontend: удаление

- [x] При нажатии «Удалить»: модалка подтверждения «Удалить сообщение?» (`window.confirm`)
- [x] При подтверждении: `DELETE /api/v1/messages/{id}`
- [x] При 403: показ ошибки «Доступ запрещён: нельзя удалить это сообщение»

### 4.8 Frontend: WebSocket

- [x] Обработка `message_updated`: находит сообщение в `messages`/`threadReplies`/`activeThread` по `id`, мерджит с обновлёнными полями
- [x] Обработка `message_deleted`: помечает сообщение как `is_deleted = true`, очищает контент; если был thread-ответом — уменьшает `thread_count` корня

### 4.9 Тестирование

- [x] Backend unit (без БД): `TestUpdateMessage_InvalidID/_Unauthorized/_EmptyContent/_TooLongContent`, `TestDeleteMessage_InvalidID/_Unauthorized`
- [x] Backend integration (PostgreSQL, skip если недоступна):
  - `TestUpdateMessage_Success_SetsEditedFields_Integration` — проверяет content/Edited/EditedAt/EditedBy
  - `TestUpdateMessage_NotAuthor_Returns403_Integration`
  - `TestUpdateMessage_NotFound_Integration`
  - `TestUpdateMessage_EditWindowExpired_Returns410_Integration`
  - `TestUpdateMessage_NoLimitWhenWindowZero_Integration`
  - `TestDeleteMessage_Author_Success_Integration`
  - `TestDeleteMessage_GlobalAdmin_DeletesOthers_Integration`
  - `TestDeleteMessage_RoomModerator_DeletesOthers_Integration`
  - `TestDeleteMessage_RoomAdmin_DeletesOthers_Integration`
  - `TestDeleteMessage_RegularUser_Forbidden_Integration`
  - `TestDeleteMessage_NotFound_Integration`
  - `TestDeleteMessage_AlreadyDeleted_Idempotent_Integration`
- [x] Backend models: `TestMetadataEditedAtSerialization`, `TestMetadataEditedBySerialization`, `TestMetadataEditedFieldsAbsentWhenNil`
- [x] Frontend: `MessageContextMenu.test.tsx` (11 тестов), `MessageBubble.test.tsx` (10 тестов)
- [x] Frontend: build (`npm run build`) проходит без TS-ошибок

### Критерии готовности этапа 4

- [x] Пользователь может редактировать свои сообщения в течение 24 часов (`MESSAGE_EDIT_WINDOW_HOURS`), отображается метка «(ред.)»
- [x] Пользователь может удалять свои сообщения с подтверждением
- [x] Админ (глобальный) и модератор/админ комнаты могут удалять чужие сообщения
- [x] Изменения и удаления транслируются через WS в реальном времени
- [x] `EditedAt` и `EditedBy` корректно заполняются в metadata

---

## Этап 5. Мобильные приложения (iOS / Android)

Оценка: 2–3 месяца  
Приоритет: критический  
Зависимости: Этапы 1–4 (полный чат-функционал)

Стратегия: **Tauri 2 Mobile** — переиспользование 100% React-фронтенда и Rust-бэкенда десктопного клиента. Альтернатива (React Native / Flutter) отклонена из-за необходимости переписывания UI и дублирования Rust-команд.

### Фаза 5.1 — Подготовка и адаптивный UI (2 недели) ✅ PR #17

- [x] Рефакторинг `MessengerPage.tsx`: извлечь в переиспользуемые компоненты
  - `RoomSidebar` — список комнат (`frontend/src/components/RoomSidebar.tsx`)
  - `ChatHeader` — шапка чата (`frontend/src/components/ChatHeader.tsx`)
  - `MessageBubble` — (из этапа 1)
  - `ThreadPanel` — (из этапа 1)
- [x] Адаптивный CSS: media queries для экранов < 768px
  - Sidebar: slide-in по hamburger-кнопке или свайпу вправо + backdrop
  - Chat: занимает 100% ширины
  - Thread panel: fullscreen overlay
  - safe-area для iOS notch / Android cutout
  - Файл: `frontend/src/index.css`
- [x] Touch UX:
  - `useLongPress` — long-press на сообщении открывает emoji-picker
  - `useSwipe` — свайп вправо/влево управляет sidebar
  - `usePullToRefresh` — каркас для будущего pull-to-refresh
  - Файлы: `frontend/src/hooks/{useLongPress,useSwipe,usePullToRefresh}.ts`
- [x] PWA: `vite-plugin-pwa` (injectManifest), manifest.webmanifest, service worker
  - `frontend/src/sw-push.ts` — обработчики push/notificationclick
  - `frontend/public/icons/icon-{192,512,maskable}.png`
- [x] Структура: `mobile/` создаётся в фазе 5.2 как отдельный Cargo-крейт workspace

### Фаза 5.2 — Android (PR-C, каркас)

- [x] Инициализация: `cargo tauri android init` через CI
  (`.github/workflows/mobile-android.yml`)
- [x] Конфигурация `mobile/src-tauri/tauri.conf.json`:
  - `identifier`: `com.focus.messenger.mobile`
  - Permissions: `INTERNET`, `CAMERA`, `RECORD_AUDIO`, `POST_NOTIFICATIONS`,
    `VIBRATE`, foreground services (см. `mobile/templates/AndroidManifest.xml`)
- [x] Rust-команды для OAuth (PKCE) с deep-link `focus://auth/callback`
  (`mobile/src-tauri/src/commands.rs`: `prepare_oauth_url`, `exchange_code`)
- [x] Push-уведомления:
  - Web Push в WebView через PWA service worker (PR-A) — работает «из коробки»
  - Backend pluggable Sender (PR-B): WebPush готов, FCM/APNs — каркасы-заглушки
  - Эндпоинты `POST /api/v1/push/{register,unregister}` и
    `GET /api/v1/push/vapid-public-key`
- [x] Реальная сборка debug APK в CI (ubuntu-latest, debug keystore,
  Android SDK 34, NDK 27, JDK 17, Rust + 4 Android-таргета)
- [x] Артефакт `focus-mobile-debug-apk` публикуется на 14 дней
- [ ] Production keystore + signed AAB для Google Play (TBD после первого reviewа)
- [ ] Реализовать FCM-провайдер вместо заглушки (для нативных push)

### Фаза 5.3 — iOS (PR-C, каркас)

- [x] Инициализация: `cargo tauri ios init` через CI
  (`.github/workflows/mobile-ios.yml`)
- [x] Конфигурация:
  - Capabilities: Push Notifications, Background Modes (remote-notification, voip),
    Camera, Microphone (`mobile/templates/Info.plist`)
  - Info.plist: URL scheme `focus://`, ATS-исключение для `focus.local`
  - Bundle ID: `com.focus.messenger.mobile`, минимальный таргет iOS 14.0
- [x] OAuth callback: custom scheme `focus://auth/callback` (Universal Links —
      готовы в манифесте, требуют публичного AASA-файла)
- [x] Push-уведомления:
  - Web Push в WebView (для PWA, добавленной в Safari, iOS 16.4+)
  - APNs-канал в backend (заглушка, готова к расширению)
- [x] CI macos-latest, debug build на симуляторе без подписи
- [ ] Подпись и публикация (требует Apple Developer аккаунта на MacBook Pro)
- [ ] Реализовать APNs-провайдер вместо заглушки (HTTP/2 + .p8 ключ)

### Фаза 5.4 — Backend: Push-инфраструктура (PR-B)

- [x] Новая таблица `push_tokens` (user_id, platform, endpoint, p256dh, auth, ...)
  - Модель: `API_Go/internal/models/push_token.go`
  - GORM AutoMigrate подключён в `cmd/server/main.go`
- [x] Эндпоинты:
  - `GET /api/v1/push/vapid-public-key` — публичный VAPID-ключ
  - `POST /api/v1/push/register` — регистрация подписки (Web Push / FCM / APNs)
  - `POST /api/v1/push/unregister` — удаление подписки
- [x] Push-сервис: `API_Go/internal/push/` (новый пакет)
  - Pluggable Sender: `WebPushSender` (готов), `FCMSender` / `APNSSender` (заглушки-каркасы)
  - `Service` параллельно отправляет уведомления и удаляет Gone-подписки
- [x] Интеграция с `CreateMessage`: офлайн-участники получают push,
      упомянутые — приоритетный push с тегом `mention-*`
- [x] Frontend: `frontend/src/lib/pushSubscribe.ts` + автоподписка в `MessengerPage`
- [x] VAPID-генератор: `cmd/vapidgen` (`go run ./cmd/vapidgen`)
- [x] Утилита online/offline в WS Hub: `IsUserOnline(userID)`

### Фаза 5.5 — Публикация (2 недели)

- [ ] Google Play Console:
  - Листинг: название, описание, скриншоты (телефон + планшет)
  - Privacy Policy URL
  - Внутреннее / закрытое тестирование → production
- [ ] Apple App Store Connect:
  - App Store listing
  - TestFlight для бета-тестирования
  - App Review submission
- [ ] OTA-обновления: Tauri updater endpoint для мобильных или встроенный механизм
- [ ] Документация: инструкции установки для пользователей

### Критерии готовности этапа 5

- [ ] Android-приложение работает на Android 8+ (API 26+)
- [ ] iOS-приложение работает на iOS 14+
- [ ] Все функции чата (треды, упоминания, реакции, редактирование) работают на мобильных
- [ ] Push-уведомления доставляются при офлайн
- [ ] OAuth через Keycloak работает на обеих платформах
- [ ] Видеозвонки (Jitsi) работают в WebView
- [ ] Приложения опубликованы в Google Play и App Store (или корпоративное распространение)

---

## Этап 6. Развёртывание stage-окружения и стабилизация

Оценка: 1 спринт (фактически выполнено 28 апреля 2026)
Приоритет: критический (блокировал доступ к мессенджеру)
Зависимости: Этапы 1–4

Сквозная задача: «запустить текущий проект полностью в кластере (Kaniko + Argo CD + Nexus), починить авторизацию (Keycloak SSO + локальная) и видеозвонки Jitsi».

### 6.1 CI/CD и реестр образов

- [x] Сборка образов через **Kaniko** в кластере (k3s) с пушем в **Nexus** (`registry.focus.local`)
  - Манифесты: `k8s/kaniko-jobs.yaml` (api-go, frontend, frontend-admin)
  - `hostAliases` для `registry.focus.local` → IP Nginx Ingress LoadBalancer (`10.204.41.11`), `--skip-tls-verify`, `--cache=true` через `registry.focus.local/focus/cache`
  - Креды реестра — Secret `nexus-docker-config`, GitHub-токен — Secret `github-creds`
- [x] **Argo CD** Application'ы (`k8s/argocd/*.yaml`) синхронизируют:
  - `messenger-stage` ← `k8s/stage/`
  - `jitsi` ← `k8s/jitsi-stage/`
- [x] CI workflow `.github/workflows/ci-cd.yml` обновлён до Go 1.25, golangci-lint v2.5.0 (конфиг `API_Go/.golangci.yml` отключает несовместимые проверки)

### 6.2 Деплой backend и фронтендов

- [x] `messenger-stage`: деплои `api-go`, `frontend`, `frontend-admin`, `postgres`, `redis`, `nginx-ingress`
  - Манифест: `k8s/stage/focus.yaml`
  - Secret `focus-secrets` (DB, Keycloak, JWT, Jitsi, Exchange и т.д.)
  - ConfigMap `frontend-admin-sso` для админки
- [x] `NetworkPolicy` `allow-internal-namespace` в `messenger-stage` — фикс `502 Bad Gateway` между frontend и api-go (PR #7)
- [x] `.dockerignore` в `frontend/` и `frontend-admin/` исправлены так, чтобы `.env.production` попадал в build-контекст Vite (исчезла надпись «SSO не настроен») (PR #8)

### 6.3 Авторизация Keycloak SSO + локальная

- [x] OIDC: разделение discovery URL (внутренний HTTP `keycloak.keycloak.svc.cluster.local`) и issuer URL (публичный `https://auth.focus.local`) через `oidc.InsecureIssuerURLContext`
  - Файл: `API_Go/internal/auth/oidc.go`
  - `KEYCLOAK_URL` = `https://auth.focus.local` фиксирует issuer mismatch (PR #11)
- [x] `UserRepository.GetOrCreate`: при существовании пользователя по email обновляет `keycloak_id` вместо вставки дубликата (фикс «duplicate key value violates unique constraint») (PR #9)
  - Покрыто тестами `TestUserRepository_GetOrCreate_LinksExistingByEmail`, `_CreatesNewWhenNoExisting`, `_IdempotentByKeycloakID`
- [x] Локальный логин: `password_hash` для `admin@focus.local` и `chat@focus.local` синхронизирован с Keycloak (исправлен `401` на `/api/v1/auth/local/login`)
- [x] В Keycloak созданы пользователи `chat@focus.local` (`Chat2026!`) и `admin@focus.local` (`Admin2026!`) с ролями `user`/`admin`

### 6.4 Развёртывание Jitsi-стека

- [x] Полный Jitsi-стек в namespace `jitsi`: `prosody`, `jicofo`, `jvb` (с `hostNetwork`), `jitsi-web` (PR #10)
  - Манифесты: `k8s/jitsi-stage/jitsi.yaml`, `k8s/argocd/jitsi-application.yaml`
  - `ENABLE_IPV6=0` в `jitsi-env` — фикс `socket() [::]:80 failed` для `jitsi-web` на k3s
  - JVB: `nodeSelector` на ноду с публичным IP, `DOCKER_HOST_ADDRESS=10.204.41.11`, UDP/10000 через hostPort
  - `NetworkPolicy` `jitsi-allow-internal` разрешает трафик от ingress-nginx, messenger-stage, нодовых CIDR (`10.204.41.0/24`) и pod CIDR (`10.42.0.0/16`)
- [x] `Ingress jitsi-stage-ingress`: `meet.focus.local` → `jitsi-web:80`
- [x] api-go: `JITSI_BASE_URL=https://meet.focus.local:30443`, `JITSI_APP_ID/ISSUER/AUDIENCE=jitsi`, `meet.focus.local` в `hostAliases`

### 6.5 Hotfix-ы JWT для Jitsi

- [x] **`sub` claim** добавлен в JWT (`mod_token_verification` в prosody читает `claims.sub` как XMPP-домен, при отсутствии падает на Lua-конкатенации) (PR #12)
  - `JitsiConfig.Subject` (env `JITSI_SUBJECT`, default `*` — wildcard, совместим с `enable_domain_verification = false`)
  - Тесты `TestGenerateTokenIncludesSubjectClaim` (явный subject и дефолт)
- [x] **`aud` claim сериализуется как строка**, а не массив (`luajwtjitsi.lib.lua/verify_claim` сравнивает `claim == accepted` и не понимает массив; стандартный `jwt.RegisteredClaims` давал `["jitsi"]`) (PR #13)
  - В `JitsiClaims` убран embedded `jwt.RegisteredClaims`, заведены собственные поля; реализован интерфейс `jwt.Claims`
  - Тест `TestGenerateTokenAudienceIsJSONString` проверяет, что `aud` в JSON — именно строка
- [x] Kaniko-ребилд `api-go:latest` после каждого фикса, `kubectl rollout restart deploy/api-go -n messenger-stage`

### Критерии готовности этапа 6

- [x] Все компоненты раскатаны в кластере через Argo CD из `master`
- [x] Доступ к чату по `https://chat.focus.local:30443` через Keycloak SSO и локальный логин
- [x] Доступ к админке по `https://admin.focus.local:30443` для глобального admin
- [x] Видеозвонки Jitsi подключаются к XMPP/prosody без ошибок `connection.passwordRequired` / `invalid 'aud' claim` / `Извините, вам не разрешено присоединиться`
- [x] CI/CD: dev → PR → master → Kaniko build → Nexus → Argo CD sync

---

## Зависимости между этапами

```
Этап 1 (Треды) ──► Этап 2 (Упоминания) ──► Этап 3 (Реакции) ──► Этап 4 (Редактирование)
                                                                          │
                                                                          ▼
                                                              Этап 6 (Stage-деплой) ──► Этап 5 (Мобильные)
```

Этапы 1–4 последовательные: каждый расширяет модель `Message`, WS-протокол и компонент `MessageBubble`.  
Этап 6 (развёртывание в кластере, Jitsi/Keycloak hotfix-ы) выполняется после функционального этапа 4, чтобы выкатить на пользователей и стабилизировать stage-окружение.  
Этап 5 зависит от завершения 1–4 и наличия рабочего stage (этап 6) для интеграционных проверок мобильных клиентов.  
Фаза 5.4 (Push-инфраструктура) может выполняться параллельно с фазами 5.2/5.3.

---

## Блокеры и риски

- [ ] Риск: Tauri 2 Mobile — зрелость мобильной платформы (beta/early stable). Fallback: React Native с общей бизнес-логикой.
- [ ] Риск: Apple App Review — возможны задержки и требования к изменениям.
- [ ] Риск: Push-уведомления через FCM/APNs требуют аккаунтов разработчика (Google Play Console + Apple Developer Program).
- [ ] Риск: адаптивный UI — тестирование на большом количестве размеров экранов.
- [ ] Риск: Jitsi в мобильном WebView — возможны ограничения камеры/микрофона, требуется тестирование.

---

## Быстрый трек (ближайший спринт)

- [x] Этап 1.1–1.3: backend треды (модель + API + WS)
- [x] Этап 1.4–1.6: frontend треды (UI + WS)
- [x] Этап 3.1–3.4: backend реакции (подключить существующий репозиторий к HTTP + WS)
- [x] Этап 4.1–4.9: редактирование/удаление сообщений (24-часовое окно, гибрид-авторизация, WS-broadcast)
- [x] Этап 6: развёртывание stage (Kaniko + Argo CD + Nexus), фикс Keycloak SSO/локального логина, фикс Jitsi JWT (`sub`, `aud` как строка)
- [x] Этап 5.1: адаптивный UI, Touch UX и PWA (PR #17)
- [x] Этап 5.4: backend push-инфраструктура (Web Push + каркасы FCM/APNs, PR #18)
- [x] Этап 5.2 / 5.3 (каркас): Tauri 2 Mobile workspace, реальный APK в CI, iOS-каркас (PR #?)
- [ ] Этап 5.5: подпись release-сборок и публикация в Google Play / App Store
