package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewRoomParticipant(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()

	participant := NewRoomParticipant(roomID, userID, ParticipantRoleMember)

	assert.Equal(t, roomID, participant.RoomID)
	assert.Equal(t, userID, participant.UserID)
	assert.Equal(t, ParticipantRoleMember, participant.Role)
	assert.NotEmpty(t, participant.JoinedAt)
	assert.NotEmpty(t, participant.LastReadAt)
	assert.Nil(t, participant.LeftAt)
}

func TestNewRoomParticipantModerator(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()

	participant := NewRoomParticipant(roomID, userID, ParticipantRoleModerator)

	assert.Equal(t, ParticipantRoleModerator, participant.Role)
}

func TestRoomParticipantTableName(t *testing.T) {
	p := RoomParticipant{}
	assert.Equal(t, "room_participants", p.TableName())
}

func TestParticipantRole(t *testing.T) {
	assert.Equal(t, ParticipantRole("member"), ParticipantRoleMember)
	assert.Equal(t, ParticipantRole("moderator"), ParticipantRoleModerator)
	assert.Equal(t, ParticipantRole("admin"), ParticipantRoleAdmin)
}

func TestRoomParticipantTimezone(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()

	participant := NewRoomParticipant(roomID, userID, ParticipantRoleMember)

	// Проверяем, что время в правильном часовом поясе
	assert.Equal(t, time.Local, participant.JoinedAt.Location())
}
