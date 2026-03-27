package bots

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotStruct(t *testing.T) {
	bot := Bot{
		ID:          uuid.New(),
		Name:        "Meeting Bot",
		Description: "Bot for creating meetings",
		OwnerID:     uuid.New(),
		Token:       "token-123",
		AvatarURL:   "https://example.com/avatar.png",
		IsActive:    true,
		Config: BotConfig{
			Commands: []BotCommand{
				{
					Command:     "create",
					Handler:     "meeting_create",
					Description: "Create a meeting",
					IsActive:    true,
				},
			},
		},
	}

	assert.Equal(t, "Meeting Bot", bot.Name)
	assert.Equal(t, "token-123", bot.Token)
	assert.True(t, bot.IsActive)
	assert.Len(t, bot.Config.Commands, 1)
}

func TestBotTypeConstants(t *testing.T) {
	assert.Equal(t, BotType("meeting"), BotTypeMeeting)
	assert.Equal(t, BotType("help"), BotTypeHelp)
	assert.Equal(t, BotType("status"), BotTypeStatus)
}

func TestBotCommand(t *testing.T) {
	cmd := BotCommand{
		Command:     "create",
		Handler:     "meeting_create",
		Description: "Create a meeting",
		IsActive:    true,
	}

	assert.Equal(t, "create", cmd.Command)
	assert.Equal(t, "meeting_create", cmd.Handler)
	assert.True(t, cmd.IsActive)
}

func TestCreateBot(t *testing.T) {
	commands := []BotCommand{
		{
			Command: "create",
			Handler: "meeting_create",
		},
	}

	bot := CreateBot("Meeting Bot", "Bot for meetings", uuid.New(), commands)

	assert.Equal(t, "Meeting Bot", bot.Name)
	assert.Equal(t, "Bot for meetings", bot.Description)
	assert.True(t, bot.IsActive)
	assert.NotEmpty(t, bot.Token)
	assert.Len(t, bot.Config.Commands, 1)
}

func TestBotEngine(t *testing.T) {
	engine := NewBotEngine()

	assert.NotNil(t, engine)
	assert.NotEmpty(t, engine.handlers)
}

func TestHandleMessageNotCommand(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	err := engine.HandleMessage(ctx, "room-123", "user-456", "Hello!")

	assert.NoError(t, err)
}

func TestHandleMessageEmptyCommand(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	err := engine.HandleMessage(ctx, "room-123", "user-456", "/")

	assert.NoError(t, err)
}

func TestHandleMessageHelpCommand(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	err := engine.HandleMessage(ctx, "room-123", "user-456", "/help")

	assert.NoError(t, err)
}

func TestHandleMessageStatusCommand(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	err := engine.HandleMessage(ctx, "room-123", "user-456", "/status")

	assert.NoError(t, err)
}

func TestHandleMessageCreateCommand(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	err := engine.HandleMessage(ctx, "room-123", "user-456", "/create meeting Test")

	assert.NoError(t, err)
}

func TestHandleMessageScheduleCommand(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	err := engine.HandleMessage(ctx, "room-123", "user-456", "/schedule meeting \"Test\" tomorrow 15:00")

	assert.NoError(t, err)
}

func TestHandleMessageUnknownCommand(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	err := engine.HandleMessage(ctx, "room-123", "user-456", "/unknown")

	assert.NoError(t, err)
}

func TestHandleMeetingCreateEmptyArgs(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	response, err := engine.handleMeetingCreate(ctx, "room-123", "user-456", "create", "")

	require.NoError(t, err)
	assert.Contains(t, response, "Использование")
}

func TestHandleMeetingCreateWithArgs(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	response, err := engine.handleMeetingCreate(ctx, "room-123", "user-456", "create", "Планёрка")

	require.NoError(t, err)
	assert.Contains(t, response, "Планёрка")
	assert.Contains(t, response, "создана")
}

func TestHandleMeetingScheduleEmptyArgs(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	response, err := engine.handleMeetingSchedule(ctx, "room-123", "user-456", "schedule", "")

	require.NoError(t, err)
	assert.Contains(t, response, "Использование")
}

