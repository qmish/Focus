package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewRoom(t *testing.T) {
	name := "Test Room"
	creatorID := uuid.New()

	room := NewRoom(name, creatorID, RoomTypePublic)

	assert.Equal(t, name, room.Name)
	assert.Equal(t, creatorID, room.CreatorID)
	assert.Equal(t, RoomTypePublic, room.Type)
	assert.False(t, room.IsPrivate)
	assert.NotEmpty(t, room.JitsiRoomName)
	assert.NotEmpty(t, room.ID)
	assert.Equal(t, "public", string(room.Type))
}

func TestNewRoomPrivate(t *testing.T) {
	creatorID := uuid.New()
	room := NewRoom("Private Room", creatorID, RoomTypePrivate)

	assert.True(t, room.IsPrivate)
	assert.Equal(t, RoomTypePrivate, room.Type)
}

func TestNewRoomMeeting(t *testing.T) {
	creatorID := uuid.New()
	room := NewRoom("Meeting Room", creatorID, RoomTypeMeeting)

	assert.Equal(t, RoomTypeMeeting, room.Type)
	assert.False(t, room.IsPrivate)
}

func TestRoomGetJitsiURL(t *testing.T) {
	creatorID := uuid.New()
	room := NewRoom("Test Room", creatorID, RoomTypePublic)

	baseURL := "https://meet.company.com"
	url := room.GetJitsiURL(baseURL)

	assert.Equal(t, baseURL+"/"+room.JitsiRoomName, url)
	assert.Contains(t, url, room.JitsiRoomName)
}

func TestRoomSettings(t *testing.T) {
	room := &Room{
		Settings: RoomSettings{
			AllowGuests:             true,
			RequireModeratorForMsgs: true,
			MaxParticipants:         50,
		},
	}

	assert.True(t, room.Settings.AllowGuests)
	assert.True(t, room.Settings.RequireModeratorForMsgs)
	assert.Equal(t, 50, room.Settings.MaxParticipants)
}

func TestRoomTableName(t *testing.T) {
	room := Room{}
	assert.Equal(t, "rooms", room.TableName())
}

func TestRoomType(t *testing.T) {
	assert.Equal(t, RoomType("public"), RoomTypePublic)
	assert.Equal(t, RoomType("private"), RoomTypePrivate)
	assert.Equal(t, RoomType("meeting"), RoomTypeMeeting)
}
