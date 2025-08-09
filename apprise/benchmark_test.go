package apprise

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"
)

// MockService implements Service interface for benchmarking
type MockService struct {
	serviceID string
	delay     time.Duration
	failures  int
	callCount int
	mu        sync.Mutex
}

func NewMockService(serviceID string, delay time.Duration) *MockService {
	return &MockService{
		serviceID: serviceID,
		delay:     delay,
	}
}

func (m *MockService) GetServiceID() string {
	return m.serviceID
}

func (m *MockService) GetDefaultPort() int {
	return 443
}

func (m *MockService) ParseURL(serviceURL *url.URL) error {
	return nil
}

func (m *MockService) Send(ctx context.Context, req NotificationRequest) error {
	m.mu.Lock()
	m.callCount++
	failures := m.failures
	m.mu.Unlock()

	// Simulate processing delay
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// Simulate failures
	if failures > 0 {
		m.mu.Lock()
		m.failures--
		m.mu.Unlock()
		return fmt.Errorf("mock failure")
	}

	return nil
}

func (m *MockService) TestURL(serviceURL string) error {
	return nil
}

func (m *MockService) SupportsAttachments() bool {
	return true
}

func (m *MockService) GetMaxBodyLength() int {
	return 4000
}

func (m *MockService) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// Benchmark basic notification sending
func BenchmarkApprise_Notify(b *testing.B) {
	app := New()

	// Add mock services
	mockService := NewMockService("mock", 0)
	app.services = append(app.services, mockService)

	title := "Benchmark Test"
	body := "This is a benchmark notification message"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		responses := app.Notify(title, body, NotifyTypeInfo)
		if len(responses) == 0 {
			b.Fatal("Expected at least one response")
		}
	}
}

// Benchmark notification with multiple services
func BenchmarkApprise_NotifyMultipleServices(b *testing.B) {
	serviceCount := []int{1, 5, 10, 25, 50}

	for _, count := range serviceCount {
		b.Run(fmt.Sprintf("Services_%d", count), func(b *testing.B) {
			app := New()

			// Add multiple mock services
			for i := 0; i < count; i++ {
				mockService := NewMockService(fmt.Sprintf("mock_%d", i), 0)
				app.services = append(app.services, mockService)
			}

			title := "Benchmark Test"
			body := "This is a benchmark notification message"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				responses := app.Notify(title, body, NotifyTypeInfo)
				if len(responses) != count {
					b.Fatalf("Expected %d responses, got %d", count, len(responses))
				}
			}
		})
	}
}

// Benchmark notification with various delays
func BenchmarkApprise_NotifyWithDelay(b *testing.B) {
	delays := []time.Duration{0, 10 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond}

	for _, delay := range delays {
		b.Run(fmt.Sprintf("Delay_%v", delay), func(b *testing.B) {
			app := New()
			mockService := NewMockService("mock", delay)
			app.services = append(app.services, mockService)

			title := "Benchmark Test"
			body := "This is a benchmark notification message"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				responses := app.Notify(title, body, NotifyTypeInfo)
				if !responses[0].Success {
					b.Fatal("Expected successful notification")
				}
			}
		})
	}
}

// Benchmark concurrent notifications
func BenchmarkApprise_ConcurrentNotify(b *testing.B) {
	app := New()
	mockService := NewMockService("mock", 1*time.Millisecond)
	app.services = append(app.services, mockService)

	title := "Benchmark Test"
	body := "This is a benchmark notification message"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			responses := app.Notify(title, body, NotifyTypeInfo)
			if !responses[0].Success {
				b.Fatal("Expected successful notification")
			}
		}
	})
}

// Benchmark attachment operations
func BenchmarkAttachmentManager_AddFile(b *testing.B) {
	// Create a temporary file
	data := bytes.Repeat([]byte("test data "), 1000) // ~10KB file

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr := NewAttachmentManager()
		err := mgr.AddData(data, "test.txt", "text/plain")
		if err != nil {
			b.Fatalf("Failed to add attachment: %v", err)
		}
	}
}

// Benchmark attachment operations with various sizes
func BenchmarkAttachmentManager_AddFileSizes(b *testing.B) {
	sizes := []int{1024, 10240, 102400, 1048576} // 1KB, 10KB, 100KB, 1MB

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dB", size), func(b *testing.B) {
			data := bytes.Repeat([]byte("x"), size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mgr := NewAttachmentManager()
				err := mgr.AddData(data, "test.txt", "text/plain")
				if err != nil {
					b.Fatalf("Failed to add attachment: %v", err)
				}
			}
		})
	}
}

// Benchmark multiple attachment operations
func BenchmarkAttachmentManager_MultipleFiles(b *testing.B) {
	fileCounts := []int{1, 5, 10, 25}

	for _, count := range fileCounts {
		b.Run(fmt.Sprintf("Files_%d", count), func(b *testing.B) {
			data := bytes.Repeat([]byte("test "), 1000) // ~5KB per file

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mgr := NewAttachmentManager()
				for j := 0; j < count; j++ {
					err := mgr.AddData(data, fmt.Sprintf("test_%d.txt", j), "text/plain")
					if err != nil {
						b.Fatalf("Failed to add attachment %d: %v", j, err)
					}
				}
			}
		})
	}
}

