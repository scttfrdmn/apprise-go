package apprise

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// NotificationMetrics represents metrics for a notification attempt
type NotificationMetrics struct {
	ID              int64             `json:"id" db:"id"`
	JobID           *int64            `json:"job_id,omitempty" db:"job_id"`
	ScheduledJobID  *int64            `json:"scheduled_job_id,omitempty" db:"scheduled_job_id"`
	ServiceID       string            `json:"service_id" db:"service_id"`
	ServiceURL      string            `json:"service_url" db:"service_url"`
	NotificationType int              `json:"notification_type" db:"notification_type"`
	Status          string            `json:"status" db:"status"`
	DurationMs      int64             `json:"duration_ms" db:"duration_ms"`
	ErrorMessage    string            `json:"error_message,omitempty" db:"error_message"`
	Metadata        map[string]string `json:"metadata" db:"metadata"`
	Timestamp       time.Time         `json:"timestamp" db:"timestamp"`
}

// MetricsCollector collects and stores notification metrics
type MetricsCollector struct {
	db     *sql.DB
}

// MetricsReport represents aggregated metrics for reporting
type MetricsReport struct {
	Period           string            `json:"period"`
	TotalNotifications int64           `json:"total_notifications"`
	SuccessfulNotifications int64      `json:"successful_notifications"`
	FailedNotifications int64          `json:"failed_notifications"`
	SuccessRate      float64           `json:"success_rate"`
	AverageDurationMs float64          `json:"average_duration_ms"`
	ServiceMetrics   map[string]ServiceMetrics `json:"service_metrics"`
	NotificationTypes map[string]int64 `json:"notification_types"`
	HourlyBreakdown  []HourlyMetrics   `json:"hourly_breakdown"`
	TopErrors        []ErrorMetrics    `json:"top_errors"`
}

// ServiceMetrics represents metrics for a specific service
type ServiceMetrics struct {
	ServiceID       string  `json:"service_id"`
	TotalNotifications int64 `json:"total_notifications"`
	SuccessfulNotifications int64 `json:"successful_notifications"`
	FailedNotifications int64 `json:"failed_notifications"`
	SuccessRate     float64 `json:"success_rate"`
	AverageDurationMs float64 `json:"average_duration_ms"`
}

