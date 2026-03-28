package exchange

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-ntlmssp"
	krb5client "github.com/jcmturner/gokrb5/v8/client"
	krb5config "github.com/jcmturner/gokrb5/v8/config"
	krb5keytab "github.com/jcmturner/gokrb5/v8/keytab"
	krb5spnego "github.com/jcmturner/gokrb5/v8/spnego"
)

const (
	defaultEWSTimeout = 15 * time.Second
	ewsSOAPNS         = "http://schemas.xmlsoap.org/soap/envelope/"
	ewsMessagesNS     = "http://schemas.microsoft.com/exchange/services/2006/messages"
	ewsTypesNS        = "http://schemas.microsoft.com/exchange/services/2006/types"
)

// CalendarService describes calendar CRUD operations used by handlers and bots.
type CalendarService interface {
	GetEvents(ctx context.Context, userEmail string, start, end time.Time) ([]CalendarEvent, error)
	CreateEvent(ctx context.Context, userEmail string, event CalendarEvent) (*CalendarEvent, error)
	GetEvent(ctx context.Context, userEmail, eventID string) (*CalendarEvent, error)
	UpdateEvent(ctx context.Context, userEmail, eventID string, event CalendarEvent) error
	DeleteEvent(ctx context.Context, userEmail, eventID string) error
}

// EWSClient is an on-prem Exchange Web Services client.
type EWSClient struct {
	url           string
	username      string
	password      string
	domain        string
	authMode      string
	impersonation bool
	httpClient    soapHTTPClient
}

// EWSConfig configures EWS client connectivity and auth.
type EWSConfig struct {
	URL            string
	Username       string
	Password       string
	Domain         string
	AuthMode       string
	CACertPath     string
	InsecureTLS    bool
	Krb5ConfigPath string
	Krb5KeytabPath string
	Krb5Realm      string
	Krb5SPN        string
	Impersonation  bool
	Timeout        time.Duration
}

type soapHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// CalendarEvent describes Exchange calendar event in API-level model.
type CalendarEvent struct {
	ID              string          `json:"id"`
	Subject         string          `json:"subject"`
	Description     string          `json:"description,omitempty"`
	StartTime       time.Time       `json:"start_time"`
	EndTime         time.Time       `json:"end_time"`
	Location        string          `json:"location,omitempty"`
	Organizer       EventAttendee   `json:"organizer"`
	Attendees       []EventAttendee `json:"attendees,omitempty"`
	JitsiURL        string          `json:"jitsi_url,omitempty"`
	ExchangeEventID string          `json:"exchange_event_id,omitempty"`
	RoomID          string          `json:"room_id,omitempty"`
	SyncStatus      string          `json:"sync_status,omitempty"`
}

