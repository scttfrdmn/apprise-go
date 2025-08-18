package apprise

import (
	"fmt"
	"reflect"
	"strings"
)

// ServiceCategory represents a category of notification services
type ServiceCategory struct {
	Name        string
	Description string
	Services    []string
}

// ServiceDocumentation represents comprehensive documentation for a service
type ServiceDocumentation struct {
	ID           string
	Name         string
	Description  string
	Category     string
	URLFormat    string
	Parameters   []ServiceParameter
	Examples     []ServiceExample
	Setup        []string
	Limitations  []string
	Since        string
}

// ServiceParameter describes a service configuration parameter
type ServiceParameter struct {
	Name        string
	Type        string
	Required    bool
	Description string
	Default     string
	Example     string
}

// ServiceExample provides usage examples for a service
type ServiceExample struct {
	Description string
	URL         string
	Code        string
}

// DocumentationGenerator generates comprehensive service documentation
type DocumentationGenerator struct {
	categories map[string]ServiceCategory
	services   map[string]ServiceDocumentation
}

// NewDocumentationGenerator creates a new documentation generator
func NewDocumentationGenerator() *DocumentationGenerator {
	dg := &DocumentationGenerator{
		categories: make(map[string]ServiceCategory),
		services:   make(map[string]ServiceDocumentation),
	}
	
	dg.initializeCategories()
	dg.initializeServiceDocumentation()
	
	return dg
}

// initializeCategories sets up service categories
func (dg *DocumentationGenerator) initializeCategories() {
	dg.categories = map[string]ServiceCategory{
		"messaging": {
			Name:        "Messaging & Chat",
			Description: "Real-time messaging platforms and chat services",
			Services:    []string{"discord", "slack", "telegram", "msteams", "matrix", "mattermost", "rocketchat"},
		},
		"social": {
			Name:        "Social Media",
			Description: "Social media platforms for public notifications",
			Services:    []string{"reddit", "mastodon", "facebook", "instagram", "youtube", "tiktok", "linkedin", "twitter"},
		},
		"email": {
			Name:        "Email Services",
			Description: "Email notification services and providers",
			Services:    []string{"email", "sendgrid", "mailgun", "aws-ses"},
		},
		"sms": {
			Name:        "SMS & Text Messaging",
			Description: "SMS and text messaging services",
			Services:    []string{"twilio", "bulksms", "clicksend", "messagebird", "nexmo", "vonage", "plivo", "textmagic", "aws-sns-sms"},
		},
		"mobile": {
			Name:        "Mobile Push Notifications",
			Description: "Mobile push notification services for iOS and Android",
			Services:    []string{"apns", "fcm", "rich-mobile-push", "batch-mobile-push"},
		},
		"instant": {
			Name:        "Instant Messaging",
			Description: "Instant messaging and secure communication platforms",
			Services:    []string{"signal", "whatsapp"},
		},
		"desktop": {
			Name:        "Desktop Notifications",
			Description: "Desktop notification systems for local alerts",
			Services:    []string{"desktop", "desktop-advanced", "desktop-interactive", "desktop-persistent"},
		},
		"push": {
			Name:        "Push Notification Services",
			Description: "Push notification platforms and services",
			Services:    []string{"pushover", "pushbullet", "gotify", "ntfy"},
		},
		"cloud": {
			Name:        "Cloud Services",
			Description: "Cloud platform notification and messaging services",
			Services:    []string{"aws-sns", "gcp-pubsub", "azure-servicebus", "aws-iot", "gcp-iot"},
		},
		"devops": {
			Name:        "DevOps & Monitoring",
			Description: "Development operations and system monitoring platforms",
			Services:    []string{"github", "gitlab", "jira", "datadog", "newrelic", "pagerduty", "opsgenie"},
		},
		"iot": {
			Name:        "IoT & Automation",
			Description: "Internet of Things and home automation platforms",
			Services:    []string{"ifttt", "zapier", "homeassistant", "hass", "nodered"},
		},
		"voice": {
			Name:        "Voice & Audio",
			Description: "Voice calling and audio notification services",
			Services:    []string{"polly", "twilio-voice"},
		},
		"webhook": {
			Name:        "Webhooks & APIs",
			Description: "Generic webhook and API notification endpoints",
			Services:    []string{"webhook", "json"},
		},
	}
}

