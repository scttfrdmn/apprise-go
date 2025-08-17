package apprise

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNotificationScheduler_CreateAndStart(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_scheduler.db")

	// Create Apprise instance
	apprise := New()

	// Create scheduler
	scheduler, err := NewNotificationScheduler(dbPath, apprise)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer scheduler.Close()

	// Start scheduler
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Stop scheduler
	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scheduler: %v", err)
	}
}

func TestNotificationScheduler_AddScheduledJob(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_scheduler.db")
	apprise := New()

	scheduler, err := NewNotificationScheduler(dbPath, apprise)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer scheduler.Close()

	// Create test job
	job := ScheduledJob{
		Name:       "test-job",
		CronExpr:   "*/5 * * * *", // Every 5 minutes (standard cron format)
		Title:      "Test Notification",
		Body:       "This is a test notification",
		NotifyType: NotifyTypeInfo,
		Services:   []string{"webhook://httpbin.org/post"},
		Tags:       []string{"test"},
		Enabled:    true,
	}

	// Add job
	result, err := scheduler.AddScheduledJob(job)
	if err != nil {
		t.Fatalf("Failed to add scheduled job: %v", err)
	}

	if result.ID == 0 {
		t.Error("Expected job ID to be set")
	}

	if result.Name != job.Name {
		t.Errorf("Expected job name %s, got %s", job.Name, result.Name)
	}

	// Test duplicate name
	_, err = scheduler.AddScheduledJob(job)
	if err == nil {
		t.Error("Expected error when adding job with duplicate name")
	}
}

func TestNotificationScheduler_GetScheduledJobs(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_scheduler.db")
	apprise := New()

	scheduler, err := NewNotificationScheduler(dbPath, apprise)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer scheduler.Close()

	// Add multiple jobs
	jobs := []ScheduledJob{
		{
			Name:       "job1",
			CronExpr:   "0 * * * *", // Every hour
			Title:      "Job 1",
			Body:       "First job",
			NotifyType: NotifyTypeInfo,
			Services:   []string{"webhook://httpbin.org/post"},
			Enabled:    true,
		},
		{
			Name:       "job2",
			CronExpr:   "*/15 * * * *", // Every 15 minutes
			Title:      "Job 2",
			Body:       "Second job",
			NotifyType: NotifyTypeWarning,
			Services:   []string{"webhook://httpbin.org/post"},
			Enabled:    false,
		},
	}

	for _, job := range jobs {
		_, err := scheduler.AddScheduledJob(job)
		if err != nil {
			t.Fatalf("Failed to add job %s: %v", job.Name, err)
		}
	}

	// Get all jobs
	retrievedJobs, err := scheduler.GetScheduledJobs()
	if err != nil {
		t.Fatalf("Failed to get scheduled jobs: %v", err)
	}

	if len(retrievedJobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(retrievedJobs))
	}

	// Verify job details
	for _, retrievedJob := range retrievedJobs {
		found := false
		for _, originalJob := range jobs {
			if retrievedJob.Name == originalJob.Name {
				found = true
				if retrievedJob.CronExpr != originalJob.CronExpr {
					t.Errorf("Job %s: expected cron %s, got %s", 
						retrievedJob.Name, originalJob.CronExpr, retrievedJob.CronExpr)
				}
				if retrievedJob.Enabled != originalJob.Enabled {
					t.Errorf("Job %s: expected enabled %t, got %t", 
						retrievedJob.Name, originalJob.Enabled, retrievedJob.Enabled)
				}
				break
			}
		}
		if !found {
			t.Errorf("Unexpected job found: %s", retrievedJob.Name)
		}
	}
}

