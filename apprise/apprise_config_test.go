package apprise

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppriseConfig_BasicFunctionality(t *testing.T) {
	tempDir := t.TempDir()
	
	config := NewAppriseConfig(tempDir, "test")
	
	if config.environment != "test" {
		t.Errorf("Expected environment 'test', got %s", config.environment)
	}
	
	if config.configManager == nil {
		t.Error("Config manager should not be nil")
	}
	
	if config.template == nil {
		t.Error("Template should not be nil")
	}
}

func TestAppriseConfig_LoadConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test environment file
	envFile := filepath.Join(tempDir, ".env.test")
	envContent := `TEST_SERVICE=test-service
TEST_PORT=9090`
	
	err := os.WriteFile(envFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create env file: %v", err)
	}
	
	// Change to temp directory for environment loading
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)
	
	config := NewAppriseConfig(tempDir, "test")
	
	err = config.LoadConfiguration()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Check if environment variables were loaded
	if os.Getenv("TEST_SERVICE") != "test-service" {
		t.Errorf("Expected TEST_SERVICE=test-service, got %s", os.Getenv("TEST_SERVICE"))
	}
	
	// Clean up
	os.Unsetenv("TEST_SERVICE")
	os.Unsetenv("TEST_PORT")
}

func TestAppriseConfig_GenerateAppriseConfig(t *testing.T) {
	tempDir := t.TempDir()
	
	// Set up test environment variables
	os.Setenv("SLACK_WEBHOOK_URL", "https://hooks.slack.com/services/T123/B456/xyz789")
	os.Setenv("ADMIN_EMAIL", "admin@test.com")
	defer func() {
		os.Unsetenv("SLACK_WEBHOOK_URL")
		os.Unsetenv("ADMIN_EMAIL")
	}()
	
	config := NewAppriseConfig(tempDir, "test")
	
	// Create a test template
	templateDir := filepath.Join(tempDir, "templates")
	os.MkdirAll(templateDir, 0755)
	
	templateFile := filepath.Join(templateDir, "test.tmpl")
	templateContent := `# Test Apprise Configuration
{{if env "SLACK_WEBHOOK_URL"}}slack://{{env "SLACK_WEBHOOK_URL" | replace "https://hooks.slack.com/services/" ""}} #slack,test{{end}}
{{if env "ADMIN_EMAIL"}}mailto://smtp.test.com/{{env "ADMIN_EMAIL"}} #email,admin{{end}}
desktop:// #desktop,dev`
	
	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}
	
	// Load configuration and generate Apprise instance
	err = config.LoadConfiguration()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}
	
	apprise, err := config.GenerateAppriseConfig("test")
	if err != nil {
		t.Fatalf("Failed to generate Apprise config: %v", err)
	}
	
	// Check that services were added
	if apprise.Count() != 3 {
		t.Errorf("Expected 3 services, got %d", apprise.Count())
	}
}

func TestAppriseConfig_ServiceHelperFunctions(t *testing.T) {
	config := NewAppriseConfig(t.TempDir(), "test")
	
	// Test buildServiceURL function
	url := config.buildServiceURL("slack", "hooks.slack.com/services/T123/B456/xyz", map[string]string{
		"format": "markdown",
		"avatar": "AppBot",
	})
	
	// Note: map iteration order may vary, so check components
	if !strings.HasPrefix(url, "slack://hooks.slack.com/services/T123/B456/xyz?") {
		t.Errorf("Expected URL to start with service and host, got: %s", url)
	}
	if !strings.Contains(url, "format=markdown") {
		t.Errorf("Expected URL to contain format parameter, got: %s", url)
	}
	if !strings.Contains(url, "avatar=AppBot") {
		t.Errorf("Expected URL to contain avatar parameter, got: %s", url)
	}
	
	// Test service filtering functions
	services := []AppriseConfigService{
		{Name: "slack1", Type: "slack", Enabled: true, Tags: []string{"team", "alerts"}, Priority: "high"},
		{Name: "slack2", Type: "slack", Enabled: false, Tags: []string{"team"}, Priority: "normal"},
		{Name: "email1", Type: "email", Enabled: true, Tags: []string{"admin", "alerts"}, Priority: "high"},
		{Name: "sms1", Type: "sms", Enabled: true, Tags: []string{"emergency"}, Priority: "critical"},
	}
	
	// Test enabled services
	enabled := config.getEnabledServices(services)
	if len(enabled) != 3 {
		t.Errorf("Expected 3 enabled services, got %d", len(enabled))
	}
	
	// Test services by tag
	alertServices := config.getServicesByTag(services, "alerts")
	if len(alertServices) != 2 {
		t.Errorf("Expected 2 services with 'alerts' tag, got %d", len(alertServices))
	}
	
	// Test services by priority
	highPriorityServices := config.getServicesByPriority(services, "high")
	if len(highPriorityServices) != 2 {
		t.Errorf("Expected 2 high priority services, got %d", len(highPriorityServices))
	}
}

