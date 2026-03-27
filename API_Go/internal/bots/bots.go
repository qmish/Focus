package bots

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
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

// BotSettingsProvider reads bot settings from persistent storage.
type BotSettingsProvider interface {
	List(ctx context.Context) ([]*models.BotSetting, error)
}

// BotEngine движок ботов
type BotEngine struct {
	handlers          map[string]BotHandler
	customHandlers    map[string]string // command -> static reply text
	messageRepo       BotMessageRepository
	roomChecker       BotRoomAccessChecker
	roomRepo          BotRoomRepository
	userRepo          BotUserRepository
	broadcaster       BotBroadcaster
	calendarScheduler BotCalendarScheduler
	eventStore        BotCommandEventStore
	settingsProvider  BotSettingsProvider
	botUserID         uuid.UUID
	jitsiBaseURL      string
	rateLimitWindow   time.Duration
	lastCommandAt     map[string]time.Time
	disabledCommands  map[string]bool
	commandRooms      map[string]map[string]bool // command -> set of allowed room IDs
	commandRateLimit  map[string]time.Duration   // command -> per-bot rate limit
	mu                sync.Mutex
	now               func() time.Time
	startedAt         time.Time
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
	GetByID(ctx context.Context, id uuid.UUID) (*models.Room, error)
	CountParticipants(ctx context.Context, roomID uuid.UUID) (int64, error)
	Search(ctx context.Context, query string, limit int) ([]*models.Room, error)
	ListParticipantsWithUsers(ctx context.Context, roomID uuid.UUID) ([]models.RoomParticipant, error)
}

// BotUserRepository retrieves user information for bot commands.
type BotUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
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
		handlers:         make(map[string]BotHandler),
		customHandlers:   make(map[string]string),
		jitsiBaseURL:     "https://meet.company.com",
		rateLimitWindow:  2 * time.Second,
		lastCommandAt:    make(map[string]time.Time),
		disabledCommands: make(map[string]bool),
		commandRooms:     make(map[string]map[string]bool),
		commandRateLimit: make(map[string]time.Duration),
		now:              time.Now,
		startedAt:        time.Now(),
	}

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

