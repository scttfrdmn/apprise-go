package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scttfrdmn/apprise-go/apprise"
)

// DocumentationOptions contains configuration for documentation generation
type DocumentationOptions struct {
	Format     string // markdown, json, html
	Output     string // output file path
	Service    string // specific service to document
	Category   string // specific category to document
	Template   string // custom template file
	Verbose    bool
}

func main() {
	var opts DocumentationOptions
	
	flag.StringVar(&opts.Format, "format", "markdown", "Output format (markdown, json, html)")
	flag.StringVar(&opts.Output, "output", "", "Output file path (stdout if empty)")
	flag.StringVar(&opts.Service, "service", "", "Document specific service only")
	flag.StringVar(&opts.Category, "category", "", "Document specific category only")
	flag.StringVar(&opts.Template, "template", "", "Custom template file")
	flag.BoolVar(&opts.Verbose, "verbose", false, "Verbose output")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate comprehensive documentation for Apprise-Go services.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -format markdown -output docs/services.md\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -format json -service discord\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -format html -category messaging -output messaging.html\n", os.Args[0])
	}
	
	flag.Parse()
	
	if err := generateDocumentation(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// generateDocumentation generates documentation based on options
func generateDocumentation(opts DocumentationOptions) error {
	if opts.Verbose {
		fmt.Printf("Generating documentation in %s format...\n", opts.Format)
	}
	
	dg := apprise.NewDocumentationGenerator()
	
	var output string
	var err error
	
	switch strings.ToLower(opts.Format) {
	case "markdown", "md":
		output, err = generateMarkdown(dg, opts)
	case "json":
		output, err = generateJSON(dg, opts)
	case "html":
		output, err = generateHTML(dg, opts)
	default:
		return fmt.Errorf("unsupported format: %s", opts.Format)
	}
	
	if err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}
	
	// Output to file or stdout
	if opts.Output == "" {
		fmt.Print(output)
	} else {
		if opts.Verbose {
			fmt.Printf("Writing to %s...\n", opts.Output)
		}
		
		// Ensure output directory exists
		if dir := filepath.Dir(opts.Output); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}
		
		if err := os.WriteFile(opts.Output, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		
		if opts.Verbose {
			fmt.Printf("Documentation written to %s\n", opts.Output)
		}
	}
	
	return nil
}

// generateMarkdown generates markdown documentation
func generateMarkdown(dg *apprise.DocumentationGenerator, opts DocumentationOptions) (string, error) {
	if opts.Service != "" {
		return generateServiceMarkdown(dg, opts.Service)
	}
	
	if opts.Category != "" {
		return generateCategoryMarkdown(dg, opts.Category)
	}
	
	return dg.GenerateMarkdownDocumentation(), nil
}

// generateServiceMarkdown generates markdown for a specific service
func generateServiceMarkdown(dg *apprise.DocumentationGenerator, serviceID string) (string, error) {
	doc, exists := dg.GetServiceDocumentation(serviceID)
	if !exists {
		return "", fmt.Errorf("service '%s' not found", serviceID)
	}
	
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("# %s Service Documentation\n\n", doc.Name))
	sb.WriteString(fmt.Sprintf("**Service ID:** `%s`\n\n", doc.ID))
	sb.WriteString(fmt.Sprintf("%s\n\n", doc.Description))
	
	// URL Format
	sb.WriteString("## URL Format\n\n```\n")
	sb.WriteString(fmt.Sprintf("%s\n", doc.URLFormat))
	sb.WriteString("```\n\n")
	
	// Parameters
	if len(doc.Parameters) > 0 {
		sb.WriteString("## Parameters\n\n")
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
		sb.WriteString("## Examples\n\n")
		for i, example := range doc.Examples {
			sb.WriteString(fmt.Sprintf("### Example %d: %s\n\n", i+1, example.Description))
			sb.WriteString("**URL:**\n```\n")
			sb.WriteString(fmt.Sprintf("%s\n", example.URL))
			sb.WriteString("```\n\n")
			sb.WriteString("**Go Code:**\n```go\n")
			sb.WriteString(fmt.Sprintf("%s\n", example.Code))
			sb.WriteString("```\n\n")
		}
	}
	
	// Setup
	if len(doc.Setup) > 0 {
		sb.WriteString("## Setup Instructions\n\n")
		for _, step := range doc.Setup {
			sb.WriteString(fmt.Sprintf("%s\n", step))
		}
		sb.WriteString("\n")
	}
	
	// Limitations
	if len(doc.Limitations) > 0 {
		sb.WriteString("## Limitations\n\n")
		for _, limitation := range doc.Limitations {
			sb.WriteString(fmt.Sprintf("- %s\n", limitation))
		}
		sb.WriteString("\n")
	}
	
	return sb.String(), nil
}

