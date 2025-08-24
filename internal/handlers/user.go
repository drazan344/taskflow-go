package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/drazan344/taskflow-go/internal/middleware"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/internal/requests"
	"github.com/drazan344/taskflow-go/pkg/errors"
	"github.com/drazan344/taskflow-go/pkg/logger"
	"github.com/drazan344/taskflow-go/pkg/response"
	"github.com/drazan344/taskflow-go/pkg/validator"
	"gorm.io/gorm"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	db        *gorm.DB
	logger    *logger.Logger
	validator *validator.Validator
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *gorm.DB, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		db:        db,
		logger:    logger,
		validator: validator.New(),
	}
}

// ListUsers returns a paginated list of users in the tenant
// @Summary List users
// @Description Get a paginated list of users in the current tenant
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		response.Unauthorized(c, "Tenant not found")
		return
	}

	// Parse pagination parameters
	var pagination requests.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		response.BadRequest(c, "Invalid pagination parameters", err.Error())
		return
	}
	pagination.DefaultPagination()

	// Parse filters
	var filters requests.UserFiltersRequest
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.BadRequest(c, "Invalid filter parameters", err.Error())
		return
	}

	// Validate pagination and filters
	if validationErrors := h.validator.ValidateStruct(&pagination); validationErrors != nil {
		response.ValidationErrors(c, validationErrors)
		return
	}
	if validationErrors := h.validator.ValidateStruct(&filters); validationErrors != nil {
		response.ValidationErrors(c, validationErrors)
		return
	}

	// Build query
	query := h.db.Where("tenant_id = ?", tenantID)

	// Apply filters
	if filters.Role != nil {
		query = query.Where("role = ?", *filters.Role)
	}
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.Search != "" {
		searchTerm := "%" + strings.ToLower(filters.Search) + "%"
		query = query.Where("LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ?", 
			searchTerm, searchTerm, searchTerm)
	}

	var users []models.User
	var total int64

	// Get total count
	if err := query.Model(&models.User{}).Count(&total).Error; err != nil {
		h.logger.WithError(err).Error("Failed to count users")
		response.InternalServerError(c, "Failed to fetch users")
		return
	}

	// Get users
	if err := query.
		Offset(pagination.GetOffset()).
		Limit(pagination.PerPage).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		h.logger.WithError(err).Error("Failed to fetch users")
		response.InternalServerError(c, "Failed to fetch users")
		return
	}

	response.Paginated(c, users, pagination.Page, pagination.PerPage, total)
}

// GetUser returns a specific user by ID
// @Summary Get user
// @Description Get a specific user by ID within the current tenant
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid user ID"))
		return
	}

	var user models.User
	if err := h.db.Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("User not found"))
			return
		}
		h.logger.WithError(err).Error("Failed to fetch user")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch user"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(user))
}

// UpdateUser updates a user (admin/manager only)
// @Summary Update user
// @Description Update a user's information (requires admin or manager role)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body map[string]interface{} true "User update data"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		response.Unauthorized(c, "Tenant not found")
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var user models.User
	if err := h.db.Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err != nil {
		if appErr := errors.HandleDBError(err, "user"); appErr != nil {
			response.NotFound(c, appErr.Message)
			return
		}
		h.logger.WithError(err).Error("Failed to fetch user")
		response.InternalServerError(c, "Failed to fetch user")
		return
	}

	var req requests.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request data", err.Error())
		return
	}

	// Validate request
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		response.ValidationErrors(c, validationErrors)
		return
	}

	// Check if email is being updated and ensure it's unique
	if req.Email != nil && *req.Email != user.Email {
		var existingUser models.User
		if err := h.db.Where("email = ? AND tenant_id = ? AND id != ?", *req.Email, tenantID, userID).First(&existingUser).Error; err == nil {
			response.Conflict(c, "Email address is already in use")
			return
		} else if err != gorm.ErrRecordNotFound {
			h.logger.WithError(err).Error("Failed to check email uniqueness")
			response.InternalServerError(c, "Failed to validate email")
			return
		}
	}

	// Build update map
	updateData := make(map[string]interface{})
	if req.FirstName != nil {
		updateData["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updateData["last_name"] = *req.LastName
	}
	if req.Email != nil {
		updateData["email"] = *req.Email
	}
	if req.Phone != nil {
		updateData["phone"] = *req.Phone
	}
	if req.Avatar != nil {
		updateData["avatar"] = *req.Avatar
	}
	if req.Timezone != nil {
		updateData["timezone"] = *req.Timezone
	}
	if req.Language != nil {
		updateData["language"] = *req.Language
	}
	if req.Role != nil {
		updateData["role"] = *req.Role
	}
	if req.Status != nil {
		updateData["status"] = *req.Status
	}

	// Update user
	if len(updateData) > 0 {
		if err := h.db.Model(&user).Updates(updateData).Error; err != nil {
			h.logger.WithError(err).Error("Failed to update user")
			response.InternalServerError(c, "Failed to update user")
			return
		}
	}

	// Reload user to get updated data
	if err := h.db.First(&user, userID).Error; err != nil {
		h.logger.WithError(err).Warn("Failed to reload user")
	}

	h.logger.WithField("user_id", userID).Info("User updated successfully")
	response.Success(c, user, "User updated successfully")
}

