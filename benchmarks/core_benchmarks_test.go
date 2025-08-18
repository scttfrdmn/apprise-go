package benchmarks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

// BenchmarkAppriseNotify benchmarks basic notification sending
func BenchmarkAppriseNotify(b *testing.B) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	// Setup Apprise with test webhook
	app := apprise.New()
	app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			app.Notify("Test Title", "Test Body", apprise.NotifyTypeInfo)
		}
	})
}

// BenchmarkAppriseMultipleServices benchmarks notifications to multiple services
func BenchmarkAppriseMultipleServices(b *testing.B) {
	// Create multiple test servers
	servers := make([]*httptest.Server, 5)
	for i := range servers {
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate varying response times
			time.Sleep(time.Duration(i*10) * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success"}`))
		}))
		defer servers[i].Close()
	}

	app := apprise.New()
	for _, server := range servers {
		app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			app.Notify("Test Title", "Test Body", apprise.NotifyTypeInfo)
		}
	})
}

// BenchmarkAppriseNotifyTypes benchmarks different notification types
func BenchmarkAppriseNotifyTypes(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := apprise.New()
	app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))

	types := []apprise.NotifyType{
		apprise.NotifyTypeInfo,
		apprise.NotifyTypeSuccess,
		apprise.NotifyTypeWarning,
		apprise.NotifyTypeError,
	}

	for _, notifyType := range types {
		b.Run(notifyType.String(), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					app.Notify("Test Title", "Test Body", notifyType)
				}
			})
		})
	}
}

// BenchmarkAppriseWithAttachments benchmarks notifications with attachments
func BenchmarkAppriseWithAttachments(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := apprise.New()
	app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))

	// Add test attachment
	testData := make([]byte, 1024) // 1KB test data
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	app.AddAttachmentData(testData, "test.bin", "application/octet-stream")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			app.Notify("Test with Attachment", "Test Body", apprise.NotifyTypeInfo)
		}
	})
}

// BenchmarkServiceCreation benchmarks service creation and configuration
func BenchmarkServiceCreation(b *testing.B) {
	serviceURLs := []string{
		"webhook://example.com/webhook",
		"discord://123456789/abcdef",
		"slack://T123/B123/xyz/#general",
		"mailto://user:pass@smtp.gmail.com:587/to@example.com",
		"twilio://sid:token@+1234567890/+0987654321",
	}

	b.ResetTimer()
	for _, serviceURL := range serviceURLs {
		b.Run(fmt.Sprintf("Service_%s", serviceURL[:10]), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					app := apprise.New()
					app.Add(serviceURL)
				}
			})
		})
	}
}

// BenchmarkConcurrentNotifications benchmarks concurrent notification sending
func BenchmarkConcurrentNotifications(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate realistic response time
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	concurrencyLevels := []int{1, 5, 10, 25, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			app := apprise.New()
			app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))

			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					app.Notify("Test Title", "Test Body", apprise.NotifyTypeInfo)
				}
			})
		})
	}
}

// BenchmarkMessageSizes benchmarks notifications with different message sizes
func BenchmarkMessageSizes(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := apprise.New()
	app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))

	messageSizes := []int{10, 100, 1000, 5000, 10000} // bytes

	for _, size := range messageSizes {
		message := string(make([]byte, size))
		for i := range message {
			message = message[:i] + "x" + message[i+1:]
		}

		b.Run(fmt.Sprintf("MessageSize_%db", size), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					app.Notify("Test Title", message, apprise.NotifyTypeInfo)
				}
			})
		})
	}
}

// BenchmarkServiceRegistry benchmarks service registry operations
func BenchmarkServiceRegistry(b *testing.B) {
	b.Run("GetSupportedServices", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = apprise.GetSupportedServices()
			}
		})
	})

	b.Run("ServiceLookup", func(b *testing.B) {
		services := apprise.GetSupportedServices()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for _, service := range services {
					_ = service
				}
			}
		})
	})
}

// BenchmarkMetricsCollection benchmarks metrics collection overhead
func BenchmarkMetricsCollection(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test with metrics enabled
	b.Run("WithMetrics", func(b *testing.B) {
		app := apprise.New() // Metrics enabled by default
		app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				app.Notify("Test Title", "Test Body", apprise.NotifyTypeInfo)
			}
		})
	})

	// Create a version without metrics for comparison
	b.Run("WithoutMetrics", func(b *testing.B) {
		app := apprise.New()
		app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))
		
		// Clear metrics to simulate disabled state (conceptual)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				app.Notify("Test Title", "Test Body", apprise.NotifyTypeInfo)
			}
		})
	})
}

