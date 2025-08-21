package jobs

import (
	"time"

	"github.com/google/uuid"
)

// JobType represents different types of background jobs
type JobType string

const (
	// Email jobs
	JobTypeWelcomeEmail        JobType = "email:welcome"
	JobTypeTaskAssignedEmail   JobType = "email:task_assigned"
	JobTypeTaskDueEmail        JobType = "email:task_due"
	JobTypePasswordResetEmail  JobType = "email:password_reset"
	JobTypeInvitationEmail     JobType = "email:invitation"
	JobTypeWeeklyDigestEmail   JobType = "email:weekly_digest"

	// Analytics jobs
	JobTypeUserActivityAnalytics JobType = "analytics:user_activity"
	JobTypeTenantUsageUpdate     JobType = "analytics:tenant_usage"
	JobTypeGenerateReport        JobType = "analytics:generate_report"

	// Notification jobs
	JobTypePushNotification   JobType = "notification:push"
	JobTypeInAppNotification  JobType = "notification:in_app"
	JobTypeWebhookNotification JobType = "notification:webhook"

	// Maintenance jobs
	JobTypeCleanupSessions     JobType = "maintenance:cleanup_sessions"
	JobTypeCleanupTokens       JobType = "maintenance:cleanup_tokens"
	JobTypeBackupData          JobType = "maintenance:backup_data"
	JobTypeOptimizeDatabase    JobType = "maintenance:optimize_db"
)

// JobPriority represents job priority levels
type JobPriority int

const (
	PriorityLow    JobPriority = 1
	PriorityNormal JobPriority = 2
	PriorityHigh   JobPriority = 3
	PriorityUrgent JobPriority = 4
)

// JobQueue represents different job queues
type JobQueue string

const (
	QueueDefault     JobQueue = "default"
	QueueEmail       JobQueue = "email"
	QueueAnalytics   JobQueue = "analytics"
	QueueMaintenance JobQueue = "maintenance"
	QueueCritical    JobQueue = "critical"
)

// BaseJobPayload contains common fields for all job payloads
type BaseJobPayload struct {
	JobID     string    `json:"job_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UserID    uuid.UUID `json:"user_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Priority  int       `json:"priority"`
}

// Email Job Payloads

// WelcomeEmailPayload for welcome email jobs
type WelcomeEmailPayload struct {
	BaseJobPayload
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	LoginURL  string `json:"login_url"`
}

// TaskAssignedEmailPayload for task assignment notifications
type TaskAssignedEmailPayload struct {
	BaseJobPayload
	TaskID       uuid.UUID `json:"task_id"`
	TaskTitle    string    `json:"task_title"`
	AssigneeID   uuid.UUID `json:"assignee_id"`
	AssigneeEmail string   `json:"assignee_email"`
	AssigneeName string    `json:"assignee_name"`
	AssignerName string    `json:"assigner_name"`
	DueDate      *time.Time `json:"due_date,omitempty"`
	TaskURL      string     `json:"task_url"`
}

// TaskDueEmailPayload for task due date reminders
type TaskDueEmailPayload struct {
	BaseJobPayload
	TaskID       uuid.UUID `json:"task_id"`
	TaskTitle    string    `json:"task_title"`
	AssigneeID   uuid.UUID `json:"assignee_id"`
	AssigneeEmail string   `json:"assignee_email"`
	AssigneeName string    `json:"assignee_name"`
	DueDate      time.Time  `json:"due_date"`
	TaskURL      string     `json:"task_url"`
	HoursUntilDue int       `json:"hours_until_due"`
}

