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

func TestEWSConfig(t *testing.T) {
	cfg := EWSConfig{
		URL:           "https://exchange.local/EWS/Exchange.asmx",
		Username:      "svc_focus",
		Password:      "secret",
		Domain:        "CORP",
		Impersonation: true,
	}

	assert.Equal(t, "https://exchange.local/EWS/Exchange.asmx", cfg.URL)
	assert.Equal(t, "svc_focus", cfg.Username)
	assert.Equal(t, "secret", cfg.Password)
	assert.Equal(t, "CORP", cfg.Domain)
	assert.True(t, cfg.Impersonation)
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

func TestBuildRequiredAttendeesXML(t *testing.T) {
	xml := buildRequiredAttendeesXML([]EventAttendee{
		{Email: "a@example.com", Name: "A"},
		{Email: "b@example.com"},
	})
	assert.Contains(t, xml, "<t:RequiredAttendees>")
	assert.Contains(t, xml, "a@example.com")
	assert.Contains(t, xml, "b@example.com")
}

func TestExtractJitsiURL(t *testing.T) {
	body := "Описание\nСсылка на встречу Focus: https://meet.focus.local/room-1\nДетали"
	assert.Equal(t, "https://meet.focus.local/room-1", extractJitsiURL(body))
}

func TestWithDomain(t *testing.T) {
	client := &EWSClient{domain: "CORP"}
	assert.Equal(t, `CORP\svc_focus`, client.withDomain("svc_focus"))
	assert.Equal(t, `CORP\svc_focus`, client.withDomain(`CORP\svc_focus`))
}

func TestParseEWSTime(t *testing.T) {
	t1, err := parseEWSTime("2030-01-01T10:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, 2030, t1.Year())
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

func TestEWSClientCreationSkipped(t *testing.T) {
	t.Skip("Skipping integration test - requires on-prem Exchange credentials")
}

func TestEWSClientCreateEventSkipped(t *testing.T) {
	t.Skip("Skipping integration test - requires on-prem Exchange credentials")
}

func TestEWSClientGetEventsSkipped(t *testing.T) {
	t.Skip("Skipping integration test - requires on-prem Exchange credentials")
}
