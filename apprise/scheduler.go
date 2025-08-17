package apprise

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
)

// NotificationScheduler manages scheduled notifications with persistent storage
type NotificationScheduler struct {
	cron     *cron.Cron
	db       *sql.DB
	apprise  *Apprise
	queue    *NotificationQueue
	mu       sync.RWMutex
	running  bool
	logger   *log.Logger
}

// ScheduledJob represents a scheduled notification job
type ScheduledJob struct {
	ID          int64             `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	CronExpr    string            `json:"cron_expression" db:"cron_expression"`
	Title       string            `json:"title" db:"title"`
	Body        string            `json:"body" db:"body"`
	NotifyType  NotifyType        `json:"notify_type" db:"notify_type"`
	Services    []string          `json:"services" db:"services"`
	Tags        []string          `json:"tags" db:"tags"`
	Metadata    map[string]string `json:"metadata" db:"metadata"`
	Template    string            `json:"template" db:"template"`
	Enabled     bool              `json:"enabled" db:"enabled"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	NextRun     *time.Time        `json:"next_run,omitempty" db:"next_run"`
	LastRun     *time.Time        `json:"last_run,omitempty" db:"last_run"`
	LastStatus  string            `json:"last_status,omitempty" db:"last_status"`
	RunCount    int64             `json:"run_count" db:"run_count"`
	CronID      cron.EntryID      `json:"-" db:"-"`
}

// NotificationQueue manages a persistent job queue with retry logic
type NotificationQueue struct {
	db     *sql.DB
	mu     sync.RWMutex
	logger *log.Logger
}

