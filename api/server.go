package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/scttfrdmn/apprise-go/apprise"
)

// ServerConfig holds the configuration for the API server
type ServerConfig struct {
	Host           string          `json:"host"`
	Port           string          `json:"port"`
	DatabasePath   string          `json:"database_path"`
	CORSOrigins    []string        `json:"cors_origins"`
	JWTSecret      string          `json:"jwt_secret"`
	LogLevel       string          `json:"log_level"`
	RequireAuth    bool            `json:"require_auth"`
	TokenDuration  int             `json:"token_duration"` // hours
	RateLimit      RateLimitConfig `json:"rate_limit"`
}

// Server represents the REST API server
type Server struct {
	config      *ServerConfig
	apprise     *apprise.Apprise
	scheduler   *apprise.NotificationScheduler
	logger      *log.Logger
	router      *mux.Router
	server      *http.Server
	rateLimiter *RateLimiter
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewServer creates a new API server instance
func NewServer(config *ServerConfig, apprise *apprise.Apprise, scheduler *apprise.NotificationScheduler, logger *log.Logger) (*Server, error) {
	// Create default logger if none provided
	if logger == nil {
		logger = log.New(os.Stderr, "apprise-api: ", log.LstdFlags)
	}
	
	s := &Server{
		config:    config,
		apprise:   apprise,
		scheduler: scheduler,
		logger:    logger,
	}

	// Initialize rate limiter if enabled
	if config.RateLimit.Enabled {
		s.rateLimiter = NewRateLimiter(config.RateLimit)
	}

	s.setupRoutes()
	return s, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	s.router = mux.NewRouter()

	// API v1 routes
	apiV1 := s.router.PathPrefix("/api/v1").Subrouter()

	// Health and info endpoints
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/version", s.handleVersion).Methods("GET")
	s.router.HandleFunc("/metrics", s.handleMetrics).Methods("GET")

	// Authentication endpoints (public)
	authV1 := apiV1.PathPrefix("/auth").Subrouter()
	authV1.HandleFunc("/login", s.handleLogin).Methods("POST")
	authV1.HandleFunc("/register", s.handleRegister).Methods("POST")
	authV1.HandleFunc("/whoami", s.handleWhoAmI).Methods("GET")
	authV1.HandleFunc("/refresh", s.handleRefreshToken).Methods("POST")
	authV1.HandleFunc("/apikeys", s.handleListAPIKeys).Methods("GET")
	authV1.HandleFunc("/apikeys", s.handleCreateAPIKey).Methods("POST")
	authV1.HandleFunc("/apikeys/{key_id}", s.handleDeleteAPIKey).Methods("DELETE")
	authV1.HandleFunc("/ratelimit", s.handleGetRateLimitStatus).Methods("GET")

	// Documentation and Dashboard
	s.router.HandleFunc("/docs", s.handleDocs).Methods("GET")
	s.router.HandleFunc("/dashboard", s.handleDashboard).Methods("GET")
	s.router.HandleFunc("/dashboard/", s.handleDashboard).Methods("GET")
	s.router.HandleFunc("/dashboard.html", s.handleDashboard).Methods("GET")
	s.router.HandleFunc("/dashboard.js", s.handleDashboard).Methods("GET")
	s.router.HandleFunc("/", s.handleRoot).Methods("GET")

	// Notification endpoints
	apiV1.HandleFunc("/notify", s.handleNotify).Methods("POST")
	apiV1.HandleFunc("/notify/bulk", s.handleBulkNotify).Methods("POST")

	// Service management endpoints
	apiV1.HandleFunc("/services", s.handleListServices).Methods("GET")
	apiV1.HandleFunc("/services", s.handleAddService).Methods("POST")
	apiV1.HandleFunc("/services/{service_id}", s.handleGetService).Methods("GET")
	apiV1.HandleFunc("/services/{service_id}", s.handleUpdateService).Methods("PUT")
	apiV1.HandleFunc("/services/{service_id}", s.handleDeleteService).Methods("DELETE")
	apiV1.HandleFunc("/services/{service_id}/test", s.handleTestService).Methods("POST")

	// Configuration endpoints
	apiV1.HandleFunc("/config", s.handleGetConfig).Methods("GET")
	apiV1.HandleFunc("/config", s.handleUpdateConfig).Methods("PUT")
	apiV1.HandleFunc("/config/load", s.handleLoadConfig).Methods("POST")

	// Scheduler endpoints (if scheduler is available)
	if s.scheduler != nil {
		schedulerV1 := apiV1.PathPrefix("/scheduler").Subrouter()
		
		// Job management
		schedulerV1.HandleFunc("/jobs", s.handleListScheduledJobs).Methods("GET")
		schedulerV1.HandleFunc("/jobs", s.handleCreateScheduledJob).Methods("POST")
		schedulerV1.HandleFunc("/jobs/{job_id}", s.handleGetScheduledJob).Methods("GET")
		schedulerV1.HandleFunc("/jobs/{job_id}", s.handleUpdateScheduledJob).Methods("PUT")
		schedulerV1.HandleFunc("/jobs/{job_id}", s.handleDeleteScheduledJob).Methods("DELETE")
		schedulerV1.HandleFunc("/jobs/{job_id}/enable", s.handleEnableScheduledJob).Methods("POST")
		schedulerV1.HandleFunc("/jobs/{job_id}/disable", s.handleDisableScheduledJob).Methods("POST")

		// Queue management
		schedulerV1.HandleFunc("/queue", s.handleListQueuedJobs).Methods("GET")
		schedulerV1.HandleFunc("/queue", s.handleAddToQueue).Methods("POST")
		schedulerV1.HandleFunc("/queue/{job_id}", s.handleGetQueuedJob).Methods("GET")
		schedulerV1.HandleFunc("/queue/{job_id}", s.handleUpdateQueuedJob).Methods("PUT")
		schedulerV1.HandleFunc("/queue/{job_id}/retry", s.handleRetryQueuedJob).Methods("POST")
		schedulerV1.HandleFunc("/queue/stats", s.handleQueueStats).Methods("GET")

		// Template management
		schedulerV1.HandleFunc("/templates", s.handleListTemplates).Methods("GET")
		schedulerV1.HandleFunc("/templates", s.handleCreateTemplate).Methods("POST")
		schedulerV1.HandleFunc("/templates/{template_name}", s.handleGetTemplate).Methods("GET")
		schedulerV1.HandleFunc("/templates/{template_name}", s.handleUpdateTemplate).Methods("PUT")
		schedulerV1.HandleFunc("/templates/{template_name}", s.handleDeleteTemplate).Methods("DELETE")
		schedulerV1.HandleFunc("/templates/{template_name}/render", s.handleRenderTemplate).Methods("POST")

		// Metrics and analytics
		schedulerV1.HandleFunc("/metrics", s.handleSchedulerMetrics).Methods("GET")
		schedulerV1.HandleFunc("/metrics/report", s.handleMetricsReport).Methods("POST")
	}

	// Add middleware (order matters!)
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.recoverMiddleware)
	s.router.Use(s.RateLimitMiddleware)  // Apply rate limiting first
	s.router.Use(s.AuthMiddleware)       // Then authentication
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe(addr string) error {
	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   s.config.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Add metrics middleware
	metricsHandler := s.apprise.GetMetrics().HTTPMiddleware(s.router)
	handler := c.Handler(metricsHandler)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Stop rate limiter first
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}
	
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// GenerateJWTSecret generates a secure random JWT secret
func GenerateJWTSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "default-secret-change-in-production"
	}
	return hex.EncodeToString(bytes)
}

// Helper methods for JSON responses
func (s *Server) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) sendSuccess(w http.ResponseWriter, message string, data interface{}) {
	response := APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
	s.sendJSON(w, http.StatusOK, response)
}

func (s *Server) sendError(w http.ResponseWriter, status int, message string, err error) {
	response := APIResponse{
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}
	if err != nil {
		response.Error = err.Error()
	}
	s.sendJSON(w, status, response)
}

// Middleware
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Printf("%s %s %s %s", r.Method, r.RequestURI, r.RemoteAddr, time.Since(start))
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Printf("Panic recovered: %v", err)
				s.sendError(w, http.StatusInternalServerError, "Internal server error", fmt.Errorf("%v", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}