// initializeServiceDocumentation sets up detailed service documentation
func (dg *DocumentationGenerator) initializeServiceDocumentation() {
	// Discord
	dg.services["discord"] = ServiceDocumentation{
		ID:          "discord",
		Name:        "Discord",
		Description: "Send notifications to Discord channels via webhooks",
		Category:    "messaging",
		URLFormat:   "discord://webhook_id/webhook_token",
		Parameters: []ServiceParameter{
			{Name: "webhook_id", Type: "string", Required: true, Description: "Discord webhook ID", Example: "123456789012345678"},
			{Name: "webhook_token", Type: "string", Required: true, Description: "Discord webhook token", Example: "abcdef123456789"},
			{Name: "avatar", Type: "string", Required: false, Description: "Custom avatar URL or name", Example: "MyBot"},
			{Name: "username", Type: "string", Required: false, Description: "Custom username for the bot", Example: "NotificationBot"},
			{Name: "tts", Type: "bool", Required: false, Description: "Enable text-to-speech", Example: "true"},
			{Name: "format", Type: "string", Required: false, Description: "Message format (text/markdown)", Default: "text", Example: "markdown"},
		},
		Examples: []ServiceExample{
			{
				Description: "Basic Discord notification",
				URL:         "discord://123456789012345678/abcdef123456789",
				Code:        `app.Add("discord://123456789012345678/abcdef123456789")`,
			},
			{
				Description: "Discord with custom avatar and username",
				URL:         "discord://123456789012345678/abcdef123456789?avatar=MyBot&username=NotificationBot",
				Code:        `app.Add("discord://123456789012345678/abcdef123456789?avatar=MyBot&username=NotificationBot")`,
			},
			{
				Description: "Discord with markdown formatting",
				URL:         "discord://123456789012345678/abcdef123456789?format=markdown",
				Code:        `app.Add("discord://123456789012345678/abcdef123456789?format=markdown")`,
			},
		},
		Setup: []string{
			"1. Go to your Discord server settings",
			"2. Navigate to 'Integrations' → 'Webhooks'",
			"3. Click 'New Webhook' or 'Create Webhook'",
			"4. Configure the webhook name and channel",
			"5. Copy the webhook URL",
			"6. Extract webhook_id and webhook_token from the URL",
		},
		Limitations: []string{
			"Requires webhook permissions in Discord server",
			"Rate limited by Discord's webhook limits",
			"Message length limited to 2000 characters",
		},
		Since: "1.0.0",
	}

	// Slack
	dg.services["slack"] = ServiceDocumentation{
		ID:          "slack",
		Name:        "Slack",
		Description: "Send notifications to Slack channels and users",
		Category:    "messaging",
		URLFormat:   "slack://TokenA/TokenB/TokenC/Channel",
		Parameters: []ServiceParameter{
			{Name: "token", Type: "string", Required: true, Description: "Slack bot token or webhook URL", Example: "xoxb-123-456-789"},
			{Name: "channel", Type: "string", Required: true, Description: "Channel name or ID", Example: "#general"},
			{Name: "username", Type: "string", Required: false, Description: "Bot username", Example: "NotificationBot"},
			{Name: "icon", Type: "string", Required: false, Description: "Bot icon emoji or URL", Example: ":robot_face:"},
			{Name: "format", Type: "string", Required: false, Description: "Message format (text/markdown)", Default: "text"},
		},
		Examples: []ServiceExample{
			{
				Description: "Slack webhook notification",
				URL:         "slack://T123456/B123456/xyz789abc/#general",
				Code:        `app.Add("slack://T123456/B123456/xyz789abc/#general")`,
			},
			{
				Description: "Slack with custom username and icon",
				URL:         "slack://T123456/B123456/xyz789abc/#alerts?username=AlertBot&icon=:warning:",
				Code:        `app.Add("slack://T123456/B123456/xyz789abc/#alerts?username=AlertBot&icon=:warning:")`,
			},
		},
		Setup: []string{
			"1. Create a Slack app at https://api.slack.com/apps",
			"2. Add bot token scopes: chat:write, chat:write.public",
			"3. Install the app to your workspace",
			"4. Copy the bot token (starts with xoxb-)",
			"5. Invite the bot to your channels",
		},
		Since: "1.0.0",
	}

	// Continue with other major services...
	dg.initializeEmailServices()
	dg.initializeSMSServices()
	dg.initializeMobileServices()
	dg.initializeDesktopServices()
}

