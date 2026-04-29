package push

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	mu             sync.Mutex
	tokens         []*models.PushToken
	deletedEndpts  []string
	touched        []uuid.UUID
	listByUserErr  error
	listByUsersErr error
}

func (m *mockRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*models.PushToken, error) {
	if m.listByUserErr != nil {
		return nil, m.listByUserErr
	}
	out := []*models.PushToken{}
	for _, t := range m.tokens {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}
func (m *mockRepo) ListByUsers(_ context.Context, userIDs []uuid.UUID) ([]*models.PushToken, error) {
	if m.listByUsersErr != nil {
		return nil, m.listByUsersErr
	}
	set := map[uuid.UUID]struct{}{}
	for _, u := range userIDs {
		set[u] = struct{}{}
	}
	out := []*models.PushToken{}
	for _, t := range m.tokens {
		if _, ok := set[t.UserID]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}
func (m *mockRepo) DeleteByEndpoint(_ context.Context, endpoint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedEndpts = append(m.deletedEndpts, endpoint)
	return nil
}
func (m *mockRepo) TouchLastUsed(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touched = append(m.touched, id)
	return nil
}

func (m *mockRepo) snapshot() (touched []uuid.UUID, deleted []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	touched = append([]uuid.UUID(nil), m.touched...)
	deleted = append([]string(nil), m.deletedEndpts...)
	return
}

type goneSender struct{ count atomic.Int32 }

func (g *goneSender) Platform() models.PushPlatform { return models.PushPlatformWeb }
func (g *goneSender) Send(_ context.Context, t *models.PushToken, _ *Notification) error {
	g.count.Add(1)
	return &SendError{Endpoint: t.Endpoint, IsGone: true, Err: errors.New("gone")}
}

func TestService_NotifyUser_DispatchesAndTouches(t *testing.T) {
	uid := uuid.New()
	repo := &mockRepo{
		tokens: []*models.PushToken{
			{ID: uuid.New(), UserID: uid, Platform: models.PushPlatformWeb, Endpoint: "e1"},
			{ID: uuid.New(), UserID: uid, Platform: models.PushPlatformFCM, Endpoint: "e2"},
		},
	}
	web := NewNoopSender(models.PushPlatformWeb)
	fcm := NewNoopSender(models.PushPlatformFCM)
	svc := NewService(repo, nil, web, fcm)

	require.NoError(t, svc.NotifyUser(context.Background(), uid, &Notification{Title: "hi", Body: "world"}))

	assert.Len(t, web.Calls, 1)
	assert.Len(t, fcm.Calls, 1)
	touched, _ := repo.snapshot()
	assert.Len(t, touched, 2)
}

func TestService_NotifyUsers_FiltersByPlatformsWithSender(t *testing.T) {
	u1, u2 := uuid.New(), uuid.New()
	repo := &mockRepo{
		tokens: []*models.PushToken{
			{ID: uuid.New(), UserID: u1, Platform: models.PushPlatformWeb, Endpoint: "w1"},
			{ID: uuid.New(), UserID: u2, Platform: models.PushPlatformAPNS, Endpoint: "a1"},
		},
	}
	web := NewNoopSender(models.PushPlatformWeb)
	// APNs sender не передан — этот токен должен быть пропущен.
	svc := NewService(repo, nil, web)

	require.NoError(t, svc.NotifyUsers(context.Background(), []uuid.UUID{u1, u2}, &Notification{Title: "x"}))
	assert.Len(t, web.Calls, 1)
	assert.Equal(t, "w1", web.Calls[0].Endpoint)
	assert.False(t, svc.HasSender(models.PushPlatformAPNS))
}

func TestService_RemovesGoneSubscriptions(t *testing.T) {
	uid := uuid.New()
	repo := &mockRepo{
		tokens: []*models.PushToken{
			{ID: uuid.New(), UserID: uid, Platform: models.PushPlatformWeb, Endpoint: "expired"},
		},
	}
	gone := &goneSender{}
	svc := NewService(repo, nil, gone)

	require.NoError(t, svc.NotifyUser(context.Background(), uid, &Notification{Title: "x"}))
	touched, deleted := repo.snapshot()
	assert.Equal(t, []string{"expired"}, deleted)
	assert.Empty(t, touched)
}

func TestService_EmptyUsersIsNoop(t *testing.T) {
	repo := &mockRepo{}
	web := NewNoopSender(models.PushPlatformWeb)
	svc := NewService(repo, nil, web)
	require.NoError(t, svc.NotifyUsers(context.Background(), nil, &Notification{}))
	assert.Empty(t, web.Calls)
}

func TestNoopSender_RecordsCalls(t *testing.T) {
	s := NewNoopSender(models.PushPlatformWeb)
	tok := &models.PushToken{Endpoint: "e", Platform: models.PushPlatformWeb}
	require.NoError(t, s.Send(context.Background(), tok, &Notification{Title: "t", Body: "b", URL: "/u"}))
	assert.Equal(t, []NoopCall{{Endpoint: "e", Title: "t", Body: "b", URL: "/u"}}, s.Calls)
	s.Reset()
	assert.Empty(t, s.Calls)
}

func TestFCMSender_DisabledByDefault(t *testing.T) {
	s := NewFCMSender(false)
	err := s.Send(context.Background(), &models.PushToken{}, &Notification{})
	assert.ErrorIs(t, err, ErrSenderDisabled)
}

func TestAPNSSender_DisabledByDefault(t *testing.T) {
	s := NewAPNSSender(false)
	err := s.Send(context.Background(), &models.PushToken{}, &Notification{})
	assert.ErrorIs(t, err, ErrSenderDisabled)
}

func TestWebPushSender_RequiresVAPID(t *testing.T) {
	_, err := NewWebPushSender(WebPushOptions{})
	assert.Error(t, err)
}
