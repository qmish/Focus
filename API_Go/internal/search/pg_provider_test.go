package search

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const searchTestEmailDomain = "@test-search.local"

func newSearchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "host=localhost port=5432 user=focus password=focus dbname=focus_test sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Skipf("Skipping DB-dependent test: %v", err)
	}
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Room{},
		&models.RoomParticipant{},
		&models.Message{},
		&models.MeetingLink{},
	))
	t.Cleanup(func() {
		db.Exec("DELETE FROM messages WHERE content LIKE 'search-test-%'")
		db.Exec("DELETE FROM meeting_links WHERE subject LIKE 'search-test-%'")
		db.Exec("DELETE FROM room_participants WHERE room_id IN (SELECT id FROM rooms WHERE name LIKE 'search-test-%')")
		db.Exec("DELETE FROM rooms WHERE name LIKE 'search-test-%'")
		db.Exec("DELETE FROM users WHERE email LIKE '%" + searchTestEmailDomain + "'")
	})
	return db
}

func mkUser(t *testing.T, db *gorm.DB, name string) *models.User {
	t.Helper()
	u := &models.User{
		ID:       uuid.New(),
		Email:    strings.ToLower(strings.ReplaceAll(name, " ", "-")) + "-" + uuid.NewString()[:6] + searchTestEmailDomain,
		Name:     name,
		Roles:    models.StringArray{"user"},
		IsActive: true,
	}
	require.NoError(t, db.Create(u).Error)
	return u
}

func mkRoom(t *testing.T, db *gorm.DB, name string, rt models.RoomType, creator *models.User) *models.Room {
	t.Helper()
	r := models.NewRoom("search-test-"+name, creator.ID, rt)
	require.NoError(t, db.Create(r).Error)
	return r
}

func mkParticipant(t *testing.T, db *gorm.DB, room *models.Room, user *models.User) {
	t.Helper()
	p := models.NewRoomParticipant(room.ID, user.ID, models.ParticipantRoleMember)
	require.NoError(t, db.Create(p).Error)
}

func mkMessage(t *testing.T, db *gorm.DB, room *models.Room, user *models.User, content string, mtype models.MessageType, meta models.Metadata) *models.Message {
	t.Helper()
	m := models.NewMessage(room.ID, user.ID, "search-test-"+content, mtype)
	m.Metadata = meta
	require.NoError(t, db.Create(m).Error)
	return m
}

