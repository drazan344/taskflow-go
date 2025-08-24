package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/drazan344/taskflow-go/internal/config"
	"github.com/drazan344/taskflow-go/internal/models"
)

// Server represents the background job server
type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	db     *gorm.DB
	logger *logrus.Logger
	config *config.Config
}

// NewServer creates a new job server
func NewServer(cfg *config.Config, db *gorm.DB, logger *logrus.Logger) *Server {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.GetRedisAddr()},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"emails":        6, // high priority for emails
				"notifications": 3, // medium priority for notifications
				"exports":       1, // low priority for exports
			},
			StrictPriority: true,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.WithFields(logrus.Fields{
					"task_type":    task.Type(),
					"task_payload": string(task.Payload()),
					"error":        err,
				}).Error("Background job failed")
			}),
			// Logger: log.New(logger.Writer(), "[ASYNQ] ", log.LstdFlags), // Commented out due to interface mismatch
		},
	)

	mux := asynq.NewServeMux()
	
	jobServer := &Server{
		server: srv,
		mux:    mux,
		db:     db,
		logger: logger,
		config: cfg,
	}

	// Register handlers
	jobServer.registerHandlers()

	return jobServer
}

// registerHandlers registers all job handlers
func (s *Server) registerHandlers() {
	s.mux.HandleFunc(TypeWelcomeEmail, s.handleWelcomeEmail)
	s.mux.HandleFunc(TypePasswordReset, s.handlePasswordReset)
	s.mux.HandleFunc(TypeTaskNotification, s.handleTaskNotification)
	s.mux.HandleFunc(TypeEmailDigest, s.handleEmailDigest)
	s.mux.HandleFunc(TypeDataExport, s.handleDataExport)
}

// Start starts the job server
func (s *Server) Start() error {
	s.logger.Info("Starting background job server...")
	return s.server.Run(s.mux)
}

// Shutdown gracefully shuts down the job server
func (s *Server) Shutdown() {
	s.logger.Info("Shutting down background job server...")
	s.server.Shutdown()
}

// handleWelcomeEmail handles welcome email jobs
func (s *Server) handleWelcomeEmail(ctx context.Context, t *asynq.Task) error {
	var payload WelcomeEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    payload.UserID,
		"tenant_id":  payload.TenantID,
		"email":      payload.Email,
		"first_name": payload.FirstName,
	}).Info("Processing welcome email job")

	// TODO: Integrate with actual email service (SendGrid, SES, etc.)
	// For now, we'll just simulate sending the email
	if err := s.sendWelcomeEmail(payload); err != nil {
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	// Create notification record
	notification := &models.Notification{
		TenantModel: models.TenantModel{TenantID: payload.TenantID},
		UserID:      payload.UserID,
		Type:        models.NotificationTypeWelcome,
		Status:      models.NotificationStatusUnread,
		Title:       "Welcome to TaskFlow!",
		Message:     fmt.Sprintf("Hello %s! Welcome to TaskFlow. We're excited to have you on board.", payload.FirstName),
		Data: models.NotificationData{
			ActorName:  payload.FirstName,
			EntityType: "welcome",
			ExtraData: map[string]interface{}{
				"email_sent": true,
				"sent_at":    time.Now(),
			},
		},
	}

	if err := s.db.Create(notification).Error; err != nil {
		s.logger.WithError(err).Error("Failed to create welcome notification")
		// Don't fail the job for this
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   payload.UserID,
		"tenant_id": payload.TenantID,
		"email":     payload.Email,
	}).Info("Welcome email sent successfully")

	return nil
}

// handlePasswordReset handles password reset email jobs
func (s *Server) handlePasswordReset(ctx context.Context, t *asynq.Task) error {
	var payload PasswordResetEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    payload.UserID,
		"tenant_id":  payload.TenantID,
		"email":      payload.Email,
	}).Info("Processing password reset email job")

	// TODO: Send actual password reset email
	if err := s.sendPasswordResetEmail(payload); err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   payload.UserID,
		"tenant_id": payload.TenantID,
		"email":     payload.Email,
	}).Info("Password reset email sent successfully")

	return nil
}

