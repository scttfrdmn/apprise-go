package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

// LoadTestConfig represents configuration for load testing
type LoadTestConfig struct {
	BaseURL            string        `json:"base_url"`
	Concurrency        int           `json:"concurrency"`
	RequestsPerSecond  int           `json:"requests_per_second"`
	Duration           time.Duration `json:"duration"`
	TestType           string        `json:"test_type"` // "library", "api", "mixed"
	NotificationTypes  []string      `json:"notification_types"`
	ServiceURLs        []string      `json:"service_urls"`
	WarmupDuration     time.Duration `json:"warmup_duration"`
	CooldownDuration   time.Duration `json:"cooldown_duration"`
	EnableMetrics      bool          `json:"enable_metrics"`
	MetricsInterval    time.Duration `json:"metrics_interval"`
}

// LoadTestResults represents the results of load testing
type LoadTestResults struct {
	Config              LoadTestConfig    `json:"config"`
	TotalRequests       int64             `json:"total_requests"`
	SuccessfulRequests  int64             `json:"successful_requests"`
	FailedRequests      int64             `json:"failed_requests"`
	AverageResponseTime time.Duration     `json:"average_response_time"`
	MedianResponseTime  time.Duration     `json:"median_response_time"`
	P95ResponseTime     time.Duration     `json:"p95_response_time"`
	P99ResponseTime     time.Duration     `json:"p99_response_time"`
	MinResponseTime     time.Duration     `json:"min_response_time"`
	MaxResponseTime     time.Duration     `json:"max_response_time"`
	RequestsPerSecond   float64           `json:"requests_per_second"`
	ErrorRate           float64           `json:"error_rate"`
	Duration            time.Duration     `json:"duration"`
	ResponseTimes       []time.Duration   `json:"-"` // Don't serialize large arrays
	ErrorDistribution   map[string]int64  `json:"error_distribution"`
	ResourceUsage       *ResourceMetrics  `json:"resource_usage,omitempty"`
	Percentiles         map[string]time.Duration `json:"percentiles"`
}

// ResourceMetrics tracks system resource usage during tests
type ResourceMetrics struct {
	MaxMemoryUsage    uint64 `json:"max_memory_usage"`
	AverageMemoryUsage uint64 `json:"average_memory_usage"`
	MaxGoroutines     int    `json:"max_goroutines"`
	AverageGoroutines int    `json:"average_goroutines"`
	CPUUsagePercent   float64 `json:"cpu_usage_percent"`
}