func TestNotificationQueue_AddAndGetJobs(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_queue.db")
	apprise := New()

	scheduler, err := NewNotificationScheduler(dbPath, apprise)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer scheduler.Close()

	queue := scheduler.queue

	// Create test job
	job := QueuedJob{
		Title:      "Test Queue Job",
		Body:       "This is a queued job",
		NotifyType: NotifyTypeError,
		Services:   []string{"webhook://httpbin.org/post"},
		Tags:       []string{"queue", "test"},
		Priority:   5,
		MaxRetries: 3,
	}

	// Add job to queue
	result, err := queue.Add(job)
	if err != nil {
		t.Fatalf("Failed to add job to queue: %v", err)
	}

	if result.ID == 0 {
		t.Error("Expected job ID to be set")
	}

	if result.Status != string(JobStatusPending) {
		t.Errorf("Expected status pending, got %s", result.Status)
	}

	// Get pending jobs
	pendingJobs, err := queue.GetPendingJobs(10)
	if err != nil {
		t.Fatalf("Failed to get pending jobs: %v", err)
	}

	if len(pendingJobs) != 1 {
		t.Errorf("Expected 1 pending job, got %d", len(pendingJobs))
	}

	retrievedJob := pendingJobs[0]
	if retrievedJob.Title != job.Title {
		t.Errorf("Expected title %s, got %s", job.Title, retrievedJob.Title)
	}
	if retrievedJob.Priority != job.Priority {
		t.Errorf("Expected priority %d, got %d", job.Priority, retrievedJob.Priority)
	}
}

func TestNotificationQueue_JobStatusUpdates(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_queue.db")
	apprise := New()

	scheduler, err := NewNotificationScheduler(dbPath, apprise)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer scheduler.Close()

	queue := scheduler.queue

	// Add job
	job := QueuedJob{
		Title:      "Status Test Job",
		Body:       "Testing status updates",
		NotifyType: NotifyTypeInfo,
		Services:   []string{"webhook://httpbin.org/post"},
		MaxRetries: 2,
	}

	result, err := queue.Add(job)
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}
	jobID := result.ID

	// Test status progression
	testCases := []struct {
		status       JobStatus
		errorMessage string
	}{
		{JobStatusRunning, ""},
		{JobStatusRetrying, "Test retry error"},
		{JobStatusFailed, "Final failure"},
	}

	for _, tc := range testCases {
		err := queue.UpdateJobStatus(jobID, tc.status, tc.errorMessage)
		if err != nil {
			t.Fatalf("Failed to update job status to %s: %v", tc.status, err)
		}

		// Verify status was updated
		job, err := queue.getQueuedJob(jobID)
		if err != nil {
			t.Fatalf("Failed to get job after status update: %v", err)
		}

		if job.Status != string(tc.status) {
			t.Errorf("Expected status %s, got %s", tc.status, job.Status)
		}

		if tc.errorMessage != "" && job.ErrorMessage != tc.errorMessage {
			t.Errorf("Expected error message %s, got %s", tc.errorMessage, job.ErrorMessage)
		}
	}
}

func TestTemplateManager_AddAndRenderTemplate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_templates.db")
	apprise := New()

	scheduler, err := NewNotificationScheduler(dbPath, apprise)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer scheduler.Close()

	tm := scheduler.GetTemplateManager()

	// Create test template
	template := NotificationTemplate{
		Name:        "test-template",
		Title:       "Alert: {{.alert_type}}",
		Body:        "System: {{.system}}\nSeverity: {{.severity}}\nMessage: {{.message}}",
		Variables:   map[string]string{"severity": "medium"},
		Description: "Test template for alerts",
	}

	// Add template
	result, err := tm.AddTemplate(template)
	if err != nil {
		t.Fatalf("Failed to add template: %v", err)
	}

	if result.ID == 0 {
		t.Error("Expected template ID to be set")
	}

	// Render template
	variables := map[string]interface{}{
		"alert_type": "System Error",
		"system":     "web-server-01",
		"message":    "High CPU usage detected",
	}

	req, err := tm.RenderTemplate("test-template", variables)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	expectedTitle := "Alert: System Error"
	if req.Title != expectedTitle {
		t.Errorf("Expected title %s, got %s", expectedTitle, req.Title)
	}

	if !strings.Contains(req.Body, "System: web-server-01") {
		t.Error("Template body should contain system name")
	}
	if !strings.Contains(req.Body, "Severity: medium") {
		t.Error("Template body should contain default severity")
	}
	if !strings.Contains(req.Body, "High CPU usage detected") {
		t.Error("Template body should contain message")
	}
}

