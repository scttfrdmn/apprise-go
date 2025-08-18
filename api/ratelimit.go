package api

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled        bool          `json:"enabled"`
	RequestsPerMin int           `json:"requests_per_minute"`
	BurstSize      int           `json:"burst_size"`
	WindowSize     time.Duration `json:"window_size"`
}

// ClientInfo tracks rate limiting info for a client
type ClientInfo struct {
	RequestCount int
	WindowStart  time.Time
	LastRequest  time.Time
	mutex        sync.RWMutex
}

// RateLimiter manages rate limiting for clients
type RateLimiter struct {
	config  RateLimitConfig
	clients map[string]*ClientInfo
	mutex   sync.RWMutex
	
	// Cleanup timer
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:      config,
		clients:     make(map[string]*ClientInfo),
		stopCleanup: make(chan bool),
	}

	// Start cleanup goroutine
	if config.Enabled {
		rl.cleanupTicker = time.NewTicker(5 * time.Minute)
		go rl.cleanup()
	}

	return rl
}

// Stop stops the rate limiter and cleanup goroutine
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
		rl.stopCleanup <- true
	}
}

// cleanup removes stale client entries
func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.cleanupStaleClients()
		case <-rl.stopCleanup:
			return
		}
	}
}

// cleanupStaleClients removes clients that haven't made requests recently
func (rl *RateLimiter) cleanupStaleClients() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	cutoff := time.Now().Add(-rl.config.WindowSize * 2)
	for clientID, client := range rl.clients {
		client.mutex.RLock()
		shouldDelete := client.LastRequest.Before(cutoff)
		client.mutex.RUnlock()

		if shouldDelete {
			delete(rl.clients, clientID)
		}
	}
}

// CheckLimit checks if a client has exceeded rate limits
func (rl *RateLimiter) CheckLimit(clientID string) (bool, int, time.Duration) {
	if !rl.config.Enabled {
		return true, rl.config.RequestsPerMin, 0
	}

	rl.mutex.Lock()
	client, exists := rl.clients[clientID]
	if !exists {
		client = &ClientInfo{
			RequestCount: 0,
			WindowStart:  time.Now(),
			LastRequest:  time.Now(),
		}
		rl.clients[clientID] = client
	}
	rl.mutex.Unlock()

	client.mutex.Lock()
	defer client.mutex.Unlock()

	now := time.Now()
	
	// Reset window if expired
	if now.Sub(client.WindowStart) >= rl.config.WindowSize {
		client.RequestCount = 0
		client.WindowStart = now
	}

	// Check if limit exceeded
	if client.RequestCount >= rl.config.RequestsPerMin {
		resetTime := client.WindowStart.Add(rl.config.WindowSize)
		return false, rl.config.RequestsPerMin - client.RequestCount, resetTime.Sub(now)
	}

	// Increment request count
	client.RequestCount++
	client.LastRequest = now

	return true, rl.config.RequestsPerMin - client.RequestCount, 0
}

// RateLimitMiddleware provides rate limiting functionality
func (s *Server) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.rateLimiter == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Get client identifier
		clientID := s.getClientID(r)
		
		// Check rate limit
		allowed, remaining, resetTime := s.rateLimiter.CheckLimit(clientID)
		
		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(s.rateLimiter.config.RequestsPerMin))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		if resetTime > 0 {
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(resetTime).Unix(), 10))
		}

		if !allowed {
			w.Header().Set("Retry-After", strconv.FormatInt(int64(resetTime.Seconds()), 10))
			s.sendError(w, http.StatusTooManyRequests, "Rate limit exceeded", 
				fmt.Errorf("too many requests, try again in %v", resetTime.Round(time.Second)))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientID extracts a unique identifier for the client
func (s *Server) getClientID(r *http.Request) string {
	// Try to get authenticated user first
	if user, ok := GetUserFromContext(r.Context()); ok {
		return "user:" + user.ID
	}

	// Check for API key
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return "apikey:" + apiKey
	}

	// Fall back to IP address
	ip := s.getClientIP(r)
	return "ip:" + ip
}

// getClientIP extracts the real client IP address
func (s *Server) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (from load balancer/proxy)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in case of multiple IPs
		ips := []string{}
		for _, ip := range []string{xff} {
			if netIP := net.ParseIP(ip); netIP != nil {
				ips = append(ips, ip)
			}
		}
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		if netIP := net.ParseIP(xri); netIP != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// GetRateLimitStatus returns current rate limit status for a client
func (rl *RateLimiter) GetRateLimitStatus(clientID string) map[string]interface{} {
	if !rl.config.Enabled {
		return map[string]interface{}{
			"enabled":    false,
			"limit":      0,
			"remaining":  0,
			"reset_time": nil,
		}
	}

	rl.mutex.RLock()
	client, exists := rl.clients[clientID]
	rl.mutex.RUnlock()

	if !exists {
		return map[string]interface{}{
			"enabled":    true,
			"limit":      rl.config.RequestsPerMin,
			"remaining":  rl.config.RequestsPerMin,
			"reset_time": nil,
		}
	}

	client.mutex.RLock()
	defer client.mutex.RUnlock()

	now := time.Now()
	var resetTime *time.Time
	remaining := rl.config.RequestsPerMin - client.RequestCount

	// Calculate reset time if in current window
	if now.Sub(client.WindowStart) < rl.config.WindowSize {
		rt := client.WindowStart.Add(rl.config.WindowSize)
		resetTime = &rt
	}

	return map[string]interface{}{
		"enabled":    true,
		"limit":      rl.config.RequestsPerMin,
		"remaining":  remaining,
		"reset_time": resetTime,
		"window_size": rl.config.WindowSize.String(),
	}
}

// GetStats returns overall rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	activeClients := 0
	totalRequests := 0
	
	for _, client := range rl.clients {
		client.mutex.RLock()
		if time.Since(client.LastRequest) < rl.config.WindowSize {
			activeClients++
		}
		totalRequests += client.RequestCount
		client.mutex.RUnlock()
	}

	return map[string]interface{}{
		"enabled":        rl.config.Enabled,
		"total_clients":  len(rl.clients),
		"active_clients": activeClients,
		"requests_per_min": rl.config.RequestsPerMin,
		"burst_size":     rl.config.BurstSize,
		"window_size":    rl.config.WindowSize.String(),
	}
}