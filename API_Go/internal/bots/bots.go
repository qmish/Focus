package bots

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"github.com/qmish/focus-api/internal/websocket"
)

// BotType тип бота
type BotType string

const (
	BotTypeMeeting BotType = "meeting"
	BotTypeHelp    BotType = "help"
	BotTypeStatus  BotType = "status"
)

// Bot модель бота
type Bot struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Token       string    `json:"-"` // Не возвращать в API
	AvatarURL   string    `json:"avatar_url"`
	IsActive    bool      `json:"is_active"`
	Config      BotConfig `json:"config"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BotCommandEvent stores command execution outcomes for observability.
type BotCommandEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	RoomID    string    `gorm:"type:varchar(64);index" json:"room_id"`
	UserID    string    `gorm:"type:varchar(64);index" json:"user_id"`
	Command   string    `gorm:"type:varchar(64);index" json:"command"`
	Args      string    `gorm:"type:text" json:"args"`
	Status    string    `gorm:"type:varchar(32);index" json:"status"`
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns table for bot command event logs.
func (BotCommandEvent) TableName() string {
	return "bot_command_events"
}

// BotConfig конфигурация бота
type BotConfig struct {
	Commands []BotCommand `json:"commands"`
}

// BotCommand команда бота
type BotCommand struct {
	Command     string `json:"command"`
	Handler     string `json:"handler"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// BotEngine движок ботов
type BotEngine struct {
	handlers          map[string]BotHandler
	messageRepo       BotMessageRepository
	roomChecker       BotRoomAccessChecker
	roomRepo          BotRoomRepository
	broadcaster       BotBroadcaster
	calendarScheduler BotCalendarScheduler
	eventStore        BotCommandEventStore
	botUserID         uuid.UUID
	jitsiBaseURL      string
	rateLimitWindow   time.Duration
	lastCommandAt     map[string]time.Time
	mu                sync.Mutex
	now               func() time.Time
}

// BotHandler обработчик команд бота
type BotHandler func(ctx context.Context, roomID, userID, command, args string) (string, error)

// BotMessageRepository stores bot responses as regular room messages.
type BotMessageRepository interface {
	Create(ctx context.Context, message *models.Message) error
}

// BotRoomAccessChecker validates whether user can execute commands in room.
type BotRoomAccessChecker interface {
	IsParticipant(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
}

// BotBroadcaster publishes bot responses over websocket.
type BotBroadcaster interface {
	BroadcastToRoom(roomID string, message websocket.WSMessage)
}

// BotRoomRepository performs room operations for bot commands.
type BotRoomRepository interface {
	Create(ctx context.Context, room *models.Room) error
	AddParticipant(ctx context.Context, roomID, userID uuid.UUID, role models.ParticipantRole) error
	List(ctx context.Context, limit, offset int) ([]*models.Room, error)
}

// BotCalendarScheduler schedules bot meetings in calendar provider.
type BotCalendarScheduler interface {
	ScheduleMeeting(ctx context.Context, userID uuid.UUID, title string, start, end time.Time, roomURL string) error
}

// BotCommandEventStore persists command execution events.
type BotCommandEventStore interface {
	CreateCommandEvent(ctx context.Context, event *BotCommandEvent) error
}

// NewBotEngine создаёт новый BotEngine
func NewBotEngine() *BotEngine {
	engine := &BotEngine{
		handlers:        make(map[string]BotHandler),
		jitsiBaseURL:    "https://meet.company.com",
		rateLimitWindow: 2 * time.Second,
		lastCommandAt:   make(map[string]time.Time),
		now:             time.Now,
	}

	// Регистрируем встроенных ботов
	engine.registerBuiltinBots()

	return engine
}

// NewBotEngineWithDelivery creates bot engine with real room delivery.
func NewBotEngineWithDelivery(
	messageRepo BotMessageRepository,
	roomChecker BotRoomAccessChecker,
	broadcaster BotBroadcaster,
	botUserID uuid.UUID,
) *BotEngine {
	engine := NewBotEngine()
	engine.messageRepo = messageRepo
	engine.roomChecker = roomChecker
	engine.broadcaster = broadcaster
	engine.botUserID = botUserID
	return engine
}

// SetRoomRepository injects room repo for create/schedule/status commands.
func (e *BotEngine) SetRoomRepository(roomRepo BotRoomRepository) {
	e.roomRepo = roomRepo
}

// SetCalendarScheduler injects calendar scheduler for /schedule integration.
func (e *BotEngine) SetCalendarScheduler(scheduler BotCalendarScheduler) {
	e.calendarScheduler = scheduler
}

// SetCommandEventStore injects store for bot command observability.
func (e *BotEngine) SetCommandEventStore(store BotCommandEventStore) {
	e.eventStore = store
}

// SetJitsiBaseURL sets base URL used in bot-created meeting links.
func (e *BotEngine) SetJitsiBaseURL(baseURL string) {
	if strings.TrimSpace(baseURL) == "" {
		return
	}
	e.jitsiBaseURL = strings.TrimRight(baseURL, "/")
}

// SetRateLimitWindow sets minimal interval between commands per user.
func (e *BotEngine) SetRateLimitWindow(window time.Duration) {
	if window <= 0 {
		return
	}
	e.rateLimitWindow = window
}

func (e *BotEngine) registerBuiltinBots() {
	// Meeting Bot
	e.handlers["create"] = e.handleMeetingCreate
	e.handlers["schedule"] = e.handleMeetingSchedule

	// Help Bot
	e.handlers["help"] = e.handleHelp

	// Status Bot
	e.handlers["status"] = e.handleStatus
}

// HandleMessage обрабатывает сообщение и проверяет на наличие команд бота
func (e *BotEngine) HandleMessage(ctx context.Context, roomID, userID, content string) error {
	// Проверяем, начинается ли сообщение с /
	if !strings.HasPrefix(content, "/") {
		return nil
	}

	// Парсим команду
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return nil
	}

	command := strings.TrimPrefix(parts[0], "/")
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	// Проверяем встроенные команды
	if handler, ok := e.handlers[command]; ok {
		if !e.canHandleInRoom(ctx, roomID, userID) {
			e.recordCommandEvent(ctx, roomID, userID, command, args, "permission_denied", "")
			return nil
		}
		if e.isRateLimited(userID) {
			e.recordCommandEvent(ctx, roomID, userID, command, args, "rate_limited", "")
			return nil
		}
		response, err := handler(ctx, roomID, userID, command, args)
		if err != nil {
			e.recordCommandEvent(ctx, roomID, userID, command, args, "failed", err.Error())
			return err
		}
		// Отправляем ответ
		if err := e.sendResponse(ctx, roomID, response); err != nil {
			e.recordCommandEvent(ctx, roomID, userID, command, args, "failed", err.Error())
			return err
		}
		e.recordCommandEvent(ctx, roomID, userID, command, args, "sent", "")
		return nil
	}

	return nil
}

