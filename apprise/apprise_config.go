package apprise

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AppriseConfig manages Apprise-specific configuration with templating support
type AppriseConfig struct {
	configManager *ConfigManager
	template      *ConfigTemplate
	services      []string
	environment   string
	configData    map[string]interface{}
}

// AppriseConfigService represents a service configuration
type AppriseConfigService struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`     // service type (discord, slack, etc.)
	URL      string                 `json:"url"`      // complete service URL
	Enabled  bool                   `json:"enabled"`  // whether service is enabled
	Tags     []string               `json:"tags"`     // service tags
	Priority string                 `json:"priority"` // high, normal, low
	Metadata map[string]interface{} `json:"metadata"` // additional service metadata
}

// AppriseConfigTemplate represents the complete Apprise configuration template
type AppriseConfigTemplate struct {
	Version     string                  `json:"version"`
	Environment string                  `json:"environment"`
	Services    []AppriseConfigService  `json:"services"`
	Defaults    map[string]interface{}  `json:"defaults"`
	Variables   map[string]interface{}  `json:"variables"`
	Tags        []string                `json:"tags"`
	Metadata    map[string]interface{}  `json:"metadata"`
}

// NewAppriseConfig creates a new Apprise configuration manager
func NewAppriseConfig(configDir, environment string) *AppriseConfig {
	return &AppriseConfig{
		configManager: NewConfigManager(configDir),
		template:      NewConfigTemplate(),
		environment:   environment,
		configData:    make(map[string]interface{}),
	}
}

// LoadConfiguration loads Apprise configuration from templates
func (ac *AppriseConfig) LoadConfiguration() error {
	// Load environment-specific variables
	envLoader := NewEnvironmentLoader(ac.environment)
	if err := envLoader.LoadEnvironment(); err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}
	
	// Set up configuration variables
	ac.setupConfigurationVariables()
	
	// Load configuration templates
	if err := ac.configManager.LoadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}
	
	// Configure all loaded templates with the same variables and defaults
	ac.configManager.SetVariableOnAllTemplates("environment", ac.environment)
	ac.configManager.SetDefaultOnAllTemplates("APPRISE_TIMEOUT", "30")
	ac.configManager.SetDefaultOnAllTemplates("APPRISE_LOG_LEVEL", "info")
	ac.configManager.SetDefaultOnAllTemplates("APPRISE_MAX_RETRIES", "3")
	ac.configManager.SetDefaultOnAllTemplates("APPRISE_RETRY_DELAY", "5")
	
	return nil
}

// setupConfigurationVariables sets up template variables for Apprise configuration
func (ac *AppriseConfig) setupConfigurationVariables() {
	// Set environment
	ac.template.SetVariable("environment", ac.environment)
	
	// Set common Apprise defaults
	ac.template.SetDefault("APPRISE_TIMEOUT", "30")
	ac.template.SetDefault("APPRISE_LOG_LEVEL", "info")
	ac.template.SetDefault("APPRISE_MAX_RETRIES", "3")
	ac.template.SetDefault("APPRISE_RETRY_DELAY", "5")
	
	// Add Apprise-specific template functions
	ac.template.AddFunction("serviceURL", ac.buildServiceURL)
	ac.template.AddFunction("enabledServices", ac.getEnabledServices)
	ac.template.AddFunction("servicesByTag", ac.getServicesByTag)
	ac.template.AddFunction("priorityServices", ac.getServicesByPriority)
}

// buildServiceURL builds a complete service URL from components
func (ac *AppriseConfig) buildServiceURL(serviceType, host string, params map[string]string) string {
	url := fmt.Sprintf("%s://%s", serviceType, host)
	
	if len(params) > 0 {
		var paramPairs []string
		for key, value := range params {
			paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", key, value))
		}
		url += "?" + strings.Join(paramPairs, "&")
	}
	
	return url
}

// getEnabledServices returns services that are enabled
func (ac *AppriseConfig) getEnabledServices(services []AppriseConfigService) []AppriseConfigService {
	var enabled []AppriseConfigService
	for _, service := range services {
		if service.Enabled {
			enabled = append(enabled, service)
		}
	}
	return enabled
}

// getServicesByTag returns services with specific tags
func (ac *AppriseConfig) getServicesByTag(services []AppriseConfigService, tag string) []AppriseConfigService {
	var filtered []AppriseConfigService
	for _, service := range services {
		for _, t := range service.Tags {
			if t == tag {
				filtered = append(filtered, service)
				break
			}
		}
	}
	return filtered
}

// getServicesByPriority returns services with specific priority
func (ac *AppriseConfig) getServicesByPriority(services []AppriseConfigService, priority string) []AppriseConfigService {
	var filtered []AppriseConfigService
	for _, service := range services {
		if service.Priority == priority {
			filtered = append(filtered, service)
		}
	}
	return filtered
}

// GenerateAppriseConfig generates Apprise configuration from template
func (ac *AppriseConfig) GenerateAppriseConfig(templateName string) (*Apprise, error) {
	// Get the configuration template
	template, exists := ac.configManager.GetTemplate(templateName)
	if !exists {
		return nil, fmt.Errorf("template %s not found", templateName)
	}
	
	// Execute template to get configuration string
	configStr, err := template.ExecuteToString()
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	
	// Parse configuration and create Apprise instance
	apprise := New()
	if err := ac.parseConfiguration(apprise, configStr); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}
	
	return apprise, nil
}

// parseConfiguration parses configuration string and configures Apprise instance
func (ac *AppriseConfig) parseConfiguration(apprise *Apprise, config string) error {
	lines := strings.Split(config, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse service URL lines
		if strings.Contains(line, "://") {
			// Extract tags if present (format: url #tag1,tag2)
			parts := strings.SplitN(line, "#", 2)
			url := strings.TrimSpace(parts[0])
			
			var tags []string
			if len(parts) > 1 {
				tagStr := strings.TrimSpace(parts[1])
				tags = strings.Split(tagStr, ",")
				for i, tag := range tags {
					tags[i] = strings.TrimSpace(tag)
				}
			}
			
			// Add service to Apprise
			if err := apprise.Add(url, tags...); err != nil {
				return fmt.Errorf("failed to add service %s: %w", url, err)
			}
		}
	}
	
	return nil
}

// CreateDefaultTemplates creates default Apprise configuration templates
func (ac *AppriseConfig) CreateDefaultTemplates() error {
	templateDir := filepath.Join(ac.configManager.configDir, "templates")
	
	// Create default service template
	serviceTemplate := `# Apprise Configuration Template
