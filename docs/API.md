# API Specification

**Версия:** 1.0  
**Дата:** 24 марта 2026 г.  
**Статус:** Черновик

---

## 1. Обзор

### 1.1. Базовый URL

```
Production:  https://api.company.com
Staging:     https://api-staging.company.com
Dev:         https://api-dev.company.com
Local:       http://localhost:8080
```

### 1.2. Аутентификация

Все запросы (кроме `/auth/*` и `/health`) требуют JWT токен:

```
Authorization: Bearer <jwt_token>
```

### 1.3. Формат запросов/ответов

- **Content-Type:** `application/json`
- **Кодировка:** UTF-8
- **Время:** RFC 3339 (`2024-01-01T12:00:00Z`)

### 1.4. Пагинация

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5,
    "next": "/api/v1/rooms?page=2",
    "prev": null
  }
}
```

---

## 2. Authentication API

### 2.1. Инициация входа

**Endpoint:** `GET /api/v1/auth/login`

**Описание:** Редирект на Keycloak для OIDC аутентификации

**Параметры query:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| redirect_uri | string | URL для возврата (опционально) |

**Ответ:** `302 Found` (редирект на Keycloak)

---

### 2.2. Callback от Keycloak

**Endpoint:** `GET /api/v1/auth/callback`

**Параметры query:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| code | string | Authorization code |
| state | string | State token для CSRF защиты |

**Ответ:** `302 Found` (редирект на фронтенд с session JWT)

```
Location: https://chat.company.com?token=<session_jwt>
```

---

### 2.3. Refresh токена

**Endpoint:** `POST /api/v1/auth/refresh`

**Headers:**
```
Authorization: Bearer <refresh_token>
```

**Body (fallback, если header не передан):**
```json
{
  "refresh_token": "<refresh_token>"
}
```

**Ответ:** `200 OK`

```json
{
  "access_token": "<new_access_token>",
  "expires_in": 86400,
  "token_type": "Bearer"
}
```

**Ошибки:**
- `400 Bad Request` — отсутствует refresh token или некорректный `Authorization`.
- `503 Service Unavailable` — OIDC provider временно недоступен.

---

### 2.4. Logout

**Endpoint:** `POST /api/v1/auth/logout`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `204 No Content`

**Поведение:**
- Текущий `session_token` добавляется в revocation blacklist до его `exp`.
- Повторное использование отозванного токена на защищенных endpoint возвращает `401 Unauthorized`.

**Ошибки:**
- `400 Bad Request` — отсутствует/некорректный `Authorization` header.
- `401 Unauthorized` — токен невалиден или истек.

---

### 2.5. Получение текущего пользователя

**Endpoint:** `GET /api/v1/auth/me`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `200 OK`

```json
{
  "id": "user-uuid",
  "keycloak_id": "kc-uuid",
  "email": "user@company.com",
  "name": "User Name",
  "avatar_url": "https://storage.company.com/avatars/user.jpg",
  "roles": ["user", "moderator"],
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

---

## 3. Rooms API

### 3.1. Список комнат

**Endpoint:** `GET /api/v1/rooms`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Параметры query:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| page | integer | Номер страницы (default: 1) |
| per_page | integer | Записей на страницу (default: 20, max: 100) |
| search | string | Поиск по названию |
| type | string | Фильтр: `public`, `private`, `meeting` |

**Ответ:** `200 OK`

```json
{
  "data": [
    {
      "id": "room-uuid",
      "name": "Общий чат",
      "type": "public",
      "creator": {
        "id": "user-uuid",
        "name": "Creator Name"
      },
      "participants_count": 15,
      "last_message": {
        "id": "msg-uuid",
        "content": "Привет!",
        "created_at": "2024-01-01T12:00:00Z",
        "user": {
          "id": "user-uuid",
          "name": "User Name"
        }
      },
      "unread_count": 3,
      "created_at": "2024-01-01T12:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 50,
    "total_pages": 3,
    "next": "/api/v1/rooms?page=2",
    "prev": null
  }
}
```

---

### 3.2. Создать комнату

**Endpoint:** `POST /api/v1/rooms`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "name": "Новая комната",
  "type": "private",
  "participant_ids": ["user-uuid-1", "user-uuid-2"],
  "is_meeting_room": false
}
```

**Валидация:**
| Поле | Тип | Требования |
|------|-----|------------|
| name | string | 3-50 символов, буквы/цифры/-/_ |
| type | string | `public`, `private`, `meeting` |
| participant_ids | array | UUID валидные |
| is_meeting_room | boolean | опционально |

**Ответ:** `201 Created`

```json
{
  "id": "room-uuid",
  "name": "Новая комната",
  "type": "private",
  "jitsi_room_name": "room-uuid-jitsi",
  "jitsi_url": "https://meet.company.com/room-uuid-jitsi",
  "jitsi_jwt": "<jwt_token>",
  "creator": {
    "id": "user-uuid",
    "name": "Creator Name"
  },
  "participants": [],
  "created_at": "2024-01-01T12:00:00Z"
}
```

---

### 3.3. Детали комнаты

**Endpoint:** `GET /api/v1/rooms/:id`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `200 OK`

```json
{
  "id": "room-uuid",
  "name": "Общий чат",
  "type": "public",
  "description": "Описание комнаты",
  "creator": {
    "id": "user-uuid",
    "name": "Creator Name",
    "email": "creator@company.com"
  },
  "participants": [
    {
      "id": "user-uuid",
      "name": "User Name",
      "email": "user@company.com",
      "role": "member",
      "joined_at": "2024-01-01T12:00:00Z"
    }
  ],
  "jitsi_room_name": "room-uuid-jitsi",
  "jitsi_url": "https://meet.company.com/room-uuid-jitsi",
  "settings": {
    "allow_guests": false,
    "require_moderator_for_messages": false,
    "max_participants": 100
  },
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

---

### 3.4. Обновить комнату

**Endpoint:** `PUT /api/v1/rooms/:id`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "name": "Обновлённое название",
  "description": "Новое описание",
  "settings": {
    "allow_guests": true,
    "max_participants": 50
  }
}
```

**Ответ:** `200 OK`

```json
{
  "id": "room-uuid",
  "name": "Обновлённое название",
  "description": "Новое описание",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

---

### 3.5. Удалить комнату

**Endpoint:** `DELETE /api/v1/rooms/:id`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `204 No Content`

---

### 3.6. Присоединиться к комнате (получить JWT)

**Endpoint:** `POST /api/v1/rooms/:id/join`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `200 OK`

```json
{
  "jitsi_room_name": "room-uuid-jitsi",
  "jitsi_url": "https://meet.company.com/room-uuid-jitsi?jwt=<jwt>",
  "jitsi_jwt": "<jwt_token>",
  "user": {
    "id": "user-uuid",
    "name": "User Name",
    "email": "user@company.com",
    "moderator": false
  },
  "expires_at": "2024-01-01T20:00:00Z"
}
```

---

### 3.7. Добавить участника

**Endpoint:** `POST /api/v1/rooms/:id/participants`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "user_ids": ["user-uuid-1", "user-uuid-2"]
}
```

**Ответ:** `200 OK`

```json
{
  "added": [
    {
      "id": "user-uuid-1",
      "name": "User 1"
    }
  ],
  "failed": [
    {
      "id": "user-uuid-2",
      "reason": "User not found"
    }
  ]
}
```

---

### 3.8. Удалить участника

**Endpoint:** `DELETE /api/v1/rooms/:id/participants/:user_id`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `204 No Content`

---

## 4. Messages API

### 4.1. Получить историю сообщений

**Endpoint:** `GET /api/v1/rooms/:id/messages`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Параметры query:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| before | string | RFC3339 timestamp, сообщения до |
| after | string | RFC3339 timestamp, сообщения после |
| limit | integer | Максимум сообщений (default: 50, max: 200) |

**Ответ:** `200 OK`

```json
{
  "data": [
    {
      "id": "msg-uuid",
      "room_id": "room-uuid",
      "user": {
        "id": "user-uuid",
        "name": "User Name",
        "avatar_url": "https://storage.company.com/avatars/user.jpg"
      },
      "content": "Привет всем!",
      "type": "text",
      "metadata": {
        "edited": false,
        "reactions": [
          {"emoji": "👍", "count": 3, "users": ["user-uuid"]}
        ]
      },
      "created_at": "2024-01-01T12:00:00Z",
      "updated_at": "2024-01-01T12:00:00Z"
    }
  ],
  "has_more": true,
  "next_cursor": "msg-uuid-prev"
}
```

---

### 4.2. Отправить сообщение

**Endpoint:** `POST /api/v1/rooms/:id/messages`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "content": "Привет!",
  "type": "text",
  "metadata": {
    "reply_to": "msg-uuid"
  }
}
```

