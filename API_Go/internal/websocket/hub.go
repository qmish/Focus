package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// MessageType тип сообщения WebSocket
type MessageType string

const (
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypeMessage     MessageType = "message"
	MessageTypeTyping      MessageType = "typing"
	MessageTypeUserJoined  MessageType = "user_joined"
	MessageTypeUserLeft    MessageType = "user_left"
	MessageTypeError       MessageType = "error"
)

// WSMessage сообщение WebSocket
type WSMessage struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}

// SubscribePayload payload для подписки
type SubscribePayload struct {
	RoomID string `json:"room_id"`
}

// MessagePayload payload для сообщения
type MessagePayload struct {
	RoomID  string          `json:"room_id"`
	UserID  string          `json:"user_id"`
	Content string          `json:"content"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// TypingPayload payload для typing indicator
type TypingPayload struct {
	RoomID   string `json:"room_id"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	IsTyping bool   `json:"is_typing"`
}

// UserEventPayload payload для событий пользователя
type UserEventPayload struct {
	RoomID   string `json:"room_id"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

// ErrorPayload payload для ошибок
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Client представляет WebSocket клиента
type Client struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	Conn      *websocket.Conn
	Hub       *Hub
	Rooms     map[string]bool // комнаты, на которые подписан
	Send      chan []byte
	mu        sync.RWMutex
	closed    bool
}

// Hub управляет клиентами
type Hub struct {
	Clients          map[string]*Client
	Rooms            map[string]map[string]*Client // room_id -> client_id -> client
	Register         chan *Client
	Unregister       chan *Client
	Broadcast        chan *BroadcastMessage
	mu               sync.RWMutex
	logger           *zap.Logger
	roomAccess       RoomAccessChecker
	allowedOrigins   map[string]struct{}
	strictRoomAccess bool
}

// RoomAccessChecker checks if user can access a room.
type RoomAccessChecker func(ctx context.Context, userID, roomID string) (bool, error)

// BroadcastMessage сообщение для рассылки
type BroadcastMessage struct {
	RoomID  string
	Message []byte
}

// baseUpgrader for websocket connections.
var baseUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// NewHub создаёт новый Hub
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		Clients:          make(map[string]*Client),
		Rooms:            make(map[string]map[string]*Client),
		Register:         make(chan *Client),
		Unregister:       make(chan *Client),
		Broadcast:        make(chan *BroadcastMessage, 256),
		logger:           logger,
		allowedOrigins:   map[string]struct{}{},
		strictRoomAccess: false,
	}
}

// SetRoomAccessChecker sets room access policy for websocket subscriptions.
func (h *Hub) SetRoomAccessChecker(checker RoomAccessChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.roomAccess = checker
}

// SetAllowedOrigins configures websocket origin allowlist.
func (h *Hub) SetAllowedOrigins(origins []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allowedOrigins = make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		normalized := normalizeOrigin(origin)
		if normalized == "" {
			continue
		}
		h.allowedOrigins[normalized] = struct{}{}
	}
}

// SetStrictRoomAccess enables fail-closed room authorization mode.
func (h *Hub) SetStrictRoomAccess(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.strictRoomAccess = enabled
}

// Run запускает Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case msg := <-h.Broadcast:
			h.broadcastToRoom(msg.RoomID, msg.Message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Clients[client.ID] = client
	h.logger.Debug("client registered", zap.String("client_id", client.ID))
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.Clients[client.ID]; ok {
		delete(h.Clients, client.ID)
		close(client.Send)

		// Отписываем от всех комнат
		for roomID := range client.Rooms {
			h.unsubscribeFromRoom(client, roomID)
		}

		h.logger.Debug("client unregistered", zap.String("client_id", client.ID))
	}
}

func (h *Hub) broadcastToRoom(roomID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.Rooms[roomID]
	if !ok {
		return
	}

	for _, client := range clients {
		select {
		case client.Send <- message:
		default:
			// Клиент не готов, закрываем
			h.unregisterClient(client)
		}
	}
}

// SubscribeToRoom подписывает клиента на комнату
func (h *Hub) SubscribeToRoom(client *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.Rooms[roomID]; !ok {
		h.Rooms[roomID] = make(map[string]*Client)
	}

	h.Rooms[roomID][client.ID] = client
	client.mu.Lock()
	client.Rooms[roomID] = true
	client.mu.Unlock()

	h.logger.Debug("client subscribed to room",
		zap.String("client_id", client.ID),
		zap.String("room_id", roomID))
}

// UnsubscribeFromRoom отписывает клиента от комнаты
func (h *Hub) UnsubscribeFromRoom(client *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.unsubscribeFromRoom(client, roomID)
}

func (h *Hub) unsubscribeFromRoom(client *Client, roomID string) {
	if rooms, ok := h.Rooms[roomID]; ok {
		delete(rooms, client.ID)
		client.mu.Lock()
		delete(client.Rooms, roomID)
		client.mu.Unlock()

		if len(rooms) == 0 {
			delete(h.Rooms, roomID)
		}

		h.logger.Debug("client unsubscribed from room",
			zap.String("client_id", client.ID),
			zap.String("room_id", roomID))
	}
}

// BroadcastToRoom рассылает сообщение в комнату
func (h *Hub) BroadcastToRoom(roomID string, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.Broadcast <- &BroadcastMessage{
		RoomID:  roomID,
		Message: data,
	}
}

// HandleWebSocket обрабатывает WebSocket подключение
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, userID string, expiresAt time.Time) {
	upgrader := baseUpgrader
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return h.isOriginAllowed(r)
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	client := &Client{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: expiresAt,
		Conn:      conn,
		Hub:       h,
		Rooms:     make(map[string]bool),
		Send:      make(chan []byte, 256),
		closed:    false,
	}

	h.Register <- client

	// Запускаем обработчики
	go client.writePump()
	go client.readPump()
}

// writePump отправляет сообщения клиенту
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		if c.tokenExpired() {
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			_ = c.Conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "token_expired"),
			)
			return
		}

		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump получает сообщения от клиента
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512 * 1024) // 512 KB
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error("websocket error", zap.Error(err))
			}
			break
		}

		c.handleMessage(message)
	}
}

func (c *Client) handleMessage(data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("invalid_message", "failed to parse message")
		return
	}

	switch msg.Type {
	case MessageTypeSubscribe:
		c.handleSubscribe(msg.Payload)
	case MessageTypeUnsubscribe:
		c.handleUnsubscribe(msg.Payload)
	case MessageTypeMessage:
		c.handleMessagePayload(msg.Payload)
	case MessageTypeTyping:
		c.handleTyping(msg.Payload)
	default:
		c.sendError("unknown_type", "unknown message type")
	}
}

func (c *Client) handleSubscribe(payload json.RawMessage) {
	var sub SubscribePayload
	if err := json.Unmarshal(payload, &sub); err != nil {
		c.sendError("invalid_payload", "failed to parse subscribe payload")
		return
	}
	if sub.RoomID == "" {
		c.sendError("invalid_payload", "room_id is required")
		return
	}
	if !c.Hub.canAccessRoom(context.Background(), c.UserID, sub.RoomID) {
		c.sendError("forbidden_room", "access denied to room")
		return
	}

	c.Hub.SubscribeToRoom(c, sub.RoomID)

	// Отправляем подтверждение
	c.sendWSMessage(WSMessage{
		Type:    MessageTypeSubscribe,
		Payload: json.RawMessage(`{"room_id":"` + sub.RoomID + `","status":"subscribed"}`),
	})
}

func (c *Client) handleUnsubscribe(payload json.RawMessage) {
	var sub SubscribePayload
	if err := json.Unmarshal(payload, &sub); err != nil {
		c.sendError("invalid_payload", "failed to parse unsubscribe payload")
		return
	}

	c.Hub.UnsubscribeFromRoom(c, sub.RoomID)
}

func (c *Client) handleMessagePayload(payload json.RawMessage) {
	var msg MessagePayload
	if err := json.Unmarshal(payload, &msg); err != nil {
		c.sendError("invalid_payload", "failed to parse message payload")
		return
	}
	if msg.RoomID == "" {
		c.sendError("invalid_payload", "room_id is required")
		return
	}
	if !c.isSubscribed(msg.RoomID) {
		c.sendError("not_subscribed", "subscribe to room before sending messages")
		return
	}
	msg.UserID = c.UserID

	sanitizedPayload, err := json.Marshal(msg)
	if err != nil {
		c.sendError("internal_error", "failed to process message payload")
		return
	}

	// Рассылаем сообщение в комнату
	c.Hub.BroadcastToRoom(msg.RoomID, WSMessage{
		Type:    MessageTypeMessage,
		Payload: sanitizedPayload,
	})
}

func (c *Client) handleTyping(payload json.RawMessage) {
	var typing TypingPayload
	if err := json.Unmarshal(payload, &typing); err != nil {
		c.sendError("invalid_payload", "failed to parse typing payload")
		return
	}
	if typing.RoomID == "" {
		c.sendError("invalid_payload", "room_id is required")
		return
	}
	if !c.isSubscribed(typing.RoomID) {
		c.sendError("not_subscribed", "subscribe to room before sending typing events")
		return
	}
	typing.UserID = c.UserID

	sanitizedPayload, err := json.Marshal(typing)
	if err != nil {
		c.sendError("internal_error", "failed to process typing payload")
		return
	}

	// Рассылаем typing indicator в комнату
	c.Hub.BroadcastToRoom(typing.RoomID, WSMessage{
		Type:    MessageTypeTyping,
		Payload: sanitizedPayload,
	})
}

func (c *Client) sendWSMessage(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case c.Send <- data:
	default:
		// Канал переполнен
	}
}

func (c *Client) sendError(code, message string) {
	c.sendWSMessage(WSMessage{
		Type:    MessageTypeError,
		Payload: json.RawMessage(`{"code":"` + code + `","message":"` + message + `"}`),
	})
}

func (h *Hub) canAccessRoom(ctx context.Context, userID, roomID string) bool {
	h.mu.RLock()
	checker := h.roomAccess
	strictRoomAccess := h.strictRoomAccess
	h.mu.RUnlock()
	if checker == nil {
		return !strictRoomAccess
	}
	allowed, err := checker(ctx, userID, roomID)
	if err != nil {
		h.logger.Warn("room access check failed",
			zap.String("user_id", userID),
			zap.String("room_id", roomID),
			zap.Error(err))
		return false
	}
	return allowed
}

func (h *Hub) isOriginAllowed(r *http.Request) bool {
	originHeader := normalizeOrigin(r.Header.Get("Origin"))
	if originHeader == "" {
		return true
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.allowedOrigins) == 0 {
		return false
	}
	_, ok := h.allowedOrigins[originHeader]
	return ok
}

func normalizeOrigin(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func (c *Client) isSubscribed(roomID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Rooms[roomID]
}

func (c *Client) tokenExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

// BroadcastUserJoined рассылает событие о присоединении пользователя
func (h *Hub) BroadcastUserJoined(ctx context.Context, roomID, userID, userName string) {
	payload, _ := json.Marshal(UserEventPayload{
		RoomID:   roomID,
		UserID:   userID,
		UserName: userName,
	})

	h.BroadcastToRoom(roomID, WSMessage{
		Type:    MessageTypeUserJoined,
		Payload: payload,
	})
}

// BroadcastUserLeft рассылает событие о выходе пользователя
func (h *Hub) BroadcastUserLeft(ctx context.Context, roomID, userID, userName string) {
	payload, _ := json.Marshal(UserEventPayload{
		RoomID:   roomID,
		UserID:   userID,
		UserName: userName,
	})

	h.BroadcastToRoom(roomID, WSMessage{
		Type:    MessageTypeUserLeft,
		Payload: payload,
	})
}
