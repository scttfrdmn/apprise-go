package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/scttfrdmn/apprise-go/api"
	"github.com/scttfrdmn/apprise-go/apprise"
)

var (
	port            = flag.String("port", "8080", "Port to listen on")
	host            = flag.String("host", "0.0.0.0", "Host to bind to")
	dbPath          = flag.String("db", "./apprise-api.db", "Database path for scheduler and config storage")
	configPath      = flag.String("config", "", "Path to configuration file")
	logLevel        = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	corsOrigin      = flag.String("cors-origin", "*", "CORS allowed origins")
	jwtSecret       = flag.String("jwt-secret", "", "JWT secret for authentication (generate if empty)")
	requireAuth     = flag.Bool("require-auth", false, "Require authentication for API access")
	tokenDuration   = flag.Int("token-duration", 24, "JWT token duration in hours")
	enableRateLimit = flag.Bool("enable-ratelimit", true, "Enable rate limiting")
	rateLimit       = flag.Int("rate-limit", 60, "Requests per minute per client")
	version         = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	if *version {
		versionInfo := apprise.GetVersionInfo()
		fmt.Printf("%s\n", versionInfo.String())
		fmt.Printf("API Server Component\n")
		os.Exit(0)
	}

	// Create logger
	logger := log.New(os.Stdout, "[apprise-api] ", log.LstdFlags|log.Lshortfile)

	// Create Apprise instance
	appriseInstance := apprise.New()

	// Create scheduler if database path is provided
	var scheduler *apprise.NotificationScheduler
	if *dbPath != "" {
		var err error
		scheduler, err = apprise.NewNotificationScheduler(*dbPath, appriseInstance)
		if err != nil {
			logger.Fatalf("Failed to create scheduler: %v", err)
		}
		defer scheduler.Close()

		// Start scheduler
		ctx := context.Background()
		if err := scheduler.Start(ctx); err != nil {
			logger.Fatalf("Failed to start scheduler: %v", err)
		}
		defer scheduler.Stop()

		logger.Printf("Scheduler started with database: %s", *dbPath)
	}

	// Load configuration if provided
	if *configPath != "" {
		// TODO: Implement configuration loading
		logger.Printf("Configuration file: %s", *configPath)
	}

	// Generate JWT secret if not provided
	if *jwtSecret == "" {
		*jwtSecret = api.GenerateJWTSecret()
		logger.Printf("Generated JWT secret (save this for production): %s", *jwtSecret)
	}

	// Create API server configuration
	serverConfig := &api.ServerConfig{
		Host:          *host,
		Port:          *port,
		DatabasePath:  *dbPath,
		CORSOrigins:   []string{*corsOrigin},
		JWTSecret:     *jwtSecret,
		LogLevel:      *logLevel,
		RequireAuth:   *requireAuth,
		TokenDuration: *tokenDuration,
		RateLimit: api.RateLimitConfig{
			Enabled:        *enableRateLimit,
			RequestsPerMin: *rateLimit,
			BurstSize:      *rateLimit / 4, // 25% of limit as burst
			WindowSize:     time.Minute,
		},
	}

	// Create and configure the API server
	server, err := api.NewServer(serverConfig, appriseInstance, scheduler, logger)
	if err != nil {
		logger.Fatalf("Failed to create API server: %v", err)
	}

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf("%s:%s", *host, *port)
		logger.Printf("Starting Apprise API server on %s", addr)
		logger.Printf("API documentation available at http://%s/docs", addr)
		logger.Printf("Health check endpoint: http://%s/health", addr)
		logger.Printf("Metrics endpoint: http://%s/metrics", addr)

		if err := server.ListenAndServe(addr); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	<-c
	logger.Println("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		logger.Printf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server exited")
}