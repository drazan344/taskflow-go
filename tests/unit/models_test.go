package unit

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/tests/fixtures"
)

func TestUserModel(t *testing.T) {
	db, err := fixtures.NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	data, err := db.SeedTestData()
	require.NoError(t, err)

	t.Run("User creation", func(t *testing.T) {
		user := &models.User{
			TenantModel:   models.TenantModel{TenantID: data.Tenant.ID},
			Email:         "newuser@test.com",
			FirstName:     "New",
			LastName:      "User",
			Role:          models.UserRoleUser,
			Status:        models.UserStatusActive,
			EmailVerified: false,
		}

		err := db.Create(user).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.NotZero(t, user.CreatedAt)
		assert.Equal(t, data.Tenant.ID, user.TenantID)
	})

	t.Run("User relationships", func(t *testing.T) {
		var user models.User
		err := db.Preload("Tenant").First(&user, "email = ?", data.User.Email).Error
		assert.NoError(t, err)
		assert.Equal(t, data.Tenant.Name, user.Tenant.Name)
	})

	t.Run("User methods", func(t *testing.T) {
		user := &models.User{Role: models.UserRoleAdmin}
		assert.True(t, user.IsAdmin())
		assert.True(t, user.CanManage())

		user.Role = models.UserRoleManager
		assert.False(t, user.IsAdmin())
		assert.True(t, user.CanManage())

		user.Role = models.UserRoleUser
		assert.False(t, user.IsAdmin())
		assert.False(t, user.CanManage())
	})

	t.Run("User preferences", func(t *testing.T) {
		user := data.User
		assert.Equal(t, "dark", user.Preferences.Theme)
		assert.Equal(t, "America/New_York", user.Timezone)
		assert.False(t, user.Preferences.WeeklyDigest)
	})
}

func TestTaskModel(t *testing.T) {
	db, err := fixtures.NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	data, err := db.SeedTestData()
	require.NoError(t, err)

	t.Run("Task creation", func(t *testing.T) {
		task := &models.Task{
			TenantModel: models.TenantModel{TenantID: data.Tenant.ID},
			Title:       "New Task",
			Description: "A new task for testing",
			Status:      models.TaskStatusTodo,
			Priority:    models.TaskPriorityMedium,
			CreatorID:   data.Admin.ID,
		}

		err := db.Create(task).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, task.ID)
		assert.NotZero(t, task.CreatedAt)
	})

	t.Run("Task relationships", func(t *testing.T) {
		var task models.Task
		err := db.Preload("Project").Preload("Assignee").Preload("Creator").
			First(&task, data.Task.ID).Error
		assert.NoError(t, err)

		assert.Equal(t, data.Project.Name, task.Project.Name)
		assert.Equal(t, data.User.Email, task.Assignee.Email)
		assert.Equal(t, data.Admin.Email, task.Creator.Email)
	})

	t.Run("Task methods", func(t *testing.T) {
		task := &models.Task{Status: models.TaskStatusTodo}
		assert.False(t, task.IsCompleted())

		task.Status = models.TaskStatusCompleted
		assert.True(t, task.IsCompleted())

		task.Status = models.TaskStatusInProgress
		assert.False(t, task.IsCompleted())
	})

	t.Run("Task due date", func(t *testing.T) {
		task := data.Task
		assert.NotNil(t, task.DueDate)
		assert.True(t, task.DueDate.After(time.Now()))
	})
}

func TestTenantModel(t *testing.T) {
	db, err := fixtures.NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	t.Run("Tenant creation", func(t *testing.T) {
		tenant := &models.Tenant{
			Name:       "New Tenant",
			Slug:       "new-tenant",
			Domain:     "new.example.com",
			Status:     models.TenantStatusActive,
			Plan:       models.TenantPlanBasic,
			MaxUsers:   50,
			MaxTasks:   500,
			MaxStorage: 500 * 1024 * 1024, // 500MB
		}

		err := db.Create(tenant).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, tenant.ID)
		assert.Equal(t, models.TenantStatusActive, tenant.Status)
	})

	t.Run("Tenant methods", func(t *testing.T) {
		tenant := &models.Tenant{Status: models.TenantStatusActive}
		assert.True(t, tenant.IsActive())

		tenant.Status = models.TenantStatusSuspended
		assert.False(t, tenant.IsActive())

		tenant.Status = models.TenantStatusCanceled
		assert.False(t, tenant.IsActive())
	})
}

func TestNotificationModel(t *testing.T) {
	db, err := fixtures.NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	data, err := db.SeedTestData()
	require.NoError(t, err)

	t.Run("Notification creation", func(t *testing.T) {
		notification := &models.Notification{
			TenantModel: models.TenantModel{TenantID: data.Tenant.ID},
			UserID:      data.User.ID,
			Type:        models.NotificationTypeTaskAssigned,
			Status:      models.NotificationStatusUnread,
			Title:       "Task Assigned",
			Message:     "A new task has been assigned to you",
			Data: models.NotificationData{
				ActorID:    &data.Admin.ID,
				ActorName:  data.Admin.FirstName + " " + data.Admin.LastName,
				EntityType: "task",
				EntityID:   data.Task.ID.String(),
			},
		}

		err := db.Create(notification).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, notification.ID)
		assert.Equal(t, models.NotificationStatusUnread, notification.Status)
	})

	t.Run("Notification methods", func(t *testing.T) {
		notification := &models.Notification{Status: models.NotificationStatusUnread}
		assert.False(t, notification.IsRead())
		assert.False(t, notification.IsArchived())

		notification.MarkAsRead()
		assert.True(t, notification.IsRead())
		assert.NotNil(t, notification.ReadAt)

		notification.Status = models.NotificationStatusArchived
		assert.True(t, notification.IsArchived())
	})
}

func TestProjectModel(t *testing.T) {
	db, err := fixtures.NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	data, err := db.SeedTestData()
	require.NoError(t, err)

	t.Run("Project creation", func(t *testing.T) {
		project := &models.Project{
			TenantModel: models.TenantModel{TenantID: data.Tenant.ID},
			Name:        "New Project",
			Description: "A new project for testing",
			Color:       "#28a745",
			IsActive:    true,
		}

		err := db.Create(project).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, project.ID)
		assert.True(t, project.IsActive)
	})

	t.Run("Project retrieval", func(t *testing.T) {
		var project models.Project
		err := db.First(&project, data.Project.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, data.Project.Name, project.Name)
		assert.Equal(t, "#007bff", project.Color)
		assert.True(t, project.IsActive)
	})
}