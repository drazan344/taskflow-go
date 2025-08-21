package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/drazan344/taskflow-go/pkg/logger"
	"github.com/sirupsen/logrus"
)

// LoggerMiddleware provides structured logging for HTTP requests
func LoggerMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		userAgent := c.Request.UserAgent()
		clientIP := c.ClientIP()

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		responseSize := c.Writer.Size()

		// Get user and tenant info from context if available
		fields := logrus.Fields{
			"method":        method,
			"path":          path,
			"status_code":   statusCode,
			"duration_ms":   duration.Milliseconds(),
			"response_size": responseSize,
			"client_ip":     clientIP,
			"user_agent":    userAgent,
		}

		// Add user context if authenticated
		if userID, exists := c.Get("user_id"); exists {
			fields["user_id"] = userID
		}

		if tenantID, exists := c.Get("tenant_id"); exists {
			fields["tenant_id"] = tenantID
		}

		if userRole, exists := c.Get("user_role"); exists {
			fields["user_role"] = userRole
		}

		// Add error information if present
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// Log based on status code
		entry := log.WithFields(fields)
		
		switch {
		case statusCode >= 500:
			entry.Error("Internal server error")
		case statusCode >= 400:
			entry.Warn("Client error")
		case statusCode >= 300:
			entry.Info("Redirect")
		default:
			entry.Info("Request completed")
		}
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	})
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation - in production, consider using a UUID or similar
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}