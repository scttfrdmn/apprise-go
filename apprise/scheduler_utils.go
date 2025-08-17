package apprise

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// parseJSONField safely parses a JSON field into the target interface
func parseJSONField(jsonStr string, target interface{}) error {
	if jsonStr == "" || jsonStr == "null" {
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), target)
}

// BatchScheduler provides convenience methods for batch operations
type BatchScheduler struct {
	scheduler *NotificationScheduler
}

// NewBatchScheduler creates a new batch scheduler
func NewBatchScheduler(scheduler *NotificationScheduler) *BatchScheduler {
	return &BatchScheduler{scheduler: scheduler}
}

// ScheduleBatch schedules multiple jobs at once
func (bs *BatchScheduler) ScheduleBatch(jobs []ScheduledJob) ([]*ScheduledJob, []error) {
	results := make([]*ScheduledJob, len(jobs))
	errors := make([]error, len(jobs))

	for i, job := range jobs {
		result, err := bs.scheduler.AddScheduledJob(job)
		results[i] = result
		errors[i] = err
	}

	return results, errors
}

// QueueBatch queues multiple notifications at once
func (bs *BatchScheduler) QueueBatch(jobs []QueuedJob) ([]*QueuedJob, []error) {
	results := make([]*QueuedJob, len(jobs))
	errors := make([]error, len(jobs))

	for i, job := range jobs {
		result, err := bs.scheduler.QueueNotification(job)
		results[i] = result
		errors[i] = err
	}

	return results, errors
}