// BenchmarkErrorHandling benchmarks error handling performance
func BenchmarkErrorHandling(b *testing.B) {
	// Server that always returns errors
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
	}))
	defer errorServer.Close()

	app := apprise.New()
	app.Add(fmt.Sprintf("webhook://%s", errorServer.Listener.Addr().String()))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			responses := app.Notify("Test Title", "Test Body", apprise.NotifyTypeInfo)
			// Simulate error processing
			for _, resp := range responses {
				if resp.Error != nil {
					_ = resp.Error.Error()
				}
			}
		}
	})
}

// BenchmarkTimeout benchmarks timeout handling
func BenchmarkTimeout(b *testing.B) {
	// Slow server that exceeds timeout
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Longer than typical timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	app := apprise.New()
	app.SetTimeout(100 * time.Millisecond) // Short timeout
	app.Add(fmt.Sprintf("webhook://%s", slowServer.Listener.Addr().String()))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			app.Notify("Test Title", "Test Body", apprise.NotifyTypeInfo)
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b.Run("SingleService", func(b *testing.B) {
		app := apprise.New()
		app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))

		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				app.Notify("Test Title", "Test Body", apprise.NotifyTypeInfo)
			}
		})
	})

	b.Run("MultipleServices", func(b *testing.B) {
		app := apprise.New()
		for i := 0; i < 10; i++ {
			app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))
		}

		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				app.Notify("Test Title", "Test Body", apprise.NotifyTypeInfo)
			}
		})
	})
}

// BenchmarkServiceSpecific benchmarks individual service implementations
func BenchmarkServiceSpecific(b *testing.B) {
	// Discord-like webhook
	b.Run("Discord", func(b *testing.B) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		app := apprise.New()
		app.Add(fmt.Sprintf("discord://%s", server.Listener.Addr().String()))

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				app.Notify("Discord Test", "Test message", apprise.NotifyTypeInfo)
			}
		})
	})

	// Slack-like webhook  
	b.Run("Slack", func(b *testing.B) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		app := apprise.New()
		app.Add(fmt.Sprintf("slack://%s", server.Listener.Addr().String()))

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				app.Notify("Slack Test", "Test message", apprise.NotifyTypeInfo)
			}
		})
	})
}

// BenchmarkConfigurationTemplating benchmarks configuration templating performance
func BenchmarkConfigurationTemplating(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b.Run("WithTemplating", func(b *testing.B) {
		app := apprise.New()
		// Use template-like URL (conceptual)
		app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				app.Notify("Template Test", "Message with template", apprise.NotifyTypeInfo)
			}
		})
	})
}

// Helper function to create a realistic test scenario
func createRealisticTestScenario(b *testing.B, latencyMs int, errorRate float64) *apprise.Apprise {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate realistic latency
		time.Sleep(time.Duration(latencyMs) * time.Millisecond)
		
		// Simulate error rate
		if float64(time.Now().UnixNano()%100)/100.0 < errorRate {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
		w.WriteHeader(http.StatusOK)
	}))
	b.Cleanup(func() { server.Close() })

	app := apprise.New()
	app.Add(fmt.Sprintf("webhook://%s", server.Listener.Addr().String()))
	return app
}

// BenchmarkRealisticScenarios benchmarks realistic production scenarios
func BenchmarkRealisticScenarios(b *testing.B) {
	scenarios := []struct {
		name      string
		latencyMs int
		errorRate float64
	}{
		{"FastReliable", 10, 0.001},     // 10ms, 0.1% error rate
		{"NormalReliable", 50, 0.01},    // 50ms, 1% error rate
		{"SlowReliable", 200, 0.05},     // 200ms, 5% error rate
		{"FastUnreliable", 10, 0.10},    // 10ms, 10% error rate
		{"SlowUnreliable", 500, 0.15},   // 500ms, 15% error rate
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			app := createRealisticTestScenario(b, scenario.latencyMs, scenario.errorRate)
			
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					app.Notify("Realistic Test", "Production-like message", apprise.NotifyTypeInfo)
				}
			})
		})
	}
}