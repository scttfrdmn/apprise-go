package apprise

import (
	"context"
	"testing"
	"time"
)

func TestAppriseErrorHandling(t *testing.T) {
	app := New()
	
	// Test adding invalid services
	invalidServices := []string{
		"invalid://scheme",
		"discord://missing_token",
		"slack://incomplete",
		"mailto://missing@host",
		"tgram://no_chat_id",
		"pushover://incomplete",
	}
	
	for _, invalidService := range invalidServices {
		err := app.Add(invalidService)
		if err == nil {
			t.Errorf("Should have failed to add invalid service: %s", invalidService)
		}
		
		// Ensure app count remains 0
		if app.Count() != 0 {
			t.Errorf("App should have 0 services after failed add, got %d", app.Count())
		}
	}
}

func TestNotificationTimeouts(t *testing.T) {
	app := New()
	app.SetTimeout(100 * time.Millisecond) // Very short timeout
	
	// Add a service that will timeout (non-routable IP)
	err := app.Add("webhook://192.0.2.1:1234/notify") // RFC5737 test IP
	if err != nil {
		t.Fatalf("Failed to add webhook service: %v", err)
	}
	
	start := time.Now()
	responses := app.Notify("Test", "Timeout test", NotifyTypeInfo)
	duration := time.Since(start)
	
	// Should complete within reasonable time (allowing for timeout + overhead)
	if duration > 5*time.Second {
		t.Errorf("Notification took too long: %v", duration)
	}
	
	if len(responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(responses))
	}
	
	response := responses[0]
	if response.Success {
		t.Error("Expected timeout to cause failure")
	}
	
	if response.Error == nil {
		t.Error("Expected error for timeout")
	}
}

func TestPartialFailures(t *testing.T) {
	app := New()
	
	// Add mix of valid and invalid services
	validService := "discord://valid_id/valid_token"
	invalidService := "webhook://192.0.2.1:1234/notify" // Will timeout/fail
	
	app.Add(validService)   // This will parse successfully
	app.Add(invalidService) // This will also parse successfully but fail on send
	
	if app.Count() != 2 {
		t.Fatalf("Expected 2 services, got %d", app.Count())
	}
	
	// Set short timeout to make webhook fail quickly
	app.SetTimeout(100 * time.Millisecond)
	
	responses := app.Notify("Test", "Partial failure test", NotifyTypeInfo)
	
	if len(responses) != 2 {
		t.Errorf("Expected 2 responses, got %d", len(responses))
	}
	
	// Both should fail (Discord with network error, webhook with timeout)
	successCount := 0
	for _, response := range responses {
		if response.Success {
			successCount++
		}
	}
	
	// We expect both to fail since we're using dummy URLs
	if successCount > 0 {
		t.Logf("Unexpectedly got %d successful sends (probably network-dependent)", successCount)
	}
}

func TestEmptyNotification(t *testing.T) {
	app := New()
	app.Add("discord://test_id/test_token")
	
	// Test empty title and body
	responses := app.Notify("", "", NotifyTypeInfo)
	
	if len(responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(responses))
	}
	
	// Should still attempt to send even with empty content
	response := responses[0]
	if response.Error == nil {
		t.Error("Expected network error for dummy Discord URL")
	}
}

func TestLargeNotificationBody(t *testing.T) {
	app := New()
	
	// Create a very large message
	largeBody := make([]byte, 10000) // 10KB message
	for i := range largeBody {
		largeBody[i] = 'A'
	}
	
	// Add services with different body length limits
	app.Add("discord://test_id/test_token")   // 2000 char limit
	app.Add("pushover://test_token@test_user") // 1024 char limit
	app.Add("webhook://example.com/notify")    // No limit
	
	responses := app.Notify("Large Body Test", string(largeBody), NotifyTypeInfo)
	
	if len(responses) != 3 {
		t.Errorf("Expected 3 responses, got %d", len(responses))
	}
	
	// All should attempt to send (truncation/limits handled by services)
	for i, response := range responses {
		if response.ServiceID == "" {
			t.Errorf("Response %d should have service ID", i)
		}
		
		// Expect network errors since these are dummy URLs
		if response.Error == nil {
			t.Errorf("Response %d: Expected network error for dummy URL", i)
		}
	}
}

