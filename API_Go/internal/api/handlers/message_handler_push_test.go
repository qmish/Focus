package handlers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/push"
	"github.com/qmish/focus-api/internal/repository"
	"github.com/qmish/focus-api/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// notifierMock записывает вызовы NotifyUsers для проверки.
type notifierMock struct {
	mu    sync.Mutex
	calls []notifierCall
}

type notifierCall struct {
	UserIDs      []uuid.UUID
	Title        string
	Body         string
	URL          string
	Tag          string
	Notification *push.Notification
}

func (m *notifierMock) NotifyUsers(_ context.Context, users []uuid.UUID, n *push.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]uuid.UUID, len(users))
	copy(cp, users)
	m.calls = append(m.calls, notifierCall{
		UserIDs:      cp,
		Title:        n.Title,
		Body:         n.Body,
		URL:          n.URL,
		Tag:          n.Tag,
		Notification: n,
	})
	return nil
}

func (m *notifierMock) totalRecipients() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	cnt := 0
	for _, c := range m.calls {
		cnt += len(c.UserIDs)
	}
	return cnt
}

// findCall возвращает первый вызов, в котором есть указанный user.
func (m *notifierMock) findCall(uid uuid.UUID) *notifierCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, c := range m.calls {
		for _, u := range c.UserIDs {
			if u == uid {
				return &m.calls[i]
			}
		}
	}
	return nil
}

// TestNotifyOffline_OfflineParticipantsAndMentions проверяет, что push
// рассылается оффлайн-участникам комнаты и упомянутым пользователям, причём
// упомянутые получают уведомление с пометкой "mention", даже если они онлайн.
func TestNotifyOffline_OfflineParticipantsAndMentions(t *testing.T) {
	db := getTestDB(t)
	roomRepo := repository.NewRoomRepository(db)
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	defer time.Sleep(10 * time.Millisecond)

	h := NewMessageHandler(
		repository.NewMessageRepository(db),
		repository.NewUserRepository(db),
		wsHub,
		nil,
	)
	h.SetRoomRepository(roomRepo)

	mock := &notifierMock{}
	h.SetPushService(mock)

	// users: автор + двое получателей. Один из них упомянут.
	authorID := uuid.New()
	offlineID := uuid.New()
	mentionedID := uuid.New()
	seedTestUser(t, db, authorID)
	seedTestUser(t, db, offlineID)
	seedTestUser(t, db, mentionedID)

	roomID := uuid.New()
	seedTestRoom(t, db, roomID, authorID)
	now := time.Now()
	require.NoError(t, db.Create(&models.RoomParticipant{
		RoomID: roomID, UserID: authorID, Role: "owner", JoinedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.RoomParticipant{
		RoomID: roomID, UserID: offlineID, Role: "member", JoinedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.RoomParticipant{
		RoomID: roomID, UserID: mentionedID, Role: "member", JoinedAt: now,
	}).Error)

	msg := models.NewMessage(roomID, authorID, "Hello @mentioned", models.MessageTypeText)
	msg.Metadata.Mentions = []string{mentionedID.String()}
	require.NoError(t, db.Create(msg).Error)

	h.notifyOffline(context.Background(), roomID, authorID, msg)

	assert.Equal(t, 2, mock.totalRecipients(),
		"должны быть оба получателя: упомянутый + оффлайн-участник")

	mentionCall := mock.findCall(mentionedID)
	require.NotNil(t, mentionCall, "ожидаем вызов для упомянутого пользователя")
	assert.Contains(t, mentionCall.Title, "упомянул", "title должен содержать ‘упомянул’")
	assert.Contains(t, mentionCall.Tag, "mention-", "tag должен начинаться с mention-")

	offlineCall := mock.findCall(offlineID)
	require.NotNil(t, offlineCall, "ожидаем вызов для оффлайн-участника")
	assert.NotContains(t, offlineCall.Title, "упомянул", "у оффлайн-уведомления заголовок без mention")
}

// TestNotifyOffline_AuthorIsExcluded — автор сообщения никогда не должен получать push о собственном сообщении.
func TestNotifyOffline_AuthorIsExcluded(t *testing.T) {
	db := getTestDB(t)
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	h := NewMessageHandler(
		repository.NewMessageRepository(db),
		repository.NewUserRepository(db),
		wsHub,
		nil,
	)
	h.SetRoomRepository(repository.NewRoomRepository(db))
	mock := &notifierMock{}
	h.SetPushService(mock)

	authorID := uuid.New()
	seedTestUser(t, db, authorID)
	roomID := uuid.New()
	seedTestRoom(t, db, roomID, authorID)
	require.NoError(t, db.Create(&models.RoomParticipant{
		RoomID: roomID, UserID: authorID, Role: "owner", JoinedAt: time.Now(),
	}).Error)

	msg := models.NewMessage(roomID, authorID, "только я", models.MessageTypeText)
	require.NoError(t, db.Create(msg).Error)
	h.notifyOffline(context.Background(), roomID, authorID, msg)

	assert.Equal(t, 0, mock.totalRecipients(), "автор не должен получать push сам себе")
}

// TestNotifyOffline_NilService — если push не сконфигурирован, метод не падает.
func TestNotifyOffline_NilService(t *testing.T) {
	db := getTestDB(t)
	h := NewMessageHandler(
		repository.NewMessageRepository(db),
		repository.NewUserRepository(db),
		websocket.NewHub(zap.NewNop()),
		nil,
	)
	h.SetRoomRepository(repository.NewRoomRepository(db))
	// pushService не задан
	authorID := uuid.New()
	seedTestUser(t, db, authorID)
	roomID := uuid.New()
	seedTestRoom(t, db, roomID, authorID)
	msg := models.NewMessage(roomID, authorID, "x", models.MessageTypeText)
	require.NoError(t, db.Create(msg).Error)
	assert.NotPanics(t, func() {
		h.notifyOffline(context.Background(), roomID, authorID, msg)
	})
}

// TestNotifyOffline_ImagePreview — для image-сообщений preview содержит эмодзи.
func TestNotifyOffline_ImagePreview(t *testing.T) {
	db := getTestDB(t)
	wsHub := websocket.NewHub(zap.NewNop())
	go wsHub.Run()
	h := NewMessageHandler(
		repository.NewMessageRepository(db),
		repository.NewUserRepository(db),
		wsHub,
		nil,
	)
	h.SetRoomRepository(repository.NewRoomRepository(db))
	mock := &notifierMock{}
	h.SetPushService(mock)

	authorID := uuid.New()
	receiverID := uuid.New()
	seedTestUser(t, db, authorID)
	seedTestUser(t, db, receiverID)
	roomID := uuid.New()
	seedTestRoom(t, db, roomID, authorID)
	require.NoError(t, db.Create(&models.RoomParticipant{
		RoomID: roomID, UserID: authorID, Role: "owner", JoinedAt: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&models.RoomParticipant{
		RoomID: roomID, UserID: receiverID, Role: "member", JoinedAt: time.Now(),
	}).Error)

	msg := models.NewMessage(roomID, authorID, "", models.MessageTypeImage)
	require.NoError(t, db.Create(msg).Error)
	h.notifyOffline(context.Background(), roomID, authorID, msg)

	c := mock.findCall(receiverID)
	require.NotNil(t, c)
	assert.Contains(t, c.Body, "Изображение")
}
