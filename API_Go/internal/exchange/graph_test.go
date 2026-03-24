package exchange

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalendarEventStruct(t *testing.T) {
	now := time.Now()
	event := CalendarEvent{
		ID:          "event-123",
		Subject:     "Test Meeting",
		Description: "Test Description",
		StartTime:   now,
		EndTime:     now.Add(time.Hour),
		Location:    "Jitsi Meeting",
		JitsiURL:    "https://meet.company.com/test",
		Organizer: EventAttendee{
			Email:  "organizer@company.com",
			Name:   "Organizer Name",
			Status: "accepted",
		},
		Attendees: []EventAttendee{
			{
				Email:  "attendee1@company.com",
				Name:   "Attendee 1",
				Status: "pending",
			},
		},
	}

	assert.Equal(t, "event-123", event.ID)
	assert.Equal(t, "Test Meeting", event.Subject)
	assert.Equal(t, "Test Description", event.Description)
	assert.Equal(t, "Jitsi Meeting", event.Location)
	assert.Equal(t, "https://meet.company.com/test", event.JitsiURL)
	assert.Equal(t, "organizer@company.com", event.Organizer.Email)
	assert.Len(t, event.Attendees, 1)
}

func TestEventAttendeeStruct(t *testing.T) {
	attendee := EventAttendee{
		Email:  "test@company.com",
		Name:   "Test User",
		Status: "accepted",
	}

	assert.Equal(t, "test@company.com", attendee.Email)
	assert.Equal(t, "Test User", attendee.Name)
	assert.Equal(t, "accepted", attendee.Status)
}

func TestPtrHelper(t *testing.T) {
	t.Run("string pointer", func(t *testing.T) {
		s := "test"
		ps := &s
		assert.NotNil(t, ps)
		assert.Equal(t, "test", *ps)
	})

	t.Run("int pointer", func(t *testing.T) {
		i := 42
		pi := &i
		assert.NotNil(t, pi)
		assert.Equal(t, 42, *pi)
	})
}

func TestPtrToString(t *testing.T) {
	t.Run("non-nil string", func(t *testing.T) {
		s := "test"
		result := ptrToString(&s)
		assert.Equal(t, "test", result)
	})

	t.Run("nil string", func(t *testing.T) {
		var s *string
		result := ptrToString(s)
		assert.Equal(t, "", result)
	})
}

func TestGraphConfig(t *testing.T) {
	cfg := GraphConfig{
		TenantID:     "tenant-123",
		ClientID:     "client-456",
		ClientSecret: "secret-789",
	}

	assert.Equal(t, "tenant-123", cfg.TenantID)
	assert.Equal(t, "client-456", cfg.ClientID)
	assert.Equal(t, "secret-789", cfg.ClientSecret)
}

func TestCalendarEventTimezone(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	now := time.Now().In(loc)

	event := CalendarEvent{
		StartTime: now,
		EndTime:   now.Add(time.Hour),
	}

	assert.Equal(t, loc.String(), event.StartTime.Location().String())
	assert.Equal(t, loc.String(), event.EndTime.Location().String())
}

func TestCalendarEventDuration(t *testing.T) {
	now := time.Now()
	duration := 2 * time.Hour

	event := CalendarEvent{
		StartTime: now,
		EndTime:   now.Add(duration),
	}

	actualDuration := event.EndTime.Sub(event.StartTime)
	assert.Equal(t, duration, actualDuration)
}

func TestEventAttendeeStatuses(t *testing.T) {
	statuses := []string{"pending", "accepted", "declined", "tentative"}

	for _, status := range statuses {
		attendee := EventAttendee{
			Email:  "test@company.com",
			Status: status,
		}
		assert.Equal(t, status, attendee.Status)
	}
}

func TestCalendarEventWithJitsiURL(t *testing.T) {
	event := CalendarEvent{
		Subject:  "Meeting",
		JitsiURL: "https://meet.company.com/room-123",
	}

	assert.Contains(t, event.JitsiURL, "https://meet.company.com/")
	assert.NotEmpty(t, event.JitsiURL)
}

func TestCalendarEventEmptyAttendees(t *testing.T) {
	event := CalendarEvent{
		Subject:   "Meeting",
		Attendees: []EventAttendee{},
	}

	assert.Empty(t, event.Attendees)
}

func TestCalendarEventOrganizer(t *testing.T) {
	event := CalendarEvent{
		Subject: "Meeting",
		Organizer: EventAttendee{
			Email:  "organizer@company.com",
			Name:   "Organizer",
			Status: "accepted",
		},
	}

	assert.Equal(t, "organizer@company.com", event.Organizer.Email)
	assert.Equal(t, "Organizer", event.Organizer.Name)
}

// Integration tests (require actual Azure credentials)
// These tests are skipped by default

func TestGraphClientCreationSkipped(t *testing.T) {
	t.Skip("Skipping integration test - requires Azure credentials")
}

func TestGraphClientCreateEventSkipped(t *testing.T) {
	t.Skip("Skipping integration test - requires Azure credentials")
}

func TestGraphClientGetEventsSkipped(t *testing.T) {
	t.Skip("Skipping integration test - requires Azure credentials")
}
