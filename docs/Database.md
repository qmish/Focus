# Database Design

**Версия:** 1.0  
**Дата:** 24 марта 2026 г.  
**Статус:** Черновик

---

## 1. ER-диаграмма

```mermaid
erDiagram
    USERS ||--o{ ROOMS : creates
    USERS ||--o{ MESSAGES : sends
    USERS ||--o{ ROOM_PARTICIPANTS : joins
    USERS ||--o{ MEETINGS : organizes
    USERS ||--o{ USER_SESSIONS : has
    USERS ||--o{ WEBHOOKS : owns
    USERS ||--o{ BOTS : creates

    ROOMS ||--o{ MESSAGES : contains
    ROOMS ||--o{ ROOM_PARTICIPANTS : has
    ROOMS ||--o{ MEETINGS : has
    ROOMS ||--o{ ROOM_BOTS : has

    MESSAGES ||--o{ MESSAGE_REACTIONS : has
    MESSAGES ||--o{ MESSAGE_EDITS : has

    MEETINGS ||--o{ MEETING_ATTENDEES : has
    MEETINGS ||--o{ MEETING_RECORDINGS : has

    BOTS ||--o{ BOT_COMMANDS : defines
    BOTS ||--o{ BOT_LOGS : generates

    WEBHOOKS ||--o{ WEBHOOK_DELIVERIES : has

    USERS {
        uuid id PK
        uuid keycloak_id UK
        varchar email UK
        varchar name
        varchar avatar_url
        varchar[] roles
        boolean is_active
        timestamp created_at
        timestamp updated_at
        timestamp last_login_at
    }

    ROOMS {
        uuid id PK
        varchar name
        text description
        uuid creator_id FK
        varchar type
        varchar jitsi_room_name UK
        boolean is_private
        jsonb settings
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    ROOM_PARTICIPANTS {
        uuid room_id PK,FK
        uuid user_id PK,FK
        varchar role
        timestamp joined_at
        timestamp last_read_at
        timestamp left_at
    }

    MESSAGES {
        uuid id PK
        uuid room_id FK
        uuid user_id FK
        text content
        varchar type
        uuid reply_to_id FK
        jsonb metadata
        boolean is_deleted
        timestamp created_at
        timestamp updated_at
    }

    MESSAGE_REACTIONS {
        uuid id PK
        uuid message_id FK
        uuid user_id FK
        varchar emoji
        timestamp created_at
    }

    MESSAGE_EDITS {
        uuid id PK
        uuid message_id FK
        text old_content
        text new_content
        uuid edited_by FK
        timestamp edited_at
    }

    MEETINGS {
        uuid id PK
        uuid room_id FK
        uuid organizer_id FK
        varchar exchange_event_id
        varchar subject
        text description
        timestamp start_time
        timestamp end_time
        varchar jitsi_url
        varchar location
        varchar status
        jsonb recurrence
        timestamp created_at
        timestamp updated_at
    }

    MEETING_ATTENDEES {
        uuid meeting_id PK,FK
        varchar email PK
        varchar name
        varchar response_status
        timestamp responded_at
    }

    MEETING_RECORDINGS {
        uuid id PK
        uuid meeting_id FK
        varchar recording_url
        varchar recording_id
        bigint duration_seconds
        bigint size_bytes
        varchar status
        timestamp recorded_at
        timestamp created_at
    }

    USER_SESSIONS {
        uuid id PK
        uuid user_id FK
        varchar refresh_token_hash
        varchar ip_address
        varchar user_agent
        timestamp expires_at
        timestamp created_at
        timestamp last_used_at
    }

    BOTS {
        uuid id PK
        varchar name UK
        varchar description
        uuid owner_id FK
        varchar token_hash UK
        varchar avatar_url
        jsonb config
        boolean is_enabled
        timestamp created_at
        timestamp updated_at
    }

    BOT_COMMANDS {
        uuid id PK
        uuid bot_id FK
        varchar command
        varchar handler
        jsonb config
        varchar description
        boolean is_enabled
        timestamp created_at
    }

    BOT_LOGS {
        uuid id PK
        uuid bot_id FK
        uuid room_id FK
        uuid user_id FK
        varchar command
        text input
        text output
        varchar status
        text error_message
        bigint execution_time_ms
        timestamp executed_at
    }

    WEBHOOKS {
        uuid id PK
        uuid owner_id FK
        varchar url
        varchar secret
        varchar[] event_types
        boolean is_enabled
        varchar signature_algorithm
        timestamp created_at
        timestamp updated_at
    }

    WEBHOOK_DELIVERIES {
        uuid id PK
        uuid webhook_id FK
        jsonb payload
        jsonb response_headers
        text response_body
        int status_code
        bigint response_time_ms
        int retry_count
        varchar status
        timestamp scheduled_at
        timestamp delivered_at
    }

    ROOM_BOTS {
        uuid room_id PK,FK
        uuid bot_id PK,FK
        jsonb config
        timestamp added_at
    }

    AUDIT_LOGS {
        uuid id PK
        uuid user_id FK
        varchar action
        varchar resource_type
        uuid resource_id
        jsonb old_values
        jsonb new_values
        varchar ip_address
        varchar user_agent
        timestamp created_at
    }
```

