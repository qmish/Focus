package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrRoomNotFound      = errors.New("room not found")
	ErrRoomAlreadyExists = errors.New("room already exists")
	ErrMessageNotFound   = errors.New("message not found")
	ErrWebhookNotFound   = errors.New("webhook not found")
	ErrBotNotFound       = errors.New("bot not found")
)

// UserRepository репозиторий для работы с пользователями
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository создаёт новый UserRepository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create создаёт нового пользователя
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrUserAlreadyExists
		}
		return err
	}
	return nil
}

// GetByID получает пользователя по ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByKeycloakID получает пользователя по Keycloak ID
func (r *UserRepository) GetByKeycloakID(ctx context.Context, keycloakID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("keycloak_id = ?", keycloakID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail получает пользователя по email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Update обновляет пользователя
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdateLastLogin обновляет время последнего входа
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", id).
		Update("last_login_at", now).Error
}

// Delete удаляет пользователя (мягкое удаление)
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// List получает список пользователей с пагинацией
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&users).Error
	return users, err
}

// Count возвращает количество пользователей
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error
	return count, err
}

// Search ищет пользователей по имени или email
func (r *UserRepository) Search(ctx context.Context, query string, limit int) ([]*models.User, error) {
	var users []*models.User
	searchPattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("name ILIKE ? OR email ILIKE ?", searchPattern, searchPattern).
		Limit(limit).
		Order("name ASC").
		Find(&users).Error
	return users, err
}

// FindByNames находит пользователей по списку имён
func (r *UserRepository) FindByNames(ctx context.Context, names []string) ([]*models.User, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var users []*models.User
	err := r.db.WithContext(ctx).
		Where("name IN ?", names).
		Find(&users).Error
	return users, err
}

// SearchInRoom ищет пользователей внутри конкретной комнаты по имени или email
func (r *UserRepository) SearchInRoom(ctx context.Context, query string, roomID uuid.UUID, limit int) ([]*models.User, error) {
	var users []*models.User
	searchPattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Joins("JOIN room_participants rp ON rp.user_id = users.id").
		Where("rp.room_id = ? AND (users.name ILIKE ? OR users.email ILIKE ?)", roomID, searchPattern, searchPattern).
		Limit(limit).
		Order("users.name ASC").
		Find(&users).Error
	return users, err
}

// GetOrCreate получает пользователя или создаёт нового
func (r *UserRepository) GetOrCreate(ctx context.Context, keycloakID uuid.UUID, email, name string) (*models.User, error) {
	user, err := r.GetByKeycloakID(ctx, keycloakID)
	if err == nil {
		return user, nil
	}

	if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	// Запись по keycloak_id не найдена — пробуем привязаться к существующему
	// пользователю по email (например, созданному через локальную регистрацию
	// или импортированному из Exchange/AD). Это позволяет одному email
	// иметь и локальный, и SSO-вход.
	if email != "" {
		existing, errByEmail := r.GetByEmail(ctx, email)
		if errByEmail == nil {
			kid := keycloakID
			existing.KeycloakID = &kid
			if name != "" && existing.Name == "" {
				existing.Name = name
			}
			if err := r.Update(ctx, existing); err != nil {
				return nil, err
			}
			return existing, nil
		}
		if !errors.Is(errByEmail, ErrUserNotFound) {
			return nil, errByEmail
		}
	}

	user = models.NewUser(keycloakID, email, name)
	if err := r.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func isUniqueViolation(err error) bool {
	// Проверка на уникальность (PostgreSQL error code 23505)
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}

