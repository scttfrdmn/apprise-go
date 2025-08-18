package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// NodeREDService implements Node-RED webhook notifications
type NodeREDService struct {
	baseURL string
	path    string
	client  *http.Client
}

// NodeREDRequest represents a Node-RED webhook request
type NodeREDRequest struct {
	Title     string `json:"title,omitempty"`
	Message   string `json:"message"`
	Type      string `json:"type,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Source    string `json:"source,omitempty"`
}

// NewNodeREDService creates a new Node-RED service instance
func NewNodeREDService() Service {
	return &NodeREDService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *NodeREDService) GetServiceID() string {
	return "nodered"
}

// GetDefaultPort returns the default port
func (s *NodeREDService) GetDefaultPort() int {
	return 1880 // Default Node-RED port
}

// ParseURL parses the service URL and configures the service
func (s *NodeREDService) ParseURL(serviceURL *url.URL) error {
	// URL format: nodered://host:port/webhook_path
	
	if serviceURL.Host == "" {
		return fmt.Errorf("Node-RED URL must specify host")
	}
	
	// Build base URL
	scheme := "http"
	if serviceURL.Scheme == "nodered+https" {
		scheme = "https"
	}
	
	port := serviceURL.Port()
	if port == "" {
		port = "1880"
	}
	
	s.baseURL = fmt.Sprintf("%s://%s:%s", scheme, serviceURL.Hostname(), port)
	
	// Extract webhook path
	if serviceURL.Path == "" || serviceURL.Path == "/" {
		s.path = "/webhook"
	} else {
		s.path = serviceURL.Path
	}
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *NodeREDService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *NodeREDService) Send(ctx context.Context, req NotificationRequest) error {
	// Send webhook to Node-RED
	return s.sendWebhook(ctx, req)
}

// sendWebhook sends a webhook to Node-RED
func (s *NodeREDService) sendWebhook(ctx context.Context, req NotificationRequest) error {
	// Node-RED webhook URL
	webhookURL := fmt.Sprintf("%s%s", s.baseURL, s.path)
	
	// Prepare payload
	payload := NodeREDRequest{
		Title:   req.Title,
		Message: req.Body,
		Type:    req.NotifyType.String(),
		Source:  "apprise-go",
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Node-RED request: %w", err)
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Node-RED request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Node-RED webhook request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Node-RED webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *NodeREDService) SupportsAttachments() bool {
	return false // Node-RED webhooks depend on flow configuration
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *NodeREDService) GetMaxBodyLength() int {
	return 0 // No specific limit for Node-RED webhooks
}