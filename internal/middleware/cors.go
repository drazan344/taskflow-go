package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"X-Requested-With",
			"X-Request-ID",
			"X-Tenant-ID",
		},
		ExposeHeaders: []string{
			"X-Request-ID",
			"X-Total-Count",
			"X-Page",
			"X-Per-Page",
		},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60, // 12 hours
	}
}

// DevelopmentCORSConfig returns a CORS configuration suitable for development
func DevelopmentCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
			"http://127.0.0.1:8080",
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"X-Requested-With",
			"X-Request-ID",
			"X-Tenant-ID",
		},
		ExposeHeaders: []string{
			"X-Request-ID",
			"X-Total-Count",
			"X-Page",
			"X-Per-Page",
		},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60, // 12 hours
	}
}

// CORSMiddleware provides CORS support
func CORSMiddleware(config *CORSConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCORSConfig()
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		requestMethod := c.Request.Header.Get("Access-Control-Request-Method")
		requestHeaders := c.Request.Header.Get("Access-Control-Request-Headers")

		// Check if origin is allowed
		if origin != "" && (contains(config.AllowOrigins, "*") || contains(config.AllowOrigins, origin)) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		// Set allowed methods
		if len(config.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
		}

		// Set allowed headers
		if len(config.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
		}

		// Set exposed headers
		if len(config.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
		}

		// Set credentials
		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Set max age
		if config.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", string(rune(config.MaxAge)))
		}

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			// Validate preflight request
			if requestMethod != "" && !contains(config.AllowMethods, requestMethod) {
				c.AbortWithStatus(http.StatusMethodNotAllowed)
				return
			}

			if requestHeaders != "" {
				requestedHeaders := strings.Split(requestHeaders, ",")
				for _, header := range requestedHeaders {
					header = strings.TrimSpace(header)
					if !contains(config.AllowHeaders, header) && !contains(config.AllowHeaders, "*") {
						c.AbortWithStatus(http.StatusForbidden)
						return
					}
				}
			}

			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}