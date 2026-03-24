package exchange

import (
	"context"
	"fmt"
	"time"

	azidentity "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

// GraphClient клиент для работы с Microsoft Graph API
type GraphClient struct {
	client   *msgraph.GraphServiceClient
	tenantID string
}

// GraphConfig конфигурация Graph API
type GraphConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
}

// NewGraphClient создаёт новый GraphClient
func NewGraphClient(cfg GraphConfig) (*GraphClient, error) {
	cred, err := azidentity.NewClientSecretCredential(
		cfg.TenantID,
		cfg.ClientID,
		cfg.ClientSecret,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	client, err := msgraph.NewGraphServiceClientWithCredentials(
		cred,
		[]string{"https://graph.microsoft.com/.default"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &GraphClient{
		client:   client,
		tenantID: cfg.TenantID,
	}, nil
}

// CalendarEvent событие календаря
type CalendarEvent struct {
	ID              string
	Subject         string
	Description     string
	StartTime       time.Time
	EndTime         time.Time
	Location        string
	Organizer       EventAttendee
	Attendees       []EventAttendee
	JitsiURL        string
	ExchangeEventID string
}

// EventAttendee участник события
type EventAttendee struct {
	Email  string
	Name   string
	Status string
}

// CreateEvent создаёт событие в календаре
func (gc *GraphClient) CreateEvent(ctx context.Context, userID string, event CalendarEvent) (*CalendarEvent, error) {
	body := models.NewItemBody()
	contentType := models.HTML_BODYTYPE
	body.SetContentType(&contentType)

	bodyContent := event.Description
	if event.JitsiURL != "" {
		bodyContent += fmt.Sprintf("<br/><br/><a href='%s'>Присоединиться к встрече</a>", event.JitsiURL)
	}
	body.SetContent(&bodyContent)

	location := models.NewLocation()
	location.SetDisplayName(&event.Location)

	graphEvent := models.NewEvent()
	graphEvent.SetSubject(&event.Subject)
	graphEvent.SetBody(body)
	graphEvent.SetLocation(location)

	if len(event.Attendees) > 0 {
		attendees := make([]models.Attendeeable, 0, len(event.Attendees))
		for _, att := range event.Attendees {
			emailAddr := models.NewEmailAddress()
			email := att.Email
			emailAddr.SetAddress(&email)

			attModel := models.NewAttendee()
			attModel.SetEmailAddress(emailAddr)
			attendees = append(attendees, attModel)
		}
		graphEvent.SetAttendees(attendees)
	}

	result, err := gc.client.Me().Calendar().Events().Post(ctx, graphEvent, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return gc.convertEvent(result), nil
}

// GetEvents получает события календаря (упрощённая версия)
func (gc *GraphClient) GetEvents(ctx context.Context, userID string, start, end time.Time) ([]CalendarEvent, error) {
	// Заглушка - будет реализовано с правильной версией SDK
	return []CalendarEvent{}, nil
}

// convertEvent преобразует Graph событие в наше
func (gc *GraphClient) convertEvent(event models.Eventable) *CalendarEvent {
	if event == nil {
		return nil
	}

	calEvent := &CalendarEvent{
		Subject:     ptrToString(event.GetSubject()),
		Description: ptrToString(event.GetBody().GetContent()),
	}

	if id := event.GetId(); id != nil {
		calEvent.ID = *id
		calEvent.ExchangeEventID = *id
	}

	if loc := event.GetLocation(); loc != nil {
		calEvent.Location = ptrToString(loc.GetDisplayName())
	}

	return calEvent
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Заглушки для методов, которые будут реализованы позже
func (gc *GraphClient) GetEvent(ctx context.Context, userID, eventID string) (*CalendarEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

func (gc *GraphClient) UpdateEvent(ctx context.Context, userID, eventID string, event CalendarEvent) error {
	return fmt.Errorf("not implemented")
}

func (gc *GraphClient) DeleteEvent(ctx context.Context, userID, eventID string) error {
	return fmt.Errorf("not implemented")
}
