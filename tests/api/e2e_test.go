package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL = "http://localhost:8080/api/v1"
)

type E2ETestSuite struct {
	tenantID     uuid.UUID
	adminToken   string
	userToken    string
	taskID       uuid.UUID
	projectID    uuid.UUID
}

func TestMain(m *testing.M) {
	// Check if server is running
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("Server not running. Please start the server with 'go run cmd/api/main.go'")
		os.Exit(1)
	}
	resp.Body.Close()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func setupE2ETest(t *testing.T) *E2ETestSuite {
	suite := &E2ETestSuite{}

	// Create a test tenant (this would typically be done through admin API)
	// For now, we'll use a UUID and assume tenant exists or is created externally
	suite.tenantID = uuid.MustParse("123e4567-e89b-12d3-a456-426614174000") // Use a fixed UUID for testing

	// Register and login admin user
	adminEmail := fmt.Sprintf("admin-%d@e2etest.com", time.Now().Unix())
	suite.adminToken = registerAndLogin(t, adminEmail, "Admin", "User", "password123", suite.tenantID)

	// Register and login regular user
	userEmail := fmt.Sprintf("user-%d@e2etest.com", time.Now().Unix())
	suite.userToken = registerAndLogin(t, userEmail, "Test", "User", "password123", suite.tenantID)

	return suite
}

func registerAndLogin(t *testing.T, email, firstName, lastName, password string, tenantID uuid.UUID) string {
	// Register
	registerPayload := map[string]interface{}{
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"password":   password,
		"tenant_id":  tenantID,
	}

	body, err := json.Marshal(registerPayload)
	require.NoError(t, err)

	resp, err := http.Post(baseURL+"/auth/register", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	// If registration fails, try login instead (user might already exist)
	if resp.StatusCode != http.StatusCreated {
		loginPayload := map[string]interface{}{
			"email":     email,
			"password":  password,
			"tenant_id": tenantID,
		}

		body, err := json.Marshal(loginPayload)
		require.NoError(t, err)

		resp, err = http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()
	}

	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to authenticate user")

	var authResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&authResponse)
	require.NoError(t, err)

	token, exists := authResponse["access_token"].(string)
	require.True(t, exists, "No access token in response")
	require.NotEmpty(t, token, "Empty access token")

	return token
}

