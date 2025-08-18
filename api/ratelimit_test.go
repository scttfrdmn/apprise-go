package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

func TestRateLimiter(t *testing.T) {
	config := RateLimitConfig{
		Enabled:        true,
		RequestsPerMin: 5, // Low limit for testing
		BurstSize:      2,
		WindowSize:     time.Minute,
	}

	rl := NewRateLimiter(config)
	defer rl.Stop()

	clientID := "test-client"

	t.Run("Allow requests within limit", func(t *testing.T) {
		for i := 0; i < config.RequestsPerMin; i++ {
			allowed, remaining, resetTime := rl.CheckLimit(clientID)
			if !allowed {
				t.Errorf("Request %d should be allowed", i+1)
			}
			if remaining != config.RequestsPerMin-i-1 {
				t.Errorf("Expected remaining %d, got %d", config.RequestsPerMin-i-1, remaining)
			}
			if resetTime != 0 {
				t.Errorf("Reset time should be 0 for allowed requests")
			}
		}
	})

	t.Run("Reject requests over limit", func(t *testing.T) {
		allowed, remaining, resetTime := rl.CheckLimit(clientID)
		if allowed {
			t.Error("Request over limit should be rejected")
		}
		if remaining != 0 {
			t.Errorf("Expected 0 remaining when limit exceeded, got %d", remaining)
		}
		if resetTime <= 0 {
			t.Error("Expected positive reset time for rejected request")
		}
	})

	t.Run("Reset window after expiry", func(t *testing.T) {
		// Create new rate limiter with short window for testing
		shortConfig := RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 2,
			BurstSize:      1,
			WindowSize:     100 * time.Millisecond,
		}

		shortRL := NewRateLimiter(shortConfig)
		defer shortRL.Stop()

		testClient := "short-window-client"

		// Use up the limit
		for i := 0; i < shortConfig.RequestsPerMin; i++ {
			allowed, _, _ := shortRL.CheckLimit(testClient)
			if !allowed {
				t.Errorf("Request %d should be allowed in fresh window", i+1)
			}
		}

		// Should be rejected
		allowed, _, _ := shortRL.CheckLimit(testClient)
		if allowed {
			t.Error("Request should be rejected after limit exceeded")
		}

		// Wait for window to reset
		time.Sleep(150 * time.Millisecond)

		// Should be allowed again
		allowed, _, _ = shortRL.CheckLimit(testClient)
		if !allowed {
			t.Error("Request should be allowed after window reset")
		}
	})

	t.Run("Different clients have separate limits", func(t *testing.T) {
		client1 := "client-1"
		client2 := "client-2"

		newRL := NewRateLimiter(config)
		defer newRL.Stop()

		// Use up limit for client1
		for i := 0; i < config.RequestsPerMin; i++ {
			allowed, _, _ := newRL.CheckLimit(client1)
			if !allowed {
				t.Errorf("Client1 request %d should be allowed", i+1)
			}
		}

		// Client1 should be blocked
		allowed, _, _ := newRL.CheckLimit(client1)
		if allowed {
			t.Error("Client1 should be blocked after exceeding limit")
		}

		// Client2 should still be allowed
		allowed, _, _ = newRL.CheckLimit(client2)
		if !allowed {
			t.Error("Client2 should be allowed with fresh limit")
		}
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	// Create server with rate limiting enabled
	config := &ServerConfig{
		Host:        "localhost",
		Port:        "8080",
		RequireAuth: false, // Disable auth for rate limit testing
		RateLimit: RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 3, // Very low limit for testing
			BurstSize:      1,
			WindowSize:     time.Minute,
		},
	}

	appriseInstance := apprise.New()
	server, err := NewServer(config, appriseInstance, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.rateLimiter.Stop()

	t.Run("Rate limit headers present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Header().Get("X-RateLimit-Limit") == "" {
			t.Error("Expected X-RateLimit-Limit header")
		}

		if w.Header().Get("X-RateLimit-Remaining") == "" {
			t.Error("Expected X-RateLimit-Remaining header")
		}
	})

	t.Run("Requests blocked after limit", func(t *testing.T) {
		// Make requests up to the limit
		for i := 0; i < config.RateLimit.RequestsPerMin; i++ {
			req := httptest.NewRequest("GET", "/health", nil)
			req.RemoteAddr = "192.168.1.100:12345" // Use consistent IP
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				t.Errorf("Request %d should not be rate limited yet", i+1)
			}
		}

		// Next request should be blocked
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status 429 (Too Many Requests), got %d", w.Code)
		}

		if w.Header().Get("Retry-After") == "" {
			t.Error("Expected Retry-After header when rate limited")
		}
	})

	t.Run("Rate limit disabled", func(t *testing.T) {
		// Create server with rate limiting disabled
		disabledConfig := &ServerConfig{
			Host:        "localhost",
			Port:        "8080",
			RequireAuth: false,
			RateLimit: RateLimitConfig{
				Enabled: false,
			},
		}

		disabledServer, err := NewServer(disabledConfig, appriseInstance, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		// Make many requests - should all succeed
		for i := 0; i < 20; i++ {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			disabledServer.router.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				t.Errorf("Request %d should not be rate limited when disabled", i+1)
			}
		}
	})
}