**Валидация:**
| Поле | Тип | Требования |
|------|-----|------------|
| content | string | 1-10000 символов |
| type | string | `text`, `image`, `file`, `system` |
| metadata | object | опционально |

**Ответ:** `201 Created`

```json
{
  "id": "msg-uuid",
  "room_id": "room-uuid",
  "user": {
    "id": "user-uuid",
    "name": "User Name"
  },
  "content": "Привет!",
  "type": "text",
  "metadata": {},
  "created_at": "2024-01-01T12:00:00Z"
}
```

---

### 4.3. Обновить сообщение

**Endpoint:** `PUT /api/v1/messages/:id`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "content": "Обновлённый текст"
}
```

**Ответ:** `200 OK`

```json
{
  "id": "msg-uuid",
  "content": "Обновлённый текст",
  "metadata": {
    "edited": true,
    "edited_at": "2024-01-01T12:05:00Z"
  },
  "updated_at": "2024-01-01T12:05:00Z"
}
```

---

### 4.4. Удалить сообщение

**Endpoint:** `DELETE /api/v1/messages/:id`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `204 No Content`

---

### 4.5. Добавить реакцию

**Endpoint:** `POST /api/v1/messages/:id/reactions`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "emoji": "👍"
}
```

**Ответ:** `200 OK`

