package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/drazan344/taskflow-go/internal/config"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/pkg/errors"
	"gorm.io/gorm"
)

// Service handles authentication operations
type Service struct {
	db         *gorm.DB
	jwtService *JWTService
	config     *config.Config
}

// NewService creates a new authentication service
func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:         db,
		jwtService: NewJWTService(cfg),
		config:     cfg,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=6"`
	FirstName   string `json:"first_name" binding:"required,min=2"`
	LastName    string `json:"last_name" binding:"required,min=2"`
	TenantName  string `json:"tenant_name" binding:"required,min=2"`
	TenantSlug  string `json:"tenant_slug" binding:"required,min=2"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// PasswordResetConfirmRequest represents a password reset confirmation
type PasswordResetConfirmRequest struct {
	Token           string `json:"token" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// Login authenticates a user and returns tokens
func (s *Service) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// Find user by email
	var user models.User
	if err := s.db.Preload("Tenant").Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.Unauthorized("Invalid email or password", errors.ErrInvalidCredentials)
		}
		return nil, errors.InternalServer("Failed to find user", err)
	}

	// Check if user is active
	if !user.IsActive() {
		return nil, errors.Unauthorized("Account is not active", errors.ErrUnauthorized)
	}

	// Check if tenant is active
	if !user.Tenant.IsActive() {
		return nil, errors.Unauthorized("Account is suspended", errors.ErrUnauthorized)
	}

	// Verify password
	if !user.CheckPassword(req.Password) {
		return nil, errors.Unauthorized("Invalid email or password", errors.ErrInvalidCredentials)
	}

	// Create user session
	session, err := s.createUserSession(&user, ipAddress, userAgent)
	if err != nil {
		return nil, errors.InternalServer("Failed to create session", err)
	}

	// Generate token pair
	accessToken, refreshToken, err := s.jwtService.GenerateTokenPair(&user, session.ID)
	if err != nil {
		return nil, errors.InternalServer("Failed to generate tokens", err)
	}

	// Update session tokens
	session.Token = accessToken
	session.RefreshToken = refreshToken
	if err := s.db.Save(session).Error; err != nil {
		return nil, errors.InternalServer("Failed to save session", err)
	}

	// Update user last login
	user.UpdateLastLogin()
	if err := s.db.Save(&user).Error; err != nil {
		return nil, errors.InternalServer("Failed to update user", err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(s.config.JWT.Expiry)

	return &LoginResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// Register creates a new user and tenant
func (s *Service) Register(req *RegisterRequest) (*LoginResponse, error) {
	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if email already exists
	var existingUser models.User
	if err := tx.Where("email = ?", req.Email).First(&existingUser).Error; err != gorm.ErrRecordNotFound {
		tx.Rollback()
		return nil, errors.Conflict("Email already exists", errors.ErrConflict)
	}

	// Check if tenant slug already exists
	var existingTenant models.Tenant
	if err := tx.Where("slug = ?", req.TenantSlug).First(&existingTenant).Error; err != gorm.ErrRecordNotFound {
		tx.Rollback()
		return nil, errors.Conflict("Tenant slug already exists", errors.ErrConflict)
	}

	// Create tenant
	tenant := &models.Tenant{
		Name:   req.TenantName,
		Slug:   req.TenantSlug,
		Status: models.TenantStatusActive,
		Plan:   models.TenantPlanFree,
		Settings: models.TenantSettings{
			AllowRegistration:        false,
			RequireEmailVerification: true,
			DefaultUserRole:         string(models.UserRoleUser),
			TaskAutoAssignment:      false,
			NotificationSettings: models.NotificationSettings{
				EmailNotifications: true,
				TaskAssignments:   true,
				TaskDueDates:      true,
				TaskCompletions:   true,
				WeeklyDigest:      false,
			},
			BrandingSettings: models.BrandingSettings{
				PrimaryColor:   "#3B82F6",
				SecondaryColor: "#6B7280",
			},
		},
	}

	if err := tx.Create(tenant).Error; err != nil {
		tx.Rollback()
		return nil, errors.InternalServer("Failed to create tenant", err)
	}

	// Create user
	user := &models.User{
		TenantModel: models.TenantModel{TenantID: tenant.ID},
		Email:       req.Email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Role:        models.UserRoleAdmin, // First user in tenant is admin
		Status:      models.UserStatusActive,
		Timezone:    "UTC",
		Language:    "en",
		Preferences: models.UserPreferences{
			Theme:               "light",
			EmailNotifications:  true,
			TaskReminders:      true,
			DefaultTaskPriority: "medium",
			TaskViewMode:       "list",
			TasksPerPage:       20,
		},
	}

	if err := user.SetPassword(req.Password); err != nil {
		tx.Rollback()
		return nil, errors.InternalServer("Failed to hash password", err)
	}

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, errors.InternalServer("Failed to create user", err)
	}

	// Create default notification preferences
	if err := s.createDefaultNotificationPreferences(tx, user.ID, user.TenantID); err != nil {
		tx.Rollback()
		return nil, errors.InternalServer("Failed to create notification preferences", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, errors.InternalServer("Failed to commit transaction", err)
	}

	// Reload user with tenant data
	if err := s.db.Preload("Tenant").First(user, user.ID).Error; err != nil {
		return nil, errors.InternalServer("Failed to reload user", err)
	}

	// Create session and generate tokens
	session, err := s.createUserSession(user, "", "")
	if err != nil {
		return nil, errors.InternalServer("Failed to create session", err)
	}

	accessToken, refreshToken, err := s.jwtService.GenerateTokenPair(user, session.ID)
	if err != nil {
		return nil, errors.InternalServer("Failed to generate tokens", err)
	}

	// Update session tokens
	session.Token = accessToken
	session.RefreshToken = refreshToken
	if err := s.db.Save(session).Error; err != nil {
		return nil, errors.InternalServer("Failed to save session", err)
	}

	expiresAt := time.Now().Add(s.config.JWT.Expiry)

	return &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// RefreshTokens refreshes access and refresh tokens
func (s *Service) RefreshTokens(req *RefreshTokenRequest) (*LoginResponse, error) {
	// Find session by refresh token
	var session models.UserSession
	if err := s.db.Preload("User.Tenant").Where("refresh_token = ? AND is_active = ?", req.RefreshToken, true).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.Unauthorized("Invalid refresh token", errors.ErrInvalidToken)
		}
		return nil, errors.InternalServer("Failed to find session", err)
	}

	// Check if session is expired
	if session.IsExpired() {
		return nil, errors.Unauthorized("Session has expired", errors.ErrTokenExpired)
	}

	// Generate new token pair
	accessToken, refreshToken, err := s.jwtService.RefreshTokens(req.RefreshToken, &session.User, session.ID)
	if err != nil {
		return nil, errors.Unauthorized("Failed to refresh tokens", err)
	}

	// Update session
	session.Token = accessToken
	session.RefreshToken = refreshToken
	session.ExpiresAt = time.Now().Add(s.config.JWT.RefreshExpiry)
	if err := s.db.Save(&session).Error; err != nil {
		return nil, errors.InternalServer("Failed to update session", err)
	}

	expiresAt := time.Now().Add(s.config.JWT.Expiry)

	return &LoginResponse{
		User:         &session.User,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// Logout invalidates user session
func (s *Service) Logout(sessionID uuid.UUID) error {
	if err := s.db.Model(&models.UserSession{}).Where("id = ?", sessionID).Update("is_active", false).Error; err != nil {
		return errors.InternalServer("Failed to logout", err)
	}
	return nil
}

// RequestPasswordReset creates a password reset token
func (s *Service) RequestPasswordReset(req *PasswordResetRequest) error {
	// Find user by email
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Don't reveal if email exists, but return success
			return nil
		}
		return errors.InternalServer("Failed to find user", err)
	}

	// Generate reset token
	token, err := generateSecureToken(32)
	if err != nil {
		return errors.InternalServer("Failed to generate reset token", err)
	}

	// Create password reset record
	passwordReset := &models.UserPasswordReset{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour), // 1 hour expiry
	}

	if err := s.db.Create(passwordReset).Error; err != nil {
		return errors.InternalServer("Failed to create password reset", err)
	}

	// TODO: Send email with reset token
	// This would be handled by the email service/background job

	return nil
}

