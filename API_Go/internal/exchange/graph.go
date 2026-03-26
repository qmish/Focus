package exchange

import (
	"context"
	"fmt"
	"time"
)

type GraphClient struct {
	tenantID string
}

type GraphConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
}

func NewGraphClient(cfg GraphConfig) (*GraphClient, error) {
	return &GraphClient{tenantID: cfg.TenantID}, nil
}

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

type EventAttendee struct {
	Email  string
	Name   string
	Status string
}

func (gc *GraphClient) CreateEvent(ctx context.Context, userID string, event CalendarEvent) (*CalendarEvent, error) {
	return nil, fmt.Errorf("exchange integration not configured")
}

func (gc *GraphClient) GetEvents(ctx context.Context, userID string, start, end time.Time) ([]CalendarEvent, error) {
	return []CalendarEvent{}, nil
}

func (gc *GraphClient) GetEvent(ctx context.Context, userID, eventID string) (*CalendarEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

func (gc *GraphClient) UpdateEvent(ctx context.Context, userID, eventID string, event CalendarEvent) error {
	return fmt.Errorf("not implemented")
}

func (gc *GraphClient) DeleteEvent(ctx context.Context, userID, eventID string) error {
	return fmt.Errorf("not implemented")
}
