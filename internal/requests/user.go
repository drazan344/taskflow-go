package requests

import (
	"github.com/google/uuid"
	"github.com/drazan344/taskflow-go/internal/models"
)

// UpdateUserRequest represents a user update request with validation
type UpdateUserRequest struct {
	FirstName *string           `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName  *string           `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Email     *string           `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Phone     *string           `json:"phone,omitempty" validate:"omitempty,max=20"`
	Avatar    *string           `json:"avatar,omitempty" validate:"omitempty,url,max=500"`
	Timezone  *string           `json:"timezone,omitempty" validate:"omitempty,max=50"`
	Language  *string           `json:"language,omitempty" validate:"omitempty,max=10"`
	Role      *models.UserRole  `json:"role,omitempty" validate:"omitempty,oneof=admin manager user"`
	Status    *models.UserStatus `json:"status,omitempty" validate:"omitempty,oneof=active inactive suspended"`
}

// UpdateUserPreferencesRequest represents a user preferences update request
type UpdateUserPreferencesRequest struct {
	Theme                    *string `json:"theme,omitempty" validate:"omitempty,oneof=light dark auto"`
	EmailNotifications       *bool   `json:"email_notifications,omitempty"`
	PushNotifications       *bool   `json:"push_notifications,omitempty"`
	TaskReminders           *bool   `json:"task_reminders,omitempty"`
	WeeklyDigest            *bool   `json:"weekly_digest,omitempty"`
	DefaultTaskPriority     *string `json:"default_task_priority,omitempty" validate:"omitempty,oneof=low medium high urgent"`
	TaskViewMode            *string `json:"task_view_mode,omitempty" validate:"omitempty,oneof=list board calendar"`
	ShowCompletedTasks      *bool   `json:"show_completed_tasks,omitempty"`
	TasksPerPage            *int    `json:"tasks_per_page,omitempty" validate:"omitempty,min=5,max=100"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,password"`
}

// UserFiltersRequest represents user filtering parameters
type UserFiltersRequest struct {
	Role     *models.UserRole   `form:"role" validate:"omitempty,oneof=admin manager user"`
	Status   *models.UserStatus `form:"status" validate:"omitempty,oneof=active inactive suspended"`
	Search   string             `form:"search" validate:"max=100"`
}

// CreateUserInvitationRequest represents a user invitation request
type CreateUserInvitationRequest struct {
	Email     string           `json:"email" validate:"required,email,max=255"`
	Role      models.UserRole  `json:"role" validate:"required,oneof=admin manager user"`
	FirstName *string          `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName  *string          `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Message   *string          `json:"message,omitempty" validate:"omitempty,max=500"`
}

// BulkUserOperationRequest represents a bulk operation on users
type BulkUserOperationRequest struct {
	UserIDs   []uuid.UUID        `json:"user_ids" validate:"required,min=1,dive,uuid"`
	Operation string             `json:"operation" validate:"required,oneof=activate deactivate suspend delete"`
	Status    *models.UserStatus `json:"status,omitempty"`
}