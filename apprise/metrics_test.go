package apprise

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMetricsManager(t *testing.T) {
	mm := NewMetricsManager("test")
	
	if mm == nil {
		t.Fatal("Expected non-nil MetricsManager")
	}
	
	if mm.serviceMetrics == nil {
		t.Fatal("Expected initialized service metrics map")
	}
	
	if mm.notificationsTotal == nil {
		t.Fatal("Expected initialized notifications counter")
	}
}

func TestMetricsManagerRegister(t *testing.T) {
	// Create a custom registry for testing
	_ = prometheus.NewRegistry()
	
	mm := NewMetricsManager("test")
	
	// Test registration
	err := mm.Register()
	if err != nil {
		t.Fatalf("Failed to register metrics: %v", err)
	}
	
	// Test double registration (should not error)
	err = mm.Register()
	if err != nil {
		t.Fatalf("Failed to register metrics twice: %v", err)
	}
	
	if !mm.registryCreated {
		t.Fatal("Expected registry created flag to be true")
	}
}

func TestRecordNotification(t *testing.T) {
	mm := NewMetricsManager("test")
	err := mm.Register()
	if err != nil {
		t.Fatalf("Failed to register metrics: %v", err)
	}
	
	// Record a successful notification
	service := "discord"
	notificationType := "info"
	status := "success"
	duration := 100 * time.Millisecond
	
	mm.RecordNotification(service, notificationType, status, duration)
	
	// Verify service metrics were updated
	serviceMetrics := mm.GetServiceMetrics(service)
	if serviceMetrics.TotalSent != 1 {
		t.Errorf("Expected TotalSent=1, got %d", serviceMetrics.TotalSent)
	}
	
	if serviceMetrics.Service != service {
		t.Errorf("Expected Service=%s, got %s", service, serviceMetrics.Service)
	}
	
	if serviceMetrics.AvgDuration != duration {
		t.Errorf("Expected AvgDuration=%v, got %v", duration, serviceMetrics.AvgDuration)
	}
	
	// Record a failed notification
	mm.RecordNotification(service, notificationType, "failed", duration)
	
	serviceMetrics = mm.GetServiceMetrics(service)
	if serviceMetrics.TotalSent != 1 {
		t.Errorf("Expected TotalSent=1, got %d", serviceMetrics.TotalSent)
	}
	
	if serviceMetrics.TotalFailed != 1 {
		t.Errorf("Expected TotalFailed=1, got %d", serviceMetrics.TotalFailed)
	}
}

func TestRecordNotificationError(t *testing.T) {
	mm := NewMetricsManager("test")
	err := mm.Register()
	if err != nil {
		t.Fatalf("Failed to register metrics: %v", err)
	}
	
	service := "slack"
	reason := "timeout"
	errorType := "network"
	
	mm.RecordNotificationError(service, reason, errorType)
	
	serviceMetrics := mm.GetServiceMetrics(service)
	if serviceMetrics.TotalFailed != 1 {
		t.Errorf("Expected TotalFailed=1, got %d", serviceMetrics.TotalFailed)
	}
}

func TestRecordHTTPRequest(t *testing.T) {
	mm := NewMetricsManager("test")
	err := mm.Register()
	if err != nil {
		t.Fatalf("Failed to register metrics: %v", err)
	}
	
	// Record successful request
	mm.RecordHTTPRequest("GET", "/api/notify", 200, 50*time.Millisecond)
	
	// Record failed request
	mm.RecordHTTPRequest("POST", "/api/notify", 500, 100*time.Millisecond)
	
	// Note: In a real test, we'd need to check the actual metric values
	// which requires accessing the registry and parsing metric families
}

func TestUpdateGauges(t *testing.T) {
	mm := NewMetricsManager("test")
	err := mm.Register()
	if err != nil {
		t.Fatalf("Failed to register metrics: %v", err)
	}
	
	// Test gauge updates
	mm.UpdateActiveConnections(5)
	mm.UpdateServicesConfigured(10)
	mm.UpdateQueueSize(15)
	mm.UpdateMemoryUsage(1024)
	mm.UpdateGoroutineCount(20)
	
	// Verify gauges can be updated without error
	// In a real implementation, we'd verify the actual values
}

func TestRecordBatchSize(t *testing.T) {
	mm := NewMetricsManager("test")
	err := mm.Register()
	if err != nil {
		t.Fatalf("Failed to register metrics: %v", err)
	}
	
	batchSizes := []int{1, 5, 10, 25, 100}
	for _, size := range batchSizes {
		mm.RecordBatchSize(size)
	}
	
	// Verify batch sizes can be recorded without error
}

