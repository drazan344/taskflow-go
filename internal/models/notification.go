package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeWelcome        NotificationType = "welcome"
	NotificationTypeTaskAssigned   NotificationType = "task_assigned"
	NotificationTypeTaskCreated    NotificationType = "task_created"
	NotificationTypeTaskCompleted  NotificationType = "task_completed"
	NotificationTypeTaskUpdated    NotificationType = "task_updated"
	NotificationTypeTaskDue        NotificationType = "task_due"
	NotificationTypeTaskOverdue    NotificationType = "task_overdue"
	NotificationTypeCommentAdded   NotificationType = "comment_added"
	NotificationTypeUserInvited    NotificationType = "user_invited"
	NotificationTypeUserJoined     NotificationType = "user_joined"
	NotificationTypeProjectCreated NotificationType = "project_created"
	NotificationTypeSystemUpdate   NotificationType = "system_update"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusUnread NotificationStatus = "unread"
	NotificationStatusRead   NotificationStatus = "read"
	NotificationStatusArchived NotificationStatus = "archived"
)

// NotificationChannel represents the delivery channel for notifications
type NotificationChannel string

const (
	NotificationChannelInApp     NotificationChannel = "in_app"
	NotificationChannelEmail     NotificationChannel = "email"
	NotificationChannelWebSocket NotificationChannel = "websocket"
	NotificationChannelPush      NotificationChannel = "push"
)

// Notification represents a notification in the system
type Notification struct {
	TenantModel
	UserID      uuid.UUID           `json:"user_id" gorm:"type:uuid;not null;index"`
	Type        NotificationType    `json:"type" gorm:"not null;size:50"`
	Status      NotificationStatus  `json:"status" gorm:"default:'unread'"`
	Title       string              `json:"title" gorm:"not null;size:255"`
	Message     string              `json:"message" gorm:"not null;type:text"`
	ActionURL   string              `json:"action_url,omitempty" gorm:"size:500"`
	Data        NotificationData    `json:"data" gorm:"type:jsonb"`
	ReadAt      *time.Time          `json:"read_at,omitempty"`
	ArchivedAt  *time.Time          `json:"archived_at,omitempty"`
	
	// Related entity references
	TaskID      *uuid.UUID `json:"task_id,omitempty" gorm:"type:uuid"`
	ProjectID   *uuid.UUID `json:"project_id,omitempty" gorm:"type:uuid"`
	CommentID   *uuid.UUID `json:"comment_id,omitempty" gorm:"type:uuid"`
	
	// Relationships
	User    User         `json:"user" gorm:"foreignKey:UserID"`
	Task    *Task        `json:"task,omitempty" gorm:"foreignKey:TaskID"`
	Project *Project     `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Comment *TaskComment `json:"comment,omitempty" gorm:"foreignKey:CommentID"`
}

// NotificationData contains additional structured data for notifications
type NotificationData struct {
	ActorID       *uuid.UUID `json:"actor_id,omitempty"`
	ActorName     string     `json:"actor_name,omitempty"`
	EntityType    string     `json:"entity_type,omitempty"`
	EntityID      string     `json:"entity_id,omitempty"`
	EntityName    string     `json:"entity_name,omitempty"`
	PreviousValue string     `json:"previous_value,omitempty"`
	NewValue      string     `json:"new_value,omitempty"`
	ExtraData     map[string]interface{} `json:"extra_data,omitempty"`
}

// NotificationPreference represents user notification preferences
type NotificationPreference struct {
	TenantModel
	UserID         uuid.UUID             `json:"user_id" gorm:"type:uuid;not null;index"`
	Type           NotificationType      `json:"type" gorm:"not null;size:50"`
	InApp          bool                  `json:"in_app" gorm:"default:true"`
	Email          bool                  `json:"email" gorm:"default:true"`
	Push           bool                  `json:"push" gorm:"default:true"`
	WebSocket      bool                  `json:"websocket" gorm:"default:true"`
	Frequency      NotificationFrequency `json:"frequency" gorm:"default:'immediate'"`
	
	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// NotificationFrequency represents how often notifications are sent
type NotificationFrequency string

const (
	NotificationFrequencyImmediate NotificationFrequency = "immediate"
	NotificationFrequencyDaily     NotificationFrequency = "daily"
	NotificationFrequencyWeekly    NotificationFrequency = "weekly"
	NotificationFrequencyNever     NotificationFrequency = "never"
)

// NotificationTemplate represents a template for generating notifications
type NotificationTemplate struct {
	BaseModel
	Type         NotificationType    `json:"type" gorm:"unique;not null;size:50"`
	Name         string              `json:"name" gorm:"not null;size:100"`
	Subject      string              `json:"subject" gorm:"not null;size:255"`
	Body         string              `json:"body" gorm:"not null;type:text"`
	HTMLBody     string              `json:"html_body" gorm:"type:text"`
	Variables    []string            `json:"variables" gorm:"type:jsonb"`
	IsActive     bool                `json:"is_active" gorm:"default:true"`
}

// NotificationQueue represents queued notifications for batch processing
type NotificationQueue struct {
	BaseModel
	TenantID    uuid.UUID           `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID      uuid.UUID           `json:"user_id" gorm:"type:uuid;not null;index"`
	Type        NotificationType    `json:"type" gorm:"not null;size:50"`
	Channel     NotificationChannel `json:"channel" gorm:"not null;size:20"`
	Priority    int                 `json:"priority" gorm:"default:1"`
	Payload     NotificationPayload `json:"payload" gorm:"type:jsonb"`
	Status      string              `json:"status" gorm:"default:'pending';size:20"`
	ScheduledAt time.Time           `json:"scheduled_at" gorm:"not null"`
	SentAt      *time.Time          `json:"sent_at,omitempty"`
	FailedAt    *time.Time          `json:"failed_at,omitempty"`
	RetryCount  int                 `json:"retry_count" gorm:"default:0"`
	LastError   string              `json:"last_error,omitempty" gorm:"type:text"`
	
	// Relationships
	Tenant Tenant `json:"tenant" gorm:"foreignKey:TenantID"`
	User   User   `json:"user" gorm:"foreignKey:UserID"`
}

