package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

// Predefined test configurations
var testConfigs = map[string]LoadTestConfig{
	"quick": {
		BaseURL:           "http://localhost:8080",
		Concurrency:       10,
		RequestsPerSecond: 50,
		Duration:          30 * time.Second,
		TestType:          "api",
		NotificationTypes: []string{"info", "success", "warning"},
		ServiceURLs:       []string{"webhook://httpbin.org/post"},
		WarmupDuration:    5 * time.Second,
		CooldownDuration:  2 * time.Second,
		EnableMetrics:     true,
		MetricsInterval:   time.Second,
	},
	"moderate": {
		BaseURL:           "http://localhost:8080",
		Concurrency:       50,
		RequestsPerSecond: 200,
		Duration:          2 * time.Minute,
		TestType:          "mixed",
		NotificationTypes: []string{"info", "success", "warning", "error"},
		ServiceURLs:       []string{"webhook://httpbin.org/post", "webhook://httpbin.org/put"},
		WarmupDuration:    10 * time.Second,
		CooldownDuration:  5 * time.Second,
		EnableMetrics:     true,
		MetricsInterval:   2 * time.Second,
	},
	"stress": {
		BaseURL:           "http://localhost:8080",
		Concurrency:       100,
		RequestsPerSecond: 500,
		Duration:          5 * time.Minute,
		TestType:          "mixed",
		NotificationTypes: []string{"info", "success", "warning", "error"},
		ServiceURLs:       []string{"webhook://httpbin.org/post", "webhook://httpbin.org/put", "webhook://httpbin.org/patch"},
		WarmupDuration:    15 * time.Second,
		CooldownDuration:  10 * time.Second,
		EnableMetrics:     true,
		MetricsInterval:   time.Second,
	},
	"endurance": {
		BaseURL:           "http://localhost:8080",
		Concurrency:       25,
		RequestsPerSecond: 100,
		Duration:          30 * time.Minute,
		TestType:          "api",
		NotificationTypes: []string{"info", "warning"},
		ServiceURLs:       []string{"webhook://httpbin.org/post"},
		WarmupDuration:    30 * time.Second,
		CooldownDuration:  30 * time.Second,
		EnableMetrics:     true,
		MetricsInterval:   5 * time.Second,
	},
	"library-only": {
		BaseURL:           "",
		Concurrency:       20,
		RequestsPerSecond: 100,
		Duration:          time.Minute,
		TestType:          "library",
		NotificationTypes: []string{"info", "success"},
		ServiceURLs:       []string{"webhook://httpbin.org/post"},
		WarmupDuration:    5 * time.Second,
		CooldownDuration:  2 * time.Second,
		EnableMetrics:     true,
		MetricsInterval:   time.Second,
	},
}

func main() {
	var (
		configName   = flag.String("config", "quick", "Predefined config to use (quick|moderate|stress|endurance|library-only)")
		configFile   = flag.String("config-file", "", "Path to custom config JSON file")
		outputFile   = flag.String("output", "", "Path to save results JSON file")
		listConfigs  = flag.Bool("list", false, "List available predefined configurations")
		baseURL      = flag.String("url", "", "Override base URL")
		concurrency  = flag.Int("concurrency", 0, "Override concurrency")
		rps          = flag.Int("rps", 0, "Override requests per second")
		duration     = flag.Duration("duration", 0, "Override test duration")
		testType     = flag.String("type", "", "Override test type (library|api|mixed)")
	)
	flag.Parse()

	if *listConfigs {
		listPredefinedConfigs()
		return
	}

	var config LoadTestConfig
	var err error

	// Load configuration
	if *configFile != "" {
		config, err = loadConfigFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}
	} else {
		var ok bool
		config, ok = testConfigs[*configName]
		if !ok {
			log.Fatalf("Unknown config: %s. Use -list to see available configs", *configName)
		}
	}

	// Apply command line overrides
	if *baseURL != "" {
		config.BaseURL = *baseURL
	}
	if *concurrency > 0 {
		config.Concurrency = *concurrency
	}
	if *rps > 0 {
		config.RequestsPerSecond = *rps
	}
	if *duration > 0 {
		config.Duration = *duration
	}
	if *testType != "" {
		config.TestType = *testType
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create and run load tester
	tester := NewLoadTester(config)
	
	fmt.Printf("Starting load test with configuration: %s\n", *configName)
	fmt.Printf("Press Ctrl+C to stop the test early\n\n")

	ctx := context.Background()
	results, err := tester.Run(ctx)
	if err != nil {
		log.Fatalf("Load test failed: %v", err)
	}

	// Print results
	tester.PrintResults()

	// Save results if requested
	if *outputFile != "" {
		if err := tester.SaveResults(*outputFile); err != nil {
			log.Printf("Failed to save results: %v", err)
		} else {
			fmt.Printf("\nResults saved to: %s\n", *outputFile)
		}
	}

	// Generate performance report
	generatePerformanceReport(results)
}