// ResetPassword resets user password using reset token
func (s *Service) ResetPassword(req *PasswordResetConfirmRequest) error {
	if req.NewPassword != req.ConfirmPassword {
		return errors.BadRequest("Passwords do not match", errors.ErrValidation)
	}

	// Find valid password reset token
	var passwordReset models.UserPasswordReset
	if err := s.db.Preload("User").Where("token = ?", req.Token).First(&passwordReset).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.BadRequest("Invalid or expired reset token", errors.ErrValidation)
		}
		return errors.InternalServer("Failed to find password reset", err)
	}

	// Check if token is valid
	if !passwordReset.IsValid() {
		return errors.BadRequest("Invalid or expired reset token", errors.ErrValidation)
	}

	// Update user password
	if err := passwordReset.User.SetPassword(req.NewPassword); err != nil {
		return errors.InternalServer("Failed to hash password", err)
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Save user
	if err := tx.Save(&passwordReset.User).Error; err != nil {
		tx.Rollback()
		return errors.InternalServer("Failed to update password", err)
	}

	// Mark reset token as used
	passwordReset.MarkAsUsed()
	if err := tx.Save(&passwordReset).Error; err != nil {
		tx.Rollback()
		return errors.InternalServer("Failed to mark token as used", err)
	}

	// Invalidate all user sessions
	if err := tx.Model(&models.UserSession{}).Where("user_id = ?", passwordReset.User.ID).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		return errors.InternalServer("Failed to invalidate sessions", err)
	}

	if err := tx.Commit().Error; err != nil {
		return errors.InternalServer("Failed to commit transaction", err)
	}

	return nil
}