```json
{
  "emoji": "👍",
  "count": 4,
  "user_has_reacted": true
}
```

---

### 4.6. Удалить реакцию

**Endpoint:** `DELETE /api/v1/messages/:id/reactions/:emoji`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `204 No Content`

---

## 5. Calendar API

### 5.1. Получить события календаря

**Endpoint:** `GET /api/v1/calendar/events`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Параметры query:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| start | string | RFC3339, начало периода |
| end | string | RFC3339, конец периода |
| user_id | string | ID пользователя (для админов) |

**Ответ:** `200 OK`

```json
{
  "data": [
    {
      "id": "event-uuid",
      "exchange_event_id": "exchange-id",
      "subject": "Планёрка команды",
      "description": "Еженедельная встреча",
      "start_time": "2024-01-02T10:00:00Z",
      "end_time": "2024-01-02T11:00:00Z",
      "location": "Jitsi Meeting",
      "jitsi_url": "https://meet.company.com/room-uuid",
      "room_id": "room-uuid",
      "organizer": {
        "id": "user-uuid",
        "name": "Organizer Name",
        "email": "organizer@company.com"
      },
      "attendees": [
        {
          "id": "user-uuid",
          "name": "Attendee Name",
          "email": "attendee@company.com",
          "status": "accepted"
        }
      ],
      "created_at": "2024-01-01T12:00:00Z",
      "updated_at": "2024-01-01T12:00:00Z"
    }
  ]
}
```

---

### 5.2. Создать встречу с Jitsi комнатой

