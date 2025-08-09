package apprise

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// NotifyType represents the type of notification
type NotifyType int

const (
	NotifyTypeInfo NotifyType = iota
	NotifyTypeSuccess
	NotifyTypeWarning
	NotifyTypeError
)

func (nt NotifyType) String() string {
	switch nt {
	case NotifyTypeInfo:
		return "info"
	case NotifyTypeSuccess:
		return "success"
	case NotifyTypeWarning:
		return "warning"
	case NotifyTypeError:
		return "error"
	default:
		return "info"
	}
}

// Attachment represents a file or URL attachment (legacy)
// Deprecated: Use AttachmentInterface and AttachmentManager instead
type Attachment struct {
	URL         string
	LocalPath   string
	Name        string
	ContentType string
	Data        []byte
}

// NotificationRequest contains all the data for a notification
type NotificationRequest struct {
	Title        string
	Body         string
	NotifyType   NotifyType
	Attachments  []Attachment              // Legacy attachment support
	AttachmentMgr *AttachmentManager       // Modern attachment support
	Tags         []string
	BodyFormat   string // html, markdown, text
	URL          string // The service URL that will handle this notification
}

// NotificationResponse contains the result of a notification attempt
type NotificationResponse struct {
	ServiceURL string
	Success    bool
	Error      error
	Duration   time.Duration
	ServiceID  string
}

// Service interface that all notification services must implement
type Service interface {
	// GetServiceID returns a unique identifier for this service type
	GetServiceID() string
	
	// GetDefaultPort returns the default port for this service
	GetDefaultPort() int
	
	// ParseURL parses a service URL and configures the service
	ParseURL(serviceURL *url.URL) error
	
	// Send sends a notification and returns the result
	Send(ctx context.Context, req NotificationRequest) error
	
	// TestURL validates that a service URL is properly formatted
	TestURL(serviceURL string) error
	
	// SupportsAttachments returns true if this service supports file attachments
	SupportsAttachments() bool
	
	// GetMaxBodyLength returns max body length (0 = unlimited)
	GetMaxBodyLength() int
}

// ServiceRegistry manages available notification services
type ServiceRegistry struct {
	services map[string]func() Service
	mu       sync.RWMutex
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]func() Service),
	}
}

// Register adds a service factory to the registry
func (r *ServiceRegistry) Register(serviceID string, factory func() Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[serviceID] = factory
}

// Create creates a new service instance by service ID
func (r *ServiceRegistry) Create(serviceID string) (Service, error) {
	r.mu.RLock()
	factory, exists := r.services[serviceID]
	r.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("unknown service: %s", serviceID)
	}
	
	return factory(), nil
}

// GetSupportedServices returns a list of supported service IDs
func (r *ServiceRegistry) GetSupportedServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	services := make([]string, 0, len(r.services))
	for serviceID := range r.services {
		services = append(services, serviceID)
	}
	return services
}

// Apprise is the main notification manager
type Apprise struct {
	services      []Service
	registry      *ServiceRegistry
	timeout       time.Duration
	tags          []string
	attachmentMgr *AttachmentManager
}

// New creates a new Apprise instance
func New() *Apprise {
	registry := NewServiceRegistry()
	
	// Register built-in services
	registerBuiltinServices(registry)
	
	return &Apprise{
		services:      make([]Service, 0),
		registry:      registry,
		timeout:       30 * time.Second,
		attachmentMgr: NewAttachmentManager(),
	}
}

// Add adds a notification service by URL
func (a *Apprise) Add(serviceURL string, tags ...string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid service URL: %w", err)
	}
	
	service, err := a.registry.Create(parsedURL.Scheme)
	if err != nil {
		return err
	}
	
	if err := service.ParseURL(parsedURL); err != nil {
		return fmt.Errorf("failed to configure service: %w", err)
	}
	
	a.services = append(a.services, service)
	return nil
}

// Notify sends a notification to all configured services
func (a *Apprise) Notify(title, body string, notifyType NotifyType, options ...NotifyOption) []NotificationResponse {
	req := NotificationRequest{
		Title:         title,
		Body:          body,
		NotifyType:    notifyType,
		Tags:          a.tags,
		AttachmentMgr: a.attachmentMgr,
	}
	
	// Apply options
	for _, option := range options {
		option(&req)
	}
	
	return a.NotifyAll(req)
}

// NotifyAll sends a notification request to all services
func (a *Apprise) NotifyAll(req NotificationRequest) []NotificationResponse {
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()
	
	responses := make([]NotificationResponse, len(a.services))
	var wg sync.WaitGroup
	
	for i, service := range a.services {
		wg.Add(1)
		go func(idx int, svc Service) {
			defer wg.Done()
			
			start := time.Now()
			err := svc.Send(ctx, req)
			duration := time.Since(start)
			
			responses[idx] = NotificationResponse{
				ServiceURL: req.URL,
				Success:    err == nil,
				Error:      err,
				Duration:   duration,
				ServiceID:  svc.GetServiceID(),
			}
		}(i, service)
	}
	
	wg.Wait()
	return responses
}

