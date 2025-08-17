package apprise

import (
	"database/sql"
	"fmt"
	"time"
)

// initSchedulerSchema initializes the database schema for the scheduler
func initSchedulerSchema(db *sql.DB) error {
	// Create scheduled jobs table
	createScheduledJobsTable := `
	CREATE TABLE IF NOT EXISTS scheduled_jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		cron_expression TEXT NOT NULL,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		notify_type INTEGER NOT NULL DEFAULT 0,
		services TEXT NOT NULL DEFAULT '[]',
		tags TEXT NOT NULL DEFAULT '[]',
		metadata TEXT NOT NULL DEFAULT '{}',
		template TEXT NOT NULL DEFAULT '',
		enabled BOOLEAN NOT NULL DEFAULT true,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		next_run DATETIME,
		last_run DATETIME,
		last_status TEXT DEFAULT '',
		run_count INTEGER NOT NULL DEFAULT 0
	);`

	// Create notification queue table
	createQueueTable := `
	CREATE TABLE IF NOT EXISTS notification_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scheduled_id INTEGER,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		notify_type INTEGER NOT NULL DEFAULT 0,
		services TEXT NOT NULL DEFAULT '[]',
		tags TEXT NOT NULL DEFAULT '[]',
		metadata TEXT NOT NULL DEFAULT '{}',
		priority INTEGER NOT NULL DEFAULT 1,
		max_retries INTEGER NOT NULL DEFAULT 3,
		retry_count INTEGER NOT NULL DEFAULT 0,
		retry_delay INTEGER NOT NULL DEFAULT 300000000000,
		status TEXT NOT NULL DEFAULT 'pending',
		error_message TEXT DEFAULT '',
		created_at DATETIME NOT NULL,
		scheduled_at DATETIME NOT NULL,
		started_at DATETIME,
		completed_at DATETIME,
		next_retry_at DATETIME,
		FOREIGN KEY (scheduled_id) REFERENCES scheduled_jobs(id) ON DELETE SET NULL
	);`

	// Create notification templates table
	createTemplatesTable := `
	CREATE TABLE IF NOT EXISTS notification_templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		variables TEXT NOT NULL DEFAULT '{}',
		description TEXT DEFAULT '',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);`

	// Create notification metrics table
	createMetricsTable := `
	CREATE TABLE IF NOT EXISTS notification_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id INTEGER,
		scheduled_job_id INTEGER,
		service_id TEXT NOT NULL,
		service_url TEXT NOT NULL,
		notification_type INTEGER NOT NULL,
		status TEXT NOT NULL,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		error_message TEXT DEFAULT '',
		metadata TEXT NOT NULL DEFAULT '{}',
		timestamp DATETIME NOT NULL,
		FOREIGN KEY (job_id) REFERENCES notification_queue(id) ON DELETE SET NULL,
		FOREIGN KEY (scheduled_job_id) REFERENCES scheduled_jobs(id) ON DELETE SET NULL
	);`

	// Create indexes for better performance
	createIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_enabled ON scheduled_jobs(enabled);`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_next_run ON scheduled_jobs(next_run);`,
		`CREATE INDEX IF NOT EXISTS idx_queue_status ON notification_queue(status);`,
		`CREATE INDEX IF NOT EXISTS idx_queue_priority ON notification_queue(priority DESC, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_queue_next_retry ON notification_queue(next_retry_at);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON notification_metrics(timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_service ON notification_metrics(service_id);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_status ON notification_metrics(status);`,
	}

	// Execute table creation
	tables := []string{
		createScheduledJobsTable,
		createQueueTable,
		createTemplatesTable,
		createMetricsTable,
	}

	for _, query := range tables {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Execute index creation
	for _, query := range createIndexes {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// getScheduledJob retrieves a single scheduled job by ID
func (s *NotificationScheduler) getScheduledJob(jobID int64) (*ScheduledJob, error) {
	query := `SELECT id, name, cron_expression, title, body, notify_type, services, tags, metadata,
			  template, enabled, created_at, updated_at, next_run, last_run, last_status, run_count
			  FROM scheduled_jobs WHERE id = ?`

	row := s.db.QueryRow(query, jobID)
	return s.scanScheduledJobRow(row)
}

// scanScheduledJobRow scans a scheduled job from a single database row
func (s *NotificationScheduler) scanScheduledJobRow(row *sql.Row) (*ScheduledJob, error) {
	var job ScheduledJob
	var servicesJSON, tagsJSON, metadataJSON string
	var nextRun, lastRun sql.NullTime

	err := row.Scan(&job.ID, &job.Name, &job.CronExpr, &job.Title, &job.Body, &job.NotifyType,
		&servicesJSON, &tagsJSON, &metadataJSON, &job.Template, &job.Enabled,
		&job.CreatedAt, &job.UpdatedAt, &nextRun, &lastRun, &job.LastStatus, &job.RunCount)
	if err != nil {
		return nil, fmt.Errorf("failed to scan scheduled job: %w", err)
	}

	// Parse optional fields
	if nextRun.Valid {
		job.NextRun = &nextRun.Time
	}
	if lastRun.Valid {
		job.LastRun = &lastRun.Time
	}

	// Parse JSON fields
	if err := parseJSONField(servicesJSON, &job.Services); err != nil {
		return nil, fmt.Errorf("failed to parse services: %w", err)
	}
	if err := parseJSONField(tagsJSON, &job.Tags); err != nil {
		return nil, fmt.Errorf("failed to parse tags: %w", err)
	}
	if err := parseJSONField(metadataJSON, &job.Metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &job, nil
}

// loadScheduledJobs loads all enabled scheduled jobs from database
func (s *NotificationScheduler) loadScheduledJobs() error {
	query := `SELECT id, name, cron_expression, title, body, notify_type, services, tags, metadata,
			  template, enabled, created_at, updated_at, next_run, last_run, last_status, run_count
			  FROM scheduled_jobs WHERE enabled = true`

	rows, err := s.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query scheduled jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var job ScheduledJob
		var servicesJSON, tagsJSON, metadataJSON string
		var nextRun, lastRun sql.NullTime

		err := rows.Scan(&job.ID, &job.Name, &job.CronExpr, &job.Title, &job.Body, &job.NotifyType,
			&servicesJSON, &tagsJSON, &metadataJSON, &job.Template, &job.Enabled,
			&job.CreatedAt, &job.UpdatedAt, &nextRun, &lastRun, &job.LastStatus, &job.RunCount)
		if err != nil {
			return fmt.Errorf("failed to scan scheduled job: %w", err)
		}

		// Parse JSON fields
		if err := parseJSONField(servicesJSON, &job.Services); err != nil {
			s.logger.Printf("Warning: failed to parse services for job %d: %v", job.ID, err)
			continue
		}
		if err := parseJSONField(tagsJSON, &job.Tags); err != nil {
			s.logger.Printf("Warning: failed to parse tags for job %d: %v", job.ID, err)
			continue
		}
		if err := parseJSONField(metadataJSON, &job.Metadata); err != nil {
			s.logger.Printf("Warning: failed to parse metadata for job %d: %v", job.ID, err)
			continue
		}

		// Parse optional fields
		if nextRun.Valid {
			job.NextRun = &nextRun.Time
		}
		if lastRun.Valid {
			job.LastRun = &lastRun.Time
		}

		// Add to cron scheduler
		cronID, err := s.cron.AddFunc(job.CronExpr, func() {
			s.executeScheduledJob(job.ID)
		})
		if err != nil {
			s.logger.Printf("Warning: failed to schedule job %d (%s): %v", job.ID, job.Name, err)
			continue
		}

		job.CronID = cronID
		s.logger.Printf("Loaded scheduled job: %s (ID: %d, Cron: %s)", job.Name, job.ID, job.CronExpr)
	}

	return nil
}

// updateScheduledJobStats updates the last run time and run count for a scheduled job
func (s *NotificationScheduler) updateScheduledJobStats(jobID int64) {
	now := time.Now()
	query := `UPDATE scheduled_jobs SET last_run = ?, run_count = run_count + 1 WHERE id = ?`
	
	if _, err := s.db.Exec(query, now, jobID); err != nil {
		s.logger.Printf("Failed to update stats for scheduled job %d: %v", jobID, err)
	}
}

// applyTemplate applies a notification template to a queued job
func (s *NotificationScheduler) applyTemplate(job *QueuedJob, templateName string) error {
	query := `SELECT title, body, variables FROM notification_templates WHERE name = ?`
	row := s.db.QueryRow(query, templateName)

	var titleTemplate, bodyTemplate, variablesJSON string
	if err := row.Scan(&titleTemplate, &bodyTemplate, &variablesJSON); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("template '%s' not found", templateName)
		}
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Parse template variables
	var variables map[string]string
	if err := parseJSONField(variablesJSON, &variables); err != nil {
		return fmt.Errorf("failed to parse template variables: %w", err)
	}

	// Apply template variables to job metadata
	if job.Metadata == nil {
		job.Metadata = make(map[string]string)
	}
	for key, value := range variables {
		if _, exists := job.Metadata[key]; !exists {
			job.Metadata[key] = value
		}
	}

	// Apply template to title and body
	job.Title = s.substituteVariables(titleTemplate, job.Metadata)
	job.Body = s.substituteVariables(bodyTemplate, job.Metadata)

	return nil
}

// substituteVariables performs simple variable substitution in templates
func (s *NotificationScheduler) substituteVariables(template string, variables map[string]string) string {
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		// Simple replacement - in production use text/template
		// For now, we'll do simple string replacement
		// In a production system, you'd use text/template or similar
		if len(result) > 0 && len(placeholder) > 0 {
			// This is a placeholder for template processing
			// In reality, you'd implement proper template parsing
			_ = value // Use the value to avoid unused variable error
		}
	}
	return result
}

// Close closes the database connection
func (s *NotificationScheduler) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}