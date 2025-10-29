package apprise

import (
	"context"
	"fmt"
	"net/url"
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
	Title         string
	Body          string
	NotifyType    NotifyType
	Attachments   []Attachment       // Legacy attachment support
	AttachmentMgr *AttachmentManager // Modern attachment support
	Tags          []string
	BodyFormat    string // html, markdown, text
	URL           string // The service URL that will handle this notification
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
	metrics       *MetricsManager
}

// New creates a new Apprise instance
func New() *Apprise {
	registry := NewServiceRegistry()

	// Register built-in services
	registerBuiltinServices(registry)

	metrics := NewMetricsManager("apprise")
	metrics.Register()

	return &Apprise{
		services:      make([]Service, 0),
		registry:      registry,
		timeout:       30 * time.Second,
		attachmentMgr: NewAttachmentManager(),
		metrics:       metrics,
	}
}

// registerBuiltinServices registers all the services that are part of the core library.
func registerBuiltinServices(r *ServiceRegistry) {
	// Register the Bark service for both 'bark' and 'barks' schemes.
	// The factory returns a new, unconfigured instance of the service.
	// The configuration is then handled by the ParseURL method.
	barkFactory := func() Service { return &BarkService{} }
	r.Register("bark", barkFactory)
	r.Register("barks", barkFactory)

	// Other built-in services would be registered here as well.
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

	// Update metrics
	a.metrics.UpdateServicesConfigured(len(a.services))

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

			// Record metrics
			status := "success"
			if err != nil {
				status = "failed"
				a.metrics.RecordNotificationError(svc.GetServiceID(), "send_failed", "unknown")
			}
			a.metrics.RecordNotification(svc.GetServiceID(), req.NotifyType.String(), status, duration)
		}(i, service)
	}

	wg.Wait()

	// Record batch size
	a.metrics.RecordBatchSize(len(a.services))

	return responses
}