func (e *BotEngine) recordCommandEvent(ctx context.Context, roomID, userID, command, args, status, errText string) {
	if e.eventStore == nil {
		return
	}
	_ = e.eventStore.CreateCommandEvent(ctx, &BotCommandEvent{
		ID:        uuid.New(),
		RoomID:    roomID,
		UserID:    userID,
		Command:   command,
		Args:      args,
		Status:    status,
		Error:     errText,
		CreatedAt: e.now().UTC(),
	})
}

func (e *BotEngine) sendResponse(ctx context.Context, roomID, content string) error {
	if e.messageRepo == nil || e.broadcaster == nil || e.botUserID == uuid.Nil {
		return nil
	}
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return nil
	}
	message := models.NewMessage(roomUUID, e.botUserID, content, models.MessageTypeSystem)
	if err := e.messageRepo.Create(ctx, message); err != nil {
		return err
	}

	payload, err := json.Marshal(map[string]interface{}{
		"id":         message.ID.String(),
		"room_id":    roomUUID.String(),
		"user_id":    e.botUserID.String(),
		"content":    content,
		"type":       string(message.Type),
		"created_at": message.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	e.broadcaster.BroadcastToRoom(roomUUID.String(), websocket.WSMessage{
		Type:    websocket.MessageTypeMessage,
		Payload: payload,
	})
	return nil
}

func (e *BotEngine) canHandleInRoom(ctx context.Context, roomID, userID string) bool {
	if e.roomChecker == nil {
		return true
	}
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return false
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false
	}
	isParticipant, err := e.roomChecker.IsParticipant(ctx, roomUUID, userUUID)
	if err != nil {
		return false
	}
	return isParticipant
}

// handleMeetingCreate обработчик команды /create
func (e *BotEngine) handleMeetingCreate(ctx context.Context, roomID, userID, command, args string) (string, error) {
	title := normalizeMeetingTitle(args)
	if title == "" {
		return "Использование: `/create meeting <название>`", nil
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Sprintf("Встреча \"%s\" создана!", title), nil
	}

	if e.roomRepo == nil {
		return fmt.Sprintf("Встреча \"%s\" создана!", title), nil
	}
	room := models.NewRoom(title, userUUID, models.RoomTypeMeeting)
	if err := e.roomRepo.Create(ctx, room); err != nil {
		return "", err
	}
	_ = e.roomRepo.AddParticipant(ctx, room.ID, userUUID, models.ParticipantRoleModerator)
	return fmt.Sprintf("Встреча \"%s\" создана: %s/%s", title, e.jitsiBaseURL, room.JitsiRoomName), nil
}

