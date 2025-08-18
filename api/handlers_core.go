package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

// NotificationRequest represents a notification request from the API
type NotificationRequest struct {
	URLs     []string          `json:"urls,omitempty"`
	Title    string            `json:"title,omitempty"`
	Body     string            `json:"body"`
	Type     string            `json:"type,omitempty"`
	Format   string            `json:"format,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// BulkNotificationRequest represents multiple notification requests
type BulkNotificationRequest struct {
	Notifications []NotificationRequest `json:"notifications"`
}

// ServiceInfo represents service information
type ServiceInfo struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	URL               string            `json:"url,omitempty"`
	Tags              []string          `json:"tags,omitempty"`
	Enabled           bool              `json:"enabled"`
	SupportsAttachments bool            `json:"supports_attachments"`
	MaxBodyLength     int               `json:"max_body_length"`
	Config            map[string]string `json:"config,omitempty"`
}

// handleRoot serves the root endpoint with API information
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"name":        "Apprise API Server",
		"version":     apprise.GetVersion(),
		"description": "REST API for Apprise-Go notification library",
		"endpoints": map[string]string{
			"dashboard":     "/dashboard",
			"health":        "/health",
			"version":       "/version",
			"documentation": "/docs",
			"api_v1":        "/api/v1",
		},
	}
	s.sendSuccess(w, "Apprise API Server", info)
}

// handleHealth provides health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"version":   apprise.GetVersion(),
		"scheduler": s.scheduler != nil,
		"services":  len(apprise.GetSupportedServices()),
	}
	s.sendSuccess(w, "API server is healthy", health)
}

// handleVersion provides version information
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	versionInfo := apprise.GetVersionInfo()
	s.sendSuccess(w, "Version information", versionInfo)
}

// handleMetrics provides basic metrics endpoint
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"services_registered": len(apprise.GetSupportedServices()),
		"version":            apprise.GetVersion(),
		"scheduler_enabled":   s.scheduler != nil,
	}
	
	if s.scheduler != nil {
		// Add scheduler-specific metrics if available
		if mc := s.scheduler.GetMetricsCollector(); mc != nil {
			// TODO: Add scheduler metrics
		}
	}
	
	s.sendSuccess(w, "Server metrics", metrics)
}

// handleDocs serves API documentation
func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	docs := `<!DOCTYPE html>
<html>
<head>
    <title>Apprise API Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { margin: 20px 0; padding: 15px; border-left: 3px solid #007cba; background: #f5f5f5; }
        .method { font-weight: bold; color: #007cba; }
        code { background: #e8e8e8; padding: 2px 4px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>Apprise API Documentation</h1>
    <p>Version: ` + apprise.GetVersion() + `</p>
    
    <h2>Core Endpoints</h2>
    
    <div class="endpoint">
        <div class="method">POST /api/v1/notify</div>
        <p>Send a single notification</p>
        <p><strong>Body:</strong> <code>{"body": "message", "urls": ["service://..."], "title": "optional"}</code></p>
    </div>
    
    <div class="endpoint">
        <div class="method">POST /api/v1/notify/bulk</div>
        <p>Send multiple notifications</p>
        <p><strong>Body:</strong> <code>{"notifications": [...]}</code></p>
    </div>
    
    <div class="endpoint">
        <div class="method">GET /api/v1/services</div>
        <p>List all configured services</p>
    </div>
    
    <div class="endpoint">
        <div class="method">POST /api/v1/services</div>
        <p>Add a new service</p>
        <p><strong>Body:</strong> <code>{"url": "service://...", "tags": ["optional"]}</code></p>
    </div>`

	if s.scheduler != nil {
		docs += `
    <h2>Scheduler Endpoints</h2>
    
    <div class="endpoint">
        <div class="method">GET /api/v1/scheduler/jobs</div>
        <p>List scheduled jobs</p>
    </div>
    
    <div class="endpoint">
        <div class="method">POST /api/v1/scheduler/jobs</div>
        <p>Create a scheduled job</p>
    </div>
    
    <div class="endpoint">
        <div class="method">GET /api/v1/scheduler/queue</div>
        <p>List queued notifications</p>
    </div>`
	}

	docs += `
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(docs))
}

// handleNotify processes a single notification request
func (s *Server) handleNotify(w http.ResponseWriter, r *http.Request) {
	var req NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Body == "" {
		s.sendError(w, http.StatusBadRequest, "Body is required", nil)
		return
	}

	// Create temporary Apprise instance for this request
	tempApprise := apprise.New()

	// Add services from URLs
	for _, url := range req.URLs {
		if err := tempApprise.Add(url); err != nil {
			s.sendError(w, http.StatusBadRequest, "Invalid service URL: "+url, err)
			return
		}
	}

	// Parse notification type
	notifyType := apprise.NotifyTypeInfo
	if req.Type != "" {
		if parsedType, err := parseNotifyType(req.Type); err == nil {
			notifyType = parsedType
		}
	}

	// Create notification request
	notification := apprise.NotificationRequest{
		Title:      req.Title,
		Body:       req.Body,
		NotifyType: notifyType,
		Tags:       req.Tags,
		BodyFormat: req.Format,
	}

	// Send notifications
	responses := tempApprise.NotifyAll(notification)

	// Process results
	successful := 0
	var errors []string
	for _, resp := range responses {
		if resp.Success {
			successful++
		} else if resp.Error != nil {
			errors = append(errors, resp.Error.Error())
		}
	}

	result := map[string]interface{}{
		"total":      len(responses),
		"successful": successful,
		"failed":     len(responses) - successful,
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	if successful == len(responses) {
		s.sendSuccess(w, "All notifications sent successfully", result)
	} else if successful > 0 {
		s.sendSuccess(w, "Some notifications sent successfully", result)
	} else {
		s.sendError(w, http.StatusInternalServerError, "All notifications failed", nil)
		// Still include the result data
		response := APIResponse{
			Success:   false,
			Message:   "All notifications failed",
			Data:      result,
			Timestamp: s.getCurrentTime(),
		}
		s.sendJSON(w, http.StatusInternalServerError, response)
		return
	}
}

// handleBulkNotify processes multiple notification requests
func (s *Server) handleBulkNotify(w http.ResponseWriter, r *http.Request) {
	var req BulkNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if len(req.Notifications) == 0 {
		s.sendError(w, http.StatusBadRequest, "No notifications provided", nil)
		return
	}

	results := make([]map[string]interface{}, len(req.Notifications))

	for i, notification := range req.Notifications {
		// Create temporary Apprise instance for this notification
		tempApprise := apprise.New()

		// Add services from URLs
		for _, url := range notification.URLs {
			if err := tempApprise.Add(url); err != nil {
				results[i] = map[string]interface{}{
					"success": false,
					"error":   "Invalid service URL: " + url,
				}
				continue
			}
		}

		// Parse notification type
		notifyType := apprise.NotifyTypeInfo
		if notification.Type != "" {
			if parsedType, err := parseNotifyType(notification.Type); err == nil {
				notifyType = parsedType
			}
		}

		// Create notification request
		notificationReq := apprise.NotificationRequest{
			Title:      notification.Title,
			Body:       notification.Body,
			NotifyType: notifyType,
			Tags:       notification.Tags,
			BodyFormat: notification.Format,
		}

		// Send notifications
		responses := tempApprise.NotifyAll(notificationReq)

		// Process results
		successful := 0
		var errors []string
		for _, resp := range responses {
			if resp.Success {
				successful++
			} else if resp.Error != nil {
				errors = append(errors, resp.Error.Error())
			}
		}

		results[i] = map[string]interface{}{
			"total":      len(responses),
			"successful": successful,
			"failed":     len(responses) - successful,
		}

		if len(errors) > 0 {
			results[i]["errors"] = errors
		}
	}

	s.sendSuccess(w, "Bulk notifications processed", map[string]interface{}{
		"notifications": len(req.Notifications),
		"results":       results,
	})
}

// Helper function to parse notification type
func parseNotifyType(typeStr string) (apprise.NotifyType, error) {
	switch strings.ToLower(typeStr) {
	case "info":
		return apprise.NotifyTypeInfo, nil
	case "success":
		return apprise.NotifyTypeSuccess, nil
	case "warning", "warn":
		return apprise.NotifyTypeWarning, nil
	case "error", "failure":
		return apprise.NotifyTypeError, nil
	default:
		return apprise.NotifyTypeInfo, nil
	}
}

// Helper to get current time (for testing override)
func (s *Server) getCurrentTime() time.Time {
	return time.Now()
}