package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"


	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drazan344/taskflow-go/internal/auth"
	"github.com/drazan344/taskflow-go/internal/config"
	"github.com/drazan344/taskflow-go/internal/handlers"
	"github.com/drazan344/taskflow-go/internal/middleware"
	"github.com/drazan344/taskflow-go/pkg/logger"
	"github.com/drazan344/taskflow-go/tests/fixtures"
)

type TestServer struct {
	router      *gin.Engine
	db          *fixtures.TestDB
	authService *auth.Service
	jwtService  *auth.JWTService
	testData    *fixtures.TestData
	logger      *logger.Logger
}

func setupTestServer(t *testing.T) *TestServer {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test database
	db, err := fixtures.NewTestDB()
	require.NoError(t, err)

	// Seed test data
	testData, err := db.SeedTestData()
	require.NoError(t, err)

	// Create config
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:               "test-secret-key-for-testing-only",
			AccessTokenDuration:  "15m",
			RefreshTokenDuration: "7d",
		},
	}

	// Create logger
	logger := &logger.Logger{}

	// Create services
	jwtService := auth.NewJWTService(cfg)
	authService := auth.NewService(db.DB, cfg)

	// Create handlers
	authHandler := handlers.NewAuthHandler(authService, logger)
	userHandler := handlers.NewUserHandler(db.DB, logger)
	taskHandler := handlers.NewTaskHandler(db.DB, logger)

	// Create router
	router := gin.New()

	// Add middleware
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.ErrorHandler(logger))

	// API routes
	v1 := router.Group("/api/v1")

	// Public routes
	public := v1.Group("")
	{
		auth := public.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshTokens)
		}
	}

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(jwtService, db.DB, logger))
	{
		auth := protected.Group("/auth")
		{
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/me", authHandler.GetCurrentUser)
		}

		users := protected.Group("/users")
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
		}

		tasks := protected.Group("/tasks")
		{
			tasks.GET("", taskHandler.ListTasks)
			tasks.POST("", taskHandler.CreateTask)
			tasks.GET("/:id", taskHandler.GetTask)
			tasks.PUT("/:id", taskHandler.UpdateTask)
			tasks.DELETE("/:id", taskHandler.DeleteTask)
		}
	}

	return &TestServer{
		router:      router,
		db:          db,
		authService: authService,
		jwtService:  jwtService,
		testData:    testData,
		logger:      logger,
	}
}

func (ts *TestServer) Close() {
	ts.db.Close()
}

func TestAuthenticationAPI(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	t.Run("Register user", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":      "newuser@test.com",
			"first_name": "New",
			"last_name":  "User",
			"password":   "password123",
			"tenant_id":  ts.testData.Tenant.ID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "access_token")
		assert.Contains(t, response, "refresh_token")
		assert.Contains(t, response, "user")

		user := response["user"].(map[string]interface{})
		assert.Equal(t, "newuser@test.com", user["email"])
		assert.Equal(t, "New", user["first_name"])
	})

	t.Run("Login user", func(t *testing.T) {
		// First register a user
		registerPayload := map[string]interface{}{
			"email":      "logintest@test.com",
			"first_name": "Login",
			"last_name":  "Test",
			"password":   "password123",
			"tenant_id":  ts.testData.Tenant.ID,
		}

		body, err := json.Marshal(registerPayload)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		ts.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Now login
		loginPayload := map[string]interface{}{
			"email":     "logintest@test.com",
			"password":  "password123",
			"tenant_id": ts.testData.Tenant.ID,
		}

		body, err = json.Marshal(loginPayload)
		require.NoError(t, err)

		req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "access_token")
		assert.Contains(t, response, "refresh_token")
		assert.Contains(t, response, "user")
	})

	t.Run("Login with invalid credentials", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":     "invalid@test.com",
			"password":  "wrongpassword",
			"tenant_id": ts.testData.Tenant.ID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Get current user with valid token", func(t *testing.T) {
		// Generate token for test user
		token, err := ts.jwtService.GenerateAccessToken(
			ts.testData.User.ID,
			ts.testData.User.TenantID,
			ts.testData.User.Role,
		)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Tenant-ID", ts.testData.Tenant.ID.String())
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "user")
		user := response["user"].(map[string]interface{})
		assert.Equal(t, ts.testData.User.Email, user["email"])
	})

	t.Run("Access protected route without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestTasksAPI(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Generate token for test user
	token, err := ts.jwtService.GenerateAccessToken(
		ts.testData.User.ID,
		ts.testData.User.TenantID,
		ts.testData.User.Role,
	)
	require.NoError(t, err)

	authHeader := "Bearer " + token
	tenantHeader := ts.testData.Tenant.ID.String()

	t.Run("List tasks", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("X-Tenant-ID", tenantHeader)
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "tasks")
		tasks := response["tasks"].([]interface{})
		assert.Len(t, tasks, 1) // We have one test task
	})

	t.Run("Get specific task", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/tasks/%s", ts.testData.Task.ID)
		req := httptest.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("X-Tenant-ID", tenantHeader)
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "task")
		task := response["task"].(map[string]interface{})
		assert.Equal(t, ts.testData.Task.Title, task["title"])
	})

	t.Run("Create new task", func(t *testing.T) {
		payload := map[string]interface{}{
			"title":       "New Test Task",
			"description": "A new task created via API",
			"status":      "todo",
			"priority":    "medium",
			"project_id":  ts.testData.Project.ID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("X-Tenant-ID", tenantHeader)
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "task")
		task := response["task"].(map[string]interface{})
		assert.Equal(t, "New Test Task", task["title"])
		assert.Equal(t, "todo", task["status"])
	})

	t.Run("Update existing task", func(t *testing.T) {
		payload := map[string]interface{}{
			"title":  "Updated Test Task",
			"status": "in_progress",
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		url := fmt.Sprintf("/api/v1/tasks/%s", ts.testData.Task.ID)
		req := httptest.NewRequest("PUT", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("X-Tenant-ID", tenantHeader)
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "task")
		task := response["task"].(map[string]interface{})
		assert.Equal(t, "Updated Test Task", task["title"])
		assert.Equal(t, "in_progress", task["status"])
	})
}

func TestUsersAPI(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Generate token for admin user
	token, err := ts.jwtService.GenerateAccessToken(
		ts.testData.Admin.ID,
		ts.testData.Admin.TenantID,
		ts.testData.Admin.Role,
	)
	require.NoError(t, err)

	authHeader := "Bearer " + token
	tenantHeader := ts.testData.Tenant.ID.String()

	t.Run("List users", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("X-Tenant-ID", tenantHeader)
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "users")
		users := response["users"].([]interface{})
		assert.Len(t, users, 2) // We have admin and regular user
	})

	t.Run("Get specific user", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/users/%s", ts.testData.User.ID)
		req := httptest.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("X-Tenant-ID", tenantHeader)
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "user")
		user := response["user"].(map[string]interface{})
		assert.Equal(t, ts.testData.User.Email, user["email"])
	})
}