func TestGetAllServiceMetrics(t *testing.T) {
	mm := NewMetricsManager("test")
	
	// Record metrics for multiple services
	services := []string{"discord", "slack", "email"}
	for _, service := range services {
		mm.RecordNotification(service, "info", "success", 100*time.Millisecond)
	}
	
	allMetrics := mm.GetAllServiceMetrics()
	
	if len(allMetrics) != len(services) {
		t.Errorf("Expected %d services, got %d", len(services), len(allMetrics))
	}
	
	for _, service := range services {
		if _, exists := allMetrics[service]; !exists {
			t.Errorf("Expected metrics for service %s", service)
		}
	}
}

func TestHTTPMiddleware(t *testing.T) {
	mm := NewMetricsManager("test")
	err := mm.Register()
	if err != nil {
		t.Fatalf("Failed to register metrics: %v", err)
	}
	
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Wrap with middleware
	wrappedHandler := mm.HTTPMiddleware(handler)
	
	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	// Execute request
	wrappedHandler.ServeHTTP(w, req)
	
	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	
	// Test default status
	if rw.statusCode != http.StatusOK {
		t.Errorf("Expected default status 200, got %d", rw.statusCode)
	}
	
	// Test status code capture
	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rw.statusCode)
	}
}

func TestMetricsHandler(t *testing.T) {
	mm := NewMetricsManager("test")
	err := mm.Register()
	if err != nil {
		t.Fatalf("Failed to register metrics: %v", err)
	}
	
	// Record some metrics
	mm.RecordNotification("discord", "info", "success", 100*time.Millisecond)
	mm.UpdateActiveConnections(5)
	
	// Create test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	// Get handler and serve
	handler := mm.Handler()
	handler.ServeHTTP(w, req)
	
	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	body := w.Body.String()
	if !strings.Contains(body, "# HELP") {
		t.Errorf("Expected Prometheus format output, got: %s", body)
	}
	
	// Check for some expected metrics
	expectedMetrics := []string{
		"test_notifications_total",
		"test_active_connections",
	}
	
	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Expected metric %s in output", metric)
		}
	}
}

func TestReset(t *testing.T) {
	mm := NewMetricsManager("test")
	
	// Record some metrics
	mm.RecordNotification("discord", "info", "success", 100*time.Millisecond)
	mm.UpdateActiveConnections(10)
	
	// Verify metrics exist
	serviceMetrics := mm.GetServiceMetrics("discord")
	if serviceMetrics.TotalSent != 1 {
		t.Errorf("Expected TotalSent=1 before reset, got %d", serviceMetrics.TotalSent)
	}
	
	// Reset metrics
	mm.Reset()
	
	// Verify reset
	serviceMetrics = mm.GetServiceMetrics("discord")
	if serviceMetrics.TotalSent != 0 {
		t.Errorf("Expected TotalSent=0 after reset, got %d", serviceMetrics.TotalSent)
	}
	
	allMetrics := mm.GetAllServiceMetrics()
	if len(allMetrics) != 0 {
		t.Errorf("Expected 0 services after reset, got %d", len(allMetrics))
	}
}

func TestConcurrentAccess(t *testing.T) {
	mm := NewMetricsManager("test")
	
	// Test concurrent access to service metrics
	done := make(chan bool, 2)
	
	// Goroutine 1: Record notifications
	go func() {
		for i := 0; i < 100; i++ {
			mm.RecordNotification("discord", "info", "success", 10*time.Millisecond)
		}
		done <- true
	}()
	
	// Goroutine 2: Read metrics
	go func() {
		for i := 0; i < 100; i++ {
			_ = mm.GetServiceMetrics("discord")
		}
		done <- true
	}()
	
	// Wait for both goroutines
	<-done
	<-done
	
	// Verify final count
	serviceMetrics := mm.GetServiceMetrics("discord")
	if serviceMetrics.TotalSent != 100 {
		t.Errorf("Expected TotalSent=100, got %d", serviceMetrics.TotalSent)
	}
}

func BenchmarkRecordNotification(b *testing.B) {
	mm := NewMetricsManager("bench")
	mm.Register()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mm.RecordNotification("discord", "info", "success", 10*time.Millisecond)
		}
	})
}

func BenchmarkGetServiceMetrics(b *testing.B) {
	mm := NewMetricsManager("bench")
	
	// Populate with some data
	for i := 0; i < 10; i++ {
		mm.RecordNotification("discord", "info", "success", 10*time.Millisecond)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = mm.GetServiceMetrics("discord")
		}
	})
}