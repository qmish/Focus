package search

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

// PgProvider — реализация SearchProvider поверх PostgreSQL с pg_trgm.
//
// Использует ILIKE %q% запросы; за счёт GIN-индексов с gin_trgm_ops
// (создаются в `internal/database/search_indexes.go`) запросы остаются
// быстрыми на больших таблицах.
type PgProvider struct {
	db             *gorm.DB
	maxQueryLength int
}

// NewPgProvider — конструктор. `db` обязателен.
func NewPgProvider(db *gorm.DB) *PgProvider {
	return &PgProvider{db: db, maxQueryLength: 200}
}

func (p *PgProvider) sanitize(q string) (string, bool) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", false
	}
	if len(q) > p.maxQueryLength {
		q = q[:p.maxQueryLength]
	}
	// Экранируем pattern-специальные символы для LIKE.
	q = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
	return q, true
}

func clampLimit(n, def, max int) int {
	if n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// SearchUsers — глобальный поиск пользователей по name/email.
// На уровне ABAC ничего не отсекаем — список зарегистрированных
// сотрудников доступен любому авторизованному.
func (p *PgProvider) SearchUsers(ctx context.Context, query string, limit int) ([]*models.User, error) {
	q, ok := p.sanitize(query)
	if !ok {
		return []*models.User{}, nil
	}
	limit = clampLimit(limit, 10, 50)
	pattern := "%" + q + "%"
	var users []*models.User
	err := p.db.WithContext(ctx).
		Where("is_active = true AND (name ILIKE ? OR email ILIKE ?)", pattern, pattern).
		Order("name ASC").
		Limit(limit).
		Find(&users).Error
	return users, err
}

// SearchRooms — поиск комнат по имени.
// Возвращает: (а) комнаты, где userID — участник; плюс (б) публичные комнаты,
// видимые всем; soft-deleted комнаты исключаются.
func (p *PgProvider) SearchRooms(ctx context.Context, userID uuid.UUID, query string, limit int) ([]*models.Room, error) {
	q, ok := p.sanitize(query)
	if !ok {
		return []*models.Room{}, nil
	}
	limit = clampLimit(limit, 10, 50)
	pattern := "%" + q + "%"
	var rooms []*models.Room
	// DISTINCT чтобы публичная комната, в которой пользователь ещё и участник,
	// не дублировалась.
	err := p.db.WithContext(ctx).
		Distinct().
		Joins("LEFT JOIN room_participants rp ON rp.room_id = rooms.id AND rp.user_id = ?", userID).
		Where("rooms.deleted_at IS NULL AND rooms.name ILIKE ? AND (rp.user_id IS NOT NULL OR rooms.type = ?)",
			pattern, models.RoomTypePublic).
		Order("rooms.name ASC").
		Limit(limit).
		Find(&rooms).Error
	return rooms, err
}

// SearchMessages — поиск сообщений по содержимому.
// ABAC: только комнаты, где userID — участник (private/public/meeting равны).
// Если задан roomID — ограничиваем им (с проверкой членства).
func (p *PgProvider) SearchMessages(ctx context.Context, userID uuid.UUID, query string, roomID *uuid.UUID, opts MessageSearchOpts) ([]*MessageHit, error) {
	q, ok := p.sanitize(query)
	if !ok {
		return []*MessageHit{}, nil
	}
	limit := clampLimit(opts.Limit, 20, 100)
	pattern := "%" + q + "%"

	type row struct {
		models.Message
		RoomName string `gorm:"column:room_name"`
	}
	var rows []row

	tx := p.db.WithContext(ctx).
		Table("messages AS m").
		Select("m.*, rooms.name AS room_name").
		Joins("JOIN room_participants rp ON rp.room_id = m.room_id AND rp.user_id = ?", userID).
		Joins("JOIN rooms ON rooms.id = m.room_id AND rooms.deleted_at IS NULL").
		Where("m.is_deleted = false AND m.content ILIKE ?", pattern)

	if roomID != nil {
		tx = tx.Where("m.room_id = ?", *roomID)
	}

	if opts.Before != nil {
		// Курсорная пагинация: сообщения старше указанного.
		tx = tx.Where("m.created_at < (SELECT created_at FROM messages WHERE id = ?)", *opts.Before)
	}

	err := tx.Order("m.created_at DESC").Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]*MessageHit, 0, len(rows))
	for i := range rows {
		msg := rows[i].Message
		out = append(out, &MessageHit{
			Message:   &msg,
			RoomID:    msg.RoomID,
			RoomName:  rows[i].RoomName,
			Highlight: HighlightSnippet(msg.Content, query, 160),
		})
	}
	return out, nil
}