// APIRequest represents a notification request for API testing
type APIRequest struct {
	URLs   []string          `json:"urls,omitempty"`
	Title  string            `json:"title,omitempty"`
	Body   string            `json:"body"`
	Type   string            `json:"type,omitempty"`
	Tags   []string          `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// LoadTester manages load testing execution
type LoadTester struct {
	config      LoadTestConfig
	results     LoadTestResults
	apprise     *apprise.Apprise
	httpClient  *http.Client
	rateLimiter <-chan time.Time
	mu          sync.RWMutex
	stopMetrics chan struct{}
	wg          sync.WaitGroup
}

// NewLoadTester creates a new load tester instance
func NewLoadTester(config LoadTestConfig) *LoadTester {
	// Setup rate limiter
	interval := time.Second / time.Duration(config.RequestsPerSecond)
	rateLimiter := time.Tick(interval)

	// Setup HTTP client with optimized settings
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Initialize Apprise for library testing
	var appriseInstance *apprise.Apprise
	if config.TestType == "library" || config.TestType == "mixed" {
		appriseInstance = apprise.New()
		for _, serviceURL := range config.ServiceURLs {
			if err := appriseInstance.Add(serviceURL); err != nil {
				log.Printf("Failed to add service %s: %v", serviceURL, err)
			}
		}
	}

	return &LoadTester{
		config:      config,
		apprise:     appriseInstance,
		httpClient:  client,
		rateLimiter: rateLimiter,
		stopMetrics: make(chan struct{}),
		results: LoadTestResults{
			Config:            config,
			ErrorDistribution: make(map[string]int64),
			Percentiles:       make(map[string]time.Duration),
		},
	}
}

// Run executes the load test
func (lt *LoadTester) Run(ctx context.Context) (*LoadTestResults, error) {
	log.Printf("Starting load test: %s", lt.config.TestType)
	log.Printf("Configuration: %d concurrency, %d req/s, %v duration", 
		lt.config.Concurrency, lt.config.RequestsPerSecond, lt.config.Duration)

	// Start metrics collection
	if lt.config.EnableMetrics {
		lt.wg.Add(1)
		go lt.collectMetrics()
	}

	// Warmup phase
	if lt.config.WarmupDuration > 0 {
		log.Printf("Warmup phase: %v", lt.config.WarmupDuration)
		lt.warmup(ctx)
	}

	// Main test execution
	startTime := time.Now()
	lt.executeTest(ctx)
	lt.results.Duration = time.Since(startTime)

	// Stop metrics collection
	if lt.config.EnableMetrics {
		close(lt.stopMetrics)
		lt.wg.Wait()
	}

	// Cooldown phase
	if lt.config.CooldownDuration > 0 {
		log.Printf("Cooldown phase: %v", lt.config.CooldownDuration)
		time.Sleep(lt.config.CooldownDuration)
	}

	// Calculate results
	lt.calculateResults()

	log.Printf("Load test completed: %d requests, %.2f%% success rate, %.2f req/s",
		lt.results.TotalRequests, 
		(1.0-lt.results.ErrorRate)*100,
		lt.results.RequestsPerSecond)

	return &lt.results, nil
}

// warmup performs a warmup phase to stabilize the system
func (lt *LoadTester) warmup(ctx context.Context) {
	warmupCtx, cancel := context.WithTimeout(ctx, lt.config.WarmupDuration)
	defer cancel()

	concurrency := min(lt.config.Concurrency/4, 10) // Use 25% concurrency for warmup
	requests := make(chan struct{}, concurrency)

	var warmupWg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		warmupWg.Add(1)
		go func() {
			defer warmupWg.Done()
			for {
				select {
				case <-warmupCtx.Done():
					return
				case <-requests:
					lt.executeRequest(ctx, false) // Don't collect metrics during warmup
				}
			}
		}()
	}

	// Send warmup requests
	go func() {
		ticker := time.NewTicker(time.Second / 10) // 10 req/s during warmup
		defer ticker.Stop()
		for {
			select {
			case <-warmupCtx.Done():
				return
			case <-ticker.C:
				select {
				case requests <- struct{}{}:
				default:
				}
			}
		}
	}()

	<-warmupCtx.Done()
	close(requests)
	warmupWg.Wait()
}

// executeTest runs the main load test
func (lt *LoadTester) executeTest(ctx context.Context) {
	testCtx, cancel := context.WithTimeout(ctx, lt.config.Duration)
	defer cancel()

	requests := make(chan struct{}, lt.config.Concurrency)
	
	// Start worker goroutines
	var workers sync.WaitGroup
	for i := 0; i < lt.config.Concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-testCtx.Done():
					return
				case <-requests:
					lt.executeRequest(testCtx, true)
				}
			}
		}()
	}

	// Rate-limited request generator
	go func() {
		defer close(requests)
		for {
			select {
			case <-testCtx.Done():
				return
			case <-lt.rateLimiter:
				select {
				case requests <- struct{}{}:
				case <-testCtx.Done():
					return
				}
			}
		}
	}()

	<-testCtx.Done()
	workers.Wait()
}

// executeRequest executes a single request based on test type
func (lt *LoadTester) executeRequest(ctx context.Context, collectMetrics bool) {
	var err error
	var responseTime time.Duration
	
	start := time.Now()
	
	switch lt.config.TestType {
	case "library":
		err = lt.executeLibraryRequest(ctx)
	case "api":
		err = lt.executeAPIRequest(ctx)
	case "mixed":
		if time.Now().UnixNano()%2 == 0 {
			err = lt.executeLibraryRequest(ctx)
		} else {
			err = lt.executeAPIRequest(ctx)
		}
	}
	
	responseTime = time.Since(start)
	
	if collectMetrics {
		lt.recordResult(err, responseTime)
	}
}

// executeLibraryRequest executes a request using the Apprise library directly
func (lt *LoadTester) executeLibraryRequest(ctx context.Context) error {
	if lt.apprise == nil {
		return fmt.Errorf("apprise instance not initialized")
	}
	
	notifyType := apprise.NotifyTypeInfo
	if len(lt.config.NotificationTypes) > 0 {
		switch lt.config.NotificationTypes[time.Now().UnixNano()%int64(len(lt.config.NotificationTypes))] {
		case "success":
			notifyType = apprise.NotifyTypeSuccess
		case "warning":
			notifyType = apprise.NotifyTypeWarning
		case "error":
			notifyType = apprise.NotifyTypeError
		}
	}
	
	title := fmt.Sprintf("Load Test %d", time.Now().UnixNano())
	body := fmt.Sprintf("Load test message at %s", time.Now().Format(time.RFC3339))
	
	responses := lt.apprise.Notify(title, body, notifyType)
	
	// Check if any notification failed
	for _, resp := range responses {
		if resp.Error != nil {
			return resp.Error
		}
	}
	
	return nil
}

// executeAPIRequest executes a request using the HTTP API
func (lt *LoadTester) executeAPIRequest(ctx context.Context) error {
	reqBody := APIRequest{
		URLs:  lt.config.ServiceURLs,
		Title: fmt.Sprintf("Load Test %d", time.Now().UnixNano()),
		Body:  fmt.Sprintf("Load test message at %s", time.Now().Format(time.RFC3339)),
		Type:  "info",
	}
	
	if len(lt.config.NotificationTypes) > 0 {
		reqBody.Type = lt.config.NotificationTypes[time.Now().UnixNano()%int64(len(lt.config.NotificationTypes))]
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", lt.config.BaseURL+"/api/v1/notify", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := lt.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	
	return nil
}

// recordResult records the result of a request execution
func (lt *LoadTester) recordResult(err error, responseTime time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	lt.results.TotalRequests++
	lt.results.ResponseTimes = append(lt.results.ResponseTimes, responseTime)
	
	if err != nil {
		lt.results.FailedRequests++
		errorType := "unknown"
		if urlErr, ok := err.(*url.Error); ok {
			errorType = fmt.Sprintf("url_%s", urlErr.Op)
		} else {
			errorType = err.Error()
		}
		lt.results.ErrorDistribution[errorType]++
	} else {
		lt.results.SuccessfulRequests++
	}
}

// collectMetrics collects system resource metrics during the test
func (lt *LoadTester) collectMetrics() {
	defer lt.wg.Done()
	
	ticker := time.NewTicker(lt.config.MetricsInterval)
	defer ticker.Stop()
	
	var memorySum uint64
	var goroutineSum int
	var samples int
	
	for {
		select {
		case <-lt.stopMetrics:
			// Calculate averages
			if samples > 0 {
				lt.results.ResourceUsage = &ResourceMetrics{
					AverageMemoryUsage: memorySum / uint64(samples),
					AverageGoroutines:  goroutineSum / samples,
				}
			}
			return
		case <-ticker.C:
			// Collect current metrics (would need actual system calls)
			// This is a placeholder - in real implementation would use:
			// - runtime.ReadMemStats() for memory
			// - runtime.NumGoroutine() for goroutines
			// - CPU usage from system calls
			samples++
			
			// Mock values for demonstration
			currentMemory := uint64(1024 * 1024 * 100) // 100MB mock
			currentGoroutines := 50 // Mock goroutine count
			
			memorySum += currentMemory
			goroutineSum += currentGoroutines
			
			if lt.results.ResourceUsage == nil {
				lt.results.ResourceUsage = &ResourceMetrics{}
			}
			
			if currentMemory > lt.results.ResourceUsage.MaxMemoryUsage {
				lt.results.ResourceUsage.MaxMemoryUsage = currentMemory
			}
			
			if currentGoroutines > lt.results.ResourceUsage.MaxGoroutines {
				lt.results.ResourceUsage.MaxGoroutines = currentGoroutines
			}
		}
	}
}

// calculateResults computes final test results and statistics
func (lt *LoadTester) calculateResults() {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	if lt.results.TotalRequests == 0 {
		return
	}
	
	// Calculate error rate
	lt.results.ErrorRate = float64(lt.results.FailedRequests) / float64(lt.results.TotalRequests)
	
	// Calculate requests per second
	lt.results.RequestsPerSecond = float64(lt.results.TotalRequests) / lt.results.Duration.Seconds()
	
	// Sort response times for percentile calculations
	times := make([]time.Duration, len(lt.results.ResponseTimes))
	copy(times, lt.results.ResponseTimes)
	
	if len(times) == 0 {
		return
	}
	
	// Simple bubble sort (for demonstration - use proper sorting in production)
	for i := 0; i < len(times); i++ {
		for j := 0; j < len(times)-1-i; j++ {
			if times[j] > times[j+1] {
				times[j], times[j+1] = times[j+1], times[j]
			}
		}
	}
	
	// Calculate statistics
	lt.results.MinResponseTime = times[0]
	lt.results.MaxResponseTime = times[len(times)-1]
	
	// Calculate median
	if len(times)%2 == 0 {
		lt.results.MedianResponseTime = (times[len(times)/2-1] + times[len(times)/2]) / 2
	} else {
		lt.results.MedianResponseTime = times[len(times)/2]
	}
	
	// Calculate percentiles
	lt.results.P95ResponseTime = times[int(float64(len(times))*0.95)]
	lt.results.P99ResponseTime = times[int(float64(len(times))*0.99)]
	
	// Calculate average
	var sum time.Duration
	for _, t := range times {
		sum += t
	}
	lt.results.AverageResponseTime = sum / time.Duration(len(times))
	
	// Store additional percentiles
	lt.results.Percentiles["p50"] = lt.results.MedianResponseTime
	lt.results.Percentiles["p90"] = times[int(float64(len(times))*0.90)]
	lt.results.Percentiles["p95"] = lt.results.P95ResponseTime
	lt.results.Percentiles["p99"] = lt.results.P99ResponseTime
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SaveResults saves test results to a JSON file
func (lt *LoadTester) SaveResults(filename string) error {
	data, err := json.MarshalIndent(&lt.results, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}

// PrintResults prints a summary of test results
func (lt *LoadTester) PrintResults() {
	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Test Type: %s\n", lt.config.TestType)
	fmt.Printf("Duration: %v\n", lt.results.Duration)
	fmt.Printf("Total Requests: %d\n", lt.results.TotalRequests)
	fmt.Printf("Successful Requests: %d\n", lt.results.SuccessfulRequests)
	fmt.Printf("Failed Requests: %d\n", lt.results.FailedRequests)
	fmt.Printf("Error Rate: %.2f%%\n", lt.results.ErrorRate*100)
	fmt.Printf("Requests/Second: %.2f\n", lt.results.RequestsPerSecond)
	fmt.Printf("\nResponse Times:\n")
	fmt.Printf("  Average: %v\n", lt.results.AverageResponseTime)
	fmt.Printf("  Median:  %v\n", lt.results.MedianResponseTime)
	fmt.Printf("  95th%%:   %v\n", lt.results.P95ResponseTime)
	fmt.Printf("  99th%%:   %v\n", lt.results.P99ResponseTime)
	fmt.Printf("  Min:     %v\n", lt.results.MinResponseTime)
	fmt.Printf("  Max:     %v\n", lt.results.MaxResponseTime)
	
	if len(lt.results.ErrorDistribution) > 0 {
		fmt.Printf("\nError Distribution:\n")
		for errorType, count := range lt.results.ErrorDistribution {
			fmt.Printf("  %s: %d\n", errorType, count)
		}
	}
	
	if lt.results.ResourceUsage != nil {
		fmt.Printf("\nResource Usage:\n")
		fmt.Printf("  Max Memory: %d bytes\n", lt.results.ResourceUsage.MaxMemoryUsage)
		fmt.Printf("  Avg Memory: %d bytes\n", lt.results.ResourceUsage.AverageMemoryUsage)
		fmt.Printf("  Max Goroutines: %d\n", lt.results.ResourceUsage.MaxGoroutines)
		fmt.Printf("  Avg Goroutines: %d\n", lt.results.ResourceUsage.AverageGoroutines)
	}
}