package apprise

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

// HTTPClientPool manages shared HTTP clients with connection pooling
type HTTPClientPool struct {
	clients    map[string]*http.Client
	transports map[string]*http.Transport
	mu         sync.RWMutex
}

// HTTPClientConfig represents configuration for HTTP client creation
type HTTPClientConfig struct {
	Timeout              time.Duration
	MaxIdleConns         int
	MaxConnsPerHost      int
	MaxIdleConnsPerHost  int
	IdleConnTimeout      time.Duration
	TLSHandshakeTimeout  time.Duration
	ExpectContinueTimeout time.Duration
	DisableKeepAlives    bool
	InsecureSkipVerify   bool
}

// DefaultHTTPClientConfig returns default HTTP client configuration
func DefaultHTTPClientConfig() HTTPClientConfig {
	return HTTPClientConfig{
		Timeout:               30 * time.Second,
		MaxIdleConns:          100,
		MaxConnsPerHost:       30,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		InsecureSkipVerify:    false,
	}
}

// CloudHTTPClientConfig returns optimized configuration for cloud services
func CloudHTTPClientConfig() HTTPClientConfig {
	return HTTPClientConfig{
		Timeout:               60 * time.Second, // Longer timeout for cloud APIs
		MaxIdleConns:          200,              // More connections for cloud scale
		MaxConnsPerHost:       50,               // Higher per-host limit
		MaxIdleConnsPerHost:   20,               // More idle connections per host
		IdleConnTimeout:       120 * time.Second, // Longer idle timeout
		TLSHandshakeTimeout:   15 * time.Second, // Longer TLS handshake
		ExpectContinueTimeout: 2 * time.Second,
		DisableKeepAlives:     false,
		InsecureSkipVerify:    false,
	}
}

// WebhookHTTPClientConfig returns configuration optimized for webhooks
func WebhookHTTPClientConfig() HTTPClientConfig {
	return HTTPClientConfig{
		Timeout:               15 * time.Second, // Faster timeout for webhooks
		MaxIdleConns:          50,
		MaxConnsPerHost:       20,
		MaxIdleConnsPerHost:   5,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		InsecureSkipVerify:    false,
	}
}

var (
	// Global HTTP client pools
	defaultPool  *HTTPClientPool
	cloudPool    *HTTPClientPool
	webhookPool  *HTTPClientPool
	poolInitOnce sync.Once
)

// initHTTPPools initializes the global HTTP client pools
func initHTTPPools() {
	poolInitOnce.Do(func() {
		defaultPool = NewHTTPClientPool()
		cloudPool = NewHTTPClientPool()
		webhookPool = NewHTTPClientPool()
	})
}

// NewHTTPClientPool creates a new HTTP client pool
func NewHTTPClientPool() *HTTPClientPool {
	return &HTTPClientPool{
		clients:    make(map[string]*http.Client),
		transports: make(map[string]*http.Transport),
	}
}

// GetClient returns a cached HTTP client or creates a new one with the given configuration
func (p *HTTPClientPool) GetClient(key string, config HTTPClientConfig) *http.Client {
	p.mu.RLock()
	client, exists := p.clients[key]
	p.mu.RUnlock()
	
	if exists {
		return client
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Double-check pattern
	if client, exists := p.clients[key]; exists {
		return client
	}
	
	// Create new transport
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          config.MaxIdleConns,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		DisableKeepAlives:     config.DisableKeepAlives,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
		// Enable HTTP/2
		ForceAttemptHTTP2: true,
	}
	
	// Create new client
	client = &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
	
	p.clients[key] = client
	p.transports[key] = transport
	
	return client
}

// CloseIdleConnections closes idle connections for all clients in the pool
func (p *HTTPClientPool) CloseIdleConnections() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	for _, transport := range p.transports {
		transport.CloseIdleConnections()
	}
}

// RemoveClient removes a client from the pool and closes its idle connections
func (p *HTTPClientPool) RemoveClient(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if transport, exists := p.transports[key]; exists {
		transport.CloseIdleConnections()
		delete(p.transports, key)
	}
	
	delete(p.clients, key)
}

// GetDefaultHTTPClient returns a default HTTP client from the global pool
func GetDefaultHTTPClient() *http.Client {
	initHTTPPools()
	return defaultPool.GetClient("default", DefaultHTTPClientConfig())
}

// GetCloudHTTPClient returns a cloud-optimized HTTP client from the global pool
func GetCloudHTTPClient(service string) *http.Client {
	initHTTPPools()
	return cloudPool.GetClient(service, CloudHTTPClientConfig())
}

// GetWebhookHTTPClient returns a webhook-optimized HTTP client from the global pool
func GetWebhookHTTPClient(service string) *http.Client {
	initHTTPPools()
	return webhookPool.GetClient(service, WebhookHTTPClientConfig())
}

// CloseAllIdleConnections closes idle connections for all global pools
func CloseAllIdleConnections() {
	initHTTPPools()
	defaultPool.CloseIdleConnections()
	cloudPool.CloseIdleConnections()
	webhookPool.CloseIdleConnections()
}