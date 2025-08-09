package apprise

import (
	"context"
	"fmt"
	"net/url"
)

// Stub implementations for services that aren't implemented yet

func NewSlackService() Service {
	return &stubService{id: "slack"}
}

func NewTelegramService() Service {
	return &stubService{id: "telegram"}
}

func NewEmailService() Service {
	return &stubService{id: "email"}
}

func NewWebhookService() Service {
	return &stubService{id: "webhook"}
}

func NewJSONService() Service {
	return &stubService{id: "json"}
}

func NewGotifyService() Service {
	return &stubService{id: "gotify"}
}

// stubService is a placeholder implementation for services not yet implemented
type stubService struct {
	id string
}

func (s *stubService) GetServiceID() string {
	return s.id
}

func (s *stubService) GetDefaultPort() int {
	return 443
}

func (s *stubService) ParseURL(serviceURL *url.URL) error {
	return fmt.Errorf("service %s is not yet implemented", s.id)
}

func (s *stubService) Send(ctx context.Context, req NotificationRequest) error {
	return fmt.Errorf("service %s is not yet implemented", s.id)
}

func (s *stubService) TestURL(serviceURL string) error {
	return fmt.Errorf("service %s is not yet implemented", s.id)
}

func (s *stubService) SupportsAttachments() bool {
	return false
}

func (s *stubService) GetMaxBodyLength() int {
	return 0
}