// QueuedJob represents a job in the notification queue
type QueuedJob struct {
	ID           int64             `json:"id" db:"id"`
	ScheduledID  *int64            `json:"scheduled_id,omitempty" db:"scheduled_id"`
	Title        string            `json:"title" db:"title"`
	Body         string            `json:"body" db:"body"`
	NotifyType   NotifyType        `json:"notify_type" db:"notify_type"`
	Services     []string          `json:"services" db:"services"`
	Tags         []string          `json:"tags" db:"tags"`
	Metadata     map[string]string `json:"metadata" db:"metadata"`
	Priority     int               `json:"priority" db:"priority"`
	MaxRetries   int               `json:"max_retries" db:"max_retries"`
	RetryCount   int               `json:"retry_count" db:"retry_count"`
	RetryDelay   time.Duration     `json:"retry_delay" db:"retry_delay"`
	Status       string            `json:"status" db:"status"`
	ErrorMessage string            `json:"error_message,omitempty" db:"error_message"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	ScheduledAt  time.Time         `json:"scheduled_at" db:"scheduled_at"`
	StartedAt    *time.Time        `json:"started_at,omitempty" db:"started_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty" db:"completed_at"`
	NextRetryAt  *time.Time        `json:"next_retry_at,omitempty" db:"next_retry_at"`
}

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusRetrying   JobStatus = "retrying"
	JobStatusCancelled  JobStatus = "cancelled"
)

// Public API methods

// GetScheduledJobs returns all scheduled jobs
func (s *NotificationScheduler) GetScheduledJobs() ([]ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, name, cron_expression, title, body, notify_type, services, tags, metadata,
			  template, enabled, created_at, updated_at, next_run, last_run, last_status, run_count
			  FROM scheduled_jobs ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var jobs []ScheduledJob
	for rows.Next() {
		job, err := s.scanScheduledJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// GetScheduledJob returns a specific scheduled job by ID  
func (s *NotificationScheduler) GetScheduledJob(jobID int64) (*ScheduledJob, error) {
	return s.getScheduledJob(jobID)
}

// UpdateScheduledJob updates an existing scheduled job
func (s *NotificationScheduler) UpdateScheduledJob(job ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job.UpdatedAt = time.Now()

	servicesJSON, _ := json.Marshal(job.Services)
	tagsJSON, _ := json.Marshal(job.Tags)
	metadataJSON, _ := json.Marshal(job.Metadata)

	query := `UPDATE scheduled_jobs SET name = ?, cron_expression = ?, title = ?, body = ?,
			  notify_type = ?, services = ?, tags = ?, metadata = ?, template = ?, enabled = ?, updated_at = ?
			  WHERE id = ?`

	_, err := s.db.Exec(query, job.Name, job.CronExpr, job.Title, job.Body, int(job.NotifyType),
		string(servicesJSON), string(tagsJSON), string(metadataJSON), job.Template, job.Enabled, job.UpdatedAt, job.ID)
	
	if err != nil {
		return fmt.Errorf("failed to update scheduled job: %w", err)
	}

	// Reload jobs to update cron schedule
	return s.loadScheduledJobs()
}

// RemoveScheduledJob removes a scheduled job
func (s *NotificationScheduler) RemoveScheduledJob(jobID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// First get the job to find its cron ID
	job, err := s.getScheduledJob(jobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	// Remove from cron scheduler
	if job.CronID > 0 {
		s.cron.Remove(job.CronID)
	}

	// Delete from database
	query := `DELETE FROM scheduled_jobs WHERE id = ?`
	_, err = s.db.Exec(query, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete scheduled job: %w", err)
	}

	s.logger.Printf("Removed scheduled job: %s (ID: %d)", job.Name, jobID)
	return nil
}

// EnableScheduledJob enables a scheduled job
func (s *NotificationScheduler) EnableScheduledJob(jobID int64) error {
	return s.setScheduledJobEnabled(jobID, true)
}

// DisableScheduledJob disables a scheduled job
func (s *NotificationScheduler) DisableScheduledJob(jobID int64) error {
	return s.setScheduledJobEnabled(jobID, false)
}

// GetQueuedJobs returns queued jobs with limit
func (s *NotificationScheduler) GetQueuedJobs(limit int) ([]QueuedJob, error) {
	return s.queue.GetPendingJobs(limit)
}

// QueueNotification adds a notification to the queue
func (s *NotificationScheduler) QueueNotification(job QueuedJob) (*QueuedJob, error) {
	return s.queue.Add(job)
}

// GetQueueStats returns queue statistics
func (s *NotificationScheduler) GetQueueStats() (map[string]int64, error) {
	return s.queue.GetJobStats()
}

// Helper method to enable/disable scheduled jobs
func (s *NotificationScheduler) setScheduledJobEnabled(jobID int64, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `UPDATE scheduled_jobs SET enabled = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, enabled, time.Now(), jobID)
	if err != nil {
		return fmt.Errorf("failed to update job enabled status: %w", err)
	}

	// Reload jobs to update cron schedule
	if err := s.loadScheduledJobs(); err != nil {
		return fmt.Errorf("failed to reload jobs after enable/disable: %w", err)
	}

	status := "disabled"
	if enabled {
		status = "enabled"
	}
	s.logger.Printf("Job %d %s", jobID, status)
	return nil
}

// scanScheduledJob scans a scheduled job from database rows
func (s *NotificationScheduler) scanScheduledJob(rows *sql.Rows) (ScheduledJob, error) {
	var job ScheduledJob
	var servicesJSON, tagsJSON, metadataJSON string
	var nextRun, lastRun sql.NullTime

	err := rows.Scan(&job.ID, &job.Name, &job.CronExpr, &job.Title, &job.Body, &job.NotifyType,
		&servicesJSON, &tagsJSON, &metadataJSON, &job.Template, &job.Enabled,
		&job.CreatedAt, &job.UpdatedAt, &nextRun, &lastRun, &job.LastStatus, &job.RunCount)
	if err != nil {
		return job, fmt.Errorf("failed to scan scheduled job: %w", err)
	}

	// Parse JSON fields
	_ = json.Unmarshal([]byte(servicesJSON), &job.Services)
	_ = json.Unmarshal([]byte(tagsJSON), &job.Tags)
	_ = json.Unmarshal([]byte(metadataJSON), &job.Metadata)

	// Parse optional time fields
	if nextRun.Valid {
		job.NextRun = &nextRun.Time
	}
	if lastRun.Valid {
		job.LastRun = &lastRun.Time
	}

	return job, nil
}

