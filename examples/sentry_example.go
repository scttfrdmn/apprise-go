package main

import (
	"fmt"

	"github.com/scttfrdmn/apprise-go/apprise"
)

func main() {
	fmt.Println("=== Sentry Error Tracking Examples ===\n")

	// Example 1: Basic error tracking
	fmt.Println("1. Basic error tracking:")
	basicExample()

	// Example 2: Production error monitoring
	fmt.Println("\n2. Production error monitoring:")
	productionExample()

	// Example 3: Self-hosted Sentry
	fmt.Println("\n3. Self-hosted Sentry instance:")
	selfHostedExample()

	// Example 4: Multi-region deployment
	fmt.Println("\n4. Multi-region deployment:")
	multiRegionExample()

	//Example 5: Error severity levels
	fmt.Println("\n5. Error severity levels:")
	severityLevelsExample()
}

func basicExample() {
	// Replace with your actual Sentry DSN from https://sentry.io
	dsnExample := "sentry://abc123def456@o123456.ingest.sentry.io/789012"

	fmt.Printf("   DSN format: %s\n", dsnExample)
	fmt.Println("   ✓ Sentry DSN configured")
	fmt.Println("   Events will appear in Sentry dashboard under 'Issues'")
}

func productionExample() {
	app := apprise.New()

	// Configure Sentry for production error tracking
	// In production, use environment variable: os.Getenv("SENTRY_DSN")
	fmt.Println("   Configure Sentry:")
	fmt.Println("   app.Add(\"sentry://\" + os.Getenv(\"SENTRY_DSN\"))")

	fmt.Println("\n   Send error notification:")
	fmt.Println("   Title: Database Connection Failed")
	fmt.Println("   Body: Unable to connect to PostgreSQL")
	fmt.Println("   Level: error")
	fmt.Println("   Tags: database, production, critical")

	// Example error notification (would require valid DSN)
	fmt.Println("\n   ✓ Error event would be sent to Sentry")
	fmt.Println("   ✓ Grouped by error message")
	fmt.Println("   ✓ Filterable by tags")
	fmt.Println("   ✓ Alerts configured team members")

	_ = app // Suppress unused warning
}

func selfHostedExample() {
	// Self-hosted Sentry instance
	fmt.Println("   Self-hosted Sentry examples:")
	fmt.Println("")
	fmt.Println("   On-premises HTTP:")
	fmt.Println("   http://my_key@sentry.internal.com:8080/project-42")
	fmt.Println("")
	fmt.Println("   Self-hosted HTTPS:")
	fmt.Println("   https://key123@sentry.example.com/1")
	fmt.Println("")
	fmt.Println("   Docker local instance:")
	fmt.Println("   http://test_key@localhost:9000/1")
	fmt.Println("")
	fmt.Println("   ✓ Works with any Sentry-compatible instance")
	fmt.Println("   ✓ Same event format as sentry.io")
}

func multiRegionExample() {
	fmt.Println("   Multi-region Sentry ingestion:")
	fmt.Println("")
	fmt.Println("   US region:")
	fmt.Println("   sentry://key@o123456.ingest.us.sentry.io/789012")
	fmt.Println("")
	fmt.Println("   EU region:")
	fmt.Println("   sentry://key@o123456.ingest.eu.sentry.io/789012")
	fmt.Println("")
	fmt.Println("   Benefits:")
	fmt.Println("   ✓ Lower latency (choose region closest to app)")
	fmt.Println("   ✓ Data residency compliance")
	fmt.Println("   ✓ Automatic regional routing")
}

func severityLevelsExample() {
	fmt.Println("   Sentry severity level mapping:")
	fmt.Println("")
	fmt.Println("   NotifyTypeError   → Sentry level: 'error'")
	fmt.Println("   Use for: Exceptions, crashes, critical failures")
	fmt.Println("")
	fmt.Println("   NotifyTypeWarning → Sentry level: 'warning'")
	fmt.Println("   Use for: Degraded performance, deprecated APIs")
	fmt.Println("")
	fmt.Println("   NotifyTypeInfo    → Sentry level: 'info'")
	fmt.Println("   Use for: Significant events, state changes")
	fmt.Println("")
	fmt.Println("   NotifyTypeSuccess → Sentry level: 'info'")
	fmt.Println("   Use for: Successful operations, milestones")
}

// Demonstrate real-world use cases
func useCaseExamples() {
	fmt.Println("\n=== Real-World Use Cases ===")

	fmt.Println("\n1. API Error Tracking:")
	fmt.Println("   app.Notify(\"API Request Failed\",")
	fmt.Println("       \"POST /api/users returned 500\",")
	fmt.Println("       apprise.NotifyTypeError,")
	fmt.Println("       apprise.WithTags(\"api\", \"http-500\"))")

	fmt.Println("\n2. Database Issues:")
	fmt.Println("   app.Notify(\"Connection Pool Exhausted\",")
	fmt.Println("       \"All 100 connections in use\",")
	fmt.Println("       apprise.NotifyTypeWarning,")
	fmt.Println("       apprise.WithTags(\"database\", \"performance\"))")

	fmt.Println("\n3. Deployment Tracking:")
	fmt.Println("   app.Notify(\"Deployment Started\",")
	fmt.Println("       \"Version 2.1.0 deploying to production\",")
	fmt.Println("       apprise.NotifyTypeInfo,")
	fmt.Println("       apprise.WithTags(\"deployment\", \"v2.1.0\"))")

	fmt.Println("\n4. Performance Degradation:")
	fmt.Println("   app.Notify(\"Slow Query Detected\",")
	fmt.Println("       \"Query took 5.2s (threshold: 1s)\",")
	fmt.Println("       apprise.NotifyTypeWarning,")
	fmt.Println("       apprise.WithTags(\"performance\", \"database\"))")
}