// initializeEmailServices adds email service documentation
func (dg *DocumentationGenerator) initializeEmailServices() {
	dg.services["email"] = ServiceDocumentation{
		ID:          "email",
		Name:        "Email (SMTP)",
		Description: "Send email notifications via SMTP servers",
		Category:    "email",
		URLFormat:   "mailto://user:pass@server:port/to_email",
		Parameters: []ServiceParameter{
			{Name: "user", Type: "string", Required: true, Description: "SMTP username", Example: "myemail@gmail.com"},
			{Name: "pass", Type: "string", Required: true, Description: "SMTP password or app password", Example: "mypassword"},
			{Name: "server", Type: "string", Required: true, Description: "SMTP server hostname", Example: "smtp.gmail.com"},
			{Name: "port", Type: "int", Required: false, Description: "SMTP port", Default: "587", Example: "587"},
			{Name: "to", Type: "string", Required: true, Description: "Recipient email address", Example: "recipient@example.com"},
			{Name: "from", Type: "string", Required: false, Description: "Sender name", Example: "Notification System"},
			{Name: "name", Type: "string", Required: false, Description: "Sender display name", Example: "MyApp Notifications"},
		},
		Examples: []ServiceExample{
			{
				Description: "Gmail SMTP email notification",
				URL:         "mailto://myemail@gmail.com:mypassword@smtp.gmail.com:587/recipient@example.com",
				Code:        `app.Add("mailto://myemail@gmail.com:mypassword@smtp.gmail.com:587/recipient@example.com")`,
			},
			{
				Description: "Custom SMTP email with sender name",
				URL:         "mailto://user:pass@mail.company.com:587/admin@company.com?name=Alert+System",
				Code:        `app.Add("mailto://user:pass@mail.company.com:587/admin@company.com?name=Alert+System")`,
			},
		},
		Setup: []string{
			"1. Configure your SMTP server credentials",
			"2. For Gmail: Enable 2FA and generate an app password",
			"3. For other providers: Use provided SMTP settings",
			"4. Test connectivity to SMTP server",
		},
		Limitations: []string{
			"Requires SMTP server access",
			"May be rate-limited by email provider",
			"Credentials stored in configuration",
		},
		Since: "1.0.0",
	}

	dg.services["sendgrid"] = ServiceDocumentation{
		ID:          "sendgrid",
		Name:        "SendGrid Email",
		Description: "Send email notifications via SendGrid API",
		Category:    "email", 
		URLFormat:   "sendgrid://api_key@from_email/to_email",
		Parameters: []ServiceParameter{
			{Name: "api_key", Type: "string", Required: true, Description: "SendGrid API key", Example: "SG.abc123..."},
			{Name: "from_email", Type: "string", Required: true, Description: "Verified sender email", Example: "noreply@myapp.com"},
			{Name: "to_email", Type: "string", Required: true, Description: "Recipient email address", Example: "user@example.com"},
			{Name: "name", Type: "string", Required: false, Description: "Sender name", Example: "MyApp Notifications"},
		},
		Examples: []ServiceExample{
			{
				Description: "SendGrid email notification",
				URL:         "sendgrid://SG.abc123@noreply@myapp.com/user@example.com",
				Code:        `app.Add("sendgrid://SG.abc123@noreply@myapp.com/user@example.com")`,
			},
		},
		Setup: []string{
			"1. Create SendGrid account at https://sendgrid.com",
			"2. Verify your sender identity",
			"3. Generate an API key with Mail Send permissions",
			"4. Configure your from address",
		},
		Since: "1.5.0",
	}
}

// initializeSMSServices adds SMS service documentation
func (dg *DocumentationGenerator) initializeSMSServices() {
	dg.services["twilio"] = ServiceDocumentation{
		ID:          "twilio",
		Name:        "Twilio SMS",
		Description: "Send SMS notifications via Twilio",
		Category:    "sms",
		URLFormat:   "twilio://account_sid:auth_token@from_number/to_number",
		Parameters: []ServiceParameter{
			{Name: "account_sid", Type: "string", Required: true, Description: "Twilio Account SID", Example: "AC123456789abcdef"},
			{Name: "auth_token", Type: "string", Required: true, Description: "Twilio Auth Token", Example: "your_auth_token"},
			{Name: "from_number", Type: "string", Required: true, Description: "Twilio phone number", Example: "+1234567890"},
			{Name: "to_number", Type: "string", Required: true, Description: "Recipient phone number", Example: "+9876543210"},
		},
		Examples: []ServiceExample{
			{
				Description: "Twilio SMS notification",
				URL:         "twilio://AC123456789abcdef:auth_token@+1234567890/+9876543210",
				Code:        `app.Add("twilio://AC123456789abcdef:auth_token@+1234567890/+9876543210")`,
			},
		},
		Setup: []string{
			"1. Create Twilio account at https://twilio.com",
			"2. Purchase a phone number",
			"3. Get Account SID and Auth Token from console",
			"4. Verify recipient numbers (for trial accounts)",
		},
		Since: "1.0.0",
	}
}

