package requests

import (
	"time"

	"github.com/google/uuid"
	"github.com/drazan344/taskflow-go/internal/models"
)

// CreateTaskRequest represents a task creation request with validation
type CreateTaskRequest struct {
	Title          string               `json:"title" validate:"required,min=1,max=200"`
	Description    string               `json:"description" validate:"max=2000"`
	Priority       models.TaskPriority  `json:"priority" validate:"required,priority"`
	DueDate        *time.Time           `json:"due_date,omitempty"`
	AssigneeID     *uuid.UUID           `json:"assignee_id,omitempty" validate:"omitempty,uuid"`
	ProjectID      *uuid.UUID           `json:"project_id,omitempty" validate:"omitempty,uuid"`
	ParentID       *uuid.UUID           `json:"parent_id,omitempty" validate:"omitempty,uuid"`
	EstimatedHours *float64             `json:"estimated_hours,omitempty" validate:"omitempty,min=0,max=9999"`
	Tags           []uuid.UUID          `json:"tags,omitempty"`
}

// UpdateTaskRequest represents a task update request with validation
type UpdateTaskRequest struct {
	Title          *string              `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Description    *string              `json:"description,omitempty" validate:"omitempty,max=2000"`
	Priority       *models.TaskPriority `json:"priority,omitempty" validate:"omitempty,priority"`
	Status         *models.TaskStatus   `json:"status,omitempty" validate:"omitempty,task_status"`
	DueDate        *time.Time           `json:"due_date,omitempty"`
	AssigneeID     *uuid.UUID           `json:"assignee_id,omitempty" validate:"omitempty,uuid"`
	ProjectID      *uuid.UUID           `json:"project_id,omitempty" validate:"omitempty,uuid"`
	ParentID       *uuid.UUID           `json:"parent_id,omitempty" validate:"omitempty,uuid"`
	EstimatedHours *float64             `json:"estimated_hours,omitempty" validate:"omitempty,min=0,max=9999"`
	ActualHours    *float64             `json:"actual_hours,omitempty" validate:"omitempty,min=0,max=9999"`
	Tags           []uuid.UUID          `json:"tags,omitempty"`
}

// CreateProjectRequest represents a project creation request with validation
type CreateProjectRequest struct {
	Name        string     `json:"name" validate:"required,min=1,max=100"`
	Description string     `json:"description" validate:"max=1000"`
	Color       string     `json:"color" validate:"omitempty,hexcolor"`
	IsActive    *bool      `json:"is_active,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

// UpdateProjectRequest represents a project update request with validation
type UpdateProjectRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	Color       *string    `json:"color,omitempty" validate:"omitempty,hexcolor"`
	IsActive    *bool      `json:"is_active,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

// CreateTagRequest represents a tag creation request with validation
type CreateTagRequest struct {
	Name  string `json:"name" validate:"required,min=1,max=50"`
	Color string `json:"color" validate:"omitempty,hexcolor"`
}

// UpdateTagRequest represents a tag update request with validation
type UpdateTagRequest struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=1,max=50"`
	Color *string `json:"color,omitempty" validate:"omitempty,hexcolor"`
}

// CreateCommentRequest represents a comment creation request with validation
type CreateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=2000"`
}

// TaskFiltersRequest represents task filtering parameters
type TaskFiltersRequest struct {
	Status     *models.TaskStatus   `form:"status" validate:"omitempty,task_status"`
	Priority   *models.TaskPriority `form:"priority" validate:"omitempty,priority"`
	AssigneeID *uuid.UUID           `form:"assignee_id" validate:"omitempty,uuid"`
	ProjectID  *uuid.UUID           `form:"project_id" validate:"omitempty,uuid"`
	CreatorID  *uuid.UUID           `form:"creator_id" validate:"omitempty,uuid"`
	TagID      *uuid.UUID           `form:"tag_id" validate:"omitempty,uuid"`
	DueFrom    *time.Time           `form:"due_from"`
	DueTo      *time.Time           `form:"due_to"`
	Search     string               `form:"search" validate:"max=100"`
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page    int `form:"page" validate:"min=1"`
	PerPage int `form:"per_page" validate:"min=1,max=100"`
}

// DefaultPagination returns default pagination values
func (p *PaginationRequest) DefaultPagination() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage == 0 {
		p.PerPage = 20
	}
}

// GetOffset calculates the offset for database queries
func (p *PaginationRequest) GetOffset() int {
	return (p.Page - 1) * p.PerPage
}