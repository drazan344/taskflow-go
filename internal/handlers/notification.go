package handlers

import (
	"time"

	"github.com/drazan344/taskflow-go/internal/middleware"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/internal/requests"
	"github.com/drazan344/taskflow-go/pkg/errors"
	"github.com/drazan344/taskflow-go/pkg/logger"
	"github.com/drazan344/taskflow-go/pkg/response"
	"github.com/drazan344/taskflow-go/pkg/validator"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationHandler handles notification-related HTTP requests
type NotificationHandler struct {
	db        *gorm.DB
	logger    *logger.Logger
	validator *validator.Validator
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(db *gorm.DB, logger *logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		db:        db,
		logger:    logger,
		validator: validator.New(),
	}
}

// ListNotifications returns a paginated list of notifications for the current user
// @Summary List notifications
// @Description Get a paginated list of notifications for the current user
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param status query string false "Filter by read/unread status"
// @Param type query string false "Filter by notification type"
// @Success 200 {object} response.PaginationResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications [get]
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Parse pagination parameters
	var pagination requests.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.BadRequest(c, "Invalid pagination parameters", err.Error())
		return
	}
	pagination.DefaultPagination()

	// Validate pagination
	if validationErrors := h.validator.ValidateStruct(&pagination); validationErrors != nil {
		response.ValidationErrors(c, validationErrors)
		return
	}

	// Build query
	query := h.db.Where("user_id = ?", userID)

	// Apply filters
	if status := c.Query("status"); status != "" {
		switch status {
		case "read":
			query = query.Where("read_at IS NOT NULL")
		case "unread":
			query = query.Where("read_at IS NULL")
		}
	}

	if notificationType := c.Query("type"); notificationType != "" {
		query = query.Where("type = ?", notificationType)
	}

	var notifications []models.Notification
	var total int64

	// Get total count
	if err := query.Model(&models.Notification{}).Count(&total).Error; err != nil {
		h.logger.WithError(err).Error("Failed to count notifications")
		response.InternalServerError(c, "Failed to fetch notifications")
		return
	}

	// Get notifications with pagination
	if err := query.
		Offset(pagination.GetOffset()).
		Limit(pagination.PerPage).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		h.logger.WithError(err).Error("Failed to fetch notifications")
		response.InternalServerError(c, "Failed to fetch notifications")
		return
	}

	response.Paginated(c, notifications, pagination.Page, pagination.PerPage, total)
}

// GetNotification returns a specific notification
// @Summary Get notification
// @Description Get a specific notification by ID
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Param id path string true "Notification ID"
// @Success 200 {object} models.Notification
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/{id} [get]
func (h *NotificationHandler) GetNotification(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID")
		return
	}

	var notification models.Notification
	if err := h.db.Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error; err != nil {
		if appErr := errors.HandleDBError(err, "notification"); appErr != nil {
			response.NotFound(c, appErr.Message)
			return
		}
		h.logger.WithError(err).Error("Failed to fetch notification")
		response.InternalServerError(c, "Failed to fetch notification")
		return
	}

	response.Success(c, notification)
}

// MarkAsRead marks a notification as read
// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Param id path string true "Notification ID"
// @Success 200 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/{id}/read [put]
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID")
		return
	}

	var notification models.Notification
	if err := h.db.Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error; err != nil {
		if appErr := errors.HandleDBError(err, "notification"); appErr != nil {
			response.NotFound(c, appErr.Message)
			return
		}
		h.logger.WithError(err).Error("Failed to fetch notification")
		response.InternalServerError(c, "Failed to fetch notification")
		return
	}

	// Mark as read if not already read
	if notification.ReadAt == nil {
		now := time.Now()
		if err := h.db.Model(&notification).Update("read_at", now).Error; err != nil {
			h.logger.WithError(err).Error("Failed to mark notification as read")
			response.InternalServerError(c, "Failed to mark notification as read")
			return
		}
		notification.ReadAt = &now
	}

	response.Success(c, notification, "Notification marked as read")
}

// MarkAsUnread marks a notification as unread
// @Summary Mark notification as unread
// @Description Mark a specific notification as unread
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Param id path string true "Notification ID"
// @Success 200 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/{id}/unread [put]
func (h *NotificationHandler) MarkAsUnread(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID")
		return
	}

	var notification models.Notification
	if err := h.db.Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error; err != nil {
		if appErr := errors.HandleDBError(err, "notification"); appErr != nil {
			response.NotFound(c, appErr.Message)
			return
		}
		h.logger.WithError(err).Error("Failed to fetch notification")
		response.InternalServerError(c, "Failed to fetch notification")
		return
	}

	// Mark as unread if currently read
	if notification.ReadAt != nil {
		if err := h.db.Model(&notification).Update("read_at", nil).Error; err != nil {
			h.logger.WithError(err).Error("Failed to mark notification as unread")
			response.InternalServerError(c, "Failed to mark notification as unread")
			return
		}
		notification.ReadAt = nil
	}

	response.Success(c, notification, "Notification marked as unread")
}

