package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/scttfrdmn/apprise-go/apprise"
)

// ScheduledJobRequest represents a request to create/update a scheduled job
type ScheduledJobRequest struct {
	Name       string            `json:"name"`
	CronExpr   string            `json:"cron_expr"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Type       string            `json:"type,omitempty"`
	Services   []string          `json:"services"`
	Tags       []string          `json:"tags,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Template   string            `json:"template,omitempty"`
	Enabled    bool              `json:"enabled"`
}

// QueuedJobRequest represents a request to add a job to the queue
type QueuedJobRequest struct {
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Type       string            `json:"type,omitempty"`
	Services   []string          `json:"services"`
	Tags       []string          `json:"tags,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Priority   int               `json:"priority,omitempty"`
	MaxRetries int               `json:"max_retries,omitempty"`
	RetryDelay string            `json:"retry_delay,omitempty"` // Duration string like "5m"
}

// TemplateRequest represents a request to create/update a template
type TemplateRequest struct {
	Name        string            `json:"name"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Variables   map[string]string `json:"variables,omitempty"`
	Description string            `json:"description,omitempty"`
}

// MetricsReportRequest represents a request for metrics report
type MetricsReportRequest struct {
	StartTime string `json:"start_time"` // RFC3339 format
	EndTime   string `json:"end_time"`   // RFC3339 format
}

// handleListScheduledJobs returns all scheduled jobs
func (s *Server) handleListScheduledJobs(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	jobs, err := s.scheduler.GetScheduledJobs()
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to retrieve scheduled jobs", err)
		return
	}

	s.sendSuccess(w, "Scheduled jobs retrieved", map[string]interface{}{
		"total": len(jobs),
		"jobs":  jobs,
	})
}

// handleCreateScheduledJob creates a new scheduled job
func (s *Server) handleCreateScheduledJob(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	var req ScheduledJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate required fields
	if req.Name == "" {
		s.sendError(w, http.StatusBadRequest, "Name is required", nil)
		return
	}
	if req.CronExpr == "" {
		s.sendError(w, http.StatusBadRequest, "Cron expression is required", nil)
		return
	}
	if req.Body == "" {
		s.sendError(w, http.StatusBadRequest, "Body is required", nil)
		return
	}
	if len(req.Services) == 0 {
		s.sendError(w, http.StatusBadRequest, "At least one service is required", nil)
		return
	}

	// Parse notification type
	notifyType := apprise.NotifyTypeInfo
	if req.Type != "" {
		if parsedType, err := parseNotifyType(req.Type); err == nil {
			notifyType = parsedType
		}
	}

	// Create scheduled job
	job := apprise.ScheduledJob{
		Name:       req.Name,
		CronExpr:   req.CronExpr,
		Title:      req.Title,
		Body:       req.Body,
		NotifyType: notifyType,
		Services:   req.Services,
		Tags:       req.Tags,
		Metadata:   req.Metadata,
		Template:   req.Template,
		Enabled:    req.Enabled,
	}

	createdJob, err := s.scheduler.AddScheduledJob(job)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "Failed to create scheduled job", err)
		return
	}

	s.sendSuccess(w, "Scheduled job created successfully", createdJob)
}

// handleGetScheduledJob returns a specific scheduled job
func (s *Server) handleGetScheduledJob(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	vars := mux.Vars(r)
	jobIDStr := vars["job_id"]
	
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid job ID", err)
		return
	}

	job, err := s.scheduler.GetScheduledJob(jobID)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Scheduled job not found", err)
		return
	}

	s.sendSuccess(w, "Scheduled job retrieved", job)
}

// handleUpdateScheduledJob updates an existing scheduled job
func (s *Server) handleUpdateScheduledJob(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	vars := mux.Vars(r)
	jobIDStr := vars["job_id"]
	
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid job ID", err)
		return
	}

	var req ScheduledJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get existing job
	existingJob, err := s.scheduler.GetScheduledJob(jobID)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Scheduled job not found", err)
		return
	}

	// Update fields
	if req.Name != "" {
		existingJob.Name = req.Name
	}
	if req.CronExpr != "" {
		existingJob.CronExpr = req.CronExpr
	}
	if req.Title != "" {
		existingJob.Title = req.Title
	}
	if req.Body != "" {
		existingJob.Body = req.Body
	}
	if req.Type != "" {
		if parsedType, err := parseNotifyType(req.Type); err == nil {
			existingJob.NotifyType = parsedType
		}
	}
	if len(req.Services) > 0 {
		existingJob.Services = req.Services
	}
	if req.Tags != nil {
		existingJob.Tags = req.Tags
	}
	if req.Metadata != nil {
		existingJob.Metadata = req.Metadata
	}
	if req.Template != "" {
		existingJob.Template = req.Template
	}
	existingJob.Enabled = req.Enabled

	if err := s.scheduler.UpdateScheduledJob(*existingJob); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to update scheduled job", err)
		return
	}

	s.sendSuccess(w, "Scheduled job updated successfully", existingJob)
}

// handleDeleteScheduledJob removes a scheduled job
func (s *Server) handleDeleteScheduledJob(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	vars := mux.Vars(r)
	jobIDStr := vars["job_id"]
	
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid job ID", err)
		return
	}

	if err := s.scheduler.RemoveScheduledJob(jobID); err != nil {
		s.sendError(w, http.StatusNotFound, "Failed to delete scheduled job", err)
		return
	}

	s.sendSuccess(w, "Scheduled job deleted successfully", map[string]interface{}{
		"job_id": jobID,
	})
}