// generateCategoryMarkdown generates markdown for a specific category
func generateCategoryMarkdown(dg *apprise.DocumentationGenerator, categoryID string) (string, error) {
	categories := dg.GetServiceCategories()
	category, exists := categories[categoryID]
	if !exists {
		return "", fmt.Errorf("category '%s' not found", categoryID)
	}
	
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("# %s\n\n", category.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", category.Description))
	
	// List services
	sb.WriteString("## Services\n\n")
	for _, serviceID := range category.Services {
		if doc, exists := dg.GetServiceDocumentation(serviceID); exists {
			sb.WriteString(fmt.Sprintf("- **[%s](#%s)** - %s\n", doc.Name, serviceID, doc.Description))
		}
	}
	sb.WriteString("\n")
	
	// Detailed documentation for each service
	for _, serviceID := range category.Services {
		if _, exists := dg.GetServiceDocumentation(serviceID); exists {
			serviceMarkdown, _ := generateServiceMarkdown(dg, serviceID)
			sb.WriteString(serviceMarkdown)
			sb.WriteString("---\n\n")
		}
	}
	
	return sb.String(), nil
}

// generateJSON generates JSON documentation
func generateJSON(dg *apprise.DocumentationGenerator, opts DocumentationOptions) (string, error) {
	var data interface{}
	
	if opts.Service != "" {
		doc, exists := dg.GetServiceDocumentation(opts.Service)
		if !exists {
			return "", fmt.Errorf("service '%s' not found", opts.Service)
		}
		data = doc
	} else if opts.Category != "" {
		categories := dg.GetServiceCategories()
		category, exists := categories[opts.Category]
		if !exists {
			return "", fmt.Errorf("category '%s' not found", opts.Category)
		}
		
		categoryData := struct {
			Category apprise.ServiceCategory                        `json:"category"`
			Services map[string]apprise.ServiceDocumentation `json:"services"`
		}{
			Category: category,
			Services: make(map[string]apprise.ServiceDocumentation),
		}
		
		// Add service documentation for category
		allDocs := dg.GetAllServiceDocumentation()
		for _, serviceID := range category.Services {
			if doc, exists := allDocs[serviceID]; exists {
				categoryData.Services[serviceID] = doc
			}
		}
		
		data = categoryData
	} else {
		// Full documentation
		data = struct {
			Categories map[string]apprise.ServiceCategory         `json:"categories"`
			Services   map[string]apprise.ServiceDocumentation `json:"services"`
		}{
			Categories: dg.GetServiceCategories(),
			Services:   dg.GetAllServiceDocumentation(),
		}
	}
	
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	return string(jsonBytes), nil
}

// generateHTML generates HTML documentation
func generateHTML(dg *apprise.DocumentationGenerator, opts DocumentationOptions) (string, error) {
	// For now, convert markdown to basic HTML
	markdown, err := generateMarkdown(dg, opts)
	if err != nil {
		return "", err
	}
	
	var sb strings.Builder
	
	// Basic HTML structure
	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html lang=\"en\">\n")
	sb.WriteString("<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString("  <title>Apprise-Go Service Documentation</title>\n")
	sb.WriteString("  <style>\n")
	sb.WriteString(getHTMLStyles())
	sb.WriteString("  </style>\n")
	sb.WriteString("</head>\n")
	sb.WriteString("<body>\n")
	sb.WriteString("  <div class=\"container\">\n")
	
	// Convert markdown to basic HTML
	html := convertMarkdownToHTML(markdown)
	sb.WriteString(html)
	
	sb.WriteString("  </div>\n")
	sb.WriteString("</body>\n")
	sb.WriteString("</html>\n")
	
	return sb.String(), nil
}

// getHTMLStyles returns CSS styles for HTML output
func getHTMLStyles() string {
	return `
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif;
      line-height: 1.6;
      color: #333;
      max-width: 1200px;
      margin: 0 auto;
      padding: 20px;
    }
    .container {
      max-width: 100%;
    }
    h1, h2, h3 {
      color: #2c3e50;
      border-bottom: 2px solid #3498db;
      padding-bottom: 10px;
    }
    table {
      border-collapse: collapse;
      width: 100%;
      margin: 20px 0;
    }
    th, td {
      border: 1px solid #ddd;
      padding: 12px;
      text-align: left;
    }
    th {
      background-color: #f8f9fa;
      font-weight: bold;
    }
    code {
      background-color: #f8f9fa;
      padding: 2px 6px;
      border-radius: 3px;
      font-family: 'Monaco', 'Consolas', monospace;
    }
    pre {
      background-color: #f8f9fa;
      padding: 15px;
      border-radius: 5px;
      overflow-x: auto;
    }
    .service-id {
      color: #e74c3c;
      font-weight: bold;
    }
    .required {
      color: #e74c3c;
      font-weight: bold;
    }
    ul {
      padding-left: 20px;
    }
    li {
      margin-bottom: 5px;
    }
  `
}

// convertMarkdownToHTML provides basic markdown to HTML conversion
func convertMarkdownToHTML(markdown string) string {
	html := markdown
	
	// Headers
	html = strings.ReplaceAll(html, "### ", "<h3>")
	html = strings.ReplaceAll(html, "## ", "<h2>")
	html = strings.ReplaceAll(html, "# ", "<h1>")
	
	// Add closing tags for headers (simplified)
	lines := strings.Split(html, "\n")
	var result strings.Builder
	
	for _, line := range lines {
		if strings.HasPrefix(line, "<h1>") {
			line = line + "</h1>"
		} else if strings.HasPrefix(line, "<h2>") {
			line = line + "</h2>"
		} else if strings.HasPrefix(line, "<h3>") {
			line = line + "</h3>"
		} else if strings.HasPrefix(line, "```") {
			if strings.Contains(line, "```") && len(line) > 3 {
				line = "<pre><code>"
			} else {
				line = "</code></pre>"
			}
		} else if line == "```" {
			// Toggle between pre tags
			if strings.Contains(result.String(), "<pre><code>") && !strings.Contains(result.String(), "</code></pre>") {
				line = "</code></pre>"
			} else {
				line = "<pre><code>"
			}
		} else if strings.HasPrefix(line, "| ") && strings.Contains(line, " |") {
			// Table row
			line = "<tr>" + strings.ReplaceAll(strings.ReplaceAll(line, "| ", "<td>"), " |", "</td>") + "</tr>"
		} else if strings.HasPrefix(line, "|---") {
			// Table header separator (skip)
			continue
		} else if strings.TrimSpace(line) == "" {
			line = "<br>"
		}
		
		result.WriteString(line + "\n")
	}
	
	return result.String()
}