---

## 2. Схема базы данных

### 2.1. Расширения PostgreSQL

```sql
-- migrations/000_extensions.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
```

### 2.2. Таблица: users

```sql
-- migrations/001_users.sql
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    keycloak_id uuid NOT NULL UNIQUE,
    email citext NOT NULL UNIQUE,
    name varchar(255) NOT NULL,
    avatar_url varchar(512),
    roles varchar(50)[] DEFAULT ARRAY['user'],
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz
);

CREATE INDEX idx_users_keycloak_id ON users(keycloak_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_roles ON users USING GIN(roles);
CREATE INDEX idx_users_is_active ON users(is_active) WHERE is_active = false;

-- Триггер для updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 2.3. Таблица: rooms

```sql
-- migrations/002_rooms.sql
CREATE TYPE room_type AS ENUM ('public', 'private', 'meeting');

CREATE TABLE rooms (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name varchar(100) NOT NULL,
    description text,
    creator_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type room_type NOT NULL DEFAULT 'public',
    jitsi_room_name varchar(100) NOT NULL UNIQUE,
    is_private boolean NOT NULL DEFAULT false,
    settings jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE INDEX idx_rooms_creator_id ON rooms(creator_id);
CREATE INDEX idx_rooms_type ON rooms(type);
CREATE INDEX idx_rooms_is_private ON rooms(is_private);
CREATE INDEX idx_rooms_deleted_at ON rooms(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_rooms_name_search ON rooms USING GIN(name gin_trgm_ops);

-- Триггер для updated_at
CREATE TRIGGER rooms_updated_at
    BEFORE UPDATE ON rooms
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 2.4. Таблица: room_participants

```sql
-- migrations/003_room_participants.sql
CREATE TYPE participant_role AS ENUM ('member', 'moderator', 'admin');

CREATE TABLE room_participants (
    room_id uuid NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role participant_role NOT NULL DEFAULT 'member',
    joined_at timestamptz NOT NULL DEFAULT now(),
    last_read_at timestamptz,
    left_at timestamptz,
    PRIMARY KEY (room_id, user_id)
);

CREATE INDEX idx_room_participants_user_id ON room_participants(user_id);
CREATE INDEX idx_room_participants_role ON room_participants(role);
CREATE INDEX idx_room_participants_active 
    ON room_participants(room_id, user_id) 
    WHERE left_at IS NULL;
```

### 2.5. Таблица: messages

```sql
-- migrations/004_messages.sql
CREATE TYPE message_type AS ENUM ('text', 'image', 'file', 'system', 'meeting');

CREATE TABLE messages (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id uuid NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    content text NOT NULL,
    type message_type NOT NULL DEFAULT 'text',
    reply_to_id uuid REFERENCES messages(id) ON DELETE SET NULL,
    metadata jsonb NOT NULL DEFAULT '{}',
    is_deleted boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_room_id ON messages(room_id);
CREATE INDEX idx_messages_user_id ON messages(user_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_messages_room_created ON messages(room_id, created_at DESC);
CREATE INDEX idx_messages_type ON messages(type);
CREATE INDEX idx_messages_reply_to ON messages(reply_to_id) WHERE reply_to_id IS NOT NULL;
CREATE INDEX idx_messages_content_search ON messages USING GIN(content gin_trgm_ops);

-- Триггер для updated_at
CREATE TRIGGER messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 2.6. Таблица: message_reactions

```sql
-- migrations/005_message_reactions.sql
CREATE TABLE message_reactions (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id uuid NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji varchar(50) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(message_id, user_id, emoji)
);

CREATE INDEX idx_message_reactions_message_id ON message_reactions(message_id);
CREATE INDEX idx_message_reactions_user_id ON message_reactions(user_id);
CREATE INDEX idx_message_reactions_emoji ON message_reactions(emoji);
```

### 2.7. Таблица: message_edits

```sql
-- migrations/006_message_edits.sql
CREATE TABLE message_edits (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id uuid NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    old_content text NOT NULL,
    new_content text NOT NULL,
    edited_by uuid REFERENCES users(id) ON DELETE SET NULL,
    edited_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_message_edits_message_id ON message_edits(message_id);
CREATE INDEX idx_message_edits_edited_at ON message_edits(edited_at);
```

### 2.8. Таблица: meetings

```sql
-- migrations/007_meetings.sql
CREATE TYPE meeting_status AS ENUM ('scheduled', 'in_progress', 'completed', 'cancelled', 'rescheduled');

CREATE TABLE meetings (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_id uuid REFERENCES rooms(id) ON DELETE SET NULL,
    organizer_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange_event_id varchar(255),
    subject varchar(255) NOT NULL,
    description text,
    start_time timestamptz NOT NULL,
    end_time timestamptz NOT NULL,
    jitsi_url varchar(512),
    location varchar(255),
    status meeting_status NOT NULL DEFAULT 'scheduled',
    recurrence jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_meeting_times CHECK (end_time > start_time)
);

CREATE INDEX idx_meetings_room_id ON meetings(room_id);
CREATE INDEX idx_meetings_organizer_id ON meetings(organizer_id);
CREATE INDEX idx_meetings_exchange_event_id ON meetings(exchange_event_id) WHERE exchange_event_id IS NOT NULL;
CREATE INDEX idx_meetings_start_time ON meetings(start_time);
CREATE INDEX idx_meetings_status ON meetings(status);
CREATE INDEX idx_meetings_upcoming 
    ON meetings(start_time) 
    WHERE status = 'scheduled' AND start_time > now();

-- Триггер для updated_at
CREATE TRIGGER meetings_updated_at
    BEFORE UPDATE ON meetings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 2.9. Таблица: meeting_attendees

```sql
-- migrations/008_meeting_attendees.sql
CREATE TYPE attendee_status AS ENUM ('pending', 'accepted', 'declined', 'tentative');

CREATE TABLE meeting_attendees (
    meeting_id uuid NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    email citext NOT NULL,
    name varchar(255),
    response_status attendee_status NOT NULL DEFAULT 'pending',
    responded_at timestamptz,
    PRIMARY KEY (meeting_id, email)
);

CREATE INDEX idx_meeting_attendees_email ON meeting_attendees(email);
CREATE INDEX idx_meeting_attendees_status ON meeting_attendees(response_status);
```

### 2.10. Таблица: meeting_recordings

```sql
-- migrations/009_meeting_recordings.sql
CREATE TYPE recording_status AS ENUM ('processing', 'ready', 'failed', 'deleted');

CREATE TABLE meeting_recordings (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    meeting_id uuid NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    recording_url varchar(512) NOT NULL,
    recording_id varchar(255) NOT NULL,
    duration_seconds bigint,
    size_bytes bigint,
    status recording_status NOT NULL DEFAULT 'processing',
    recorded_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_meeting_recordings_meeting_id ON meeting_recordings(meeting_id);
CREATE INDEX idx_meeting_recordings_status ON meeting_recordings(status);
CREATE INDEX idx_meeting_recordings_recorded_at ON meeting_recordings(recorded_at);
```

### 2.11. Таблица: user_sessions

```sql
-- migrations/010_user_sessions.sql
CREATE TABLE user_sessions (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash varchar(255) NOT NULL,
    ip_address inet,
    user_agent text,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_refresh_token_hash ON user_sessions(refresh_token_hash);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);
CREATE INDEX idx_user_sessions_active 
    ON user_sessions(user_id, expires_at) 
    WHERE expires_at > now();
```

### 2.12. Таблица: bots

```sql
-- migrations/011_bots.sql
CREATE TABLE bots (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name varchar(100) NOT NULL UNIQUE,
    description text,
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash varchar(255) NOT NULL UNIQUE,
    avatar_url varchar(512),
    config jsonb NOT NULL DEFAULT '{}',
    is_enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_bots_owner_id ON bots(owner_id);
CREATE INDEX idx_bots_is_enabled ON bots(is_enabled) WHERE is_enabled = true;

-- Триггер для updated_at
CREATE TRIGGER bots_updated_at
    BEFORE UPDATE ON bots
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 2.13. Таблица: bot_commands

```sql
-- migrations/012_bot_commands.sql
CREATE TABLE bot_commands (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    bot_id uuid NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    command varchar(50) NOT NULL,
    handler varchar(100) NOT NULL,
    config jsonb NOT NULL DEFAULT '{}',
    description text,
    is_enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(bot_id, command)
);

CREATE INDEX idx_bot_commands_bot_id ON bot_commands(bot_id);
CREATE INDEX idx_bot_commands_command ON bot_commands(command);
CREATE INDEX idx_bot_commands_enabled ON bot_commands(bot_id, is_enabled) WHERE is_enabled = true;
```

### 2.14. Таблица: bot_logs

```sql
-- migrations/013_bot_logs.sql
CREATE TABLE bot_logs (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    bot_id uuid NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    room_id uuid REFERENCES rooms(id) ON DELETE SET NULL,
    user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    command varchar(50) NOT NULL,
    input text,
    output text,
    status varchar(50) NOT NULL,
    error_message text,
    execution_time_ms bigint,
    executed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_bot_logs_bot_id ON bot_logs(bot_id);
CREATE INDEX idx_bot_logs_room_id ON bot_logs(room_id);
CREATE INDEX idx_bot_logs_user_id ON bot_logs(user_id);
CREATE INDEX idx_bot_logs_command ON bot_logs(command);
CREATE INDEX idx_bot_logs_executed_at ON bot_logs(executed_at);
CREATE INDEX idx_bot_logs_status ON bot_logs(status);

-- Partitioning by date for large volumes
-- CREATE TABLE bot_logs_2024_01 PARTITION OF bot_logs
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

### 2.15. Таблица: webhooks

```sql
-- migrations/014_webhooks.sql
CREATE TABLE webhooks (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url varchar(512) NOT NULL,
    secret varchar(255) NOT NULL,
    event_types varchar(100)[] NOT NULL,
    is_enabled boolean NOT NULL DEFAULT true,
    signature_algorithm varchar(50) NOT NULL DEFAULT 'sha256',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_webhooks_owner_id ON webhooks(owner_id);
CREATE INDEX idx_webhooks_enabled ON webhooks(is_enabled) WHERE is_enabled = true;
CREATE INDEX idx_webhooks_event_types ON webhooks USING GIN(event_types);

-- Триггер для updated_at
CREATE TRIGGER webhooks_updated_at
    BEFORE UPDATE ON webhooks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 2.16. Таблица: webhook_deliveries

```sql
-- migrations/015_webhook_deliveries.sql
CREATE TYPE webhook_status AS ENUM ('pending', 'delivered', 'failed', 'retrying');

CREATE TABLE webhook_deliveries (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    webhook_id uuid NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    payload jsonb NOT NULL,
    response_headers jsonb,
    response_body text,
    status_code int,
    response_time_ms bigint,
    retry_count int NOT NULL DEFAULT 0,
    status webhook_status NOT NULL DEFAULT 'pending',
    scheduled_at timestamptz NOT NULL DEFAULT now(),
    delivered_at timestamptz
);

CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX idx_webhook_deliveries_status ON webhook_deliveries(status);
CREATE INDEX idx_webhook_deliveries_scheduled_at ON webhook_deliveries(scheduled_at);
CREATE INDEX idx_webhook_deliveries_pending 
    ON webhook_deliveries(scheduled_at) 
    WHERE status = 'pending' AND scheduled_at <= now();
```

### 2.17. Таблица: room_bots

```sql
-- migrations/016_room_bots.sql
CREATE TABLE room_bots (
    room_id uuid NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    bot_id uuid NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    config jsonb NOT NULL DEFAULT '{}',
    added_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, bot_id)
);

CREATE INDEX idx_room_bots_bot_id ON room_bots(bot_id);
```

### 2.18. Таблица: audit_logs

```sql
-- migrations/017_audit_logs.sql
CREATE TABLE audit_logs (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    action varchar(100) NOT NULL,
    resource_type varchar(100) NOT NULL,
    resource_id uuid,
    old_values jsonb,
    new_values jsonb,
    ip_address inet,
    user_agent text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_ip_address ON audit_logs(ip_address);

-- Partitioning by date
-- CREATE TABLE audit_logs_2024_01 PARTITION OF audit_logs
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

---

## 3. GORM модели

### 3.1. User модель

```go
// internal/models/user.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID           uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
    KeycloakID   uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"keycloak_id"`
    Email        string     `gorm:"type:citext;uniqueIndex;not null" json:"email"`
    Name         string     `gorm:"type:varchar(255);not null" json:"name"`
    AvatarURL    string     `gorm:"type:varchar(512)" json:"avatar_url"`
    Roles        []string   `gorm:"type:varchar(50)[]" json:"roles"`
    IsActive     bool       `gorm:"not null;default:true" json:"is_active"`
    LastLoginAt  *time.Time `json:"last_login_at"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
    
    Rooms        []Room     `gorm:"foreignKey:CreatorID" json:"rooms"`
    Messages     []Message  `gorm:"foreignKey:UserID" json:"messages"`
    Sessions     []UserSession `gorm:"foreignKey:UserID" json:"sessions"`
}

func (User) TableName() string {
    return "users"
}
```

### 3.2. Room модель

```go
// internal/models/room.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type RoomType string

const (
    RoomTypePublic   RoomType = "public"
    RoomTypePrivate  RoomType = "private"
    RoomTypeMeeting  RoomType = "meeting"
)

type RoomSettings struct {
    AllowGuests              bool `json:"allow_guests"`
    RequireModeratorForMsgs  bool `json:"require_moderator_for_messages"`
    MaxParticipants          int  `json:"max_participants"`
}

type Room struct {
    ID              uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
    Name            string         `gorm:"type:varchar(100);not null" json:"name"`
    Description     string         `gorm:"type:text" json:"description"`
    CreatorID       uuid.UUID      `gorm:"type:uuid;not null" json:"creator_id"`
    Type            RoomType       `gorm:"type:room_type;not null" json:"type"`
    JitsiRoomName   string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"jitsi_room_name"`
    IsPrivate       bool           `gorm:"not null;default:false" json:"is_private"`
    Settings        RoomSettings   `gorm:"type:jsonb" json:"settings"`
    DeletedAt       *time.Time     `gorm:"index" json:"-"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`
    
    Creator         User           `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
    Participants    []RoomParticipant `gorm:"foreignKey:RoomID" json:"participants,omitempty"`
    Messages        []Message      `gorm:"foreignKey:RoomID" json:"-"`
    Meetings        []Meeting      `gorm:"foreignKey:RoomID" json:"meetings,omitempty"`
}

func (Room) TableName() string {
    return "rooms"
}
```

### 3.3. Message модель

```go
// internal/models/message.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type MessageType string

const (
    MessageTypeText    MessageType = "text"
    MessageTypeImage   MessageType = "image"
    MessageTypeFile    MessageType = "file"
    MessageTypeSystem  MessageType = "system"
    MessageTypeMeeting MessageType = "meeting"
)

type MessageMetadata struct {
    Edited      bool              `json:"edited,omitempty"`
    EditedAt    *time.Time        `json:"edited_at,omitempty"`
    ReplyTo     *uuid.UUID        `json:"reply_to,omitempty"`
    Reactions   []ReactionSummary `json:"reactions,omitempty"`
    Attachments []Attachment      `json:"attachments,omitempty"`
}

type ReactionSummary struct {
    Emoji string   `json:"emoji"`
    Count int      `json:"count"`
    Users []string `json:"users"`
}

type Attachment struct {
    URL       string `json:"url"`
    Name      string `json:"name"`
    Size      int64  `json:"size"`
    MimeType  string `json:"mime_type"`
}

type Message struct {
    ID         uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
    RoomID     uuid.UUID      `gorm:"type:uuid;not null;index:idx_room_created" json:"room_id"`
    UserID     uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
    Content    string         `gorm:"type:text;not null" json:"content"`
    Type       MessageType    `gorm:"type:message_type;not null" json:"type"`
    ReplyToID  *uuid.UUID     `gorm:"type:uuid" json:"reply_to_id"`
    Metadata   MessageMetadata `gorm:"type:jsonb" json:"metadata"`
    IsDeleted  bool           `gorm:"not null;default:false" json:"is_deleted"`
    CreatedAt  time.Time      `gorm:"index:idx_room_created" json:"created_at"`
    UpdatedAt  time.Time      `json:"updated_at"`
    
    Room       Room           `gorm:"foreignKey:RoomID" json:"room,omitempty"`
    User       User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
    ReplyTo    *Message       `gorm:"foreignKey:ReplyToID" json:"reply_to,omitempty"`
    Reactions  []MessageReaction `gorm:"foreignKey:MessageID" json:"reactions,omitempty"`
}

func (Message) TableName() string {
    return "messages"
}
```

---

## 4. Миграции

### 4.1. Запуск миграций

```go
// cmd/server/main.go
func runMigrations(db *gorm.DB) error {
    migrations := []interface{}{
        &models.User{},
        &models.Room{},
        &models.RoomParticipant{},
        &models.Message{},
        &models.MessageReaction{},
        &models.Meeting{},
        &models.MeetingAttendee{},
        &models.Bot{},
        &models.BotCommand{},
        &models.Webhook{},
        &models.AuditLog{},
    }
    
    for _, model := range migrations {
        if err := db.AutoMigrate(model); err != nil {
            return fmt.Errorf("migration failed for %T: %w", model, err)
        }
    }
    
    return nil
}
```

### 4.2. SQL миграции (golang-migrate)

```bash
# Установка
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Создать миграцию
migrate create -ext sql -dir migrations -seq create_users_table

# Применить миграции
migrate -path migrations -database "postgres://user:pass@localhost:5432/db?sslmode=disable" up
```

---

## 5. Индексы и оптимизация

### 5.1. Составные индексы

```sql
-- Для быстрого получения истории сообщений
CREATE INDEX idx_messages_room_user_created 
    ON messages(room_id, user_id, created_at DESC);

-- Для поиска активных сессий
CREATE INDEX idx_sessions_user_active 
    ON user_sessions(user_id, expires_at) 
    WHERE expires_at > now();

-- Для поиска предстоящих встреч
CREATE INDEX idx_meetings_upcoming 
    ON meetings(organizer_id, start_time) 
    WHERE status = 'scheduled' AND start_time > now();
```

### 5.2. Частичные индексы

```sql
-- Только активные комнаты
CREATE INDEX idx_rooms_active 
    ON rooms(creator_id, type) 
    WHERE deleted_at IS NULL;

-- Только включённые боты
CREATE INDEX idx_bots_active 
    ON bots(owner_id, name) 
    WHERE is_enabled = true;

-- Только неудалённые сообщения
CREATE INDEX idx_messages_not_deleted 
    ON messages(room_id, created_at DESC) 
    WHERE is_deleted = false;
```

### 5.3. Анализ запросов

```sql
-- Включить отслеживание медленных запросов
ALTER SYSTEM SET log_min_duration_statement = 1000; -- 1 секунда

-- Просмотр статистики использования индексов
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Поиск таблиц без первичных ключей
SELECT 
    schemaname,
    tablename
FROM pg_tables
WHERE schemaname = 'public'
EXCEPT
SELECT 
    n.nspname,
    c.relname
FROM pg_index i
JOIN pg_class c ON c.oid = i.indrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE i.indisprimary;
```

---

## 6. Резервное копирование

### 6.1. pg_dump

```bash
# Полный дамп
pg_dump -h localhost -U messenger -d messenger > backup.sql

# Дамп в формате custom (сжатый)
pg_dump -h localhost -U messenger -d messenger -Fc > backup.dump

# Только схема
pg_dump -h localhost -U messenger -d messenger -s > schema.sql

# Только данные
pg_dump -h localhost -U messenger -d messenger -a > data.sql
```

### 6.2. Восстановление

```bash
# Из SQL файла
psql -h localhost -U messenger -d messenger < backup.sql

# Из custom формата
pg_restore -h localhost -U messenger -d messenger backup.dump
```

### 6.3. Непрерывное архивирование (WAL)

```postgresql
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'cp %p /var/lib/postgresql/wal_archive/%f'
archive_timeout = 300  # 5 минут
```

---

## 7. Приложения

### 7.1. Ссылки

- [Architecture.md](./Architecture.md)
- [HLD.md](./HLD.md)
- [API.md](./API.md)

### 7.2. Инструменты

- [pgAdmin](https://www.pgadmin.org/) — GUI для PostgreSQL
- [DBeaver](https://dbeaver.io/) — Универсальный DB клиент
- [pgMustard](https://www.pgmustard.com/) — Анализ EXPLAIN