// NotificationPayload contains the data needed to send a notification
type NotificationPayload struct {
	To          string                 `json:"to"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	HTMLBody    string                 `json:"html_body,omitempty"`
	ActionURL   string                 `json:"action_url,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// TableName specifies the table name for Notification
func (Notification) TableName() string {
	return "notifications"
}

// TableName specifies the table name for NotificationPreference
func (NotificationPreference) TableName() string {
	return "notification_preferences"
}

// TableName specifies the table name for NotificationTemplate
func (NotificationTemplate) TableName() string {
	return "notification_templates"
}

// TableName specifies the table name for NotificationQueue
func (NotificationQueue) TableName() string {
	return "notification_queue"
}

// IsRead checks if the notification has been read
func (n *Notification) IsRead() bool {
	return n.Status == NotificationStatusRead
}

// IsArchived checks if the notification has been archived
func (n *Notification) IsArchived() bool {
	return n.Status == NotificationStatusArchived
}

// MarkAsRead marks the notification as read
func (n *Notification) MarkAsRead() {
	n.Status = NotificationStatusRead
	now := time.Now()
	n.ReadAt = &now
}

// MarkAsArchived marks the notification as archived
func (n *Notification) MarkAsArchived() {
	n.Status = NotificationStatusArchived
	now := time.Now()
	n.ArchivedAt = &now
}

// ShouldSend checks if a notification should be sent based on user preferences
func (np *NotificationPreference) ShouldSend(channel NotificationChannel) bool {
	switch channel {
	case NotificationChannelInApp:
		return np.InApp
	case NotificationChannelEmail:
		return np.Email
	case NotificationChannelPush:
		return np.Push
	case NotificationChannelWebSocket:
		return np.WebSocket
	default:
		return false
	}
}

// IsPending checks if the queued notification is pending
func (nq *NotificationQueue) IsPending() bool {
	return nq.Status == "pending"
}

// IsSent checks if the queued notification has been sent
func (nq *NotificationQueue) IsSent() bool {
	return nq.Status == "sent"
}

// IsFailed checks if the queued notification has failed
func (nq *NotificationQueue) IsFailed() bool {
	return nq.Status == "failed"
}

// MarkAsSent marks the queued notification as sent
func (nq *NotificationQueue) MarkAsSent() {
	nq.Status = "sent"
	now := time.Now()
	nq.SentAt = &now
}

// MarkAsFailed marks the queued notification as failed
func (nq *NotificationQueue) MarkAsFailed(err error) {
	nq.Status = "failed"
	now := time.Now()
	nq.FailedAt = &now
	nq.RetryCount++
	if err != nil {
		nq.LastError = err.Error()
	}
}