**Endpoint:** `POST /api/v1/calendar/events`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "subject": "Планёрка команды",
  "description": "Еженедельная встреча",
  "start_time": "2024-01-02T10:00:00Z",
  "end_time": "2024-01-02T11:00:00Z",
  "attendee_emails": ["user1@company.com", "user2@company.com"],
  "create_jitsi_room": true,
  "jitsi_room_name": "team-meeting"
}
```

**Валидация:**
| Поле | Тип | Требования |
|------|-----|------------|
| subject | string | 1-200 символов |
| start_time | string | RFC3339, будущее время |
| end_time | string | RFC3339, после start_time |
| attendee_emails | array | валидные email |
| create_jitsi_room | boolean | опционально, default: true |

**Ответ:** `201 Created`

```json
{
  "id": "event-uuid",
  "exchange_event_id": "exchange-id",
  "subject": "Планёрка команды",
  "jitsi_url": "https://meet.company.com/room-uuid",
  "room_id": "room-uuid",
  "invitations_sent": 2,
  "created_at": "2024-01-01T12:00:00Z"
}
```

---

### 5.3. Обновить событие

**Endpoint:** `PUT /api/v1/calendar/events/:id`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "subject": "Новое название",
  "start_time": "2024-01-02T11:00:00Z",
  "end_time": "2024-01-02T12:00:00Z"
}
```

**Ответ:** `200 OK`

```json
{
  "id": "event-uuid",
  "subject": "Новое название",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

---

### 5.4. Отменить встречу

**Endpoint:** `DELETE /api/v1/calendar/events/:id`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Параметры query:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| send_cancellation | boolean | Отправить cancellation notification (default: true). При `false` удаляется только событие. |

**Ответ:** `204 No Content`

---

## 6. Webhooks API

### 6.1. Список вебхуков

**Endpoint:** `GET /api/v1/webhooks`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `200 OK`

```json
{
  "data": [
    {
      "id": "webhook-uuid",
      "url": "https://external.system.com/webhook",
      "event_types": ["conference.created", "conference.ended"],
      "enabled": true,
      "secret": "whsec_xxx",
      "created_at": "2024-01-01T12:00:00Z"
    }
  ]
}
```

---

### 6.2. Создать вебхук

**Endpoint:** `POST /api/v1/webhooks`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "url": "https://external.system.com/webhook",
  "event_types": ["conference.created", "participant.joined"],
  "secret": "my-secret"
}
```

**Ответ:** `201 Created`

```json
{
  "id": "webhook-uuid",
  "url": "https://external.system.com/webhook",
  "event_types": ["conference.created", "participant.joined"],
  "enabled": true,
  "created_at": "2024-01-01T12:00:00Z"
}
```

---

### 6.3. Входящий webhook от Jitsi

**Endpoint:** `POST /api/v1/webhooks/jitsi`

**Headers:**
```
X-Jitsi-Signature: sha256=<hmac_sha256_hex(payload)>
X-Idempotency-Key: <optional_unique_event_key>
```

Поведение:
- подпись проверяется относительно `JITSI_APP_SECRET`;
- если `X-Idempotency-Key` не передан, используется `sha256(payload)`;
- дубликаты по `source + idempotency_key` не обрабатываются повторно.

**Body:**
```json
{
  "event": "conference.created",
  "conference_name": "room-uuid",
  "timestamp": "2024-01-01T12:00:00Z",
  "data": {
    "room": "room-uuid",
    "creator": "user-uuid"
  }
}
```

**Ответ:** `200 OK`

```json
{
  "status": "accepted"
}
```

Для дубликата:
```json
{
  "status": "duplicate"
}
```

**Ошибки:**
- `401 Unauthorized` — отсутствует/неверная подпись.
- `400 Bad Request` — невалидный payload.

---

### 6.4. Удалить вебхук

**Endpoint:** `DELETE /api/v1/webhooks/:id`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `204 No Content`

---

### 6.5. Политика outbound-доставки

Для исходящих webhook-событий используется dispatcher со следующей политикой:
- выборка только активных webhook по `event_type`;
- до `3` попыток доставки (1 основная + 2 retry);
- exponential backoff между retry (базовая задержка 250ms);
- при исчерпании retry событие фиксируется как `dead_letter` в delivery log.

Служебные заголовки исходящих webhook:
```
X-Webhook-Event: <event_type>
X-Webhook-Timestamp: <RFC3339>
X-Webhook-Signature: <base64(hmac_sha256(timestamp + "." + payload))>
```

---

## 7. Bots API

### 7.0. Авто-ответы ботов в комнатах

