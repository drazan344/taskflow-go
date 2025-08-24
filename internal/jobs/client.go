package jobs

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

// Client wraps the Asynq client for background job processing
type Client struct {
	client *asynq.Client
}

// NewClient creates a new job client
func NewClient(redisAddr string) *Client {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	return &Client{
		client: client,
	}
}

// Close closes the job client connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Job types
const (
	TypeWelcomeEmail     = "email:welcome"
	TypePasswordReset    = "email:password_reset"
	TypeTaskNotification = "notification:task"
	TypeEmailDigest      = "email:digest"
	TypeDataExport       = "data:export"
)


// EnqueueWelcomeEmail enqueues a welcome email job
func (c *Client) EnqueueWelcomeEmail(payload WelcomeEmailPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeWelcomeEmail, data)
	_, err = c.client.Enqueue(task, asynq.Queue("emails"))
	return err
}

// EnqueuePasswordResetEmail enqueues a password reset email job
func (c *Client) EnqueuePasswordResetEmail(payload PasswordResetEmailPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypePasswordReset, data)
	_, err = c.client.Enqueue(task, asynq.Queue("emails"))
	return err
}

// EnqueueTaskNotification enqueues a task notification job  
func (c *Client) EnqueueTaskNotification(payload TaskAssignedEmailPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeTaskNotification, data)
	_, err = c.client.Enqueue(task, asynq.Queue("notifications"))
	return err
}

// EnqueueEmailDigest enqueues an email digest job
func (c *Client) EnqueueEmailDigest(payload WeeklyDigestEmailPayload, processAt time.Time) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeEmailDigest, data)
	_, err = c.client.Enqueue(task, 
		asynq.Queue("emails"),
		asynq.ProcessAt(processAt),
	)
	return err
}

// EnqueueDataExport enqueues a data export job
func (c *Client) EnqueueDataExport(payload GenerateReportPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeDataExport, data)
	_, err = c.client.Enqueue(task, 
		asynq.Queue("exports"),
		asynq.MaxRetry(3),
		asynq.Timeout(10*time.Minute),
	)
	return err
}