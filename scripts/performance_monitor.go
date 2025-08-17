package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

// PerformanceMetrics represents performance measurement data
type PerformanceMetrics struct {
	Timestamp          time.Time     `json:"timestamp"`
	TotalRequests      int           `json:"total_requests"`
	SuccessfulRequests int           `json:"successful_requests"`
	FailedRequests     int           `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	MinLatency         time.Duration `json:"min_latency"`
	MaxLatency         time.Duration `json:"max_latency"`
	RequestsPerSecond  float64       `json:"requests_per_second"`
	GoroutineCount     int           `json:"goroutine_count"`
	MemoryUsage        MemoryMetrics `json:"memory_usage"`
	ErrorRate          float64       `json:"error_rate"`
}

// MemoryMetrics represents memory usage statistics
type MemoryMetrics struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	NumGC        uint32  `json:"num_gc"`
	HeapAllocMB  float64 `json:"heap_alloc_mb"`
	HeapSysMB    float64 `json:"heap_sys_mb"`
}

// PerformanceMonitor manages continuous performance monitoring
type PerformanceMonitor struct {
	apprise         *apprise.Apprise
	metrics         []PerformanceMetrics
	mu              sync.RWMutex
	startTime       time.Time
	totalRequests   int
	successRequests int
	failedRequests  int
	latencies       []time.Duration
	running         bool
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		apprise:   apprise.New(),
		metrics:   make([]PerformanceMetrics, 0),
		startTime: time.Now(),
		latencies: make([]time.Duration, 0),
	}
}

// AddService adds a service to monitor
func (pm *PerformanceMonitor) AddService(serviceURL string) error {
	return pm.apprise.Add(serviceURL)
}

// Start begins performance monitoring
func (pm *PerformanceMonitor) Start(ctx context.Context, interval time.Duration) {
	pm.mu.Lock()
	pm.running = true
	pm.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			pm.mu.Lock()
			pm.running = false
			pm.mu.Unlock()
			return
		case <-ticker.C:
			pm.collectMetrics()
		}
	}
}

// SendNotification sends a test notification and records metrics
func (pm *PerformanceMonitor) SendNotification(title, body string, notifyType apprise.NotifyType) {
	start := time.Now()
	responses := pm.apprise.Notify(title, body, notifyType)
	latency := time.Since(start)

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.totalRequests++
	pm.latencies = append(pm.latencies, latency)

	// Count successes and failures
	allSuccess := true
	for _, resp := range responses {
		if !resp.Success {
			allSuccess = false
			break
		}
	}

	if allSuccess && len(responses) > 0 {
		pm.successRequests++
	} else {
		pm.failedRequests++
	}
}

// collectMetrics gathers current performance metrics
func (pm *PerformanceMonitor) collectMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate latency statistics
	var avgLatency, minLatency, maxLatency time.Duration
	if len(pm.latencies) > 0 {
		var totalLatency time.Duration
		minLatency = pm.latencies[0]
		maxLatency = pm.latencies[0]

		for _, latency := range pm.latencies {
			totalLatency += latency
			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		}
		avgLatency = totalLatency / time.Duration(len(pm.latencies))
	}

	// Calculate requests per second
	elapsed := time.Since(pm.startTime).Seconds()
	rps := float64(pm.totalRequests) / elapsed

	// Calculate error rate
	errorRate := 0.0
	if pm.totalRequests > 0 {
		errorRate = float64(pm.failedRequests) / float64(pm.totalRequests) * 100
	}

	metrics := PerformanceMetrics{
		Timestamp:          time.Now(),
		TotalRequests:      pm.totalRequests,
		SuccessfulRequests: pm.successRequests,
		FailedRequests:     pm.failedRequests,
		AverageLatency:     avgLatency,
		MinLatency:         minLatency,
		MaxLatency:         maxLatency,
		RequestsPerSecond:  rps,
		GoroutineCount:     runtime.NumGoroutine(),
		ErrorRate:          errorRate,
		MemoryUsage: MemoryMetrics{
			AllocMB:      float64(memStats.Alloc) / 1024 / 1024,
			TotalAllocMB: float64(memStats.TotalAlloc) / 1024 / 1024,
			SysMB:        float64(memStats.Sys) / 1024 / 1024,
			NumGC:        memStats.NumGC,
			HeapAllocMB:  float64(memStats.HeapAlloc) / 1024 / 1024,
			HeapSysMB:    float64(memStats.HeapSys) / 1024 / 1024,
		},
	}

	pm.metrics = append(pm.metrics, metrics)
}

// GetMetrics returns current performance metrics
func (pm *PerformanceMonitor) GetMetrics() []PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	metrics := make([]PerformanceMetrics, len(pm.metrics))
	copy(metrics, pm.metrics)
	return metrics
}

// PrintSummary prints a performance summary
func (pm *PerformanceMonitor) PrintSummary() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.metrics) == 0 {
		fmt.Println("No metrics collected")
		return
	}

	latest := pm.metrics[len(pm.metrics)-1]
	elapsed := time.Since(pm.startTime)

	fmt.Println("\n=== Performance Monitor Summary ===")
	fmt.Printf("Monitoring Duration: %v\n", elapsed)
	fmt.Printf("Total Requests: %d\n", latest.TotalRequests)
	fmt.Printf("Successful Requests: %d\n", latest.SuccessfulRequests)
	fmt.Printf("Failed Requests: %d\n", latest.FailedRequests)
	fmt.Printf("Success Rate: %.2f%%\n", 100-latest.ErrorRate)
	fmt.Printf("Error Rate: %.2f%%\n", latest.ErrorRate)
	fmt.Printf("Average Latency: %v\n", latest.AverageLatency)
	fmt.Printf("Min Latency: %v\n", latest.MinLatency)
	fmt.Printf("Max Latency: %v\n", latest.MaxLatency)
	fmt.Printf("Requests/Second: %.2f\n", latest.RequestsPerSecond)
	fmt.Printf("Active Goroutines: %d\n", latest.GoroutineCount)
	fmt.Printf("Memory Usage: %.2f MB\n", latest.MemoryUsage.AllocMB)
	fmt.Printf("Heap Usage: %.2f MB\n", latest.MemoryUsage.HeapAllocMB)
	fmt.Printf("GC Count: %d\n", latest.MemoryUsage.NumGC)
}

// SaveMetrics saves metrics to a JSON file
func (pm *PerformanceMonitor) SaveMetrics(filename string) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create metrics file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(pm.metrics)
}

// LoadTest runs a performance load test
func (pm *PerformanceMonitor) LoadTest(ctx context.Context, duration time.Duration, requestsPerSecond int) {
	fmt.Printf("Starting load test: %v duration, %d RPS target\n", duration, requestsPerSecond)
	
	interval := time.Second / time.Duration(requestsPerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	endTime := time.Now().Add(duration)
	requestCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Now().After(endTime) {
				return
			}
			
			requestCount++
			go pm.SendNotification(
				"Load Test",
				fmt.Sprintf("Load test request #%d", requestCount),
				apprise.NotifyTypeInfo,
			)
		}
	}
}

// StressTest runs a stress test with increasing load
func (pm *PerformanceMonitor) StressTest(ctx context.Context) {
	fmt.Println("Starting stress test with increasing load...")
	
	loadLevels := []int{1, 5, 10, 25, 50, 100, 200}
	testDuration := 30 * time.Second

	for _, rps := range loadLevels {
		fmt.Printf("\nTesting at %d RPS for %v...\n", rps, testDuration)
		
		loadCtx, cancel := context.WithTimeout(ctx, testDuration)
		pm.LoadTest(loadCtx, testDuration, rps)
		cancel()
		
		// Brief pause between load levels
		time.Sleep(5 * time.Second)
		
		// Print intermediate results
		pm.collectMetrics()
		if len(pm.metrics) > 0 {
			latest := pm.metrics[len(pm.metrics)-1]
			fmt.Printf("Current RPS: %.2f, Error Rate: %.2f%%, Memory: %.2f MB\n",
				latest.RequestsPerSecond, latest.ErrorRate, latest.MemoryUsage.AllocMB)
		}
	}
}

// EnduranceTest runs a long-duration test to check for memory leaks
func (pm *PerformanceMonitor) EnduranceTest(ctx context.Context, duration time.Duration) {
	fmt.Printf("Starting endurance test for %v...\n", duration)
	
	rps := 10 // Moderate load for extended duration
	pm.LoadTest(ctx, duration, rps)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run performance_monitor.go <command> [options]")
		fmt.Println("Commands:")
		fmt.Println("  monitor    - Start continuous monitoring")
		fmt.Println("  load       - Run load test")
		fmt.Println("  stress     - Run stress test")
		fmt.Println("  endurance  - Run endurance test")
		return
	}

	monitor := NewPerformanceMonitor()
	
	// Add test webhook service
	err := monitor.AddService("webhook://perf-test@httpbin.org/post")
	if err != nil {
		log.Fatalf("Failed to add test service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	command := os.Args[1]
	
	switch command {
	case "monitor":
		fmt.Println("Starting continuous performance monitoring...")
		fmt.Println("Press Ctrl+C to stop")
		
		// Start metrics collection
		go monitor.Start(ctx, 10*time.Second)
		
		// Send periodic test notifications
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for i := 0; i < 60; i++ { // Run for 5 minutes
			select {
			case <-ctx.Done():
				goto cleanup
			case <-ticker.C:
				monitor.SendNotification("Monitor Test", 
					fmt.Sprintf("Test notification #%d", i+1), 
					apprise.NotifyTypeInfo)
			}
		}
		
	case "load":
		duration := 2 * time.Minute
		rps := 20
		
		go monitor.Start(ctx, 5*time.Second)
		monitor.LoadTest(ctx, duration, rps)
		
	case "stress":
		go monitor.Start(ctx, 5*time.Second)
		monitor.StressTest(ctx)
		
	case "endurance":
		duration := 10 * time.Minute
		go monitor.Start(ctx, 30*time.Second)
		monitor.EnduranceTest(ctx, duration)
		
	default:
		fmt.Printf("Unknown command: %s\n", command)
		return
	}

cleanup:
	// Final metrics collection
	time.Sleep(2 * time.Second)
	monitor.PrintSummary()
	
	// Save metrics to file
	filename := fmt.Sprintf("performance_metrics_%s.json", time.Now().Format("20060102_150405"))
	if err := monitor.SaveMetrics(filename); err != nil {
		log.Printf("Failed to save metrics: %v", err)
	} else {
		fmt.Printf("\nMetrics saved to: %s\n", filename)
	}
}