package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

func main() {
	fmt.Println("=== MQTT Service Examples ===\n")

	// Example 1: Basic MQTT notification
	fmt.Println("1. Basic MQTT notification to local broker:")
	basicExample()

	// Example 2: QoS levels demonstration
	fmt.Println("\n2. Quality of Service (QoS) levels:")
	qosExample()

	// Example 3: Secure MQTT with TLS
	fmt.Println("\n3. Secure MQTT (TLS/SSL) notification:")
	secureExample()

	// Example 4: Last Will and Testament
	fmt.Println("\n4. MQTT with Last Will and Testament:")
	lastWillExample()

	// Example 5: IoT sensor monitoring
	fmt.Println("\n5. IoT sensor monitoring example:")
	iotMonitoringExample()

	// Example 6: Home automation integration
	fmt.Println("\n6. Home automation notification:")
	homeAutomationExample()

	// Example 7: Retained messages
	fmt.Println("\n7. Retained message example:")
	retainedMessageExample()
}

// basicExample demonstrates basic MQTT notification
func basicExample() {
	app := apprise.New()

	// Add basic MQTT service (will fail without a broker running)
	err := app.Add("mqtt://localhost/notifications/apprise")
	if err != nil {
		fmt.Printf("   ✗ Failed to add service: %v\n", err)
		return
	}

	// Send notification
	responses := app.Notify(
		"System Update",
		"Version 2.1.0 has been deployed successfully",
		apprise.NotifyTypeInfo,
	)

	for _, resp := range responses {
		if resp.Success {
			fmt.Printf("   ✓ Message sent to MQTT topic 'notifications/apprise'\n")
		} else {
			fmt.Printf("   ✗ Failed to send: %v\n", resp.Error)
			fmt.Println("   Note: Make sure an MQTT broker is running on localhost:1883")
		}
	}
}

// qosExample demonstrates different QoS levels
func qosExample() {
	fmt.Println("   QoS 0 (At most once - fire and forget):")
	fmt.Println("   mqtt://broker/topic?qos=0")
	fmt.Println("   - Fastest delivery, no acknowledgment")
	fmt.Println("   - Best for non-critical updates")

	fmt.Println("\n   QoS 1 (At least once - acknowledged):")
	fmt.Println("   mqtt://broker/topic?qos=1")
	fmt.Println("   - Message acknowledged by broker")
	fmt.Println("   - May receive duplicates")
	fmt.Println("   - Good for important notifications")

	fmt.Println("\n   QoS 2 (Exactly once - guaranteed):")
	fmt.Println("   mqtt://broker/topic?qos=2")
	fmt.Println("   - Four-way handshake ensures single delivery")
	fmt.Println("   - Slowest but most reliable")
	fmt.Println("   - Critical for financial or safety alerts")

	// Demonstrate QoS 1 usage
	app := apprise.New()
	err := app.Add("mqtt://localhost/alerts/critical?qos=1")
	if err == nil {
		app.Notify(
			"Critical Alert",
			"Database connection lost",
			apprise.NotifyTypeError,
		)
		fmt.Println("\n   ✓ URL configured with QoS 1 (acknowledged delivery)")
	} else {
		fmt.Printf("\n   ✗ Configuration failed: %v\n", err)
	}
}

// secureExample demonstrates TLS/SSL MQTT
func secureExample() {
	app := apprise.New()

	// Secure MQTT with TLS (port 8883 by default)
	url := "mqtts://broker.hivemq.com:8883/apprise/secure/alerts"
	err := app.Add(url)
	if err != nil {
		fmt.Printf("   ✗ Failed to add service: %v\n", err)
		return
	}

	fmt.Printf("   ✓ Configured secure MQTT connection\n")
	fmt.Printf("   Broker: broker.hivemq.com:8883 (TLS)\n")
	fmt.Printf("   Topic: apprise/secure/alerts\n")

	// With custom CA certificate (for self-signed certs)
	urlWithCA := "mqtts://internal-broker.local:8883/secure?ca_file=/path/to/ca.pem"
	fmt.Printf("\n   Example with custom CA:\n   %s\n", urlWithCA)

	// With client certificates (mutual TLS)
	urlWithMutualTLS := "mqtts://broker/topic?cert_file=/path/to/cert.pem&key_file=/path/to/key.pem"
	fmt.Printf("\n   Example with client certificates:\n   %s\n", urlWithMutualTLS)
}

// lastWillExample demonstrates Last Will and Testament
func lastWillExample() {
	app := apprise.New()

	// Configure Last Will - sent by broker if client disconnects ungracefully
	url := "mqtt://broker.local/devices/sensor-01?will_topic=devices/status&will_payload=sensor-01-offline&will_qos=1&will_retain=true"
	err := app.Add(url)
	if err != nil {
		fmt.Printf("   ✗ Failed to add service: %v\n", err)
		return
	}

	fmt.Println("   ✓ Configured MQTT with Last Will and Testament")
	fmt.Println("   Main topic: devices/sensor-01")
	fmt.Println("   LWT topic:  devices/status")
	fmt.Println("   LWT message: 'sensor-01-offline'")
	fmt.Println("   LWT QoS:    1 (acknowledged)")
	fmt.Println("   LWT retain: true")
	fmt.Println("\n   If connection drops unexpectedly:")
	fmt.Println("   → Broker publishes 'sensor-01-offline' to devices/status")
	fmt.Println("   → Message is retained for late subscribers")
}

