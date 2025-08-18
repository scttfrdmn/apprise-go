package apprise

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsManager handles Prometheus metrics collection and export
type MetricsManager struct {
	// Counters
	notificationsTotal    *prometheus.CounterVec
	notificationsFailed   *prometheus.CounterVec
	httpRequestsTotal     *prometheus.CounterVec
	httpRequestsFailed    *prometheus.CounterVec

	// Histograms
	notificationDuration *prometheus.HistogramVec
	httpRequestDuration  *prometheus.HistogramVec

	// Gauges
	activeConnections     prometheus.Gauge
	servicesConfigured    prometheus.Gauge
	queueSize             prometheus.Gauge
	memoryUsage           prometheus.Gauge
	goroutineCount        prometheus.Gauge

	// Summary
	notificationBatchSize prometheus.Summary

	// Internal tracking
	mu              sync.RWMutex
	serviceMetrics  map[string]*PrometheusServiceMetrics
	registryCreated bool
}

// PrometheusServiceMetrics tracks per-service statistics for Prometheus
type PrometheusServiceMetrics struct {
	Service       string
	TotalSent     uint64
	TotalFailed   uint64
	LastSent      time.Time
	LastError     error
	AvgDuration   time.Duration
	mu            sync.RWMutex
}

// NewMetricsManager creates a new metrics manager with all metrics initialized
func NewMetricsManager(namespace string) *MetricsManager {
	if namespace == "" {
		namespace = "apprise"
	}

	mm := &MetricsManager{
		serviceMetrics: make(map[string]*PrometheusServiceMetrics),
	}

	// Initialize counters
	mm.notificationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "notifications_total",
			Help:      "Total number of notifications sent by service and type",
		},
		[]string{"service", "type", "status"},
	)

	mm.notificationsFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "notifications_failed_total",
			Help:      "Total number of failed notifications by service and reason",
		},
		[]string{"service", "reason", "error_type"},
	)

	mm.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests by method, endpoint and status",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	mm.httpRequestsFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_failed_total",
			Help:      "Total number of failed HTTP requests by method and endpoint",
		},
		[]string{"method", "endpoint", "error_type"},
	)

	// Initialize histograms
	mm.notificationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "notification_duration_seconds",
			Help:      "Time spent processing notifications by service",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"service", "type"},
	)

	mm.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latencies by method and endpoint",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Initialize gauges
	mm.activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "active_connections",
			Help:      "Number of active HTTP connections",
		},
	)

	mm.servicesConfigured = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "services_configured_total",
			Help:      "Number of notification services configured",
		},
	)

	mm.queueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "notification_queue_size",
			Help:      "Number of notifications waiting in queue",
		},
	)

	mm.memoryUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "memory_usage_bytes",
			Help:      "Current memory usage in bytes",
		},
	)

	mm.goroutineCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "goroutines_total",
			Help:      "Number of active goroutines",
		},
	)

	// Initialize summary
	mm.notificationBatchSize = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace:  namespace,
			Name:       "notification_batch_size",
			Help:       "Size of notification batches processed",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
	)

	return mm
}

// Register registers all metrics with the default Prometheus registry
func (mm *MetricsManager) Register() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.registryCreated {
		return nil
	}

	collectors := []prometheus.Collector{
		mm.notificationsTotal,
		mm.notificationsFailed,
		mm.httpRequestsTotal,
		mm.httpRequestsFailed,
		mm.notificationDuration,
		mm.httpRequestDuration,
		mm.activeConnections,
		mm.servicesConfigured,
		mm.queueSize,
		mm.memoryUsage,
		mm.goroutineCount,
		mm.notificationBatchSize,
	}

	for _, collector := range collectors {
		if err := prometheus.Register(collector); err != nil {
			// If already registered, ignore the error
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				return err
			}
		}
	}

	mm.registryCreated = true
	return nil
}

// RecordNotification records a notification attempt with timing
func (mm *MetricsManager) RecordNotification(service, notificationType, status string, duration time.Duration) {
	mm.notificationsTotal.WithLabelValues(service, notificationType, status).Inc()
	mm.notificationDuration.WithLabelValues(service, notificationType).Observe(duration.Seconds())

	// Update service metrics
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	serviceMetric, exists := mm.serviceMetrics[service]
	if !exists {
		serviceMetric = &PrometheusServiceMetrics{
			Service: service,
		}
		mm.serviceMetrics[service] = serviceMetric
	}

	serviceMetric.mu.Lock()
	defer serviceMetric.mu.Unlock()
	
	if status == "success" {
		serviceMetric.TotalSent++
		serviceMetric.LastSent = time.Now()
	} else {
		serviceMetric.TotalFailed++
	}

	// Calculate rolling average duration
	if serviceMetric.AvgDuration == 0 {
		serviceMetric.AvgDuration = duration
	} else {
		serviceMetric.AvgDuration = (serviceMetric.AvgDuration + duration) / 2
	}
}