func TestConcurrentNotifications(t *testing.T) {
	app := New()
	
	// Add multiple services
	services := []string{
		"discord://id1/token1",
		"slack://token1/token2/token3",
		"webhook://api1.example.com/notify",
		"webhook://api2.example.com/notify",
		"webhook://api3.example.com/notify",
	}
	
	for _, service := range services {
		app.Add(service)
	}
	
	if app.Count() != 5 {
		t.Fatalf("Expected 5 services, got %d", app.Count())
	}
	
	// Set reasonable timeout
	app.SetTimeout(1 * time.Second)
	
	start := time.Now()
	responses := app.Notify("Concurrent Test", "Testing concurrent sends", NotifyTypeInfo)
	duration := time.Since(start)
	
	// Should complete in roughly the timeout period (not 5x timeout)
	// due to concurrent execution
	if duration > 3*time.Second {
		t.Errorf("Concurrent notifications took too long: %v", duration)
	}
	
	if len(responses) != 5 {
		t.Errorf("Expected 5 responses, got %d", len(responses))
	}
	
	// Verify all responses have service IDs
	for i, response := range responses {
		if response.ServiceID == "" {
			t.Errorf("Response %d missing service ID", i)
		}
		
		if response.Duration <= 0 {
			t.Errorf("Response %d has invalid duration: %v", i, response.Duration)
		}
	}
}

func TestServiceClear(t *testing.T) {
	app := New()
	
	// Add multiple services
	app.Add("discord://id/token")
	app.Add("slack://a/b/c")
	app.Add("webhook://example.com/notify")
	
	if app.Count() != 3 {
		t.Fatalf("Expected 3 services before clear, got %d", app.Count())
	}
	
	// Clear all services
	app.Clear()
	
	if app.Count() != 0 {
		t.Errorf("Expected 0 services after clear, got %d", app.Count())
	}
	
	// Should be able to send with no services (returns empty responses)
	responses := app.Notify("Test", "Should return empty", NotifyTypeInfo)
	
	if len(responses) != 0 {
		t.Errorf("Expected 0 responses with no services, got %d", len(responses))
	}
}

func TestInvalidNotifyOptions(t *testing.T) {
	app := New()
	app.Add("discord://test_id/test_token")
	
	// Test with various notify options
	responses := app.Notify("Test", "Testing options", NotifyTypeInfo,
		WithTags("non-existent-tag"), // Service doesn't have this tag
		WithBodyFormat("invalid-format"), // Invalid format
	)
	
	if len(responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(responses))
	}
	
	// Should still attempt to send despite invalid options
	response := responses[0]
	if response.ServiceID != "discord" {
		t.Errorf("Expected discord service ID, got %q", response.ServiceID)
	}
}

func TestContextCancellation(t *testing.T) {
	app := New()
	app.Add("webhook://httpbin.org/delay/10") // Will take 10 seconds
	
	// Create a context that cancels after 100ms
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	start := time.Now()
	
	// Manually create notification request to use custom context
	req := NotificationRequest{
		Title:      "Context Test",
		Body:       "This should be cancelled",
		NotifyType: NotifyTypeInfo,
	}
	
	// Get the service and test cancellation
	if len(app.services) > 0 {
		err := app.services[0].Send(ctx, req)
		duration := time.Since(start)
		
		// Should complete quickly due to context cancellation
		if duration > 2*time.Second {
			t.Errorf("Context cancellation took too long: %v", duration)
		}
		
		if err == nil {
			t.Error("Expected context cancellation error")
		}
	}
}