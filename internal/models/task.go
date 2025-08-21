package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusInReview   TaskStatus = "in_review"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusCanceled   TaskStatus = "canceled"
)

// TaskPriority represents the priority level of a task
type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
	TaskPriorityUrgent TaskPriority = "urgent"
)

// Task represents a task in the system
type Task struct {
	TenantModel
	Title         string        `json:"title" gorm:"not null;size:255"`
	Description   string        `json:"description" gorm:"type:text"`
	Status        TaskStatus    `json:"status" gorm:"default:'todo'"`
	Priority      TaskPriority  `json:"priority" gorm:"default:'medium'"`
	DueDate       *time.Time    `json:"due_date,omitempty"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
	EstimatedHours *float64     `json:"estimated_hours,omitempty"`
	ActualHours   *float64      `json:"actual_hours,omitempty"`
	
	// Relationships
	CreatorID   uuid.UUID  `json:"creator_id" gorm:"type:uuid;not null"`
	AssigneeID  *uuid.UUID `json:"assignee_id,omitempty" gorm:"type:uuid"`
	ProjectID   *uuid.UUID `json:"project_id,omitempty" gorm:"type:uuid"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty" gorm:"type:uuid"`
	
	Creator    User       `json:"creator" gorm:"foreignKey:CreatorID"`
	Assignee   *User      `json:"assignee,omitempty" gorm:"foreignKey:AssigneeID"`
	Project    *Project   `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Parent     *Task      `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Subtasks   []Task     `json:"subtasks,omitempty" gorm:"foreignKey:ParentID"`
	Comments   []TaskComment   `json:"comments,omitempty" gorm:"foreignKey:TaskID"`
	Attachments []TaskAttachment `json:"attachments,omitempty" gorm:"foreignKey:TaskID"`
	Tags       []Tag      `json:"tags,omitempty" gorm:"many2many:task_tags;"`
	Activities []TaskActivity  `json:"activities,omitempty" gorm:"foreignKey:TaskID"`
}

// TaskComment represents a comment on a task
type TaskComment struct {
	TenantModel
	TaskID    uuid.UUID `json:"task_id" gorm:"type:uuid;not null;index"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty" gorm:"type:uuid"`
	
	// Relationships
	Task     Task          `json:"task" gorm:"foreignKey:TaskID"`
	User     User          `json:"user" gorm:"foreignKey:UserID"`
	Parent   *TaskComment  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Replies  []TaskComment `json:"replies,omitempty" gorm:"foreignKey:ParentID"`
}

// TaskAttachment represents a file attachment on a task
type TaskAttachment struct {
	TenantModel
	TaskID      uuid.UUID `json:"task_id" gorm:"type:uuid;not null;index"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	FileName    string    `json:"file_name" gorm:"not null;size:255"`
	OriginalName string   `json:"original_name" gorm:"not null;size:255"`
	FileSize    int64     `json:"file_size" gorm:"not null"`
	MimeType    string    `json:"mime_type" gorm:"not null;size:100"`
	FilePath    string    `json:"file_path" gorm:"not null;size:500"`
	
	// Relationships
	Task Task `json:"task" gorm:"foreignKey:TaskID"`
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// TaskActivity represents an activity/change log for a task
type TaskActivity struct {
	TenantModel
	TaskID      uuid.UUID    `json:"task_id" gorm:"type:uuid;not null;index"`
	UserID      uuid.UUID    `json:"user_id" gorm:"type:uuid;not null"`
	Action      string       `json:"action" gorm:"not null;size:50"`
	Field       string       `json:"field,omitempty" gorm:"size:50"`
	OldValue    string       `json:"old_value,omitempty" gorm:"type:text"`
	NewValue    string       `json:"new_value,omitempty" gorm:"type:text"`
	Description string       `json:"description,omitempty" gorm:"type:text"`
	
	// Relationships
	Task Task `json:"task" gorm:"foreignKey:TaskID"`
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// Project represents a project that can contain tasks
type Project struct {
	TenantModel
	Name        string    `json:"name" gorm:"not null;size:255"`
	Description string    `json:"description" gorm:"type:text"`
	Color       string    `json:"color" gorm:"size:7"` // Hex color code
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	
	// Relationships
	Tasks []Task `json:"tasks,omitempty" gorm:"foreignKey:ProjectID"`
}

// Tag represents a tag that can be applied to tasks
type Tag struct {
	TenantModel
	Name  string `json:"name" gorm:"not null;size:100"`
	Color string `json:"color" gorm:"size:7"` // Hex color code
	
	// Relationships
	Tasks []Task `json:"tasks,omitempty" gorm:"many2many:task_tags;"`
}

// TaskTag represents the many-to-many relationship between tasks and tags
type TaskTag struct {
	TaskID uuid.UUID `json:"task_id" gorm:"type:uuid;primaryKey"`
	TagID  uuid.UUID `json:"tag_id" gorm:"type:uuid;primaryKey"`
	
	// Relationships
	Task Task `json:"task" gorm:"foreignKey:TaskID"`
	Tag  Tag  `json:"tag" gorm:"foreignKey:TagID"`
}

// TableName specifies the table name for Task
func (Task) TableName() string {
	return "tasks"
}

// TableName specifies the table name for TaskComment
func (TaskComment) TableName() string {
	return "task_comments"
}

// TableName specifies the table name for TaskAttachment
func (TaskAttachment) TableName() string {
	return "task_attachments"
}

// TableName specifies the table name for TaskActivity
func (TaskActivity) TableName() string {
	return "task_activities"
}

// TableName specifies the table name for Project
func (Project) TableName() string {
	return "projects"
}

// TableName specifies the table name for Tag
func (Tag) TableName() string {
	return "tags"
}

// TableName specifies the table name for TaskTag
func (TaskTag) TableName() string {
	return "task_tags"
}

// IsCompleted checks if the task is completed
func (t *Task) IsCompleted() bool {
	return t.Status == TaskStatusCompleted
}

// IsOverdue checks if the task is overdue
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	return time.Now().After(*t.DueDate) && !t.IsCompleted()
}

// MarkAsCompleted marks the task as completed
func (t *Task) MarkAsCompleted() {
	t.Status = TaskStatusCompleted
	now := time.Now()
	t.CompletedAt = &now
}

// MarkAsIncomplete marks the task as incomplete
func (t *Task) MarkAsIncomplete() {
	t.Status = TaskStatusTodo
	t.CompletedAt = nil
}

// GetProgress returns the progress percentage of the task based on subtasks
func (t *Task) GetProgress() float64 {
	if len(t.Subtasks) == 0 {
		if t.IsCompleted() {
			return 100.0
		}
		return 0.0
	}
	
	completed := 0
	for _, subtask := range t.Subtasks {
		if subtask.IsCompleted() {
			completed++
		}
	}
	
	return float64(completed) / float64(len(t.Subtasks)) * 100.0
}

// CanBeAssignedTo checks if the task can be assigned to a specific user
func (t *Task) CanBeAssignedTo(userID uuid.UUID, user *User) bool {
	if user == nil {
		return false
	}
	
	// Users can only be assigned tasks within their tenant
	return user.TenantID == t.TenantID && user.IsActive()
}

// GetFileSizeFormatted returns the file size in a human-readable format
func (ta *TaskAttachment) GetFileSizeFormatted() string {
	const unit = 1024
	if ta.FileSize < unit {
		return fmt.Sprintf("%d B", ta.FileSize)
	}
	div, exp := int64(unit), 0
	for n := ta.FileSize / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(ta.FileSize)/float64(div), "KMGTPE"[exp])
}