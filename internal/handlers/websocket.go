package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/drazan344/taskflow-go/internal/middleware"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/internal/websocket"
	"github.com/drazan344/taskflow-go/pkg/logger"
)

// WebSocketHandler handles WebSocket-related operations
type WebSocketHandler struct {
	hub    *websocket.Hub
	logger *logger.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *websocket.Hub, logger *logger.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
	}
}

// HandleConnection handles WebSocket connection upgrades
// @Summary WebSocket connection
// @Description Establish WebSocket connection for real-time updates
// @Tags websocket
// @Security BearerAuth
// @Router /ws [get]
func (h *WebSocketHandler) HandleConnection() gin.HandlerFunc {
	return websocket.HandleWebSocket(h.hub, h.logger)
}

// GetOnlineUsers returns online users in the current tenant
// @Summary Get online users
// @Description Get list of users currently online in the tenant
// @Tags websocket
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /ws/online-users [get]
func (h *WebSocketHandler) GetOnlineUsers(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	onlineUsers := h.hub.GetOnlineUsers(tenantID)
	clientCount := h.hub.GetClientCount(tenantID)

	c.JSON(http.StatusOK, middleware.SuccessResponse(gin.H{
		"online_users":  onlineUsers,
		"client_count":  clientCount,
		"user_count":    len(onlineUsers),
	}))
}

// BroadcastTaskUpdate broadcasts a task update to all connected clients
func (h *WebSocketHandler) BroadcastTaskUpdate(task *models.Task, action string) {
	data := map[string]interface{}{
		"action": action, // "created", "updated", "deleted"
		"task":   task,
	}

	h.hub.BroadcastToTenant(task.TenantID, websocket.MessageTypeTaskUpdate, data)
	
	h.logger.WithFields(map[string]interface{}{
		"task_id":   task.ID,
		"tenant_id": task.TenantID,
		"action":    action,
	}).Debug("Broadcasted task update")
}

// BroadcastNotification broadcasts a notification to a specific user
func (h *WebSocketHandler) BroadcastNotification(notification *models.Notification) {
	data := map[string]interface{}{
		"notification": notification,
	}

	h.hub.BroadcastToUser(
		notification.TenantID, 
		notification.UserID, 
		websocket.MessageTypeNotification, 
		data,
	)
	
	h.logger.WithFields(map[string]interface{}{
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
		"tenant_id":       notification.TenantID,
		"type":           notification.Type,
	}).Debug("Broadcasted notification")
}

// BroadcastToTenant broadcasts a message to all users in a tenant
func (h *WebSocketHandler) BroadcastToTenant(tenantID uuid.UUID, messageType websocket.MessageType, data interface{}) {
	h.hub.BroadcastToTenant(tenantID, messageType, data)
}

// BroadcastToUser broadcasts a message to a specific user
func (h *WebSocketHandler) BroadcastToUser(tenantID, userID uuid.UUID, messageType websocket.MessageType, data interface{}) {
	h.hub.BroadcastToUser(tenantID, userID, messageType, data)
}

// GetHub returns the WebSocket hub (for use in other services)
func (h *WebSocketHandler) GetHub() *websocket.Hub {
	return h.hub
}