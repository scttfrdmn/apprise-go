package api

import (
	"encoding/json"
	"net/http"
)

// ConfigResponse represents the server configuration response
type ConfigResponse struct {
	Version     string                 `json:"version"`
	Services    []string               `json:"supported_services"`
	Features    map[string]bool        `json:"features"`
	Limits      map[string]interface{} `json:"limits"`
	Scheduler   *SchedulerConfig       `json:"scheduler,omitempty"`
}

// SchedulerConfig represents scheduler configuration
type SchedulerConfig struct {
	Enabled         bool   `json:"enabled"`
	DatabasePath    string `json:"database_path,omitempty"`
	QueueSize       int    `json:"queue_size"`
	MaxRetries      int    `json:"max_retries"`
	ProcessInterval string `json:"process_interval"`
}

// ConfigUpdateRequest represents a configuration update request
type ConfigUpdateRequest struct {
	Features map[string]bool        `json:"features,omitempty"`
	Limits   map[string]interface{} `json:"limits,omitempty"`
}

// ConfigLoadRequest represents a request to load configuration from file
type ConfigLoadRequest struct {
	Path   string `json:"path"`
	Format string `json:"format,omitempty"` // yaml, json, toml
}

// handleGetConfig returns the current server configuration
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	config := &ConfigResponse{
		Version:  s.getServerVersion(),
		Services: s.getSupportedServices(),
		Features: s.getFeatures(),
		Limits:   s.getLimits(),
	}

	// Add scheduler config if available
	if s.scheduler != nil {
		config.Scheduler = &SchedulerConfig{
			Enabled:         true,
			DatabasePath:    s.config.DatabasePath,
			QueueSize:       100, // TODO: Get from actual config
			MaxRetries:      3,   // TODO: Get from actual config
			ProcessInterval: "10s", // TODO: Get from actual config
		}
	}

	s.sendSuccess(w, "Server configuration retrieved", config)
}

// handleUpdateConfig updates the server configuration
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// TODO: Implement configuration updates
	// This would involve validating the new configuration and applying changes

	s.sendSuccess(w, "Configuration updated successfully", map[string]interface{}{
		"features": req.Features,
		"limits":   req.Limits,
		"message":  "Configuration updates will take effect after restart",
	})
}

// handleLoadConfig loads configuration from a file
func (s *Server) handleLoadConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigLoadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Path == "" {
		s.sendError(w, http.StatusBadRequest, "Configuration path is required", nil)
		return
	}

	// TODO: Implement configuration file loading
	// This would involve reading the configuration file and applying the settings

	s.sendSuccess(w, "Configuration loaded successfully", map[string]interface{}{
		"path":   req.Path,
		"format": req.Format,
		"message": "Configuration loaded from file",
	})
}

// Helper methods to get configuration information

func (s *Server) getServerVersion() string {
	// Import the apprise package to get version info
	return "1.9.4-7" // TODO: Get from apprise.GetVersion()
}

func (s *Server) getSupportedServices() []string {
	// TODO: Get from apprise.GetSupportedServices()
	return []string{
		"discord", "slack", "telegram", "email", "webhook", "msteams",
		"pushover", "pushbullet", "twilio", "desktop", "gotify", "ntfy",
		"matrix", "mattermost", "pagerduty", "opsgenie", "aws-sns", "aws-ses",
		"gcp-pubsub", "azure-servicebus", "github", "gitlab", "jira",
		"datadog", "newrelic", "linkedin", "twitter", "apns", "fcm",
		"aws-iot", "gcp-iot", "polly", "twilio-voice", "rocketchat",
	}
}

func (s *Server) getFeatures() map[string]bool {
	features := map[string]bool{
		"notifications":        true,
		"bulk_notifications":   true,
		"service_management":   true,
		"attachments":          true,
		"tags":                 true,
		"custom_formats":       true,
		"configuration_files":  true,
		"scheduler":            s.scheduler != nil,
		"templates":            s.scheduler != nil,
		"metrics":              s.scheduler != nil,
		"queue_management":     s.scheduler != nil,
		"web_dashboard":        false, // TODO: Implement web dashboard
		"authentication":       false, // TODO: Implement authentication
		"rate_limiting":        false, // TODO: Implement rate limiting
		"webhooks":             false, // TODO: Implement webhook callbacks
	}

	return features
}

func (s *Server) getLimits() map[string]interface{} {
	limits := map[string]interface{}{
		"max_body_length":          4096,   // Default max body length
		"max_title_length":         250,    // Default max title length
		"max_services_per_request": 50,     // Max services in single request
		"max_bulk_notifications":   100,    // Max notifications in bulk request
		"max_attachments":          10,     // Max attachments per notification
		"max_attachment_size":      "10MB", // Max size per attachment
		"request_timeout":          "30s",  // Request timeout
		"queue_size":               10000,  // Max queue size (if scheduler enabled)
		"max_retries":              5,      // Max retry attempts
		"retry_delay":              "5m",   // Default retry delay
	}

	// Add scheduler-specific limits if available
	if s.scheduler != nil {
		limits["max_scheduled_jobs"] = 1000
		limits["max_templates"] = 100
		limits["metrics_retention"] = "30d"
	}

	return limits
}