При отправке командного сообщения (например `/help`) в `/api/v1/messages` бот:
- создает реальное системное сообщение в таблице `messages` (`type: "system"`);
- публикует это сообщение в websocket комнаты как обычный `message` event;
- отвечает только если отправитель является участником комнаты.

---

### 7.1. Список ботов

**Endpoint:** `GET /api/v1/bots`

**Headers:**
```
Authorization: Bearer <session_token>
```

**Ответ:** `200 OK`

```json
{
  "data": [
    {
      "id": "bot-uuid",
      "name": "Meeting Bot",
      "description": "Бот для создания встреч",
      "avatar_url": "https://storage.company.com/bots/meeting.png",
      "enabled": true,
      "commands": ["/create", "/schedule", "/help"],
      "created_at": "2024-01-01T12:00:00Z"
    }
  ]
}
```

---

### 7.2. Создать бота

**Endpoint:** `POST /api/v1/bots`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "name": "Custom Bot",
  "description": "Описание бота",
  "commands": [
    {
      "command": "/hello",
      "handler": "greeting"
    }
  ]
}
```

**Ответ:** `201 Created`

```json
{
  "id": "bot-uuid",
  "name": "Custom Bot",
  "token": "bot-token-xxx",
  "enabled": true,
  "created_at": "2024-01-01T12:00:00Z"
}
```

---

### 7.3. Выполнить команду бота

**Endpoint:** `POST /api/v1/bots/command`

**Headers:**
```
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Body:**
```json
{
  "bot_id": "bot-uuid",
  "room_id": "room-uuid",
  "command": "/create",
  "args": ["meeting", "Планёрка", "tomorrow", "10:00"]
}
```

**Ответ:** `200 OK`

```json
{
  "success": true,
  "response": "Встреча создана: https://meet.company.com/room-uuid",
  "data": {
    "room_id": "room-uuid",
    "jitsi_url": "https://meet.company.com/room-uuid"
  }
}
```

Примечание: endpoint зарезервирован, основная обработка bot-команд выполняется через `POST /api/v1/messages` (сообщения, начинающиеся с `/`).

---

### 7.4. Команды через сообщения (`POST /api/v1/messages`)

Поддерживаемые команды:
- `/create meeting <название>` — создает meeting-room, добавляет отправителя как `moderator`, возвращает Jitsi URL.
- `/schedule meeting <название> at <YYYY-MM-DD HH:MM|RFC3339>` — создает meeting-room и планирует встречу; при доступном Exchange создается календарное событие.
- `/status` — показывает сводку по комнатам и активным встречам.

Ограничения:
- команды обрабатываются только для участников комнаты;
- применяется rate-limit на bot-команды per-user (защита от spam/flood).

---

## 8. Admin API

### 8.1. Список пользователей

**Endpoint:** `GET /api/v1/admin/users`

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Ответ:** `200 OK`

```json
{
  "data": [
    {
      "id": "user-uuid",
      "email": "user@company.com",
      "name": "User Name",
      "roles": ["user"],
      "active": true,
      "last_login": "2024-01-01T12:00:00Z",
      "created_at": "2024-01-01T12:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 100
  }
}
```

---

### 8.2. Обновить роли пользователя

**Endpoint:** `PUT /api/v1/admin/users/:id/roles`

**Headers:**
```
Authorization: Bearer <admin_token>
Content-Type: application/json
```

**Body:**
```json
{
  "roles": ["user", "moderator"]
}
```

**Ответ:** `200 OK`

```json
{
  "id": "user-uuid",
  "roles": ["user", "moderator"],
  "updated_at": "2024-01-01T12:00:00Z"
}
```

---

### 8.3. Заблокировать пользователя

**Endpoint:** `POST /api/v1/admin/users/:id/ban`

**Headers:**
```
Authorization: Bearer <admin_token>
Content-Type: application/json
```

**Body:**
```json
{
  "reason": "Нарушение правил",
  "duration_hours": 24
}
```

**Ответ:** `200 OK`

