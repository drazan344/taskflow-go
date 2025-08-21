package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/drazan344/taskflow-go/internal/middleware"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/pkg/logger"
	"gorm.io/gorm"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *gorm.DB, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		db:     db,
		logger: logger,
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
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	var users []models.User
	var total int64

	// Get total count
	if err := h.db.Model(&models.User{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		h.logger.WithError(err).Error("Failed to count users")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch users"))
		return
	}

	// Get users
	if err := h.db.Where("tenant_id = ?", tenantID).
		Offset(offset).
		Limit(perPage).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		h.logger.WithError(err).Error("Failed to fetch users")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch users"))
		return
	}

	c.JSON(http.StatusOK, middleware.PaginationResponse(users, page, perPage, int(total)))
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

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request data"))
		return
	}

	// Update allowed fields
	if err := h.db.Model(&user).Updates(updateData).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update user")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to update user"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(user, "User updated successfully"))
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