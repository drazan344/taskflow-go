package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/drazan344/taskflow-go/internal/middleware"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/internal/requests"
	"github.com/drazan344/taskflow-go/pkg/errors"
	"github.com/drazan344/taskflow-go/pkg/logger"
	"github.com/drazan344/taskflow-go/pkg/response"
	"github.com/drazan344/taskflow-go/pkg/validator"
	"gorm.io/gorm"
)

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	db        *gorm.DB
	logger    *logger.Logger
	validator *validator.Validator
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(db *gorm.DB, logger *logger.Logger) *TaskHandler {
	return &TaskHandler{
		db:        db,
		logger:    logger,
		validator: validator.New(),
	}
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
// @Param request body requests.CreateTaskRequest true "Task creation data"
// @Success 201 {object} models.Task
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		response.Unauthorized(c, "Tenant not found")
		return
	}

	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req requests.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request data", err.Error())
		return
	}

	// Validate request
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		response.ValidationErrors(c, validationErrors)
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
		DueDate:        req.DueDate,
	}

	if err := h.db.Create(task).Error; err != nil {
		if appErr := errors.HandleDBError(err, "task"); appErr != nil {
			response.InternalServerError(c, appErr.Message)
			return
		}
		h.logger.WithError(err).Error("Failed to create task")
		response.InternalServerError(c, "Failed to create task")
		return
	}

	// Handle tags if provided
	if len(req.Tags) > 0 {
		var tags []models.Tag
		if err := h.db.Where("id IN ? AND tenant_id = ?", req.Tags, tenantID).Find(&tags).Error; err != nil {
			h.logger.WithError(err).Warn("Failed to find tags")
		} else if len(tags) > 0 {
			if err := h.db.Model(task).Association("Tags").Append(tags); err != nil {
				h.logger.WithError(err).Warn("Failed to associate tags")
			}
		}
	}

	// Reload task with relationships
	if err := h.db.
		Preload("Creator").
		Preload("Assignee").
		Preload("Project").
		Preload("Tags").
		First(task, task.ID).Error; err != nil {
		h.logger.WithError(err).Warn("Failed to reload task with relationships")
	}

	h.logger.WithField("task_id", task.ID).Info("Task created successfully")
	response.Created(c, task, "Task created successfully")
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

// CreateCommentRequest represents a comment creation request
type CreateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

// Comment-related methods
func (h *TaskHandler) AddComment(c *gin.Context) {
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

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid task ID"))
		return
	}

	// Verify task exists and belongs to tenant
	var task models.Task
	if err := h.db.Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Task not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to verify task"))
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request", err.Error()))
		return
	}

	comment := &models.TaskComment{
		TaskID:  taskID,
		UserID:  userID,
		Content: req.Content,
	}

	if err := h.db.Create(comment).Error; err != nil {
		h.logger.WithError(err).Error("Failed to create comment")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to create comment"))
		return
	}

	// Reload comment with user information
	if err := h.db.Preload("User").First(comment, comment.ID).Error; err != nil {
		h.logger.WithError(err).Warn("Failed to reload comment with user")
	}

	c.JSON(http.StatusCreated, middleware.SuccessResponse(comment, "Comment added successfully"))
}

func (h *TaskHandler) ListComments(c *gin.Context) {
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

	// Verify task exists and belongs to tenant
	var task models.Task
	if err := h.db.Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Task not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to verify task"))
		return
	}

	var comments []models.TaskComment
	if err := h.db.
		Preload("User").
		Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		h.logger.WithError(err).Error("Failed to fetch comments")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch comments"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(comments))
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
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	var projects []models.Project
	if err := h.db.Where("tenant_id = ?", tenantID).Find(&projects).Error; err != nil {
		h.logger.WithError(err).Error("Failed to fetch projects")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch projects"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(projects))
}

