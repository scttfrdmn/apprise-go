package apprise

import (
	"fmt"
	"strings"
)

// GetSupportedServices returns a list of all supported service IDs
func GetSupportedServices() []string {
	return []string{
		"discord", "slack", "telegram", "email", "sendgrid", "mailgun", "webhook", "msteams",
		"pushover", "pushbullet", "twilio", "bulksms", "clicksend", "messagebird", "nexmo", "vonage", "plivo", "textmagic", "aws-sns-sms", "signal", "whatsapp",
		"desktop", "desktop-advanced", "desktop-interactive", "desktop-persistent", "gotify", "ntfy", "matrix", "reddit", "mastodon", "facebook", "instagram", "youtube", "tiktok",
		"mattermost", "pagerduty", "opsgenie",
		"aws-sns", "aws-ses", "gcp-pubsub", "azure-servicebus", "github", "gitlab",
		"jira", "datadog", "newrelic", "linkedin", "twitter", "apns", "fcm", "rich-mobile-push", "batch-mobile-push",
		"aws-iot", "gcp-iot", "polly", "twilio-voice", "rocketchat",
		"ifttt", "zapier", "homeassistant", "hass", "nodered",
	}
}

// CreateService creates a service instance by ID for testing and inspection
func CreateService(serviceID string) Service {
	switch strings.ToLower(serviceID) {
	case "discord":
		return &DiscordService{}
	case "slack":
		return &SlackService{}
	case "telegram":
		return &TelegramService{}
	case "email", "smtp":
		return &EmailService{}
	case "sendgrid":
		return &SendGridService{}
	case "mailgun":
		return &MailgunService{}
	case "webhook", "json":
		return &WebhookService{}
	case "msteams", "teams":
		return &MSTeamsService{}
	case "pushover":
		return &PushoverService{}
	case "pushbullet":
		return &PushbulletService{}
	case "twilio":
		return &TwilioService{}
	case "bulksms":
		return &BulkSMSService{}
	case "clicksend":
		return &ClickSendService{}
	case "messagebird":
		return &MessageBirdService{}
	case "nexmo", "vonage":
		return &NexmoService{}
	case "plivo":
		return &PlivoService{}
	case "textmagic":
		return &TextMagicService{}
	case "aws-sns-sms":
		return &AWSSNSSMSService{}
	case "signal":
		return &SignalService{}
	case "whatsapp":
		return &WhatsAppService{}
	case "desktop":
		return &DesktopService{}
	case "desktop-advanced":
		return NewAdvancedDesktopService()
	case "desktop-interactive":
		return NewInteractiveDesktopService()
	case "desktop-persistent":
		return NewPersistentDesktopService()
	case "matrix":
		return &MatrixService{}
	case "reddit":
		return &RedditService{}
	case "mastodon":
		return &MastodonService{}
	case "facebook":
		return &FacebookService{}
	case "instagram":
		return &InstagramService{}
	case "youtube":
		return &YouTubeService{}
	case "tiktok":
		return &TikTokService{}
	case "mattermost":
		return &MattermostService{}
	case "pagerduty":
		return &PagerDutyService{}
	case "opsgenie":
		return &OpsgenieService{}
	case "github":
		return &GitHubService{}
	case "gitlab":
		return &GitLabService{}
	case "jira":
		return &JiraService{}
	case "datadog":
		return &DatadogService{}
	case "newrelic":
		return &NewRelicService{}
	case "linkedin":
		return &LinkedInService{}
	case "twitter":
		return &TwitterService{}
	case "apns":
		return &APNSService{}
	case "fcm":
		return &FCMService{}
	case "rich-mobile-push":
		return NewRichMobilePushService()
	case "batch-mobile-push":
		return NewBatchMobilePushService()
	case "rocketchat":
		return &RocketChatService{}
	case "ifttt":
		return &IFTTTService{}
	case "zapier":
		return &ZapierService{}
	case "homeassistant", "hass":
		return &HomeAssistantService{}
	case "nodered":
		return &NodeREDService{}
	default:
		return nil
	}
}

