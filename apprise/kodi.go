package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const (
	kodiDefaultPort    = 8080
	kodiDefaultPortTLS = 443
	kodiRPCPath        = "/jsonrpc"
	kodiDisplayTime    = 5000 // milliseconds
)

// KodiService implements Kodi/XBMC media center notifications via JSON-RPC
type KodiService struct {
	hostname string
	port     int
	username string
	password string
	secure   bool // kodis:// for HTTPS
	client   *http.Client
}

// KodiRPCRequest represents a JSON-RPC request
type KodiRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	ID      int                    `json:"id"`
}

// KodiRPCResponse represents a JSON-RPC response
type KodiRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	ID int `json:"id"`
}

// NewKodiService creates a new Kodi service instance
func NewKodiService() Service {
	return &KodiService{
		client: GetDefaultHTTPClient(),
		port:   -1,
	}
}

func (k *KodiService) GetServiceID() string      { return "kodi" }
func (k *KodiService) SupportsAttachments() bool { return false }
func (k *KodiService) GetMaxBodyLength() int     { return 250 }

func (k *KodiService) GetDefaultPort() int {
	if k.secure {
		return kodiDefaultPortTLS
	}
	return kodiDefaultPort
}

// ParseURL parses a Kodi service URL
// Format: kodi://user:pass@hostname[:port]/
// Format: kodis://user:pass@hostname[:port]/  (HTTPS)
// Credentials are optional
func (k *KodiService) ParseURL(serviceURL *url.URL) error {
	switch serviceURL.Scheme {
	case "kodis":
		k.secure = true
	case "kodi":
		k.secure = false
	default:
		return fmt.Errorf("invalid scheme: expected 'kodi' or 'kodis', got '%s'", serviceURL.Scheme)
	}

	k.hostname = serviceURL.Hostname()
	if k.hostname == "" {
		return fmt.Errorf("kodi hostname is required")
	}

	if serviceURL.User != nil {
		k.username = serviceURL.User.Username()
		k.password, _ = serviceURL.User.Password()
	}

	if portStr := serviceURL.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %s", portStr)
		}
		k.port = port
	}

	return nil
}

// Send sends a notification via Kodi JSON-RPC
func (k *KodiService) Send(ctx context.Context, req NotificationRequest) error {
	scheme := "http"
	if k.secure {
		scheme = "https"
	}

	port := k.port
	if port == -1 {
		port = k.GetDefaultPort()
	}

	var rpcURL string
	defaultPort := k.GetDefaultPort()
	if port == defaultPort {
		rpcURL = fmt.Sprintf("%s://%s%s", scheme, k.hostname, kodiRPCPath)
	} else {
		rpcURL = fmt.Sprintf("%s://%s:%d%s", scheme, k.hostname, port, kodiRPCPath)
	}

	title := req.Title
	if title == "" {
		title = "Apprise"
	}

	rpcReq := KodiRPCRequest{
		JSONRPC: "2.0",
		Method:  "GUI.ShowNotification",
		Params: map[string]interface{}{
			"title":       title,
			"message":     req.Body,
			"displaytime": kodiDisplayTime,
		},
		ID: 1,
	}

	jsonData, err := json.Marshal(rpcReq)
	if err != nil {
		return fmt.Errorf("failed to marshal Kodi JSON-RPC request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if k.username != "" {
		httpReq.SetBasicAuth(k.username, k.password)
	}

	resp, err := k.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Kodi notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("kodi JSON-RPC error (status %d): %s", resp.StatusCode, string(body))
	}

	var rpcResp KodiRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil // Non-fatal; HTTP status was OK
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("kodi JSON-RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return nil
}

// TestURL validates a Kodi service URL
func (k *KodiService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	return k.ParseURL(parsedURL)
}