// NewNotificationScheduler creates a new notification scheduler
func NewNotificationScheduler(dbPath string, apprise *Apprise) (*NotificationScheduler, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize database schema
	if err := initSchedulerSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	queue := &NotificationQueue{
		db:     db,
		logger: log.Default(),
	}

	scheduler := &NotificationScheduler{
		cron:    cron.New(),
		db:      db,
		apprise: apprise,
		queue:   queue,
		logger:  log.Default(),
	}

	return scheduler, nil
}

// Start starts the scheduler and queue processing
func (s *NotificationScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	// Load existing jobs from database
	if err := s.loadScheduledJobs(); err != nil {
		return fmt.Errorf("failed to load scheduled jobs: %w", err)
	}

	// Start cron scheduler
	s.cron.Start()

	// Start queue processor
	go s.processQueue(ctx)

	s.running = true
	s.logger.Println("Notification scheduler started")

	return nil
}

// Stop stops the scheduler and queue processing
func (s *NotificationScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	s.cron.Stop()
	s.running = false
	s.logger.Println("Notification scheduler stopped")

	return nil
}

// AddScheduledJob adds a new scheduled notification job
func (s *NotificationScheduler) AddScheduledJob(job ScheduledJob) (*ScheduledJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate cron expression
	if _, err := cron.ParseStandard(job.CronExpr); err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Set timestamps
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now

	// Insert into database
	servicesJSON, _ := json.Marshal(job.Services)
	tagsJSON, _ := json.Marshal(job.Tags)
	metadataJSON, _ := json.Marshal(job.Metadata)

	query := `INSERT INTO scheduled_jobs (name, cron_expression, title, body, notify_type, services, tags, metadata, template, enabled, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := s.db.Exec(query, job.Name, job.CronExpr, job.Title, job.Body, int(job.NotifyType),
		string(servicesJSON), string(tagsJSON), string(metadataJSON), job.Template, job.Enabled,
		job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert scheduled job: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get job ID: %w", err)
	}
	job.ID = id

	// Add to cron scheduler if enabled
	if job.Enabled {
		cronID, err := s.cron.AddFunc(job.CronExpr, func() {
			s.executeScheduledJob(job.ID)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron job: %w", err)
		}
		job.CronID = cronID

		// Calculate next run time
		entry := s.cron.Entry(cronID)
		job.NextRun = &entry.Next
	}

	s.logger.Printf("Added scheduled job: %s (ID: %d)", job.Name, job.ID)
	return &job, nil
}

// executeScheduledJob executes a scheduled job by adding it to the queue
func (s *NotificationScheduler) executeScheduledJob(jobID int64) {
	job, err := s.getScheduledJob(jobID)
	if err != nil {
		s.logger.Printf("Failed to get scheduled job %d: %v", jobID, err)
		return
	}

	if !job.Enabled {
		return
	}

	// Create queued job from scheduled job
	queuedJob := QueuedJob{
		ScheduledID: &job.ID,
		Title:       job.Title,
		Body:        job.Body,
		NotifyType:  job.NotifyType,
		Services:    job.Services,
		Tags:        job.Tags,
		Metadata:    job.Metadata,
		Priority:    1,
		MaxRetries:  3,
		RetryDelay:  time.Minute * 5,
		Status:      string(JobStatusPending),
		ScheduledAt: time.Now(),
	}

	// Apply template if specified
	if job.Template != "" {
		if err := s.applyTemplate(&queuedJob, job.Template); err != nil {
			s.logger.Printf("Failed to apply template to job %d: %v", jobID, err)
		}
	}

	// Add to queue
	if _, err := s.queue.Add(queuedJob); err != nil {
		s.logger.Printf("Failed to queue scheduled job %d: %v", jobID, err)
		return
	}

	// Update last run time and run count
	s.updateScheduledJobStats(jobID)
}

// Database schema initialization and helper functions continue...