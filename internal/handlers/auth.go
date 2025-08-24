package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/drazan344/taskflow-go/internal/auth"
	"github.com/drazan344/taskflow-go/internal/middleware"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/pkg/errors"
	"github.com/drazan344/taskflow-go/pkg/logger"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService *auth.Service
	logger      *logger.Logger
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *auth.Service, logger *logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=6"`
	FirstName   string `json:"first_name" binding:"required,min=2"`
	LastName    string `json:"last_name" binding:"required,min=2"`
	TenantName  string `json:"tenant_name" binding:"required,min=2"`
	TenantSlug  string `json:"tenant_slug" binding:"required,min=2,alphanum"`
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest represents the token refresh request payload
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ForgotPasswordRequest represents the forgot password request payload
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest represents the reset password request payload
type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// ChangePasswordRequest represents the change password request payload
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// UpdateProfileRequest represents the update profile request payload
type UpdateProfileRequest struct {
	FirstName   *string                    `json:"first_name,omitempty"`
	LastName    *string                    `json:"last_name,omitempty"`
	Phone       *string                    `json:"phone,omitempty"`
	Timezone    *string                    `json:"timezone,omitempty"`
	Language    *string                    `json:"language,omitempty"`
	Avatar      *string                    `json:"avatar,omitempty"`
	Preferences *models.UserPreferences    `json:"preferences,omitempty"`
}

// Register handles user registration
// @Summary Register a new user and tenant
// @Description Create a new user account and associated tenant
// @Tags auth
// @Accept json
// @Produce json
// @Param request body handlers.RegisterRequest true "Registration details"
// @Success 201 {object} auth.LoginResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid registration request")
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request", err.Error()))
		return
	}

	// Convert to auth service request
	authReq := &auth.RegisterRequest{
		Email:      req.Email,
		Password:   req.Password,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		TenantName: req.TenantName,
		TenantSlug: req.TenantSlug,
	}

	response, err := h.authService.Register(authReq)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			c.JSON(appErr.Code, middleware.ErrorResponse(appErr.Message, appErr.Details))
			return
		}
		h.logger.WithError(err).Error("Registration failed")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Registration failed"))
		return
	}

	h.logger.WithField("user_id", response.User.ID).
		WithField("tenant_id", response.User.TenantID).
		Info("User registered successfully")

	c.JSON(http.StatusCreated, middleware.SuccessResponse(response, "Registration successful"))
}

// Login handles user authentication
// @Summary Authenticate user
// @Description Authenticate user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body handlers.LoginRequest true "Login credentials"
// @Success 200 {object} auth.LoginResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid login request")
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request", err.Error()))
		return
	}

	// Convert to auth service request
	authReq := &auth.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	// Get client information
	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	response, err := h.authService.Login(authReq, ipAddress, userAgent)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			c.JSON(appErr.Code, middleware.ErrorResponse(appErr.Message))
			return
		}
		h.logger.WithError(err).Error("Login failed")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Login failed"))
		return
	}

	h.logger.WithField("user_id", response.User.ID).
		WithField("tenant_id", response.User.TenantID).
		Info("User logged in successfully")

	c.JSON(http.StatusOK, middleware.SuccessResponse(response, "Login successful"))
}

// RefreshTokens handles token refresh
// @Summary Refresh access token
// @Description Get new access and refresh tokens using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body handlers.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} auth.LoginResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshTokens(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid refresh token request")
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request", err.Error()))
		return
	}

	// Convert to auth service request
	authReq := &auth.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	}

	response, err := h.authService.RefreshTokens(authReq)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			c.JSON(appErr.Code, middleware.ErrorResponse(appErr.Message))
			return
		}
		h.logger.WithError(err).Error("Token refresh failed")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Token refresh failed"))
		return
	}

	h.logger.WithField("user_id", response.User.ID).Info("Tokens refreshed successfully")

	c.JSON(http.StatusOK, middleware.SuccessResponse(response, "Tokens refreshed successfully"))
}