// SetUserRepository injects user repo for user-related commands.
func (e *BotEngine) SetUserRepository(userRepo BotUserRepository) {
	e.userRepo = userRepo
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

// SetBotSettingsProvider injects a provider that reads BotSetting from DB.
func (e *BotEngine) SetBotSettingsProvider(p BotSettingsProvider) {
	e.settingsProvider = p
}

// ReloadSettings reads bot_settings from DB and updates runtime config.
func (e *BotEngine) ReloadSettings(ctx context.Context) error {
	if e.settingsProvider == nil {
		return nil
	}
	settings, err := e.settingsProvider.List(ctx)
	if err != nil {
		return fmt.Errorf("reload bot settings: %w", err)
	}

	disabled := make(map[string]bool)
	rooms := make(map[string]map[string]bool)
	rateLimit := make(map[string]time.Duration)
	custom := make(map[string]string)

	for _, s := range settings {
		if s == nil {
			continue
		}

		var cmds []BotCommand
		if err := json.Unmarshal([]byte(s.CommandsJSON), &cmds); err != nil {
			cmds = nil
		}

		for _, cmd := range cmds {
			cmdName := strings.TrimPrefix(strings.TrimSpace(cmd.Command), "/")
			if cmdName == "" {
				continue
			}
			if !s.IsEnabled || !cmd.IsActive {
				disabled[cmdName] = true
				continue
			}
			if cmd.Handler == "static-reply" && cmd.Description != "" {
				custom[cmdName] = cmd.Description
			}
			if s.RateLimitMs > 0 {
				rateLimit[cmdName] = time.Duration(s.RateLimitMs) * time.Millisecond
			}
			if len(s.AllowedRooms) > 0 {
				roomSet := make(map[string]bool, len(s.AllowedRooms))
				for _, rid := range s.AllowedRooms {
					roomSet[rid] = true
				}
				rooms[cmdName] = roomSet
			}
		}

		if !s.IsEnabled {
			for _, cmd := range cmds {
				cmdName := strings.TrimPrefix(strings.TrimSpace(cmd.Command), "/")
				if cmdName != "" {
					disabled[cmdName] = true
				}
			}
		}
	}

	e.mu.Lock()
	e.disabledCommands = disabled
	e.commandRooms = rooms
	e.commandRateLimit = rateLimit
	e.customHandlers = custom
	e.mu.Unlock()
	return nil
}

func (e *BotEngine) registerBuiltinBots() {
	e.handlers["create"] = e.handleMeetingCreate
	e.handlers["schedule"] = e.handleMeetingSchedule
	e.handlers["help"] = e.handleHelp
	e.handlers["status"] = e.handleStatus
	e.handlers["members"] = e.handleMembers
	e.handlers["whoami"] = e.handleWhoami
	e.handlers["dice"] = e.handleDice
	e.handlers["find"] = e.handleFind
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

	e.mu.Lock()
	isDisabled := e.disabledCommands[command]
	allowedRooms := e.commandRooms[command]
	cmdRateLimit := e.commandRateLimit[command]
	customReply := e.customHandlers[command]
	e.mu.Unlock()

	if isDisabled {
		e.recordCommandEvent(ctx, roomID, userID, command, args, "disabled", "")
		return nil
	}

	if len(allowedRooms) > 0 && !allowedRooms[roomID] {
		e.recordCommandEvent(ctx, roomID, userID, command, args, "room_not_allowed", "")
		return nil
	}

	handler, isBuiltin := e.handlers[command]
	if !isBuiltin && customReply == "" {
		return nil
	}

	if !e.canHandleInRoom(ctx, roomID, userID) {
		e.recordCommandEvent(ctx, roomID, userID, command, args, "permission_denied", "")
		return nil
	}
	if cmdRateLimit > 0 {
		if e.isRateLimitedWithWindow(userID, cmdRateLimit) {
			e.recordCommandEvent(ctx, roomID, userID, command, args, "rate_limited", "")
			return nil
		}
	} else if e.isRateLimited(userID) {
		e.recordCommandEvent(ctx, roomID, userID, command, args, "rate_limited", "")
		return nil
	}

	var response string
	var err error
	if isBuiltin {
		response, err = handler(ctx, roomID, userID, command, args)
	} else {
		response = customReply
	}
	if err != nil {
		e.recordCommandEvent(ctx, roomID, userID, command, args, "failed", err.Error())
		return err
	}
	if err := e.sendResponse(ctx, roomID, response); err != nil {
		e.recordCommandEvent(ctx, roomID, userID, command, args, "failed", err.Error())
		return err
	}
	e.recordCommandEvent(ctx, roomID, userID, command, args, "sent", "")
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
  /schedule meeting "название" at <YYYY-MM-DD HH:MM> — Запланировать встречу

👥 Комната:
  /members — Список участников этой комнаты
  /find <запрос> — Найти комнаты по названию

👤 Пользователь:
  /whoami — Информация о вашем профиле

ℹ️ Система:
  /status — Статус комнат и аптайм бота
  /help — Показать эту справку

🎲 Развлечения:
  /dice [грани] — Бросить кубик (по умолчанию 6 граней)

Примеры:
  /create meeting Планёрка
  /schedule meeting Обзор спринта at 2026-04-01 15:00
  /find планёрка
  /dice 20`

	return helpText, nil
}

// handleStatus обработчик команды /status
func (e *BotEngine) handleStatus(ctx context.Context, roomID, userID, command, args string) (string, error) {
	uptime := e.now().Sub(e.startedAt).Truncate(time.Second)
	if e.roomRepo == nil {
		return fmt.Sprintf("📊 Статус системы:\n\nАктивных встреч: 0\n⏱ Аптайм бота: %s", uptime), nil
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
	return fmt.Sprintf("📊 Статус системы:\n\nВсего комнат: %d\nАктивных встреч: %d\n⏱ Аптайм бота: %s", totalRooms, activeMeetings, uptime), nil
}

// handleMembers обработчик команды /members
func (e *BotEngine) handleMembers(ctx context.Context, roomID, userID, command, args string) (string, error) {
	if e.roomRepo == nil {
		return "⚠️ Информация о комнате недоступна", nil
	}
	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return "⚠️ Некорректный ID комнаты", nil
	}
	room, err := e.roomRepo.GetByID(ctx, roomUUID)
	if err != nil {
		return "⚠️ Комната не найдена", nil
	}
	participants, err := e.roomRepo.ListParticipantsWithUsers(ctx, roomUUID)
	if err != nil {
		return "⚠️ Не удалось получить участников", nil
	}
	if len(participants) == 0 {
		return fmt.Sprintf("👥 Комната «%s»\n\nНет участников", room.Name), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👥 Комната «%s» — %d участник(ов):\n\n", room.Name, len(participants)))
	roleIcons := map[models.ParticipantRole]string{
		models.ParticipantRoleAdmin:     "👑",
		models.ParticipantRoleModerator: "⭐",
		models.ParticipantRoleMember:    "👤",
	}
	for _, p := range participants {
		icon := roleIcons[p.Role]
		if icon == "" {
			icon = "👤"
		}
		name := "—"
		if p.User != nil {
			name = p.User.Name
			if p.User.Email != "" {
				name += " (" + p.User.Email + ")"
			}
		}
		sb.WriteString(fmt.Sprintf("  %s %s — %s\n", icon, name, p.Role))
	}
	return sb.String(), nil
}

// handleWhoami обработчик команды /whoami
func (e *BotEngine) handleWhoami(ctx context.Context, roomID, userID, command, args string) (string, error) {
	if e.userRepo == nil {
		return fmt.Sprintf("👤 Ваш ID: %s", userID), nil
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Sprintf("👤 Ваш ID: %s", userID), nil
	}
	user, err := e.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		return fmt.Sprintf("👤 Ваш ID: %s", userID), nil
	}
	roles := "—"
	if len(user.Roles) > 0 {
		roles = strings.Join(user.Roles, ", ")
	}
	return fmt.Sprintf("👤 Профиль:\n\n  Имя: %s\n  Email: %s\n  Роли: %s\n  Регистрация: %s",
		user.Name, user.Email, roles, user.CreatedAt.Format("02.01.2006 15:04")), nil
}

// handleDice обработчик команды /dice
func (e *BotEngine) handleDice(ctx context.Context, roomID, userID, command, args string) (string, error) {
	sides := 6
	if s := strings.TrimSpace(args); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 2 && n <= 1000 {
			sides = n
		}
	}
	result := rand.Intn(sides) + 1
	return fmt.Sprintf("🎲 Бросок кубика (d%d): **%d**", sides, result), nil
}

// handleFind обработчик команды /find
func (e *BotEngine) handleFind(ctx context.Context, roomID, userID, command, args string) (string, error) {
	query := strings.TrimSpace(args)
	if query == "" {
		return "Использование: `/find <запрос>`\nПример: `/find планёрка`", nil
	}
	if e.roomRepo == nil {
		return "⚠️ Поиск недоступен", nil
	}
	rooms, err := e.roomRepo.Search(ctx, query, 10)
	if err != nil {
		return "", err
	}
	if len(rooms) == 0 {
		return fmt.Sprintf("🔍 По запросу «%s» ничего не найдено", query), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 Найдено комнат: %d\n\n", len(rooms)))
	typeIcons := map[models.RoomType]string{
		models.RoomTypePublic:  "💬",
		models.RoomTypePrivate: "🔒",
		models.RoomTypeMeeting: "📹",
	}
	for _, room := range rooms {
		icon := typeIcons[room.Type]
		if icon == "" {
			icon = "💬"
		}
		sb.WriteString(fmt.Sprintf("  %s %s (%s)\n", icon, room.Name, room.Type))
	}
	return sb.String(), nil
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
	return e.isRateLimitedWithWindow(userID, e.rateLimitWindow)
}

func (e *BotEngine) isRateLimitedWithWindow(userID string, window time.Duration) bool {
	if window <= 0 {
		return false
	}
	now := e.now()
	e.mu.Lock()
	defer e.mu.Unlock()
	lastAt, ok := e.lastCommandAt[userID]
	if ok && now.Sub(lastAt) < window {
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
