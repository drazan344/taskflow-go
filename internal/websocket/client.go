package websocket

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/pkg/logger"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Upgrader upgrades HTTP connections to WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// Client represents a WebSocket client
type Client struct {
	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Hub reference
	hub *Hub

	// User information
	UserID   uuid.UUID `json:"user_id"`
	UserName string    `json:"user_name"`
	TenantID uuid.UUID `json:"tenant_id"`
	Role     string    `json:"role"`

	// Connection metadata
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	ConnectedAt time.Time `json:"connected_at"`

	// Logger
	logger *logger.Logger
}

// ClientInfo represents client information sent to other clients
type ClientInfo struct {
	UserID      uuid.UUID `json:"user_id"`
	UserName    string    `json:"user_name"`
	ConnectedAt time.Time `json:"connected_at"`
}

// HandleWebSocket handles WebSocket upgrade and client management
func HandleWebSocket(hub *Hub, logger *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user information from auth middleware
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context required"})
			return
		}

		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User context required"})
			return
		}

		// Type assertions
		uid, ok := userID.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
			return
		}

		tid, ok := tenantID.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant ID"})
			return
		}

		// Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.WithError(err).Error("Failed to upgrade connection to WebSocket")
			return
		}

		// Create client
		client := &Client{
			conn:        conn,
			send:        make(chan []byte, 256),
			hub:         hub,
			UserID:      uid,
			UserName:    getUsername(user), // Helper function to extract username
			TenantID:    tid,
			Role:        getRole(user),      // Helper function to extract role
			IPAddress:   c.ClientIP(),
			UserAgent:   c.Request.UserAgent(),
			ConnectedAt: time.Now(),
			logger:      logger,
		}

		// Register client with hub
		client.hub.register <- client

		// Start goroutines for reading and writing
		go client.writePump()
		go client.readPump()
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.WithError(err).Error("WebSocket error")
			}
			break
		}

		messageBytes = bytes.TrimSpace(bytes.Replace(messageBytes, newline, space, -1))
		
		// Parse incoming message
		var incomingMessage Message
		if err := json.Unmarshal(messageBytes, &incomingMessage); err != nil {
			c.logger.WithError(err).Error("Failed to parse WebSocket message")
			c.sendError("Invalid message format")
			continue
		}

		// Set client information
		incomingMessage.UserID = c.UserID
		incomingMessage.TenantID = c.TenantID
		incomingMessage.Timestamp = time.Now().Unix()
		incomingMessage.MessageID = uuid.New().String()

		// Handle different message types
		c.handleMessage(&incomingMessage)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming WebSocket messages
func (c *Client) handleMessage(message *Message) {
	switch message.Type {
	case MessageTypePing:
		c.handlePing()
	case MessageTypeTyping:
		c.handleTyping(message)
	default:
		c.logger.WithField("message_type", message.Type).
			Debug("Received WebSocket message")
		
		// Broadcast the message to the tenant room
		c.hub.broadcast <- message
	}
}

// handlePing responds to ping messages
func (c *Client) handlePing() {
	pongMessage := &Message{
		Type:      MessageTypePong,
		UserID:    c.UserID,
		TenantID:  c.TenantID,
		Timestamp: time.Now().Unix(),
		MessageID: uuid.New().String(),
	}

	messageBytes, err := json.Marshal(pongMessage)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal pong message")
		return
	}

	select {
	case c.send <- messageBytes:
	default:
		close(c.send)
	}
}

// handleTyping handles typing indicator messages
func (c *Client) handleTyping(message *Message) {
	// Broadcast typing indicator to other users in the tenant
	// (but not back to the sender)
	message.UserID = c.UserID
	message.TenantID = c.TenantID
	message.Timestamp = time.Now().Unix()
	
	c.hub.broadcast <- message
}

// sendError sends an error message to the client
func (c *Client) sendError(errorMsg string) {
	errorMessage := &Message{
		Type:      MessageTypeError,
		UserID:    c.UserID,
		TenantID:  c.TenantID,
		Timestamp: time.Now().Unix(),
		MessageID: uuid.New().String(),
		Data: map[string]interface{}{
			"error": errorMsg,
		},
	}

	messageBytes, err := json.Marshal(errorMessage)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal error message")
		return
	}

	select {
	case c.send <- messageBytes:
	default:
		close(c.send)
	}
}

// GetInfo returns client information
func (c *Client) GetInfo() ClientInfo {
	return ClientInfo{
		UserID:      c.UserID,
		UserName:    c.UserName,
		ConnectedAt: c.ConnectedAt,
	}
}

// Helper functions

func getUsername(user interface{}) string {
	if u, ok := user.(*models.User); ok {
		return u.GetFullName()
	}
	return "Unknown User"
}

func getRole(user interface{}) string {
	if u, ok := user.(*models.User); ok {
		return string(u.Role)
	}
	return "user"
}