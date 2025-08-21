package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/drazan344/taskflow-go/internal/auth"
	"github.com/drazan344/taskflow-go/internal/config"
	"github.com/drazan344/taskflow-go/internal/database"
	"github.com/drazan344/taskflow-go/internal/handlers"
	"github.com/drazan344/taskflow-go/internal/middleware"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/internal/websocket"
	"github.com/drazan344/taskflow-go/pkg/logger"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"
)

// @title TaskFlow API
// @version 1.0
// @description A multi-tenant task management platform API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.taskflow.com/support
// @contact.email support@taskflow.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the token with the `Bearer ` prefix, e.g. "Bearer abcde12345"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize logger
	logger := logger.New(cfg.Log.Level, cfg.Log.Format)

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Initialize database
	db, err := database.Connect(cfg)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Initialize Redis
	redis, err := database.ConnectRedis(cfg)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to Redis")
	}
	defer redis.Close()

	// Run database migrations
	if err := runMigrations(db); err != nil {
		logger.WithError(err).Fatal("Failed to run database migrations")
	}

	// Initialize services
	jwtService := auth.NewJWTService(cfg)
	authService := auth.NewService(db.DB, cfg)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(logger)
	go wsHub.Run() // Start the hub in a separate goroutine

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, logger)
	userHandler := handlers.NewUserHandler(db.DB, logger)
	taskHandler := handlers.NewTaskHandler(db.DB, logger)
	tenantHandler := handlers.NewTenantHandler(db.DB, logger)
	wsHandler := handlers.NewWebSocketHandler(wsHub, logger)

	// Setup routes
	router := setupRoutes(cfg, db, redis, jwtService, authHandler, userHandler, taskHandler, tenantHandler, wsHandler, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:    cfg.GetServerAddr(),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("addr", server.Addr).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Fatal("Server forced to shutdown")
	}

	logger.Info("Server exited")
}

func setupRoutes(
	cfg *config.Config,
	db *database.DB,
	redis *database.Redis,
	jwtService *auth.JWTService,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	taskHandler *handlers.TaskHandler,
	tenantHandler *handlers.TenantHandler,
	wsHandler *handlers.WebSocketHandler,
	logger *logger.Logger,
) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.LoggerMiddleware(logger))
	router.Use(middleware.ErrorHandler(logger))
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.CORSMiddleware(middleware.DevelopmentCORSConfig()))
	router.Use(middleware.TenantMiddleware(db.DB, logger))

	// Global rate limiting
	router.Use(middleware.RateLimitMiddleware(redis, middleware.DefaultRateLimitConfig(), logger))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		// Check database health
		if err := db.Health(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unhealthy",
				"database": "down",
				"error":    err.Error(),
			})
			return
		}

		// Check Redis health
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := redis.Health(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"redis":  "down",
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"database":  "up",
			"redis":     "up",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Swagger documentation
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	v1 := router.Group("/api/v1")

	// Public routes (no authentication required)
	public := v1.Group("")
	{
		// Authentication routes
		auth := public.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshTokens)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}

		// Tenant invitation routes (public but require valid token)
		public.POST("/invitations/:token/accept", tenantHandler.AcceptInvitation)
	}

	// Protected routes (authentication required)
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(jwtService, db.DB, logger))
	{
		// User-specific rate limiting for authenticated routes
		protected.Use(middleware.RateLimitMiddleware(redis, middleware.UserRateLimitConfig(), logger))

		// Authentication management
		auth := protected.Group("/auth")
		{
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/me", authHandler.GetCurrentUser)
			auth.PUT("/me", authHandler.UpdateProfile)
			auth.POST("/change-password", authHandler.ChangePassword)
		}

		// User management
		users := protected.Group("/users")
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", middleware.RequireManagerOrAdmin(), userHandler.UpdateUser)
			users.DELETE("/:id", middleware.RequireAdmin(), userHandler.DeleteUser)
		}

		// Task management
		tasks := protected.Group("/tasks")
		{
			tasks.GET("", taskHandler.ListTasks)
			tasks.POST("", taskHandler.CreateTask)
			tasks.GET("/:id", taskHandler.GetTask)
			tasks.PUT("/:id", taskHandler.UpdateTask)
			tasks.DELETE("/:id", taskHandler.DeleteTask)
			tasks.POST("/:id/comments", taskHandler.AddComment)
			tasks.GET("/:id/comments", taskHandler.ListComments)
			tasks.POST("/:id/attachments", taskHandler.AddAttachment)
			tasks.GET("/:id/attachments", taskHandler.ListAttachments)
			tasks.DELETE("/attachments/:attachment_id", taskHandler.DeleteAttachment)
		}

		// Project management
		projects := protected.Group("/projects")
		{
			projects.GET("", taskHandler.ListProjects)
			projects.POST("", middleware.RequireManagerOrAdmin(), taskHandler.CreateProject)
			projects.GET("/:id", taskHandler.GetProject)
			projects.PUT("/:id", middleware.RequireManagerOrAdmin(), taskHandler.UpdateProject)
			projects.DELETE("/:id", middleware.RequireManagerOrAdmin(), taskHandler.DeleteProject)
		}

		// Tag management
		tags := protected.Group("/tags")
		{
			tags.GET("", taskHandler.ListTags)
			tags.POST("", taskHandler.CreateTag)
			tags.GET("/:id", taskHandler.GetTag)
			tags.PUT("/:id", taskHandler.UpdateTag)
			tags.DELETE("/:id", taskHandler.DeleteTag)
		}

		// Tenant management (admin only)
		tenant := protected.Group("/tenant")
		tenant.Use(middleware.RequireAdmin())
		{
			tenant.GET("", tenantHandler.GetTenant)
			tenant.PUT("", tenantHandler.UpdateTenant)
			tenant.POST("/invitations", tenantHandler.CreateInvitation)
			tenant.GET("/invitations", tenantHandler.ListInvitations)
			tenant.DELETE("/invitations/:id", tenantHandler.CancelInvitation)
			tenant.GET("/usage", tenantHandler.GetUsage)
			tenant.GET("/analytics", tenantHandler.GetAnalytics)
		}

		// WebSocket routes
		ws := protected.Group("/ws")
		{
			ws.GET("", wsHandler.HandleConnection())
			ws.GET("/online-users", wsHandler.GetOnlineUsers)
		}
	}

	// Handle 404
	router.NoRoute(middleware.NotFoundHandler())

	// Handle 405
	router.NoMethod(middleware.MethodNotAllowedHandler())

	return router
}

func runMigrations(db *database.DB) error {
	models := []interface{}{
		&models.Tenant{},
		&models.User{},
		&models.UserSession{},
		&models.Task{},
	}

	return db.Migrate(models...)
}