package apprise

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigTemplate_BasicFunctionality(t *testing.T) {
	template := NewConfigTemplate()
	
	// Test basic template loading and execution
	tmplContent := `Hello {{.Env.USER | default "anonymous"}}!`
	err := template.LoadTemplate("test", tmplContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	if !strings.Contains(result, "Hello") {
		t.Errorf("Expected result to contain 'Hello', got: %s", result)
	}
}

func TestConfigTemplate_EnvironmentVariables(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_CONFIG_VAR", "test_value")
	defer os.Unsetenv("TEST_CONFIG_VAR")
	
	template := NewConfigTemplate()
	tmplContent := `Value: {{env "TEST_CONFIG_VAR" "default"}}`
	
	err := template.LoadTemplate("env_test", tmplContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	expected := "Value: test_value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConfigTemplate_DefaultValues(t *testing.T) {
	template := NewConfigTemplate()
	template.SetDefault("DEFAULT_KEY", "default_value")
	
	tmplContent := `Default: {{default "DEFAULT_KEY" "fallback"}}`
	
	err := template.LoadTemplate("default_test", tmplContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	expected := "Default: default_value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConfigTemplate_Variables(t *testing.T) {
	template := NewConfigTemplate()
	template.SetVariable("service_name", "test-service")
	template.SetVariable("port", 8080)
	template.SetVariable("enabled", true)
	
	tmplContent := `Service: {{var "service_name"}}
Port: {{var "port"}}
Enabled: {{var "enabled"}}`
	
	err := template.LoadTemplate("var_test", tmplContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	if !strings.Contains(result, "Service: test-service") {
		t.Errorf("Expected result to contain service name, got: %s", result)
	}
	if !strings.Contains(result, "Port: 8080") {
		t.Errorf("Expected result to contain port, got: %s", result)
	}
	if !strings.Contains(result, "Enabled: true") {
		t.Errorf("Expected result to contain enabled flag, got: %s", result)
	}
}

func TestConfigTemplate_StringFunctions(t *testing.T) {
	template := NewConfigTemplate()
	template.SetVariable("service_name", "Test Service")
	
	tmplContent := `Upper: {{var "service_name" | upper}}
Lower: {{var "service_name" | lower}}
Title: {{var "service_name" | title}}`
	
	err := template.LoadTemplate("string_test", tmplContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	if !strings.Contains(result, "Upper: TEST SERVICE") {
		t.Errorf("Expected uppercase transformation, got: %s", result)
	}
	if !strings.Contains(result, "Lower: test service") {
		t.Errorf("Expected lowercase transformation, got: %s", result)
	}
}

func TestConfigTemplate_ConditionalFunctions(t *testing.T) {
	template := NewConfigTemplate()
	template.SetVariable("debug", true)
	template.SetVariable("env", "production")
	
	tmplContent := `Debug Mode: {{if (var "debug")}}enabled{{else}}disabled{{end}}
Environment: {{if (eq (var "env") "production")}}prod{{else}}dev{{end}}`
	
	err := template.LoadTemplate("conditional_test", tmplContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	if !strings.Contains(result, "Debug Mode: enabled") {
		t.Errorf("Expected debug mode enabled, got: %s", result)
	}
}

func TestConfigTemplate_CustomFunctions(t *testing.T) {
	template := NewConfigTemplate()
	
	// Add custom function
	template.AddFunction("double", func(n int) int {
		return n * 2
	})
	
	template.SetVariable("number", 21)
	
	tmplContent := `Number: {{var "number"}}
Double: {{double (var "number")}}`
	
	err := template.LoadTemplate("custom_test", tmplContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	if !strings.Contains(result, "Number: 21") {
		t.Errorf("Expected original number, got: %s", result)
	}
	if !strings.Contains(result, "Double: 42") {
		t.Errorf("Expected doubled number, got: %s", result)
	}
}

func TestConfigTemplate_FileOperations(t *testing.T) {
	// Create temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello from file!"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	template := NewConfigTemplate()
	template.SetVariable("file_path", testFile)
	
	tmplContent := `File exists: {{fileExists (var "file_path")}}
File content: {{file (var "file_path")}}
Base name: {{basename (var "file_path")}}`
	
	err = template.LoadTemplate("file_test", tmplContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	if !strings.Contains(result, "File exists: true") {
		t.Errorf("Expected file exists to be true, got: %s", result)
	}
	if !strings.Contains(result, "File content: Hello from file!") {
		t.Errorf("Expected file content, got: %s", result)
	}
	if !strings.Contains(result, "Base name: test.txt") {
		t.Errorf("Expected base name, got: %s", result)
	}
}

func TestConfigTemplate_TimeAndUtilityFunctions(t *testing.T) {
	template := NewConfigTemplate()
	
	tmplContent := `Current time: {{formatTime "2006-01-02"}}
Unix timestamp: {{unix}}
Repeated: {{repeat 3 "Hi! "}}`
	
	err := template.LoadTemplate("time_test", tmplContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	currentDate := time.Now().Format("2006-01-02")
	if !strings.Contains(result, "Current time: "+currentDate) {
		t.Errorf("Expected current date, got: %s", result)
	}
	if !strings.Contains(result, "Repeated: Hi! Hi! Hi! ") {
		t.Errorf("Expected repeated string, got: %s", result)
	}
}

func TestConfigManager_BasicOperations(t *testing.T) {
	tempDir := t.TempDir()
	
	manager := NewConfigManager(tempDir)
	
	// Verify directories were created
	templateDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "generated")
	
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Errorf("Template directory should have been created")
	}
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("Output directory should have been created")
	}
	
	// Create a test template file
	templateFile := filepath.Join(templateDir, "service.tmpl")
	templateContent := `Service: {{env "SERVICE_NAME" "default-service"}}
Port: {{env "PORT" "8080"}}`
	
	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}
	
	// Load templates
	err = manager.LoadTemplates()
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}
	
	// Check if template was loaded
	template, exists := manager.GetTemplate("service")
	if !exists {
		t.Errorf("Template 'service' should have been loaded")
	}
	if template == nil {
		t.Errorf("Template should not be nil")
	}
}

func TestConfigManager_GenerateConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewConfigManager(tempDir)
	
	// Set test environment variables
	os.Setenv("SERVICE_NAME", "test-service")
	os.Setenv("PORT", "9090")
	defer func() {
		os.Unsetenv("SERVICE_NAME")
		os.Unsetenv("PORT")
	}()
	
	// Create template file
	templateDir := filepath.Join(tempDir, "templates")
	templateFile := filepath.Join(templateDir, "config.tmpl")
	templateContent := `[service]
name = {{env "SERVICE_NAME" "default"}}
port = {{env "PORT" "8080"}}
debug = {{env "DEBUG" "false"}}`
	
	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}
	
	// Load templates and generate config
	err = manager.LoadTemplates()
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}
	
	err = manager.GenerateConfig("config", "service.conf")
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}
	
	// Check generated file
	outputFile := filepath.Join(tempDir, "generated", "service.conf")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated config: %v", err)
	}
	
	contentStr := string(content)
	if !strings.Contains(contentStr, "name = test-service") {
		t.Errorf("Expected service name in config, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "port = 9090") {
		t.Errorf("Expected port in config, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "debug = false") {
		t.Errorf("Expected debug default value in config, got: %s", contentStr)
	}
}

func TestEnvironmentLoader_LoadEnvironment(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test environment files
	envFile := filepath.Join(tempDir, ".env")
	envContent := `# Main environment file
MAIN_VAR=main_value
SHARED_VAR=main_shared`
	
	err := os.WriteFile(envFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}
	
	envTestFile := filepath.Join(tempDir, ".env.test")
	envTestContent := `# Test environment file
TEST_VAR=test_value
SHARED_VAR=test_shared`
	
	err = os.WriteFile(envTestFile, []byte(envTestContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .env.test file: %v", err)
	}
	
	// Change to temp directory for relative path testing
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)
	
	loader := NewEnvironmentLoader("test")
	loader.SetOverride("OVERRIDE_VAR", "override_value")
	
	// Clear any existing environment variables
	testVars := []string{"MAIN_VAR", "TEST_VAR", "SHARED_VAR", "OVERRIDE_VAR"}
	for _, v := range testVars {
		os.Unsetenv(v)
	}
	
	err = loader.LoadEnvironment()
	if err != nil {
		t.Fatalf("Failed to load environment: %v", err)
	}
	
	// Check loaded variables
	if os.Getenv("MAIN_VAR") != "main_value" {
		t.Errorf("Expected MAIN_VAR=main_value, got %s", os.Getenv("MAIN_VAR"))
	}
	if os.Getenv("TEST_VAR") != "test_value" {
		t.Errorf("Expected TEST_VAR=test_value, got %s", os.Getenv("TEST_VAR"))
	}
	// Test environment file should override main
	if os.Getenv("SHARED_VAR") != "test_shared" {
		t.Errorf("Expected SHARED_VAR=test_shared, got %s", os.Getenv("SHARED_VAR"))
	}
	if os.Getenv("OVERRIDE_VAR") != "override_value" {
		t.Errorf("Expected OVERRIDE_VAR=override_value, got %s", os.Getenv("OVERRIDE_VAR"))
	}
	
	// Clean up
	for _, v := range testVars {
		os.Unsetenv(v)
	}
}

func TestEnvironmentLoader_ValidateRequired(t *testing.T) {
	// Set up test environment
	os.Setenv("REQUIRED_VAR1", "value1")
	os.Setenv("REQUIRED_VAR2", "value2")
	defer func() {
		os.Unsetenv("REQUIRED_VAR1")
		os.Unsetenv("REQUIRED_VAR2")
		os.Unsetenv("OPTIONAL_VAR")
	}()
	
	loader := NewEnvironmentLoader("test")
	
	// Test with all required variables present
	required := []string{"REQUIRED_VAR1", "REQUIRED_VAR2"}
	err := loader.ValidateRequired(required)
	if err != nil {
		t.Errorf("Validation should pass when all required variables are set: %v", err)
	}
	
	// Test with missing required variable
	required = []string{"REQUIRED_VAR1", "REQUIRED_VAR2", "MISSING_VAR"}
	err = loader.ValidateRequired(required)
	if err == nil {
		t.Errorf("Validation should fail when required variables are missing")
	}
	if !strings.Contains(err.Error(), "MISSING_VAR") {
		t.Errorf("Error should mention missing variable, got: %v", err)
	}
}

func TestConfigTemplate_ComplexTemplate(t *testing.T) {
	// Set environment variables first
	os.Setenv("CLUSTER_NAME", "prod-cluster")
	defer os.Unsetenv("CLUSTER_NAME")
	
	// Create template after setting environment variables
	template := NewConfigTemplate()
	
	// Set up complex template context
	template.SetVariable("services", []map[string]interface{}{
		{"name": "web", "port": 8080, "replicas": 3},
		{"name": "api", "port": 9090, "replicas": 2},
	})
	template.SetVariable("environment", "production")
	template.SetDefault("LOG_LEVEL", "info")
	
	complexTemplate := `# Generated configuration for {{env "CLUSTER_NAME"}}
# Environment: {{var "environment"}}
# Generated at: {{formatTime "2006-01-02 15:04:05"}}

log_level = {{default "LOG_LEVEL" "warn"}}

{{range $service := var "services"}}
[service.{{$service.name}}]
port = {{$service.port}}
replicas = {{$service.replicas}}
image = "{{env "REGISTRY_URL" "docker.io"}}/{{$service.name}}:{{env "VERSION" "latest"}}"
{{end}}

[cluster]
name = "{{env "CLUSTER_NAME"}}"
environment = "{{var "environment"}}"
debug = {{if (eq (var "environment") "development")}}true{{else}}false{{end}}`
	
	err := template.LoadTemplate("complex", complexTemplate)
	if err != nil {
		t.Fatalf("Failed to load complex template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute complex template: %v", err)
	}
	
	// Verify template output
	if !strings.Contains(result, "# Generated configuration for prod-cluster") {
		t.Errorf("Expected cluster name in header, got: %s", result)
	}
	if !strings.Contains(result, "Environment: production") {
		t.Errorf("Expected environment in header, got: %s", result)
	}
	if !strings.Contains(result, "log_level = info") {
		t.Errorf("Expected log level from default, got: %s", result)
	}
	if !strings.Contains(result, "[service.web]") {
		t.Errorf("Expected web service section, got: %s", result)
	}
	if !strings.Contains(result, "[service.api]") {
		t.Errorf("Expected api service section, got: %s", result)
	}
	if !strings.Contains(result, "debug = false") {
		t.Errorf("Expected debug false for production, got: %s", result)
	}
}

func TestConfigTemplate_SecretHandling(t *testing.T) {
	tempDir := t.TempDir()
	secretsFile := filepath.Join(tempDir, "secrets.env")
	
	// Create secrets file
	secretsContent := `# Secrets file
API_SECRET=super_secret_key
DB_PASSWORD=secret_password
# Comment line
EMPTY_LINE_ABOVE=value`
	
	err := os.WriteFile(secretsFile, []byte(secretsContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create secrets file: %v", err)
	}
	
	template := NewConfigTemplate()
	template.SetSecretsFile(secretsFile)
	
	tmplContent := `API Secret: {{index .Secrets "API_SECRET"}}
DB Password: {{index .Secrets "DB_PASSWORD"}}
Missing Secret: {{index .Secrets "MISSING_SECRET"}}`
	
	err = template.LoadTemplate("secrets_test", tmplContent)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	result, err := template.ExecuteToString()
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	
	if !strings.Contains(result, "API Secret: super_secret_key") {
		t.Errorf("Expected API secret, got: %s", result)
	}
	if !strings.Contains(result, "DB Password: secret_password") {
		t.Errorf("Expected DB password, got: %s", result)
	}
	if !strings.Contains(result, "Missing Secret: ") {
		t.Errorf("Expected missing secret to be empty, got: %s", result)
	}
}