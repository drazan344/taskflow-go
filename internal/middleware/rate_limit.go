package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/drazan344/taskflow-go/internal/database"
	"github.com/drazan344/taskflow-go/pkg/logger"
)

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Requests int           // Number of requests allowed
	Window   time.Duration // Time window for the limit
	KeyFunc  KeyFunc       // Function to generate rate limit key
}

// KeyFunc generates a rate limit key for the request
type KeyFunc func(c *gin.Context) string

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Requests: 100,
		Window:   time.Minute,
		KeyFunc:  IPKeyFunc,
	}
}

// TenantRateLimitConfig returns tenant-based rate limiting configuration
func TenantRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Requests: 1000,
		Window:   time.Minute,
		KeyFunc:  TenantKeyFunc,
	}
}

// UserRateLimitConfig returns user-based rate limiting configuration
func UserRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Requests: 200,
		Window:   time.Minute,
		KeyFunc:  UserKeyFunc,
	}
}

// RateLimitMiddleware provides rate limiting functionality
func RateLimitMiddleware(redis *database.Redis, config *RateLimitConfig, log *logger.Logger) gin.HandlerFunc {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	return func(c *gin.Context) {
		// Generate rate limit key
		key := config.KeyFunc(c)
		if key == "" {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create rate limit key with window
		rateLimitKey := fmt.Sprintf("rate_limit:%s", key)
		windowKey := fmt.Sprintf("%s:%d", rateLimitKey, time.Now().Unix()/int64(config.Window.Seconds()))

		// Get current count
		countStr, err := redis.Get(ctx, windowKey)
		if err != nil && err.Error() != "redis: nil" {
			log.WithError(err).Error("Failed to get rate limit count from Redis")
			// Allow request if Redis is unavailable
			c.Next()
			return
		}

		var count int64
		if countStr != "" {
			count, _ = strconv.ParseInt(countStr, 10, 64)
		}

		// Check if limit exceeded
		if count >= int64(config.Requests) {
			// Set rate limit headers
			c.Header("X-RateLimit-Limit", strconv.Itoa(config.Requests))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(config.Window).Unix(), 10))

			log.WithField("rate_limit_key", key).
				WithField("current_count", count).
				WithField("limit", config.Requests).
				Warn("Rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Limit: %d per %v", config.Requests, config.Window),
			})
			c.Abort()
			return
		}

		// Increment counter
		newCount, err := redis.Client.Incr(ctx, windowKey).Result()
		if err != nil {
			log.WithError(err).Error("Failed to increment rate limit counter")
			// Allow request if Redis is unavailable
			c.Next()
			return
		}

		// Set expiration on first increment
		if newCount == 1 {
			redis.Client.Expire(ctx, windowKey, config.Window)
		}

		// Set rate limit headers
		remaining := config.Requests - int(newCount)
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(config.Requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(config.Window).Unix(), 10))

		c.Next()
	}
}

// Key generation functions

// IPKeyFunc generates a rate limit key based on client IP
func IPKeyFunc(c *gin.Context) string {
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// TenantKeyFunc generates a rate limit key based on tenant ID
func TenantKeyFunc(c *gin.Context) string {
	if tenantID, exists := c.Get("tenant_id"); exists {
		return fmt.Sprintf("tenant:%v", tenantID)
	}
	// Fallback to IP if no tenant context
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// UserKeyFunc generates a rate limit key based on user ID
func UserKeyFunc(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%v", userID)
	}
	// Fallback to tenant or IP if no user context
	return TenantKeyFunc(c)
}

// APIKeyFunc generates a rate limit key based on API key (if implemented)
func APIKeyFunc(c *gin.Context) string {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" {
		return fmt.Sprintf("api_key:%s", apiKey)
	}
	// Fallback to other methods
	return TenantKeyFunc(c)
}

// CompositeKeyFunc generates a composite rate limit key
func CompositeKeyFunc(funcs ...KeyFunc) KeyFunc {
	return func(c *gin.Context) string {
		keys := make([]string, 0, len(funcs))
		for _, fn := range funcs {
			if key := fn(c); key != "" {
				keys = append(keys, key)
			}
		}
		if len(keys) == 0 {
			return ""
		}
		return fmt.Sprintf("composite:%s", keys[0]) // Use first available key
	}
}

// Different rate limiting strategies

// SlidingWindowRateLimit implements sliding window rate limiting
func SlidingWindowRateLimit(redis *database.Redis, key string, limit int, window time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	windowStart := now.Add(-window)

	// Remove expired entries
	redis.Client.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.UnixNano(), 10))

	// Count current entries
	count, err := redis.Client.ZCard(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count >= int64(limit) {
		return false, nil // Rate limit exceeded
	}

	// Add current request
	redis.Client.ZAdd(ctx, key, struct {
		Score  float64
		Member interface{}
	}{
		Score:  float64(now.UnixNano()),
		Member: now.UnixNano(),
	})

	// Set expiration
	redis.Client.Expire(ctx, key, window)

	return true, nil
}

// TokenBucketRateLimit implements token bucket rate limiting
func TokenBucketRateLimit(redis *database.Redis, key string, capacity, refillRate int, refillPeriod time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	bucketKey := fmt.Sprintf("bucket:%s", key)

	// Get current bucket state
	pipe := redis.Client.TxPipeline()
	tokensCmd := pipe.HGet(ctx, bucketKey, "tokens")
	lastRefillCmd := pipe.HGet(ctx, bucketKey, "last_refill")
	_, err := pipe.Exec(ctx)

	var tokens int
	var lastRefill time.Time

	if err == nil {
		if tokensStr := tokensCmd.Val(); tokensStr != "" {
			tokens, _ = strconv.Atoi(tokensStr)
		} else {
			tokens = capacity
		}

		if lastRefillStr := lastRefillCmd.Val(); lastRefillStr != "" {
			if timestamp, err := strconv.ParseInt(lastRefillStr, 10, 64); err == nil {
				lastRefill = time.Unix(timestamp, 0)
			}
		} else {
			lastRefill = now
		}
	} else {
		// Initialize bucket
		tokens = capacity
		lastRefill = now
	}

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(lastRefill)
	if elapsed > 0 {
		tokensToAdd := int(elapsed / refillPeriod * time.Duration(refillRate))
		tokens = min(capacity, tokens+tokensToAdd)
		lastRefill = now
	}

	// Check if request can be processed
	if tokens <= 0 {
		return false, nil
	}

	// Consume a token
	tokens--

	// Update bucket state
	redis.Client.HSet(ctx, bucketKey, map[string]interface{}{
		"tokens":      tokens,
		"last_refill": lastRefill.Unix(),
	})
	redis.Client.Expire(ctx, bucketKey, time.Hour) // Expire bucket after inactivity

	return true, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}