func TestHandleMeetingScheduleWithArgs(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	response, err := engine.handleMeetingSchedule(ctx, "room-123", "user-456", "schedule", "Планёрка tomorrow 15:00")

	require.NoError(t, err)
	assert.Contains(t, response, "Планёрка")
	assert.Contains(t, response, "запланирована")
}

func TestHandleHelp(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	response, err := engine.handleHelp(ctx, "room-123", "user-456", "help", "")

	require.NoError(t, err)
	assert.Contains(t, response, "Доступные команды")
	assert.Contains(t, response, "/create")
	assert.Contains(t, response, "/help")
	assert.Contains(t, response, "/status")
	assert.Contains(t, response, "/members")
	assert.Contains(t, response, "/whoami")
	assert.Contains(t, response, "/dice")
	assert.Contains(t, response, "/find")
}

func TestHandleStatus(t *testing.T) {
	engine := NewBotEngine()

	ctx := context.Background()
	response, err := engine.handleStatus(ctx, "room-123", "user-456", "status", "")

	require.NoError(t, err)
	assert.Contains(t, response, "Статус системы")
}

func TestBotConfig(t *testing.T) {
	config := BotConfig{
		Commands: []BotCommand{
			{
				Command:     "create",
				Handler:     "meeting_create",
				Description: "Create meeting",
				IsActive:    true,
			},
			{
				Command:     "help",
				Handler:     "help",
				Description: "Show help",
				IsActive:    true,
			},
		},
	}

	assert.Len(t, config.Commands, 2)
	assert.Equal(t, "create", config.Commands[0].Command)
	assert.Equal(t, "help", config.Commands[1].Command)
}

func TestBotConfigEmptyCommands(t *testing.T) {
	config := BotConfig{
		Commands: []BotCommand{},
	}

	assert.Empty(t, config.Commands)
}

func TestHandleMessageSendsBotResponseToRoom(t *testing.T) {
	repo := &fakeBotMessageRepo{}
	roomChecker := &fakeBotRoomChecker{isParticipant: true}
	broadcaster := &fakeBotBroadcaster{}
	roomID := uuid.New()
	userID := uuid.New()
	botUserID := uuid.New()
	engine := NewBotEngineWithDelivery(repo, roomChecker, broadcaster, botUserID)

	err := engine.HandleMessage(context.Background(), roomID.String(), userID.String(), "/help")
	require.NoError(t, err)
	require.Len(t, repo.messages, 1)
	assert.Equal(t, roomID, repo.messages[0].RoomID)
	assert.Equal(t, botUserID, repo.messages[0].UserID)
	assert.Equal(t, models.MessageTypeSystem, repo.messages[0].Type)
	assert.Len(t, broadcaster.published, 1)
	assert.Equal(t, roomID.String(), broadcaster.published[0].roomID)

	var payload map[string]interface{}
	err = json.Unmarshal(broadcaster.published[0].message.Payload, &payload)
	require.NoError(t, err)
	assert.Equal(t, botUserID.String(), payload["user_id"])
}

func TestHandleMessageSkipsBotResponseWhenUserNotParticipant(t *testing.T) {
	repo := &fakeBotMessageRepo{}
	roomChecker := &fakeBotRoomChecker{isParticipant: false}
	broadcaster := &fakeBotBroadcaster{}
	engine := NewBotEngineWithDelivery(repo, roomChecker, broadcaster, uuid.New())

	err := engine.HandleMessage(context.Background(), uuid.New().String(), uuid.New().String(), "/help")
	require.NoError(t, err)
	assert.Empty(t, repo.messages)
	assert.Empty(t, broadcaster.published)
}

func TestHandleMeetingCreateCreatesRoomAndReturnsJitsiURL(t *testing.T) {
	repo := &fakeBotMessageRepo{}
	roomChecker := &fakeBotRoomChecker{isParticipant: true}
	broadcaster := &fakeBotBroadcaster{}
	roomRepo := &fakeBotRoomRepo{}
	userID := uuid.New()
	engine := NewBotEngineWithDelivery(repo, roomChecker, broadcaster, uuid.New())
	engine.SetRoomRepository(roomRepo)
	engine.SetJitsiBaseURL("https://meet.example.com")

	err := engine.HandleMessage(context.Background(), uuid.New().String(), userID.String(), "/create meeting Планерка")
	require.NoError(t, err)
	require.Len(t, roomRepo.created, 1)
	assert.Equal(t, models.RoomTypeMeeting, roomRepo.created[0].Type)
	assert.Contains(t, repo.messages[0].Content, "https://meet.example.com/")
}