// ValidateSession validates a user session
func (s *Service) ValidateSession(sessionID uuid.UUID) (*models.User, error) {
	var session models.UserSession
	if err := s.db.Preload("User.Tenant").Where("id = ? AND is_active = ?", sessionID, true).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.Unauthorized("Invalid session", errors.ErrUnauthorized)
		}
		return nil, errors.InternalServer("Failed to find session", err)
	}

	if session.IsExpired() {
		return nil, errors.Unauthorized("Session has expired", errors.ErrTokenExpired)
	}

	return &session.User, nil
}

// createUserSession creates a new user session
func (s *Service) createUserSession(user *models.User, ipAddress, userAgent string) (*models.UserSession, error) {
	session := &models.UserSession{
		UserID:    user.ID,
		TenantID:  user.TenantID,
		ExpiresAt: time.Now().Add(s.config.JWT.RefreshExpiry),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		IsActive:  true,
	}

	if err := s.db.Create(session).Error; err != nil {
		return nil, err
	}

	return session, nil
}

// createDefaultNotificationPreferences creates default notification preferences for a user
func (s *Service) createDefaultNotificationPreferences(tx *gorm.DB, userID, tenantID uuid.UUID) error {
	preferences := []models.NotificationPreference{
		{TenantModel: models.TenantModel{TenantID: tenantID}, UserID: userID, Type: models.NotificationTypeTaskAssigned, InApp: true, Email: true, Push: true, WebSocket: true},
		{TenantModel: models.TenantModel{TenantID: tenantID}, UserID: userID, Type: models.NotificationTypeTaskCompleted, InApp: true, Email: false, Push: true, WebSocket: true},
		{TenantModel: models.TenantModel{TenantID: tenantID}, UserID: userID, Type: models.NotificationTypeTaskDue, InApp: true, Email: true, Push: true, WebSocket: true},
		{TenantModel: models.TenantModel{TenantID: tenantID}, UserID: userID, Type: models.NotificationTypeCommentAdded, InApp: true, Email: false, Push: true, WebSocket: true},
	}

	for _, pref := range preferences {
		if err := tx.Create(&pref).Error; err != nil {
			return err
		}
	}

	return nil
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}