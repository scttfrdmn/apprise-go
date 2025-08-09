package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

// CLIOptions holds command line options
type CLIOptions struct {
	Title         string
	Body          string
	NotifyType    string
	ConfigPaths   []string
	URLs          []string
	Tags          []string
	Attachments   []string
	BodyFormat    string
	Verbose       int
	DryRun        bool
	Timeout       time.Duration
	Version       bool
	Help          bool
}

const (
	AppName = "apprise-cli"
)

func main() {
	opts := parseFlags()

	if opts.Help {
		printUsage()
		os.Exit(0)
	}

	if opts.Version {
		versionInfo := apprise.GetVersionInfo()
		fmt.Printf("%s %s\n", AppName, versionInfo.String())
		os.Exit(0)
	}

	// Create Apprise instance
	app := apprise.New()
	app.SetTimeout(opts.Timeout)

	// Load configurations if specified
	if len(opts.ConfigPaths) > 0 {
		if err := loadConfigurations(app, opts.ConfigPaths, opts.Verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Try to load default configurations
		config := apprise.NewAppriseConfig(app)
		if err := config.LoadDefaultConfigs(); err != nil {
			if opts.Verbose > 0 {
				fmt.Printf("Warning: %v\n", err)
			}
		}
		config.ApplyToApprise()
	}

	// Add URLs from command line
	for _, url := range opts.URLs {
		if err := app.Add(url, opts.Tags...); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding URL %s: %v\n", url, err)
			os.Exit(1)
		}
	}

	// Check if we have any services configured
	if app.Count() == 0 {
		fmt.Fprintf(os.Stderr, "Error: No notification services configured.\n")
		fmt.Fprintf(os.Stderr, "Use --config to specify a configuration file or provide URLs directly.\n")
		os.Exit(1)
	}

	// Get notification body
	body := opts.Body
	if body == "" {
		// Read from stdin if no body specified
		var err error
		body, err = readFromStdin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
	}

	if body == "" {
		fmt.Fprintf(os.Stderr, "Error: No notification body specified.\n")
		os.Exit(1)
	}

	// Parse notification type
	notifyType := parseNotifyType(opts.NotifyType)

	// Create attachments
	var attachments []apprise.Attachment
	for _, attachPath := range opts.Attachments {
		attachment := apprise.Attachment{}
		if strings.HasPrefix(attachPath, "http://") || strings.HasPrefix(attachPath, "https://") {
			attachment.URL = attachPath
		} else {
			attachment.LocalPath = attachPath
			attachment.Name = filepath.Base(attachPath)
		}
		attachments = append(attachments, attachment)
	}

	if opts.Verbose > 0 {
		fmt.Printf("Sending notification to %d service(s)...\n", app.Count())
		if opts.Title != "" {
			fmt.Printf("Title: %s\n", opts.Title)
		}
		fmt.Printf("Body: %s\n", truncateString(body, 100))
		if len(attachments) > 0 {
			fmt.Printf("Attachments: %d\n", len(attachments))
		}
	}

	// Dry run mode
	if opts.DryRun {
		fmt.Println("DRY RUN: Notification would be sent to the following services:")
		// Here you would list the configured services
		return
	}

	// Send notification
	options := []apprise.NotifyOption{
		apprise.WithTags(opts.Tags...),
		apprise.WithBodyFormat(opts.BodyFormat),
	}
	
	if len(attachments) > 0 {
		options = append(options, apprise.WithAttachments(attachments...))
	}

	responses := app.Notify(opts.Title, body, notifyType, options...)

	// Process results
	successCount := 0
	for i, response := range responses {
		if response.Success {
			successCount++
			if opts.Verbose > 0 {
				fmt.Printf("✓ Service %d: %s (%.2fs)\n", i+1, response.ServiceID, response.Duration.Seconds())
			}
		} else {
			fmt.Fprintf(os.Stderr, "✗ Service %d (%s): %v\n", i+1, response.ServiceID, response.Error)
		}
	}

	if opts.Verbose > 0 || successCount < len(responses) {
		fmt.Printf("Notification sent successfully to %d/%d services.\n", successCount, len(responses))
	}

	// Exit with error code if any notifications failed
	if successCount < len(responses) {
		os.Exit(1)
	}
}

