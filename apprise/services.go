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
		"desktop", "gotify", "ntfy", "matrix", "reddit", "mastodon", "facebook", "instagram", "youtube", "tiktok",
		"mattermost", "pagerduty", "opsgenie",
		"aws-sns", "aws-ses", "gcp-pubsub", "azure-servicebus", "github", "gitlab",
		"jira", "datadog", "newrelic", "linkedin", "twitter", "apns", "fcm",
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
