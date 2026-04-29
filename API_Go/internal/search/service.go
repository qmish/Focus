package search

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"golang.org/x/sync/errgroup"
)

// ErrEmptyQuery — пустой/слишком короткий запрос.
var ErrEmptyQuery = errors.New("search: query is empty")

// MinQueryLen — минимальная длина запроса в рунах.
const MinQueryLen = 2

// Scope — какие категории искать в глобальном поиске.
type Scope struct {
	Users    bool
	Rooms    bool
	Messages bool
	Files    bool
	Meetings bool
}

// DefaultScope — все категории включены.
func DefaultScope() Scope {
	return Scope{Users: true, Rooms: true, Messages: true, Files: true, Meetings: true}
}

// IsEmpty — true, если ни одна категория не выбрана.
func (s Scope) IsEmpty() bool {
	return !s.Users && !s.Rooms && !s.Messages && !s.Files && !s.Meetings
}

// GlobalResult — общий ответ глобального поиска.
type GlobalResult struct {
	Users    []*models.User `json:"users"`
	Rooms    []*models.Room `json:"rooms"`
	Messages []*MessageHit  `json:"messages"`
	Files    []*FileHit     `json:"files"`
	Meetings []*MeetingHit  `json:"meetings"`
}

// Service — фасад поиска с единой точкой ABAC и метрик.
type Service struct {
	provider SearchProvider
}

// NewService создаёт сервис поиска.
func NewService(p SearchProvider) *Service {
	return &Service{provider: p}
}

// CountRunes — подсчёт длины запроса в рунах (используется хендлерами для
// валидации перед хождением в БД).
func CountRunes(q string) int {
	count := 0
	for range q {
		count++
	}
	return count
}

// Global выполняет fan-out по всем выбранным типам.
// Все запросы идут параллельно через errgroup, любая ошибка отменяет ctx.
func (s *Service) Global(ctx context.Context, userID uuid.UUID, query string, scope Scope, limit int) (*GlobalResult, error) {
	query = strings.TrimSpace(query)
	if CountRunes(query) < MinQueryLen {
		return nil, ErrEmptyQuery
	}
	if scope.IsEmpty() {
		scope = DefaultScope()
	}
	res := &GlobalResult{
		Users:    []*models.User{},
		Rooms:    []*models.Room{},
		Messages: []*MessageHit{},
		Files:    []*FileHit{},
		Meetings: []*MeetingHit{},
	}
	g, gctx := errgroup.WithContext(ctx)
	if scope.Users {
		g.Go(func() error {
			users, err := s.provider.SearchUsers(gctx, query, limit)
			if err != nil {
				return err
			}
			res.Users = users
			return nil
		})
	}
	if scope.Rooms {
		g.Go(func() error {
			rooms, err := s.provider.SearchRooms(gctx, userID, query, limit)
			if err != nil {
				return err
			}
			res.Rooms = rooms
			return nil
		})
	}
	if scope.Messages {
		g.Go(func() error {
			msgs, err := s.provider.SearchMessages(gctx, userID, query, nil, MessageSearchOpts{Limit: limit})
			if err != nil {
				return err
			}
			res.Messages = msgs
			return nil
		})
	}
	if scope.Files {
		g.Go(func() error {
			files, err := s.provider.SearchFiles(gctx, userID, query, limit)
			if err != nil {
				return err
			}
			res.Files = files
			return nil
		})
	}
	if scope.Meetings {
		g.Go(func() error {
			meetings, err := s.provider.SearchMeetings(gctx, userID, query, limit)
			if err != nil {
				return err
			}
			res.Meetings = meetings
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return res, nil
}

// LocalMessages — локальный поиск в одной комнате с курсорной пагинацией.
// Проверка членства возложена на провайдер (ABAC).
func (s *Service) LocalMessages(ctx context.Context, userID, roomID uuid.UUID, query string, opts MessageSearchOpts) ([]*MessageHit, error) {
	query = strings.TrimSpace(query)
	if CountRunes(query) < MinQueryLen {
		return nil, ErrEmptyQuery
	}
	return s.provider.SearchMessages(ctx, userID, query, &roomID, opts)
}
