package search

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/qmish/focus-api/internal/models"
)

// fakeProvider — детерминированная реализация SearchProvider для unit-тестов.
type fakeProvider struct {
	mu sync.Mutex

	users    []*models.User
	rooms    []*models.Room
	messages []*MessageHit
	files    []*FileHit
	meetings []*MeetingHit

	usersErr error

	calledUsers    int
	calledRooms    int
	calledMessages int
	calledFiles    int
	calledMeetings int

	gotRoomID *uuid.UUID
}

func (f *fakeProvider) SearchUsers(_ context.Context, _ string, _ int) ([]*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calledUsers++
	if f.usersErr != nil {
		return nil, f.usersErr
	}
	return f.users, nil
}
func (f *fakeProvider) SearchRooms(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*models.Room, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calledRooms++
	return f.rooms, nil
}
func (f *fakeProvider) SearchMessages(_ context.Context, _ uuid.UUID, _ string, roomID *uuid.UUID, _ MessageSearchOpts) ([]*MessageHit, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calledMessages++
	if roomID != nil {
		copy := *roomID
		f.gotRoomID = &copy
	}
	return f.messages, nil
}
func (f *fakeProvider) SearchFiles(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*FileHit, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calledFiles++
	return f.files, nil
}
func (f *fakeProvider) SearchMeetings(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*MeetingHit, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calledMeetings++
	return f.meetings, nil
}

func TestService_Global_RejectsShortQuery(t *testing.T) {
	svc := NewService(&fakeProvider{})
	_, err := svc.Global(context.Background(), uuid.New(), "a", DefaultScope(), 20)
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
	_, err = svc.Global(context.Background(), uuid.New(), "  ", DefaultScope(), 20)
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery for spaces, got %v", err)
	}
}

func TestService_Global_AllScope_FansOut(t *testing.T) {
	fp := &fakeProvider{
		users:    []*models.User{{ID: uuid.New(), Name: "Alice"}},
		rooms:    []*models.Room{{ID: uuid.New(), Name: "general"}},
		messages: []*MessageHit{{RoomID: uuid.New(), Highlight: "<mark>x</mark>"}},
		files:    []*FileHit{{FileName: "report.pdf"}},
		meetings: []*MeetingHit{{Subject: "Daily"}},
	}
	svc := NewService(fp)
	res, err := svc.Global(context.Background(), uuid.New(), "report", DefaultScope(), 20)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(res.Users) != 1 || len(res.Rooms) != 1 || len(res.Messages) != 1 || len(res.Files) != 1 || len(res.Meetings) != 1 {
		t.Fatalf("expected one item per type, got %+v", res)
	}
	if fp.calledUsers != 1 || fp.calledRooms != 1 || fp.calledMessages != 1 || fp.calledFiles != 1 || fp.calledMeetings != 1 {
		t.Fatalf("expected one call per provider method, got %+v", fp)
	}
}

func TestService_Global_PartialScope_OnlyCallsRequested(t *testing.T) {
	fp := &fakeProvider{}
	svc := NewService(fp)
	_, err := svc.Global(context.Background(), uuid.New(), "report", Scope{Users: true, Files: true}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if fp.calledUsers != 1 || fp.calledFiles != 1 {
		t.Fatalf("expected users+files called, got %+v", fp)
	}
	if fp.calledRooms != 0 || fp.calledMessages != 0 || fp.calledMeetings != 0 {
		t.Fatalf("expected non-requested methods not called, got %+v", fp)
	}
}

func TestService_Global_PropagatesProviderError(t *testing.T) {
	fp := &fakeProvider{usersErr: errors.New("boom")}
	svc := NewService(fp)
	_, err := svc.Global(context.Background(), uuid.New(), "report", Scope{Users: true}, 20)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
}

func TestService_LocalMessages_PassesRoomID(t *testing.T) {
	roomID := uuid.New()
	fp := &fakeProvider{messages: []*MessageHit{{RoomID: roomID}}}
	svc := NewService(fp)
	hits, err := svc.LocalMessages(context.Background(), uuid.New(), roomID, "report", MessageSearchOpts{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if fp.gotRoomID == nil || *fp.gotRoomID != roomID {
		t.Fatalf("expected roomID %s passed to provider, got %v", roomID, fp.gotRoomID)
	}
}

func TestScope_IsEmpty(t *testing.T) {
	if !((Scope{}).IsEmpty()) {
		t.Error("zero Scope should be empty")
	}
	if (Scope{Users: true}).IsEmpty() {
		t.Error("Users=true Scope should not be empty")
	}
	if DefaultScope().IsEmpty() {
		t.Error("DefaultScope should not be empty")
	}
}

func TestCountRunes(t *testing.T) {
	if CountRunes("") != 0 {
		t.Error("empty string")
	}
	if CountRunes("abc") != 3 {
		t.Error("ascii")
	}
	if CountRunes("привет") != 6 {
		t.Error("cyrillic")
	}
}