// SchedulerConfig represents configuration for the scheduler
type SchedulerConfig struct {
	DatabasePath        string        `json:"database_path" yaml:"database_path"`
	ProcessingInterval  time.Duration `json:"processing_interval" yaml:"processing_interval"`
	BatchSize          int           `json:"batch_size" yaml:"batch_size"`
	MaxRetries         int           `json:"max_retries" yaml:"max_retries"`
	DefaultRetryDelay  time.Duration `json:"default_retry_delay" yaml:"default_retry_delay"`
	MetricsRetention   time.Duration `json:"metrics_retention" yaml:"metrics_retention"`
	CleanupInterval    time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`
	EnableMetrics      bool          `json:"enable_metrics" yaml:"enable_metrics"`
}

// DefaultSchedulerConfig returns a default scheduler configuration
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		DatabasePath:       "./apprise_scheduler.db",
		ProcessingInterval: 10 * time.Second,
		BatchSize:          10,
		MaxRetries:         3,
		DefaultRetryDelay:  5 * time.Minute,
		MetricsRetention:   30 * 24 * time.Hour, // 30 days
		CleanupInterval:    24 * time.Hour,      // Daily cleanup
		EnableMetrics:      true,
	}
}

// CronExpressionBuilder helps build cron expressions
type CronExpressionBuilder struct {
	minutes string
	hours   string
	dayOfMonth string
	month   string
	dayOfWeek string
}

// NewCronExpressionBuilder creates a new cron expression builder
func NewCronExpressionBuilder() *CronExpressionBuilder {
	return &CronExpressionBuilder{
		minutes:    "*",
		hours:      "*",
		dayOfMonth: "*",
		month:      "*",
		dayOfWeek:  "*",
	}
}

// Every sets up a recurring schedule
func (c *CronExpressionBuilder) Every(interval time.Duration) *CronExpressionBuilder {
	switch {
	case interval < time.Hour:
		minutes := int(interval.Minutes())
		c.minutes = fmt.Sprintf("*/%d", minutes)
	case interval < 24*time.Hour:
		hours := int(interval.Hours())
		c.hours = fmt.Sprintf("*/%d", hours)
		c.minutes = "0"
	}
	return c
}

// At sets a specific time
func (c *CronExpressionBuilder) At(hour, minute int) *CronExpressionBuilder {
	c.hours = fmt.Sprintf("%d", hour)
	c.minutes = fmt.Sprintf("%d", minute)
	return c
}

// OnDays sets specific days of the week (0 = Sunday)
func (c *CronExpressionBuilder) OnDays(days ...int) *CronExpressionBuilder {
	dayStrs := make([]string, len(days))
	for i, day := range days {
		dayStrs[i] = fmt.Sprintf("%d", day)
	}
	c.dayOfWeek = strings.Join(dayStrs, ",")
	return c
}

// OnWeekdays sets schedule for weekdays only
func (c *CronExpressionBuilder) OnWeekdays() *CronExpressionBuilder {
	c.dayOfWeek = "1-5" // Monday to Friday
	return c
}

// OnWeekends sets schedule for weekends only
func (c *CronExpressionBuilder) OnWeekends() *CronExpressionBuilder {
	c.dayOfWeek = "0,6" // Sunday and Saturday
	return c
}

// Daily sets a daily schedule
func (c *CronExpressionBuilder) Daily() *CronExpressionBuilder {
	c.dayOfWeek = "*"
	return c
}

// Build builds the cron expression
func (c *CronExpressionBuilder) Build() string {
	return fmt.Sprintf("%s %s %s %s %s",
		c.minutes, c.hours, c.dayOfMonth, c.month, c.dayOfWeek)
}

// Common cron expression helpers
func CronEveryMinute() string {
	return "* * * * *"
}

func CronEvery5Minutes() string {
	return "*/5 * * * *"
}

func CronEvery15Minutes() string {
	return "*/15 * * * *"
}

func CronEvery30Minutes() string {
	return "*/30 * * * *"
}

func CronHourly() string {
	return "0 * * * *"
}

func CronDaily(hour, minute int) string {
	return fmt.Sprintf("%d %d * * *", minute, hour)
}

func CronWeekly(dayOfWeek, hour, minute int) string {
	return fmt.Sprintf("%d %d * * %d", minute, hour, dayOfWeek)
}

func CronMonthly(dayOfMonth, hour, minute int) string {
	return fmt.Sprintf("%d %d %d * *", minute, hour, dayOfMonth)
}

// NotificationBuilder helps build notification requests
type NotificationBuilder struct {
	req NotificationRequest
}

// NewNotificationBuilder creates a new notification builder
func NewNotificationBuilder() *NotificationBuilder {
	return &NotificationBuilder{
		req: NotificationRequest{
			NotifyType: NotifyTypeInfo,
			Tags:       []string{},
		},
	}
}

// WithTitle sets the notification title
func (nb *NotificationBuilder) WithTitle(title string) *NotificationBuilder {
	nb.req.Title = title
	return nb
}

// WithBody sets the notification body
func (nb *NotificationBuilder) WithBody(body string) *NotificationBuilder {
	nb.req.Body = body
	return nb
}

// WithType sets the notification type
func (nb *NotificationBuilder) WithType(notifyType NotifyType) *NotificationBuilder {
	nb.req.NotifyType = notifyType
	return nb
}

// WithTags adds tags to the notification
func (nb *NotificationBuilder) WithTags(tags ...string) *NotificationBuilder {
	nb.req.Tags = append(nb.req.Tags, tags...)
	return nb
}

// WithBodyFormat sets the body format
func (nb *NotificationBuilder) WithBodyFormat(format string) *NotificationBuilder {
	nb.req.BodyFormat = format
	return nb
}

// WithURL sets the URL for the notification
func (nb *NotificationBuilder) WithURL(url string) *NotificationBuilder {
	nb.req.URL = url
	return nb
}

// Build builds the notification request
func (nb *NotificationBuilder) Build() NotificationRequest {
	return nb.req
}

// JobBuilder helps build scheduled and queued jobs
type JobBuilder struct {
	scheduledJob *ScheduledJob
	queuedJob    *QueuedJob
}

// NewScheduledJobBuilder creates a builder for scheduled jobs
func NewScheduledJobBuilder() *JobBuilder {
	return &JobBuilder{
		scheduledJob: &ScheduledJob{
			NotifyType: NotifyTypeInfo,
			Services:   []string{},
			Tags:       []string{},
			Metadata:   make(map[string]string),
			Enabled:    true,
		},
	}
}

// NewQueuedJobBuilder creates a builder for queued jobs
func NewQueuedJobBuilder() *JobBuilder {
	return &JobBuilder{
		queuedJob: &QueuedJob{
			NotifyType: NotifyTypeInfo,
			Services:   []string{},
			Tags:       []string{},
			Metadata:   make(map[string]string),
			Priority:   1,
			MaxRetries: 3,
			RetryDelay: 5 * time.Minute,
		},
	}
}

// WithName sets the job name (scheduled jobs only)
func (jb *JobBuilder) WithName(name string) *JobBuilder {
	if jb.scheduledJob != nil {
		jb.scheduledJob.Name = name
	}
	return jb
}

// WithCron sets the cron expression (scheduled jobs only)
func (jb *JobBuilder) WithCron(cronExpr string) *JobBuilder {
	if jb.scheduledJob != nil {
		jb.scheduledJob.CronExpr = cronExpr
	}
	return jb
}

// WithTitle sets the notification title
func (jb *JobBuilder) WithTitle(title string) *JobBuilder {
	if jb.scheduledJob != nil {
		jb.scheduledJob.Title = title
	}
	if jb.queuedJob != nil {
		jb.queuedJob.Title = title
	}
	return jb
}

// WithBody sets the notification body
func (jb *JobBuilder) WithBody(body string) *JobBuilder {
	if jb.scheduledJob != nil {
		jb.scheduledJob.Body = body
	}
	if jb.queuedJob != nil {
		jb.queuedJob.Body = body
	}
	return jb
}

// WithType sets the notification type
func (jb *JobBuilder) WithType(notifyType NotifyType) *JobBuilder {
	if jb.scheduledJob != nil {
		jb.scheduledJob.NotifyType = notifyType
	}
	if jb.queuedJob != nil {
		jb.queuedJob.NotifyType = notifyType
	}
	return jb
}

// WithServices sets the services to notify
func (jb *JobBuilder) WithServices(services ...string) *JobBuilder {
	if jb.scheduledJob != nil {
		jb.scheduledJob.Services = services
	}
	if jb.queuedJob != nil {
		jb.queuedJob.Services = services
	}
	return jb
}

// WithTags sets the tags
func (jb *JobBuilder) WithTags(tags ...string) *JobBuilder {
	if jb.scheduledJob != nil {
		jb.scheduledJob.Tags = tags
	}
	if jb.queuedJob != nil {
		jb.queuedJob.Tags = tags
	}
	return jb
}

// WithMetadata adds metadata
func (jb *JobBuilder) WithMetadata(key, value string) *JobBuilder {
	if jb.scheduledJob != nil {
		jb.scheduledJob.Metadata[key] = value
	}
	if jb.queuedJob != nil {
		jb.queuedJob.Metadata[key] = value
	}
	return jb
}

// WithTemplate sets the template (scheduled jobs only)
func (jb *JobBuilder) WithTemplate(template string) *JobBuilder {
	if jb.scheduledJob != nil {
		jb.scheduledJob.Template = template
	}
	return jb
}

// WithPriority sets the priority (queued jobs only)
func (jb *JobBuilder) WithPriority(priority int) *JobBuilder {
	if jb.queuedJob != nil {
		jb.queuedJob.Priority = priority
	}
	return jb
}

// WithRetries sets the retry configuration (queued jobs only)
func (jb *JobBuilder) WithRetries(maxRetries int, retryDelay time.Duration) *JobBuilder {
	if jb.queuedJob != nil {
		jb.queuedJob.MaxRetries = maxRetries
		jb.queuedJob.RetryDelay = retryDelay
	}
	return jb
}

// BuildScheduled builds a scheduled job
func (jb *JobBuilder) BuildScheduled() *ScheduledJob {
	return jb.scheduledJob
}

// BuildQueued builds a queued job
func (jb *JobBuilder) BuildQueued() *QueuedJob {
	return jb.queuedJob
}


// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NewTimeRange creates a new time range
func NewTimeRange(start, end time.Time) TimeRange {
	return TimeRange{Start: start, End: end}
}

// Last24Hours returns a time range for the last 24 hours
func Last24Hours() TimeRange {
	now := time.Now()
	return TimeRange{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	}
}

// Last7Days returns a time range for the last 7 days
func Last7Days() TimeRange {
	now := time.Now()
	return TimeRange{
		Start: now.Add(-7 * 24 * time.Hour),
		End:   now,
	}
}

// Last30Days returns a time range for the last 30 days
func Last30Days() TimeRange {
	now := time.Now()
	return TimeRange{
		Start: now.Add(-30 * 24 * time.Hour),
		End:   now,
	}
}

// ThisMonth returns a time range for the current month
func ThisMonth() TimeRange {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	return TimeRange{
		Start: start,
		End:   now,
	}
}

// SystemVariables provides system variables for templates
func SystemVariables() map[string]interface{} {
	now := time.Now()
	return map[string]interface{}{
		"timestamp":     now.Format(time.RFC3339),
		"unix_time":     now.Unix(),
		"date":          now.Format("2006-01-02"),
		"time":          now.Format("15:04:05"),
		"year":          now.Year(),
		"month":         now.Month(),
		"day":           now.Day(),
		"hour":          now.Hour(),
		"minute":        now.Minute(),
		"second":        now.Second(),
		"weekday":       now.Weekday().String(),
		"timezone":      now.Location().String(),
	}
}

// TemplateHelpers provides helper functions for templates
func TemplateHelpers() template.FuncMap {
	return template.FuncMap{
		"default": func(defaultValue interface{}, value interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
		},
		"formatTime": func(format string, t time.Time) string {
			return t.Format(format)
		},
		"now": time.Now,
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int { 
			if b == 0 { return 0 }
			return a / b 
		},
	}
}