// SetTimeout sets the timeout for notification requests
func (a *Apprise) SetTimeout(timeout time.Duration) {
	a.timeout = timeout
}

// SetTags sets default tags for all notifications
func (a *Apprise) SetTags(tags ...string) {
	a.tags = tags
}

// Clear removes all configured services
func (a *Apprise) Clear() {
	a.services = a.services[:0]
}

// Count returns the number of configured services
func (a *Apprise) Count() int {
	return len(a.services)
}

// AddAttachment adds an attachment from a file path or URL
func (a *Apprise) AddAttachment(source string, name ...string) error {
	return a.attachmentMgr.Add(source, name...)
}

// AddAttachmentData adds an attachment from raw data
func (a *Apprise) AddAttachmentData(data []byte, filename, mimeType string) error {
	return a.attachmentMgr.AddData(data, filename, mimeType)
}

// GetAttachments returns all attachments
func (a *Apprise) GetAttachments() []AttachmentInterface {
	return a.attachmentMgr.GetAll()
}

// AttachmentCount returns the number of attachments
func (a *Apprise) AttachmentCount() int {
	return a.attachmentMgr.Count()
}

// ClearAttachments removes all attachments
func (a *Apprise) ClearAttachments() {
	a.attachmentMgr.Clear()
}

// GetAttachmentManager returns the attachment manager for advanced operations
func (a *Apprise) GetAttachmentManager() *AttachmentManager {
	return a.attachmentMgr
}

// NotifyOption allows customization of notification requests
type NotifyOption func(*NotificationRequest)

// WithAttachments adds attachments to the notification
func WithAttachments(attachments ...Attachment) NotifyOption {
	return func(req *NotificationRequest) {
		req.Attachments = append(req.Attachments, attachments...)
	}
}

// WithTags adds tags to the notification
func WithTags(tags ...string) NotifyOption {
	return func(req *NotificationRequest) {
		req.Tags = append(req.Tags, tags...)
	}
}

// WithBodyFormat sets the body format (html, markdown, text)
func WithBodyFormat(format string) NotifyOption {
	return func(req *NotificationRequest) {
		req.BodyFormat = format
	}
}

// Helper function to extract scheme from URL
func extractScheme(serviceURL string) string {
	if idx := strings.Index(serviceURL, "://"); idx != -1 {
		return serviceURL[:idx]
	}
	return ""
}

// registerBuiltinServices registers all built-in notification services
func registerBuiltinServices(registry *ServiceRegistry) {
	// Messaging/Chat platforms
	registry.Register("discord", func() Service { return NewDiscordService() })
	registry.Register("slack", func() Service { return NewSlackService() })
	registry.Register("telegram", func() Service { return NewTelegramService() })
	registry.Register("tgram", func() Service { return NewTelegramService() })
	
	// Email services
	registry.Register("mailto", func() Service { return NewEmailService() })
	registry.Register("mailtos", func() Service { return NewEmailService() })
	
	// Webhook services
	registry.Register("webhook", func() Service { return NewWebhookService() })
	registry.Register("webhooks", func() Service { return NewWebhookService() })
	registry.Register("json", func() Service { return NewJSONService() })
	
	// Push notification services
	registry.Register("pushover", func() Service { return NewPushoverService() })
	registry.Register("pover", func() Service { return NewPushoverService() })
	registry.Register("pushbullet", func() Service { return NewPushbulletService() })
	registry.Register("pball", func() Service { return NewPushbulletService() })
	
	// Enterprise messaging
	registry.Register("msteams", func() Service { return NewMSTeamsService() })
	
	// SMS services
	registry.Register("twilio", func() Service { return NewTwilioService() })
	
	// Self-hosted services
	registry.Register("gotify", func() Service { return NewGotifyService() })
	registry.Register("gotifys", func() Service { return NewGotifyService() })
	
	// Desktop notification services
	registry.Register("desktop", func() Service { return NewDesktopService() })
	registry.Register("macosx", func() Service { return NewDesktopService() })
	registry.Register("windows", func() Service { return NewDesktopService() })
	registry.Register("linux", func() Service { return NewDesktopService() })
	registry.Register("dbus", func() Service { return NewLinuxDBusService() })
	registry.Register("gnome", func() Service { return NewLinuxDBusService() })
	registry.Register("kde", func() Service { return NewLinuxDBusService() })
	registry.Register("glib", func() Service { return NewLinuxDBusService() })
	registry.Register("qt", func() Service { return NewLinuxDBusService() })
	
	// Add more services as needed...
}