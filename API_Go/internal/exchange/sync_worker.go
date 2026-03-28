package exchange

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/pkg/logger"
	"go.uber.org/zap"
)

// SyncWorker periodically synchronizes Exchange calendar events into Focus meeting rooms.
type SyncWorker struct {
	calendar     CalendarService
	userRepo     *repository.UserRepository
	roomRepo     *repository.RoomRepository
	meetingLinks *repository.MeetingLinkRepository
	interval     time.Duration
	lookback     time.Duration
	lookahead    time.Duration
}

func NewSyncWorker(
	calendar CalendarService,
	userRepo *repository.UserRepository,
	roomRepo *repository.RoomRepository,
	meetingLinks *repository.MeetingLinkRepository,
	interval time.Duration,
	lookback time.Duration,
	lookahead time.Duration,
) *SyncWorker {
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	if lookback <= 0 {
		lookback = 12 * time.Hour
	}
	if lookahead <= 0 {
		lookahead = 14 * 24 * time.Hour
	}
	return &SyncWorker{
		calendar:     calendar,
		userRepo:     userRepo,
		roomRepo:     roomRepo,
		meetingLinks: meetingLinks,
		interval:     interval,
		lookback:     lookback,
		lookahead:    lookahead,
	}
}

func (w *SyncWorker) Start(ctx context.Context) {
	if w == nil || w.calendar == nil || w.userRepo == nil || w.roomRepo == nil || w.meetingLinks == nil {
		return
	}
	// Initial sync at startup.
	w.syncOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.syncOnce(ctx)
		}
	}
}

func (w *SyncWorker) syncOnce(ctx context.Context) {
	users, err := w.userRepo.List(ctx, 1000, 0)
	if err != nil {
		logger.Warn("Exchange sync: failed to list users", zap.Error(err))
		return
	}
	now := time.Now().UTC()
	from := now.Add(-w.lookback)
	to := now.Add(w.lookahead)

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for _, user := range users {
		if user == nil || !user.IsActive {
			continue
		}
		email := strings.TrimSpace(user.Email)
		if email == "" {
			continue
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(u *models.User, email string) {
			defer func() { <-sem; wg.Done() }()
			if err := w.syncUserWindow(ctx, u, email, from, to); err != nil {
				logger.Warn("Exchange sync: user sync failed", zap.String("email", email), zap.Error(err))
			}
		}(user, email)
	}
	wg.Wait()
}

func (w *SyncWorker) syncUserWindow(ctx context.Context, user *models.User, email string, from, to time.Time) error {
	events, err := w.calendar.GetEvents(ctx, email, from, to)
	if err != nil {
		return err
	}
	seenEventIDs := make(map[string]struct{}, len(events))
	for _, event := range events {
		eventID := strings.TrimSpace(event.ID)
		if eventID == "" {
			continue
		}
		seenEventIDs[eventID] = struct{}{}
		if err := w.upsertEventLink(ctx, user, email, event); err != nil {
			logger.Warn("Exchange sync: upsert event failed", zap.String("event_id", eventID), zap.Error(err))
		}
	}
	links, err := w.meetingLinks.ListByOrganizerAndWindow(ctx, email, from, to, 2000)
	if err != nil {
		return err
	}
	for _, link := range links {
		if link == nil {
			continue
		}
		if _, ok := seenEventIDs[strings.TrimSpace(link.ExchangeEventID)]; ok {
			continue
		}
		link.Status = "cancelled"
		ts := time.Now().UTC()
		link.LastSyncAt = &ts
		_ = w.meetingLinks.Update(ctx, link)
	}
	return nil
}

func (w *SyncWorker) upsertEventLink(ctx context.Context, user *models.User, organizerEmail string, event CalendarEvent) error {
	link, err := w.meetingLinks.GetByExchangeEventID(ctx, event.ID)
	if err != nil && err != repository.ErrMeetingLinkNotFound {
		return err
	}
	now := time.Now().UTC()
	if err == repository.ErrMeetingLinkNotFound {
		room := models.NewRoom(defaultMeetingName(event.Subject), user.ID, models.RoomTypeMeeting)
		if createErr := w.roomRepo.Create(ctx, room); createErr != nil {
			return createErr
		}
		status := "scheduled"
		if event.EndTime.Before(now) {
			status = "completed"
		}
		newLink := &models.MeetingLink{
			ID:              uuid.New(),
			RoomID:          room.ID,
			ExchangeEventID: event.ID,
			OrganizerEmail:  organizerEmail,
			Subject:         defaultMeetingName(event.Subject),
			StartAt:         event.StartTime.UTC(),
			EndAt:           event.EndTime.UTC(),
			Status:          status,
			SyncSource:      "exchange",
			LastSyncAt:      &now,
		}
		return w.meetingLinks.Create(ctx, newLink)
	}

	// Existing link: update timing and status.
	link.Subject = defaultMeetingName(event.Subject)
	if !event.StartTime.IsZero() {
		link.StartAt = event.StartTime.UTC()
	}
	if !event.EndTime.IsZero() {
		link.EndAt = event.EndTime.UTC()
	}
	if link.EndAt.Before(now) {
		link.Status = "completed"
	} else {
		link.Status = "scheduled"
	}
	link.LastSyncAt = &now
	return w.meetingLinks.Update(ctx, link)
}

func defaultMeetingName(subject string) string {
	trimmed := strings.TrimSpace(subject)
	if trimmed == "" {
		return "Встреча Exchange"
	}
	return trimmed
}

