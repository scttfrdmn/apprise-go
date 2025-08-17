package apprise

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Add adds a job to the notification queue
func (q *NotificationQueue) Add(job QueuedJob) (*QueuedJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Set defaults
	now := time.Now()
	job.CreatedAt = now
	job.ScheduledAt = now
	job.Status = string(JobStatusPending)

	if job.Priority == 0 {
		job.Priority = 1
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}
	if job.RetryDelay == 0 {
		job.RetryDelay = time.Minute * 5
	}

	// Insert into database
	servicesJSON, _ := json.Marshal(job.Services)
	tagsJSON, _ := json.Marshal(job.Tags)
	metadataJSON, _ := json.Marshal(job.Metadata)

	query := `INSERT INTO notification_queue (scheduled_id, title, body, notify_type, services, tags, metadata,
			  priority, max_retries, retry_count, retry_delay, status, created_at, scheduled_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := q.db.Exec(query, job.ScheduledID, job.Title, job.Body, int(job.NotifyType),
		string(servicesJSON), string(tagsJSON), string(metadataJSON), job.Priority,
		job.MaxRetries, job.RetryCount, int64(job.RetryDelay), job.Status, job.CreatedAt, job.ScheduledAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert queued job: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get job ID: %w", err)
	}
	job.ID = id

	q.logger.Printf("Added job to queue: %s (ID: %d, Priority: %d)", job.Title, job.ID, job.Priority)
	return &job, nil
}

// GetPendingJobs returns jobs ready for processing
func (q *NotificationQueue) GetPendingJobs(limit int) ([]QueuedJob, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	query := `SELECT id, scheduled_id, title, body, notify_type, services, tags, metadata,
			  priority, max_retries, retry_count, retry_delay, status, error_message,
			  created_at, scheduled_at, started_at, completed_at, next_retry_at
			  FROM notification_queue
			  WHERE status IN ('pending', 'retrying') AND (next_retry_at IS NULL OR next_retry_at <= ?)
			  ORDER BY priority DESC, created_at ASC
			  LIMIT ?`

	rows, err := q.db.Query(query, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var jobs []QueuedJob
	for rows.Next() {
		job, err := q.scanQueuedJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// UpdateJobStatus updates the status of a queued job
func (q *NotificationQueue) UpdateJobStatus(jobID int64, status JobStatus, errorMessage string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	var query string
	var args []interface{}

	switch status {
	case JobStatusRunning:
		query = `UPDATE notification_queue SET status = ?, started_at = ? WHERE id = ?`
		args = []interface{}{string(status), now, jobID}
	case JobStatusCompleted:
		query = `UPDATE notification_queue SET status = ?, completed_at = ? WHERE id = ?`
		args = []interface{}{string(status), now, jobID}
	case JobStatusFailed:
		query = `UPDATE notification_queue SET status = ?, error_message = ?, completed_at = ? WHERE id = ?`
		args = []interface{}{string(status), errorMessage, now, jobID}
	case JobStatusRetrying:
		// Calculate next retry time with exponential backoff
		job, err := q.getQueuedJob(jobID)
		if err != nil {
			return fmt.Errorf("failed to get job for retry: %w", err)
		}
		
		backoffMultiplier := 1 << job.RetryCount // 2^retry_count
		if backoffMultiplier > 64 {
			backoffMultiplier = 64 // Cap at 64x
		}
		nextRetry := now.Add(job.RetryDelay * time.Duration(backoffMultiplier))
		
		query = `UPDATE notification_queue SET status = ?, retry_count = retry_count + 1, 
				 error_message = ?, next_retry_at = ? WHERE id = ?`
		args = []interface{}{string(status), errorMessage, nextRetry, jobID}
	default:
		query = `UPDATE notification_queue SET status = ?, error_message = ? WHERE id = ?`
		args = []interface{}{string(status), errorMessage, jobID}
	}

	_, err := q.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// DeleteJob removes a job from the queue
func (q *NotificationQueue) DeleteJob(jobID int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	query := `DELETE FROM notification_queue WHERE id = ?`
	_, err := q.db.Exec(query, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete queued job: %w", err)
	}

	return nil
}

// GetJobStats returns queue statistics
func (q *NotificationQueue) GetJobStats() (map[string]int64, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := make(map[string]int64)

	// Count by status
	query := `SELECT status, COUNT(*) as count FROM notification_queue GROUP BY status`
	rows, err := q.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query job stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan job stats: %w", err)
		}
		stats[status] = count
	}

	// Total jobs
	var total int64
	query = `SELECT COUNT(*) FROM notification_queue`
	if err := q.db.QueryRow(query).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to query total jobs: %w", err)
	}
	stats["total"] = total

	return stats, nil
}

// CleanupCompletedJobs removes old completed/failed jobs
func (q *NotificationQueue) CleanupCompletedJobs(olderThan time.Duration) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM notification_queue WHERE status IN ('completed', 'failed') AND completed_at < ?`
	
	result, err := q.db.Exec(query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup completed jobs: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	q.logger.Printf("Cleaned up %d completed jobs older than %v", deleted, olderThan)
	return deleted, nil
}

// getQueuedJob retrieves a single queued job by ID
func (q *NotificationQueue) getQueuedJob(jobID int64) (*QueuedJob, error) {
	query := `SELECT id, scheduled_id, title, body, notify_type, services, tags, metadata,
			  priority, max_retries, retry_count, retry_delay, status, error_message,
			  created_at, scheduled_at, started_at, completed_at, next_retry_at
			  FROM notification_queue WHERE id = ?`

	row := q.db.QueryRow(query, jobID)
	return q.scanQueuedJobRow(row)
}

// scanQueuedJob scans a queued job from database rows
func (q *NotificationQueue) scanQueuedJob(rows *sql.Rows) (QueuedJob, error) {
	var job QueuedJob
	var servicesJSON, tagsJSON, metadataJSON string
	var scheduledID sql.NullInt64
	var startedAt, completedAt, nextRetryAt sql.NullTime

	err := rows.Scan(&job.ID, &scheduledID, &job.Title, &job.Body, &job.NotifyType,
		&servicesJSON, &tagsJSON, &metadataJSON, &job.Priority, &job.MaxRetries,
		&job.RetryCount, &job.RetryDelay, &job.Status, &job.ErrorMessage,
		&job.CreatedAt, &job.ScheduledAt, &startedAt, &completedAt, &nextRetryAt)
	if err != nil {
		return job, fmt.Errorf("failed to scan queued job: %w", err)
	}

	// Parse optional fields
	if scheduledID.Valid {
		job.ScheduledID = &scheduledID.Int64
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if nextRetryAt.Valid {
		job.NextRetryAt = &nextRetryAt.Time
	}

	// Parse JSON fields
	_ = json.Unmarshal([]byte(servicesJSON), &job.Services)
	_ = json.Unmarshal([]byte(tagsJSON), &job.Tags)
	_ = json.Unmarshal([]byte(metadataJSON), &job.Metadata)

	// Convert retry delay from int64 nanoseconds
	job.RetryDelay = time.Duration(job.RetryDelay)

	return job, nil
}

// scanQueuedJobRow scans a queued job from a single database row
func (q *NotificationQueue) scanQueuedJobRow(row *sql.Row) (*QueuedJob, error) {
	var job QueuedJob
	var servicesJSON, tagsJSON, metadataJSON string
	var scheduledID sql.NullInt64
	var startedAt, completedAt, nextRetryAt sql.NullTime

	err := row.Scan(&job.ID, &scheduledID, &job.Title, &job.Body, &job.NotifyType,
		&servicesJSON, &tagsJSON, &metadataJSON, &job.Priority, &job.MaxRetries,
		&job.RetryCount, &job.RetryDelay, &job.Status, &job.ErrorMessage,
		&job.CreatedAt, &job.ScheduledAt, &startedAt, &completedAt, &nextRetryAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan queued job: %w", err)
	}

	// Parse optional fields
	if scheduledID.Valid {
		job.ScheduledID = &scheduledID.Int64
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if nextRetryAt.Valid {
		job.NextRetryAt = &nextRetryAt.Time
	}

	// Parse JSON fields
	_ = json.Unmarshal([]byte(servicesJSON), &job.Services)
	_ = json.Unmarshal([]byte(tagsJSON), &job.Tags)
	_ = json.Unmarshal([]byte(metadataJSON), &job.Metadata)

	// Convert retry delay from int64 nanoseconds
	job.RetryDelay = time.Duration(job.RetryDelay)

	return &job, nil
}

// processQueue processes pending jobs from the notification queue
func (s *NotificationScheduler) processQueue(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Process every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Println("Queue processor stopping")
			return
		case <-ticker.C:
			s.processQueueBatch()
		}
	}
}

// processQueueBatch processes a batch of pending jobs
func (s *NotificationScheduler) processQueueBatch() {
	jobs, err := s.queue.GetPendingJobs(10) // Process up to 10 jobs at a time
	if err != nil {
		s.logger.Printf("Failed to get pending jobs: %v", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	s.logger.Printf("Processing %d pending jobs", len(jobs))

	for _, job := range jobs {
		s.processQueuedJob(job)
	}
}

// processQueuedJob processes a single queued job
func (s *NotificationScheduler) processQueuedJob(job QueuedJob) {
	// Mark job as running
	if err := s.queue.UpdateJobStatus(job.ID, JobStatusRunning, ""); err != nil {
		s.logger.Printf("Failed to mark job %d as running: %v", job.ID, err)
		return
	}

	s.logger.Printf("Processing job %d: %s", job.ID, job.Title)

	// Execute notification
	req := NotificationRequest{
		Title:      job.Title,
		Body:       job.Body,
		NotifyType: job.NotifyType,
		Tags:       job.Tags,
	}

	// Create temporary Apprise instance for this job
	tempApprise := New()
	for _, serviceURL := range job.Services {
		if err := tempApprise.Add(serviceURL); err != nil {
			s.logger.Printf("Failed to add service %s for job %d: %v", serviceURL, job.ID, err)
			continue
		}
	}

	// Send notifications
	responses := tempApprise.NotifyAll(req)

	// Check results
	successful := 0
	var errors []string
	for _, resp := range responses {
		if resp.Success {
			successful++
		} else {
			errors = append(errors, resp.Error.Error())
		}
	}

	if successful == len(responses) {
		// All notifications succeeded
		if err := s.queue.UpdateJobStatus(job.ID, JobStatusCompleted, ""); err != nil {
			s.logger.Printf("Failed to mark job %d as completed: %v", job.ID, err)
		} else {
			s.logger.Printf("Job %d completed successfully (%d/%d services)", 
				job.ID, successful, len(responses))
		}
	} else if successful > 0 {
		// Partial success - mark as completed with warning
		errorMsg := fmt.Sprintf("Partial success: %d/%d services failed", 
			len(responses)-successful, len(responses))
		if err := s.queue.UpdateJobStatus(job.ID, JobStatusCompleted, errorMsg); err != nil {
			s.logger.Printf("Failed to mark job %d as completed: %v", job.ID, err)
		} else {
			s.logger.Printf("Job %d completed with warnings: %s", job.ID, errorMsg)
		}
	} else {
		// All notifications failed
		errorMsg := fmt.Sprintf("All services failed: %v", errors)
		
		if job.RetryCount < job.MaxRetries {
			// Retry job
			if err := s.queue.UpdateJobStatus(job.ID, JobStatusRetrying, errorMsg); err != nil {
				s.logger.Printf("Failed to mark job %d for retry: %v", job.ID, err)
			} else {
				s.logger.Printf("Job %d scheduled for retry (%d/%d): %s", 
					job.ID, job.RetryCount+1, job.MaxRetries, errorMsg)
			}
		} else {
			// Max retries reached - mark as failed
			if err := s.queue.UpdateJobStatus(job.ID, JobStatusFailed, errorMsg); err != nil {
				s.logger.Printf("Failed to mark job %d as failed: %v", job.ID, err)
			} else {
				s.logger.Printf("Job %d failed after %d retries: %s", 
					job.ID, job.RetryCount, errorMsg)
			}
		}
	}
}