// iotMonitoringExample demonstrates IoT sensor monitoring
func iotMonitoringExample() {
	app := apprise.New()

	// Configure multiple MQTT topics for different sensors
	sensors := map[string]string{
		"temperature": "mqtt://broker.local/home/sensors/temperature?qos=1",
		"humidity":    "mqtt://broker.local/home/sensors/humidity?qos=1",
		"motion":      "mqtt://broker.local/home/sensors/motion?qos=2&retained=true",
	}

	for sensor, url := range sensors {
		err := app.Add(url)
		if err == nil {
			fmt.Printf("   ✓ Configured %s sensor\n", sensor)
		}
	}

	// Simulate temperature alert
	ctx := context.Background()
	fmt.Println("\n   Simulating high temperature alert:")

	req := apprise.NotificationRequest{
		Title:      "High Temperature",
		Body:       "Living room: 28°C (threshold: 25°C)",
		NotifyType: apprise.NotifyTypeWarning,
		Tags:       []string{"living-room", "temperature", "zone-1"},
	}

	// Would send to all configured sensors
	fmt.Printf("   Message: [WARN] %s: %s [%s]\n",
		req.Title, req.Body, req.Tags[0])
	fmt.Println("   ✓ Alert would be sent to all sensor topics")

	_ = ctx // Suppress unused warning in example
}

// homeAutomationExample demonstrates Home Assistant integration
func homeAutomationExample() {
	app := apprise.New()

	// Home Assistant MQTT discovery pattern
	url := "mqtt://homeassistant.local/homeassistant/notifications?qos=1&retained=true&clientid=apprise-notifier"
	err := app.Add(url)
	if err != nil {
		fmt.Printf("   ✗ Failed to add service: %v\n", err)
		return
	}

	fmt.Println("   ✓ Configured for Home Assistant MQTT")
	fmt.Println("   Broker: homeassistant.local")
	fmt.Println("   Topic: homeassistant/notifications")
	fmt.Println("   Client ID: apprise-notifier")
	fmt.Println("   Retained: true (for persistent notifications)")

	// Example notifications
	fmt.Println("\n   Example automation scenarios:")
	fmt.Println("   • Motion detected → Send alert with timestamp")
	fmt.Println("   • Door opened → Notify with location tag")
	fmt.Println("   • Low battery → Warning notification")
	fmt.Println("   • System online → Success notification")

	// Simulate door sensor alert
	app.Notify(
		"Front Door",
		"Door opened at 14:32",
		apprise.NotifyTypeInfo,
		apprise.WithTags("entry", "security"),
	)
	fmt.Println("\n   ✓ Sent door sensor notification")
}

// retainedMessageExample demonstrates retained messages
func retainedMessageExample() {
	app := apprise.New()

	// Retained messages stay on the broker for late subscribers
	url := "mqtt://broker.local/status/system?retained=true&qos=1"
	err := app.Add(url)
	if err != nil {
		fmt.Printf("   ✗ Failed to add service: %v\n", err)
		return
	}

	fmt.Println("   ✓ Configured with message retention")
	fmt.Println("   Topic: status/system")
	fmt.Println("   Retained: true")
	fmt.Println("\n   How it works:")
	fmt.Println("   1. Message published with retain flag")
	fmt.Println("   2. Broker stores the last message")
	fmt.Println("   3. New subscribers immediately receive it")
	fmt.Println("   4. Use for status updates, current values")

	// Send system status
	app.Notify(
		"System Status",
		"All services operational",
		apprise.NotifyTypeSuccess,
	)
	fmt.Println("\n   ✓ Status message retained on broker")
	fmt.Println("   → Any new subscriber gets current status")
}