// loadConfigFromFile loads configuration from a JSON file
func loadConfigFromFile(filename string) (LoadTestConfig, error) {
	var config LoadTestConfig
	
	data, err := os.ReadFile(filename)
	if err != nil {
		return config, err
	}
	
	err = json.Unmarshal(data, &config)
	return config, err
}

// validateConfig validates the load test configuration
func validateConfig(config LoadTestConfig) error {
	if config.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be > 0")
	}
	if config.RequestsPerSecond <= 0 {
		return fmt.Errorf("requests_per_second must be > 0")
	}
	if config.Duration <= 0 {
		return fmt.Errorf("duration must be > 0")
	}
	if config.TestType == "" {
		return fmt.Errorf("test_type is required")
	}
	if config.TestType != "library" && config.TestType != "api" && config.TestType != "mixed" {
		return fmt.Errorf("test_type must be 'library', 'api', or 'mixed'")
	}
	if (config.TestType == "api" || config.TestType == "mixed") && config.BaseURL == "" {
		return fmt.Errorf("base_url is required for API tests")
	}
	if len(config.ServiceURLs) == 0 {
		return fmt.Errorf("at least one service URL is required")
	}
	
	return nil
}

// listPredefinedConfigs lists all available predefined configurations
func listPredefinedConfigs() {
	fmt.Println("Available predefined configurations:")
	fmt.Println()
	
	configs := []struct {
		name        string
		description string
	}{
		{"quick", "Quick smoke test (10 concurrent, 50 rps, 30s)"},
		{"moderate", "Moderate load test (50 concurrent, 200 rps, 2m)"},
		{"stress", "High load stress test (100 concurrent, 500 rps, 5m)"},
		{"endurance", "Long-running endurance test (25 concurrent, 100 rps, 30m)"},
		{"library-only", "Library-only testing (20 concurrent, 100 rps, 1m)"},
	}
	
	for _, cfg := range configs {
		fmt.Printf("  %-15s %s\n", cfg.name, cfg.description)
	}
	
	fmt.Println()
	fmt.Println("Usage examples:")
	fmt.Println("  go run . -config quick")
	fmt.Println("  go run . -config stress -url http://production:8080")
	fmt.Println("  go run . -config moderate -concurrency 25 -rps 100")
	fmt.Println("  go run . -config-file custom-config.json -output results.json")
}

// generatePerformanceReport generates a detailed performance analysis
func generatePerformanceReport(results *LoadTestResults) {
	fmt.Printf("\n=== Performance Analysis ===\n")
	
	// Calculate performance rating
	rating := calculatePerformanceRating(results)
	fmt.Printf("Overall Performance Rating: %s\n", rating)
	
	// SLA Analysis
	fmt.Printf("\nSLA Compliance:\n")
	sla95 := 500 * time.Millisecond
	if results.P95ResponseTime <= sla95 {
		fmt.Printf("  ✓ 95th percentile response time: %v (within %v SLA)\n", results.P95ResponseTime, sla95)
	} else {
		fmt.Printf("  ✗ 95th percentile response time: %v (exceeds %v SLA)\n", results.P95ResponseTime, sla95)
	}
	
	successRate := (1.0 - results.ErrorRate) * 100
	if successRate >= 99.0 {
		fmt.Printf("  ✓ Success rate: %.2f%% (meets 99%% SLA)\n", successRate)
	} else {
		fmt.Printf("  ✗ Success rate: %.2f%% (below 99%% SLA)\n", successRate)
	}
	
	// Throughput analysis
	fmt.Printf("\nThroughput Analysis:\n")
	expectedRPS := float64(results.Config.RequestsPerSecond)
	actualRPS := results.RequestsPerSecond
	efficiency := (actualRPS / expectedRPS) * 100
	
	fmt.Printf("  Expected RPS: %.2f\n", expectedRPS)
	fmt.Printf("  Actual RPS: %.2f\n", actualRPS)
	fmt.Printf("  Efficiency: %.1f%%\n", efficiency)
	
	if efficiency >= 95 {
		fmt.Printf("  ✓ High efficiency - system handling load well\n")
	} else if efficiency >= 80 {
		fmt.Printf("  ⚠ Moderate efficiency - some performance degradation\n")
	} else {
		fmt.Printf("  ✗ Low efficiency - significant performance issues\n")
	}
	
	// Response time distribution
	fmt.Printf("\nResponse Time Distribution:\n")
	for percentile, duration := range results.Percentiles {
		fmt.Printf("  %s: %v\n", percentile, duration)
	}
	
	// Recommendations
	generateRecommendations(results)
}