# Environment: {{var "environment"}}
# Generated: {{formatTime "2006-01-02 15:04:05"}}

# Slack notifications for {{env "TEAM_NAME" "development"}} team
{{if env "SLACK_WEBHOOK_URL"}}slack://{{env "SLACK_WEBHOOK_URL" | replace "https://hooks.slack.com/services/" ""}} #slack,team{{end}}

# Discord notifications for alerts
{{if env "DISCORD_WEBHOOK_URL"}}{{env "DISCORD_WEBHOOK_URL"}} #discord,alerts{{end}}

# Email notifications for critical events
{{if env "SMTP_SERVER"}}mailto://{{env "SMTP_USER"}}:{{env "SMTP_PASS"}}@{{env "SMTP_SERVER"}}:{{env "SMTP_PORT" "587"}}/{{env "ADMIN_EMAIL"}}?smtp_tls=true #email,critical{{end}}

# SMS notifications for emergencies (if Twilio configured)
{{if and (env "TWILIO_SID") (env "TWILIO_TOKEN")}}twilio://{{env "TWILIO_SID"}}:{{env "TWILIO_TOKEN"}}@{{env "TWILIO_FROM_NUMBER" | replace "+" ""}}/{{env "EMERGENCY_PHONE" | replace "+" ""}} #sms,emergency{{end}}

# Desktop notifications for development
{{if eq (var "environment") "development"}}desktop:// #desktop,dev{{end}}

# Rich mobile push for production alerts
{{if and (eq .Vars.environment "production") (env "MOBILE_DEVICE_TOKENS")}}rich-mobile-push://both@{{env "MOBILE_DEVICE_TOKENS"}}?priority=high&sound=alert&badge=1 #mobile,production{{end}}

