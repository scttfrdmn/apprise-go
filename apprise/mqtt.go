package apprise

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTService implements MQTT protocol notifications
// MQTT is the standard protocol for IoT device communication
// Perfect for home automation, industrial IoT, and edge computing
type MQTTService struct {
	broker      string // MQTT broker URL (tcp://host:port or ssl://host:port)
	topic       string // MQTT topic to publish to
	username    string // Optional username for authentication
	password    string // Optional password for authentication
	clientID    string // MQTT client identifier
	qos         byte   // Quality of Service level (0, 1, or 2)
	retained    bool   // Whether messages should be retained
	clean       bool   // Clean session flag
	willTopic   string // Last Will and Testament topic
	willPayload string // Last Will and Testament payload
	willQos     byte   // Last Will QoS
	willRetain  bool   // Last Will retain flag
	useTLS      bool   // Whether to use TLS/SSL
	insecure    bool   // Skip TLS certificate verification
	caFile      string // Path to CA certificate file
	certFile    string // Path to client certificate file
	keyFile     string // Path to client key file
}

// NewMQTTService creates a new MQTT service instance
func NewMQTTService() Service {
	return &MQTTService{
		clientID: fmt.Sprintf("apprise-go-%d", time.Now().Unix()),
		qos:      0, // Default: At most once
		retained: false,
		clean:    true,
	}
}

// GetServiceID returns the service identifier
func (m *MQTTService) GetServiceID() string {
	return "mqtt"
}

// GetDefaultPort returns the default MQTT port (1883 for TCP, 8883 for TLS)
func (m *MQTTService) GetDefaultPort() int {
	if m.useTLS {
		return 8883
	}
	return 1883
}

// ParseURL parses an MQTT service URL
// Format: mqtt://[username:password@]broker[:port]/topic[?qos=0&retained=true&clientid=id]
// Format: mqtts://[username:password@]broker[:port]/topic (TLS/SSL)
//
// Query Parameters:
//   - qos=0|1|2 - Quality of Service level (default: 0)
//   - retained=true|false - Retain message on broker (default: false)
//   - clientid=string - MQTT client identifier (default: auto-generated)
//   - clean=true|false - Clean session flag (default: true)
//   - will_topic=string - Last Will and Testament topic
//   - will_payload=string - Last Will and Testament message
//   - will_qos=0|1|2 - Last Will QoS (default: 0)
//   - will_retain=true|false - Last Will retain flag (default: false)
//   - insecure=true|false - Skip TLS certificate verification (default: false)
//   - ca_file=path - Path to CA certificate file
//   - cert_file=path - Path to client certificate file
//   - key_file=path - Path to client key file
//
// Examples:
//   mqtt://localhost/notifications
//   mqtt://user:pass@broker.example.com:1883/alerts?qos=1&retained=true
//   mqtts://broker.hivemq.com:8883/home/sensors?qos=2
//   mqtt://broker/topic?will_topic=offline&will_payload=disconnected
func (m *MQTTService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme
	if scheme != "mqtt" && scheme != "mqtts" {
		return fmt.Errorf("invalid scheme: expected 'mqtt' or 'mqtts', got '%s'", scheme)
	}

	// Determine if TLS should be used
	m.useTLS = (scheme == "mqtts")

	// Extract broker host and port
	host := serviceURL.Host
	if host == "" {
		return fmt.Errorf("missing broker host")
	}

	// Add default port if not specified
	if !strings.Contains(host, ":") {
		defaultPort := m.GetDefaultPort()
		host = fmt.Sprintf("%s:%d", host, defaultPort)
	}

	// Build broker URL
	protocol := "tcp"
	if m.useTLS {
		protocol = "ssl"
	}
	m.broker = fmt.Sprintf("%s://%s", protocol, host)

	// Extract topic from path
	m.topic = strings.Trim(serviceURL.Path, "/")
	if m.topic == "" {
		return fmt.Errorf("missing MQTT topic")
	}

	// Parse authentication
	if serviceURL.User != nil {
		m.username = serviceURL.User.Username()
		if password, hasPassword := serviceURL.User.Password(); hasPassword {
			m.password = password
		}
	}

	// Parse query parameters
	query := serviceURL.Query()

	if qosStr := query.Get("qos"); qosStr != "" {
		qos, err := strconv.Atoi(qosStr)
		if err != nil || qos < 0 || qos > 2 {
			return fmt.Errorf("invalid QoS value: %s (must be 0, 1, or 2)", qosStr)
		}
		m.qos = byte(qos)
	}

	if retainedStr := query.Get("retained"); retainedStr != "" {
		m.retained = (retainedStr == "true" || retainedStr == "1")
	}

	if clientID := query.Get("clientid"); clientID != "" {
		m.clientID = clientID
	}

	if cleanStr := query.Get("clean"); cleanStr != "" {
		m.clean = (cleanStr == "true" || cleanStr == "1")
	}

	// Last Will and Testament
	if willTopic := query.Get("will_topic"); willTopic != "" {
		m.willTopic = willTopic
		m.willPayload = query.Get("will_payload")

		if willQosStr := query.Get("will_qos"); willQosStr != "" {
			qos, err := strconv.Atoi(willQosStr)
			if err == nil && qos >= 0 && qos <= 2 {
				m.willQos = byte(qos)
			}
		}

		if willRetainStr := query.Get("will_retain"); willRetainStr != "" {
			m.willRetain = (willRetainStr == "true" || willRetainStr == "1")
		}
	}

	// TLS options
	if insecureStr := query.Get("insecure"); insecureStr != "" {
		m.insecure = (insecureStr == "true" || insecureStr == "1")
	}

	if caFile := query.Get("ca_file"); caFile != "" {
		m.caFile = caFile
	}

	if certFile := query.Get("cert_file"); certFile != "" {
		m.certFile = certFile
	}

	if keyFile := query.Get("key_file"); keyFile != "" {
		m.keyFile = keyFile
	}

	return nil
}

