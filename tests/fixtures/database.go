package fixtures

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/drazan344/taskflow-go/internal/models"
)

// TestDB provides a test database setup
type TestDB struct {
	*gorm.DB
}

// NewTestDB creates a new in-memory SQLite database for testing
func NewTestDB() (*TestDB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Auto-migrate all models
	err = db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.UserSession{},
		&models.Task{},
		&models.TaskComment{},
		&models.TaskAttachment{},
		&models.Project{},
		&models.Tag{},
		&models.Notification{},
		&models.NotificationPreference{},
		&models.NotificationTemplate{},
		&models.NotificationQueue{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate test database: %w", err)
	}

	return &TestDB{DB: db}, nil
}

// SeedTestData creates test data in the database
func (tdb *TestDB) SeedTestData() (*TestData, error) {
	data := &TestData{}

	// Create test tenant
	tenant := &models.Tenant{
		Name:       "Test Tenant",
		Slug:       "test-tenant",
		Domain:     "test.example.com",
		Status:     models.TenantStatusActive,
		Plan:       models.TenantPlanPro,
		MaxUsers:   100,
		MaxTasks:   1000,
		MaxStorage: 1024 * 1024 * 1024, // 1GB
		Settings: models.TenantSettings{
			AllowRegistration:            true,
			RequireEmailVerification:     false,
			DefaultUserRole:              "user",
			TaskAutoAssignment:           false,
			NotificationSettings: models.NotificationSettings{
				EmailNotifications: true,
				TaskAssignments:    true,
				TaskDueDates:       true,
				TaskCompletions:    true,
				WeeklyDigest:       true,
			},
			BrandingSettings: models.BrandingSettings{
				PrimaryColor:   "#007bff",
				SecondaryColor: "#6c757d",
				LogoURL:        "",
				FaviconURL:     "",
			},
		},
	}
	if err := tdb.Create(tenant).Error; err != nil {
		return nil, fmt.Errorf("failed to create test tenant: %w", err)
	}
	data.Tenant = tenant

	// Create test admin user
	admin := &models.User{
		TenantModel:   models.TenantModel{TenantID: tenant.ID},
		Email:         "admin@test.com",
		FirstName:     "Admin",
		LastName:      "User",
		Role:          models.UserRoleAdmin,
		Status:        models.UserStatusActive,
		Timezone:      "UTC",
		Language:      "en",
		EmailVerified: true,
		Preferences: models.UserPreferences{
			Theme:               "light",
			EmailNotifications:  true,
			PushNotifications:   true,
			TaskReminders:       true,
			WeeklyDigest:        true,
			DefaultTaskPriority: "medium",
			TaskViewMode:        "list",
			ShowCompletedTasks:  false,
			TasksPerPage:        25,
		},
	}
	if err := tdb.Create(admin).Error; err != nil {
		return nil, fmt.Errorf("failed to create test admin: %w", err)
	}
	data.Admin = admin

	// Create test regular user
	user := &models.User{
		TenantModel:   models.TenantModel{TenantID: tenant.ID},
		Email:         "user@test.com",
		FirstName:     "Test",
		LastName:      "User",
		Role:          models.UserRoleUser,
		Status:        models.UserStatusActive,
		Timezone:      "America/New_York",
		Language:      "en",
		EmailVerified: true,
		Preferences: models.UserPreferences{
			Theme:               "dark",
			EmailNotifications:  false,
			PushNotifications:   false,
			TaskReminders:       false,
			WeeklyDigest:        false,
			DefaultTaskPriority: "low",
			TaskViewMode:        "board",
			ShowCompletedTasks:  true,
			TasksPerPage:        50,
		},
	}
	if err := tdb.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create test user: %w", err)
	}
	data.User = user

	// Create test project
	project := &models.Project{
		TenantModel: models.TenantModel{TenantID: tenant.ID},
		Name:        "Test Project",
		Description: "A test project for testing",
		Color:       "#007bff",
		IsActive:    true,
	}
	if err := tdb.Create(project).Error; err != nil {
		return nil, fmt.Errorf("failed to create test project: %w", err)
	}
	data.Project = project

	// Create test tags
	tag1 := &models.Tag{
		TenantModel: models.TenantModel{TenantID: tenant.ID},
		Name:        "urgent",
		Color:       "#FF0000",
	}
	tag2 := &models.Tag{
		TenantModel: models.TenantModel{TenantID: tenant.ID},
		Name:        "bug",
		Color:       "#FFA500",
	}
	if err := tdb.Create(&tag1).Error; err != nil {
		return nil, fmt.Errorf("failed to create test tag1: %w", err)
	}
	if err := tdb.Create(&tag2).Error; err != nil {
		return nil, fmt.Errorf("failed to create test tag2: %w", err)
	}
	data.Tags = []*models.Tag{tag1, tag2}

	// Create test task
	task := &models.Task{
		TenantModel: models.TenantModel{TenantID: tenant.ID},
		Title:       "Test Task",
		Description: "A test task for testing",
		Status:      models.TaskStatusTodo,
		Priority:    models.TaskPriorityHigh,
		ProjectID:   &project.ID,
		AssigneeID:  &user.ID,
		CreatorID:   admin.ID,
		DueDate:     timePtr(time.Now().AddDate(0, 0, 7)),
	}
	if err := tdb.Create(task).Error; err != nil {
		return nil, fmt.Errorf("failed to create test task: %w", err)
	}
	data.Task = task

	// Create task comment
	comment := &models.TaskComment{
		TenantModel: models.TenantModel{TenantID: tenant.ID},
		TaskID:      task.ID,
		UserID:      user.ID,
		Content:     "This is a test comment",
	}
	if err := tdb.Create(comment).Error; err != nil {
		return nil, fmt.Errorf("failed to create test comment: %w", err)
	}
	data.Comment = comment

	return data, nil
}

// TestData holds test data references
type TestData struct {
	Tenant  *models.Tenant
	Admin   *models.User
	User    *models.User
	Project *models.Project
	Task    *models.Task
	Comment *models.TaskComment
	Tags    []*models.Tag
}

// Close closes the test database
func (tdb *TestDB) Close() error {
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// timePtr returns a pointer to the given time
func timePtr(t time.Time) *time.Time {
	return &t
}