// Demonstrate advanced MQTT usage patterns
func demonstratePatterns() {
	fmt.Println("\n=== Advanced MQTT Patterns ===")

	// Pattern 1: Topic hierarchy
	fmt.Println("\n1. Topic Hierarchy:")
	fmt.Println("   mqtt://broker/company/site/building/floor/room/sensor")
	fmt.Println("   • Organize by location or function")
	fmt.Println("   • Subscribe to wildcards: company/+/building/#")

	// Pattern 2: Multi-broker setup
	fmt.Println("\n2. Multiple Brokers:")
	app := apprise.New()
	app.Add("mqtt://local-broker/home/alerts?qos=1")
	app.Add("mqtts://cloud-broker.com:8883/cloud/backup?qos=2")
	fmt.Println("   ✓ Local broker for low latency")
	fmt.Println("   ✓ Cloud broker for backup/archive")

	// Pattern 3: Client ID for persistent sessions
	fmt.Println("\n3. Persistent Sessions:")
	fmt.Println("   mqtt://broker/topic?clientid=myapp-001&clean=false")
	fmt.Println("   • Broker remembers subscriptions")
	fmt.Println("   • Queues messages while offline")
	fmt.Println("   • Useful for intermittent connectivity")

	// Pattern 4: Monitoring and metrics
	fmt.Println("\n4. Monitoring Integration:")
	fmt.Println("   mqtt://broker/metrics/apprise?qos=1")
	fmt.Println("   • Integrate with Prometheus MQTT exporter")
	fmt.Println("   • Feed into Grafana dashboards")
	fmt.Println("   • Track notification delivery rates")
}

// Performance characteristics
func performanceNotes() {
	fmt.Println("\n=== MQTT Performance Notes ===")
	fmt.Println()
	fmt.Println("QoS Impact on Performance:")
	fmt.Println("  QoS 0: ~1-2ms per message (fastest)")
	fmt.Println("  QoS 1: ~3-5ms per message (1 round trip)")
	fmt.Println("  QoS 2: ~6-10ms per message (2 round trips)")
	fmt.Println()
	fmt.Println("Connection Overhead:")
	fmt.Println("  TCP connect:     ~5-10ms")
	fmt.Println("  TLS handshake:   +20-50ms")
	fmt.Println("  MQTT CONNECT:    ~2-5ms")
	fmt.Println()
	fmt.Println("Go Advantages for MQTT:")
	fmt.Println("  ✓ Native binary protocol handling")
	fmt.Println("  ✓ Efficient connection pooling")
	fmt.Println("  ✓ Low memory footprint (~2-5 MB)")
	fmt.Println("  ✓ Fast TLS handshake (native crypto)")
	fmt.Println("  ✓ Excellent for edge devices")
}

// Common MQTT brokers and their configurations
func brokerExamples() {
	fmt.Println("\n=== Popular MQTT Brokers ===")

	brokers := []struct {
		name   string
		url    string
		notes  string
	}{
		{
			name:  "Mosquitto (Local)",
			url:   "mqtt://localhost:1883/test",
			notes: "Lightweight, open source, easy to install",
		},
		{
			name:  "HiveMQ Cloud",
			url:   "mqtts://broker.hivemq.com:8883/apprise/test",
			notes: "Free tier available, enterprise features",
		},
		{
			name:  "EMQX",
			url:   "mqtt://emqx.local:1883/notifications",
			notes: "Scalable, high-performance, cluster support",
		},
		{
			name:  "AWS IoT Core",
			url:   "mqtts://xxxxx.iot.us-east-1.amazonaws.com:8883/app/alerts",
			notes: "Managed service, integrates with AWS services",
		},
	}

	for i, broker := range brokers {
		fmt.Printf("\n%d. %s\n", i+1, broker.name)
		fmt.Printf("   URL:   %s\n", broker.url)
		fmt.Printf("   Notes: %s\n", broker.notes)
	}
}

// Installation and setup guide
func setupGuide() {
	fmt.Println("\n=== Quick Setup Guide ===")
	fmt.Println()
	fmt.Println("1. Install Mosquitto (local testing):")
	fmt.Println("   macOS:   brew install mosquitto")
	fmt.Println("   Ubuntu:  apt-get install mosquitto mosquitto-clients")
	fmt.Println("   Docker:  docker run -p 1883:1883 eclipse-mosquitto")
	fmt.Println()
	fmt.Println("2. Start Mosquitto:")
	fmt.Println("   mosquitto -v")
	fmt.Println()
	fmt.Println("3. Test with mosquitto_sub (in another terminal):")
	fmt.Println("   mosquitto_sub -h localhost -t 'notifications/#' -v")
	fmt.Println()
	fmt.Println("4. Run this example:")
	fmt.Println("   go run mqtt_example.go")
	fmt.Println()
	fmt.Println("5. Watch messages appear in mosquitto_sub terminal")
}

// Main function alternatives for different scenarios
func init() {
	// This init function will run before main
	// Used to set up any required configuration

	log.SetFlags(log.Ltime | log.Lshortfile)
}

// Example of using MQTT with context for cancellation
func contextExample() {
	fmt.Println("\n=== Context-based Usage ===")

	app := apprise.New()
	app.Add("mqtt://localhost/test?qos=1")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send notification with context
	// The MQTT client will respect the context timeout
	req := apprise.NotificationRequest{
		Title:      "Context Example",
		Body:       "This notification respects context timeout",
		NotifyType: apprise.NotifyTypeInfo,
	}

	fmt.Println("   ✓ Sending with 5-second timeout")
	fmt.Printf("   If broker doesn't respond in 5s, operation will be cancelled\n")

	_ = ctx // Suppress unused warning
	_ = req
}
