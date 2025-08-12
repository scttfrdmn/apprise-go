package apprise

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppriseConfigYAML(t *testing.T) {
	// Create a temporary YAML config file
	yamlContent := `
version: 1
urls:
  - url: discord://webhook_id/webhook_token
    tag:
      - team
      - alerts
  - url: slack://TokenA/TokenB/TokenC/general
    tag:
      - team
  - url: mailto://user:pass@smtp.gmail.com/admin@company.com
    tag:
      - admin
`

	tmpFile, err := os.CreateTemp("", "apprise_test_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	// Test loading the config
	app := New()
	config := NewAppriseConfig(app)

	err = config.AddFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	if len(config.configs) != 1 {
		t.Errorf("Expected 1 config loaded, got %d", len(config.configs))
	}

	if len(config.configs[0].URLs) != 3 {
		t.Errorf("Expected 3 URLs in config, got %d", len(config.configs[0].URLs))
	}

	// Test applying config to Apprise
	err = config.ApplyToApprise()
	if err != nil {
		t.Fatalf("Failed to apply config to Apprise: %v", err)
	}

	if app.Count() != 3 {
		t.Errorf("Expected 3 services after applying config, got %d", app.Count())
	}
}

func TestAppriseConfigText(t *testing.T) {
	// Create a temporary text config file
	textContent := `
# Team notifications
discord://webhook_id/webhook_token [team,alerts]

# Slack channel
slack://TokenA/TokenB/TokenC/general [team]

# Admin email
mailto://user:pass@smtp.gmail.com/admin@company.com [admin]

# Simple webhook without tags
webhook://api.example.com/notify
`

	tmpFile, err := os.CreateTemp("", "apprise_test_*.conf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(textContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	// Test loading the config
	app := New()
	config := NewAppriseConfig(app)

	err = config.AddFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load text config: %v", err)
	}

	if len(config.configs) != 1 {
		t.Errorf("Expected 1 config loaded, got %d", len(config.configs))
	}

	if len(config.configs[0].URLs) != 4 {
		t.Errorf("Expected 4 URLs in config, got %d", len(config.configs[0].URLs))
	}

	// Check tags were parsed correctly
	discordConfig := config.configs[0].URLs[0]
	if len(discordConfig.Tags) != 2 {
		t.Errorf("Expected 2 tags for Discord, got %d", len(discordConfig.Tags))
	}

	if !contains(discordConfig.Tags, "team") || !contains(discordConfig.Tags, "alerts") {
		t.Errorf("Discord tags not parsed correctly: %v", discordConfig.Tags)
	}
}

func TestAppriseConfigInvalidFile(t *testing.T) {
	app := New()
	config := NewAppriseConfig(app)

	// Test non-existent file
	err := config.AddFromFile("/non/existent/file.yaml")
	if err == nil {
		t.Error("Should have failed to load non-existent file")
	}

	// Test invalid YAML
	invalidYAML := `
version: 1
urls:
  - url: discord://webhook_id/webhook_token
    tag:
      - team
      - alerts
    invalid_field: [unclosed array
`

	tmpFile, err := os.CreateTemp("", "apprise_invalid_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(invalidYAML); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	err = config.AddFromFile(tmpFile.Name())
	if err == nil {
		t.Error("Should have failed to parse invalid YAML")
	}
}

func TestAppriseConfigDefaultPaths(t *testing.T) {
	// Test getting default config paths
	paths := getDefaultConfigPaths()

	if len(paths) == 0 {
		t.Error("Should return at least some default config paths")
	}

	// Verify paths are absolute
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			t.Errorf("Config path should be absolute: %s", path)
		}
	}
}

func TestTextConfigLineParsing(t *testing.T) {
	config := NewAppriseConfig(New())

	testCases := []struct {
		line         string
		expectedURL  string
		expectedTags []string
	}{
		{
			line:         "discord://webhook_id/webhook_token",
			expectedURL:  "discord://webhook_id/webhook_token",
			expectedTags: nil,
		},
		{
			line:         "slack://TokenA/TokenB/TokenC [team,alerts]",
			expectedURL:  "slack://TokenA/TokenB/TokenC",
			expectedTags: []string{"team", "alerts"},
		},
		{
			line:         "mailto://user@domain.com [admin,   critical   ]",
			expectedURL:  "mailto://user@domain.com",
			expectedTags: []string{"admin", "critical"},
		},
		{
			line:         "webhook://api.example.com/notify []",
			expectedURL:  "webhook://api.example.com/notify",
			expectedTags: nil,
		},
	}

	for _, tc := range testCases {
		urlConfig := config.parseTextLine(tc.line)

		if urlConfig.URL != tc.expectedURL {
			t.Errorf("For line %q, expected URL %q, got %q",
				tc.line, tc.expectedURL, urlConfig.URL)
		}

		if !stringSlicesEqual(urlConfig.Tags, tc.expectedTags) {
			t.Errorf("For line %q, expected tags %v, got %v",
				tc.line, tc.expectedTags, urlConfig.Tags)
		}
	}
}

func TestYAMLFormatDetection(t *testing.T) {
	config := NewAppriseConfig(New())

	yamlContent := `
version: 1
urls:
  - url: discord://webhook_id/webhook_token
`

	textContent := `
discord://webhook_id/webhook_token
slack://TokenA/TokenB/TokenC
`

	if !config.isYAMLFormat(yamlContent) {
		t.Error("Should detect YAML format")
	}

	if config.isYAMLFormat(textContent) {
		t.Error("Should not detect text as YAML format")
	}

	// Edge case: empty content
	if config.isYAMLFormat("") {
		t.Error("Empty content should not be detected as YAML")
	}

	// Edge case: comments only
	commentsOnly := `
# This is just comments
# No actual content
`
	if config.isYAMLFormat(commentsOnly) {
		t.Error("Comments-only content should not be detected as YAML")
	}
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return len(a) == 0 && len(b) == 0
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
