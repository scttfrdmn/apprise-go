package apprise

import (
	"fmt"
	"strings"
)

// MigrationGuide provides migration assistance from Python Apprise
type MigrationGuide struct {
	pythonToGoMapping map[string]ServiceMigration
}

// ServiceMigration contains migration information for a service
type ServiceMigration struct {
	PythonSchema   string
	GoSchema       string
	Changes        []MigrationChange
	Examples       []MigrationExample
	Notes          []string
}

// MigrationChange describes a specific change in migration
type MigrationChange struct {
	Type        string // "parameter", "url", "behavior"
	Description string
	Before      string
	After       string
	Required    bool
}

// MigrationExample shows before and after migration examples
type MigrationExample struct {
	Description string
	Python      string
	Go          string
}

// NewMigrationGuide creates a migration guide from Python Apprise
func NewMigrationGuide() *MigrationGuide {
	mg := &MigrationGuide{
		pythonToGoMapping: make(map[string]ServiceMigration),
	}
	
	mg.initializeMigrations()
	return mg
}

// initializeMigrations sets up migration information
func (mg *MigrationGuide) initializeMigrations() {
	// Discord migration
	mg.pythonToGoMapping["discord"] = ServiceMigration{
		PythonSchema: "discord://webhook_id/webhook_token",
		GoSchema:     "discord://webhook_id/webhook_token",
		Changes: []MigrationChange{
			{
				Type:        "parameter",
				Description: "Avatar parameter syntax changed",
				Before:      "?avatar=https://example.com/image.png",
				After:       "?avatar=MyBot (name) or ?avatar=https://example.com/image.png (URL)",
				Required:    false,
			},
			{
				Type:        "behavior", 
				Description: "Default message format is now text instead of markdown",
				Before:      "markdown by default",
				After:       "text by default, use ?format=markdown for markdown",
				Required:    false,
			},
		},
		Examples: []MigrationExample{
			{
				Description: "Basic Discord notification",
				Python:      `apprise.Apprise().add("discord://123456789/abcdef123456789")`,
				Go:          `app := apprise.New(); app.Add("discord://123456789/abcdef123456789")`,
			},
			{
				Description: "Discord with custom formatting",
				Python:      `apprise.Apprise().add("discord://123456789/abcdef123456789?format=markdown")`,
				Go:          `app.Add("discord://123456789/abcdef123456789?format=markdown")`,
			},
		},
		Notes: []string{
			"URL format is identical between Python and Go versions",
			"Parameter handling is consistent",
			"Error handling uses Go idioms instead of Python exceptions",
		},
	}
	
	// Slack migration
	mg.pythonToGoMapping["slack"] = ServiceMigration{
		PythonSchema: "slack://TokenA/TokenB/TokenC/Channel",
		GoSchema:     "slack://TokenA/TokenB/TokenC/Channel",
		Changes: []MigrationChange{
			{
				Type:        "parameter",
				Description: "Channel parameter can now include # prefix",
				Before:      "slack://token/token/token/general",
				After:       "slack://token/token/token/general or slack://token/token/token/#general",
				Required:    false,
			},
			{
				Type:        "behavior",
				Description: "Bot token validation is more strict",
				Before:      "Accepts various token formats",
				After:       "Requires properly formatted bot tokens",
				Required:    true,
			},
		},
		Examples: []MigrationExample{
			{
				Description: "Slack webhook notification",
				Python:      `apprise.Apprise().add("slack://T123/B456/xyz789/general")`,
				Go:          `app.Add("slack://T123/B456/xyz789/#general")`,
			},
		},
		Notes: []string{
			"Token format validation is stricter in Go version",
			"Channel names are normalized consistently",
		},
	}
	
	// Email migration
	mg.pythonToGoMapping["email"] = ServiceMigration{
		PythonSchema: "mailto://user:pass@server:port/to_email",
		GoSchema:     "mailto://user:pass@server:port/to_email",
		Changes: []MigrationChange{
			{
				Type:        "parameter",
				Description: "TLS handling is automatic based on port",
				Before:      "?secure=yes for TLS",
				After:       "Automatic TLS detection, use ?tls=false to disable",
				Required:    false,
			},
			{
				Type:        "behavior",
				Description: "SMTP authentication is more robust",
				Before:      "Basic SMTP auth",
				After:       "Supports PLAIN, LOGIN, and CRAM-MD5",
				Required:    false,
			},
		},
		Examples: []MigrationExample{
			{
				Description: "Gmail SMTP configuration",
				Python:      `apprise.Apprise().add("mailto://user:pass@smtp.gmail.com:587/to@example.com")`,
				Go:          `app.Add("mailto://user:pass@smtp.gmail.com:587/to@example.com")`,
			},
		},
		Notes: []string{
			"SMTP connection handling is more reliable",
			"TLS negotiation is automatic",
			"Better error reporting for authentication failures",
		},
	}
	
	// Twilio migration
	mg.pythonToGoMapping["twilio"] = ServiceMigration{
		PythonSchema: "twilio://account_sid:auth_token@from_number/to_number",
		GoSchema:     "twilio://account_sid:auth_token@from_number/to_number",
		Changes: []MigrationChange{
			{
				Type:        "parameter",
				Description: "Phone number formatting is normalized",
				Before:      "Various formats accepted",
				After:       "E.164 format preferred (+1234567890)",
				Required:    false,
			},
			{
				Type:        "behavior",
				Description: "SMS length limits are enforced",
				Before:      "Long messages may fail silently",
				After:       "Messages over 160 chars are split or rejected with error",
				Required:    false,
			},
		},
		Examples: []MigrationExample{
			{
				Description: "Twilio SMS configuration",
				Python:      `apprise.Apprise().add("twilio://SID:TOKEN@+1234567890/+0987654321")`,
				Go:          `app.Add("twilio://SID:TOKEN@+1234567890/+0987654321")`,
			},
		},
		Notes: []string{
			"Phone number validation is stricter",
			"Better error messages for invalid numbers",
			"Supports international formatting",
		},
	}
	
	// Desktop migration
	mg.pythonToGoMapping["desktop"] = ServiceMigration{
		PythonSchema: "desktop://",
		GoSchema:     "desktop://",
		Changes: []MigrationChange{
			{
				Type:        "behavior",
				Description: "Cross-platform notification handling improved",
				Before:      "Platform-specific implementations",
				After:       "Unified API with platform-specific optimizations",
				Required:    false,
			},
			{
				Type:        "parameter",
				Description: "New advanced desktop notification options",
				Before:      "desktop://",
				After:       "desktop-advanced:// for enhanced features",
				Required:    false,
			},
		},
		Examples: []MigrationExample{
			{
				Description: "Basic desktop notification",
				Python:      `apprise.Apprise().add("desktop://")`,
				Go:          `app.Add("desktop://")`,
			},
			{
				Description: "Advanced desktop notification (Go only)",
				Python:      "N/A",
				Go:          `app.Add("desktop-advanced://?sound=alert&timeout=10")`,
			},
		},
		Notes: []string{
			"Go version includes advanced desktop notification features",
			"Better cross-platform compatibility",
			"Interactive notifications available on supported platforms",
		},
	}
}