// DeleteUser soft deletes a user (admin only)
// @Summary Delete user
// @Description Soft delete a user (requires admin role)
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid user ID"))
		return
	}

	var user models.User
	if err := h.db.Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("User not found"))
			return
		}
		h.logger.WithError(err).Error("Failed to fetch user")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch user"))
		return
	}

	if err := h.db.Delete(&user).Error; err != nil {
		h.logger.WithError(err).Error("Failed to delete user")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to delete user"))
		return
	}

	h.logger.WithField("user_id", userID).Info("User deleted successfully")

	c.JSON(http.StatusOK, middleware.SuccessResponse(nil, "User deleted successfully"))
}

// UpdateUserPreferences updates user preferences for the current user
// @Summary Update user preferences
// @Description Update preferences for the current user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body requests.UpdateUserPreferencesRequest true "User preferences"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /users/preferences [put]
func (h *UserHandler) UpdateUserPreferences(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		if appErr := errors.HandleDBError(err, "user"); appErr != nil {
			response.NotFound(c, appErr.Message)
			return
		}
		h.logger.WithError(err).Error("Failed to fetch user")
		response.InternalServerError(c, "Failed to fetch user")
		return
	}

	var req requests.UpdateUserPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request data", err.Error())
		return
	}

	// Validate request
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		response.ValidationErrors(c, validationErrors)
		return
	}

	// Update preferences
	preferences := user.Preferences
	if req.Theme != nil {
		preferences.Theme = *req.Theme
	}
	if req.EmailNotifications != nil {
		preferences.EmailNotifications = *req.EmailNotifications
	}
	if req.PushNotifications != nil {
		preferences.PushNotifications = *req.PushNotifications
	}
	if req.TaskReminders != nil {
		preferences.TaskReminders = *req.TaskReminders
	}
	if req.WeeklyDigest != nil {
		preferences.WeeklyDigest = *req.WeeklyDigest
	}
	if req.DefaultTaskPriority != nil {
		preferences.DefaultTaskPriority = *req.DefaultTaskPriority
	}
	if req.TaskViewMode != nil {
		preferences.TaskViewMode = *req.TaskViewMode
	}
	if req.ShowCompletedTasks != nil {
		preferences.ShowCompletedTasks = *req.ShowCompletedTasks
	}
	if req.TasksPerPage != nil {
		preferences.TasksPerPage = *req.TasksPerPage
	}

	if err := h.db.Model(&user).Update("preferences", preferences).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update user preferences")
		response.InternalServerError(c, "Failed to update preferences")
		return
	}

	response.Success(c, preferences, "Preferences updated successfully")
}

// ChangePassword allows a user to change their password
// @Summary Change password
// @Description Change password for the current user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body requests.ChangePasswordRequest true "Password change data"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /users/change-password [post]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		if appErr := errors.HandleDBError(err, "user"); appErr != nil {
			response.NotFound(c, appErr.Message)
			return
		}
		h.logger.WithError(err).Error("Failed to fetch user")
		response.InternalServerError(c, "Failed to fetch user")
		return
	}

	var req requests.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request data", err.Error())
		return
	}

	// Validate request
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		response.ValidationErrors(c, validationErrors)
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		response.BadRequest(c, "Current password is incorrect")
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.logger.WithError(err).Error("Failed to hash password")
		response.InternalServerError(c, "Failed to process password")
		return
	}

	// Update password
	if err := h.db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update password")
		response.InternalServerError(c, "Failed to update password")
		return
	}

	h.logger.WithField("user_id", userID).Info("Password changed successfully")
	response.Success(c, nil, "Password changed successfully")
}

// GetUserStats returns user statistics
// @Summary Get user statistics
// @Description Get statistics for a specific user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 404 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /users/{id}/stats [get]
func (h *UserHandler) GetUserStats(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		response.Unauthorized(c, "Tenant not found")
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	// Verify user exists in tenant
	var user models.User
	if err := h.db.Where("id = ? AND tenant_id = ?", userID, tenantID).First(&user).Error; err != nil {
		if appErr := errors.HandleDBError(err, "user"); appErr != nil {
			response.NotFound(c, appErr.Message)
			return
		}
		h.logger.WithError(err).Error("Failed to fetch user")
		response.InternalServerError(c, "Failed to fetch user")
		return
	}

	// Get task statistics
	var taskStats struct {
		TotalAssigned int64 `json:"total_assigned"`
		TotalCreated  int64 `json:"total_created"`
		Completed     int64 `json:"completed"`
		InProgress    int64 `json:"in_progress"`
		Overdue       int64 `json:"overdue"`
	}

	// Count assigned tasks
	h.db.Model(&models.Task{}).Where("assignee_id = ? AND tenant_id = ?", userID, tenantID).Count(&taskStats.TotalAssigned)
	
	// Count created tasks
	h.db.Model(&models.Task{}).Where("creator_id = ? AND tenant_id = ?", userID, tenantID).Count(&taskStats.TotalCreated)
	
	// Count completed tasks
	h.db.Model(&models.Task{}).Where("assignee_id = ? AND tenant_id = ? AND status = ?", userID, tenantID, models.TaskStatusCompleted).Count(&taskStats.Completed)
	
	// Count in progress tasks
	h.db.Model(&models.Task{}).Where("assignee_id = ? AND tenant_id = ? AND status = ?", userID, tenantID, models.TaskStatusInProgress).Count(&taskStats.InProgress)
	
	// Count overdue tasks (due date in the past and not completed)
	h.db.Model(&models.Task{}).Where("assignee_id = ? AND tenant_id = ? AND due_date < NOW() AND status NOT IN ?", 
		userID, tenantID, []models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusCanceled}).Count(&taskStats.Overdue)

	response.Success(c, gin.H{
		"user":  user,
		"stats": taskStats,
	})
}