// RecordNotificationError records a failed notification with error details
func (mm *MetricsManager) RecordNotificationError(service, reason, errorType string) {
	mm.notificationsFailed.WithLabelValues(service, reason, errorType).Inc()

	// Update service error tracking
	mm.mu.Lock()
	serviceMetric, exists := mm.serviceMetrics[service]
	if !exists {
		serviceMetric = &PrometheusServiceMetrics{
			Service: service,
		}
		mm.serviceMetrics[service] = serviceMetric
	}
	mm.mu.Unlock()

	serviceMetric.mu.Lock()
	serviceMetric.TotalFailed++
	serviceMetric.mu.Unlock()
}

// RecordHTTPRequest records an HTTP request with timing
func (mm *MetricsManager) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	statusStr := strconv.Itoa(statusCode)
	mm.httpRequestsTotal.WithLabelValues(method, endpoint, statusStr).Inc()
	mm.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())

	if statusCode >= 400 {
		errorType := "client_error"
		if statusCode >= 500 {
			errorType = "server_error"
		}
		mm.httpRequestsFailed.WithLabelValues(method, endpoint, errorType).Inc()
	}
}

// UpdateActiveConnections updates the active connections gauge
func (mm *MetricsManager) UpdateActiveConnections(count int) {
	mm.activeConnections.Set(float64(count))
}

// UpdateServicesConfigured updates the configured services count
func (mm *MetricsManager) UpdateServicesConfigured(count int) {
	mm.servicesConfigured.Set(float64(count))
}

// UpdateQueueSize updates the notification queue size
func (mm *MetricsManager) UpdateQueueSize(size int) {
	mm.queueSize.Set(float64(size))
}

// UpdateMemoryUsage updates memory usage in bytes
func (mm *MetricsManager) UpdateMemoryUsage(bytes uint64) {
	mm.memoryUsage.Set(float64(bytes))
}

// UpdateGoroutineCount updates the active goroutine count
func (mm *MetricsManager) UpdateGoroutineCount(count int) {
	mm.goroutineCount.Set(float64(count))
}

// RecordBatchSize records the size of a notification batch
func (mm *MetricsManager) RecordBatchSize(size int) {
	mm.notificationBatchSize.Observe(float64(size))
}

// GetServiceMetrics returns current metrics for a specific service
func (mm *MetricsManager) GetServiceMetrics(service string) *PrometheusServiceMetrics {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	serviceMetric, exists := mm.serviceMetrics[service]
	if !exists {
		return &PrometheusServiceMetrics{
			Service: service,
		}
	}

	// Return a copy to avoid concurrent access issues
	serviceMetric.mu.RLock()
	defer serviceMetric.mu.RUnlock()
	
	return &PrometheusServiceMetrics{
		Service:     serviceMetric.Service,
		TotalSent:   serviceMetric.TotalSent,
		TotalFailed: serviceMetric.TotalFailed,
		LastSent:    serviceMetric.LastSent,
		LastError:   serviceMetric.LastError,
		AvgDuration: serviceMetric.AvgDuration,
	}
}

// GetAllServiceMetrics returns metrics for all services
func (mm *MetricsManager) GetAllServiceMetrics() map[string]*PrometheusServiceMetrics {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	result := make(map[string]*PrometheusServiceMetrics)
	for service := range mm.serviceMetrics {
		result[service] = mm.GetServiceMetrics(service)
	}
	return result
}

// Handler returns an HTTP handler for serving Prometheus metrics
func (mm *MetricsManager) Handler() http.Handler {
	return promhttp.Handler()
}

// HandlerWithGatherer returns an HTTP handler with custom gatherer
func (mm *MetricsManager) HandlerWithGatherer(gatherer prometheus.Gatherer) http.Handler {
	return promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
}

// HTTPMiddleware creates middleware for automatically recording HTTP metrics
func (mm *MetricsManager) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer that captures status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Update active connections
		mm.UpdateActiveConnections(1) // Increment would require tracking
		defer func() {
			// Record the request
			duration := time.Since(start)
			mm.RecordHTTPRequest(r.Method, r.URL.Path, rw.statusCode, duration)
		}()

		next.ServeHTTP(rw, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Reset resets all metrics (useful for testing)
func (mm *MetricsManager) Reset() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	// Clear service metrics
	mm.serviceMetrics = make(map[string]*PrometheusServiceMetrics)
	
	// Reset gauges to zero
	mm.activeConnections.Set(0)
	mm.servicesConfigured.Set(0)
	mm.queueSize.Set(0)
	mm.memoryUsage.Set(0)
	mm.goroutineCount.Set(0)
}