// MarkAllAsRead marks all notifications as read for the current user
// @Summary Mark all notifications as read
// @Description Mark all notifications as read for the current user
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/mark-all-read [put]
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	now := time.Now()
	result := h.db.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now)

	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to mark all notifications as read")
		response.InternalServerError(c, "Failed to mark notifications as read")
		return
	}

	response.Success(c, gin.H{
		"updated_count": result.RowsAffected,
	}, "All notifications marked as read")
}

// DeleteNotification deletes a notification
// @Summary Delete notification
// @Description Delete a specific notification
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Param id path string true "Notification ID"
// @Success 200 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/{id} [delete]
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID")
		return
	}

	var notification models.Notification
	if err := h.db.Where("id = ? AND user_id = ?", notificationID, userID).First(&notification).Error; err != nil {
		if appErr := errors.HandleDBError(err, "notification"); appErr != nil {
			response.NotFound(c, appErr.Message)
			return
		}
		h.logger.WithError(err).Error("Failed to fetch notification")
		response.InternalServerError(c, "Failed to fetch notification")
		return
	}

	if err := h.db.Delete(&notification).Error; err != nil {
		h.logger.WithError(err).Error("Failed to delete notification")
		response.InternalServerError(c, "Failed to delete notification")
		return
	}

	h.logger.WithField("notification_id", notificationID).Info("Notification deleted successfully")
	response.Success(c, nil, "Notification deleted successfully")
}

// GetUnreadCount returns the count of unread notifications
// @Summary Get unread notification count
// @Description Get the count of unread notifications for the current user
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/unread-count [get]
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var count int64
	if err := h.db.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&count).Error; err != nil {
		h.logger.WithError(err).Error("Failed to count unread notifications")
		response.InternalServerError(c, "Failed to get unread count")
		return
	}

	response.Success(c, gin.H{
		"unread_count": count,
	})
}

// GetNotificationSettings returns the notification settings for the current user
// @Summary Get notification settings
// @Description Get notification preferences for the current user
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/settings [get]
func (h *NotificationHandler) GetNotificationSettings(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		h.logger.WithError(err).Error("Failed to fetch user")
		response.InternalServerError(c, "Failed to fetch user")
		return
	}

	// Create preferences response object
	preferences := gin.H{
		"theme":                      user.Theme,
		"enable_email_notifications": user.EnableEmailNotifications,
		"enable_push_notifications":  user.EnablePushNotifications,
		"task_reminders":            user.TaskReminders,
		"weekly_digest":             user.WeeklyDigest,
		"default_task_priority":     user.DefaultTaskPriority,
		"task_view_mode":            user.TaskViewMode,
		"show_completed_tasks":      user.ShowCompletedTasks,
		"tasks_per_page":           user.TasksPerPage,
	}
	response.Success(c, preferences)
}

// UpdateNotificationSettings updates notification settings for the current user
// @Summary Update notification settings
// @Description Update notification preferences for the current user
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]interface{} true "Notification preferences"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /notifications/settings [put]
func (h *NotificationHandler) UpdateNotificationSettings(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		h.logger.WithError(err).Error("Failed to fetch user")
		response.InternalServerError(c, "Failed to fetch user")
		return
	}

	var preferences map[string]interface{}
	if err := c.ShouldBindJSON(&preferences); err != nil {
		response.BadRequest(c, "Invalid request data", err.Error())
		return
	}

	// Update individual preference fields
	updates := make(map[string]interface{})
	
	if theme, ok := preferences["theme"]; ok {
		updates["theme"] = theme
	}
	if emailNotifications, ok := preferences["enable_email_notifications"]; ok {
		updates["enable_email_notifications"] = emailNotifications
	}
	if pushNotifications, ok := preferences["enable_push_notifications"]; ok {
		updates["enable_push_notifications"] = pushNotifications
	}
	if taskReminders, ok := preferences["task_reminders"]; ok {
		updates["task_reminders"] = taskReminders
	}
	if weeklyDigest, ok := preferences["weekly_digest"]; ok {
		updates["weekly_digest"] = weeklyDigest
	}
	if defaultTaskPriority, ok := preferences["default_task_priority"]; ok {
		updates["default_task_priority"] = defaultTaskPriority
	}
	if taskViewMode, ok := preferences["task_view_mode"]; ok {
		updates["task_view_mode"] = taskViewMode
	}
	if showCompletedTasks, ok := preferences["show_completed_tasks"]; ok {
		updates["show_completed_tasks"] = showCompletedTasks
	}
	if tasksPerPage, ok := preferences["tasks_per_page"]; ok {
		updates["tasks_per_page"] = tasksPerPage
	}

	// Update user preferences
	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update notification settings")
		response.InternalServerError(c, "Failed to update notification settings")
		return
	}

	response.Success(c, preferences, "Notification settings updated successfully")
}