// HourlyMetrics represents metrics broken down by hour
type HourlyMetrics struct {
	Hour        string `json:"hour"`
	Total       int64  `json:"total"`
	Successful  int64  `json:"successful"`
	Failed      int64  `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

// ErrorMetrics represents top error messages and their counts
type ErrorMetrics struct {
	ErrorMessage string `json:"error_message"`
	Count        int64  `json:"count"`
	LastOccurred time.Time `json:"last_occurred"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(db *sql.DB) *MetricsCollector {
	return &MetricsCollector{
		db: db,
	}
}

// RecordMetrics records metrics for a notification attempt
func (mc *MetricsCollector) RecordMetrics(metrics NotificationMetrics) error {
	metadataJSON, err := json.Marshal(metrics.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `INSERT INTO notification_metrics (job_id, scheduled_job_id, service_id, service_url,
			  notification_type, status, duration_ms, error_message, metadata, timestamp)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = mc.db.Exec(query, metrics.JobID, metrics.ScheduledJobID, metrics.ServiceID,
		metrics.ServiceURL, metrics.NotificationType, metrics.Status, metrics.DurationMs,
		metrics.ErrorMessage, string(metadataJSON), metrics.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to insert metrics: %w", err)
	}

	return nil
}

// GetMetricsReport generates a comprehensive metrics report
func (mc *MetricsCollector) GetMetricsReport(startTime, endTime time.Time) (*MetricsReport, error) {
	report := &MetricsReport{
		Period: fmt.Sprintf("%s to %s", startTime.Format("2006-01-02 15:04"), endTime.Format("2006-01-02 15:04")),
		ServiceMetrics: make(map[string]ServiceMetrics),
		NotificationTypes: make(map[string]int64),
	}

	// Get overall statistics
	if err := mc.getOverallStats(report, startTime, endTime); err != nil {
		return nil, fmt.Errorf("failed to get overall stats: %w", err)
	}

	// Get service-specific metrics
	if err := mc.getServiceMetrics(report, startTime, endTime); err != nil {
		return nil, fmt.Errorf("failed to get service metrics: %w", err)
	}

	// Get notification type breakdown
	if err := mc.getNotificationTypeMetrics(report, startTime, endTime); err != nil {
		return nil, fmt.Errorf("failed to get notification type metrics: %w", err)
	}

	// Get hourly breakdown
	if err := mc.getHourlyMetrics(report, startTime, endTime); err != nil {
		return nil, fmt.Errorf("failed to get hourly metrics: %w", err)
	}

	// Get top errors
	if err := mc.getTopErrors(report, startTime, endTime); err != nil {
		return nil, fmt.Errorf("failed to get top errors: %w", err)
	}

	return report, nil
}

// getOverallStats gets overall statistics for the report
func (mc *MetricsCollector) getOverallStats(report *MetricsReport, startTime, endTime time.Time) error {
	query := `SELECT 
				COUNT(*) as total,
				SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
				AVG(duration_ms) as avg_duration
			  FROM notification_metrics 
			  WHERE timestamp BETWEEN ? AND ?`

	row := mc.db.QueryRow(query, startTime, endTime)

	var avgDuration sql.NullFloat64
	err := row.Scan(&report.TotalNotifications, &report.SuccessfulNotifications, 
		&report.FailedNotifications, &avgDuration)
	if err != nil {
		return fmt.Errorf("failed to scan overall stats: %w", err)
	}

	if avgDuration.Valid {
		report.AverageDurationMs = avgDuration.Float64
	}

	if report.TotalNotifications > 0 {
		report.SuccessRate = float64(report.SuccessfulNotifications) / float64(report.TotalNotifications) * 100
	}

	return nil
}

// getServiceMetrics gets metrics broken down by service
func (mc *MetricsCollector) getServiceMetrics(report *MetricsReport, startTime, endTime time.Time) error {
	query := `SELECT 
				service_id,
				COUNT(*) as total,
				SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
				AVG(duration_ms) as avg_duration
			  FROM notification_metrics 
			  WHERE timestamp BETWEEN ? AND ?
			  GROUP BY service_id
			  ORDER BY total DESC`

	rows, err := mc.db.Query(query, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query service metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var serviceID string
		var total, successful, failed int64
		var avgDuration sql.NullFloat64

		err := rows.Scan(&serviceID, &total, &successful, &failed, &avgDuration)
		if err != nil {
			return fmt.Errorf("failed to scan service metrics: %w", err)
		}

		metrics := ServiceMetrics{
			ServiceID: serviceID,
			TotalNotifications: total,
			SuccessfulNotifications: successful,
			FailedNotifications: failed,
		}

		if avgDuration.Valid {
			metrics.AverageDurationMs = avgDuration.Float64
		}

		if total > 0 {
			metrics.SuccessRate = float64(successful) / float64(total) * 100
		}

		report.ServiceMetrics[serviceID] = metrics
	}

	return nil
}

// getNotificationTypeMetrics gets metrics broken down by notification type
func (mc *MetricsCollector) getNotificationTypeMetrics(report *MetricsReport, startTime, endTime time.Time) error {
	query := `SELECT 
				notification_type,
				COUNT(*) as total
			  FROM notification_metrics 
			  WHERE timestamp BETWEEN ? AND ?
			  GROUP BY notification_type
			  ORDER BY total DESC`

	rows, err := mc.db.Query(query, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query notification type metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var notificationType int
		var total int64

		err := rows.Scan(&notificationType, &total)
		if err != nil {
			return fmt.Errorf("failed to scan notification type metrics: %w", err)
		}

		// Convert notification type to string
		typeStr := NotifyType(notificationType).String()
		report.NotificationTypes[typeStr] = total
	}

	return nil
}

// getHourlyMetrics gets metrics broken down by hour
func (mc *MetricsCollector) getHourlyMetrics(report *MetricsReport, startTime, endTime time.Time) error {
	query := `SELECT 
				strftime('%Y-%m-%d %H:00', timestamp) as hour,
				COUNT(*) as total,
				SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
			  FROM notification_metrics 
			  WHERE timestamp BETWEEN ? AND ?
			  GROUP BY strftime('%Y-%m-%d %H:00', timestamp)
			  ORDER BY hour`

	rows, err := mc.db.Query(query, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query hourly metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var hour string
		var total, successful, failed int64

		err := rows.Scan(&hour, &total, &successful, &failed)
		if err != nil {
			return fmt.Errorf("failed to scan hourly metrics: %w", err)
		}

		metrics := HourlyMetrics{
			Hour:       hour,
			Total:      total,
			Successful: successful,
			Failed:     failed,
		}

		if total > 0 {
			metrics.SuccessRate = float64(successful) / float64(total) * 100
		}

		report.HourlyBreakdown = append(report.HourlyBreakdown, metrics)
	}

	return nil
}

// getTopErrors gets the most common error messages
func (mc *MetricsCollector) getTopErrors(report *MetricsReport, startTime, endTime time.Time) error {
	query := `SELECT 
				error_message,
				COUNT(*) as count,
				MAX(timestamp) as last_occurred
			  FROM notification_metrics 
			  WHERE timestamp BETWEEN ? AND ? 
			    AND status = 'failed' 
			    AND error_message != ''
			  GROUP BY error_message
			  ORDER BY count DESC
			  LIMIT 10`

	rows, err := mc.db.Query(query, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query top errors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var errorMessage string
		var count int64
		var lastOccurredStr string

		err := rows.Scan(&errorMessage, &count, &lastOccurredStr)
		if err != nil {
			return fmt.Errorf("failed to scan top errors: %w", err)
		}

		// Parse the timestamp string
		lastOccurred, err := time.Parse("2006-01-02 15:04:05", lastOccurredStr)
		if err != nil {
			// Try alternative format
			lastOccurred, err = time.Parse(time.RFC3339, lastOccurredStr)
			if err != nil {
				// Use current time as fallback
				lastOccurred = time.Now()
			}
		}

		report.TopErrors = append(report.TopErrors, ErrorMetrics{
			ErrorMessage: errorMessage,
			Count:        count,
			LastOccurred: lastOccurred,
		})
	}

	return nil
}

// CleanupOldMetrics removes metrics older than the specified duration
func (mc *MetricsCollector) CleanupOldMetrics(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM notification_metrics WHERE timestamp < ?`

	result, err := mc.db.Exec(query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old metrics: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return deleted, nil
}

// GetMetricsCollector returns the metrics collector for the scheduler
func (s *NotificationScheduler) GetMetricsCollector() *MetricsCollector {
	return NewMetricsCollector(s.db)
}

