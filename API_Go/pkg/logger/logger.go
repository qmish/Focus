package logger

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Init инициализирует логгер
func Init(level string, format string) {
	config := zap.NewProductionConfig()

	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		lvl = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	config.Level = lvl

	if format == "console" {
		config.Encoding = "console"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	log, _ = config.Build()
}

// Debug логирует сообщение уровня DEBUG
func Debug(msg string, fields ...zap.Field) {
	log.Debug(msg, fields...)
}

// Info логирует сообщение уровня INFO
func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
}

// Warn логирует сообщение уровня WARN
func Warn(msg string, fields ...zap.Field) {
	log.Warn(msg, fields...)
}

// Error логирует сообщение уровня ERROR
func Error(msg string, fields ...zap.Field) {
	log.Error(msg, fields...)
}

// WithContext возвращает logger с контекстом
func WithContext(ctx context.Context) *zap.Logger {
	return log
}

// Sync синхронизирует буферы
func Sync() error {
	return log.Sync()
}

// Fields распространённые поля для логирования
type Fields struct {
	UserID    string
	RequestID string
	IP        string
	Method    string
	Path      string
	Status    int
	Duration  time.Duration
}

// ToZapFields конвертирует в zap поля
func (f Fields) ToZapFields() []zap.Field {
	fields := make([]zap.Field, 0, 8)

	if f.UserID != "" {
		fields = append(fields, zap.String("user_id", f.UserID))
	}
	if f.RequestID != "" {
		fields = append(fields, zap.String("request_id", f.RequestID))
	}
	if f.IP != "" {
		fields = append(fields, zap.String("ip", f.IP))
	}
	if f.Method != "" {
		fields = append(fields, zap.String("method", f.Method))
	}
	if f.Path != "" {
		fields = append(fields, zap.String("path", f.Path))
	}
	if f.Status > 0 {
		fields = append(fields, zap.Int("status", f.Status))
	}
	if f.Duration > 0 {
		fields = append(fields, zap.Duration("duration", f.Duration))
	}

	return fields
}

// Caller добавляет информацию о вызове
func Caller() zap.Field {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return zap.String("caller", "unknown")
	}
	return zap.String("caller", fmt.Sprintf("%s:%d", file, line))
}

// NewTestLogger создаёт тестовый логгер
func NewTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func init() {
	// Инициализация по умолчанию
	Init("info", "json")
}
