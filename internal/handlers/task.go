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

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(db *gorm.DB, logger *logger.Logger) *TaskHandler {
	return &TaskHandler{
		db:     db,
		logger: logger,
	}
}

// CreateTaskRequest represents a task creation request
type CreateTaskRequest struct {
	Title          string               `json:"title" binding:"required"`
	Description    string               `json:"description"`
	Priority       models.TaskPriority  `json:"priority"`
	DueDate        *string              `json:"due_date,omitempty"`
	AssigneeID     *uuid.UUID           `json:"assignee_id,omitempty"`
	ProjectID      *uuid.UUID           `json:"project_id,omitempty"`
	ParentID       *uuid.UUID           `json:"parent_id,omitempty"`
	EstimatedHours *float64             `json:"estimated_hours,omitempty"`
}

// ListTasks returns a paginated list of tasks
// @Summary List tasks
// @Description Get a paginated list of tasks in the current tenant
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param status query string false "Filter by status"
// @Param assignee_id query string false "Filter by assignee ID"
// @Param project_id query string false "Filter by project ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /tasks [get]
func (h *TaskHandler) ListTasks(c *gin.Context) {
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

	// Build query
	query := h.db.Where("tenant_id = ?", tenantID)

	// Apply filters
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if assigneeID := c.Query("assignee_id"); assigneeID != "" {
		if id, err := uuid.Parse(assigneeID); err == nil {
			query = query.Where("assignee_id = ?", id)
		}
	}
	if projectID := c.Query("project_id"); projectID != "" {
		if id, err := uuid.Parse(projectID); err == nil {
			query = query.Where("project_id = ?", id)
		}
	}

	var tasks []models.Task
	var total int64

	// Get total count
	if err := query.Model(&models.Task{}).Count(&total).Error; err != nil {
		h.logger.WithError(err).Error("Failed to count tasks")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch tasks"))
		return
	}

	// Get tasks with preloaded relationships
	if err := query.
		Preload("Creator").
		Preload("Assignee").
		Preload("Project").
		Preload("Tags").
		Offset(offset).
		Limit(perPage).
		Order("created_at DESC").
		Find(&tasks).Error; err != nil {
		h.logger.WithError(err).Error("Failed to fetch tasks")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch tasks"))
		return
	}

	c.JSON(http.StatusOK, middleware.PaginationResponse(tasks, page, perPage, int(total)))
}

// CreateTask creates a new task
// @Summary Create task
// @Description Create a new task in the current tenant
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateTaskRequest true "Task creation data"
// @Success 201 {object} models.Task
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("User not authenticated"))
		return
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request", err.Error()))
		return
	}

	task := &models.Task{
		TenantModel:    models.TenantModel{TenantID: tenantID},
		Title:          req.Title,
		Description:    req.Description,
		Priority:       req.Priority,
		Status:         models.TaskStatusTodo,
		CreatorID:      userID,
		AssigneeID:     req.AssigneeID,
		ProjectID:      req.ProjectID,
		ParentID:       req.ParentID,
		EstimatedHours: req.EstimatedHours,
	}

	// Parse due date if provided
	if req.DueDate != nil {
		// Parse due date string - in production, you'd want better date parsing
		// For now, this is a simplified example
	}

	if err := h.db.Create(task).Error; err != nil {
		h.logger.WithError(err).Error("Failed to create task")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to create task"))
		return
	}

	// Reload task with relationships
	if err := h.db.
		Preload("Creator").
		Preload("Assignee").
		Preload("Project").
		First(task, task.ID).Error; err != nil {
		h.logger.WithError(err).Warn("Failed to reload task with relationships")
	}

	h.logger.WithField("task_id", task.ID).Info("Task created successfully")

	c.JSON(http.StatusCreated, middleware.SuccessResponse(task, "Task created successfully"))
}

// GetTask returns a specific task by ID
// @Summary Get task
// @Description Get a specific task by ID within the current tenant
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Success 200 {object} models.Task
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /tasks/{id} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid task ID"))
		return
	}

	var task models.Task
	if err := h.db.
		Preload("Creator").
		Preload("Assignee").
		Preload("Project").
		Preload("Tags").
		Preload("Comments.User").
		Preload("Attachments").
		Where("id = ? AND tenant_id = ?", taskID, tenantID).
		First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Task not found"))
			return
		}
		h.logger.WithError(err).Error("Failed to fetch task")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch task"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(task))
}

// UpdateTask updates a task
// @Summary Update task
// @Description Update a task's information
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Param request body map[string]interface{} true "Task update data"
// @Success 200 {object} models.Task
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /tasks/{id} [put]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid task ID"))
		return
	}

	var task models.Task
	if err := h.db.Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Task not found"))
			return
		}
		h.logger.WithError(err).Error("Failed to fetch task")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch task"))
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request data"))
		return
	}

	// Update allowed fields
	if err := h.db.Model(&task).Updates(updateData).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update task")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to update task"))
		return
	}

	// Reload task with relationships
	if err := h.db.
		Preload("Creator").
		Preload("Assignee").
		Preload("Project").
		Preload("Tags").
		First(&task, task.ID).Error; err != nil {
		h.logger.WithError(err).Warn("Failed to reload task with relationships")
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(task, "Task updated successfully"))
}

// DeleteTask soft deletes a task
// @Summary Delete task
// @Description Soft delete a task
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /tasks/{id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid task ID"))
		return
	}

	var task models.Task
	if err := h.db.Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Task not found"))
			return
		}
		h.logger.WithError(err).Error("Failed to fetch task")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch task"))
		return
	}

	if err := h.db.Delete(&task).Error; err != nil {
		h.logger.WithError(err).Error("Failed to delete task")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to delete task"))
		return
	}

	h.logger.WithField("task_id", taskID).Info("Task deleted successfully")

	c.JSON(http.StatusOK, middleware.SuccessResponse(nil, "Task deleted successfully"))
}

// Placeholder methods for additional task functionality
func (h *TaskHandler) AddComment(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) ListComments(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) AddAttachment(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) ListAttachments(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) DeleteAttachment(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

// Project-related methods
func (h *TaskHandler) ListProjects(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) CreateProject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) GetProject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) UpdateProject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) DeleteProject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

// Tag-related methods
func (h *TaskHandler) ListTags(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) CreateTag(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) GetTag(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) UpdateTag(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}

func (h *TaskHandler) DeleteTag(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, middleware.ErrorResponse("Not implemented yet"))
}