// GetMigrationGuide returns migration information for a service
func (mg *MigrationGuide) GetMigrationGuide(serviceID string) (ServiceMigration, bool) {
	migration, exists := mg.pythonToGoMapping[serviceID]
	return migration, exists
}

// GetAllMigrationGuides returns all migration guides
func (mg *MigrationGuide) GetAllMigrationGuides() map[string]ServiceMigration {
	return mg.pythonToGoMapping
}

// GenerateMigrationDocumentation generates comprehensive migration documentation
func (mg *MigrationGuide) GenerateMigrationDocumentation() string {
	var sb strings.Builder
	
	sb.WriteString("# Migration Guide: Python Apprise to Apprise-Go\n\n")
	sb.WriteString("This guide helps you migrate from Python Apprise to Apprise-Go.\n\n")
	
	sb.WriteString("## Overview\n\n")
	sb.WriteString("Apprise-Go maintains compatibility with Python Apprise URL formats while providing:\n")
	sb.WriteString("- Better performance and lower memory usage\n")
	sb.WriteString("- Native Go concurrency and error handling\n") 
	sb.WriteString("- Enhanced service features and reliability\n")
	sb.WriteString("- Simplified deployment (single binary)\n\n")
	
	sb.WriteString("## General Migration Steps\n\n")
	sb.WriteString("1. Install Apprise-Go: `go get github.com/scttfrdmn/apprise-go/apprise`\n")
	sb.WriteString("2. Replace Python imports with Go imports\n")
	sb.WriteString("3. Adapt Python syntax to Go syntax\n")
	sb.WriteString("4. Update configuration files if needed\n")
	sb.WriteString("5. Test notifications with your existing URLs\n\n")
	
	sb.WriteString("## Basic Syntax Comparison\n\n")
	sb.WriteString("### Python\n")
	sb.WriteString("```python\n")
	sb.WriteString("import apprise\n")
	sb.WriteString("\n")
	sb.WriteString("apobj = apprise.Apprise()\n")
	sb.WriteString("apobj.add('discord://webhook_id/webhook_token')\n")
	sb.WriteString("apobj.notify('Hello World', title='Test')\n")
	sb.WriteString("```\n\n")
	
	sb.WriteString("### Go\n")
	sb.WriteString("```go\n")
	sb.WriteString("import \"github.com/scttfrdmn/apprise-go/apprise\"\n")
	sb.WriteString("\n")
	sb.WriteString("app := apprise.New()\n")
	sb.WriteString("app.Add(\"discord://webhook_id/webhook_token\")\n")
	sb.WriteString("app.Notify(\"Test\", \"Hello World\", apprise.NotifyTypeInfo)\n")
	sb.WriteString("```\n\n")
	
	sb.WriteString("## Service-Specific Migration\n\n")
	
	for serviceID, migration := range mg.pythonToGoMapping {
		sb.WriteString(fmt.Sprintf("### %s\n\n", strings.Title(serviceID)))
		
		// Schema comparison
		sb.WriteString("**URL Schema:**\n")
		sb.WriteString("- Python: `" + migration.PythonSchema + "`\n")
		sb.WriteString("- Go: `" + migration.GoSchema + "`\n\n")
		
		// Changes
		if len(migration.Changes) > 0 {
			sb.WriteString("**Changes:**\n")
			for _, change := range migration.Changes {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n", strings.Title(change.Type), change.Description))
				if change.Before != "" {
					sb.WriteString(fmt.Sprintf("  - Before: `%s`\n", change.Before))
				}
				if change.After != "" {
					sb.WriteString(fmt.Sprintf("  - After: `%s`\n", change.After))
				}
				if change.Required {
					sb.WriteString("  - **Required Change**\n")
				}
			}
			sb.WriteString("\n")
		}
		
		// Examples
		if len(migration.Examples) > 0 {
			sb.WriteString("**Migration Examples:**\n\n")
			for _, example := range migration.Examples {
				sb.WriteString(fmt.Sprintf("*%s:*\n\n", example.Description))
				if example.Python != "" {
					sb.WriteString("Python:\n```python\n" + example.Python + "\n```\n\n")
				}
				if example.Go != "" {
					sb.WriteString("Go:\n```go\n" + example.Go + "\n```\n\n")
				}
			}
		}
		
		// Notes
		if len(migration.Notes) > 0 {
			sb.WriteString("**Migration Notes:**\n")
			for _, note := range migration.Notes {
				sb.WriteString(fmt.Sprintf("- %s\n", note))
			}
			sb.WriteString("\n")
		}
		
		sb.WriteString("---\n\n")
	}
	
	// Common migration patterns
	sb.WriteString("## Common Migration Patterns\n\n")
	
	sb.WriteString("### Error Handling\n\n")
	sb.WriteString("**Python:**\n")
	sb.WriteString("```python\n")
	sb.WriteString("try:\n")
	sb.WriteString("    apobj.notify('message')\n")
	sb.WriteString("except Exception as e:\n")
	sb.WriteString("    print(f'Error: {e}')\n")
	sb.WriteString("```\n\n")
	
	sb.WriteString("**Go:**\n")
	sb.WriteString("```go\n")
	sb.WriteString("responses := app.Notify(\"Title\", \"Message\", apprise.NotifyTypeInfo)\n")
	sb.WriteString("for _, response := range responses {\n")
	sb.WriteString("    if response.Error != nil {\n")
	sb.WriteString("        fmt.Printf(\"Error: %v\\n\", response.Error)\n")
	sb.WriteString("    }\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")
	
	sb.WriteString("### Batch Notifications\n\n")
	sb.WriteString("**Python:**\n")
	sb.WriteString("```python\n")
	sb.WriteString("apobj.add(['discord://...', 'slack://...'])\n")
	sb.WriteString("apobj.notify('message')\n")
	sb.WriteString("```\n\n")
	
	sb.WriteString("**Go:**\n")
	sb.WriteString("```go\n")
	sb.WriteString("urls := []string{\"discord://...\", \"slack://...\"}\n")
	sb.WriteString("for _, url := range urls {\n")
	sb.WriteString("    app.Add(url)\n")
	sb.WriteString("}\n")
	sb.WriteString("app.Notify(\"Title\", \"Message\", apprise.NotifyTypeInfo)\n")
	sb.WriteString("```\n\n")
	
	sb.WriteString("### Configuration Files\n\n")
	sb.WriteString("**Python:**\n")
	sb.WriteString("```python\n")
	sb.WriteString("config = apprise.AppriseConfig()\n")
	sb.WriteString("config.add('path/to/config.yml')\n")
	sb.WriteString("apobj.add(config)\n")
	sb.WriteString("```\n\n")
	
	sb.WriteString("**Go:**\n")
	sb.WriteString("```go\n")
	sb.WriteString("config := apprise.NewConfigLoader(app)\n")
	sb.WriteString("config.AddFromFile(\"path/to/config.yml\")\n")
	sb.WriteString("config.ApplyToApprise()\n")
	sb.WriteString("```\n\n")
	
	sb.WriteString("## Advanced Features in Apprise-Go\n\n")
	sb.WriteString("The Go version includes several enhancements not available in Python:\n\n")
	sb.WriteString("- **Rich Mobile Push**: Enhanced push notifications with actions and rich content\n")
	sb.WriteString("- **Advanced Desktop**: Interactive desktop notifications with callbacks\n")
	sb.WriteString("- **Configuration Templating**: Environment-based configuration with templates\n")
	sb.WriteString("- **Batch Processing**: Optimized batch notification handling\n")
	sb.WriteString("- **Concurrent Delivery**: Native Go concurrency for faster delivery\n")
	sb.WriteString("- **Better Error Handling**: Detailed error responses per service\n\n")
	
	sb.WriteString("## Troubleshooting\n\n")
	sb.WriteString("### URL Format Issues\n")
	sb.WriteString("- Ensure special characters are properly URL-encoded\n")
	sb.WriteString("- Check that tokens and credentials are valid\n")
	sb.WriteString("- Verify service-specific parameter requirements\n\n")
	
	sb.WriteString("### Service Differences\n")
	sb.WriteString("- Some services have stricter validation in Go version\n")
	sb.WriteString("- Error messages are more descriptive\n")
	sb.WriteString("- Rate limiting behavior may differ slightly\n\n")
	
	sb.WriteString("### Performance Considerations\n")
	sb.WriteString("- Go version typically uses less memory\n")
	sb.WriteString("- Concurrent notifications are faster\n")
	sb.WriteString("- Startup time is significantly reduced\n\n")
	
	return sb.String()
}

// ValidateMigration checks if a Python URL works with Go version
func (mg *MigrationGuide) ValidateMigration(pythonURL string) (bool, []string, error) {
	var issues []string
	
	// Try to parse the URL with Go version
	app := New()
	err := app.Add(pythonURL)
	
	if err != nil {
		issues = append(issues, fmt.Sprintf("URL parsing failed: %v", err))
		return false, issues, err
	}
	
	// Extract service type from URL
	if idx := strings.Index(pythonURL, "://"); idx > 0 {
		serviceType := pythonURL[:idx]
		
		// Check if migration guide exists
		if migration, exists := mg.pythonToGoMapping[serviceType]; exists {
			// Check for required changes
			for _, change := range migration.Changes {
				if change.Required && strings.Contains(pythonURL, change.Before) {
					issues = append(issues, fmt.Sprintf("Required change needed: %s", change.Description))
				}
			}
		} else {
			issues = append(issues, fmt.Sprintf("No migration guide available for service: %s", serviceType))
		}
	}
	
	return len(issues) == 0, issues, nil
}