// Send publishes a notification message to the MQTT broker
func (m *MQTTService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message payload
	message := m.buildMessage(req)

	// Create MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(m.broker)
	opts.SetClientID(m.clientID)
	opts.SetCleanSession(m.clean)
	opts.SetUsername(m.username)
	opts.SetPassword(m.password)
	opts.SetAutoReconnect(false) // Single publish, no need to reconnect
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetWriteTimeout(10 * time.Second)

	// Set Last Will and Testament if configured
	if m.willTopic != "" {
		opts.SetWill(m.willTopic, m.willPayload, m.willQos, m.willRetain)
	}

	// Configure TLS if needed
	if m.useTLS {
		tlsConfig, err := m.buildTLSConfig()
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}
		opts.SetTLSConfig(tlsConfig)
	}

	// Create and connect client
	client := mqtt.NewClient(opts)
	token := client.Connect()

	// Wait for connection with timeout
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("connection timeout")
	}
	if token.Error() != nil {
		return fmt.Errorf("failed to connect to broker: %w", token.Error())
	}
	defer client.Disconnect(250)

	// Publish message
	pubToken := client.Publish(m.topic, m.qos, m.retained, message)

	// Wait for publish to complete
	if !pubToken.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("publish timeout")
	}
	if pubToken.Error() != nil {
		return fmt.Errorf("failed to publish message: %w", pubToken.Error())
	}

	return nil
}

// buildMessage constructs the MQTT message payload
func (m *MQTTService) buildMessage(req NotificationRequest) string {
	var builder strings.Builder

	// Add notification type indicator
	typePrefix := m.getTypePrefix(req.NotifyType)
	if typePrefix != "" {
		builder.WriteString(typePrefix)
		builder.WriteString(" ")
	}

	// Add title if present
	if req.Title != "" {
		builder.WriteString(req.Title)
		if req.Body != "" {
			builder.WriteString(": ")
		}
	}

	// Add body
	if req.Body != "" {
		builder.WriteString(req.Body)
	}

	// Add tags if present
	if len(req.Tags) > 0 {
		builder.WriteString(" [")
		builder.WriteString(strings.Join(req.Tags, ", "))
		builder.WriteString("]")
	}

	return builder.String()
}

// getTypePrefix returns a prefix string based on notification type
func (m *MQTTService) getTypePrefix(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeInfo:
		return "[INFO]"
	case NotifyTypeSuccess:
		return "[OK]"
	case NotifyTypeWarning:
		return "[WARN]"
	case NotifyTypeError:
		return "[ERROR]"
	default:
		return ""
	}
}

// buildTLSConfig creates TLS configuration for secure MQTT connections
func (m *MQTTService) buildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: m.insecure,
	}

	// Load CA certificate if provided
	if m.caFile != "" {
		caCert, err := os.ReadFile(m.caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate and key if provided
	if m.certFile != "" && m.keyFile != "" {
		cert, err := tls.LoadX509KeyPair(m.certFile, m.keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// TestURL validates the MQTT service URL format
func (m *MQTTService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return m.ParseURL(parsedURL)
}

// SupportsAttachments returns false as MQTT is designed for lightweight messages
func (m *MQTTService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns 0 (no strict limit, but MQTT is designed for small messages)
// Note: While MQTT itself can handle up to 256MB, most brokers limit to smaller sizes
func (m *MQTTService) GetMaxBodyLength() int {
	return 0 // No strict limit enforced by service
}