// parseFlags parses command line flags and returns CLIOptions
func parseFlags() CLIOptions {
	opts := CLIOptions{}

	flag.StringVar(&opts.Title, "title", "", "Notification title")
	flag.StringVar(&opts.Title, "t", "", "Notification title (short)")
	flag.StringVar(&opts.Body, "body", "", "Notification body")
	flag.StringVar(&opts.Body, "b", "", "Notification body (short)")
	flag.StringVar(&opts.NotifyType, "type", "info", "Notification type (info, success, warning, error)")
	flag.StringVar(&opts.NotifyType, "n", "info", "Notification type (short)")
	flag.StringVar(&opts.BodyFormat, "format", "text", "Body format (text, html, markdown)")
	flag.DurationVar(&opts.Timeout, "timeout", 30*time.Second, "Timeout for notifications")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "Show what would be sent without actually sending")
	flag.BoolVar(&opts.Version, "version", false, "Show version information")
	flag.BoolVar(&opts.Help, "help", false, "Show help information")
	flag.BoolVar(&opts.Help, "h", false, "Show help information (short)")

	// Custom flag parsing for repeated flags
	configPaths := flag.String("config", "", "Configuration file path(s), comma-separated")
	configPathsShort := flag.String("c", "", "Configuration file path(s), comma-separated (short)")
	urls := flag.String("url", "", "Notification service URL(s), comma-separated")
	tags := flag.String("tag", "", "Tag(s) for filtering notifications, comma-separated")
	attachments := flag.String("attach", "", "Attachment file path(s) or URL(s), comma-separated")
	verbose := flag.Int("verbose", 0, "Verbosity level (0-2)")
	verboseShort := flag.Int("v", 0, "Verbosity level (0-2, use -vv for level 2)")

	flag.Parse()

	// Handle verbosity with -vv syntax
	if *verboseShort > *verbose {
		opts.Verbose = *verboseShort
	} else {
		opts.Verbose = *verbose
	}

	// Parse comma-separated values
	if *configPaths != "" {
		opts.ConfigPaths = strings.Split(*configPaths, ",")
	}
	if *configPathsShort != "" && *configPaths == "" {
		opts.ConfigPaths = strings.Split(*configPathsShort, ",")
	}
	if *urls != "" {
		opts.URLs = strings.Split(*urls, ",")
	}
	if *tags != "" {
		opts.Tags = strings.Split(*tags, ",")
	}
	if *attachments != "" {
		opts.Attachments = strings.Split(*attachments, ",")
	}

	// Add remaining arguments as URLs
	for _, arg := range flag.Args() {
		if strings.Contains(arg, "://") {
			opts.URLs = append(opts.URLs, arg)
		}
	}

	return opts
}

// loadConfigurations loads configuration files
func loadConfigurations(app *apprise.Apprise, configPaths []string, verbose int) error {
	config := apprise.NewAppriseConfig(app)

	for _, path := range configPaths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		if verbose > 0 {
			fmt.Printf("Loading configuration from: %s\n", path)
		}

		var err error
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			err = config.AddFromURL(path)
		} else {
			err = config.AddFromFile(path)
		}

		if err != nil {
			return err
		}
	}

	return config.ApplyToApprise()
}

// readFromStdin reads the notification body from stdin
func readFromStdin() (string, error) {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

// parseNotifyType converts string to NotifyType
func parseNotifyType(typeStr string) apprise.NotifyType {
	switch strings.ToLower(typeStr) {
	case "success":
		return apprise.NotifyTypeSuccess
	case "warning", "warn":
		return apprise.NotifyTypeWarning
	case "error", "err":
		return apprise.NotifyTypeError
	case "info":
		fallthrough
	default:
		return apprise.NotifyTypeInfo
	}
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// printUsage prints usage information
func printUsage() {
	fmt.Printf(`%s - Universal notification library for Go

USAGE:
    %s [OPTIONS] [URL...]

OPTIONS:
    -t, --title TITLE           Notification title
    -b, --body BODY            Notification body (reads from stdin if not provided)
    -n, --type TYPE            Notification type: info, success, warning, error (default: info)
    -c, --config PATH          Configuration file path (can be used multiple times)
        --format FORMAT        Body format: text, html, markdown (default: text)
        --tag TAG              Tag for filtering notifications (can be used multiple times)
        --attach PATH/URL      Attachment file path or URL (can be used multiple times)
        --timeout DURATION     Timeout for notifications (default: 30s)
        --dry-run             Show what would be sent without sending
    -v, --verbose             Increase verbosity (-v or -vv)
        --version             Show version information
    -h, --help                Show this help message

EXAMPLES:
    # Send a simple notification
    %s -t "Hello" -b "World" discord://webhook_id/webhook_token

    # Send from stdin with config file
    echo "Server is down!" | %s -t "Alert" -c config.yaml

    # Send with attachments
    %s -t "Report" -b "See attached" --attach report.pdf mailto://user:pass@gmail.com

    # Use multiple services with tags
    %s -t "Deploy" -b "Success" --tag production

For more information and examples, visit: https://github.com/yourusername/go-apprise

`, AppName, AppName, AppName, AppName, AppName, AppName)
}