// SearchFiles — поиск вложений по имени файла из Metadata.
// ABAC: те же правила, что и для сообщений.
func (p *PgProvider) SearchFiles(ctx context.Context, userID uuid.UUID, query string, limit int) ([]*FileHit, error) {
	q, ok := p.sanitize(query)
	if !ok {
		return []*FileHit{}, nil
	}
	limit = clampLimit(limit, 20, 100)
	pattern := "%" + q + "%"

	type row struct {
		MessageID uuid.UUID       `gorm:"column:message_id"`
		RoomID    uuid.UUID       `gorm:"column:room_id"`
		RoomName  string          `gorm:"column:room_name"`
		Type      string          `gorm:"column:type"`
		Metadata  models.Metadata `gorm:"column:metadata"`
		CreatedAt time.Time       `gorm:"column:created_at"`
	}
	var rows []row

	err := p.db.WithContext(ctx).
		Table("messages AS m").
		Select("m.id AS message_id, m.room_id, m.type, m.metadata, m.created_at, rooms.name AS room_name").
		Joins("JOIN room_participants rp ON rp.room_id = m.room_id AND rp.user_id = ?", userID).
		Joins("JOIN rooms ON rooms.id = m.room_id AND rooms.deleted_at IS NULL").
		Where("m.is_deleted = false AND m.type IN ?", []string{string(models.MessageTypeFile), string(models.MessageTypeImage)}).
		Where("(m.metadata->>'file_name') ILIKE ?", pattern).
		Order("m.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]*FileHit, 0, len(rows))
	for _, r := range rows {
		out = append(out, &FileHit{
			MessageID:  r.MessageID,
			RoomID:     r.RoomID,
			RoomName:   r.RoomName,
			FileID:     r.Metadata.FileID,
			FileName:   r.Metadata.FileName,
			FileMIME:   r.Metadata.FileMIME,
			FileSize:   r.Metadata.FileSize,
			UploadedAt: r.CreatedAt.Format(time.RFC3339),
			Type:       r.Type,
		})
	}
	return out, nil
}

// SearchMeetings — поиск запланированных встреч по subject (MeetingLink).
// ABAC: возвращаем только встречи в комнатах, где user — участник, либо где
// он организатор (по email).
func (p *PgProvider) SearchMeetings(ctx context.Context, userID uuid.UUID, query string, limit int) ([]*MeetingHit, error) {
	q, ok := p.sanitize(query)
	if !ok {
		return []*MeetingHit{}, nil
	}
	limit = clampLimit(limit, 10, 50)
	pattern := "%" + q + "%"

	type row struct {
		ID             uuid.UUID `gorm:"column:id"`
		RoomID         uuid.UUID `gorm:"column:room_id"`
		RoomName       string    `gorm:"column:room_name"`
		Subject        string    `gorm:"column:subject"`
		OrganizerEmail string    `gorm:"column:organizer_email"`
		StartAt        time.Time `gorm:"column:start_at"`
		EndAt          time.Time `gorm:"column:end_at"`
		Status         string    `gorm:"column:status"`
	}
	var rows []row

	err := p.db.WithContext(ctx).
		Table("meeting_links AS ml").
		Select("ml.id, ml.room_id, ml.subject, ml.organizer_email, ml.start_at, ml.end_at, ml.status, rooms.name AS room_name").
		Joins("JOIN rooms ON rooms.id = ml.room_id AND rooms.deleted_at IS NULL").
		Joins("LEFT JOIN room_participants rp ON rp.room_id = ml.room_id AND rp.user_id = ?", userID).
		Joins("LEFT JOIN users u ON u.id = ?", userID).
		Where("ml.subject ILIKE ?", pattern).
		Where("rp.user_id IS NOT NULL OR ml.organizer_email = u.email").
		Order("ml.start_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]*MeetingHit, 0, len(rows))
	for _, r := range rows {
		out = append(out, &MeetingHit{
			ID:             r.ID,
			RoomID:         r.RoomID,
			RoomName:       r.RoomName,
			Subject:        r.Subject,
			OrganizerEmail: r.OrganizerEmail,
			StartAt:        r.StartAt.Format(time.RFC3339),
			EndAt:          r.EndAt.Format(time.RFC3339),
			Status:         r.Status,
		})
	}
	return out, nil
}

// Compile-time проверка, что *PgProvider удовлетворяет SearchProvider.
var _ SearchProvider = (*PgProvider)(nil)