func (h *TaskHandler) CreateProject(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		response.Unauthorized(c, "Tenant not found")
		return
	}

	var req requests.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request data", err.Error())
		return
	}

	// Validate request
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		response.ValidationErrors(c, validationErrors)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	project := &models.Project{
		TenantModel: models.TenantModel{TenantID: tenantID},
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		IsActive:    isActive,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	}

	if err := h.db.Create(project).Error; err != nil {
		if appErr := errors.HandleDBError(err, "project"); appErr != nil {
			response.InternalServerError(c, appErr.Message)
			return
		}
		h.logger.WithError(err).Error("Failed to create project")
		response.InternalServerError(c, "Failed to create project")
		return
	}

	response.Created(c, project, "Project created successfully")
}

func (h *TaskHandler) GetProject(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid project ID"))
		return
	}

	var project models.Project
	if err := h.db.Where("id = ? AND tenant_id = ?", projectID, tenantID).First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Project not found"))
			return
		}
		h.logger.WithError(err).Error("Failed to fetch project")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch project"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(project))
}

func (h *TaskHandler) UpdateProject(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid project ID"))
		return
	}

	var project models.Project
	if err := h.db.Where("id = ? AND tenant_id = ?", projectID, tenantID).First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Project not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch project"))
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request data"))
		return
	}

	if err := h.db.Model(&project).Updates(updateData).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update project")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to update project"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(project, "Project updated successfully"))
}

func (h *TaskHandler) DeleteProject(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid project ID"))
		return
	}

	var project models.Project
	if err := h.db.Where("id = ? AND tenant_id = ?", projectID, tenantID).First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Project not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch project"))
		return
	}

	if err := h.db.Delete(&project).Error; err != nil {
		h.logger.WithError(err).Error("Failed to delete project")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to delete project"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(nil, "Project deleted successfully"))
}

// CreateTagRequest represents a tag creation request
type CreateTagRequest struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color"`
}

// Tag-related methods
func (h *TaskHandler) ListTags(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	var tags []models.Tag
	if err := h.db.Where("tenant_id = ?", tenantID).Find(&tags).Error; err != nil {
		h.logger.WithError(err).Error("Failed to fetch tags")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch tags"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(tags))
}

func (h *TaskHandler) CreateTag(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	var req CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request", err.Error()))
		return
	}

	tag := &models.Tag{
		TenantModel: models.TenantModel{TenantID: tenantID},
		Name:        req.Name,
		Color:       req.Color,
	}

	if err := h.db.Create(tag).Error; err != nil {
		h.logger.WithError(err).Error("Failed to create tag")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to create tag"))
		return
	}

	c.JSON(http.StatusCreated, middleware.SuccessResponse(tag, "Tag created successfully"))
}

func (h *TaskHandler) GetTag(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	tagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid tag ID"))
		return
	}

	var tag models.Tag
	if err := h.db.Where("id = ? AND tenant_id = ?", tagID, tenantID).First(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Tag not found"))
			return
		}
		h.logger.WithError(err).Error("Failed to fetch tag")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch tag"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(tag))
}

func (h *TaskHandler) UpdateTag(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	tagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid tag ID"))
		return
	}

	var tag models.Tag
	if err := h.db.Where("id = ? AND tenant_id = ?", tagID, tenantID).First(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Tag not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch tag"))
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request data"))
		return
	}

	if err := h.db.Model(&tag).Updates(updateData).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update tag")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to update tag"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(tag, "Tag updated successfully"))
}

func (h *TaskHandler) DeleteTag(c *gin.Context) {
	tenantID, err := middleware.GetCurrentTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Tenant not found"))
		return
	}

	tagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid tag ID"))
		return
	}

	var tag models.Tag
	if err := h.db.Where("id = ? AND tenant_id = ?", tagID, tenantID).First(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse("Tag not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to fetch tag"))
		return
	}

	if err := h.db.Delete(&tag).Error; err != nil {
		h.logger.WithError(err).Error("Failed to delete tag")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to delete tag"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(nil, "Tag deleted successfully"))
}