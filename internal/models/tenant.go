package models

import (
	"time"

	"github.com/google/uuid"
)

// TenantStatus represents the status of a tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusCanceled  TenantStatus = "canceled"
)

// TenantPlan represents the subscription plan of a tenant
type TenantPlan string

const (
	TenantPlanFree       TenantPlan = "free"
	TenantPlanBasic      TenantPlan = "basic"
	TenantPlanPro        TenantPlan = "pro"
	TenantPlanEnterprise TenantPlan = "enterprise"
)

// Tenant represents a company/organization in the multi-tenant system
type Tenant struct {
	BaseModel
	Name        string       `json:"name" gorm:"not null;size:255"`
	Slug        string       `json:"slug" gorm:"unique;not null;size:100"`
	Domain      string       `json:"domain,omitempty" gorm:"size:255"`
	Status      TenantStatus `json:"status" gorm:"default:'active'"`
	Plan        TenantPlan   `json:"plan" gorm:"default:'free'"`
	MaxUsers    int          `json:"max_users" gorm:"default:10"`
	MaxTasks    int          `json:"max_tasks" gorm:"default:1000"`
	MaxStorage  int64        `json:"max_storage" gorm:"default:1073741824"` // 1GB in bytes
	Settings    TenantSettings `json:"settings" gorm:"type:jsonb"`
	
	// Relationships
	Users []User `json:"users,omitempty" gorm:"foreignKey:TenantID"`
	Tasks []Task `json:"tasks,omitempty" gorm:"foreignKey:TenantID"`
}

// TenantSettings contains configurable settings for a tenant
type TenantSettings struct {
	AllowRegistration     bool   `json:"allow_registration"`
	RequireEmailVerification bool `json:"require_email_verification"`
	DefaultUserRole       string `json:"default_user_role"`
	TaskAutoAssignment    bool   `json:"task_auto_assignment"`
	NotificationSettings  NotificationSettings `json:"notification_settings"`
	BrandingSettings      BrandingSettings     `json:"branding_settings"`
}

// NotificationSettings contains notification preferences
type NotificationSettings struct {
	EmailNotifications    bool `json:"email_notifications"`
	TaskAssignments      bool `json:"task_assignments"`
	TaskDueDates         bool `json:"task_due_dates"`
	TaskCompletions      bool `json:"task_completions"`
	WeeklyDigest         bool `json:"weekly_digest"`
}

// BrandingSettings contains branding customization
type BrandingSettings struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	LogoURL        string `json:"logo_url"`
	FaviconURL     string `json:"favicon_url"`
}

// TenantInvitation represents an invitation to join a tenant
type TenantInvitation struct {
	BaseModel
	TenantID    uuid.UUID          `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Email       string             `json:"email" gorm:"not null;size:255"`
	Role        UserRole           `json:"role" gorm:"not null"`
	Status      InvitationStatus   `json:"status" gorm:"default:'pending'"`
	Token       string             `json:"token" gorm:"unique;not null;size:100"`
	InvitedBy   uuid.UUID          `json:"invited_by" gorm:"type:uuid;not null"`
	ExpiresAt   time.Time          `json:"expires_at" gorm:"not null"`
	AcceptedAt  *time.Time         `json:"accepted_at,omitempty"`
	
	// Relationships
	Tenant    Tenant `json:"tenant" gorm:"foreignKey:TenantID"`
	Inviter   User   `json:"inviter" gorm:"foreignKey:InvitedBy"`
}

// InvitationStatus represents the status of a tenant invitation
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusExpired  InvitationStatus = "expired"
	InvitationStatusCanceled InvitationStatus = "canceled"
)

// TenantUsage tracks usage metrics for a tenant
type TenantUsage struct {
	BaseModel
	TenantID      uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Date          time.Time `json:"date" gorm:"type:date;not null"`
	UserCount     int       `json:"user_count"`
	TaskCount     int       `json:"task_count"`
	StorageUsed   int64     `json:"storage_used"`
	APIRequests   int       `json:"api_requests"`
	WebSocketConns int      `json:"websocket_connections"`
	
	// Relationships
	Tenant Tenant `json:"tenant" gorm:"foreignKey:TenantID"`
}

// TableName specifies the table name for Tenant
func (Tenant) TableName() string {
	return "tenants"
}

// TableName specifies the table name for TenantInvitation
func (TenantInvitation) TableName() string {
	return "tenant_invitations"
}

// TableName specifies the table name for TenantUsage
func (TenantUsage) TableName() string {
	return "tenant_usage"
}

// IsActive checks if the tenant is active
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// CanAddUser checks if the tenant can add more users
func (t *Tenant) CanAddUser() bool {
	return len(t.Users) < t.MaxUsers
}

// CanAddTask checks if the tenant can add more tasks
func (t *Tenant) CanAddTask() bool {
	return len(t.Tasks) < t.MaxTasks
}

// IsExpired checks if the invitation has expired
func (ti *TenantInvitation) IsExpired() bool {
	return time.Now().After(ti.ExpiresAt)
}

// IsPending checks if the invitation is still pending
func (ti *TenantInvitation) IsPending() bool {
	return ti.Status == InvitationStatusPending && !ti.IsExpired()
}