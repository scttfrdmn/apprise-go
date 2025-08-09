package main

import (
	"bytes"
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

func TestParseNotifyType(t *testing.T) {
	testCases := []struct {
		input    string
		expected apprise.NotifyType
	}{
		{"success", apprise.NotifyTypeSuccess},
		{"warning", apprise.NotifyTypeWarning},
		{"warn", apprise.NotifyTypeWarning},
		{"error", apprise.NotifyTypeError},
		{"err", apprise.NotifyTypeError},
		{"info", apprise.NotifyTypeInfo},
		{"INFO", apprise.NotifyTypeInfo},
		{"unknown", apprise.NotifyTypeInfo}, // defaults to info
		{"", apprise.NotifyTypeInfo},        // defaults to info
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := parseNotifyType(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	testCases := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten!", 10, "exactly te..."},
		{"", 5, ""},
		{"a", 5, "a"},
		{"this is a very long string", 10, "this is a ..."},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := truncateString(tc.input, tc.maxLen)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestParseFlagsBasic(t *testing.T) {
	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Simulate command line arguments
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"apprise", "-title", "Test Title", "-body", "Test Body", "-type", "success"}

	opts := parseFlags()

	if opts.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", opts.Title)
	}
	if opts.Body != "Test Body" {
		t.Errorf("Expected body 'Test Body', got '%s'", opts.Body)
	}
	if opts.NotifyType != "success" {
		t.Errorf("Expected type 'success', got '%s'", opts.NotifyType)
	}
}

func TestParseFlagsShortOptions(t *testing.T) {
	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"apprise", "-t", "Short Title", "-b", "Short Body", "-n", "error"}

	opts := parseFlags()

	if opts.Title != "Short Title" {
		t.Errorf("Expected title 'Short Title', got '%s'", opts.Title)
	}
	if opts.Body != "Short Body" {
		t.Errorf("Expected body 'Short Body', got '%s'", opts.Body)
	}
	if opts.NotifyType != "error" {
		t.Errorf("Expected type 'error', got '%s'", opts.NotifyType)
	}
}

func TestParseFlagsCommaSeparatedValues(t *testing.T) {
	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"apprise",
		"-config", "config1.yaml,config2.yaml",
		"-tag", "production,alerts",
		"-url", "discord://token1,slack://token2",
		"-attach", "file1.txt,file2.pdf",
	}

	opts := parseFlags()

	expectedConfigs := []string{"config1.yaml", "config2.yaml"}
	if !reflect.DeepEqual(opts.ConfigPaths, expectedConfigs) {
		t.Errorf("Expected configs %v, got %v", expectedConfigs, opts.ConfigPaths)
	}

	expectedTags := []string{"production", "alerts"}
	if !reflect.DeepEqual(opts.Tags, expectedTags) {
		t.Errorf("Expected tags %v, got %v", expectedTags, opts.Tags)
	}

	expectedURLs := []string{"discord://token1", "slack://token2"}
	if !reflect.DeepEqual(opts.URLs, expectedURLs) {
		t.Errorf("Expected URLs %v, got %v", expectedURLs, opts.URLs)
	}

	expectedAttachments := []string{"file1.txt", "file2.pdf"}
	if !reflect.DeepEqual(opts.Attachments, expectedAttachments) {
		t.Errorf("Expected attachments %v, got %v", expectedAttachments, opts.Attachments)
	}
}

func TestParseFlagsPositionalURLs(t *testing.T) {
	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"apprise", "-title", "Test", "discord://webhook_id/webhook_token", "slack://token/channel"}

	opts := parseFlags()

	expectedURLs := []string{"discord://webhook_id/webhook_token", "slack://token/channel"}
	if !reflect.DeepEqual(opts.URLs, expectedURLs) {
		t.Errorf("Expected URLs %v, got %v", expectedURLs, opts.URLs)
	}
}

func TestParseFlagsVerbosity(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected int
	}{
		{"default", []string{"apprise"}, 0},
		{"verbose", []string{"apprise", "-verbose", "1"}, 1},
		{"short verbose", []string{"apprise", "-v", "2"}, 2},
		{"both verbose (short wins)", []string{"apprise", "-v", "2", "-verbose", "1"}, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset flag package state
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = tc.args
			opts := parseFlags()

			if opts.Verbose != tc.expected {
				t.Errorf("Expected verbose %d, got %d", tc.expected, opts.Verbose)
			}
		})
	}
}

func TestParseFlagsBooleanOptions(t *testing.T) {
	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"apprise", "-dry-run", "-version", "-help"}

	opts := parseFlags()

	if !opts.DryRun {
		t.Error("Expected DryRun to be true")
	}
	if !opts.Version {
		t.Error("Expected Version to be true")
	}
	if !opts.Help {
		t.Error("Expected Help to be true")
	}
}

func TestParseFlagsTimeout(t *testing.T) {
	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"apprise", "-timeout", "45s"}

	opts := parseFlags()

	expected := 45 * time.Second
	if opts.Timeout != expected {
		t.Errorf("Expected timeout %v, got %v", expected, opts.Timeout)
	}
}

func TestReadFromStdinEmpty(t *testing.T) {
	// Create a pipe to simulate empty stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, _ := os.Pipe()
	os.Stdin = r
	w.Close() // Close immediately to simulate empty input

	result, err := readFromStdin()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestReadFromStdinWithContent(t *testing.T) {
	// Create a pipe to simulate stdin with content
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write test content
	go func() {
		defer w.Close()
		w.Write([]byte("Line 1\nLine 2\nLine 3"))
	}()

	result, err := readFromStdin()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "Line 1\nLine 2\nLine 3"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestLoadConfigurationsFileError(t *testing.T) {
	app := apprise.New()

	// Try to load non-existent file
	err := loadConfigurations(app, []string{"/non/existent/config.yaml"}, 0)
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}
}

func TestLoadConfigurationsEmptyPath(t *testing.T) {
	app := apprise.New()

	// Load with empty path (should be skipped)
	err := loadConfigurations(app, []string{""}, 0)
	if err != nil {
		t.Errorf("Unexpected error with empty path: %v", err)
	}
}

func TestLoadConfigurationsMultiplePaths(t *testing.T) {
	app := apprise.New()

	// Create temporary config files
	tmpDir := t.TempDir()
	config1Path := tmpDir + "/config1.yaml"
	config2Path := tmpDir + "/config2.yaml"

	// Write basic YAML configs
	config1Content := `version: 1
urls:
  - url: discord://webhook_id1/webhook_token1
`
	config2Content := `version: 1
urls:
  - url: slack://TokenA/TokenB/TokenC/general
`

	if err := os.WriteFile(config1Path, []byte(config1Content), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	if err := os.WriteFile(config2Path, []byte(config2Content), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test loading multiple configs
	err := loadConfigurations(app, []string{config1Path, config2Path, ""}, 1) // Include empty path and verbose
	if err != nil {
		t.Errorf("Unexpected error loading configs: %v", err)
	}

	// Should have 2 services loaded
	if app.Count() != 2 {
		t.Errorf("Expected 2 services, got %d", app.Count())
	}
}

// Test the main usage printing function
func TestPrintUsage(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printUsage()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check that usage contains expected elements
	expectedElements := []string{
		"apprise-cli",
		"USAGE:",
		"OPTIONS:",
		"EXAMPLES:",
		"--title",
		"--body",
		"--type",
		"--config",
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("Usage output missing expected element: %s", element)
		}
	}
}
