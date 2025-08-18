package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/scttfrdmn/apprise-go/apprise"
)

// MigrateOptions contains options for migration assistance
type MigrateOptions struct {
	URL      string
	Guide    string  // Service ID for migration guide
	Output   string  // Output format (text, markdown)
	Validate bool    // Validate Python URL compatibility
	Verbose  bool
}

func main() {
	var opts MigrateOptions
	
	flag.StringVar(&opts.URL, "url", "", "Python Apprise URL to validate/migrate")
	flag.StringVar(&opts.Guide, "guide", "", "Generate migration guide for service")
	flag.StringVar(&opts.Output, "output", "text", "Output format (text, markdown)")
	flag.BoolVar(&opts.Validate, "validate", false, "Validate URL compatibility")
	flag.BoolVar(&opts.Verbose, "verbose", false, "Verbose output")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Migration assistance tool for Python Apprise to Apprise-Go.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -url 'discord://123/abc' -validate\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -guide discord -output markdown\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -output markdown > migration-guide.md\n", os.Args[0])
	}
	
	flag.Parse()
	
	if err := runMigration(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runMigration executes migration operations based on options
func runMigration(opts MigrateOptions) error {
	mg := apprise.NewMigrationGuide()
	
	if opts.URL != "" {
		return validateURL(mg, opts.URL, opts.Verbose)
	}
	
	if opts.Guide != "" {
		return generateServiceGuide(mg, opts.Guide, opts.Output)
	}
	
	// Generate full migration guide
	return generateFullGuide(mg, opts.Output)
}

// validateURL validates a Python Apprise URL for Go compatibility
func validateURL(mg *apprise.MigrationGuide, url string, verbose bool) error {
	fmt.Printf("Validating Python Apprise URL: %s\n\n", url)
	
	valid, issues, err := mg.ValidateMigration(url)
	
	if err != nil {
		fmt.Printf("❌ URL Validation Failed: %v\n", err)
		return nil
	}
	
	if valid {
		fmt.Printf("✅ URL is compatible with Apprise-Go\n")
	} else {
		fmt.Printf("⚠️  URL has compatibility issues:\n")
		for _, issue := range issues {
			fmt.Printf("   - %s\n", issue)
		}
	}
	
	// Extract service type and show migration guide if available
	if idx := strings.Index(url, "://"); idx > 0 {
		serviceType := url[:idx]
		
		if migration, exists := mg.GetMigrationGuide(serviceType); exists {
			fmt.Printf("\n📋 Migration Guide for %s:\n", strings.Title(serviceType))
			
			if len(migration.Changes) > 0 {
				fmt.Printf("\nChanges from Python version:\n")
				for _, change := range migration.Changes {
					status := "Optional"
					if change.Required {
						status = "Required"
					}
					fmt.Printf("  • [%s] %s: %s\n", status, strings.Title(change.Type), change.Description)
					if change.Before != "" && change.After != "" {
						fmt.Printf("    Before: %s\n", change.Before)
						fmt.Printf("    After:  %s\n", change.After)
					}
				}
			}
			
			if len(migration.Notes) > 0 {
				fmt.Printf("\nNotes:\n")
				for _, note := range migration.Notes {
					fmt.Printf("  • %s\n", note)
				}
			}
			
			if verbose && len(migration.Examples) > 0 {
				fmt.Printf("\nExamples:\n")
				for _, example := range migration.Examples {
					fmt.Printf("\n%s:\n", example.Description)
					if example.Python != "" {
						fmt.Printf("  Python: %s\n", example.Python)
					}
					if example.Go != "" {
						fmt.Printf("  Go:     %s\n", example.Go)
					}
				}
			}
		} else {
			fmt.Printf("\n❓ No specific migration guide available for service: %s\n", serviceType)
		}
	}
	
	return nil
}

// generateServiceGuide generates a migration guide for a specific service
func generateServiceGuide(mg *apprise.MigrationGuide, serviceID, format string) error {
	migration, exists := mg.GetMigrationGuide(serviceID)
	if !exists {
		return fmt.Errorf("no migration guide available for service: %s", serviceID)
	}
	
	if format == "markdown" {
		output := generateServiceMarkdownGuide(serviceID, migration)
		fmt.Print(output)
	} else {
		output := generateServiceTextGuide(serviceID, migration)
		fmt.Print(output)
	}
	
	return nil
}

// generateFullGuide generates the complete migration guide
func generateFullGuide(mg *apprise.MigrationGuide, format string) error {
	if format == "markdown" {
		fmt.Print(mg.GenerateMigrationDocumentation())
	} else {
		// Generate text version
		fmt.Print(convertMarkdownToText(mg.GenerateMigrationDocumentation()))
	}
	
	return nil
}

// generateServiceMarkdownGuide generates markdown for a single service migration
func generateServiceMarkdownGuide(serviceID string, migration apprise.ServiceMigration) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("# %s Migration Guide\n\n", strings.Title(serviceID)))
	
	sb.WriteString("## URL Schema\n\n")
	sb.WriteString(fmt.Sprintf("- **Python:** `%s`\n", migration.PythonSchema))
	sb.WriteString(fmt.Sprintf("- **Go:** `%s`\n\n", migration.GoSchema))
	
	if len(migration.Changes) > 0 {
		sb.WriteString("## Changes\n\n")
		for _, change := range migration.Changes {
			required := ""
			if change.Required {
				required = " ⚠️ **Required**"
			}
			sb.WriteString(fmt.Sprintf("### %s%s\n\n", strings.Title(change.Type), required))
			sb.WriteString(fmt.Sprintf("%s\n\n", change.Description))
			if change.Before != "" {
				sb.WriteString(fmt.Sprintf("**Before:** `%s`\n\n", change.Before))
			}
			if change.After != "" {
				sb.WriteString(fmt.Sprintf("**After:** `%s`\n\n", change.After))
			}
		}
	}
	
	if len(migration.Examples) > 0 {
		sb.WriteString("## Examples\n\n")
		for _, example := range migration.Examples {
			sb.WriteString(fmt.Sprintf("### %s\n\n", example.Description))
			if example.Python != "" {
				sb.WriteString("**Python:**\n```python\n" + example.Python + "\n```\n\n")
			}
			if example.Go != "" {
				sb.WriteString("**Go:**\n```go\n" + example.Go + "\n```\n\n")
			}
		}
	}
	
	if len(migration.Notes) > 0 {
		sb.WriteString("## Notes\n\n")
		for _, note := range migration.Notes {
			sb.WriteString(fmt.Sprintf("- %s\n", note))
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}

// generateServiceTextGuide generates plain text for a single service migration  
func generateServiceTextGuide(serviceID string, migration apprise.ServiceMigration) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("%s Migration Guide\n", strings.ToUpper(serviceID)))
	sb.WriteString(strings.Repeat("=", len(serviceID)+16) + "\n\n")
	
	sb.WriteString("URL Schema:\n")
	sb.WriteString(fmt.Sprintf("  Python: %s\n", migration.PythonSchema))
	sb.WriteString(fmt.Sprintf("  Go:     %s\n\n", migration.GoSchema))
	
	if len(migration.Changes) > 0 {
		sb.WriteString("Changes:\n")
		for _, change := range migration.Changes {
			required := ""
			if change.Required {
				required = " [REQUIRED]"
			}
			sb.WriteString(fmt.Sprintf("  • %s%s: %s\n", strings.Title(change.Type), required, change.Description))
			if change.Before != "" {
				sb.WriteString(fmt.Sprintf("    Before: %s\n", change.Before))
			}
			if change.After != "" {
				sb.WriteString(fmt.Sprintf("    After:  %s\n", change.After))
			}
		}
		sb.WriteString("\n")
	}
	
	if len(migration.Examples) > 0 {
		sb.WriteString("Examples:\n")
		for _, example := range migration.Examples {
			sb.WriteString(fmt.Sprintf("  %s:\n", example.Description))
			if example.Python != "" {
				sb.WriteString(fmt.Sprintf("    Python: %s\n", example.Python))
			}
			if example.Go != "" {
				sb.WriteString(fmt.Sprintf("    Go:     %s\n", example.Go))
			}
		}
		sb.WriteString("\n")
	}
	
	if len(migration.Notes) > 0 {
		sb.WriteString("Notes:\n")
		for _, note := range migration.Notes {
			sb.WriteString(fmt.Sprintf("  • %s\n", note))
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}

// convertMarkdownToText provides basic markdown to text conversion
func convertMarkdownToText(markdown string) string {
	text := markdown
	
	// Remove markdown formatting
	text = strings.ReplaceAll(text, "# ", "")
	text = strings.ReplaceAll(text, "## ", "")
	text = strings.ReplaceAll(text, "### ", "")
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "`", "")
	text = strings.ReplaceAll(text, "```go", "")
	text = strings.ReplaceAll(text, "```python", "")
	text = strings.ReplaceAll(text, "```", "")
	
	return text
}