// initializeMobileServices adds mobile service documentation  
func (dg *DocumentationGenerator) initializeMobileServices() {
	dg.services["rich-mobile-push"] = ServiceDocumentation{
		ID:          "rich-mobile-push",
		Name:        "Rich Mobile Push Notifications",
		Description: "Advanced mobile push notifications with rich content for iOS and Android",
		Category:    "mobile",
		URLFormat:   "rich-mobile-push://platform@device_tokens",
		Parameters: []ServiceParameter{
			{Name: "platform", Type: "string", Required: true, Description: "Target platform (ios/android/both)", Example: "both"},
			{Name: "device_tokens", Type: "string", Required: true, Description: "Comma-separated device tokens", Example: "token1,token2,token3"},
			{Name: "priority", Type: "string", Required: false, Description: "Notification priority (low/normal/high)", Default: "normal"},
			{Name: "sound", Type: "string", Required: false, Description: "Notification sound", Example: "alert"},
			{Name: "badge", Type: "int", Required: false, Description: "App badge count", Example: "1"},
			{Name: "category", Type: "string", Required: false, Description: "Notification category", Example: "ALERT"},
		},
		Examples: []ServiceExample{
			{
				Description: "Rich mobile push for both platforms",
				URL:         "rich-mobile-push://both@token1,token2,token3?priority=high&sound=alert&badge=1",
				Code:        `app.Add("rich-mobile-push://both@token1,token2,token3?priority=high&sound=alert&badge=1")`,
			},
		},
		Setup: []string{
			"1. Configure push certificates for iOS (APNS)",
			"2. Configure service account for Android (FCM)",
			"3. Collect device tokens from client applications",
			"4. Test with development tokens first",
		},
		Since: "1.9.0",
	}
}

// initializeDesktopServices adds desktop service documentation
func (dg *DocumentationGenerator) initializeDesktopServices() {
	dg.services["desktop-advanced"] = ServiceDocumentation{
		ID:          "desktop-advanced",
		Name:        "Advanced Desktop Notifications",
		Description: "Enhanced desktop notifications with actions, timeouts, and custom UI",
		Category:    "desktop",
		URLFormat:   "desktop-advanced://",
		Parameters: []ServiceParameter{
			{Name: "sound", Type: "string", Required: false, Description: "Notification sound", Default: "default"},
			{Name: "timeout", Type: "int", Required: false, Description: "Timeout in seconds", Default: "5"},
			{Name: "category", Type: "string", Required: false, Description: "Notification category", Example: "ALERT"},
			{Name: "action1", Type: "string", Required: false, Description: "First action (id:title:url)", Example: "view:View Details:https://example.com"},
			{Name: "action2", Type: "string", Required: false, Description: "Second action", Example: "dismiss:Dismiss"},
		},
		Examples: []ServiceExample{
			{
				Description: "Advanced desktop notification with actions",
				URL:         "desktop-advanced://?sound=alert&timeout=10&action1=view:View:https://example.com&action2=dismiss:Dismiss",
				Code:        `app.Add("desktop-advanced://?sound=alert&timeout=10&action1=view:View:https://example.com&action2=dismiss:Dismiss")`,
			},
		},
		Setup: []string{
			"1. No setup required for basic functionality",
			"2. On Linux: Install notify-send or similar",
			"3. On macOS: Uses built-in notification center",
			"4. On Windows: Uses Windows toast notifications",
		},
		Since: "1.9.0",
	}
}

// GetServiceCategories returns all service categories
func (dg *DocumentationGenerator) GetServiceCategories() map[string]ServiceCategory {
	return dg.categories
}

// GetServiceDocumentation returns documentation for a specific service
func (dg *DocumentationGenerator) GetServiceDocumentation(serviceID string) (ServiceDocumentation, bool) {
	doc, exists := dg.services[serviceID]
	return doc, exists
}

// GetAllServiceDocumentation returns documentation for all services
func (dg *DocumentationGenerator) GetAllServiceDocumentation() map[string]ServiceDocumentation {
	return dg.services
}

