package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/drazan344/taskflow-go/pkg/errors"
	"github.com/drazan344/taskflow-go/pkg/logger"
)

// ErrorHandler middleware handles panics and errors
func ErrorHandler(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				log.WithFields(map[string]interface{}{
					"panic": err,
					"stack": string(debug.Stack()),
					"method": c.Request.Method,
					"path": c.Request.URL.Path,
					"user_agent": c.Request.UserAgent(),
				}).Error("Panic recovered")

				// Return internal server error
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
				c.Abort()
			}
		}()

		c.Next()

		// Handle errors that were added to the context
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Handle custom application errors
			if appErr, ok := err.(*errors.AppError); ok {
				log.WithError(appErr).WithFields(map[string]interface{}{
					"code": appErr.Code,
					"method": c.Request.Method,
					"path": c.Request.URL.Path,
					"user_agent": c.Request.UserAgent(),
				}).Warn("Application error")

				response := gin.H{
					"error": appErr.Message,
				}

				if appErr.Details != "" {
					response["details"] = appErr.Details
				}

				c.JSON(appErr.Code, response)
				return
			}

			// Handle validation errors
			if validationErr, ok := err.(*errors.ValidationErrors); ok {
				log.WithError(validationErr).WithFields(map[string]interface{}{
					"validation_errors": validationErr.Errors,
					"method": c.Request.Method,
					"path": c.Request.URL.Path,
					"user_agent": c.Request.UserAgent(),
				}).Warn("Validation error")

				c.JSON(http.StatusBadRequest, gin.H{
					"error":             "Validation failed",
					"validation_errors": validationErr.Errors,
				})
				return
			}

			// Handle other errors
			log.WithError(err).WithFields(map[string]interface{}{
				"method": c.Request.Method,
				"path": c.Request.URL.Path,
				"user_agent": c.Request.UserAgent(),
			}).Error("Unhandled error")

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
		}
	}
}

// NotFoundHandler handles 404 errors
func NotFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Endpoint not found",
			"path":  c.Request.URL.Path,
		})
	}
}

// MethodNotAllowedHandler handles 405 errors
func MethodNotAllowedHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":  "Method not allowed",
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		})
	}
}

// HealthCheckMiddleware provides health check endpoint
func HealthCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"time":   gin.H{},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Prevent XSS attacks
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Enforce HTTPS (in production)
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		// Content Security Policy - allow inline scripts for docs pages
		path := c.Request.URL.Path
		if path == "/docs/index.html" || path == "/docs/" || path == "/docs" {
			// Relaxed CSP for documentation pages
			c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		} else {
			// Strict CSP for API endpoints
			c.Header("Content-Security-Policy", "default-src 'self'")
		}
		
		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Permissions Policy (formerly Feature Policy)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// TimeoutMiddleware adds request timeout handling
func TimeoutMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add timeout context if needed
		// This is a basic implementation - for production use gin-timeout or similar
		c.Next()
	}
}

// ValidationErrorResponse formats validation errors for API responses
func ValidationErrorResponse(err *errors.ValidationErrors) gin.H {
	return gin.H{
		"error":             "Validation failed",
		"validation_errors": err.Errors,
	}
}

// ErrorResponse creates a standard error response
func ErrorResponse(message string, details ...string) gin.H {
	response := gin.H{
		"error": message,
	}
	
	if len(details) > 0 && details[0] != "" {
		response["details"] = details[0]
	}
	
	return response
}

// SuccessResponse creates a standard success response
func SuccessResponse(data interface{}, message ...string) gin.H {
	response := gin.H{
		"data": data,
	}
	
	if len(message) > 0 && message[0] != "" {
		response["message"] = message[0]
	}
	
	return response
}

// PaginationResponse creates a paginated response
func PaginationResponse(data interface{}, page, perPage, total int) gin.H {
	totalPages := (total + perPage - 1) / perPage
	
	return gin.H{
		"data": data,
		"pagination": gin.H{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	}
}