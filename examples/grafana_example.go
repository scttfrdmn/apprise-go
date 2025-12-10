package main

import (
	"fmt"
	"log"

	"github.com/scttfrdmn/apprise-go/apprise"
)

func main() {
	// Example 1: Basic Grafana webhook integration
	basicExample()

	// Example 2: With authentication
	authExample()

	// Example 3: With HMAC signature verification
	hmacExample()

	// Example 4: Multiple alert scenarios
	alertScenariosExample()
}

// basicExample demonstrates basic Grafana webhook usage
func basicExample() {
	fmt.Println("=== Basic Grafana Webhook Example ===")

	app := apprise.New()

	// Add Grafana webhook endpoint
	// Format: grafana://hostname/webhook/path
	err := app.Add("grafana://alerts.example.com/api/webhooks/apprise")
	if err != nil {
		log.Fatalf("Failed to add Grafana service: %v", err)
	}

	// Send a firing alert
	responses := app.Notify(
		"High CPU Usage",
		"CPU usage has exceeded 85% for 5 minutes",
		apprise.NotifyTypeWarning,
	)

	for _, resp := range responses {
		if resp.Success {
			fmt.Printf("✓ Alert sent successfully (took %v)\n", resp.Duration)
		} else {
			fmt.Printf("✗ Alert failed: %v\n", resp.Error)
		}
	}
}

// authExample demonstrates authentication methods
func authExample() {
	fmt.Println("\n=== Grafana with Authentication ===")

	app := apprise.New()

	// Example 1: HTTP Basic Authentication
	// Format: grafana://username:password@hostname/path
	err := app.Add("grafana://admin:secretpass@alerts.example.com/webhook")
	if err != nil {
		log.Printf("Basic auth example: %v", err)
	}

	// Example 2: Bearer Token
	// Format: grafana://token@hostname/path
	err = app.Add("grafana://eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9@alerts.example.com/webhook")
	if err != nil {
		log.Printf("Bearer token example: %v", err)
	}

	fmt.Println("Authentication examples configured")
}

// hmacExample demonstrates HMAC signature verification
func hmacExample() {
	fmt.Println("\n=== Grafana with HMAC Signature ===")

	app := apprise.New()

	// Add Grafana webhook with HMAC secret for signature verification
	// Format: grafana://hostname/path?hmac_secret=yoursecret
	err := app.Add("grafana://alerts.example.com/webhook?hmac_secret=my-shared-secret-key")
	if err != nil {
		log.Fatalf("Failed to add Grafana service with HMAC: %v", err)
	}

	// When sending, the service will automatically add:
	// - X-Grafana-Alerting-Signature header with HMAC-SHA256 signature
	// - X-Grafana-Timestamp header with current timestamp
	responses := app.Notify(
		"Database Connection Lost",
		"Unable to connect to primary database instance",
		apprise.NotifyTypeError,
	)

	for _, resp := range responses {
		if resp.Success {
			fmt.Println("✓ Signed alert sent successfully")
		} else {
			fmt.Printf("✗ Failed: %v\n", resp.Error)
		}
	}
}

// alertScenariosExample demonstrates different alert types
func alertScenariosExample() {
	fmt.Println("\n=== Different Alert Scenarios ===")

	app := apprise.New()

	// Configure with custom options
	err := app.Add("grafana://alerts.example.com/webhook?method=POST&max_alerts=100")
	if err != nil {
		log.Fatalf("Failed to configure: %v", err)
	}

	// Scenario 1: Info alert (firing)
	fmt.Println("\n1. Info Alert:")
	app.Notify(
		"Deployment Started",
		"Application v2.1.0 deployment in progress",
		apprise.NotifyTypeInfo,
		apprise.WithTags("deployment", "production"),
	)

	// Scenario 2: Warning alert (firing)
	fmt.Println("\n2. Warning Alert:")
	app.Notify(
		"High Memory Usage",
		"Memory usage at 78% - approaching threshold",
		apprise.NotifyTypeWarning,
		apprise.WithTags("monitoring", "memory"),
	)

	// Scenario 3: Critical alert (firing)
	fmt.Println("\n3. Critical Alert:")
	app.Notify(
		"Service Unavailable",
		"API service is not responding to health checks",
		apprise.NotifyTypeError,
		apprise.WithTags("critical", "api", "production"),
	)

	// Scenario 4: Resolved alert
	fmt.Println("\n4. Resolved Alert:")
	app.Notify(
		"Service Recovered",
		"API service has recovered and is healthy",
		apprise.NotifyTypeSuccess,
		apprise.WithTags("recovery", "api", "production"),
	)

	fmt.Println("\n✓ All scenarios executed")
}

// Advanced example: Custom headers and options
func advancedExample() {
	fmt.Println("\n=== Advanced Configuration ===")

	app := apprise.New()

	// Complex URL with multiple options:
	// - Custom HTTP method (PUT instead of POST)
	// - Max alerts limit
	// - Custom headers for routing/filtering
	// - HMAC signature
	complexURL := "grafana://alerts.example.com/webhook" +
		"?method=PUT" +
		"&max_alerts=50" +
		"&header_X-Environment=production" +
		"&header_X-Team=platform" +
		"&hmac_secret=production-webhook-secret"

	err := app.Add(complexURL)
	if err != nil {
		log.Fatalf("Failed to configure: %v", err)
	}

	// Send with timezone
	app.Notify(
		"Scheduled Maintenance",
		"Planned maintenance window starting in 1 hour",
		apprise.NotifyTypeWarning,
		apprise.WithTimezone("America/New_York"),
		apprise.WithTags("maintenance", "scheduled"),
	)

	fmt.Println("✓ Advanced configuration example complete")
}

// Integration with Grafana Alerting setup
func grafanaIntegrationGuide() {
	fmt.Println(`
=== Grafana Integration Guide ===

To integrate apprise-go with Grafana Alerting:

1. In Grafana, go to: Alerts & IRM → Alerting → Contact points

2. Click "+ Add contact point"

3. Configure:
   - Name: "Apprise-Go"
   - Integration: Select "Webhook"
   - URL: Your apprise-go webhook endpoint
   - HTTP Method: POST (default)

4. Optional Security:
   a) Basic Auth: Add username/password
   b) Authorization Header: Add Bearer token
   c) HMAC Signature:
      - In apprise-go URL: add ?hmac_secret=yoursecret
      - Grafana will validate using X-Grafana-Alerting-Signature header

5. Test the contact point

6. Create Alert Rules that use this contact point

Example Grafana Alert Rule:
- Alert name: "High CPU Usage"
- Condition: avg(cpu_usage) > 85
- For: 5m
- Contact point: "Apprise-Go"

The alert will be sent to your apprise-go service, which can then:
- Forward to multiple notification services
- Apply custom filtering/routing
- Add custom formatting
- Store alert history
- Implement custom escalation logic

For more information:
- Grafana Webhook Docs: https://grafana.com/docs/grafana/latest/alerting/
- Apprise-Go Docs: https://github.com/scttfrdmn/apprise-go
`)
}