// calculatePerformanceRating provides an overall performance rating
func calculatePerformanceRating(results *LoadTestResults) string {
	score := 0
	
	// Error rate scoring (40 points max)
	if results.ErrorRate == 0 {
		score += 40
	} else if results.ErrorRate <= 0.01 {
		score += 35
	} else if results.ErrorRate <= 0.05 {
		score += 25
	} else if results.ErrorRate <= 0.10 {
		score += 15
	}
	
	// Response time scoring (40 points max)
	if results.P95ResponseTime <= 100*time.Millisecond {
		score += 40
	} else if results.P95ResponseTime <= 300*time.Millisecond {
		score += 35
	} else if results.P95ResponseTime <= 500*time.Millisecond {
		score += 25
	} else if results.P95ResponseTime <= 1*time.Second {
		score += 15
	} else if results.P95ResponseTime <= 2*time.Second {
		score += 5
	}
	
	// Throughput efficiency scoring (20 points max)
	expectedRPS := float64(results.Config.RequestsPerSecond)
	actualRPS := results.RequestsPerSecond
	efficiency := actualRPS / expectedRPS
	
	if efficiency >= 0.95 {
		score += 20
	} else if efficiency >= 0.80 {
		score += 15
	} else if efficiency >= 0.60 {
		score += 10
	} else if efficiency >= 0.40 {
		score += 5
	}
	
	// Convert to letter grade
	switch {
	case score >= 90:
		return "A (Excellent)"
	case score >= 80:
		return "B (Good)"
	case score >= 70:
		return "C (Fair)"
	case score >= 60:
		return "D (Poor)"
	default:
		return "F (Failing)"
	}
}

// generateRecommendations provides performance optimization recommendations
func generateRecommendations(results *LoadTestResults) {
	fmt.Printf("\nPerformance Recommendations:\n")
	
	recommendations := []string{}
	
	// Error rate recommendations
	if results.ErrorRate > 0.05 {
		recommendations = append(recommendations,
			"High error rate detected - investigate service reliability and error handling")
	}
	
	// Response time recommendations
	if results.P95ResponseTime > 500*time.Millisecond {
		recommendations = append(recommendations,
			"Response times exceed SLA - consider optimizing notification delivery")
	}
	
	if results.AverageResponseTime > 200*time.Millisecond {
		recommendations = append(recommendations,
			"Average response time is high - check for performance bottlenecks")
	}
	
	// Throughput recommendations
	expectedRPS := float64(results.Config.RequestsPerSecond)
	actualRPS := results.RequestsPerSecond
	if actualRPS < expectedRPS*0.8 {
		recommendations = append(recommendations,
			"Throughput significantly below target - increase concurrency or optimize processing")
	}
	
	// Resource usage recommendations
	if results.ResourceUsage != nil {
		if results.ResourceUsage.MaxMemoryUsage > 1024*1024*1024 { // 1GB
			recommendations = append(recommendations,
				"High memory usage detected - monitor for memory leaks")
		}
		
		if results.ResourceUsage.MaxGoroutines > 1000 {
			recommendations = append(recommendations,
				"High goroutine count - check for goroutine leaks or excessive concurrency")
		}
	}
	
	// General recommendations
	if results.Config.TestType == "mixed" {
		recommendations = append(recommendations,
			"Mixed test shows real-world usage - consider running separate library and API tests")
	}
	
	if len(recommendations) == 0 {
		fmt.Printf("  ✓ No specific recommendations - performance looks good!\n")
	} else {
		for i, rec := range recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}
	
	// Additional optimization suggestions
	fmt.Printf("\nOptimization Suggestions:\n")
	fmt.Printf("  • Use connection pooling for HTTP clients\n")
	fmt.Printf("  • Implement circuit breaker patterns for external services\n")
	fmt.Printf("  • Consider async notification processing for high throughput\n")
	fmt.Printf("  • Monitor and tune garbage collection if memory usage is high\n")
	fmt.Printf("  • Implement retries with exponential backoff for failed notifications\n")
}