// PasswordResetEmailPayload for password reset emails
type PasswordResetEmailPayload struct {
	BaseJobPayload
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	ResetToken string `json:"reset_token"`
	ResetURL   string `json:"reset_url"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// InvitationEmailPayload for tenant invitation emails
type InvitationEmailPayload struct {
	BaseJobPayload
	InvitationID uuid.UUID `json:"invitation_id"`
	Email        string    `json:"email"`
	TenantName   string    `json:"tenant_name"`
	InviterName  string    `json:"inviter_name"`
	Role         string    `json:"role"`
	InviteToken  string    `json:"invite_token"`
	InviteURL    string    `json:"invite_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// WeeklyDigestEmailPayload for weekly digest emails
type WeeklyDigestEmailPayload struct {
	BaseJobPayload
	Email            string    `json:"email"`
	UserName         string    `json:"user_name"`
	WeekStartDate    time.Time `json:"week_start_date"`
	CompletedTasks   int       `json:"completed_tasks"`
	PendingTasks     int       `json:"pending_tasks"`
	OverdueTasks     int       `json:"overdue_tasks"`
	TeamPerformance  interface{} `json:"team_performance"`
}

// Analytics Job Payloads

// UserActivityAnalyticsPayload for user activity analysis
type UserActivityAnalyticsPayload struct {
	BaseJobPayload
	AnalysisDate    time.Time `json:"analysis_date"`
	IncludeDetails  bool      `json:"include_details"`
}

// TenantUsageUpdatePayload for tenant usage tracking
type TenantUsageUpdatePayload struct {
	BaseJobPayload
	Date          time.Time `json:"date"`
	UserCount     int       `json:"user_count"`
	TaskCount     int       `json:"task_count"`
	StorageUsed   int64     `json:"storage_used"`
	APIRequests   int       `json:"api_requests"`
	WSConnections int       `json:"websocket_connections"`
}

// GenerateReportPayload for report generation
type GenerateReportPayload struct {
	BaseJobPayload
	ReportType   string    `json:"report_type"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	Format       string    `json:"format"` // "pdf", "csv", "excel"
	DeliveryMethod string  `json:"delivery_method"` // "email", "download"
	Recipients   []string  `json:"recipients,omitempty"`
}

// Notification Job Payloads

// PushNotificationPayload for push notifications
type PushNotificationPayload struct {
	BaseJobPayload
	DeviceTokens []string               `json:"device_tokens"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Sound        string                 `json:"sound,omitempty"`
	Badge        int                    `json:"badge,omitempty"`
}

// InAppNotificationPayload for in-app notifications
type InAppNotificationPayload struct {
	BaseJobPayload
	NotificationID uuid.UUID              `json:"notification_id"`
	Type           string                 `json:"type"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Data           map[string]interface{} `json:"data,omitempty"`
	ActionURL      string                 `json:"action_url,omitempty"`
}

// WebhookNotificationPayload for webhook notifications
type WebhookNotificationPayload struct {
	BaseJobPayload
	WebhookURL  string                 `json:"webhook_url"`
	Event       string                 `json:"event"`
	Data        map[string]interface{} `json:"data"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
}

// Maintenance Job Payloads

// CleanupSessionsPayload for session cleanup
type CleanupSessionsPayload struct {
	BaseJobPayload
	ExpiredBefore time.Time `json:"expired_before"`
	BatchSize     int       `json:"batch_size"`
}

// CleanupTokensPayload for token cleanup
type CleanupTokensPayload struct {
	BaseJobPayload
	ExpiredBefore time.Time `json:"expired_before"`
	TokenType     string    `json:"token_type"` // "reset", "verification", "invitation"
	BatchSize     int       `json:"batch_size"`
}

// BackupDataPayload for data backup
type BackupDataPayload struct {
	BaseJobPayload
	BackupType    string   `json:"backup_type"` // "full", "incremental"
	Tables        []string `json:"tables,omitempty"`
	S3Bucket      string   `json:"s3_bucket"`
	BackupPath    string   `json:"backup_path"`
	RetentionDays int      `json:"retention_days"`
}

// OptimizeDatabasePayload for database optimization
type OptimizeDatabasePayload struct {
	BaseJobPayload
	OperationType string   `json:"operation_type"` // "vacuum", "reindex", "analyze"
	Tables        []string `json:"tables,omitempty"`
}

// JobResult represents the result of a job execution
type JobResult struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	CreatedAt time.Time              `json:"created_at"`
}

// JobRetryConfig defines retry behavior for jobs
type JobRetryConfig struct {
	MaxRetries   int           `json:"max_retries"`
	InitialDelay time.Duration `json:"initial_delay"`
	BackoffType  string        `json:"backoff_type"` // "exponential", "linear", "fixed"
	MaxDelay     time.Duration `json:"max_delay"`
}

// JobScheduleConfig defines scheduling for recurring jobs
type JobScheduleConfig struct {
	CronExpression string    `json:"cron_expression"`
	Timezone       string    `json:"timezone"`
	StartDate      time.Time `json:"start_date"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	MaxRuns        *int      `json:"max_runs,omitempty"`
}