// handleTaskNotification handles task notification jobs
func (s *Server) handleTaskNotification(ctx context.Context, t *asynq.Task) error {
	var payload TaskAssignedEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	s.logger.WithFields(logrus.Fields{
		"task_id":      payload.TaskID,
		"tenant_id":    payload.TenantID,
		"assignee_id":  payload.AssigneeID,
		"task_title":   payload.TaskTitle,
	}).Info("Processing task notification job")

	// Create in-app notification
	var message string
	var notificationType models.NotificationType

	// This is a task assignment notification
	message = fmt.Sprintf("Task assigned to you: %s", payload.TaskTitle)
	notificationType = models.NotificationTypeTaskAssigned

	notification := &models.Notification{
		TenantModel: models.TenantModel{TenantID: payload.TenantID},
		UserID:      payload.AssigneeID,
		Type:        notificationType,
		Status:      models.NotificationStatusUnread,
		Title:       "Task Assigned",
		Message:     message,
		Data: models.NotificationData{
			EntityType: "task",
			EntityID:   payload.TaskID.String(),
			EntityName: payload.TaskTitle,
			ActorName:  payload.AssignerName,
			ExtraData: map[string]interface{}{
				"task_id":    payload.TaskID,
				"task_title": payload.TaskTitle,
				"action":     "assigned",
			},
		},
	}

	if err := s.db.Create(notification).Error; err != nil {
		return fmt.Errorf("failed to create task notification: %w", err)
	}

	// TODO: Send push notification if enabled
	// TODO: Send email notification if enabled

	s.logger.WithFields(logrus.Fields{
		"task_id":         payload.TaskID,
		"tenant_id":       payload.TenantID,
		"assignee_id":     payload.AssigneeID,
		"notification_id": notification.ID,
	}).Info("Task notification created successfully")

	return nil
}

// handleEmailDigest handles email digest jobs
func (s *Server) handleEmailDigest(ctx context.Context, t *asynq.Task) error {
	var payload WeeklyDigestEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    payload.UserID,
		"tenant_id":  payload.TenantID,
		"email":      payload.Email,
		"user_name":  payload.UserName,
	}).Info("Processing email digest job")

	// TODO: Generate and send email digest
	if err := s.sendEmailDigest(payload); err != nil {
		return fmt.Errorf("failed to send email digest: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   payload.UserID,
		"tenant_id": payload.TenantID,
		"user_name": payload.UserName,
	}).Info("Email digest sent successfully")

	return nil
}

// handleDataExport handles data export jobs
func (s *Server) handleDataExport(ctx context.Context, t *asynq.Task) error {
	var payload GenerateReportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":     payload.UserID,
		"tenant_id":   payload.TenantID,
		"report_type": payload.ReportType,
		"format":      payload.Format,
	}).Info("Processing data export job")

	// TODO: Generate export file and send download link via email
	if err := s.generateAndSendExport(payload); err != nil {
		return fmt.Errorf("failed to generate data export: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":     payload.UserID,
		"tenant_id":   payload.TenantID,
		"report_type": payload.ReportType,
		"format":      payload.Format,
	}).Info("Data export generated and sent successfully")

	return nil
}

// Email service methods (stubs for now)
func (s *Server) sendWelcomeEmail(payload WelcomeEmailPayload) error {
	// TODO: Implement actual email sending
	s.logger.WithFields(logrus.Fields{
		"to":         payload.Email,
		"first_name": payload.FirstName,
		"template":   "welcome",
	}).Info("Simulating welcome email send")
	
	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)
	
	return nil
}

func (s *Server) sendPasswordResetEmail(payload PasswordResetEmailPayload) error {
	// TODO: Implement actual email sending with reset link
	s.logger.WithFields(logrus.Fields{
		"to":         payload.Email,
		"first_name": payload.FirstName,
		"template":   "password_reset",
		"token":      payload.ResetToken[:8] + "...", // Log only first 8 chars for security
	}).Info("Simulating password reset email send")
	
	time.Sleep(100 * time.Millisecond)
	
	return nil
}

func (s *Server) sendEmailDigest(payload WeeklyDigestEmailPayload) error {
	// TODO: Generate digest content and send email
	s.logger.WithFields(logrus.Fields{
		"to":        payload.Email,
		"user_name": payload.UserName,
		"template":  "digest",
	}).Info("Simulating email digest send")
	
	time.Sleep(200 * time.Millisecond)
	
	return nil
}

func (s *Server) generateAndSendExport(payload GenerateReportPayload) error {
	// TODO: Generate export file and send download link
	s.logger.WithFields(logrus.Fields{
		"recipients":   payload.Recipients,
		"report_type":  payload.ReportType,
		"format":       payload.Format,
		"template":     "export_ready",
	}).Info("Simulating data export generation and email send")
	
	time.Sleep(500 * time.Millisecond) // Simulate longer processing time
	
	return nil
}