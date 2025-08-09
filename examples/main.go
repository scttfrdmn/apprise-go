package main

import (
	"fmt"
	"log"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

func main() {
	// Example 1: Basic usage
	basicExample()

	// Example 2: Using configuration files
	configExample()

	// Example 3: Multiple services with tags
	taggedExample()

	// Example 4: With attachments
	attachmentExample()

	// Example 5: Error handling and response checking
	errorHandlingExample()
}

// basicExample demonstrates basic notification sending
func basicExample() {
	fmt.Println("=== Basic Example ===")

	// Create Apprise instance
	app := apprise.New()

	// Add notification services
	services := []string{
		"discord://webhook_id/webhook_token",
		"slack://TokenA/TokenB/TokenC/general",
		"tgram://bot_token/chat_id",
		"mailto://user:password@smtp.gmail.com/recipient@email.com",
		"pushover://token@userkey",
		"msteams://team_name/tokenA/tokenB/tokenC",
		"pball://access_token/device_id",
		"twilio://SID:TOKEN@+15551234567/+15559876543",
		"webhook://api.example.com/notify",
	}

	for _, serviceURL := range services {
		if err := app.Add(serviceURL); err != nil {
			log.Printf("Error adding service %s: %v", serviceURL, err)
		}
	}

	// Send notification
	responses := app.Notify(
		"Hello from Go Apprise!",
		"This is a test notification sent from the Go version of Apprise.",
		apprise.NotifyTypeInfo,
	)

	// Check responses
	for i, response := range responses {
		if response.Success {
			fmt.Printf("✓ Service %d (%s): Sent successfully in %v\n",
				i+1, response.ServiceID, response.Duration)
		} else {
			fmt.Printf("✗ Service %d (%s): Failed - %v\n",
				i+1, response.ServiceID, response.Error)
		}
	}

	fmt.Println()
}

// configExample demonstrates loading configuration from files
func configExample() {
	fmt.Println("=== Configuration Example ===")

	app := apprise.New()
	config := apprise.NewAppriseConfig(app)

	// Load from file (assuming config.yaml exists)
	err := config.AddFromFile("config.yaml")
	if err != nil {
		fmt.Printf("Could not load config.yaml: %v\n", err)
		// Continue with manual configuration
		app.Add("discord://webhook_id/webhook_token")
	} else {
		// Apply configuration to Apprise instance
		config.ApplyToApprise()
	}

	// Load default configurations
	config.LoadDefaultConfigs()
	config.ApplyToApprise()

	fmt.Printf("Loaded %d services from configuration\n", app.Count())

	// Send notification
	app.Notify(
		"Configuration Test",
		"This notification was sent using configuration files.",
		apprise.NotifyTypeSuccess,
	)

	fmt.Println()
}

// taggedExample demonstrates using tags for targeted notifications
func taggedExample() {
	fmt.Println("=== Tagged Example ===")

	app := apprise.New()

	// Add services with different tags
	app.Add("discord://webhook_id/webhook_token", "team", "alerts")
	app.Add("mailto://admin@company.com", "admin")
	app.Add("slack://TokenA/TokenB/TokenC/Channel", "team", "general")

	// Send to all services
	fmt.Println("Sending to all services:")
	responses := app.Notify(
		"System Update",
		"The system has been updated successfully.",
		apprise.NotifyTypeSuccess,
	)
	fmt.Printf("Sent to %d services\n", len(responses))

	// Send to specific tags
	fmt.Println("Sending to 'admin' tag only:")
	responses = app.Notify(
		"Admin Alert",
		"This message is for administrators only.",
		apprise.NotifyTypeWarning,
		apprise.WithTags("admin"),
	)
	fmt.Printf("Sent to %d services\n", len(responses))

	fmt.Println()
}

// attachmentExample demonstrates sending notifications with attachments
func attachmentExample() {
	fmt.Println("=== Attachment Example ===")

	app := apprise.New()
	app.Add("discord://webhook_id/webhook_token")

	// Create attachments
	attachments := []apprise.Attachment{
		{
			LocalPath: "/path/to/report.pdf",
			Name:      "monthly_report.pdf",
		},
		{
			URL:  "https://example.com/image.png",
			Name: "diagram.png",
		},
	}

	// Send notification with attachments
	responses := app.Notify(
		"Monthly Report",
		"Please find the monthly report and supporting diagram attached.",
		apprise.NotifyTypeInfo,
		apprise.WithAttachments(attachments...),
		apprise.WithBodyFormat("markdown"),
	)

	for _, response := range responses {
		if response.Success {
			fmt.Printf("✓ Sent with attachments to %s\n", response.ServiceID)
		} else {
			fmt.Printf("✗ Failed to send to %s: %v\n", response.ServiceID, response.Error)
		}
	}

	fmt.Println()
}

// errorHandlingExample demonstrates proper error handling
func errorHandlingExample() {
	fmt.Println("=== Error Handling Example ===")

	app := apprise.New()
	app.SetTimeout(5 * time.Second)

	// Add a service with invalid configuration (will fail)
	err := app.Add("discord://invalid_webhook")
	if err != nil {
		fmt.Printf("Expected error adding invalid service: %v\n", err)
	}

	// Add a valid service
	err = app.Add("discord://webhook_id/webhook_token")
	if err != nil {
		fmt.Printf("Error adding valid service: %v\n", err)
		return
	}

	// Send notification and check each response
	responses := app.Notify(
		"Error Handling Test",
		"Testing error handling in Go Apprise",
		apprise.NotifyTypeError,
	)

	successCount := 0
	for _, response := range responses {
		if response.Success {
			successCount++
			fmt.Printf("✓ %s: Success (took %v)\n",
				response.ServiceID, response.Duration)
		} else {
			fmt.Printf("✗ %s: Failed - %v\n",
				response.ServiceID, response.Error)
		}
	}

	fmt.Printf("Successfully sent to %d/%d services\n", successCount, len(responses))

	// Handle partial failures
	if successCount > 0 && successCount < len(responses) {
		fmt.Println("Warning: Some notifications failed to send")
	} else if successCount == 0 {
		fmt.Println("Error: All notifications failed")
	} else {
		fmt.Println("All notifications sent successfully")
	}

	fmt.Println()
}

// Additional utility functions and examples

// customServiceExample shows how to create a custom notification service
func customServiceExample() {
	fmt.Println("=== Custom Service Example ===")

	// This would require implementing the Service interface
	// See the Discord service implementation as a reference

	app := apprise.New()

	// Register custom service
	// registry := app.GetRegistry() // You'd need to expose this
	// registry.Register("custom", func() apprise.Service {
	//     return NewCustomService()
	// })

	// Use custom service
	err := app.Add("custom://custom_config_here")
	if err != nil {
		fmt.Printf("Error adding custom service: %v\n", err)
		return
	}

	app.Notify(
		"Custom Service Test",
		"Testing custom notification service",
		apprise.NotifyTypeInfo,
	)

	fmt.Println()
}

// bulkNotificationExample demonstrates sending bulk notifications efficiently
func bulkNotificationExample() {
	fmt.Println("=== Bulk Notification Example ===")

	app := apprise.New()
	app.Add("discord://webhook_id/webhook_token")

	// Send multiple notifications
	notifications := []struct {
		title string
		body  string
		nType apprise.NotifyType
	}{
		{"Server 1", "Server 1 is healthy", apprise.NotifyTypeSuccess},
		{"Server 2", "Server 2 has high CPU usage", apprise.NotifyTypeWarning},
		{"Server 3", "Server 3 is down", apprise.NotifyTypeError},
	}

	for _, notif := range notifications {
		responses := app.Notify(notif.title, notif.body, notif.nType)
		for _, response := range responses {
			if response.Success {
				fmt.Printf("✓ Sent: %s\n", notif.title)
			} else {
				fmt.Printf("✗ Failed: %s - %v\n", notif.title, response.Error)
			}
		}
	}

	fmt.Println()
}

/* Example configuration files:

config.yaml:
---
version: 1
urls:
  - url: discord://webhook_id/webhook_token
    tag:
      - team
      - alerts
  - url: mailto://user:pass@gmail.com/admin@company.com
    tag:
      - admin
  - url: slack://TokenA/TokenB/TokenC/general
    tag:
      - team
      - general

config.txt:
# Team notifications
discord://webhook_id/webhook_token [team,alerts]

# Admin email
mailto://user:pass@gmail.com/admin@company.com [admin]

# General Slack channel
slack://TokenA/TokenB/TokenC/general [team,general]

*/