```json
{
  "id": "user-uuid",
  "banned": true,
  "reason": "Нарушение правил",
  "banned_until": "2024-01-02T12:00:00Z"
}
```

---

### 8.4. Активные конференции

**Endpoint:** `GET /api/v1/admin/conferences`

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Ответ:** `200 OK`

```json
{
  "data": [
    {
      "id": "room-uuid",
      "room_id": "room-uuid",
      "room_name": "Общий чат",
      "jitsi_room": "room-uuid-jitsi",
      "participants_count": 5,
      "started_at": "2024-01-01T12:00:00Z",
      "last_activity_at": "2024-01-01T12:25:00Z",
      "status": "active"
    }
  ]
}
```

---

### 8.5. Завершить конференцию

**Endpoint:** `POST /api/v1/admin/conferences/:id/end`

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Ответ:** `200 OK`

```json
{
  "id": "room-uuid",
  "room_id": "room-uuid",
  "ended": true,
  "ended_at": "2024-01-01T12:30:00Z"
}
```

**Ошибки:**
- `400 Bad Request` — пустой/некорректный `id`, либо комната не является meeting-room.
- `404 Not Found` — конференция не найдена.

---

### 8.6. Логи доставок webhook

**Endpoint:** `GET /api/v1/admin/webhooks/deliveries`

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Параметры query:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| limit | number | Количество записей (1..200, default: 50) |

**Ответ:** `200 OK`

```json
{
  "data": [
    {
      "id": "delivery-uuid",
      "webhook_id": "webhook-uuid",
      "response_code": 200,
      "response_body": "ok",
      "success": true,
      "retry_count": 0,
      "delivered_at": "2024-01-01T12:00:00Z",
      "created_at": "2024-01-01T12:00:00Z"
    }
  ]
}
```

---

### 8.7. Ошибки доставок webhook

**Endpoint:** `GET /api/v1/admin/webhooks/errors`

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Параметры query:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| limit | number | Количество записей (1..200, default: 50) |

**Ответ:** `200 OK`

```json
{
  "total": 1,
  "data": [
    {
      "id": "delivery-uuid",
      "webhook_id": "webhook-uuid",
      "response_code": 500,
      "response_body": "dead_letter: status=500 body=failure",
      "success": false,
      "retry_count": 2,
      "delivered_at": "2024-01-01T12:05:00Z",
      "created_at": "2024-01-01T12:05:00Z"
    }
  ]
}
```

---

### 8.8. Ошибки bot-команд

**Endpoint:** `GET /api/v1/admin/bots/errors`

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Параметры query:**
| Параметр | Тип | Описание |
|----------|-----|----------|
| limit | number | Количество записей (1..200, default: 50) |

**Ответ:** `200 OK`

```json
{
  "total": 2,
  "data": [
    {
      "id": "event-uuid",
      "room_id": "room-uuid",
      "user_id": "user-uuid",
      "command": "schedule",
      "args": "meeting Demo at 2030-01-02 15:04",
      "status": "failed",
      "error": "calendar unavailable",
      "created_at": "2024-01-01T12:10:00Z"
    },
    {
      "id": "event-uuid-2",
      "room_id": "room-uuid",
      "user_id": "user-uuid",
      "command": "help",
      "args": "",
      "status": "rate_limited",
      "created_at": "2024-01-01T12:10:01Z"
    }
  ]
}
```

В выборку ошибок включаются статусы:
- `failed`
- `permission_denied`
- `rate_limited`

---

### 8.9. Сводная статистика админ-панели

**Endpoint:** `GET /api/v1/admin/stats`

**Headers:**
```
Authorization: Bearer <admin_token>
```

**Ответ:** `200 OK`

```json
{
  "users": {
    "total": 152
  },
  "rooms": {
    "total": 47
  },
  "conferences": {
    "active": 6
  },
  "messages": {
    "today": 1243
  }
}
```

Примечания:
- `conferences.active` считается по meeting-room (`type = meeting`);
- `messages.today` считается по не удаленным сообщениям, созданным с начала текущих суток.

---

## 9. Health Check API

### 9.1. Health check

**Endpoint:** `GET /health`