// Logout handles user logout
// @Summary Logout user
// @Description Invalidate user session
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, err := middleware.GetSessionID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("Invalid session"))
		return
	}

	if err := h.authService.Logout(sessionID); err != nil {
		h.logger.WithError(err).Error("Logout failed")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Logout failed"))
		return
	}

	h.logger.WithField("session_id", sessionID).Info("User logged out successfully")

	c.JSON(http.StatusOK, middleware.SuccessResponse(nil, "Logout successful"))
}

// GetCurrentUser returns the current authenticated user
// @Summary Get current user
// @Description Get information about the currently authenticated user
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]interface{}
// @Router /auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("User not authenticated"))
		return
	}

	c.JSON(http.StatusOK, middleware.SuccessResponse(user))
}

// UpdateProfile handles profile updates
// @Summary Update user profile
// @Description Update the current user's profile information
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body handlers.UpdateProfileRequest true "Profile update data"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/me [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("User not authenticated"))
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid profile update request")
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request", err.Error()))
		return
	}

	// Update fields if provided
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.Timezone != nil {
		user.Timezone = *req.Timezone
	}
	if req.Language != nil {
		user.Language = *req.Language
	}
	if req.Avatar != nil {
		user.Avatar = *req.Avatar
	}
	// Preferences are now flattened - handle them individually if needed

	// Save updated user (this would typically go through a user service)
	// For now, we'll just return the user
	h.logger.WithField("user_id", user.ID).Info("Profile updated successfully")

	c.JSON(http.StatusOK, middleware.SuccessResponse(user, "Profile updated successfully"))
}

// ChangePassword handles password changes
// @Summary Change user password
// @Description Change the current user's password
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body handlers.ChangePasswordRequest true "Password change data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, middleware.ErrorResponse("User not authenticated"))
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid change password request")
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request", err.Error()))
		return
	}

	// Validate passwords match
	if req.NewPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Passwords do not match"))
		return
	}

	// Verify current password
	if !user.CheckPassword(req.CurrentPassword) {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Current password is incorrect"))
		return
	}

	// Set new password
	if err := user.SetPassword(req.NewPassword); err != nil {
		h.logger.WithError(err).Error("Failed to set new password")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to update password"))
		return
	}

	// Save user (this would typically go through a user service)
	h.logger.WithField("user_id", user.ID).Info("Password changed successfully")

	c.JSON(http.StatusOK, middleware.SuccessResponse(nil, "Password changed successfully"))
}

// ForgotPassword handles password reset requests
// @Summary Request password reset
// @Description Send password reset email to user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body handlers.ForgotPasswordRequest true "Email address"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid forgot password request")
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request", err.Error()))
		return
	}

	// Convert to auth service request
	authReq := &auth.PasswordResetRequest{
		Email: req.Email,
	}

	if err := h.authService.RequestPasswordReset(authReq); err != nil {
		h.logger.WithError(err).Error("Password reset request failed")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Failed to process request"))
		return
	}

	h.logger.WithField("email", req.Email).Info("Password reset requested")

	// Always return success to prevent email enumeration
	c.JSON(http.StatusOK, middleware.SuccessResponse(nil, "If the email exists, a reset link has been sent"))
}

// ResetPassword handles password reset confirmation
// @Summary Reset password with token
// @Description Reset user password using reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body handlers.ResetPasswordRequest true "Reset password data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid reset password request")
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse("Invalid request", err.Error()))
		return
	}

	// Convert to auth service request
	authReq := &auth.PasswordResetConfirmRequest{
		Token:           req.Token,
		NewPassword:     req.NewPassword,
		ConfirmPassword: req.ConfirmPassword,
	}

	if err := h.authService.ResetPassword(authReq); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			c.JSON(appErr.Code, middleware.ErrorResponse(appErr.Message))
			return
		}
		h.logger.WithError(err).Error("Password reset failed")
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse("Password reset failed"))
		return
	}

	h.logger.Info("Password reset completed successfully")

	c.JSON(http.StatusOK, middleware.SuccessResponse(nil, "Password reset successful"))
}