func TestHandleMeetingScheduleSchedulesCalendar(t *testing.T) {
	repo := &fakeBotMessageRepo{}
	roomChecker := &fakeBotRoomChecker{isParticipant: true}
	broadcaster := &fakeBotBroadcaster{}
	roomRepo := &fakeBotRoomRepo{}
	scheduler := &fakeBotScheduler{}
	userID := uuid.New()
	engine := NewBotEngineWithDelivery(repo, roomChecker, broadcaster, uuid.New())
	engine.SetRoomRepository(roomRepo)
	engine.SetCalendarScheduler(scheduler)
	engine.SetJitsiBaseURL("https://meet.example.com")

	err := engine.HandleMessage(
		context.Background(),
		uuid.New().String(),
		userID.String(),
		"/schedule meeting Обзор at 2030-01-02 15:04",
	)
	require.NoError(t, err)
	require.Len(t, scheduler.calls, 1)
	assert.Equal(t, "Обзор", scheduler.calls[0].title)
	assert.Contains(t, scheduler.calls[0].roomURL, "https://meet.example.com/")
}

func TestHandleStatusUsesRoomRepository(t *testing.T) {
	repo := &fakeBotMessageRepo{}
	roomChecker := &fakeBotRoomChecker{isParticipant: true}
	broadcaster := &fakeBotBroadcaster{}
	roomRepo := &fakeBotRoomRepo{
		listRooms: []*models.Room{
			{ID: uuid.New(), Type: models.RoomTypeMeeting},
			{ID: uuid.New(), Type: models.RoomTypePublic},
		},
	}
	engine := NewBotEngineWithDelivery(repo, roomChecker, broadcaster, uuid.New())
	engine.SetRoomRepository(roomRepo)

	err := engine.HandleMessage(context.Background(), uuid.New().String(), uuid.New().String(), "/status")
	require.NoError(t, err)
	require.Len(t, repo.messages, 1)
	assert.Contains(t, repo.messages[0].Content, "Всего комнат: 2")
	assert.Contains(t, repo.messages[0].Content, "Активных встреч: 1")
}

func TestHandleMessageRateLimited(t *testing.T) {
	repo := &fakeBotMessageRepo{}
	roomChecker := &fakeBotRoomChecker{isParticipant: true}
	broadcaster := &fakeBotBroadcaster{}
	engine := NewBotEngineWithDelivery(repo, roomChecker, broadcaster, uuid.New())
	engine.SetRateLimitWindow(1 * time.Hour)

	roomID := uuid.New().String()
	userID := uuid.New().String()
	err := engine.HandleMessage(context.Background(), roomID, userID, "/help")
	require.NoError(t, err)
	err = engine.HandleMessage(context.Background(), roomID, userID, "/status")
	require.NoError(t, err)
	assert.Len(t, repo.messages, 1)
}

func TestHandleMessageRecordsFailedEvent(t *testing.T) {
	repo := &fakeBotMessageRepo{}
	roomChecker := &fakeBotRoomChecker{isParticipant: false}
	broadcaster := &fakeBotBroadcaster{}
	eventStore := &fakeBotEventStore{}
	engine := NewBotEngineWithDelivery(repo, roomChecker, broadcaster, uuid.New())
	engine.SetCommandEventStore(eventStore)

	err := engine.HandleMessage(context.Background(), uuid.New().String(), uuid.New().String(), "/help")
	require.NoError(t, err)
	require.Len(t, eventStore.events, 1)
	assert.Equal(t, "permission_denied", eventStore.events[0].Status)
}