**Аутентификация:** Не требуется

**Ответ:** `200 OK`

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 86400
}
```

---

### 9.2. Readiness check

**Endpoint:** `GET /ready`

**Аутентификация:** Не требуется

**Ответ:** `200 OK` или `503 Service Unavailable`

```json
{
  "status": "ready",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "keycloak": "ok",
    "jitsi": "ok"
  }
}
```

---

### 9.3. Liveness check

**Endpoint:** `GET /live`

**Аутентификация:** Не требуется

**Ответ:** `200 OK` или `503 Service Unavailable`

---

## 10. Errors

### 10.1. Формат ошибок

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": [
      {
        "field": "email",
        "message": "Invalid email format"
      }
    ],
    "request_id": "req-uuid"
  }
}
```

### 10.2. Коды ошибок

| Код | HTTP Status | Описание |
|-----|-------------|----------|
| `UNAUTHORIZED` | 401 | Токен отсутствует или истёк |
| `FORBIDDEN` | 403 | Недостаточно прав |
| `NOT_FOUND` | 404 | Ресурс не найден |
| `VALIDATION_ERROR` | 400 | Ошибка валидации |
| `CONFLICT` | 409 | Конфликт (дубликат) |
| `RATE_LIMIT_EXCEEDED` | 429 | Превышен лимит запросов |
| `INTERNAL_ERROR` | 500 | Внутренняя ошибка |
| `SERVICE_UNAVAILABLE` | 503 | Сервис недоступен |

---

## 11. WebSocket API

### 11.1. Подключение

```
wss://api.company.com/api/v1/ws
Authorization: Bearer <session_token>
```

Также поддерживаются query параметры для подключения:
- `?token=<session_token>`
- `?access_token=<session_token>`

Важно:
- при истекшем токене сервер возвращает `401 token_expired`;
- при отозванной API-сессии сервер возвращает `401 session_revoked`;
- при истечении токена во время активной сессии сервер закрывает WebSocket с reason `token_expired`;
- для reconnect клиент должен получить новый токен и переподключиться.

### 11.2. Клиент → Сервер

**Подписка на комнату:**
```json
{
  "type": "subscribe",
  "payload": {
    "room_id": "room-uuid"
  }
}
```

Ограничения доступа:
- подписка на комнату разрешается только участнику комнаты;
- отправка `message` и `typing` разрешается только после успешной подписки на комнату.

**Отправка сообщения:**
```json
{
  "type": "message",
  "payload": {
    "room_id": "room-uuid",
    "content": "Привет!",
    "type": "text"
  }
}
```

**Typing indicator:**
```json
{
  "type": "typing",
  "payload": {
    "room_id": "room-uuid",
    "is_typing": true
  }
}
```

### 11.3. Сервер → Клиент

**Новое сообщение:**
```json
{
  "type": "message",
  "payload": {
    "id": "msg-uuid",
    "room_id": "room-uuid",
    "user": {
      "id": "user-uuid",
      "name": "User Name"
    },
    "content": "Привет!",
    "type": "text",
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

**Пользователь присоединился:**
```json
{
  "type": "user_joined",
  "payload": {
    "room_id": "room-uuid",
    "user": {
      "id": "user-uuid",
      "name": "User Name"
    },
    "joined_at": "2024-01-01T12:00:00Z"
  }
}
```

**Typing status:**
```json
{
  "type": "typing",
  "payload": {
    "room_id": "room-uuid",
    "user": {
      "id": "user-uuid",
      "name": "User Name"
    },
    "is_typing": true
  }
}
```

---

## 12. Приложения

### 12.1. OpenAPI спецификация

Полная спецификация доступна по адресу:
- Swagger UI: `https://api.company.com/swagger`
- JSON: `https://api.company.com/swagger.json`
- YAML: `https://api.company.com/swagger.yaml`

### 12.2. Postman коллекция

Импортировать коллекцию:
```
https://api.company.com/postman/collection.json
```

### 12.3. Ссылки

- [Architecture.md](./Architecture.md)
- [HLD.md](./HLD.md)
- [Security.md](./Security.md)
