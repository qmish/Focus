package push

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"go.uber.org/zap"
)

// TokenRepo — минимальный контракт сервиса к репозиторию push-токенов.
// Используется в т. ч. для unit-тестов с моками.
type TokenRepo interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.PushToken, error)
	ListByUsers(ctx context.Context, userIDs []uuid.UUID) ([]*models.PushToken, error)
	DeleteByEndpoint(ctx context.Context, endpoint string) error
	TouchLastUsed(ctx context.Context, id uuid.UUID) error
}

// Service — фасад для отправки push-уведомлений.
type Service struct {
	repo    TokenRepo
	senders map[models.PushPlatform]Sender
	logger  *zap.Logger
}

// NewService собирает сервис из перечня sender'ов. Платформы, для которых
// sender не передан, будут пропущены.
func NewService(repo TokenRepo, logger *zap.Logger, senders ...Sender) *Service {
	m := make(map[models.PushPlatform]Sender, len(senders))
	for _, s := range senders {
		if s == nil {
			continue
		}
		m[s.Platform()] = s
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repo: repo, senders: m, logger: logger}
}

// HasSender проверяет, поддерживается ли платформа.
func (s *Service) HasSender(p models.PushPlatform) bool {
	_, ok := s.senders[p]
	return ok
}

// NotifyUsers отправляет уведомление всем подпискам перечисленных пользователей.
// Ошибки логируются, но не прерывают рассылку. Устаревшие подписки удаляются.
func (s *Service) NotifyUsers(ctx context.Context, userIDs []uuid.UUID, n *Notification) error {
	if len(userIDs) == 0 {
		return nil
	}
	tokens, err := s.repo.ListByUsers(ctx, userIDs)
	if err != nil {
		return err
	}
	return s.dispatch(ctx, tokens, n)
}

// NotifyUser — удобный helper для одного получателя.
func (s *Service) NotifyUser(ctx context.Context, userID uuid.UUID, n *Notification) error {
	tokens, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	return s.dispatch(ctx, tokens, n)
}

// dispatch отправляет уведомление параллельно по всем токенам и обрабатывает Gone.
func (s *Service) dispatch(ctx context.Context, tokens []*models.PushToken, n *Notification) error {
	if len(tokens) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	for _, t := range tokens {
		sender, ok := s.senders[t.Platform]
		if !ok {
			continue
		}
		wg.Add(1)
		go func(tok *models.PushToken, sn Sender) {
			defer wg.Done()
			err := sn.Send(ctx, tok, n)
			if err == nil {
				if errTouch := s.repo.TouchLastUsed(ctx, tok.ID); errTouch != nil {
					s.logger.Warn("push: touch last_used failed",
						zap.String("endpoint", tok.Endpoint),
						zap.Error(errTouch))
				}
				return
			}
			var sendErr *SendError
			if errors.As(err, &sendErr) && sendErr.IsGone {
				if errDel := s.repo.DeleteByEndpoint(ctx, tok.Endpoint); errDel != nil {
					s.logger.Warn("push: delete expired token failed",
						zap.String("endpoint", tok.Endpoint),
						zap.Error(errDel))
				} else {
					s.logger.Info("push: expired subscription removed",
						zap.String("endpoint", tok.Endpoint))
				}
				return
			}
			s.logger.Warn("push: send failed",
				zap.String("platform", string(tok.Platform)),
				zap.String("endpoint", tok.Endpoint),
				zap.Error(err))
		}(t, sender)
	}
	wg.Wait()
	return nil
}

// Compile-time проверка: *repository.PushTokenRepository удовлетворяет TokenRepo.
var _ TokenRepo = (*repository.PushTokenRepository)(nil)