func TestHandleMessageRecordsRateLimitedEvent(t *testing.T) {
	repo := &fakeBotMessageRepo{}
	roomChecker := &fakeBotRoomChecker{isParticipant: true}
	broadcaster := &fakeBotBroadcaster{}
	eventStore := &fakeBotEventStore{}
	engine := NewBotEngineWithDelivery(repo, roomChecker, broadcaster, uuid.New())
	engine.SetCommandEventStore(eventStore)
	engine.SetRateLimitWindow(1 * time.Hour)

	roomID := uuid.New().String()
	userID := uuid.New().String()
	err := engine.HandleMessage(context.Background(), roomID, userID, "/help")
	require.NoError(t, err)
	err = engine.HandleMessage(context.Background(), roomID, userID, "/status")
	require.NoError(t, err)
	require.Len(t, eventStore.events, 2)
	assert.Equal(t, "sent", eventStore.events[0].Status)
	assert.Equal(t, "rate_limited", eventStore.events[1].Status)
}

func TestHandleDiceDefault(t *testing.T) {
	engine := NewBotEngine()
	response, err := engine.handleDice(context.Background(), "room-123", "user-456", "dice", "")
	require.NoError(t, err)
	assert.Contains(t, response, "d6")
	assert.Contains(t, response, "🎲")
}

func TestHandleDiceCustomSides(t *testing.T) {
	engine := NewBotEngine()
	response, err := engine.handleDice(context.Background(), "room-123", "user-456", "dice", "20")
	require.NoError(t, err)
	assert.Contains(t, response, "d20")
}

func TestHandleDiceInvalidSides(t *testing.T) {
	engine := NewBotEngine()
	response, err := engine.handleDice(context.Background(), "room-123", "user-456", "dice", "abc")
	require.NoError(t, err)
	assert.Contains(t, response, "d6")
}

func TestHandleFindEmptyQuery(t *testing.T) {
	engine := NewBotEngine()
	response, err := engine.handleFind(context.Background(), "room-123", "user-456", "find", "")
	require.NoError(t, err)
	assert.Contains(t, response, "Использование")
}

func TestHandleFindWithResults(t *testing.T) {
	roomRepo := &fakeBotRoomRepo{
		searchResults: []*models.Room{
			{ID: uuid.New(), Name: "Планёрка", Type: models.RoomTypePublic},
			{ID: uuid.New(), Name: "Планёрка-2", Type: models.RoomTypeMeeting},
		},
	}
	engine := NewBotEngine()
	engine.SetRoomRepository(roomRepo)
	response, err := engine.handleFind(context.Background(), uuid.New().String(), uuid.New().String(), "find", "планёрка")
	require.NoError(t, err)
	assert.Contains(t, response, "Найдено комнат: 2")
	assert.Contains(t, response, "Планёрка")
}

func TestHandleFindNoResults(t *testing.T) {
	roomRepo := &fakeBotRoomRepo{searchResults: []*models.Room{}}
	engine := NewBotEngine()
	engine.SetRoomRepository(roomRepo)
	response, err := engine.handleFind(context.Background(), uuid.New().String(), uuid.New().String(), "find", "несуществующая")
	require.NoError(t, err)
	assert.Contains(t, response, "ничего не найдено")
}

func TestHandleMembersNoRepo(t *testing.T) {
	engine := NewBotEngine()
	response, err := engine.handleMembers(context.Background(), uuid.New().String(), uuid.New().String(), "members", "")
	require.NoError(t, err)
	assert.Contains(t, response, "недоступна")
}

func TestHandleMembersWithParticipants(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	roomRepo := &fakeBotRoomRepo{
		rooms: map[uuid.UUID]*models.Room{
			roomID: {ID: roomID, Name: "Тестовая", Type: models.RoomTypePublic},
		},
		participants: []models.RoomParticipant{
			{RoomID: roomID, UserID: userID, Role: models.ParticipantRoleModerator, User: &models.User{Name: "Иванов", Email: "ivanov@test.com"}},
		},
	}
	engine := NewBotEngine()
	engine.SetRoomRepository(roomRepo)
	response, err := engine.handleMembers(context.Background(), roomID.String(), userID.String(), "members", "")
	require.NoError(t, err)
	assert.Contains(t, response, "Тестовая")
	assert.Contains(t, response, "Иванов")
	assert.Contains(t, response, "1 участник")
}

func TestHandleWhoamiNoRepo(t *testing.T) {
	engine := NewBotEngine()
	response, err := engine.handleWhoami(context.Background(), "room-123", "user-456", "whoami", "")
	require.NoError(t, err)
	assert.Contains(t, response, "user-456")
}

