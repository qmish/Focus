package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"gorm.io/gorm"
)

// RoomRepository репозиторий для работы с комнатами
type RoomRepository struct {
	db *gorm.DB
}

// NewRoomRepository создаёт новый RoomRepository
func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

// Create создаёт новую комнату
func (r *RoomRepository) Create(ctx context.Context, room *models.Room) error {
	if err := r.db.WithContext(ctx).Create(room).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrRoomAlreadyExists
		}
		return err
	}
	return nil
}

// GetByID получает комнату по ID
func (r *RoomRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Room, error) {
	var room models.Room
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL").First(&room, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}
	return &room, nil
}

// GetByJitsiRoomName получает комнату по имени Jitsi комнаты
func (r *RoomRepository) GetByJitsiRoomName(ctx context.Context, jitsiRoomName string) (*models.Room, error) {
	var room models.Room
	if err := r.db.WithContext(ctx).Where("jitsi_room_name = ? AND deleted_at IS NULL", jitsiRoomName).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}
	return &room, nil
}

// Update обновляет комнату
func (r *RoomRepository) Update(ctx context.Context, room *models.Room) error {
	room.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(room).Error
}

// Delete удаляет комнату (мягкое удаление)
func (r *RoomRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Room{}).
		Where("id = ?", id).
		Update("deleted_at", &now).Error
}

// List получает список комнат с пагинацией
func (r *RoomRepository) List(ctx context.Context, limit, offset int) ([]*models.Room, error) {
	var rooms []*models.Room
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&rooms).Error
	return rooms, err
}

// ListByCreator получает список комнат созданных пользователем
func (r *RoomRepository) ListByCreator(ctx context.Context, creatorID uuid.UUID, limit, offset int) ([]*models.Room, error) {
	var rooms []*models.Room
	err := r.db.WithContext(ctx).
		Where("creator_id = ? AND deleted_at IS NULL", creatorID).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&rooms).Error
	return rooms, err
}

// ListByParticipant получает список комнат где пользователь является участником
func (r *RoomRepository) ListByParticipant(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Room, error) {
	var rooms []*models.Room
	err := r.db.WithContext(ctx).
		Joins("JOIN room_participants ON room_participants.room_id = rooms.id").
		Where("room_participants.user_id = ? AND rooms.deleted_at IS NULL", userID).
		Limit(limit).
		Offset(offset).
		Order("rooms.created_at DESC").
		Find(&rooms).Error
	return rooms, err
}

// Count возвращает количество комнат
func (r *RoomRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Room{}).Where("deleted_at IS NULL").Count(&count).Error
	return count, err
}

// CountByType returns number of rooms by room type.
func (r *RoomRepository) CountByType(ctx context.Context, roomType models.RoomType) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Room{}).
		Where("deleted_at IS NULL AND type = ?", roomType).
		Count(&count).Error
	return count, err
}

// AddParticipant добавляет участника в комнату
func (r *RoomRepository) AddParticipant(ctx context.Context, roomID, userID uuid.UUID, role models.ParticipantRole) error {
	participant := models.NewRoomParticipant(roomID, userID, role)
	return r.db.WithContext(ctx).Create(participant).Error
}

// RemoveParticipant удаляет участника из комнаты
func (r *RoomRepository) RemoveParticipant(ctx context.Context, roomID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.RoomParticipant{}, "room_id = ? AND user_id = ?", roomID, userID).Error
}

// GetParticipant получает участника комнаты
func (r *RoomRepository) GetParticipant(ctx context.Context, roomID, userID uuid.UUID) (*models.RoomParticipant, error) {
	var participant models.RoomParticipant
	if err := r.db.WithContext(ctx).Where("room_id = ? AND user_id = ?", roomID, userID).First(&participant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &participant, nil
}

// IsParticipant проверяет является ли пользователь участником комнаты
func (r *RoomRepository) IsParticipant(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.RoomParticipant{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountParticipants returns number of participants in a room.
func (r *RoomRepository) CountParticipants(ctx context.Context, roomID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.RoomParticipant{}).
		Where("room_id = ?", roomID).
		Count(&count).Error
	return count, err
}

// Search ищет комнаты по названию
func (r *RoomRepository) Search(ctx context.Context, query string, limit int) ([]*models.Room, error) {
	var rooms []*models.Room
	searchPattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND name ILIKE ?", searchPattern).
		Limit(limit).
		Order("name ASC").
		Find(&rooms).Error
	return rooms, err
}