// Benchmark Base64 encoding performance
func BenchmarkAttachment_Base64(b *testing.B) {
	sizes := []int{1024, 10240, 102400, 1048576} // 1KB, 10KB, 100KB, 1MB

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dB", size), func(b *testing.B) {
			data := bytes.Repeat([]byte("x"), size)
			attachment := NewMemoryAttachment(data, "test.txt", "text/plain")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := attachment.Base64()
				if err != nil {
					b.Fatalf("Failed to encode base64: %v", err)
				}
			}
		})
	}
}

// Benchmark hash generation performance
func BenchmarkAttachment_Hash(b *testing.B) {
	sizes := []int{1024, 10240, 102400, 1048576} // 1KB, 10KB, 100KB, 1MB

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dB", size), func(b *testing.B) {
			data := bytes.Repeat([]byte("x"), size)
			attachment := NewMemoryAttachment(data, "test.txt", "text/plain")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := attachment.Hash()
				if err != nil {
					b.Fatalf("Failed to generate hash: %v", err)
				}
			}
		})
	}
}

// Benchmark service registry operations
func BenchmarkServiceRegistry_Create(b *testing.B) {
	registry := NewServiceRegistry()
	registry.Register("mock", func() Service { return NewMockService("mock", 0) })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service, err := registry.Create("mock")
		if err != nil {
			b.Fatalf("Failed to create service: %v", err)
		}
		if service.GetServiceID() != "mock" {
			b.Fatal("Wrong service created")
		}
	}
}

// Benchmark URL parsing for different services
func BenchmarkService_ParseURL(b *testing.B) {
	testCases := []struct {
		name string
		url  string
	}{
		{"Discord", "discord://1234567890/abcdefghijklmnopqrstuvwxyz"},
		{"Slack", "slack://T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX/general"},
		{"Telegram", "tgram://1234567890123/123456789"},
		{"Email", "mailto://user:pass@smtp.gmail.com:587/recipient@domain.com"},
		{"Webhook", "webhook://api.example.com/webhook?token=secret"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			app := New()
			parsedURL, err := url.Parse(tc.url)
			if err != nil {
				b.Fatalf("Failed to parse URL: %v", err)
			}

			service, err := app.registry.Create(parsedURL.Scheme)
			if err != nil {
				b.Skip("Service not available:", parsedURL.Scheme)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := service.ParseURL(parsedURL)
				if err != nil {
					b.Fatalf("Failed to parse URL: %v", err)
				}
			}
		})
	}
}

// Benchmark HTTP webhook performance with real server
func BenchmarkWebhookService_RealHTTP(b *testing.B) {
	// Skip this test as it requires network connectivity
	b.Skip("Skipping network-dependent benchmark")
}

// Benchmark notification with attachments
func BenchmarkApprise_NotifyWithAttachments(b *testing.B) {
	app := New()
	mockService := NewMockService("mock", 0)
	app.services = append(app.services, mockService)

	// Add test attachments
	smallData := bytes.Repeat([]byte("test"), 250)   // 1KB
	largeData := bytes.Repeat([]byte("test"), 25000) // 100KB

	title := "Benchmark Test"
	body := "This is a benchmark notification with attachments"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear previous attachments
		app.ClearAttachments()

		// Add attachments
		app.AddAttachmentData(smallData, "small.txt", "text/plain")
		app.AddAttachmentData(largeData, "large.txt", "text/plain")

		responses := app.Notify(title, body, NotifyTypeInfo)
		if !responses[0].Success {
			b.Fatal("Expected successful notification")
		}
	}
}

// Benchmark memory allocation patterns
func BenchmarkApprise_MemoryAllocation(b *testing.B) {
	title := "Benchmark Test"
	body := "This is a benchmark notification message for memory testing"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := New()
		mockService := NewMockService("mock", 0)
		app.services = append(app.services, mockService)

		responses := app.Notify(title, body, NotifyTypeInfo)
		if !responses[0].Success {
			b.Fatal("Expected successful notification")
		}
	}
}

// Benchmark timeout handling
func BenchmarkApprise_Timeout(b *testing.B) {
	app := New()
	app.SetTimeout(50 * time.Millisecond) // Short timeout

	// Add a slow mock service that will timeout
	slowService := NewMockService("slow", 100*time.Millisecond)
	app.services = append(app.services, slowService)

	title := "Benchmark Test"
	body := "This is a benchmark notification for timeout testing"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		responses := app.Notify(title, body, NotifyTypeInfo)
		// Should complete but may timeout (we just measure the performance)
		_ = responses[0].Success
	}
}