func TestHandleWhoamiWithRepo(t *testing.T) {
	userID := uuid.New()
	userRepo := &fakeBotUserRepo{
		users: map[uuid.UUID]*models.User{
			userID: {ID: userID, Name: "Петров", Email: "petrov@test.com", Roles: models.StringArray{"admin", "user"}, CreatedAt: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)},
		},
	}
	engine := NewBotEngine()
	engine.SetUserRepository(userRepo)
	response, err := engine.handleWhoami(context.Background(), uuid.New().String(), userID.String(), "whoami", "")
	require.NoError(t, err)
	assert.Contains(t, response, "Петров")
	assert.Contains(t, response, "petrov@test.com")
	assert.Contains(t, response, "admin, user")
}

func TestHandleStatusShowsUptime(t *testing.T) {
	engine := NewBotEngine()
	engine.startedAt = time.Now().Add(-5 * time.Minute)
	response, err := engine.handleStatus(context.Background(), "room-123", "user-456", "status", "")
	require.NoError(t, err)
	assert.Contains(t, response, "Аптайм бота")
	assert.Contains(t, response, "5m")
}

type fakeBotMessageRepo struct {
	messages []*models.Message
}

func (f *fakeBotMessageRepo) Create(ctx context.Context, message *models.Message) error {
	f.messages = append(f.messages, message)
	return nil
}

type fakeBotRoomChecker struct {
	isParticipant bool
}

func (f *fakeBotRoomChecker) IsParticipant(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return f.isParticipant, nil
}

type publishedBotMessage struct {
	roomID  string
	message websocket.WSMessage
}

type fakeBotBroadcaster struct {
	published []publishedBotMessage
}

func (f *fakeBotBroadcaster) BroadcastToRoom(roomID string, message websocket.WSMessage) {
	f.published = append(f.published, publishedBotMessage{
		roomID:  roomID,
		message: message,
	})
}

type fakeBotRoomRepo struct {
	created       []*models.Room
	listRooms     []*models.Room
	rooms         map[uuid.UUID]*models.Room
	participants  []models.RoomParticipant
	searchResults []*models.Room
}

func (f *fakeBotRoomRepo) Create(ctx context.Context, room *models.Room) error {
	f.created = append(f.created, room)
	return nil
}

func (f *fakeBotRoomRepo) AddParticipant(ctx context.Context, roomID, userID uuid.UUID, role models.ParticipantRole) error {
	return nil
}

func (f *fakeBotRoomRepo) List(ctx context.Context, limit, offset int) ([]*models.Room, error) {
	if f.listRooms == nil {
		return f.created, nil
	}
	return f.listRooms, nil
}

func (f *fakeBotRoomRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Room, error) {
	if f.rooms != nil {
		if r, ok := f.rooms[id]; ok {
			return r, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (f *fakeBotRoomRepo) CountParticipants(ctx context.Context, roomID uuid.UUID) (int64, error) {
	return int64(len(f.participants)), nil
}

func (f *fakeBotRoomRepo) Search(ctx context.Context, query string, limit int) ([]*models.Room, error) {
	return f.searchResults, nil
}

func (f *fakeBotRoomRepo) ListParticipantsWithUsers(ctx context.Context, roomID uuid.UUID) ([]models.RoomParticipant, error) {
	return f.participants, nil
}

type fakeBotUserRepo struct {
	users map[uuid.UUID]*models.User
}

func (f *fakeBotUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if f.users != nil {
		if u, ok := f.users[id]; ok {
			return u, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

type fakeBotScheduleCall struct {
	userID  uuid.UUID
	title   string
	start   time.Time
	end     time.Time
	roomURL string
}

type fakeBotScheduler struct {
	calls []fakeBotScheduleCall
}

func (f *fakeBotScheduler) ScheduleMeeting(ctx context.Context, userID uuid.UUID, title string, start, end time.Time, roomURL string) error {
	f.calls = append(f.calls, fakeBotScheduleCall{
		userID:  userID,
		title:   title,
		start:   start,
		end:     end,
		roomURL: roomURL,
	})
	return nil
}

type fakeBotEventStore struct {
	events []*BotCommandEvent
}

func (f *fakeBotEventStore) CreateCommandEvent(ctx context.Context, event *BotCommandEvent) error {
	f.events = append(f.events, event)
	return nil
}