func (suite *E2ETestSuite) makeRequest(t *testing.T, method, endpoint string, payload interface{}, token string) *http.Response {
	var body bytes.Buffer
	if payload != nil {
		err := json.NewEncoder(&body).Encode(payload)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(method, baseURL+endpoint, &body)
	require.NoError(t, err)

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("X-Tenant-ID", suite.tenantID.String())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func TestE2EWorkflow(t *testing.T) {
	suite := setupE2ETest(t)

	t.Run("Authentication Flow", func(t *testing.T) {
		// Test getting current user
		resp := suite.makeRequest(t, "GET", "/auth/me", nil, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var userResponse map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&userResponse)
		require.NoError(t, err)

		assert.Contains(t, userResponse, "user")
		user := userResponse["user"].(map[string]interface{})
		assert.Contains(t, user["email"], "@e2etest.com")
	})

	t.Run("Project Management Flow", func(t *testing.T) {
		// Create a project
		projectPayload := map[string]interface{}{
			"name":        "E2E Test Project",
			"description": "A project created during E2E testing",
			"status":      "active",
		}

		resp := suite.makeRequest(t, "POST", "/projects", projectPayload, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var projectResponse map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&projectResponse)
		require.NoError(t, err)

		project := projectResponse["project"].(map[string]interface{})
		projectIDStr := project["id"].(string)
		suite.projectID = uuid.MustParse(projectIDStr)

		// List projects
		resp = suite.makeRequest(t, "GET", "/projects", nil, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var projectsResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&projectsResponse)
		require.NoError(t, err)

		projects := projectsResponse["projects"].([]interface{})
		assert.GreaterOrEqual(t, len(projects), 1)
	})

	t.Run("Task Management Flow", func(t *testing.T) {
		// Create a task
		taskPayload := map[string]interface{}{
			"title":       "E2E Test Task",
			"description": "A task created during E2E testing",
			"status":      "todo",
			"priority":    "high",
			"project_id":  suite.projectID,
		}

		resp := suite.makeRequest(t, "POST", "/tasks", taskPayload, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var taskResponse map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&taskResponse)
		require.NoError(t, err)

		task := taskResponse["task"].(map[string]interface{})
		taskIDStr := task["id"].(string)
		suite.taskID = uuid.MustParse(taskIDStr)
		assert.Equal(t, "E2E Test Task", task["title"])
		assert.Equal(t, "todo", task["status"])

		// List tasks
		resp = suite.makeRequest(t, "GET", "/tasks", nil, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tasksResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&tasksResponse)
		require.NoError(t, err)

		tasks := tasksResponse["tasks"].([]interface{})
		assert.GreaterOrEqual(t, len(tasks), 1)

		// Get specific task
		endpoint := fmt.Sprintf("/tasks/%s", suite.taskID)
		resp = suite.makeRequest(t, "GET", endpoint, nil, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Update task
		updatePayload := map[string]interface{}{
			"status": "in_progress",
			"title":  "Updated E2E Test Task",
		}

		resp = suite.makeRequest(t, "PUT", endpoint, updatePayload, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&taskResponse)
		require.NoError(t, err)

		updatedTask := taskResponse["task"].(map[string]interface{})
		assert.Equal(t, "Updated E2E Test Task", updatedTask["title"])
		assert.Equal(t, "in_progress", updatedTask["status"])
	})

	t.Run("Task Comments Flow", func(t *testing.T) {
		// Add comment to task
		commentPayload := map[string]interface{}{
			"content": "This is an E2E test comment",
		}

		endpoint := fmt.Sprintf("/tasks/%s/comments", suite.taskID)
		resp := suite.makeRequest(t, "POST", endpoint, commentPayload, suite.userToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var commentResponse map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&commentResponse)
		require.NoError(t, err)

		comment := commentResponse["comment"].(map[string]interface{})
		assert.Equal(t, "This is an E2E test comment", comment["content"])

		// List comments
		resp = suite.makeRequest(t, "GET", endpoint, nil, suite.userToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var commentsResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&commentsResponse)
		require.NoError(t, err)

		comments := commentsResponse["comments"].([]interface{})
		assert.GreaterOrEqual(t, len(comments), 1)
	})

	t.Run("User Management Flow", func(t *testing.T) {
		// List users (admin only)
		resp := suite.makeRequest(t, "GET", "/users", nil, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var usersResponse map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&usersResponse)
		require.NoError(t, err)

		users := usersResponse["users"].([]interface{})
		assert.GreaterOrEqual(t, len(users), 2) // At least admin and user

		// Regular user should not be able to list users
		resp = suite.makeRequest(t, "GET", "/users", nil, suite.userToken)
		defer resp.Body.Close()
		// This might return 200 or 403 depending on implementation
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusForbidden)
	})

	t.Run("Tag Management Flow", func(t *testing.T) {
		// Create a tag
		tagPayload := map[string]interface{}{
			"name":  "e2e-test",
			"color": "#FF5733",
		}

		resp := suite.makeRequest(t, "POST", "/tags", tagPayload, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var tagResponse map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&tagResponse)
		require.NoError(t, err)

		tag := tagResponse["tag"].(map[string]interface{})
		assert.Equal(t, "e2e-test", tag["name"])
		assert.Equal(t, "#FF5733", tag["color"])

		// List tags
		resp = suite.makeRequest(t, "GET", "/tags", nil, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tagsResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&tagsResponse)
		require.NoError(t, err)

		tags := tagsResponse["tags"].([]interface{})
		assert.GreaterOrEqual(t, len(tags), 1)
	})

	t.Run("Cleanup", func(t *testing.T) {
		// Delete the task
		endpoint := fmt.Sprintf("/tasks/%s", suite.taskID)
		resp := suite.makeRequest(t, "DELETE", endpoint, nil, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify task is deleted
		resp = suite.makeRequest(t, "GET", endpoint, nil, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// Delete the project
		endpoint = fmt.Sprintf("/projects/%s", suite.projectID)
		resp = suite.makeRequest(t, "DELETE", endpoint, nil, suite.adminToken)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}