// Best practices
func bestPractices() {
	fmt.Println("\n=== Best Practices ===")
	fmt.Println("")
	fmt.Println("1. Use Environment Variables:")
	fmt.Println("   export SENTRY_DSN=\"sentry://key@host/project\"")
	fmt.Println("   app.Add(os.Getenv(\"SENTRY_DSN\"))")
	fmt.Println("")
	fmt.Println("2. Tag Effectively:")
	fmt.Println("   • environment: production, staging, development")
	fmt.Println("   • component: api, database, cache, queue")
	fmt.Println("   • severity: critical, high, medium, low")
	fmt.Println("")
	fmt.Println("3. Error Grouping:")
	fmt.Println("   • Use consistent error messages")
	fmt.Println("   • Include error types in title")
	fmt.Println("   • Sentry groups similar errors automatically")
	fmt.Println("")
	fmt.Println("4. Avoid Noise:")
	fmt.Println("   • Don't send expected errors (404s, validation)")
	fmt.Println("   • Use appropriate severity levels")
	fmt.Println("   • Set up alerts for critical errors only")
	fmt.Println("")
	fmt.Println("5. Include Context:")
	fmt.Println("   • Add relevant tags (user_id, request_id)")
	fmt.Println("   • Include error details in body")
	fmt.Println("   • Use descriptive titles")
}

// Integration patterns
func integrationPatterns() {
	fmt.Println("\n=== Integration Patterns ===")
	fmt.Println("")
	fmt.Println("1. Panic Recovery:")
	fmt.Println("   defer func() {")
	fmt.Println("       if r := recover(); r != nil {")
	fmt.Println("           app.Notify(\"Panic Recovered\",")
	fmt.Println("               fmt.Sprintf(\"%v\", r),")
	fmt.Println("               apprise.NotifyTypeError)")
	fmt.Println("       }")
	fmt.Println("   }()")
	fmt.Println("")
	fmt.Println("2. HTTP Middleware:")
	fmt.Println("   if resp.StatusCode >= 500 {")
	fmt.Println("       app.Notify(\"Server Error\",")
	fmt.Println("           fmt.Sprintf(\"%d %s\", code, path),")
	fmt.Println("           apprise.NotifyTypeError)")
	fmt.Println("   }")
	fmt.Println("")
	fmt.Println("3. Database Errors:")
	fmt.Println("   if err := db.Query(...); err != nil {")
	fmt.Println("       app.Notify(\"Database Query Failed\",")
	fmt.Println("           err.Error(),")
	fmt.Println("           apprise.NotifyTypeError,")
	fmt.Println("           apprise.WithTags(\"database\"))")
	fmt.Println("   }")
}

// Sentry dashboard features
func dashboardFeatures() {
	fmt.Println("\n=== Sentry Dashboard Features ===")
	fmt.Println("")
	fmt.Println("After sending events, view in Sentry:")
	fmt.Println("")
	fmt.Println("1. Issues Tab:")
	fmt.Println("   • Grouped errors with frequency")
	fmt.Println("   • First/last seen timestamps")
	fmt.Println("   • Affected users count")
	fmt.Println("   • Event details and context")
	fmt.Println("")
	fmt.Println("2. Releases:")
	fmt.Println("   • Track errors by version")
	fmt.Println("   • Compare releases")
	fmt.Println("   • Monitor release health")
	fmt.Println("")
	fmt.Println("3. Alerts:")
	fmt.Println("   • Configure alert rules")
	fmt.Println("   • Notify via email, Slack, etc.")
	fmt.Println("   • Set thresholds and conditions")
	fmt.Println("")
	fmt.Println("4. Performance:")
	fmt.Println("   • Transaction traces")
	fmt.Println("   • Slow queries")
	fmt.Println("   • Performance trends")
}

// Go-specific advantages
func goAdvantages() {
	fmt.Println("\n=== Go Advantages for Sentry ===")
	fmt.Println("")
	fmt.Println("1. Native UUID Generation:")
	fmt.Println("   • crypto/rand for secure event IDs")
	fmt.Println("   • No external dependencies")
	fmt.Println("")
	fmt.Println("2. Type Safety:")
	fmt.Println("   • Compile-time DSN validation")
	fmt.Println("   • Type-safe event structures")
	fmt.Println("   • No runtime type errors")
	fmt.Println("")
	fmt.Println("3. Performance:")
	fmt.Println("   • Fast JSON serialization")
	fmt.Println("   • Efficient HTTP client pooling")
	fmt.Println("   • Low memory overhead")
	fmt.Println("")
	fmt.Println("4. Concurrency:")
	fmt.Println("   • Send events asynchronously")
	fmt.Println("   • Non-blocking error tracking")
	fmt.Println("   • Goroutine-safe operations")
	fmt.Println("")
	fmt.Println("5. Single Binary:")
	fmt.Println("   • No Python/Node.js runtime")
	fmt.Println("   • Easy deployment")
	fmt.Println("   • Consistent behavior")
}
