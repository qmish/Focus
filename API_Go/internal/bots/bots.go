package bots

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
	handlers map[string]BotHandler
}

// BotHandler обработчик команд бота
type BotHandler func(ctx context.Context, roomID, userID, command, args string) (string, error)

// NewBotEngine создаёт новый BotEngine
func NewBotEngine() *BotEngine {
	engine := &BotEngine{
		handlers: make(map[string]BotHandler),
	}

	// Регистрируем встроенных ботов
	engine.registerBuiltinBots()

	return engine
}

func (e *BotEngine) registerBuiltinBots() {
	// Meeting Bot
	e.handlers["meeting_create"] = e.handleMeetingCreate
	e.handlers["meeting_schedule"] = e.handleMeetingSchedule

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
		response, err := handler(ctx, roomID, userID, command, args)
		if err != nil {
			return err
		}
		// Отправляем ответ
		return e.sendResponse(ctx, roomID, response)
	}

	return nil
}

func (e *BotEngine) sendResponse(ctx context.Context, roomID, content string) error {
	// TODO: Использовать реальную реализацию отправки сообщений
	return nil
}

// handleMeetingCreate обработчик команды /create
func (e *BotEngine) handleMeetingCreate(ctx context.Context, roomID, userID, command, args string) (string, error) {
	// Парсим аргументы: /create meeting [название]
	if args == "" {
		return "Использование: `/create meeting [название]`", nil
	}

	// TODO: Создать встречу
	return fmt.Sprintf("Встреча \"%s\" создана!", args), nil
}

// handleMeetingSchedule обработчик команды /schedule
func (e *BotEngine) handleMeetingSchedule(ctx context.Context, roomID, userID, command, args string) (string, error) {
	// Парсим аргументы: /schedule meeting "название" tomorrow 15:00
	if args == "" {
		return "Использование: `/schedule meeting \"название\" <дата> <время>`", nil
	}

	// TODO: Запланировать встречу
	return fmt.Sprintf("Встреча \"%s\" запланирована!", args), nil
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
	// TODO: Получить статус активных комнат
	return "📊 Статус комнат:\n\nАктивных встреч: 0", nil
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
