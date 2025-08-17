package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/scttfrdmn/apprise-go/apprise"
)

const (
	appName        = "apprise-cli"
	configFileName = "apprise"
)

// Config represents the CLI configuration
type Config struct {
	Services []ServiceConfig `json:"services" yaml:"services" toml:"services"`
	Default  DefaultConfig   `json:"default" yaml:"default" toml:"default"`
}

// ServiceConfig represents a service configuration
type ServiceConfig struct {
	Name        string            `json:"name" yaml:"name" toml:"name"`
	URL         string            `json:"url" yaml:"url" toml:"url"`
	Tags        []string          `json:"tags,omitempty" yaml:"tags,omitempty" toml:"tags,omitempty"`
	Enabled     bool              `json:"enabled" yaml:"enabled" toml:"enabled"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty" toml:"metadata,omitempty"`
}

// DefaultConfig represents default CLI settings
type DefaultConfig struct {
	NotifyType string        `json:"notify_type" yaml:"notify_type" toml:"notify_type"`
	Timeout    time.Duration `json:"timeout" yaml:"timeout" toml:"timeout"`
	Tags       []string      `json:"tags,omitempty" yaml:"tags,omitempty" toml:"tags,omitempty"`
	BodyFormat string        `json:"body_format,omitempty" yaml:"body_format,omitempty" toml:"body_format,omitempty"`
}

// CLIOptions represents command-line options
type CLIOptions struct {
	ConfigFile   string
	Services     []string
	Tags         []string
	NotifyType   string
	Timeout      time.Duration
	BodyFormat   string
	Attachments  []string
	URL          string
	Interactive  bool
	Dry          bool
	Verbose      bool
	JSON         bool
	Title        string
	Body         string
}

var (
	opts      CLIOptions
	rootCmd   *cobra.Command
	config    Config
)

func init() {
	rootCmd = &cobra.Command{
		Use:   appName,
		Short: "A comprehensive command-line interface for Apprise-Go notifications",
		Long: `Apprise-CLI is a powerful command-line tool for sending notifications through
multiple services using the Apprise-Go library. It supports configuration files,
interactive mode, and comprehensive service management.`,
		Version: apprise.GetVersion(),
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&opts.ConfigFile, "config", "c", "", "config file (default is $HOME/.apprise.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&opts.JSON, "json", "j", false, "output in JSON format")

	// Add subcommands
	rootCmd.AddCommand(
		createNotifyCommand(),
		createServicesCommand(),
		createConfigCommand(),
		createTestCommand(),
		createInteractiveCommand(),
		createVersionCommand(),
	)

	// Initialize configuration
	initConfig()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	if opts.ConfigFile != "" {
		viper.SetConfigFile(opts.ConfigFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not find home directory: %v\n", err)
			return
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(configFileName)
		viper.SetConfigType("yaml") // Default to YAML
	}

	// Environment variable support
	viper.SetEnvPrefix("APPRISE")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "No config file found, using defaults: %v\n", err)
		}
		
		// Set defaults
		config = Config{
			Services: []ServiceConfig{},
			Default: DefaultConfig{
				NotifyType: "info",
				Timeout:    30 * time.Second,
				Tags:       []string{},
				BodyFormat: "text",
			},
		}
	} else {
		if err := viper.Unmarshal(&config); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing config file: %v\n", err)
			os.Exit(1)
		}
		
		if opts.Verbose {
			fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
		}
	}
}

func createNotifyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notify [title] [body]",
		Short: "Send a notification",
		Long: `Send a notification to configured services. You can specify the title and body
as arguments or use flags. If neither are provided, you'll be prompted to enter them.`,
		Args:    cobra.MaximumNArgs(2),
		Example: `  apprise-cli notify "Alert" "System maintenance starting"
  apprise-cli notify --service discord --type error "Critical Error" "Database down"
  apprise-cli notify --tags urgent,system --attach /path/to/log.txt "Log Alert"`,
		RunE: runNotifyCommand,
	}

	cmd.Flags().StringSliceVarP(&opts.Services, "service", "s", []string{}, "specific services to notify (names or URLs)")
	cmd.Flags().StringSliceVarP(&opts.Tags, "tags", "t", []string{}, "tags for filtering services")
	cmd.Flags().StringVarP(&opts.NotifyType, "type", "T", "", "notification type (info, success, warning, error)")
	cmd.Flags().DurationVarP(&opts.Timeout, "timeout", "o", 0, "timeout for notifications")
	cmd.Flags().StringVarP(&opts.BodyFormat, "format", "f", "", "body format (text, html, markdown)")
	cmd.Flags().StringSliceVarP(&opts.Attachments, "attach", "a", []string{}, "file attachments")
	cmd.Flags().StringVarP(&opts.URL, "url", "u", "", "URL to include with notification")
	cmd.Flags().BoolVarP(&opts.Dry, "dry-run", "d", false, "show what would be sent without actually sending")
	cmd.Flags().StringVarP(&opts.Title, "title", "", "", "notification title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "notification body")

	return cmd
}

func createServicesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "Manage notification services",
		Long:  "List, add, remove, and test notification services",
	}

	// List services
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured services",
		RunE:  runListServicesCommand,
	}
	
	// Add service
	addCmd := &cobra.Command{
		Use:   "add [name] [url]",
		Short: "Add a new service",
		Args:  cobra.ExactArgs(2),
		RunE:  runAddServiceCommand,
	}
	addCmd.Flags().StringSliceVarP(&opts.Tags, "tags", "t", []string{}, "tags for the service")
	addCmd.Flags().String("description", "", "service description")

	// Remove service
	removeCmd := &cobra.Command{
		Use:   "remove [name]",
		Short: "Remove a service",
		Args:  cobra.ExactArgs(1),
		RunE:  runRemoveServiceCommand,
	}

	// Test service
	testCmd := &cobra.Command{
		Use:   "test [name_or_url]",
		Short: "Test a service configuration",
		Args:  cobra.ExactArgs(1),
		RunE:  runTestServiceCommand,
	}

	// Supported services
	supportedCmd := &cobra.Command{
		Use:   "supported",
		Short: "List all supported service types",
		RunE:  runSupportedServicesCommand,
	}

	cmd.AddCommand(listCmd, addCmd, removeCmd, testCmd, supportedCmd)
	return cmd
}

func createConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  "Generate, validate, and manage configuration files",
	}

	// Generate config
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a sample configuration file",
		RunE:  runGenerateConfigCommand,
	}
	generateCmd.Flags().String("output", "", "output file (default: stdout)")
	generateCmd.Flags().String("format", "yaml", "output format (yaml, json, toml)")

	// Validate config
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		RunE:  runValidateConfigCommand,
	}

	// Show config
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE:  runShowConfigCommand,
	}

	cmd.AddCommand(generateCmd, validateCmd, showCmd)
	return cmd
}

func createTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test [service_name_or_url]",
		Short: "Test service configurations",
		Long: `Test one or more notification services to ensure they're working correctly.
If no service is specified, all configured services will be tested.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runTestCommand,
	}
}

func createInteractiveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "interactive",
		Short: "Interactive mode",
		Long:  "Enter interactive mode for sending notifications with prompts",
		RunE:  runInteractiveCommand,
	}
}

func createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE:  runVersionCommand,
	}
}

func runNotifyCommand(cmd *cobra.Command, args []string) error {
	// Parse title and body from args or flags
	title := opts.Title
	body := opts.Body

	if len(args) >= 1 && title == "" {
		title = args[0]
	}
	if len(args) >= 2 && body == "" {
		body = args[1]
	}

	// Prompt for missing required fields
	if title == "" {
		title = promptForInput("Enter notification title: ")
	}
	if body == "" {
		body = promptForInput("Enter notification body: ")
	}

	// Create Apprise instance
	app := apprise.New()

	// Apply configuration
	notifyType := parseNotifyType(opts.NotifyType)
	if opts.Timeout > 0 {
		app.SetTimeout(opts.Timeout)
	} else if config.Default.Timeout > 0 {
		app.SetTimeout(config.Default.Timeout)
	}

	// Add services
	services := getServicesToNotify()
	if len(services) == 0 {
		return fmt.Errorf("no services configured or specified")
	}

	for _, service := range services {
		if err := app.Add(service.URL); err != nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to add service %s: %v\n", service.Name, err)
			}
			continue
		}
	}

	// Add attachments
	for _, attachment := range opts.Attachments {
		if err := app.AddAttachment(attachment); err != nil {
			return fmt.Errorf("failed to add attachment %s: %w", attachment, err)
		}
	}

	// Prepare notification options
	var notifyOptions []apprise.NotifyOption
	
	// Add tags
	allTags := append(opts.Tags, config.Default.Tags...)
	if len(allTags) > 0 {
		notifyOptions = append(notifyOptions, apprise.WithTags(allTags...))
	}

	// Add body format
	bodyFormat := opts.BodyFormat
	if bodyFormat == "" {
		bodyFormat = config.Default.BodyFormat
	}
	if bodyFormat != "" {
		notifyOptions = append(notifyOptions, apprise.WithBodyFormat(bodyFormat))
	}

	// Dry run check
	if opts.Dry {
		fmt.Printf("DRY RUN - Would send notification:\n")
		fmt.Printf("  Title: %s\n", title)
		fmt.Printf("  Body: %s\n", body)
		fmt.Printf("  Type: %s\n", notifyType.String())
		fmt.Printf("  Services: %d configured\n", len(services))
		if len(allTags) > 0 {
			fmt.Printf("  Tags: %v\n", allTags)
		}
		if len(opts.Attachments) > 0 {
			fmt.Printf("  Attachments: %v\n", opts.Attachments)
		}
		return nil
	}

	// Send notification
	start := time.Now()
	responses := app.Notify(title, body, notifyType, notifyOptions...)
	duration := time.Since(start)

	// Process results
	successful := 0
	failed := 0
	
	for _, resp := range responses {
		if resp.Success {
			successful++
		} else {
			failed++
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "Failed to send to %s: %v\n", resp.ServiceID, resp.Error)
			}
		}
	}

	// Output results
	if opts.JSON {
		result := map[string]interface{}{
			"success":    successful,
			"failed":     failed,
			"total":      len(responses),
			"duration":   duration.String(),
			"responses":  responses,
		}
		
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON response: %w", err)
		}
		
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Notification sent to %d services (%d successful, %d failed) in %v\n", 
			len(responses), successful, failed, duration)
		
		if failed > 0 && !opts.Verbose {
			fmt.Printf("Use --verbose flag to see error details\n")
		}
	}

	if failed > 0 {
		os.Exit(1)
	}

	return nil
}

func runListServicesCommand(cmd *cobra.Command, args []string) error {
	if len(config.Services) == 0 {
		fmt.Println("No services configured")
		return nil
	}

	if opts.JSON {
		jsonData, err := json.MarshalIndent(config.Services, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	fmt.Printf("Configured Services (%d):\n\n", len(config.Services))
	
	for _, service := range config.Services {
		status := "✓ Enabled"
		if !service.Enabled {
			status = "✗ Disabled"
		}
		
		fmt.Printf("  %s %s\n", status, service.Name)
		if service.Description != "" {
			fmt.Printf("    Description: %s\n", service.Description)
		}
		
		// Parse URL to show service type
		if parsedURL, err := parseServiceURL(service.URL); err == nil {
			fmt.Printf("    Type: %s\n", parsedURL.Scheme)
		}
		
		if len(service.Tags) > 0 {
			fmt.Printf("    Tags: %v\n", service.Tags)
		}
		fmt.Println()
	}

	return nil
}

func runAddServiceCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	serviceURL := args[1]
	
	// Check if service name already exists
	for _, service := range config.Services {
		if service.Name == name {
			return fmt.Errorf("service with name '%s' already exists", name)
		}
	}
	
	// Test the service URL
	app := apprise.New()
	if err := app.Add(serviceURL); err != nil {
		return fmt.Errorf("invalid service URL: %w", err)
	}
	
	// Get additional flags
	tags, _ := cmd.Flags().GetStringSlice("tags")
	description, _ := cmd.Flags().GetString("description")
	
	// Add to config
	newService := ServiceConfig{
		Name:        name,
		URL:         serviceURL,
		Tags:        tags,
		Enabled:     true,
		Description: description,
		Metadata:    make(map[string]string),
	}
	
	config.Services = append(config.Services, newService)
	
	// Save config
	if err := saveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	fmt.Printf("Service '%s' added successfully\n", name)
	return nil
}

func runRemoveServiceCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	// Find and remove service
	for i, service := range config.Services {
		if service.Name == name {
			config.Services = append(config.Services[:i], config.Services[i+1:]...)
			
			// Save config
			if err := saveConfig(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}
			
			fmt.Printf("Service '%s' removed successfully\n", name)
			return nil
		}
	}
	
	return fmt.Errorf("service '%s' not found", name)
}

func runTestServiceCommand(cmd *cobra.Command, args []string) error {
	nameOrURL := args[0]
	
	// Check if it's a configured service name or a URL
	var serviceURL string
	var serviceName string
	
	// First try to find by name
	for _, service := range config.Services {
		if service.Name == nameOrURL {
			serviceURL = service.URL
			serviceName = service.Name
			break
		}
	}
	
	// If not found, assume it's a URL
	if serviceURL == "" {
		serviceURL = nameOrURL
		serviceName = "test-service"
	}
	
	// Test the service
	app := apprise.New()
	if err := app.Add(serviceURL); err != nil {
		return fmt.Errorf("service test failed: %w", err)
	}
	
	// Send test notification
	responses := app.Notify("Test Notification", 
		fmt.Sprintf("This is a test notification from Apprise-CLI at %s", time.Now().Format(time.RFC3339)),
		apprise.NotifyTypeInfo)
	
	if len(responses) == 0 {
		return fmt.Errorf("no responses received")
	}
	
	resp := responses[0]
	if resp.Success {
		fmt.Printf("✓ Service '%s' test successful (took %v)\n", serviceName, resp.Duration)
	} else {
		return fmt.Errorf("✗ Service '%s' test failed: %v", serviceName, resp.Error)
	}
	
	return nil
}

func runSupportedServicesCommand(cmd *cobra.Command, args []string) error {
	// This is a bit of a hack since the registry doesn't expose supported services
	// In a real implementation, we'd add a GetSupportedServices method
	supportedServices := []string{
		"discord", "slack", "telegram", "tgram", "pushover", "pover", "pushbullet", "pball",
		"mailto", "mailtos", "webhook", "webhooks", "json", "fcm", "apns", "msteams",
		"mattermost", "mmosts", "rocketchat", "rocket", "pagerduty", "opsgenie", "matrix",
		"twilio", "twilio-voice", "polly", "aws-iot", "gcp-iot", "gotify", "gotifys",
		"ntfy", "ntfys", "sns", "ses", "azuresb", "pubsub", "datadog", "newrelic",
		"gitlab", "github", "jira", "twitter", "linkedin", "desktop", "macosx", "windows",
		"linux", "dbus", "gnome", "kde", "glib", "qt",
	}
	
	sort.Strings(supportedServices)
	
	if opts.JSON {
		jsonData, err := json.MarshalIndent(supportedServices, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}
	
	fmt.Printf("Supported Service Types (%d):\n\n", len(supportedServices))
	
	// Group by category for better readability
	categories := map[string][]string{
		"Messaging/Chat":     {"discord", "slack", "telegram", "tgram", "msteams", "mattermost", "mmosts", "rocketchat", "rocket", "matrix"},
		"Email":             {"mailto", "mailtos"},
		"Push Notifications": {"pushover", "pover", "pushbullet", "pball", "fcm", "apns"},
		"Webhooks":          {"webhook", "webhooks", "json"},
		"SMS/Voice":         {"twilio", "twilio-voice", "polly"},
		"IoT":               {"aws-iot", "gcp-iot"},
		"Self-Hosted":       {"gotify", "gotifys", "ntfy", "ntfys"},
		"Cloud Services":    {"sns", "ses", "azuresb", "pubsub"},
		"Monitoring":        {"datadog", "newrelic", "pagerduty", "opsgenie"},
		"DevOps":           {"gitlab", "github", "jira"},
		"Social Media":      {"twitter", "linkedin"},
		"Desktop":          {"desktop", "macosx", "windows", "linux", "dbus", "gnome", "kde", "glib", "qt"},
	}
	
	for category, services := range categories {
		fmt.Printf("%s:\n", category)
		for _, service := range services {
			// Only show if it's in our supported list
			for _, supported := range supportedServices {
				if service == supported {
					fmt.Printf("  %s\n", service)
					break
				}
			}
		}
		fmt.Println()
	}
	
	return nil
}

// Helper functions continue in the next part...

func parseNotifyType(typeStr string) apprise.NotifyType {
	switch strings.ToLower(typeStr) {
	case "error":
		return apprise.NotifyTypeError
	case "warning", "warn":
		return apprise.NotifyTypeWarning
	case "success":
		return apprise.NotifyTypeSuccess
	case "info", "":
		return apprise.NotifyTypeInfo
	default:
		return apprise.NotifyTypeInfo
	}
}

func promptForInput(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func getServicesToNotify() []ServiceConfig {
	var services []ServiceConfig
	
	// If specific services are requested
	if len(opts.Services) > 0 {
		for _, serviceNameOrURL := range opts.Services {
			// Check if it's a configured service name
			found := false
			for _, configService := range config.Services {
				if configService.Name == serviceNameOrURL && configService.Enabled {
					services = append(services, configService)
					found = true
					break
				}
			}
			
			// If not found, assume it's a direct URL
			if !found {
				services = append(services, ServiceConfig{
					Name: "direct",
					URL:  serviceNameOrURL,
					Enabled: true,
				})
			}
		}
		return services
	}
	
	// Use all enabled services
	for _, service := range config.Services {
		if service.Enabled {
			// Filter by tags if specified
			if len(opts.Tags) > 0 {
				hasTag := false
				for _, optTag := range opts.Tags {
					for _, serviceTag := range service.Tags {
						if strings.EqualFold(optTag, serviceTag) {
							hasTag = true
							break
						}
					}
					if hasTag {
						break
					}
				}
				if hasTag {
					services = append(services, service)
				}
			} else {
				services = append(services, service)
			}
		}
	}
	
	return services
}

func parseServiceURL(serviceURL string) (*url.URL, error) {
	return url.Parse(serviceURL)
}

func saveConfig() error {
	if opts.ConfigFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		opts.ConfigFile = filepath.Join(home, ".apprise.yaml")
	}
	
	viper.Set("services", config.Services)
	viper.Set("default", config.Default)
	
	return viper.WriteConfigAs(opts.ConfigFile)
}

// Additional command implementations
func runGenerateConfigCommand(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	
	sampleConfig := Config{
		Services: []ServiceConfig{
			{
				Name:        "discord-alerts",
				URL:         "discord://webhook_id/webhook_token",
				Tags:        []string{"alerts", "urgent"},
				Enabled:     true,
				Description: "Discord channel for urgent alerts",
			},
			{
				Name:        "slack-general",
				URL:         "slack://bottoken@workspace.slack.com/general",
				Tags:        []string{"general", "info"},
				Enabled:     true,
				Description: "Slack general channel for notifications",
			},
		},
		Default: DefaultConfig{
			NotifyType: "info",
			Timeout:    30 * time.Second,
			Tags:       []string{},
			BodyFormat: "text",
		},
	}
	
	var data []byte
	var err error
	
	switch format {
	case "json":
		data, err = json.MarshalIndent(sampleConfig, "", "  ")
	case "yaml":
		// Would use yaml.Marshal in real implementation
		data = []byte(`# Apprise CLI Configuration File