func TestGetClientID(t *testing.T) {
	server := &Server{}

	t.Run("IP-based client ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		clientID := server.getClientID(req)
		expected := "ip:192.168.1.1"

		if clientID != expected {
			t.Errorf("Expected client ID %s, got %s", expected, clientID)
		}
	})

	t.Run("API key based client ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "test-api-key")

		clientID := server.getClientID(req)
		expected := "apikey:test-api-key"

		if clientID != expected {
			t.Errorf("Expected client ID %s, got %s", expected, clientID)
		}
	})

	t.Run("X-Forwarded-For header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		clientID := server.getClientID(req)
		expected := "ip:203.0.113.1"

		if clientID != expected {
			t.Errorf("Expected client ID %s, got %s", expected, clientID)
		}
	})

	t.Run("X-Real-IP header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Real-IP", "203.0.113.2")

		clientID := server.getClientID(req)
		expected := "ip:203.0.113.2"

		if clientID != expected {
			t.Errorf("Expected client ID %s, got %s", expected, clientID)
		}
	})
}

func TestRateLimiterStats(t *testing.T) {
	config := RateLimitConfig{
		Enabled:        true,
		RequestsPerMin: 10,
		BurstSize:      3,
		WindowSize:     time.Minute,
	}

	rl := NewRateLimiter(config)
	defer rl.Stop()

	t.Run("Initial stats", func(t *testing.T) {
		stats := rl.GetStats()

		if !stats["enabled"].(bool) {
			t.Error("Expected rate limiter to be enabled")
		}

		if stats["total_clients"].(int) != 0 {
			t.Error("Expected 0 initial clients")
		}

		if stats["requests_per_min"].(int) != config.RequestsPerMin {
			t.Error("Expected correct requests per minute in stats")
		}
	})

	t.Run("Stats after requests", func(t *testing.T) {
		// Make requests from different clients
		rl.CheckLimit("client1")
		rl.CheckLimit("client2")
		rl.CheckLimit("client1")

		stats := rl.GetStats()

		if stats["total_clients"].(int) != 2 {
			t.Errorf("Expected 2 total clients, got %d", stats["total_clients"].(int))
		}
	})

	t.Run("Client status", func(t *testing.T) {
		status := rl.GetRateLimitStatus("new-client")

		if !status["enabled"].(bool) {
			t.Error("Expected rate limiting to be enabled")
		}

		if status["remaining"].(int) != config.RequestsPerMin {
			t.Errorf("Expected %d remaining for new client, got %d", config.RequestsPerMin, status["remaining"].(int))
		}
	})
}

func TestCleanupOldClients(t *testing.T) {
	config := RateLimitConfig{
		Enabled:        true,
		RequestsPerMin: 10,
		BurstSize:      3,
		WindowSize:     100 * time.Millisecond, // Short window for testing
	}

	rl := NewRateLimiter(config)
	defer rl.Stop()

	// Make request to create client
	rl.CheckLimit("test-client")

	// Initial client count
	stats := rl.GetStats()
	initialClients := stats["total_clients"].(int)

	if initialClients != 1 {
		t.Errorf("Expected 1 initial client, got %d", initialClients)
	}

	// Wait for cleanup period
	time.Sleep(300 * time.Millisecond)

	// Trigger cleanup manually for testing
	rl.cleanupStaleClients()

	stats = rl.GetStats()
	finalClients := stats["total_clients"].(int)

	if finalClients >= initialClients {
		t.Errorf("Expected cleanup to reduce client count from %d to less, got %d", initialClients, finalClients)
	}
}