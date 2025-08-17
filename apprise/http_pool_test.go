package apprise

import (
	"net/http"
	"testing"
	"time"
)

func TestHTTPClientPool_GetClient(t *testing.T) {
	pool := NewHTTPClientPool()
	config := DefaultHTTPClientConfig()

	// Test getting a new client
	client1 := pool.GetClient("test-service", config)
	if client1 == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// Test getting the same client (should be cached)
	client2 := pool.GetClient("test-service", config)
	if client1 != client2 {
		t.Error("Expected same client instance from cache")
	}

	// Test getting different client with different key
	client3 := pool.GetClient("other-service", config)
	if client1 == client3 {
		t.Error("Expected different client instances for different keys")
	}
}

func TestHTTPClientPool_CloseIdleConnections(t *testing.T) {
	pool := NewHTTPClientPool()
	config := DefaultHTTPClientConfig()

	// Create a client
	client := pool.GetClient("test-service", config)
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// This should not panic
	pool.CloseIdleConnections()
}

func TestHTTPClientPool_RemoveClient(t *testing.T) {
	pool := NewHTTPClientPool()
	config := DefaultHTTPClientConfig()

	// Create a client
	client1 := pool.GetClient("test-service", config)
	if client1 == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// Remove the client
	pool.RemoveClient("test-service")

	// Get client again (should create new one)
	client2 := pool.GetClient("test-service", config)
	if client1 == client2 {
		t.Error("Expected different client instance after removal")
	}
}

func TestDefaultHTTPClientConfig(t *testing.T) {
	config := DefaultHTTPClientConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if config.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns 100, got %d", config.MaxIdleConns)
	}

	if config.MaxConnsPerHost != 30 {
		t.Errorf("Expected MaxConnsPerHost 30, got %d", config.MaxConnsPerHost)
	}

	if config.DisableKeepAlives {
		t.Error("Expected DisableKeepAlives to be false")
	}
}

func TestCloudHTTPClientConfig(t *testing.T) {
	config := CloudHTTPClientConfig()

	if config.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", config.Timeout)
	}

	if config.MaxIdleConns != 200 {
		t.Errorf("Expected MaxIdleConns 200, got %d", config.MaxIdleConns)
	}

	if config.MaxConnsPerHost != 50 {
		t.Errorf("Expected MaxConnsPerHost 50, got %d", config.MaxConnsPerHost)
	}

	if config.MaxIdleConnsPerHost != 20 {
		t.Errorf("Expected MaxIdleConnsPerHost 20, got %d", config.MaxIdleConnsPerHost)
	}
}

func TestWebhookHTTPClientConfig(t *testing.T) {
	config := WebhookHTTPClientConfig()

	if config.Timeout != 15*time.Second {
		t.Errorf("Expected timeout 15s, got %v", config.Timeout)
	}

	if config.MaxConnsPerHost != 20 {
		t.Errorf("Expected MaxConnsPerHost 20, got %d", config.MaxConnsPerHost)
	}

	if config.MaxIdleConnsPerHost != 5 {
		t.Errorf("Expected MaxIdleConnsPerHost 5, got %d", config.MaxIdleConnsPerHost)
	}
}

func TestGetDefaultHTTPClient(t *testing.T) {
	client1 := GetDefaultHTTPClient()
	if client1 == nil {
		t.Fatal("Expected client to be non-nil")
	}

	client2 := GetDefaultHTTPClient()
	if client1 != client2 {
		t.Error("Expected same default client instance")
	}
}

func TestGetCloudHTTPClient(t *testing.T) {
	client1 := GetCloudHTTPClient("aws-sns")
	if client1 == nil {
		t.Fatal("Expected client to be non-nil")
	}

	client2 := GetCloudHTTPClient("aws-sns")
	if client1 != client2 {
		t.Error("Expected same cloud client instance for same service")
	}

	client3 := GetCloudHTTPClient("gcp-pubsub")
	if client1 == client3 {
		t.Error("Expected different cloud client instances for different services")
	}
}

func TestGetWebhookHTTPClient(t *testing.T) {
	client1 := GetWebhookHTTPClient("discord")
	if client1 == nil {
		t.Fatal("Expected client to be non-nil")
	}

	client2 := GetWebhookHTTPClient("discord")
	if client1 != client2 {
		t.Error("Expected same webhook client instance for same service")
	}

	client3 := GetWebhookHTTPClient("slack")
	if client1 == client3 {
		t.Error("Expected different webhook client instances for different services")
	}
}

func TestCloseAllIdleConnections(t *testing.T) {
	// Create some clients
	GetDefaultHTTPClient()
	GetCloudHTTPClient("test-cloud")
	GetWebhookHTTPClient("test-webhook")

	// This should not panic
	CloseAllIdleConnections()
}

func TestHTTPClientTimeout(t *testing.T) {
	pool := NewHTTPClientPool()
	config := HTTPClientConfig{
		Timeout:               5 * time.Second,
		MaxIdleConns:          10,
		MaxConnsPerHost:       5,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		InsecureSkipVerify:    false,
	}

	client := pool.GetClient("timeout-test", config)
	if client.Timeout != 5*time.Second {
		t.Errorf("Expected client timeout 5s, got %v", client.Timeout)
	}
}

func TestHTTPClientTransportConfiguration(t *testing.T) {
	pool := NewHTTPClientPool()
	config := HTTPClientConfig{
		Timeout:               10 * time.Second,
		MaxIdleConns:          50,
		MaxConnsPerHost:       10,
		MaxIdleConnsPerHost:   5,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     true,
		InsecureSkipVerify:    true,
	}

	client := pool.GetClient("transport-test", config)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected HTTP transport")
	}

	if transport.MaxIdleConns != 50 {
		t.Errorf("Expected MaxIdleConns 50, got %d", transport.MaxIdleConns)
	}

	if transport.MaxConnsPerHost != 10 {
		t.Errorf("Expected MaxConnsPerHost 10, got %d", transport.MaxConnsPerHost)
	}

	if transport.MaxIdleConnsPerHost != 5 {
		t.Errorf("Expected MaxIdleConnsPerHost 5, got %d", transport.MaxIdleConnsPerHost)
	}

	if transport.IdleConnTimeout != 60*time.Second {
		t.Errorf("Expected IdleConnTimeout 60s, got %v", transport.IdleConnTimeout)
	}

	if !transport.DisableKeepAlives {
		t.Error("Expected DisableKeepAlives to be true")
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}

	if !transport.ForceAttemptHTTP2 {
		t.Error("Expected ForceAttemptHTTP2 to be true")
	}
}

func TestHTTPClientPoolConcurrency(t *testing.T) {
	pool := NewHTTPClientPool()
	config := DefaultHTTPClientConfig()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()

			// Each goroutine gets the same client
			client := pool.GetClient("concurrent-test", config)
			if client == nil {
				t.Errorf("Goroutine %d: Expected client to be non-nil", i)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGlobalPoolInitialization(t *testing.T) {
	// Ensure pools are properly initialized
	client1 := GetDefaultHTTPClient()
	client2 := GetCloudHTTPClient("test")
	client3 := GetWebhookHTTPClient("test")

	if client1 == nil || client2 == nil || client3 == nil {
		t.Error("Expected all global pool clients to be initialized")
	}
}

func BenchmarkHTTPClientPoolGetClient(b *testing.B) {
	pool := NewHTTPClientPool()
	config := DefaultHTTPClientConfig()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.GetClient("benchmark-test", config)
		}
	})
}

func BenchmarkGlobalHTTPClientAccess(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GetDefaultHTTPClient()
		}
	})
}
