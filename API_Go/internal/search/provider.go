// Package search реализует pluggable полнотекстовый поиск Focus.
//
// Текущая реализация — `PgProvider` на pg_trgm + GIN. Контракт `SearchProvider`
// заранее совместим с будущей реализацией поверх Meilisearch / OpenSearch.
package search

import (
	"context"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
)

// MessageSearchOpts — параметры локального и глобального поиска по сообщениям.
type MessageSearchOpts struct {
	Limit  int
	Before *uuid.UUID // курсор: вернуть сообщения старше этого id
}

// MessageHit — найденное сообщение с минимальным контекстом для UI.
type MessageHit struct {
	Message   *models.Message `json:"message"`
	RoomID    uuid.UUID       `json:"room_id"`
	RoomName  string          `json:"room_name"`
	Highlight string          `json:"highlight,omitempty"`
}

// FileHit — найденное вложение (через сообщение типа image/file).
type FileHit struct {
	MessageID  uuid.UUID `json:"message_id"`
	RoomID     uuid.UUID `json:"room_id"`
	RoomName   string    `json:"room_name"`
	FileID     string    `json:"file_id"`
	FileName   string    `json:"file_name"`
	FileMIME   string    `json:"file_mime,omitempty"`
	FileSize   int64     `json:"file_size,omitempty"`
	UploadedAt string    `json:"uploaded_at"`
	Type       string    `json:"type"`
}

// MeetingHit — найденная встреча (MeetingLink + Room).
type MeetingHit struct {
	ID             uuid.UUID `json:"id"`
	RoomID         uuid.UUID `json:"room_id"`
	RoomName       string    `json:"room_name"`
	Subject        string    `json:"subject"`
	OrganizerEmail string    `json:"organizer_email"`
	StartAt        string    `json:"start_at"`
	EndAt          string    `json:"end_at"`
	Status         string    `json:"status"`
}

// SearchProvider — единый контракт всех бэкендов поиска.
//
// Все методы получают `userID` для ABAC-фильтрации (видны только те объекты,
// к которым у пользователя есть доступ). Реализации не должны делать
// HTTP-запросов к самим себе — только на свои индексы / БД.
type SearchProvider interface {
	SearchUsers(ctx context.Context, query string, limit int) ([]*models.User, error)
	SearchRooms(ctx context.Context, userID uuid.UUID, query string, limit int) ([]*models.Room, error)
	SearchMessages(ctx context.Context, userID uuid.UUID, query string, roomID *uuid.UUID, opts MessageSearchOpts) ([]*MessageHit, error)
	SearchFiles(ctx context.Context, userID uuid.UUID, query string, limit int) ([]*FileHit, error)
	SearchMeetings(ctx context.Context, userID uuid.UUID, query string, limit int) ([]*MeetingHit, error)
}