services:
  - name: discord-alerts
    url: discord://webhook_id/webhook_token
    tags:
      - alerts
      - urgent
    enabled: true
    description: Discord channel for urgent alerts
    
  - name: slack-general
    url: slack://bottoken@workspace.slack.com/general
    tags:
      - general  
      - info
    enabled: true
    description: Slack general channel for notifications

default:
  notify_type: info
  timeout: 30s
  tags: []
  body_format: text
`)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}
	
	if output == "" {
		fmt.Print(string(data))
	} else {
		if err := os.WriteFile(output, data, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
		fmt.Printf("Configuration written to %s\n", output)
	}
	
	return nil
}

func runValidateConfigCommand(cmd *cobra.Command, args []string) error {
	// Config is already loaded and validated during init
	fmt.Printf("✓ Configuration is valid\n")
	fmt.Printf("  Config file: %s\n", viper.ConfigFileUsed())
	fmt.Printf("  Services: %d\n", len(config.Services))
	fmt.Printf("  Enabled services: %d\n", countEnabledServices())
	
	return nil
}

func runShowConfigCommand(cmd *cobra.Command, args []string) error {
	if opts.JSON {
		jsonData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}
	
	fmt.Printf("Current Configuration:\n\n")
	fmt.Printf("Config File: %s\n", viper.ConfigFileUsed())
	fmt.Printf("Services: %d (%d enabled)\n", len(config.Services), countEnabledServices())
	fmt.Printf("Default Type: %s\n", config.Default.NotifyType)
	fmt.Printf("Default Timeout: %v\n", config.Default.Timeout)
	
	return nil
}

func runTestCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 1 {
		return runTestServiceCommand(cmd, args)
	}
	
	// Test all services
	services := getServicesToNotify()
	if len(services) == 0 {
		return fmt.Errorf("no services to test")
	}
	
	fmt.Printf("Testing %d services...\n\n", len(services))
	
	successful := 0
	failed := 0
	
	for _, service := range services {
		fmt.Printf("Testing %s... ", service.Name)
		
		app := apprise.New()
		if err := app.Add(service.URL); err != nil {
			fmt.Printf("✗ Configuration error: %v\n", err)
			failed++
			continue
		}
		
		responses := app.Notify("Test Notification", 
			fmt.Sprintf("Test from Apprise-CLI at %s", time.Now().Format(time.RFC3339)),
			apprise.NotifyTypeInfo)
		
		if len(responses) > 0 && responses[0].Success {
			fmt.Printf("✓ Success (%v)\n", responses[0].Duration)
			successful++
		} else {
			fmt.Printf("✗ Failed")
			if len(responses) > 0 && responses[0].Error != nil {
				fmt.Printf(": %v", responses[0].Error)
			}
			fmt.Println()
			failed++
		}
	}
	
	fmt.Printf("\nTest Results: %d successful, %d failed\n", successful, failed)
	
	if failed > 0 {
		os.Exit(1)
	}
	
	return nil
}

func runInteractiveCommand(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Apprise-CLI Interactive Mode ===")
	fmt.Println("Enter your notification details. Type 'quit' to exit.")
	
	for {
		fmt.Println("--- New Notification ---")
		
		title := promptForInput("Title: ")
		if title == "quit" {
			break
		}
		
		body := promptForInput("Body: ")
		if body == "quit" {
			break
		}
		
		typeStr := promptForInput("Type (info/success/warning/error) [info]: ")
		if typeStr == "quit" {
			break
		}
		if typeStr == "" {
			typeStr = "info"
		}
		
		services := promptForInput("Services (comma-separated, or 'all') [all]: ")
		if services == "quit" {
			break
		}
		if services == "" {
			services = "all"
		}
		
		// Send notification
		opts.Title = title
		opts.Body = body
		opts.NotifyType = typeStr
		
		if services != "all" {
			opts.Services = strings.Split(services, ",")
			for i, s := range opts.Services {
				opts.Services[i] = strings.TrimSpace(s)
			}
		} else {
			opts.Services = []string{}
		}
		
		fmt.Println("\nSending notification...")
		if err := runNotifyCommand(cmd, []string{}); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		
		fmt.Println()
		if promptForInput("Send another? (y/N): ") != "y" {
			break
		}
		fmt.Println()
	}
	
	fmt.Println("Goodbye!")
	return nil
}

func runVersionCommand(cmd *cobra.Command, args []string) error {
	versionInfo := apprise.GetVersionInfo()
	
	if opts.JSON {
		jsonData, err := json.MarshalIndent(versionInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}
	
	fmt.Println(versionInfo.String())
	return nil
}

func countEnabledServices() int {
	count := 0
	for _, service := range config.Services {
		if service.Enabled {
			count++
		}
	}
	return count
}