// EventAttendee describes attendee mail identity/status.
type EventAttendee struct {
	Email  string `json:"email"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

func NewEWSClient(cfg EWSConfig) (*EWSClient, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("EWS URL is required")
	}
	authMode := strings.ToLower(strings.TrimSpace(cfg.AuthMode))
	if authMode == "" {
		authMode = "basic"
	}
	if authMode != "basic" && authMode != "ntlm" && authMode != "kerberos" {
		return nil, fmt.Errorf("unsupported EXCHANGE_AUTH_MODE: %s", authMode)
	}
	if strings.TrimSpace(cfg.Username) == "" {
		return nil, fmt.Errorf("EWS username is required")
	}
	if authMode != "kerberos" && strings.TrimSpace(cfg.Password) == "" {
		return nil, fmt.Errorf("EWS password is required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultEWSTimeout
	}
	httpClient, err := buildHTTPClient(timeout, authMode, cfg)
	if err != nil {
		return nil, err
	}
	return &EWSClient{
		url:           strings.TrimSpace(cfg.URL),
		username:      strings.TrimSpace(cfg.Username),
		password:      cfg.Password,
		domain:        strings.TrimSpace(cfg.Domain),
		authMode:      authMode,
		impersonation: cfg.Impersonation,
		httpClient:    httpClient,
	}, nil
}

func (c *EWSClient) GetEvents(ctx context.Context, userEmail string, start, end time.Time) ([]CalendarEvent, error) {
	if end.Before(start) {
		return nil, fmt.Errorf("invalid time range")
	}
	body := fmt.Sprintf(`
<m:FindItem Traversal="Shallow">
  <m:ItemShape>
    <t:BaseShape>AllProperties</t:BaseShape>
  </m:ItemShape>
  <m:CalendarView StartDate="%s" EndDate="%s"/>
  <m:ParentFolderIds>
    <t:DistinguishedFolderId Id="calendar"/>
  </m:ParentFolderIds>
</m:FindItem>`, xmlEscape(start.UTC().Format(time.RFC3339)), xmlEscape(end.UTC().Format(time.RFC3339)))

	respBody, err := c.doSOAP(ctx, "FindItem", body, userEmail)
	if err != nil {
		return nil, err
	}
	var envelope ewsFindItemEnvelope
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse FindItem response: %w", err)
	}
	message := envelope.Body.FindItemResponse.ResponseMessages.FindItemResponseMessage
	if !strings.EqualFold(message.ResponseCode, "NoError") {
		return nil, fmt.Errorf("FindItem failed: %s", message.ResponseCode)
	}
	events := make([]CalendarEvent, 0, len(message.RootFolder.Items.CalendarItems))
	for _, item := range message.RootFolder.Items.CalendarItems {
		events = append(events, mapCalendarItem(item))
	}
	return events, nil
}

func (c *EWSClient) CreateEvent(ctx context.Context, userEmail string, event CalendarEvent) (*CalendarEvent, error) {
	description := event.Description
	if strings.TrimSpace(event.JitsiURL) != "" {
		if strings.TrimSpace(description) != "" {
			description += "\n\n"
		}
		description += "Ссылка на встречу Focus: " + strings.TrimSpace(event.JitsiURL)
	}
	body := fmt.Sprintf(`
<m:CreateItem SendMeetingInvitations="SendToAllAndSaveCopy">
  <m:SavedItemFolderId>
    <t:DistinguishedFolderId Id="calendar"/>
  </m:SavedItemFolderId>
  <m:Items>
    <t:CalendarItem>
      <t:Subject>%s</t:Subject>
      <t:Body BodyType="Text">%s</t:Body>
      <t:Start>%s</t:Start>
      <t:End>%s</t:End>
      <t:Location>%s</t:Location>
      %s
    </t:CalendarItem>
  </m:Items>
</m:CreateItem>`,
		xmlEscape(event.Subject),
		xmlEscape(description),
		xmlEscape(event.StartTime.UTC().Format(time.RFC3339)),
		xmlEscape(event.EndTime.UTC().Format(time.RFC3339)),
		xmlEscape(event.Location),
		buildRequiredAttendeesXML(event.Attendees),
	)

	respBody, err := c.doSOAP(ctx, "CreateItem", body, userEmail)
	if err != nil {
		return nil, err
	}
	var envelope ewsCreateItemEnvelope
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse CreateItem response: %w", err)
	}
	message := envelope.Body.CreateItemResponse.ResponseMessages.CreateItemResponseMessage
	if !strings.EqualFold(message.ResponseCode, "NoError") {
		return nil, fmt.Errorf("CreateItem failed: %s", message.ResponseCode)
	}
	created := mapCalendarItem(message.Items.CalendarItem)
	if created.ID == "" {
		return nil, fmt.Errorf("CreateItem succeeded but item id is empty")
	}
	if created.Subject == "" {
		created.Subject = event.Subject
	}
	if created.StartTime.IsZero() {
		created.StartTime = event.StartTime
	}
	if created.EndTime.IsZero() {
		created.EndTime = event.EndTime
	}
	if created.Location == "" {
		created.Location = event.Location
	}
	if created.Description == "" {
		created.Description = description
	}
	created.JitsiURL = event.JitsiURL
	created.Attendees = event.Attendees
	created.Organizer = event.Organizer
	return &created, nil
}

func (c *EWSClient) GetEvent(ctx context.Context, userEmail, eventID string) (*CalendarEvent, error) {
	if strings.TrimSpace(eventID) == "" {
		return nil, fmt.Errorf("event id is required")
	}
	body := fmt.Sprintf(`
<m:GetItem>
  <m:ItemShape>
    <t:BaseShape>AllProperties</t:BaseShape>
  </m:ItemShape>
  <m:ItemIds>
    <t:ItemId Id="%s"/>
  </m:ItemIds>
</m:GetItem>`, xmlEscape(eventID))

	respBody, err := c.doSOAP(ctx, "GetItem", body, userEmail)
	if err != nil {
		return nil, err
	}
	var envelope ewsGetItemEnvelope
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse GetItem response: %w", err)
	}
	message := envelope.Body.GetItemResponse.ResponseMessages.GetItemResponseMessage
	if !strings.EqualFold(message.ResponseCode, "NoError") {
		return nil, fmt.Errorf("GetItem failed: %s", message.ResponseCode)
	}
	item := mapCalendarItem(message.Items.CalendarItem)
	if item.ID == "" {
		item.ID = eventID
	}
	return &item, nil
}

func (c *EWSClient) UpdateEvent(ctx context.Context, userEmail, eventID string, event CalendarEvent) error {
	if strings.TrimSpace(eventID) == "" {
		return fmt.Errorf("event id is required")
	}
	changes := make([]string, 0, 6)
	if strings.TrimSpace(event.Subject) != "" {
		changes = append(changes, fmt.Sprintf(`<t:SetItemField><t:FieldURI FieldURI="item:Subject"/><t:CalendarItem><t:Subject>%s</t:Subject></t:CalendarItem></t:SetItemField>`, xmlEscape(event.Subject)))
	}
	if strings.TrimSpace(event.Description) != "" {
		changes = append(changes, fmt.Sprintf(`<t:SetItemField><t:FieldURI FieldURI="item:Body"/><t:CalendarItem><t:Body BodyType="Text">%s</t:Body></t:CalendarItem></t:SetItemField>`, xmlEscape(event.Description)))
	}
	if !event.StartTime.IsZero() {
		changes = append(changes, fmt.Sprintf(`<t:SetItemField><t:FieldURI FieldURI="calendar:Start"/><t:CalendarItem><t:Start>%s</t:Start></t:CalendarItem></t:SetItemField>`, xmlEscape(event.StartTime.UTC().Format(time.RFC3339))))
	}
	if !event.EndTime.IsZero() {
		changes = append(changes, fmt.Sprintf(`<t:SetItemField><t:FieldURI FieldURI="calendar:End"/><t:CalendarItem><t:End>%s</t:End></t:CalendarItem></t:SetItemField>`, xmlEscape(event.EndTime.UTC().Format(time.RFC3339))))
	}
	if strings.TrimSpace(event.Location) != "" {
		changes = append(changes, fmt.Sprintf(`<t:SetItemField><t:FieldURI FieldURI="calendar:Location"/><t:CalendarItem><t:Location>%s</t:Location></t:CalendarItem></t:SetItemField>`, xmlEscape(event.Location)))
	}
	if len(event.Attendees) > 0 {
		changes = append(changes, fmt.Sprintf(`<t:SetItemField><t:FieldURI FieldURI="calendar:RequiredAttendees"/><t:CalendarItem>%s</t:CalendarItem></t:SetItemField>`, buildRequiredAttendeesXML(event.Attendees)))
	}
	if len(changes) == 0 {
		return nil
	}
	body := fmt.Sprintf(`
<m:UpdateItem ConflictResolution="AutoResolve" MessageDisposition="SaveOnly" SendMeetingInvitationsOrCancellations="SendToAllAndSaveCopy">
  <m:ItemChanges>
    <t:ItemChange>
      <t:ItemId Id="%s"/>
      <t:Updates>%s</t:Updates>
    </t:ItemChange>
  </m:ItemChanges>
</m:UpdateItem>`, xmlEscape(eventID), strings.Join(changes, ""))
	respBody, err := c.doSOAP(ctx, "UpdateItem", body, userEmail)
	if err != nil {
		return err
	}
	var envelope ewsSimpleResponseEnvelope
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("failed to parse UpdateItem response: %w", err)
	}
	if !strings.EqualFold(envelope.Body.UpdateItemResponse.ResponseMessages.UpdateItemResponseMessage.ResponseCode, "NoError") {
		return fmt.Errorf("UpdateItem failed: %s", envelope.Body.UpdateItemResponse.ResponseMessages.UpdateItemResponseMessage.ResponseCode)
	}
	return nil
}

func (c *EWSClient) DeleteEvent(ctx context.Context, userEmail, eventID string) error {
	if strings.TrimSpace(eventID) == "" {
		return fmt.Errorf("event id is required")
	}
	body := fmt.Sprintf(`
<m:DeleteItem DeleteType="MoveToDeletedItems" SendMeetingCancellations="SendToAllAndSaveCopy" AffectedTaskOccurrences="AllOccurrences">
  <m:ItemIds>
    <t:ItemId Id="%s"/>
  </m:ItemIds>
</m:DeleteItem>`, xmlEscape(eventID))
	respBody, err := c.doSOAP(ctx, "DeleteItem", body, userEmail)
	if err != nil {
		return err
	}
	var envelope ewsSimpleResponseEnvelope
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("failed to parse DeleteItem response: %w", err)
	}
	if !strings.EqualFold(envelope.Body.DeleteItemResponse.ResponseMessages.DeleteItemResponseMessage.ResponseCode, "NoError") {
		return fmt.Errorf("DeleteItem failed: %s", envelope.Body.DeleteItemResponse.ResponseMessages.DeleteItemResponseMessage.ResponseCode)
	}
	return nil
}

func (c *EWSClient) doSOAP(ctx context.Context, action, body, userEmail string) ([]byte, error) {
	envelope := c.wrapEnvelope(body, userEmail)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewBufferString(envelope))
	if err != nil {
		return nil, err
	}
	if c.authMode == "basic" || c.authMode == "ntlm" {
		req.SetBasicAuth(c.withDomain(c.username), c.password)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("Accept", "text/xml")
	req.Header.Set("SOAPAction", fmt.Sprintf("http://schemas.microsoft.com/exchange/services/2006/messages/%s", action))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("EWS %s failed: status=%d body=%s", action, resp.StatusCode, truncate(string(respBody), 300))
	}
	return respBody, nil
}

func (c *EWSClient) wrapEnvelope(body, userEmail string) string {
	impersonation := ""
	if c.impersonation && strings.TrimSpace(userEmail) != "" {
		impersonation = fmt.Sprintf(`<t:ExchangeImpersonation><t:ConnectingSID><t:PrimarySmtpAddress>%s</t:PrimarySmtpAddress></t:ConnectingSID></t:ExchangeImpersonation>`, xmlEscape(userEmail))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="%s" xmlns:m="%s" xmlns:t="%s">
  <soap:Header>
    <t:RequestServerVersion Version="Exchange2016"/>
    %s
  </soap:Header>
  <soap:Body>%s</soap:Body>
</soap:Envelope>`, ewsSOAPNS, ewsMessagesNS, ewsTypesNS, impersonation, body)
}