func TestPgProvider_SearchUsers(t *testing.T) {
	db := newSearchTestDB(t)
	provider := NewPgProvider(db)
	ctx := context.Background()

	u1 := mkUser(t, db, "Alice Searchington")
	u2 := mkUser(t, db, "Bob Findler")
	_ = u2

	got, err := provider.SearchUsers(ctx, "searchington", 10)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	found := false
	for _, u := range got {
		if u.ID == u1.ID {
			found = true
		}
	}
	assert.True(t, found, "ожидали найти Alice Searchington")

	empty, err := proverEmpty(provider.SearchUsers(ctx, "", 10))
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func proverEmpty(users []*models.User, err error) ([]*models.User, error) {
	return users, err
}

func TestPgProvider_SearchRooms_ABAC(t *testing.T) {
	db := newSearchTestDB(t)
	provider := NewPgProvider(db)
	ctx := context.Background()

	creator := mkUser(t, db, "Creator Sr")
	visitor := mkUser(t, db, "Visitor Vr")

	publicRoom := mkRoom(t, db, "public-projects", models.RoomTypePublic, creator)
	privateRoom := mkRoom(t, db, "private-projects", models.RoomTypePrivate, creator)
	mkParticipant(t, db, publicRoom, creator)
	mkParticipant(t, db, privateRoom, creator)

	got, err := provider.SearchRooms(ctx, visitor.ID, "projects", 10)
	require.NoError(t, err)
	ids := map[uuid.UUID]bool{}
	for _, r := range got {
		ids[r.ID] = true
	}
	assert.True(t, ids[publicRoom.ID], "видим публичную")
	assert.False(t, ids[privateRoom.ID], "приватная не должна быть видна не-участнику")

	mkParticipant(t, db, privateRoom, visitor)
	got2, err := provider.SearchRooms(ctx, visitor.ID, "projects", 10)
	require.NoError(t, err)
	found := false
	for _, r := range got2 {
		if r.ID == privateRoom.ID {
			found = true
		}
	}
	assert.True(t, found, "после добавления участника приватная стала видимой")
}

func TestPgProvider_SearchMessages_ABAC(t *testing.T) {
	db := newSearchTestDB(t)
	provider := NewPgProvider(db)
	ctx := context.Background()

	owner := mkUser(t, db, "Owner Mm")
	intruder := mkUser(t, db, "Intruder Ii")

	priv := mkRoom(t, db, "secret", models.RoomTypePrivate, owner)
	mkParticipant(t, db, priv, owner)

	mkMessage(t, db, priv, owner, "needle alpha", models.MessageTypeText, models.Metadata{})

	hits, err := provider.SearchMessages(ctx, intruder.ID, "needle", nil, MessageSearchOpts{Limit: 10})
	require.NoError(t, err)
	assert.Empty(t, hits, "сообщения чужой комнаты не выдаются")

	hits2, err := provider.SearchMessages(ctx, owner.ID, "needle", nil, MessageSearchOpts{Limit: 10})
	require.NoError(t, err)
	require.NotEmpty(t, hits2)
	assert.Contains(t, hits2[0].Highlight, "<mark>")
}

func TestPgProvider_SearchMessages_ScopedToRoom(t *testing.T) {
	db := newSearchTestDB(t)
	provider := NewPgProvider(db)
	ctx := context.Background()

	user := mkUser(t, db, "Scoper Ss")
	a := mkRoom(t, db, "scope-a", models.RoomTypePrivate, user)
	b := mkRoom(t, db, "scope-b", models.RoomTypePrivate, user)
	mkParticipant(t, db, a, user)
	mkParticipant(t, db, b, user)

	mkMessage(t, db, a, user, "needle in a", models.MessageTypeText, models.Metadata{})
	mkMessage(t, db, b, user, "needle in b", models.MessageTypeText, models.Metadata{})

	roomA := a.ID
	hits, err := provider.SearchMessages(ctx, user.ID, "needle", &roomA, MessageSearchOpts{Limit: 10})
	require.NoError(t, err)
	require.NotEmpty(t, hits)
	for _, h := range hits {
		assert.Equal(t, roomA, h.RoomID, "должны быть только сообщения из комнаты A")
	}
}

func TestPgProvider_SearchFiles_ByMetadataFileName(t *testing.T) {
	db := newSearchTestDB(t)
	provider := NewPgProvider(db)
	ctx := context.Background()

	user := mkUser(t, db, "Filer Ff")
	room := mkRoom(t, db, "files", models.RoomTypePrivate, user)
	mkParticipant(t, db, room, user)

	meta := models.Metadata{FileName: "Annual-Report-2025.pdf", FileMIME: "application/pdf", FileID: uuid.NewString(), FileSize: 12345}
	mkMessage(t, db, room, user, "uploaded report", models.MessageTypeFile, meta)

	files, err := provider.SearchFiles(ctx, user.ID, "annual", 10)
	require.NoError(t, err)
	require.NotEmpty(t, files)
	assert.Equal(t, "Annual-Report-2025.pdf", files[0].FileName)
}

func TestPgProvider_SearchMeetings_ABAC(t *testing.T) {
	db := newSearchTestDB(t)
	provider := NewPgProvider(db)
	ctx := context.Background()

	organizer := mkUser(t, db, "Organizer Mo")
	other := mkUser(t, db, "Other Mo")

	room := mkRoom(t, db, "meeting-room", models.RoomTypePrivate, organizer)
	mkParticipant(t, db, room, organizer)

	now := time.Now()
	link := &models.MeetingLink{
		ID:              uuid.New(),
		RoomID:          room.ID,
		ExchangeEventID: "ex-" + uuid.NewString(),
		OrganizerEmail:  organizer.Email,
		Subject:         "search-test-Quarterly-Planning",
		StartAt:         now.Add(time.Hour),
		EndAt:           now.Add(2 * time.Hour),
		Status:          "scheduled",
		SyncSource:      "focus",
	}
	require.NoError(t, db.Create(link).Error)

	hitsOrg, err := provider.SearchMeetings(ctx, organizer.ID, "Quarterly", 10)
	require.NoError(t, err)
	require.NotEmpty(t, hitsOrg)
	assert.Equal(t, link.ID, hitsOrg[0].ID)

	hitsOther, err := provider.SearchMeetings(ctx, other.ID, "Quarterly", 10)
	require.NoError(t, err)
	for _, h := range hitsOther {
		assert.NotEqual(t, link.ID, h.ID, "не-участник и не-организатор не должен видеть встречу")
	}
}
