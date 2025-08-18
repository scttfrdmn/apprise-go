package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// HomeAssistantService implements Home Assistant notifications
type HomeAssistantService struct {
	baseURL     string
	accessToken string
	service     string
	client      *http.Client
}

// HomeAssistantRequest represents a Home Assistant service call request
type HomeAssistantRequest struct {
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
	Target  string `json:"target,omitempty"`
}

// NewHomeAssistantService creates a new Home Assistant service instance
func NewHomeAssistantService() Service {
	return &HomeAssistantService{
		client:  &http.Client{},
		service: "persistent_notification.create", // Default service
	}
}

// GetServiceID returns the service identifier
func (s *HomeAssistantService) GetServiceID() string {
	return "homeassistant"
}

// GetDefaultPort returns the default port
func (s *HomeAssistantService) GetDefaultPort() int {
	return 8123 // Default Home Assistant port
}

// ParseURL parses the service URL and configures the service
func (s *HomeAssistantService) ParseURL(serviceURL *url.URL) error {
	// URL format: homeassistant://access_token@host:port/domain/service?target=entity
	
	if serviceURL.User == nil {
		return fmt.Errorf("Home Assistant URL must include access token")
	}
	
	s.accessToken = serviceURL.User.Username()
	if s.accessToken == "" {
		return fmt.Errorf("Home Assistant access token cannot be empty")
	}
	
	if serviceURL.Hostname() == "" {
		return fmt.Errorf("Home Assistant URL must include host")
	}
	
	// Build base URL
	scheme := "http"
	if serviceURL.Scheme == "homeassistant+https" || serviceURL.Port() == "443" {
		scheme = "https"
	}
	
	port := serviceURL.Port()
	if port == "" {
		port = "8123"
	}
	
	s.baseURL = fmt.Sprintf("%s://%s:%s", scheme, serviceURL.Hostname(), port)
	
	// Extract service from path
	if serviceURL.Path != "" && serviceURL.Path != "/" {
		// Path format: /domain/service
		pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		if len(pathParts) >= 2 {
			s.service = fmt.Sprintf("%s.%s", pathParts[0], pathParts[1])
		}
	}
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *HomeAssistantService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *HomeAssistantService) Send(ctx context.Context, req NotificationRequest) error {
	// Call Home Assistant service
	return s.callService(ctx, req)
}

// callService calls a Home Assistant service
func (s *HomeAssistantService) callService(ctx context.Context, req NotificationRequest) error {
	// Home Assistant API endpoint
	apiURL := fmt.Sprintf("%s/api/services/%s", s.baseURL, strings.Replace(s.service, ".", "/", 1))
	
	// Prepare service data
	payload := HomeAssistantRequest{
		Title:   req.Title,
		Message: req.Body,
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Home Assistant request: %w", err)
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Home Assistant request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.accessToken)
	
	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Home Assistant API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Home Assistant API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *HomeAssistantService) SupportsAttachments() bool {
	return false // Home Assistant service calls don't typically support attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *HomeAssistantService) GetMaxBodyLength() int {
	return 0 // No specific limit for Home Assistant service calls
}