func TestAppriseConfig_CreateDefaultTemplates(t *testing.T) {
	tempDir := t.TempDir()
	
	config := NewAppriseConfig(tempDir, "development")
	
	err := config.CreateDefaultTemplates()
	if err != nil {
		t.Fatalf("Failed to create default templates: %v", err)
	}
	
	// Check that template files were created
	templateDir := filepath.Join(tempDir, "templates")
	
	expectedFiles := []string{
		"services.tmpl",
		"environment.tmpl",
		"complete.tmpl",
	}
	
	for _, filename := range expectedFiles {
		filepath := filepath.Join(templateDir, filename)
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			t.Errorf("Expected template file %s was not created", filename)
		}
		
		// Check that file has content
		content, err := os.ReadFile(filepath)
		if err != nil {
			t.Errorf("Failed to read template file %s: %v", filename, err)
		}
		if len(content) == 0 {
			t.Errorf("Template file %s should not be empty", filename)
		}
	}
}

func TestAppriseConfig_CreateSampleEnvironmentFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	config := NewAppriseConfig(tempDir, "development")
	
	err := config.CreateSampleEnvironmentFiles()
	if err != nil {
		t.Fatalf("Failed to create sample environment files: %v", err)
	}
	
	// Check that environment files were created
	expectedFiles := []string{
		".env.sample",
		".env.production",
	}
	
	for _, filename := range expectedFiles {
		filepath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			t.Errorf("Expected environment file %s was not created", filename)
		}
		
		// Check that file has content
		content, err := os.ReadFile(filepath)
		if err != nil {
			t.Errorf("Failed to read environment file %s: %v", filename, err)
		}
		if len(content) == 0 {
			t.Errorf("Environment file %s should not be empty", filename)
		}
		
		// Check for expected environment variables
		contentStr := string(content)
		expectedVars := []string{"SERVICE_NAME", "ENVIRONMENT", "SLACK_WEBHOOK_URL"}
		for _, envVar := range expectedVars {
			if !strings.Contains(contentStr, envVar) {
				t.Errorf("Environment file %s should contain %s", filename, envVar)
			}
		}
	}
}

func TestAppriseConfig_ParseConfiguration(t *testing.T) {
	config := NewAppriseConfig(t.TempDir(), "test")
	apprise := New()
	
	configStr := `# Test Apprise Configuration
# This is a comment line

# Slack service
slack://T123456/B789012/abcdef123456 #slack,team

# Email service  
mailto://smtp.test.com/admin@test.com #email,admin

# Desktop notification
desktop:// #desktop,dev

# Empty line above should be ignored`
	
	err := config.parseConfiguration(apprise, configStr)
	if err != nil {
		t.Fatalf("Failed to parse configuration: %v", err)
	}
	
	// Check that services were added (should be 3)
	if apprise.Count() != 3 {
		t.Errorf("Expected 3 services to be configured, got %d", apprise.Count())
	}
}

func TestLoadFromTemplate(t *testing.T) {
	tempDir := t.TempDir()
	
	// Set up test environment
	os.Setenv("TEST_SLACK_URL", "slack://T123/B456/xyz")
	defer os.Unsetenv("TEST_SLACK_URL")
	
	// Create template
	templateDir := filepath.Join(tempDir, "templates")
	os.MkdirAll(templateDir, 0755)
	
	templateFile := filepath.Join(templateDir, "integration.tmpl")
	templateContent := `{{if env "TEST_SLACK_URL"}}{{env "TEST_SLACK_URL"}} #slack,test{{end}}
desktop:// #desktop,test`
	
	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}
	
	// Load from template
	apprise, err := LoadFromTemplate(tempDir, "test", "integration")
	if err != nil {
		t.Fatalf("Failed to load from template: %v", err)
	}
	
	if apprise.Count() != 2 {
		t.Errorf("Expected 2 services from template, got %d", apprise.Count())
	}
}

func TestSetupDefaultConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	
	err := SetupDefaultConfiguration(tempDir)
	if err != nil {
		t.Fatalf("Failed to setup default configuration: %v", err)
	}
	
	// Check that templates directory exists with files
	templateDir := filepath.Join(tempDir, "templates")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Error("Templates directory should have been created")
	}
	
	// Check that sample environment files exist
	sampleEnvFile := filepath.Join(tempDir, ".env.sample")
	if _, err := os.Stat(sampleEnvFile); os.IsNotExist(err) {
		t.Error("Sample environment file should have been created")
	}
	
	prodEnvFile := filepath.Join(tempDir, ".env.production")
	if _, err := os.Stat(prodEnvFile); os.IsNotExist(err) {
		t.Error("Production environment file should have been created")
	}
}

func TestAppriseConfig_CompleteIntegration(t *testing.T) {
	tempDir := t.TempDir()
	
	// Set up realistic test environment
	testEnvs := map[string]string{
		"ENVIRONMENT":         "test",
		"SERVICE_NAME":        "TestApp",
		"TEAM_NAME":          "TestTeam",
		"SLACK_WEBHOOK_URL":  "https://hooks.slack.com/services/T123/B456/xyz789",
		"ADMIN_EMAIL":        "admin@testapp.com",
		"SMTP_SERVER":        "smtp.testapp.com",
		"SMTP_USER":          "notifications@testapp.com",
		"SMTP_PASS":          "testpassword",
		"MOBILE_PUSH_TOKENS": "token1,token2,token3",
		"CUSTOM_WEBHOOK_URL": "https://webhook.testapp.com/notify",
	}
	
	for key, value := range testEnvs {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}
	
	// Setup default configuration
	err := SetupDefaultConfiguration(tempDir)
	if err != nil {
		t.Fatalf("Failed to setup default configuration: %v", err)
	}
	
	// Load from complete template
	apprise, err := LoadFromTemplate(tempDir, "test", "complete")
	if err != nil {
		t.Fatalf("Failed to load complete template: %v", err)
	}
	
	// Should have loaded multiple services based on environment variables
	if apprise.Count() == 0 {
		t.Error("Expected services to be loaded from complete template")
	}
	
	// Test that we can send a notification
	responses := apprise.Notify("Test Integration", "This is a test notification from the integration test", NotifyTypeInfo)
	
	// We expect some responses (even if they fail due to invalid URLs)
	if len(responses) == 0 {
		t.Error("Expected notification responses")
	}
	
	// Check that at least some services were configured
	successCount := 0
	for _, response := range responses {
		if response.Error == nil || strings.Contains(response.Error.Error(), "connection") ||
		   strings.Contains(response.Error.Error(), "invalid") {
			// Count successful attempts or expected connection failures
			successCount++
		}
	}
	
	if successCount == 0 {
		t.Error("Expected at least some services to be properly configured")
	}
}

func TestAppriseConfig_EnvironmentSpecificTemplates(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test different environments
	environments := []string{"development", "staging", "production"}
	
	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			config := NewAppriseConfig(tempDir, env)
			
			err := config.CreateDefaultTemplates()
			if err != nil {
				t.Fatalf("Failed to create templates for %s: %v", env, err)
			}
			
			err = config.LoadConfiguration()
			if err != nil {
				t.Fatalf("Failed to load configuration for %s: %v", env, err)
			}
			
			// Try to load environment template
			template, exists := config.configManager.GetTemplate("environment")
			if !exists {
				t.Errorf("Environment template should exist for %s", env)
				return
			}
			
			// Execute template to see if it works
			result, err := template.ExecuteToString()
			if err != nil {
				t.Errorf("Failed to execute environment template for %s: %v", env, err)
				return
			}
			
			// Check that environment-specific content is generated
			if !strings.Contains(result, env) {
				t.Errorf("Template result should contain environment %s", env)
			}
		})
	}
}

func TestAppriseConfig_TemplateValidation(t *testing.T) {
	tempDir := t.TempDir()
	
	config := NewAppriseConfig(tempDir, "test")
	
	// Create a template with invalid syntax
	templateDir := filepath.Join(tempDir, "templates")
	os.MkdirAll(templateDir, 0755)
	
	invalidTemplate := filepath.Join(templateDir, "invalid.tmpl")
	invalidContent := `{{if env "TEST_VAR"} # Missing closing braces
This should cause a parse error`
	
	err := os.WriteFile(invalidTemplate, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid template: %v", err)
	}
	
	// Try to load templates - should fail
	err = config.LoadConfiguration()
	if err == nil {
		t.Error("Expected error when loading invalid template")
	}
	
	if !strings.Contains(err.Error(), "template") {
		t.Errorf("Expected template-related error, got: %v", err)
	}
}