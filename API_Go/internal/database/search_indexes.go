package database

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// searchExtensionsSQL — список SQL-инструкций, выполняемых идемпотентно
// для подключения pg_trgm и создания GIN-индексов под полнотекстовый поиск.
//
// Все индексы используют `IF NOT EXISTS`, поэтому повторный вызов безопасен.
// На стенде, где у роли БД нет прав на CREATE EXTENSION, расширение нужно
// создать вручную (см. docs/Search.md).
var searchExtensionsSQL = []string{
	`CREATE EXTENSION IF NOT EXISTS pg_trgm`,
	`CREATE INDEX IF NOT EXISTS idx_users_name_trgm ON users USING GIN (name gin_trgm_ops)`,
	`CREATE INDEX IF NOT EXISTS idx_users_email_trgm ON users USING GIN (email gin_trgm_ops)`,
	`CREATE INDEX IF NOT EXISTS idx_rooms_name_trgm ON rooms USING GIN (name gin_trgm_ops) WHERE deleted_at IS NULL`,
	`CREATE INDEX IF NOT EXISTS idx_messages_content_trgm ON messages USING GIN (content gin_trgm_ops) WHERE is_deleted = false`,
}

// EnsureSearchExtensions проверяет наличие pg_trgm и создаёт GIN-индексы.
// Используется в `cmd/server/main.go` за флагом ENSURE_SEARCH_INDEXES=true.
//
// Ошибки `permission denied` логируются как Warn — это позволяет процессу
// успешно стартовать на стендах, где CREATE EXTENSION недоступен (там
// расширение и индексы должен создать DBA вручную).
func EnsureSearchExtensions(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	start := time.Now()
	for _, stmt := range searchExtensionsSQL {
		if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
			if isPermissionDenied(err) {
				logger.Warn("search: skipped (permission denied) — apply manually as DBA",
					zap.String("sql", shortenSQL(stmt)),
					zap.Error(err))
				continue
			}
			return err
		}
	}
	logger.Info("search: pg_trgm + GIN indexes ensured",
		zap.Duration("took", time.Since(start)))
	return nil
}

func isPermissionDenied(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "must be owner") ||
		strings.Contains(msg, "must be superuser")
}

func shortenSQL(s string) string {
	const limit = 80
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