// IsServiceSupported checks if a service ID is supported
func IsServiceSupported(serviceID string) bool {
	supportedServices := GetSupportedServices()
	serviceID = strings.ToLower(serviceID)
	
	for _, supported := range supportedServices {
		if strings.ToLower(supported) == serviceID {
			return true
		}
	}
	return false
}

// GetServiceFriendlyName returns a human-readable name for a service
func GetServiceFriendlyName(serviceID string) string {
	switch strings.ToLower(serviceID) {
	case "discord":
		return "Discord"
	case "slack":
		return "Slack"
	case "telegram":
		return "Telegram"
	case "email", "smtp":
		return "Email (SMTP)"
	case "sendgrid":
		return "SendGrid Email"
	case "mailgun":
		return "Mailgun Email"
	case "webhook", "json":
		return "Webhook"
	case "msteams", "teams":
		return "Microsoft Teams"
	case "pushover":
		return "Pushover"
	case "pushbullet":
		return "Pushbullet"
	case "twilio":
		return "Twilio SMS"
	case "bulksms":
		return "BulkSMS"
	case "clicksend":
		return "ClickSend SMS"
	case "messagebird":
		return "MessageBird SMS"
	case "nexmo":
		return "Nexmo SMS"
	case "vonage":
		return "Vonage SMS"
	case "plivo":
		return "Plivo SMS"
	case "textmagic":
		return "TextMagic SMS"
	case "aws-sns-sms":
		return "AWS SNS SMS"
	case "signal":
		return "Signal Messenger"
	case "whatsapp":
		return "WhatsApp Business API"
	case "desktop":
		return "Desktop Notifications"
	case "desktop-advanced":
		return "Advanced Desktop Notifications"
	case "desktop-interactive":
		return "Interactive Desktop Notifications"
	case "desktop-persistent":
		return "Persistent Desktop Notifications"
	case "matrix":
		return "Matrix"
	case "reddit":
		return "Reddit"
	case "mastodon":
		return "Mastodon"
	case "facebook":
		return "Facebook Pages"
	case "instagram":
		return "Instagram"
	case "youtube":
		return "YouTube"
	case "tiktok":
		return "TikTok"
	case "mattermost":
		return "Mattermost"
	case "pagerduty":
		return "PagerDuty"
	case "opsgenie":
		return "Opsgenie"
	case "aws-sns":
		return "Amazon SNS"
	case "aws-ses":
		return "Amazon SES"
	case "gcp-pubsub":
		return "Google Cloud Pub/Sub"
	case "azure-servicebus":
		return "Azure Service Bus"
	case "github":
		return "GitHub"
	case "gitlab":
		return "GitLab"
	case "jira":
		return "Jira"
	case "datadog":
		return "Datadog"
	case "newrelic":
		return "New Relic"
	case "linkedin":
		return "LinkedIn"
	case "twitter":
		return "Twitter"
	case "apns":
		return "Apple Push Notification Service"
	case "fcm":
		return "Firebase Cloud Messaging"
	case "rich-mobile-push":
		return "Rich Mobile Push Notifications"
	case "batch-mobile-push":
		return "Batch Mobile Push Notifications"
	case "aws-iot":
		return "AWS IoT Core"
	case "gcp-iot":
		return "Google Cloud IoT Core"
	case "polly":
		return "Amazon Polly"
	case "twilio-voice":
		return "Twilio Voice"
	case "rocketchat":
		return "Rocket.Chat"
	case "ifttt":
		return "IFTTT Webhooks"
	case "zapier":
		return "Zapier Webhooks"
	case "homeassistant", "hass":
		return "Home Assistant"
	case "nodered":
		return "Node-RED"
	default:
		return fmt.Sprintf("Unknown Service (%s)", serviceID)
	}
}