# Webhook for monitoring system integration
{{if env "MONITORING_WEBHOOK_URL"}}{{env "MONITORING_WEBHOOK_URL"}} #webhook,monitoring{{end}}`

	serviceTemplateFile := filepath.Join(templateDir, "services.tmpl")
	if err := os.WriteFile(serviceTemplateFile, []byte(serviceTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create service template: %w", err)
	}

	// Create environment-specific template
	envTemplate := `# Environment-specific Apprise Configuration
# Environment: {{var "environment"}}

{{if eq (var "environment") "development"}}
# Development environment - local services only
desktop:// #dev,local
{{if env "DEV_WEBHOOK_URL"}}{{env "DEV_WEBHOOK_URL"}} #dev,webhook{{end}}

{{else if eq (var "environment") "staging"}}
# Staging environment - limited notifications
{{if env "STAGING_SLACK_URL"}}{{env "STAGING_SLACK_URL"}} #staging,slack{{end}}
{{if env "STAGING_EMAIL"}}mailto://{{env "SMTP_SERVER"}}/{{env "STAGING_EMAIL"}} #staging,email{{end}}

{{else if eq (var "environment") "production"}}
# Production environment - full notification suite
{{if env "PROD_SLACK_URL"}}{{env "PROD_SLACK_URL"}} #prod,slack,alerts{{end}}
{{if env "PROD_DISCORD_URL"}}{{env "PROD_DISCORD_URL"}} #prod,discord,alerts{{end}}
{{if env "PROD_EMAIL"}}mailto://{{env "SMTP_SERVER"}}/{{env "PROD_EMAIL"}} #prod,email,alerts{{end}}
{{if env "PROD_SMS_NUMBER"}}twilio://{{env "TWILIO_SID"}}:{{env "TWILIO_TOKEN"}}@{{env "TWILIO_FROM"}}/{{env "PROD_SMS_NUMBER"}} #prod,sms,critical{{end}}
{{if env "PROD_MOBILE_TOKENS"}}rich-mobile-push://both@{{env "PROD_MOBILE_TOKENS"}}?priority=high&sound=critical&badge=1 #prod,mobile,critical{{end}}

{{end}}

# Common webhook endpoints
{{if env "DATADOG_WEBHOOK"}}{{env "DATADOG_WEBHOOK"}} #monitoring,datadog{{end}}
{{if env "NEWRELIC_WEBHOOK"}}{{env "NEWRELIC_WEBHOOK"}} #monitoring,newrelic{{end}}
{{if env "PAGERDUTY_URL"}}{{env "PAGERDUTY_URL"}} #incident,pagerduty{{end}}`

	envTemplateFile := filepath.Join(templateDir, "environment.tmpl")
	if err := os.WriteFile(envTemplateFile, []byte(envTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create environment template: %w", err)
	}

	// Create complete configuration template
	completeTemplate := `# Complete Apprise Configuration
# Environment: {{var "environment"}}
# Team: {{env "TEAM_NAME" "default"}}
# Generated: {{formatTime "2006-01-02 15:04:05"}}
# Configuration version: {{default "CONFIG_VERSION" "1.0"}}

# =================================
# COMMUNICATION PLATFORMS
# =================================

# Slack integration
{{if env "SLACK_WEBHOOK_URL"}}
# Slack webhook for general notifications
slack://{{env "SLACK_WEBHOOK_URL" | replace "https://hooks.slack.com/services/" ""}}?format=markdown #slack,general
{{end}}

# Discord integration  
{{if env "DISCORD_WEBHOOK_URL"}}
# Discord webhook for team alerts
{{env "DISCORD_WEBHOOK_URL"}}?avatar=Apprise&format=markdown #discord,alerts
{{end}}

# Microsoft Teams
{{if env "TEAMS_WEBHOOK_URL"}}
# Microsoft Teams for enterprise notifications
{{env "TEAMS_WEBHOOK_URL"}} #teams,enterprise
{{end}}

# =================================
# EMAIL NOTIFICATIONS
# =================================

{{if and (env "SMTP_SERVER") (env "ADMIN_EMAIL")}}
# SMTP email for administrative notifications
mailto://{{env "SMTP_USER" | urlEncode}}:{{env "SMTP_PASS" | urlEncode}}@{{env "SMTP_SERVER"}}:{{env "SMTP_PORT" "587"}}/{{env "ADMIN_EMAIL"}}?smtp_tls={{env "SMTP_TLS" "true"}}&name={{env "SERVICE_NAME" "Apprise"}} #email,admin

{{if env "ALERT_EMAIL"}}
# High-priority email alerts
mailto://{{env "SMTP_USER" | urlEncode}}:{{env "SMTP_PASS" | urlEncode}}@{{env "SMTP_SERVER"}}:{{env "SMTP_PORT" "587"}}/{{env "ALERT_EMAIL"}}?smtp_tls={{env "SMTP_TLS" "true"}}&name={{env "SERVICE_NAME" "Apprise"}} #email,alerts,critical
{{end}}
{{end}}

# =================================
# SMS AND MOBILE
# =================================

{{if and (env "TWILIO_SID") (env "TWILIO_TOKEN") (env "EMERGENCY_PHONE")}}
# Twilio SMS for emergency notifications
twilio://{{env "TWILIO_SID"}}:{{env "TWILIO_TOKEN"}}@{{env "TWILIO_FROM" | replace "+" ""}}/{{env "EMERGENCY_PHONE" | replace "+" ""}} #sms,emergency,critical
{{end}}

{{if env "MOBILE_PUSH_TOKENS"}}
# Rich mobile push notifications
rich-mobile-push://both@{{env "MOBILE_PUSH_TOKENS"}}?sound={{env "PUSH_SOUND" "alert"}}&priority={{env "PUSH_PRIORITY" "high"}}&badge=1&reply=true #mobile,push,interactive
{{end}}

# =================================
# DEVELOPMENT TOOLS
# =================================

{{if eq (var "environment") "development"}}
# Desktop notifications for development
desktop-advanced://?sound=default&timeout=10&category=development #desktop,dev,local
{{end}}

{{if env "DEV_WEBHOOK_URL"}}
# Development webhook for testing
{{env "DEV_WEBHOOK_URL"}}?format=json #webhook,dev,testing
{{end}}

# =================================
# MONITORING AND INCIDENT MANAGEMENT
# =================================

{{if env "PAGERDUTY_ROUTING_KEY"}}
# PagerDuty for incident management
pagerduty://{{env "PAGERDUTY_ROUTING_KEY"}}@{{env "PAGERDUTY_REGION" "us"}}/?priority=high #incident,pagerduty,critical
{{end}}

{{if env "OPSGENIE_API_KEY"}}
# Opsgenie for operations alerts
opsgenie://{{env "OPSGENIE_API_KEY"}}@{{env "OPSGENIE_REGION" "us"}}/?priority=P1&tags=apprise,{{var "environment"}} #incident,opsgenie,operations
{{end}}

{{if env "DATADOG_API_KEY"}}
# Datadog events integration
datadog://{{env "DATADOG_API_KEY"}}@{{env "DATADOG_REGION" "us"}}/?tags=service:{{env "SERVICE_NAME" "apprise"}},env:{{var "environment"}} #monitoring,datadog,events
{{end}}

# =================================
# IOT AND AUTOMATION
# =================================

{{if env "IFTTT_WEBHOOK_KEY"}}
# IFTTT automation triggers
ifttt://{{env "IFTTT_WEBHOOK_KEY"}}@{{env "IFTTT_EVENT" "apprise_notification"}} #iot,ifttt,automation
{{end}}

{{if env "ZAPIER_WEBHOOK_URL"}}
# Zapier workflow integration
{{env "ZAPIER_WEBHOOK_URL"}} #automation,zapier,workflow
{{end}}

{{if env "HOME_ASSISTANT_TOKEN"}}
# Home Assistant smart home notifications
homeassistant://{{env "HOME_ASSISTANT_TOKEN"}}@{{env "HOME_ASSISTANT_HOST" "localhost"}}:{{env "HOME_ASSISTANT_PORT" "8123"}}/{{env "HOME_ASSISTANT_SERVICE" "persistent_notification/create"}} #iot,homeassistant,smarthome
{{end}}

# =================================
# CUSTOM WEBHOOKS
# =================================

{{if env "CUSTOM_WEBHOOK_URL"}}
# Custom webhook endpoint
webhook://{{env "CUSTOM_WEBHOOK_URL" | replace "https://" "" | replace "http://" ""}}?format=json&timeout={{env "WEBHOOK_TIMEOUT" "30"}} #webhook,custom
{{end}}

# Configuration ends here`

	completeTemplateFile := filepath.Join(templateDir, "complete.tmpl")
	if err := os.WriteFile(completeTemplateFile, []byte(completeTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create complete template: %w", err)
	}

	return nil
}

// CreateSampleEnvironmentFiles creates sample environment files for different environments
func (ac *AppriseConfig) CreateSampleEnvironmentFiles() error {
	configDir := ac.configManager.configDir
	
	// Create sample .env file
	sampleEnv := `# Apprise Configuration Environment Variables
# Copy this file to .env and customize for your environment

# Environment setting
ENVIRONMENT=development

# Service identification
SERVICE_NAME=MyApp
TEAM_NAME=MyTeam

# Slack integration
# SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK

# Discord integration  
# DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR/DISCORD/WEBHOOK

# Email configuration
# SMTP_SERVER=smtp.gmail.com
# SMTP_PORT=587
# SMTP_USER=your-email@gmail.com
# SMTP_PASS=your-app-password
# SMTP_TLS=true
# ADMIN_EMAIL=admin@yourcompany.com
# ALERT_EMAIL=alerts@yourcompany.com

# SMS configuration (Twilio)
# TWILIO_SID=your-twilio-sid
# TWILIO_TOKEN=your-twilio-token
# TWILIO_FROM=+1234567890
# EMERGENCY_PHONE=+1987654321

# Mobile push notifications
# MOBILE_PUSH_TOKENS=device_token1,device_token2
# PUSH_SOUND=alert
# PUSH_PRIORITY=high

# Incident management
# PAGERDUTY_ROUTING_KEY=your-pagerduty-routing-key
# OPSGENIE_API_KEY=your-opsgenie-api-key

# Monitoring
# DATADOG_API_KEY=your-datadog-api-key
# NEWRELIC_API_KEY=your-newrelic-api-key

# IoT and automation
# IFTTT_WEBHOOK_KEY=your-ifttt-webhook-key
# IFTTT_EVENT=apprise_notification
# ZAPIER_WEBHOOK_URL=your-zapier-webhook-url
# HOME_ASSISTANT_TOKEN=your-ha-token
# HOME_ASSISTANT_HOST=localhost

# Custom webhooks
# CUSTOM_WEBHOOK_URL=https://your-custom-webhook.com/notify`

	sampleEnvFile := filepath.Join(configDir, ".env.sample")
	if err := os.WriteFile(sampleEnvFile, []byte(sampleEnv), 0644); err != nil {
		return fmt.Errorf("failed to create sample env file: %w", err)
	}

	// Create production environment sample
	prodEnv := `# Production Environment Variables
# Copy this file to .env.production and customize

ENVIRONMENT=production
SERVICE_NAME=MyApp-Production
TEAM_NAME=ProductionTeam

# Production Slack
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/PRODUCTION/WEBHOOK

# Production email
SMTP_SERVER=smtp.yourcompany.com
SMTP_PORT=587
SMTP_USER=notifications@yourcompany.com
SMTP_PASS=production-smtp-password
ADMIN_EMAIL=admin@yourcompany.com
ALERT_EMAIL=alerts@yourcompany.com

# Production SMS for critical alerts
TWILIO_SID=production-twilio-sid
TWILIO_TOKEN=production-twilio-token
TWILIO_FROM=+1234567890
EMERGENCY_PHONE=+1987654321

# Production mobile push
MOBILE_PUSH_TOKENS=prod_token1,prod_token2
PUSH_SOUND=critical
PUSH_PRIORITY=high

# Incident management
PAGERDUTY_ROUTING_KEY=production-pagerduty-key
OPSGENIE_API_KEY=production-opsgenie-key

# Production monitoring
DATADOG_API_KEY=production-datadog-key`

	prodEnvFile := filepath.Join(configDir, ".env.production")
	if err := os.WriteFile(prodEnvFile, []byte(prodEnv), 0644); err != nil {
		return fmt.Errorf("failed to create production env file: %w", err)
	}

	return nil
}

// LoadFromTemplate loads an Apprise instance from a configuration template
func LoadFromTemplate(configDir, environment, templateName string) (*Apprise, error) {
	config := NewAppriseConfig(configDir, environment)
	
	if err := config.LoadConfiguration(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	apprise, err := config.GenerateAppriseConfig(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Apprise config: %w", err)
	}
	
	return apprise, nil
}

// SetupDefaultConfiguration sets up default Apprise configuration with templates
func SetupDefaultConfiguration(configDir string) error {
	config := NewAppriseConfig(configDir, "development")
	
	// Create default templates
	if err := config.CreateDefaultTemplates(); err != nil {
		return fmt.Errorf("failed to create templates: %w", err)
	}
	
	// Create sample environment files
	if err := config.CreateSampleEnvironmentFiles(); err != nil {
		return fmt.Errorf("failed to create sample env files: %w", err)
	}
	
	return nil
}