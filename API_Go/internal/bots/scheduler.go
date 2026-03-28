package bots

import (
	"context"
	"encoding/json"
	"time"

	"github.com/qmish/focus-api/internal/models"
)

// BotScheduler runs periodic checks for reminders and scheduled messages.
type BotScheduler struct {
	engine          *BotEngine
	reminderStore   BotReminderStore
	settingsProvider BotSettingsProvider
	interval        time.Duration
}

// NewBotScheduler creates a scheduler that checks for pending reminders.
func NewBotScheduler(engine *BotEngine, reminderStore BotReminderStore, settingsProvider BotSettingsProvider) *BotScheduler {
	return &BotScheduler{
		engine:          engine,
		reminderStore:   reminderStore,
		settingsProvider: settingsProvider,
		interval:        15 * time.Second,
	}
}

// Start begins the scheduler loop. Blocks until ctx is cancelled.
func (s *BotScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processReminders(ctx)
			s.processScheduledMessages(ctx)
		}
	}
}

func (s *BotScheduler) processReminders(ctx context.Context) {
	if s.reminderStore == nil {
		return
	}
	reminders, err := s.reminderStore.ListPendingReminders(ctx, time.Now())
	if err != nil {
		return
	}
	for _, r := range reminders {
		if r == nil {
			continue
		}
		msg := "⏰ Напоминание: " + r.Message
		_ = s.engine.sendResponse(ctx, r.RoomID, msg)
		_ = s.reminderStore.MarkFired(ctx, r.ID)
	}
}

func (s *BotScheduler) processScheduledMessages(ctx context.Context) {
	if s.settingsProvider == nil {
		return
	}
	settings, err := s.settingsProvider.List(ctx)
	if err != nil {
		return
	}
	now := time.Now()
	for _, setting := range settings {
		if setting == nil || !setting.IsEnabled || setting.ScheduleJSON == "" || setting.ScheduleJSON == "[]" {
			continue
		}
		var entries []models.BotScheduleEntry
		if err := json.Unmarshal([]byte(setting.ScheduleJSON), &entries); err != nil {
			continue
		}
		for _, entry := range entries {
			if shouldFireCron(entry.Cron, now) {
				_ = s.engine.sendResponse(ctx, entry.RoomID, entry.Message)
			}
		}
	}
}

// shouldFireCron checks if a cron expression matches the current minute.
// Supports simplified cron: "minute hour day-of-month month day-of-week"
func shouldFireCron(expr string, now time.Time) bool {
	if expr == "" {
		return false
	}
	fields := splitFields(expr)
	if len(fields) != 5 {
		return false
	}
	return matchField(fields[0], now.Minute()) &&
		matchField(fields[1], now.Hour()) &&
		matchField(fields[2], now.Day()) &&
		matchField(fields[3], int(now.Month())) &&
		matchField(fields[4], int(now.Weekday()))
}

func splitFields(s string) []string {
	var fields []string
	current := ""
	for _, ch := range s {
		if ch == ' ' || ch == '\t' {
			if current != "" {
				fields = append(fields, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		fields = append(fields, current)
	}
	return fields
}

func matchField(field string, value int) bool {
	if field == "*" {
		return true
	}
	for _, part := range splitByComma(field) {
		if matchRange(part, value) {
			return true
		}
	}
	return false
}

func splitByComma(s string) []string {
	var parts []string
	current := ""
	for _, ch := range s {
		if ch == ',' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func matchRange(part string, value int) bool {
	dashIdx := -1
	for i, ch := range part {
		if ch == '-' {
			dashIdx = i
			break
		}
	}
	if dashIdx >= 0 {
		lo := parseInt(part[:dashIdx], -1)
		hi := parseInt(part[dashIdx+1:], -1)
		return lo >= 0 && hi >= 0 && value >= lo && value <= hi
	}
	return parseInt(part, -1) == value
}

func parseInt(s string, def int) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	if s == "" {
		return def
	}
	return n
}
