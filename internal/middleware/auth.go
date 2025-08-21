package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/drazan344/taskflow-go/internal/auth"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/pkg/errors"
	"github.com/drazan344/taskflow-go/pkg/logger"
	"gorm.io/gorm"
)

// AuthMiddleware handles JWT authentication
func AuthMiddleware(jwtService *auth.JWTService, db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.WithFields(map[string]interface{}{
				"method": c.Request.Method,
				"path": c.Request.URL.Path,
				"user_agent": c.Request.UserAgent(),
			}).Warn("Missing authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from header
		tokenString, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			log.WithError(err).WithFields(map[string]interface{}{
				"method": c.Request.Method,
				"path": c.Request.URL.Path,
				"user_agent": c.Request.UserAgent(),
			}).Warn("Invalid authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtService.ValidateToken(tokenString, auth.AccessToken)
		if err != nil {
			log.WithError(err).WithFields(map[string]interface{}{
				"method": c.Request.Method,
				"path": c.Request.URL.Path,
				"user_agent": c.Request.UserAgent(),
			}).Warn("Invalid access token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Validate session
		var session models.UserSession
		if err := db.Preload("User.Tenant").Where("id = ? AND is_active = ?", claims.SessionID, true).First(&session).Error; err != nil {
			log.WithError(err).WithField("user_id", claims.UserID.String()).
				Warn("Invalid session")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
			c.Abort()
			return
		}

		// Check if session is expired
		if session.IsExpired() {
			log.WithField("user_id", claims.UserID.String()).WithField("session_id", claims.SessionID).
				Warn("Session expired")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			c.Abort()
			return
		}

		// Check if user is active
		if !session.User.IsActive() {
			log.WithField("user_id", claims.UserID.String()).
				Warn("User account is not active")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is not active"})
			c.Abort()
			return
		}

		// Check if tenant is active
		if !session.User.Tenant.IsActive() {
			log.WithField("tenant_id", claims.TenantID.String()).WithField("user_id", claims.UserID.String()).
				Warn("Tenant is not active")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is suspended"})
			c.Abort()
			return
		}

		// Set user context
		c.Set("user", &session.User)
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("user_role", claims.Role)
		c.Set("session_id", claims.SessionID)
		c.Set("claims", claims)

		log.WithField("tenant_id", claims.TenantID.String()).WithField("user_id", claims.UserID.String()).
			Debug("Request authenticated successfully")

		c.Next()
	}
}

// OptionalAuthMiddleware handles optional JWT authentication
func OptionalAuthMiddleware(jwtService *auth.JWTService, db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.Next()
			return
		}

		claims, err := jwtService.ValidateToken(tokenString, auth.AccessToken)
		if err != nil {
			c.Next()
			return
		}

		var session models.UserSession
		if err := db.Preload("User.Tenant").Where("id = ? AND is_active = ?", claims.SessionID, true).First(&session).Error; err != nil {
			c.Next()
			return
		}

		if session.IsExpired() || !session.User.IsActive() || !session.User.Tenant.IsActive() {
			c.Next()
			return
		}

		// Set user context if authentication is successful
		c.Set("user", &session.User)
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("user_role", claims.Role)
		c.Set("session_id", claims.SessionID)
		c.Set("claims", claims)

		c.Next()
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		role, ok := userRole.(models.UserRole)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role"})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range roles {
			if role == requiredRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin middleware checks if user is admin
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.UserRoleAdmin)
}

// RequireManagerOrAdmin middleware checks if user is manager or admin
func RequireManagerOrAdmin() gin.HandlerFunc {
	return RequireRole(models.UserRoleAdmin, models.UserRoleManager)
}

// TenantMiddleware adds tenant context to requests
func TenantMiddleware(db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get tenant ID from authenticated user first
		if tenantID, exists := c.Get("tenant_id"); exists {
			if tid, ok := tenantID.(uuid.UUID); ok {
				// Load tenant data
				var tenant models.Tenant
				if err := db.Where("id = ? AND status = ?", tid, models.TenantStatusActive).First(&tenant).Error; err == nil {
					c.Set("tenant", &tenant)
				}
			}
			c.Next()
			return
		}

		// Try to get tenant from subdomain or custom domain
		host := c.Request.Host
		if host == "" {
			c.Next()
			return
		}

		// Remove port if present
		if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
			host = host[:colonIndex]
		}

		// Try to find tenant by domain or extract from subdomain
		var tenant models.Tenant
		
		// First try custom domain
		if err := db.Where("domain = ? AND status = ?", host, models.TenantStatusActive).First(&tenant).Error; err == nil {
			c.Set("tenant", &tenant)
			c.Set("tenant_id", tenant.ID)
			c.Next()
			return
		}

		// Try subdomain (format: tenant.example.com)
		if strings.Contains(host, ".") {
			parts := strings.Split(host, ".")
			if len(parts) >= 2 {
				subdomain := parts[0]
				if subdomain != "www" && subdomain != "api" {
					if err := db.Where("slug = ? AND status = ?", subdomain, models.TenantStatusActive).First(&tenant).Error; err == nil {
						c.Set("tenant", &tenant)
						c.Set("tenant_id", tenant.ID)
						c.Next()
						return
					}
				}
			}
		}

		c.Next()
	}
}

// Helper functions to get context values

// GetCurrentUser returns the current authenticated user from context
func GetCurrentUser(c *gin.Context) (*models.User, error) {
	user, exists := c.Get("user")
	if !exists {
		return nil, errors.Unauthorized("User not authenticated", errors.ErrUnauthorized)
	}

	u, ok := user.(*models.User)
	if !ok {
		return nil, errors.InternalServer("Invalid user context", nil)
	}

	return u, nil
}

// GetCurrentUserID returns the current user ID from context
func GetCurrentUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.Unauthorized("User not authenticated", errors.ErrUnauthorized)
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.InternalServer("Invalid user ID context", nil)
	}

	return uid, nil
}

// GetCurrentTenantID returns the current tenant ID from context
func GetCurrentTenantID(c *gin.Context) (uuid.UUID, error) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, errors.Unauthorized("Tenant not found", errors.ErrInvalidTenant)
	}

	tid, ok := tenantID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.InternalServer("Invalid tenant ID context", nil)
	}

	return tid, nil
}

// GetCurrentTenant returns the current tenant from context
func GetCurrentTenant(c *gin.Context) (*models.Tenant, error) {
	tenant, exists := c.Get("tenant")
	if !exists {
		return nil, errors.Unauthorized("Tenant not found", errors.ErrTenantNotFound)
	}

	t, ok := tenant.(*models.Tenant)
	if !ok {
		return nil, errors.InternalServer("Invalid tenant context", nil)
	}

	return t, nil
}

// GetCurrentUserRole returns the current user role from context
func GetCurrentUserRole(c *gin.Context) (models.UserRole, error) {
	userRole, exists := c.Get("user_role")
	if !exists {
		return "", errors.Unauthorized("User role not found", errors.ErrUnauthorized)
	}

	role, ok := userRole.(models.UserRole)
	if !ok {
		return "", errors.InternalServer("Invalid user role context", nil)
	}

	return role, nil
}

// GetSessionID returns the current session ID from context
func GetSessionID(c *gin.Context) (uuid.UUID, error) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		return uuid.Nil, errors.Unauthorized("Session not found", errors.ErrUnauthorized)
	}

	sid, ok := sessionID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.InternalServer("Invalid session ID context", nil)
	}

	return sid, nil
}