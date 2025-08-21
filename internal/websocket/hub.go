package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/drazan344/taskflow-go/pkg/logger"
)

// MessageType represents different types of WebSocket messages
type MessageType string

const (
	MessageTypeTaskUpdate      MessageType = "task_update"
	MessageTypeTaskCreate      MessageType = "task_create"
	MessageTypeTaskDelete      MessageType = "task_delete"
	MessageTypeNotification    MessageType = "notification"
	MessageTypeUserJoined      MessageType = "user_joined"
	MessageTypeUserLeft        MessageType = "user_left"
	MessageTypeTyping          MessageType = "typing"
	MessageTypePing            MessageType = "ping"
	MessageTypePong            MessageType = "pong"
	MessageTypeError           MessageType = "error"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType            `json:"type"`
	Data      interface{}            `json:"data,omitempty"`
	UserID    uuid.UUID              `json:"user_id"`
	TenantID  uuid.UUID              `json:"tenant_id"`
	Timestamp int64                  `json:"timestamp"`
	MessageID string                 `json:"message_id"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients by tenant
	tenants map[uuid.UUID]*TenantRoom

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Inbound messages from the clients
	broadcast chan *Message

	// Logger
	logger *logger.Logger

	// Mutex for thread safety
	mu sync.RWMutex
}

// TenantRoom represents a room for a specific tenant
type TenantRoom struct {
	// Tenant ID
	TenantID uuid.UUID

	// Connected clients in this tenant
	clients map[*Client]bool

	// Broadcast channel for this tenant
	broadcast chan *Message

	// Mutex for thread safety
	mu sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub(logger *logger.Logger) *Hub {
	return &Hub{
		tenants:    make(map[uuid.UUID]*TenantRoom),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
		logger:     logger,
	}
}

// Run starts the hub and handles client registration/unregistration
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Get or create tenant room
	room, exists := h.tenants[client.TenantID]
	if !exists {
		room = &TenantRoom{
			TenantID:  client.TenantID,
			clients:   make(map[*Client]bool),
			broadcast: make(chan *Message, 256),
		}
		h.tenants[client.TenantID] = room
		
		// Start the room's broadcast goroutine
		go room.run()
	}

	// Add client to room
	room.mu.Lock()
	room.clients[client] = true
	room.mu.Unlock()

	h.logger.WithFields(map[string]interface{}{
		"user_id":   client.UserID,
		"tenant_id": client.TenantID,
		"client_count": len(room.clients),
	}).Info("Client connected")

	// Notify other clients that a user joined
	joinMessage := &Message{
		Type:      MessageTypeUserJoined,
		UserID:    client.UserID,
		TenantID:  client.TenantID,
		Timestamp: getCurrentTimestamp(),
		MessageID: generateMessageID(),
		Data: map[string]interface{}{
			"user_id": client.UserID,
			"user_name": client.UserName,
		},
	}
	
	room.broadcast <- joinMessage
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.tenants[client.TenantID]
	if !exists {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if _, ok := room.clients[client]; ok {
		delete(room.clients, client)
		close(client.send)

		h.logger.WithFields(map[string]interface{}{
			"user_id":   client.UserID,
			"tenant_id": client.TenantID,
			"client_count": len(room.clients),
		}).Info("Client disconnected")

		// Notify other clients that a user left
		leaveMessage := &Message{
			Type:      MessageTypeUserLeft,
			UserID:    client.UserID,
			TenantID:  client.TenantID,
			Timestamp: getCurrentTimestamp(),
			MessageID: generateMessageID(),
			Data: map[string]interface{}{
				"user_id": client.UserID,
				"user_name": client.UserName,
			},
		}
		
		room.broadcast <- leaveMessage

		// If no clients left, clean up the room
		if len(room.clients) == 0 {
			delete(h.tenants, client.TenantID)
		}
	}
}

// broadcastMessage broadcasts a message to all clients in the tenant
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.tenants[message.TenantID]
	if !exists {
		h.logger.WithField("tenant_id", message.TenantID).
			Warn("Attempted to broadcast to non-existent tenant room")
		return
	}

	// Send to tenant room's broadcast channel
	select {
	case room.broadcast <- message:
	default:
		// Channel is full, log warning
		h.logger.WithField("tenant_id", message.TenantID).
			Warn("Tenant room broadcast channel is full")
	}
}

// BroadcastToTenant sends a message to all clients in a tenant
func (h *Hub) BroadcastToTenant(tenantID uuid.UUID, messageType MessageType, data interface{}) {
	message := &Message{
		Type:      messageType,
		TenantID:  tenantID,
		Timestamp: getCurrentTimestamp(),
		MessageID: generateMessageID(),
		Data:      data,
	}

	select {
	case h.broadcast <- message:
	default:
		h.logger.WithField("tenant_id", tenantID).
			Warn("Hub broadcast channel is full")
	}
}

// BroadcastToUser sends a message to a specific user (if online)
func (h *Hub) BroadcastToUser(tenantID, userID uuid.UUID, messageType MessageType, data interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.tenants[tenantID]
	if !exists {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	message := &Message{
		Type:      messageType,
		UserID:    userID,
		TenantID:  tenantID,
		Timestamp: getCurrentTimestamp(),
		MessageID: generateMessageID(),
		Data:      data,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal message")
		return
	}

	// Send to specific user's clients
	for client := range room.clients {
		if client.UserID == userID {
			select {
			case client.send <- messageBytes:
			default:
				close(client.send)
				delete(room.clients, client)
			}
		}
	}
}

// GetOnlineUsers returns a list of online users in a tenant
func (h *Hub) GetOnlineUsers(tenantID uuid.UUID) []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.tenants[tenantID]
	if !exists {
		return []uuid.UUID{}
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	userMap := make(map[uuid.UUID]bool)
	for client := range room.clients {
		userMap[client.UserID] = true
	}

	users := make([]uuid.UUID, 0, len(userMap))
	for userID := range userMap {
		users = append(users, userID)
	}

	return users
}

// GetClientCount returns the number of connected clients for a tenant
func (h *Hub) GetClientCount(tenantID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.tenants[tenantID]
	if !exists {
		return 0
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	return len(room.clients)
}

// run handles broadcasting for a tenant room
func (tr *TenantRoom) run() {
	for message := range tr.broadcast {
		tr.mu.RLock()
		
		messageBytes, err := json.Marshal(message)
		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			tr.mu.RUnlock()
			continue
		}

		// Send message to all clients in this tenant room
		for client := range tr.clients {
			select {
			case client.send <- messageBytes:
			default:
				close(client.send)
				delete(tr.clients, client)
			}
		}
		
		tr.mu.RUnlock()
	}
}

// Helper functions

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

func generateMessageID() string {
	return uuid.New().String()
}