package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

// BenchmarkSingleServiceNotification benchmarks notification to a single service
func BenchmarkSingleServiceNotification(b *testing.B) {
	app := apprise.New()
	
	// Add a webhook service for consistent performance testing
	webhookURL := "webhook://test-key@httpbin.org/post"
	err := app.Add(webhookURL)
	if err != nil {
		b.Fatalf("Failed to add webhook service: %v", err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		responses := app.Notify("Benchmark Test", "Performance test notification", apprise.NotifyTypeInfo)
		if len(responses) != 1 {
			b.Errorf("Expected 1 response, got %d", len(responses))
		}
	}
}

// BenchmarkMultipleServiceNotification benchmarks notification to multiple services
func BenchmarkMultipleServiceNotification(b *testing.B) {
	app := apprise.New()
	
	// Add multiple webhook services to simulate real-world usage
	services := []string{
		"webhook://test1@httpbin.org/post",
		"webhook://test2@httpbin.org/post", 
		"webhook://test3@httpbin.org/post",
		"webhook://test4@httpbin.org/post",
		"webhook://test5@httpbin.org/post",
	}
	
	for _, service := range services {
		err := app.Add(service)
		if err != nil {
			b.Fatalf("Failed to add service %s: %v", service, err)
		}
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		responses := app.Notify("Benchmark Test", "Multi-service performance test", apprise.NotifyTypeInfo)
		if len(responses) != len(services) {
			b.Errorf("Expected %d responses, got %d", len(services), len(responses))
		}
	}
}

// BenchmarkConcurrentNotifications benchmarks concurrent notifications
func BenchmarkConcurrentNotifications(b *testing.B) {
	app := apprise.New()
	
	webhookURL := "webhook://concurrent-test@httpbin.org/post"
	err := app.Add(webhookURL)
	if err != nil {
		b.Fatalf("Failed to add webhook service: %v", err)
	}

	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			responses := app.Notify("Concurrent Test", "Parallel notification test", apprise.NotifyTypeInfo)
			if len(responses) != 1 {
				b.Errorf("Expected 1 response, got %d", len(responses))
			}
		}
	})
}

// BenchmarkServiceCreation benchmarks service creation and configuration
func BenchmarkServiceCreation(b *testing.B) {
	urls := []string{
		"discord://webhook_id/webhook_token@discord.com",
		"slack://bottoken@workspace.slack.com/channel",
		"telegram://bottoken@telegram/chatid",
		"pushover://userkey@appkey",
		"email://smtp.gmail.com/user@example.com",
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		app := apprise.New()
		url := urls[i%len(urls)]
		
		err := app.Add(url)
		if err != nil {
			// Expected for some URLs without proper credentials
			continue
		}
	}
}

// BenchmarkAttachmentHandling benchmarks notification with attachments
func BenchmarkAttachmentHandling(b *testing.B) {
	app := apprise.New()
	
	webhookURL := "webhook://attachment-test@httpbin.org/post"
	err := app.Add(webhookURL)
	if err != nil {
		b.Fatalf("Failed to add webhook service: %v", err)
	}

	// Add test attachments
	testData := make([]byte, 1024) // 1KB test file
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	
	err = app.AddAttachmentData(testData, "test.bin", "application/octet-stream")
	if err != nil {
		b.Fatalf("Failed to add attachment: %v", err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		responses := app.Notify("Attachment Test", "Performance test with attachment", apprise.NotifyTypeInfo)
		if len(responses) != 1 {
			b.Errorf("Expected 1 response, got %d", len(responses))
		}
	}
}

// BenchmarkMemoryUsage benchmarks memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	app := apprise.New()
	webhookURL := "webhook://memory-test@httpbin.org/post"
	err := app.Add(webhookURL)
	if err != nil {
		b.Fatalf("Failed to add webhook service: %v", err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		app.Notify("Memory Test", "Testing memory allocation patterns", apprise.NotifyTypeInfo)
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
}

// BenchmarkLargePayload benchmarks handling of large notification payloads
func BenchmarkLargePayload(b *testing.B) {
	app := apprise.New()
	
	webhookURL := "webhook://large-payload@httpbin.org/post"
	err := app.Add(webhookURL)
	if err != nil {
		b.Fatalf("Failed to add webhook service: %v", err)
	}

	// Create large payload (10KB message)
	largeBody := string(make([]byte, 10240))
	for i := range largeBody {
		largeBody = largeBody[:i] + "A" + largeBody[i+1:]
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		responses := app.Notify("Large Payload Test", largeBody, apprise.NotifyTypeInfo)
		if len(responses) != 1 {
			b.Errorf("Expected 1 response, got %d", len(responses))
		}
	}
}

// BenchmarkServiceTypeVariations benchmarks different service types
func BenchmarkServiceTypeVariations(b *testing.B) {
	serviceTypes := map[string]string{
		"webhook":   "webhook://test@httpbin.org/post",
		"discord":   "discord://123/abc@discord.com",
		"slack":     "slack://xoxb-test@workspace.slack.com/general",
		"telegram":  "telegram://123:ABC@telegram/123456",
		"pushover":  "pushover://user@app.pushover.net",
	}
	
	for name, url := range serviceTypes {
		b.Run(name, func(b *testing.B) {
			app := apprise.New()
			err := app.Add(url)
			if err != nil {
				b.Skipf("Skipping %s due to configuration error: %v", name, err)
			}
			
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				responses := app.Notify("Service Type Test", 
					fmt.Sprintf("Testing %s service performance", name), 
					apprise.NotifyTypeInfo)
				if len(responses) != 1 {
					b.Errorf("Expected 1 response, got %d", len(responses))
				}
			}
		})
	}
}

// BenchmarkErrorHandling benchmarks error handling performance
func BenchmarkErrorHandling(b *testing.B) {
	app := apprise.New()
	
	// Add service with invalid URL that will cause errors
	invalidURL := "webhook://test@invalid-domain-that-does-not-exist.com/post"
	err := app.Add(invalidURL)
	if err != nil {
		b.Fatalf("Failed to add webhook service: %v", err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		responses := app.Notify("Error Test", "Testing error handling performance", apprise.NotifyTypeError)
		if len(responses) != 1 {
			b.Errorf("Expected 1 response, got %d", len(responses))
		}
		
		// Verify error was handled
		if responses[0].Success {
			b.Error("Expected error response, but got success")
		}
	}
}

// BenchmarkHighConcurrency benchmarks high concurrency scenarios
func BenchmarkHighConcurrency(b *testing.B) {
	app := apprise.New()
	
	webhookURL := "webhook://concurrency@httpbin.org/post"
	err := app.Add(webhookURL)
	if err != nil {
		b.Fatalf("Failed to add webhook service: %v", err)
	}

	// Test with different concurrency levels
	concurrencyLevels := []int{1, 10, 50, 100, 500}
	
	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(concurrency)
				
				start := time.Now()
				
				for j := 0; j < concurrency; j++ {
					go func() {
						defer wg.Done()
						app.Notify("Concurrency Test", 
							fmt.Sprintf("High concurrency test with %d goroutines", concurrency), 
							apprise.NotifyTypeInfo)
					}()
				}
				
				wg.Wait()
				
				duration := time.Since(start)
				b.ReportMetric(float64(duration.Nanoseconds()), "ns/op")
				b.ReportMetric(float64(concurrency)/duration.Seconds(), "ops/sec")
			}
		})
	}
}

