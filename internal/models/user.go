package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserRole represents the role of a user within a tenant
type UserRole string

const (
	UserRoleAdmin   UserRole = "admin"
	UserRoleManager UserRole = "manager"
	UserRoleUser    UserRole = "user"
)

// UserStatus represents the status of a user
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
)

// User represents a user in the system
type User struct {
	TenantModel
	Email         string     `json:"email" gorm:"not null;size:255;index"`
	FirstName     string     `json:"first_name" gorm:"not null;size:100"`
	LastName      string     `json:"last_name" gorm:"not null;size:100"`
	Password      string     `json:"-" gorm:"not null;size:255"`
	Role          UserRole   `json:"role" gorm:"not null;default:'user'"`
	Status        UserStatus `json:"status" gorm:"default:'active'"`
	Avatar        string     `json:"avatar,omitempty" gorm:"size:500"`
	Phone         string     `json:"phone,omitempty" gorm:"size:20"`
	Timezone      string     `json:"timezone" gorm:"default:'UTC';size:50"`
	Language      string     `json:"language" gorm:"default:'en';size:10"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	EmailVerified bool       `json:"email_verified" gorm:"default:false"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	
	// User preferences
	Preferences UserPreferences `json:"preferences" gorm:"type:jsonb"`
	
	// Relationships
	Tenant          Tenant            `json:"tenant" gorm:"foreignKey:TenantID"`
	AssignedTasks   []Task            `json:"assigned_tasks,omitempty" gorm:"foreignKey:AssigneeID"`
	CreatedTasks    []Task            `json:"created_tasks,omitempty" gorm:"foreignKey:CreatorID"`
	Sessions        []UserSession     `json:"-" gorm:"foreignKey:UserID"`
	Notifications   []Notification    `json:"notifications,omitempty" gorm:"foreignKey:UserID"`
}

// UserPreferences contains user-specific preferences
type UserPreferences struct {
	Theme                    string `json:"theme"` // light, dark, auto
	EmailNotifications       bool   `json:"email_notifications"`
	PushNotifications       bool   `json:"push_notifications"`
	TaskReminders           bool   `json:"task_reminders"`
	WeeklyDigest            bool   `json:"weekly_digest"`
	DefaultTaskPriority     string `json:"default_task_priority"`
	TaskViewMode            string `json:"task_view_mode"` // list, board, calendar
	ShowCompletedTasks      bool   `json:"show_completed_tasks"`
	TasksPerPage            int    `json:"tasks_per_page"`
}

// UserSession represents an active user session
type UserSession struct {
	BaseModel
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID    uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Token       string    `json:"token" gorm:"unique;not null;size:500"`
	RefreshToken string   `json:"refresh_token" gorm:"unique;not null;size:500"`
	ExpiresAt   time.Time `json:"expires_at" gorm:"not null"`
	IPAddress   string    `json:"ip_address" gorm:"size:45"`
	UserAgent   string    `json:"user_agent" gorm:"size:500"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	LastUsedAt  time.Time `json:"last_used_at" gorm:"autoUpdateTime"`
	
	// Relationships
	User   User   `json:"user" gorm:"foreignKey:UserID"`
	Tenant Tenant `json:"tenant" gorm:"foreignKey:TenantID"`
}

// UserPasswordReset represents a password reset request
type UserPasswordReset struct {
	BaseModel
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Token     string    `json:"token" gorm:"unique;not null;size:100"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	
	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// UserEmailVerification represents an email verification token
type UserEmailVerification struct {
	BaseModel
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Email     string    `json:"email" gorm:"not null;size:255"`
	Token     string    `json:"token" gorm:"unique;not null;size:100"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	
	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// TableName specifies the table name for UserSession
func (UserSession) TableName() string {
	return "user_sessions"
}

// TableName specifies the table name for UserPasswordReset
func (UserPasswordReset) TableName() string {
	return "user_password_resets"
}

// TableName specifies the table name for UserEmailVerification
func (UserEmailVerification) TableName() string {
	return "user_email_verifications"
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	return u.FirstName + " " + u.LastName
}

// IsActive checks if the user is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsAdmin checks if the user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// IsManager checks if the user is a manager
func (u *User) IsManager() bool {
	return u.Role == UserRoleManager
}

// CanManage checks if the user can manage (admin or manager)
func (u *User) CanManage() bool {
	return u.Role == UserRoleAdmin || u.Role == UserRoleManager
}

// SetPassword hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword verifies if the provided password matches the user's password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// UpdateLastLogin updates the user's last login timestamp
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
}

// IsExpired checks if the session has expired
func (us *UserSession) IsExpired() bool {
	return time.Now().After(us.ExpiresAt)
}

// IsExpired checks if the password reset token has expired
func (upr *UserPasswordReset) IsExpired() bool {
	return time.Now().After(upr.ExpiresAt)
}

// IsUsed checks if the password reset token has been used
func (upr *UserPasswordReset) IsUsed() bool {
	return upr.UsedAt != nil
}

// IsValid checks if the password reset token is valid (not expired and not used)
func (upr *UserPasswordReset) IsValid() bool {
	return !upr.IsExpired() && !upr.IsUsed()
}

// MarkAsUsed marks the password reset token as used
func (upr *UserPasswordReset) MarkAsUsed() {
	now := time.Now()
	upr.UsedAt = &now
}

// IsExpired checks if the email verification token has expired
func (uev *UserEmailVerification) IsExpired() bool {
	return time.Now().After(uev.ExpiresAt)
}

// IsVerified checks if the email has been verified
func (uev *UserEmailVerification) IsVerified() bool {
	return uev.VerifiedAt != nil
}

// IsValid checks if the email verification token is valid
func (uev *UserEmailVerification) IsValid() bool {
	return !uev.IsExpired() && !uev.IsVerified()
}

// MarkAsVerified marks the email as verified
func (uev *UserEmailVerification) MarkAsVerified() {
	now := time.Now()
	uev.VerifiedAt = &now
}