package apprise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// BarkScheme is the URL scheme for Bark notifications.
	BarkScheme = "bark"
	// BarksScheme is the secure URL scheme for Bark notifications.
	BarksScheme = "barks"

	// BarkDefaultPort is the default port for insecure Bark connections.
	BarkDefaultPort = 80
	// BarksDefaultPort is the default port for secure Bark connections.
	BarksDefaultPort = 443

	// BarkAPIPushPath is the API path for sending push notifications.
	BarkAPIPushPath = "/push"

	// ObscuredDeviceKey is a placeholder for obscured device keys in URLs.
	ObscuredDeviceKey = "******"
)

// BarkService implements the Service interface for Bark notifications.
type BarkService struct {
	Scheme    string
	Host      string
	Port      int
	DeviceKey string

	// Optional parameters from URL query or SendOptions, with URL query acting as defaults
	// that can be overridden by SendOptions.
	IconURL   string
	Sound     string
	Badge     int
	ActionURL string // 'url' in Bark API for action URL
	Category  string
	Title     string // 'title' from Apprise options or URL query

	client *http.Client
}

// NewBarkService creates a new BarkService instance from a URL.
// It parses the URL to extract Bark-specific parameters.
// URL formats supported:
// - bark://{device_key}@{hostname}/{path}
// - bark://{hostname}/{device_key}/{path}
// - barks://{device_key}@{hostname}:{port}/{path}
func NewBarkService(serviceURL *url.URL) (Service, error) {
	if serviceURL == nil {
		return nil, fmt.Errorf("BarkService: serviceURL cannot be nil")
	}

	service := &BarkService{
		Scheme: serviceURL.Scheme,
		Host:   serviceURL.Hostname(),
		client: &http.Client{
			Timeout: 10 * time.Second, // Default HTTP client timeout
		},
	}

	// Determine port
	portStr := serviceURL.Port()
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("BarkService: invalid port specified: %s", portStr)
		}
		service.Port = port
	} else {
		switch service.Scheme {
		case BarksScheme:
			service.Port = BarksDefaultPort
		case BarkScheme:
			service.Port = BarkDefaultPort
		default:
			return nil, fmt.Errorf("BarkService: unsupported scheme: %s", service.Scheme)
		}
	}

	// DeviceKey can be in UserInfo (device_key@host) or as the first path segment (host/device_key).
	// UserInfo takes precedence if both are present, as per common URL parsing conventions.
	if serviceURL.User != nil {
		service.DeviceKey = serviceURL.User.Username()
	} else {
		// If no UserInfo, check the first path segment
		pathSegments := strings.Split(strings.TrimPrefix(serviceURL.Path, "/"), "/")
		if len(pathSegments) > 0 && pathSegments[0] != "" {
			service.DeviceKey = pathSegments[0]
		}
	}

	// Parse query parameters, which act as defaults
	query := serviceURL.Query()
	service.IconURL = query.Get("icon")
	service.Sound = query.Get("sound")
	if badgeStr := query.Get("badge"); badgeStr != "" {
		badge, err := strconv.Atoi(badgeStr)
		if err != nil {
			return nil, fmt.Errorf("BarkService: invalid badge specified: %s", badgeStr)
		}
		service.Badge = badge
	}
	service.ActionURL = query.Get("url") // 'url' is a Bark specific parameter for action URL
	service.Category = query.Get("category")
	service.Title = query.Get("title") // 'title' from URL query

	return service, service.ValidateURL()
}

// ValidateURL ensures the BarkService has all required fields to send a notification.
func (s *BarkService) ValidateURL() error {
	if s.Host == "" {
		return fmt.Errorf("BarkService: host is required")
	}
	if s.DeviceKey == "" {
		return fmt.Errorf("BarkService: device key is required")
	}
	return nil
}