// BenchmarkTimeoutHandling benchmarks timeout scenarios
func BenchmarkTimeoutHandling(b *testing.B) {
	app := apprise.New()
	
	// Set short timeout for testing
	app.SetTimeout(100 * time.Millisecond)
	
	// Use a slow-responding service (httpbin delay endpoint)
	slowURL := "webhook://timeout-test@httpbin.org/delay/1" // 1 second delay
	err := app.Add(slowURL)
	if err != nil {
		b.Fatalf("Failed to add slow webhook service: %v", err)
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		responses := app.Notify("Timeout Test", "Testing timeout handling", apprise.NotifyTypeWarning)
		if len(responses) != 1 {
			b.Errorf("Expected 1 response, got %d", len(responses))
		}
		
		// Should timeout and return error
		if responses[0].Success {
			b.Error("Expected timeout error, but got success")
		}
	}
}

// BenchmarkGoroutineUsage benchmarks goroutine efficiency
func BenchmarkGoroutineUsage(b *testing.B) {
	initialGoroutines := runtime.NumGoroutine()
	
	app := apprise.New()
	
	// Add multiple services
	for i := 0; i < 10; i++ {
		webhookURL := fmt.Sprintf("webhook://goroutine-test-%d@httpbin.org/post", i)
		err := app.Add(webhookURL)
		if err != nil {
			b.Fatalf("Failed to add webhook service %d: %v", i, err)
		}
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		app.Notify("Goroutine Test", "Testing goroutine usage efficiency", apprise.NotifyTypeInfo)
	}
	
	// Allow goroutines to finish
	time.Sleep(100 * time.Millisecond)
	
	finalGoroutines := runtime.NumGoroutine()
	b.ReportMetric(float64(finalGoroutines-initialGoroutines), "goroutines-delta")
}

// Performance stress test for extended runs
func BenchmarkStressTest(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping stress test in short mode")
	}
	
	app := apprise.New()
	
	// Add multiple services
	services := []string{
		"webhook://stress1@httpbin.org/post",
		"webhook://stress2@httpbin.org/post",
		"webhook://stress3@httpbin.org/post",
	}
	
	for _, service := range services {
		err := app.Add(service)
		if err != nil {
			b.Fatalf("Failed to add service: %v", err)
		}
	}

	b.ResetTimer()
	
	// Run for extended period to detect memory leaks or performance degradation
	for i := 0; i < b.N; i++ {
		if i%100 == 0 {
			runtime.GC() // Occasional GC to test memory management
		}
		
		responses := app.Notify("Stress Test", 
			fmt.Sprintf("Extended stress test iteration %d", i), 
			apprise.NotifyTypeInfo)
		
		if len(responses) != len(services) {
			b.Errorf("Expected %d responses, got %d", len(services), len(responses))
		}
		
		// Verify all responses processed
		for _, resp := range responses {
			if resp.Duration == 0 {
				b.Error("Response duration should not be zero")
			}
		}
	}
}

// Example benchmark runner that can be called from tests
func RunPerformanceProfile() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	app := apprise.New()
	app.Add("webhook://profile@httpbin.org/post")
	
	start := time.Now()
	iterations := 1000
	
	for i := 0; i < iterations && ctx.Err() == nil; i++ {
		app.Notify("Profile Test", fmt.Sprintf("Iteration %d", i), apprise.NotifyTypeInfo)
		
		if i%100 == 0 {
			elapsed := time.Since(start)
			rate := float64(i) / elapsed.Seconds()
			fmt.Printf("Completed %d iterations in %v (%.2f ops/sec)\n", i, elapsed, rate)
		}
	}
	
	total := time.Since(start)
	fmt.Printf("Total: %d iterations in %v (%.2f ops/sec average)\n", 
		iterations, total, float64(iterations)/total.Seconds())
}