func TestCronExpressionBuilder(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *CronExpressionBuilder
		expected string
	}{
		{
			name: "Every 5 minutes",
			builder: func() *CronExpressionBuilder {
				return NewCronExpressionBuilder().Every(5 * time.Minute)
			},
			expected: "*/5 * * * *",
		},
		{
			name: "Daily at 9:30",
			builder: func() *CronExpressionBuilder {
				return NewCronExpressionBuilder().At(9, 30).Daily()
			},
			expected: "30 9 * * *",
		},
		{
			name: "Weekdays at 8:00",
			builder: func() *CronExpressionBuilder {
				return NewCronExpressionBuilder().At(8, 0).OnWeekdays()
			},
			expected: "0 8 * * 1-5",
		},
		{
			name: "Weekends only",
			builder: func() *CronExpressionBuilder {
				return NewCronExpressionBuilder().OnWeekends()
			},
			expected: "* * * * 0,6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.builder().Build()
			if result != tt.expected {
				t.Errorf("Expected cron expression %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestJobBuilder(t *testing.T) {
	// Test scheduled job builder
	scheduledJob := NewScheduledJobBuilder().
		WithName("test-scheduled").
		WithCron("0 * * * *").
		WithTitle("Scheduled Notification").
		WithBody("This runs on schedule").
		WithType(NotifyTypeWarning).
		WithServices("webhook://example.com/hook").
		WithTags("scheduled", "test").
		WithMetadata("environment", "test").
		WithTemplate("alert-template").
		BuildScheduled()

	if scheduledJob.Name != "test-scheduled" {
		t.Errorf("Expected name 'test-scheduled', got %s", scheduledJob.Name)
	}
	if scheduledJob.CronExpr != "0 * * * *" {
		t.Errorf("Expected cron '0 * * * *', got %s", scheduledJob.CronExpr)
	}
	if scheduledJob.NotifyType != NotifyTypeWarning {
		t.Errorf("Expected notify type warning, got %v", scheduledJob.NotifyType)
	}
	if len(scheduledJob.Services) != 1 || scheduledJob.Services[0] != "webhook://example.com/hook" {
		t.Errorf("Expected service webhook://example.com/hook, got %v", scheduledJob.Services)
	}
	if scheduledJob.Metadata["environment"] != "test" {
		t.Errorf("Expected metadata environment=test, got %v", scheduledJob.Metadata)
	}

	// Test queued job builder
	queuedJob := NewQueuedJobBuilder().
		WithTitle("Queued Notification").
		WithBody("This is in the queue").
		WithType(NotifyTypeError).
		WithServices("webhook://example.com/hook").
		WithTags("queued", "urgent").
		WithMetadata("source", "api").
		WithPriority(10).
		WithRetries(5, 2*time.Minute).
		BuildQueued()

	if queuedJob.Title != "Queued Notification" {
		t.Errorf("Expected title 'Queued Notification', got %s", queuedJob.Title)
	}
	if queuedJob.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", queuedJob.Priority)
	}
	if queuedJob.MaxRetries != 5 {
		t.Errorf("Expected max retries 5, got %d", queuedJob.MaxRetries)
	}
	if queuedJob.RetryDelay != 2*time.Minute {
		t.Errorf("Expected retry delay 2m, got %v", queuedJob.RetryDelay)
	}
}

func TestMetricsCollector(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_metrics.db")
	apprise := New()

	scheduler, err := NewNotificationScheduler(dbPath, apprise)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer scheduler.Close()

	mc := scheduler.GetMetricsCollector()

	// Record test metrics
	baseTime := time.Now().Truncate(time.Hour)
	metrics := []NotificationMetrics{
		{
			ServiceID:        "discord",
			ServiceURL:       "discord://webhook",
			NotificationType: int(NotifyTypeInfo),
			Status:           "success",
			DurationMs:       150,
			Timestamp:        baseTime,
		},
		{
			ServiceID:        "slack",
			ServiceURL:       "slack://webhook",
			NotificationType: int(NotifyTypeError),
			Status:           "failed",
			DurationMs:       5000,
			ErrorMessage:     "Connection timeout",
			Timestamp:        baseTime.Add(10 * time.Minute),
		},
		{
			ServiceID:        "discord",
			ServiceURL:       "discord://webhook",
			NotificationType: int(NotifyTypeWarning),
			Status:           "success",
			DurationMs:       200,
			Timestamp:        baseTime.Add(20 * time.Minute),
		},
	}

	for _, metric := range metrics {
		if err := mc.RecordMetrics(metric); err != nil {
			t.Fatalf("Failed to record metrics: %v", err)
		}
	}

	// Generate report
	startTime := baseTime.Add(-time.Hour)
	endTime := baseTime.Add(time.Hour)
	
	report, err := mc.GetMetricsReport(startTime, endTime)
	if err != nil {
		t.Fatalf("Failed to generate metrics report: %v", err)
	}

	// Verify report
	if report.TotalNotifications != 3 {
		t.Errorf("Expected 3 total notifications, got %d", report.TotalNotifications)
	}
	if report.SuccessfulNotifications != 2 {
		t.Errorf("Expected 2 successful notifications, got %d", report.SuccessfulNotifications)
	}
	if report.FailedNotifications != 1 {
		t.Errorf("Expected 1 failed notification, got %d", report.FailedNotifications)
	}

	expectedSuccessRate := float64(2) / float64(3) * 100
	if abs(report.SuccessRate-expectedSuccessRate) > 0.01 {
		t.Errorf("Expected success rate %.2f%%, got %.2f%%", expectedSuccessRate, report.SuccessRate)
	}

	// Check service metrics
	if len(report.ServiceMetrics) != 2 {
		t.Errorf("Expected 2 services in metrics, got %d", len(report.ServiceMetrics))
	}

	discordMetrics, exists := report.ServiceMetrics["discord"]
	if !exists {
		t.Error("Expected Discord metrics in report")
	} else {
		if discordMetrics.TotalNotifications != 2 {
			t.Errorf("Expected 2 Discord notifications, got %d", discordMetrics.TotalNotifications)
		}
		if discordMetrics.SuccessfulNotifications != 2 {
			t.Errorf("Expected 2 successful Discord notifications, got %d", discordMetrics.SuccessfulNotifications)
		}
	}

	// Check notification types
	if len(report.NotificationTypes) == 0 {
		t.Error("Expected notification types in report")
	}
}


func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func TestScheduler_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_integration.db")
	
	// Create apprise instance with webhook service for testing
	apprise := New()
	
	scheduler, err := NewNotificationScheduler(dbPath, apprise)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer scheduler.Close()

	// Start scheduler
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// Add a job that runs every minute for testing
	job := ScheduledJob{
		Name:       "integration-test",
		CronExpr:   "* * * * *", // Every minute
		Title:      "Integration Test",
		Body:       "This is an integration test notification",
		NotifyType: NotifyTypeInfo,
		Services:   []string{"webhook://httpbin.org/post"},
		Tags:       []string{"integration", "test"},
		Enabled:    true,
	}

	_, err = scheduler.AddScheduledJob(job)
	if err != nil {
		t.Fatalf("Failed to add scheduled job: %v", err)
	}

	// Wait for a few seconds to let the job run
	time.Sleep(6 * time.Second)

	// Check metrics were recorded
	mc := scheduler.GetMetricsCollector()
	startTime := time.Now().Add(-10 * time.Second)
	endTime := time.Now()
	
	report, err := mc.GetMetricsReport(startTime, endTime)
	if err != nil {
		t.Fatalf("Failed to generate metrics report: %v", err)
	}

	// Should have some notifications recorded
	if report.TotalNotifications < 1 {
		t.Errorf("Expected at least 1 notification, got %d", report.TotalNotifications)
	}

	t.Logf("Integration test completed: %d notifications processed with %.2f%% success rate", 
		report.TotalNotifications, report.SuccessRate)
}

// Cleanup test files
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Cleanup any test files if needed
	os.Exit(code)
}