// Send sends a notification to the Bark server.
func (s *BarkService) Send(message string, options *SendOptions) error {
	if err := s.ValidateURL(); err != nil {
		return err
	}

	endpointScheme := "http"
	if s.Scheme == BarksScheme {
		endpointScheme = "https"
	}

	// Construct the full Bark API endpoint URL.
	// Example: https://my.bark.server:443/push
	endpointURL := fmt.Sprintf("%s://%s", endpointScheme, s.Host)
	// Only append port if it's non-default for the scheme
	if (s.Scheme == BarkScheme && s.Port != BarkDefaultPort) ||
		(s.Scheme == BarksScheme && s.Port != BarksDefaultPort) {
		endpointURL += fmt.Sprintf(":%d", s.Port)
	}
	endpointURL += BarkAPIPushPath

	// Prepare the JSON payload for the Bark API.
	payload := map[string]interface{}{
		"device_key": s.DeviceKey,
		"body":       message,
	}

	// Add 'title', prioritizing SendOptions over URL query parameter.
	if options != nil && options.Title != "" {
		payload["title"] = options.Title
	} else if s.Title != "" {
		payload["title"] = s.Title
	}

	// Add 'icon', prioritizing SendOptions over URL query parameter.
	if options != nil && options.IconURL != "" {
		payload["icon"] = options.IconURL
	} else if s.IconURL != "" {
		payload["icon"] = s.IconURL
	}

	// Add 'sound', prioritizing SendOptions over URL query parameter.
	if options != nil && options.Sound != "" {
		payload["sound"] = options.Sound
	} else if s.Sound != "" {
		payload["sound"] = s.Sound
	}

	// Add 'badge', prioritizing SendOptions over URL query parameter.
	if options != nil && options.Badge != 0 {
		payload["badge"] = options.Badge
	} else if s.Badge != 0 {
		payload["badge"] = s.Badge
	}

	// Add 'url' (action URL), prioritizing SendOptions over URL query parameter.
	if options != nil && options.ActionURL != "" {
		payload["url"] = options.ActionURL
	} else if s.ActionURL != "" {
		payload["url"] = s.ActionURL
	}

	// Add 'category', prioritizing SendOptions over URL query parameter.
	if options != nil && options.Category != "" {
		payload["category"] = options.Category
	} else if s.Category != "" {
		payload["category"] = s.Category
	}

	// Incorporate other Bark-specific parameters from SendOptions.CustomData,
	// ensuring not to overwrite fields already explicitly set.
	if options != nil && options.CustomData != nil {
		for k, v := range options.CustomData {
			// Only add if not already explicitly set (e.g., title, icon, etc.)
			if _, exists := payload[k]; !exists {
				payload[k] = v
			}
		}
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("BarkService: failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequest("POST", endpointURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("BarkService: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("BarkService: failed to send request to Bark server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var barkError struct {
			Code int    `json:"code"`
			Msg  string `json:"message"`
		}
		// Attempt to parse Bark's specific error response for more details.
		if err := json.NewDecoder(resp.Body).Decode(&barkError); err == nil && barkError.Msg != "" {
			return fmt.Errorf("BarkService: Bark server returned error %d: %s", resp.StatusCode, barkError.Msg)
		}
		// Fallback to generic error if Bark's error message cannot be parsed.
		return fmt.Errorf("BarkService: Bark server returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}

// GetURL reconstructs a sanitized Bark URL for logging or display purposes.
// The DeviceKey is obscured for security reasons.
func (s *BarkService) GetURL() string {
	u := &url.URL{
		Scheme: s.Scheme,
		Host:   s.Host,
	}

	// Add port if it's non-default for the scheme.
	if (s.Scheme == BarkScheme && s.Port != BarkDefaultPort) ||
		(s.Scheme == BarksScheme && s.Port != BarksDefaultPort) {
		u.Host = fmt.Sprintf("%s:%d", s.Host, s.Port)
	}

	// Obscure the device key for security when reconstructing the URL.
	u.User = url.User(ObscuredDeviceKey)

	// Append query parameters if they exist, to reflect the service configuration.
	query := url.Values{}
	if s.IconURL != "" {
		query.Set("icon", s.IconURL)
	}
	if s.Sound != "" {
		query.Set("sound", s.Sound)
	}
	if s.Badge != 0 {
		query.Set("badge", strconv.Itoa(s.Badge))
	}
	if s.ActionURL != "" {
		query.Set("url", s.ActionURL)
	}
	if s.Category != "" {
		query.Set("category", s.Category)
	}
	if s.Title != "" {
		query.Set("title", s.Title)
	}

	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	return u.String()
}