// handleEnableScheduledJob enables a scheduled job
func (s *Server) handleEnableScheduledJob(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	vars := mux.Vars(r)
	jobIDStr := vars["job_id"]
	
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid job ID", err)
		return
	}

	if err := s.scheduler.EnableScheduledJob(jobID); err != nil {
		s.sendError(w, http.StatusNotFound, "Failed to enable scheduled job", err)
		return
	}

	s.sendSuccess(w, "Scheduled job enabled successfully", map[string]interface{}{
		"job_id": jobID,
	})
}

// handleDisableScheduledJob disables a scheduled job
func (s *Server) handleDisableScheduledJob(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	vars := mux.Vars(r)
	jobIDStr := vars["job_id"]
	
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid job ID", err)
		return
	}

	if err := s.scheduler.DisableScheduledJob(jobID); err != nil {
		s.sendError(w, http.StatusNotFound, "Failed to disable scheduled job", err)
		return
	}

	s.sendSuccess(w, "Scheduled job disabled successfully", map[string]interface{}{
		"job_id": jobID,
	})
}

// handleListQueuedJobs returns all queued jobs
func (s *Server) handleListQueuedJobs(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	jobs, err := s.scheduler.GetQueuedJobs(limit)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to retrieve queued jobs", err)
		return
	}

	s.sendSuccess(w, "Queued jobs retrieved", map[string]interface{}{
		"total": len(jobs),
		"limit": limit,
		"jobs":  jobs,
	})
}

// handleAddToQueue adds a job to the notification queue
func (s *Server) handleAddToQueue(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	var req QueuedJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate required fields
	if req.Body == "" {
		s.sendError(w, http.StatusBadRequest, "Body is required", nil)
		return
	}
	if len(req.Services) == 0 {
		s.sendError(w, http.StatusBadRequest, "At least one service is required", nil)
		return
	}

	// Parse notification type
	notifyType := apprise.NotifyTypeInfo
	if req.Type != "" {
		if parsedType, err := parseNotifyType(req.Type); err == nil {
			notifyType = parsedType
		}
	}

	// Parse retry delay
	var retryDelay time.Duration = 5 * time.Minute // default
	if req.RetryDelay != "" {
		if parsedDelay, err := time.ParseDuration(req.RetryDelay); err == nil {
			retryDelay = parsedDelay
		}
	}

	// Set defaults
	priority := req.Priority
	if priority == 0 {
		priority = 1
	}
	
	maxRetries := req.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	// Create queued job
	job := apprise.QueuedJob{
		Title:      req.Title,
		Body:       req.Body,
		NotifyType: notifyType,
		Services:   req.Services,
		Tags:       req.Tags,
		Metadata:   req.Metadata,
		Priority:   priority,
		MaxRetries: maxRetries,
		RetryDelay: retryDelay,
	}

	queuedJob, err := s.scheduler.QueueNotification(job)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to queue notification", err)
		return
	}

	s.sendSuccess(w, "Notification queued successfully", queuedJob)
}

// handleQueueStats returns queue statistics
func (s *Server) handleQueueStats(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	stats, err := s.scheduler.GetQueueStats()
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to retrieve queue stats", err)
		return
	}

	s.sendSuccess(w, "Queue statistics retrieved", stats)
}

// handleSchedulerMetrics returns scheduler metrics
func (s *Server) handleSchedulerMetrics(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	mc := s.scheduler.GetMetricsCollector()
	if mc == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Metrics collector not available", nil)
		return
	}

	// Return basic metrics for now
	s.sendSuccess(w, "Scheduler metrics retrieved", map[string]interface{}{
		"collector_available": true,
		"message": "Detailed metrics available via /metrics/report endpoint",
	})
}

// handleMetricsReport generates a detailed metrics report
func (s *Server) handleMetricsReport(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Scheduler not available", nil)
		return
	}

	var req MetricsReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Parse time range
	var startTime, endTime time.Time
	var err error

	if req.StartTime != "" {
		startTime, err = time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			s.sendError(w, http.StatusBadRequest, "Invalid start time format", err)
			return
		}
	} else {
		startTime = time.Now().Add(-24 * time.Hour) // default: last 24 hours
	}

	if req.EndTime != "" {
		endTime, err = time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			s.sendError(w, http.StatusBadRequest, "Invalid end time format", err)
			return
		}
	} else {
		endTime = time.Now()
	}

	mc := s.scheduler.GetMetricsCollector()
	if mc == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Metrics collector not available", nil)
		return
	}

	report, err := mc.GetMetricsReport(startTime, endTime)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to generate metrics report", err)
		return
	}

	s.sendSuccess(w, "Metrics report generated", report)
}

// Placeholder handlers for other endpoints
func (s *Server) handleGetQueuedJob(w http.ResponseWriter, r *http.Request) {
	s.sendError(w, http.StatusNotImplemented, "Not implemented yet", nil)
}

func (s *Server) handleUpdateQueuedJob(w http.ResponseWriter, r *http.Request) {
	s.sendError(w, http.StatusNotImplemented, "Not implemented yet", nil)
}

func (s *Server) handleRetryQueuedJob(w http.ResponseWriter, r *http.Request) {
	s.sendError(w, http.StatusNotImplemented, "Not implemented yet", nil)
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	s.sendError(w, http.StatusNotImplemented, "Not implemented yet", nil)
}

func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	s.sendError(w, http.StatusNotImplemented, "Not implemented yet", nil)
}

func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	s.sendError(w, http.StatusNotImplemented, "Not implemented yet", nil)
}

func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	s.sendError(w, http.StatusNotImplemented, "Not implemented yet", nil)
}

func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	s.sendError(w, http.StatusNotImplemented, "Not implemented yet", nil)
}

func (s *Server) handleRenderTemplate(w http.ResponseWriter, r *http.Request) {
	s.sendError(w, http.StatusNotImplemented, "Not implemented yet", nil)
}