func (c *EWSClient) withDomain(username string) string {
	if c.domain == "" || strings.Contains(username, "\\") {
		return username
	}
	return c.domain + `\` + username
}

func buildRequiredAttendeesXML(attendees []EventAttendee) string {
	if len(attendees) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("<t:RequiredAttendees>")
	for _, attendee := range attendees {
		if strings.TrimSpace(attendee.Email) == "" {
			continue
		}
		builder.WriteString("<t:Attendee><t:Mailbox>")
		builder.WriteString("<t:EmailAddress>" + xmlEscape(attendee.Email) + "</t:EmailAddress>")
		if strings.TrimSpace(attendee.Name) != "" {
			builder.WriteString("<t:Name>" + xmlEscape(attendee.Name) + "</t:Name>")
		}
		builder.WriteString("</t:Mailbox></t:Attendee>")
	}
	builder.WriteString("</t:RequiredAttendees>")
	return builder.String()
}

func mapCalendarItem(item ewsCalendarItem) CalendarEvent {
	start, _ := parseEWSTime(item.Start)
	end, _ := parseEWSTime(item.End)
	result := CalendarEvent{
		ID:          item.ItemID.ID,
		Subject:     item.Subject,
		Description: strings.TrimSpace(item.Body.Value),
		StartTime:   start,
		EndTime:     end,
		Location:    item.Location.DisplayName,
		Organizer: EventAttendee{
			Email: item.Organizer.Mailbox.EmailAddress,
			Name:  item.Organizer.Mailbox.Name,
		},
		Attendees: extractAttendees(item.RequiredAttendees),
	}
	if result.Location == "" {
		result.Location = item.Location.Value
	}
	result.JitsiURL = extractJitsiURL(result.Description)
	return result
}

func extractAttendees(required ewsRequiredAttendees) []EventAttendee {
	result := make([]EventAttendee, 0, len(required.Attendees))
	for _, attendee := range required.Attendees {
		result = append(result, EventAttendee{
			Email:  attendee.Mailbox.EmailAddress,
			Name:   attendee.Mailbox.Name,
			Status: attendee.ResponseType,
		})
	}
	return result
}

func extractJitsiURL(description string) string {
	for _, line := range strings.Split(description, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "http://") || strings.Contains(trimmed, "https://") {
			if strings.Contains(trimmed, "meet") || strings.Contains(strings.ToLower(trimmed), "focus") {
				return strings.TrimSpace(strings.TrimPrefix(trimmed, "Ссылка на встречу Focus:"))
			}
		}
	}
	return ""
}

func parseEWSTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported datetime %q", value)
}

func xmlEscape(value string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(value)); err != nil {
		return value
	}
	return buf.String()
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func buildHTTPClient(timeout time.Duration, authMode string, cfg EWSConfig) (soapHTTPClient, error) {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfg.InsecureTLS,
	}
	if strings.TrimSpace(cfg.CACertPath) != "" {
		pemBytes, err := os.ReadFile(cfg.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read EXCHANGE_CA_CERT_PATH: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if ok := pool.AppendCertsFromPEM(pemBytes); !ok {
			return nil, fmt.Errorf("failed to append custom CA certificate")
		}
		tlsConfig.RootCAs = pool
	}
	baseTransport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	if authMode == "ntlm" {
		return &http.Client{
			Timeout:   timeout,
			Transport: ntlmssp.Negotiator{RoundTripper: baseTransport},
		}, nil
	}
	baseClient := &http.Client{
		Timeout:   timeout,
		Transport: baseTransport,
	}
	if authMode != "kerberos" {
		return baseClient, nil
	}
	krbConfPath := strings.TrimSpace(cfg.Krb5ConfigPath)
	if krbConfPath == "" {
		return nil, fmt.Errorf("EXCHANGE_KRB5_CONFIG_PATH is required for kerberos auth")
	}
	krbRealm := strings.TrimSpace(cfg.Krb5Realm)
	if krbRealm == "" {
		return nil, fmt.Errorf("EXCHANGE_KRB5_REALM is required for kerberos auth")
	}
	krbConf, err := krb5config.Load(krbConfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load krb5 config: %w", err)
	}
	var krbClient *krb5client.Client
	if strings.TrimSpace(cfg.Krb5KeytabPath) != "" {
		kt, err := krb5keytab.Load(strings.TrimSpace(cfg.Krb5KeytabPath))
		if err != nil {
			return nil, fmt.Errorf("failed to load keytab: %w", err)
		}
		krbClient = krb5client.NewWithKeytab(strings.TrimSpace(cfg.Username), krbRealm, kt, krbConf, krb5client.DisablePAFXFAST(true))
	} else if strings.TrimSpace(cfg.Password) != "" {
		krbClient = krb5client.NewWithPassword(strings.TrimSpace(cfg.Username), krbRealm, cfg.Password, krbConf, krb5client.DisablePAFXFAST(true))
	} else {
		return nil, fmt.Errorf("for kerberos auth set EXCHANGE_KRB5_KEYTAB_PATH or EXCHANGE_PASSWORD")
	}
	if err := krbClient.Login(); err != nil {
		return nil, fmt.Errorf("kerberos login failed: %w", err)
	}
	return krb5spnego.NewClient(krbClient, baseClient, strings.TrimSpace(cfg.Krb5SPN)), nil
}

func (e CalendarEvent) MarshalJSON() ([]byte, error) {
	type alias CalendarEvent
	return json.Marshal(alias(e))
}

type ewsFindItemEnvelope struct {
	Body struct {
		FindItemResponse struct {
			ResponseMessages struct {
				FindItemResponseMessage struct {
					ResponseCode string `xml:"ResponseCode"`
					RootFolder   struct {
						Items struct {
							CalendarItems []ewsCalendarItem `xml:"CalendarItem"`
						} `xml:"Items"`
					} `xml:"RootFolder"`
				} `xml:"FindItemResponseMessage"`
			} `xml:"ResponseMessages"`
		} `xml:"FindItemResponse"`
	} `xml:"Body"`
}

type ewsGetItemEnvelope struct {
	Body struct {
		GetItemResponse struct {
			ResponseMessages struct {
				GetItemResponseMessage struct {
					ResponseCode string `xml:"ResponseCode"`
					Items        struct {
						CalendarItem ewsCalendarItem `xml:"CalendarItem"`
					} `xml:"Items"`
				} `xml:"GetItemResponseMessage"`
			} `xml:"ResponseMessages"`
		} `xml:"GetItemResponse"`
	} `xml:"Body"`
}

type ewsCreateItemEnvelope struct {
	Body struct {
		CreateItemResponse struct {
			ResponseMessages struct {
				CreateItemResponseMessage struct {
					ResponseCode string `xml:"ResponseCode"`
					Items        struct {
						CalendarItem ewsCalendarItem `xml:"CalendarItem"`
					} `xml:"Items"`
				} `xml:"CreateItemResponseMessage"`
			} `xml:"ResponseMessages"`
		} `xml:"CreateItemResponse"`
	} `xml:"Body"`
}

type ewsSimpleResponseEnvelope struct {
	Body struct {
		UpdateItemResponse struct {
			ResponseMessages struct {
				UpdateItemResponseMessage struct {
					ResponseCode string `xml:"ResponseCode"`
				} `xml:"UpdateItemResponseMessage"`
			} `xml:"ResponseMessages"`
		} `xml:"UpdateItemResponse"`
		DeleteItemResponse struct {
			ResponseMessages struct {
				DeleteItemResponseMessage struct {
					ResponseCode string `xml:"ResponseCode"`
				} `xml:"DeleteItemResponseMessage"`
			} `xml:"ResponseMessages"`
		} `xml:"DeleteItemResponse"`
	} `xml:"Body"`
}

type ewsCalendarItem struct {
	ItemID struct {
		ID        string `xml:"Id,attr"`
		ChangeKey string `xml:"ChangeKey,attr"`
	} `xml:"ItemId"`
	Subject  string `xml:"Subject"`
	Start    string `xml:"Start"`
	End      string `xml:"End"`
	Location struct {
		Value       string `xml:",chardata"`
		DisplayName string `xml:"DisplayName"`
	} `xml:"Location"`
	Body struct {
		Value string `xml:",chardata"`
	} `xml:"Body"`
	Organizer struct {
		Mailbox struct {
			Name         string `xml:"Name"`
			EmailAddress string `xml:"EmailAddress"`
		} `xml:"Mailbox"`
	} `xml:"Organizer"`
	RequiredAttendees ewsRequiredAttendees `xml:"RequiredAttendees"`
}

type ewsRequiredAttendees struct {
	Attendees []struct {
		ResponseType string `xml:"ResponseType"`
		Mailbox      struct {
			Name         string `xml:"Name"`
			EmailAddress string `xml:"EmailAddress"`
		} `xml:"Mailbox"`
	} `xml:"Attendee"`
}