// handleMeetingSchedule обработчик команды /schedule
func (e *BotEngine) handleMeetingSchedule(ctx context.Context, roomID, userID, command, args string) (string, error) {
	title, startAt, ok := parseScheduleArgs(args, e.now())
	if !ok {
		legacyTitle := normalizeMeetingTitle(args)
		if legacyTitle != "" {
			return fmt.Sprintf("Встреча \"%s\" запланирована!", legacyTitle), nil
		}
		return "Использование: `/schedule meeting <название> at <YYYY-MM-DD HH:MM|RFC3339>`", nil
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Sprintf("Встреча \"%s\" запланирована на %s", title, startAt.Format("02.01.2006 15:04")), nil
	}

	if e.roomRepo == nil {
		return fmt.Sprintf("Встреча \"%s\" запланирована на %s", title, startAt.Format("02.01.2006 15:04")), nil
	}

	room := models.NewRoom(title, userUUID, models.RoomTypeMeeting)
	room.Description = "scheduled_at=" + startAt.UTC().Format(time.RFC3339)
	if err := e.roomRepo.Create(ctx, room); err != nil {
		return "", err
	}
	_ = e.roomRepo.AddParticipant(ctx, room.ID, userUUID, models.ParticipantRoleModerator)

	roomURL := fmt.Sprintf("%s/%s", e.jitsiBaseURL, room.JitsiRoomName)
	if e.calendarScheduler != nil {
		_ = e.calendarScheduler.ScheduleMeeting(ctx, userUUID, title, startAt, startAt.Add(time.Hour), roomURL)
	}
	return fmt.Sprintf("Встреча \"%s\" запланирована на %s: %s", title, startAt.Format("02.01.2006 15:04"), roomURL), nil
}

// handleHelp обработчик команды /help
func (e *BotEngine) handleHelp(ctx context.Context, roomID, userID, command, args string) (string, error) {
	helpText := `🤖 Доступные команды:

📅 Встречи:
  /create meeting [название] — Создать встречу
  /schedule meeting "название" <дата> <время> — Запланировать встречу

ℹ️ Информация:
  /help — Показать эту справку
  /status — Показать статус комнат

Примеры:
  /create meeting Планёрка
  /schedule meeting "Обзор спринта" tomorrow 15:00`

	return helpText, nil
}

// handleStatus обработчик команды /status
func (e *BotEngine) handleStatus(ctx context.Context, roomID, userID, command, args string) (string, error) {
	if e.roomRepo == nil {
		return "📊 Статус комнат:\n\nАктивных встреч: 0", nil
	}
	rooms, err := e.roomRepo.List(ctx, 500, 0)
	if err != nil {
		return "", err
	}
	totalRooms := 0
	activeMeetings := 0
	for _, room := range rooms {
		if room == nil {
			continue
		}
		totalRooms++
		if room.Type == models.RoomTypeMeeting {
			activeMeetings++
		}
	}
	return fmt.Sprintf("📊 Статус комнат:\n\nВсего комнат: %d\nАктивных встреч: %d", totalRooms, activeMeetings), nil
}

// CreateBot создаёт нового бота
func CreateBot(name, description string, ownerID uuid.UUID, commands []BotCommand) *Bot {
	return &Bot{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		Token:       uuid.New().String(),
		IsActive:    true,
		Config: BotConfig{
			Commands: commands,
		},
	}
}

func (e *BotEngine) isRateLimited(userID string) bool {
	if e.rateLimitWindow <= 0 {
		return false
	}
	now := e.now()
	e.mu.Lock()
	defer e.mu.Unlock()
	lastAt, ok := e.lastCommandAt[userID]
	if ok && now.Sub(lastAt) < e.rateLimitWindow {
		return true
	}
	e.lastCommandAt[userID] = now
	return false
}

func normalizeMeetingTitle(args string) string {
	title := strings.TrimSpace(args)
	title = strings.TrimPrefix(title, "meeting")
	title = strings.TrimSpace(title)
	return title
}

func parseScheduleArgs(args string, now time.Time) (string, time.Time, bool) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return "", time.Time{}, false
	}
	trimmed = strings.TrimPrefix(trimmed, "meeting")
	trimmed = strings.TrimSpace(trimmed)
	parts := strings.Split(trimmed, " at ")
	if len(parts) != 2 {
		return "", time.Time{}, false
	}
	title := strings.TrimSpace(parts[0])
	if title == "" {
		return "", time.Time{}, false
	}
	candidates := []string{
		time.RFC3339,
		"2006-01-02 15:04",
		"02.01.2006 15:04",
	}
	for _, layout := range candidates {
		if t, err := time.ParseInLocation(layout, strings.TrimSpace(parts[1]), now.Location()); err == nil {
			return title, t, true
		}
	}
	return "", time.Time{}, false
}