// GenerateMarkdownDocumentation generates comprehensive markdown documentation
func (dg *DocumentationGenerator) GenerateMarkdownDocumentation() string {
	var sb strings.Builder
	
	// Header
	sb.WriteString("# Apprise-Go Service Documentation\n\n")
	sb.WriteString("This document provides comprehensive documentation for all notification services supported by Apprise-Go.\n\n")
	
	// Table of Contents
	sb.WriteString("## Table of Contents\n\n")
	for _, category := range dg.categories {
		sb.WriteString(fmt.Sprintf("- [%s](#%s)\n", category.Name, strings.ToLower(strings.ReplaceAll(category.Name, " ", "-"))))
	}
	sb.WriteString("\n")
	
	// Categories and Services
	for _, category := range dg.categories {
		sb.WriteString(fmt.Sprintf("## %s\n\n", category.Name))
		sb.WriteString(fmt.Sprintf("%s\n\n", category.Description))
		
		// List services in category
		sb.WriteString("### Services\n\n")
		for _, serviceID := range category.Services {
			if doc, exists := dg.services[serviceID]; exists {
				sb.WriteString(fmt.Sprintf("- [%s](#%s) - %s\n", doc.Name, serviceID, doc.Description))
			}
		}
		sb.WriteString("\n")
		
		// Detailed service documentation
		for _, serviceID := range category.Services {
			if doc, exists := dg.services[serviceID]; exists {
				sb.WriteString(dg.generateServiceMarkdown(doc))
			}
		}
	}
	
	return sb.String()
}

// generateServiceMarkdown generates markdown documentation for a single service
func (dg *DocumentationGenerator) generateServiceMarkdown(doc ServiceDocumentation) string {
	var sb strings.Builder
	
	// Service header
	sb.WriteString(fmt.Sprintf("### %s\n\n", doc.Name))
	sb.WriteString(fmt.Sprintf("**Service ID:** `%s`\n\n", doc.ID))
	sb.WriteString(fmt.Sprintf("%s\n\n", doc.Description))
	
	// URL Format
	sb.WriteString("**URL Format:**\n```\n")
	sb.WriteString(fmt.Sprintf("%s\n", doc.URLFormat))
	sb.WriteString("```\n\n")
	
	// Parameters
	if len(doc.Parameters) > 0 {
		sb.WriteString("**Parameters:**\n\n")
		sb.WriteString("| Name | Type | Required | Description | Default | Example |\n")
		sb.WriteString("|------|------|----------|-------------|---------|----------|\n")
		
		for _, param := range doc.Parameters {
			required := "No"
			if param.Required {
				required = "**Yes**"
			}
			
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s | `%s` |\n",
				param.Name, param.Type, required, param.Description, param.Default, param.Example))
		}
		sb.WriteString("\n")
	}
	
	// Examples
	if len(doc.Examples) > 0 {
		sb.WriteString("**Examples:**\n\n")
		for _, example := range doc.Examples {
			sb.WriteString(fmt.Sprintf("*%s:*\n", example.Description))
			sb.WriteString("```go\n")
			sb.WriteString(fmt.Sprintf("%s\n", example.Code))
			sb.WriteString("```\n\n")
		}
	}
	
	// Setup Instructions
	if len(doc.Setup) > 0 {
		sb.WriteString("**Setup:**\n\n")
		for _, step := range doc.Setup {
			sb.WriteString(fmt.Sprintf("%s\n", step))
		}
		sb.WriteString("\n")
	}
	
	// Limitations
	if len(doc.Limitations) > 0 {
		sb.WriteString("**Limitations:**\n\n")
		for _, limitation := range doc.Limitations {
			sb.WriteString(fmt.Sprintf("- %s\n", limitation))
		}
		sb.WriteString("\n")
	}
	
	sb.WriteString("---\n\n")
	
	return sb.String()
}

// GetServiceByReflection analyzes a service using reflection
func (dg *DocumentationGenerator) GetServiceByReflection(serviceID string) map[string]interface{} {
	service := CreateService(serviceID)
	if service == nil {
		return nil
	}
	
	result := make(map[string]interface{})
	
	// Get type information
	serviceType := reflect.TypeOf(service).Elem()
	result["name"] = serviceType.Name()
	result["package"] = serviceType.PkgPath()
	
	// Get struct fields
	fields := make([]map[string]interface{}, 0)
	for i := 0; i < serviceType.NumField(); i++ {
		field := serviceType.Field(i)
		fieldInfo := map[string]interface{}{
			"name": field.Name,
			"type": field.Type.String(),
			"tag":  string(field.Tag),
		}
		fields = append(fields, fieldInfo)
	}
	result["fields"] = fields
	
	// Get method names
	serviceValue := reflect.ValueOf(service)
	methods := make([]string, 0)
	for i := 0; i < serviceValue.NumMethod(); i++ {
		method := serviceValue.Type().Method(i)
		methods = append(methods, method.Name)
	}
	result["methods"] = methods
	
	return result
}