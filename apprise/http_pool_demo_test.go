package apprise

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// This file demonstrates the benefits of HTTP connection pooling

func TestHTTPConnectionPoolingDemo(t *testing.T) {
	// Create a test server that tracks connections
	var connectionCount int64
	var mu sync.Mutex
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		connectionCount++
		mu.Unlock()
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()
	
	// Test with pooled client (reused connections)
	pooledClient := GetDefaultHTTPClient()
	var pooledCount int64
	
	// Make multiple concurrent requests with pooled client
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := pooledClient.Get(server.URL)
			if err == nil {
				resp.Body.Close()
				mu.Lock()
				pooledCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	
	t.Logf("Connection pooling demo completed successfully")
	t.Logf("Made %d requests with pooled client", pooledCount)
}

func TestHTTPPoolPerformanceComparison(t *testing.T) {
	// This test demonstrates performance benefits of connection pooling
	// by comparing pooled vs non-pooled HTTP clients
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(1 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()
	
	const requestCount = 50
	
	// Test with pooled client
	pooledClient := GetDefaultHTTPClient()
	pooledStart := time.Now()
	var wg sync.WaitGroup
	
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := pooledClient.Get(server.URL)
			if err == nil {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()
	pooledDuration := time.Since(pooledStart)
	
	// Test with individual clients (no pooling)
	individualStart := time.Now()
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Get(server.URL)
			if err == nil {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()
	individualDuration := time.Since(individualStart)
	
	t.Logf("Performance comparison:")
	t.Logf("  Pooled client:     %v for %d requests", pooledDuration, requestCount)
	t.Logf("  Individual clients: %v for %d requests", individualDuration, requestCount)
	
	// Connection pooling should generally be faster, but we don't enforce it
	// since performance can vary based on system conditions
	if pooledDuration < individualDuration {
		t.Logf("✓ Pooled client was faster by %v", individualDuration-pooledDuration)
	} else {
		t.Logf("ℹ Individual clients were faster by %v (system dependent)", pooledDuration-individualDuration)
	}
}

func TestHTTPPoolResourceSharing(t *testing.T) {
	// Demonstrate that clients are properly shared within pools
	
	// Get cloud clients for different services
	awsClient1 := GetCloudHTTPClient("aws-sns")
	awsClient2 := GetCloudHTTPClient("aws-sns")
	
	if awsClient1 != awsClient2 {
		t.Error("Expected same client instance for same service")
	}
	
	azureClient := GetCloudHTTPClient("azure-servicebus")
	if awsClient1 == azureClient {
		t.Error("Expected different client instances for different services")
	}
	
	// Get webhook clients
	discordClient1 := GetWebhookHTTPClient("discord")
	discordClient2 := GetWebhookHTTPClient("discord")
	
	if discordClient1 != discordClient2 {
		t.Error("Expected same webhook client instance for same service")
	}
	
	slackClient := GetWebhookHTTPClient("slack")
	if discordClient1 == slackClient {
		t.Error("Expected different webhook client instances for different services")
	}
	
	t.Log("✓ HTTP client resource sharing is working correctly")
}

func TestHTTPPoolIdleConnectionManagement(t *testing.T) {
	// Test idle connection management
	pool := NewHTTPClientPool()
	config := DefaultHTTPClientConfig()
	
	// Create multiple clients
	client1 := pool.GetClient("service1", config)
	client2 := pool.GetClient("service2", config)
	
	if client1 == nil || client2 == nil {
		t.Fatal("Failed to create clients")
	}
	
	// Close idle connections (should not panic)
	pool.CloseIdleConnections()
	
	// Remove a client
	pool.RemoveClient("service1")
	
	// Get the client again (should create new instance)
	client3 := pool.GetClient("service1", config)
	
	if client1 == client3 {
		t.Error("Expected new client instance after removal")
	}
	
	t.Log("✓ Idle connection management is working correctly")
}

func BenchmarkServiceCreationWithPooling(b *testing.B) {
	// Benchmark service creation with connection pooling
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Create services that use connection pooling
			_ = NewDiscordService()
			_ = NewSlackService()
			_ = NewMSTeamsService()
			_ = NewAWSSNSService()
		}
	})
}

func TestHTTPPoolConfigurationValidation(t *testing.T) {
	// Test different HTTP client configurations
	configs := map[string]HTTPClientConfig{
		"default": DefaultHTTPClientConfig(),
		"cloud":   CloudHTTPClientConfig(),
		"webhook": WebhookHTTPClientConfig(),
	}
	
	for name, config := range configs {
		t.Run(name, func(t *testing.T) {
			if config.Timeout <= 0 {
				t.Errorf("Invalid timeout for %s config: %v", name, config.Timeout)
			}
			
			if config.MaxIdleConns <= 0 {
				t.Errorf("Invalid MaxIdleConns for %s config: %d", name, config.MaxIdleConns)
			}
			
			if config.MaxConnsPerHost <= 0 {
				t.Errorf("Invalid MaxConnsPerHost for %s config: %d", name, config.MaxConnsPerHost)
			}
			
			if config.IdleConnTimeout <= 0 {
				t.Errorf("Invalid IdleConnTimeout for %s config: %v", name, config.IdleConnTimeout)
			}
			
			t.Logf("✓ %s configuration is valid", name)
		})
	}
}

func ExampleHTTPClientPool() {
	// Example of using HTTP connection pooling
	
	// Get optimized clients for different service types
	cloudClient := GetCloudHTTPClient("my-cloud-service")
	webhookClient := GetWebhookHTTPClient("my-webhook-service")
	defaultClient := GetDefaultHTTPClient()
	
	fmt.Printf("Cloud client timeout: %v\n", cloudClient.Timeout)
	fmt.Printf("Webhook client timeout: %v\n", webhookClient.Timeout)
	fmt.Printf("Default client timeout: %v\n", defaultClient.Timeout)
	
	// Clean up idle connections when needed
	CloseAllIdleConnections()
	
	// Output:
	// Cloud client timeout: 1m0s
	// Webhook client timeout: 15s
	// Default client timeout: 30s
}