package apprise

import (
	"strings"
	"testing"
)

func TestDocumentationGenerator_BasicFunctionality(t *testing.T) {
	dg := NewDocumentationGenerator()
	
	// Test categories initialization
	categories := dg.GetServiceCategories()
	if len(categories) == 0 {
		t.Error("Expected service categories to be initialized")
	}
	
	// Check for essential categories
	expectedCategories := []string{"messaging", "email", "sms", "mobile", "desktop", "social"}
	for _, expectedCat := range expectedCategories {
		if _, exists := categories[expectedCat]; !exists {
			t.Errorf("Expected category '%s' to exist", expectedCat)
		}
	}
}

func TestDocumentationGenerator_ServiceDocumentation(t *testing.T) {
	dg := NewDocumentationGenerator()
	
	// Test Discord documentation
	discordDoc, exists := dg.GetServiceDocumentation("discord")
	if !exists {
		t.Error("Expected Discord documentation to exist")
	}
	
	if discordDoc.Name != "Discord" {
		t.Errorf("Expected Discord name, got %s", discordDoc.Name)
	}
	
	if discordDoc.Category != "messaging" {
		t.Errorf("Expected messaging category, got %s", discordDoc.Category)
	}
	
	if len(discordDoc.Parameters) == 0 {
		t.Error("Expected Discord to have parameters")
	}
	
	if len(discordDoc.Examples) == 0 {
		t.Error("Expected Discord to have examples")
	}
}

func TestDocumentationGenerator_MarkdownGeneration(t *testing.T) {
	dg := NewDocumentationGenerator()
	
	markdown := dg.GenerateMarkdownDocumentation()
	
	// Check basic structure
	if !strings.Contains(markdown, "# Apprise-Go Service Documentation") {
		t.Error("Expected markdown to contain main header")
	}
	
	if !strings.Contains(markdown, "## Table of Contents") {
		t.Error("Expected markdown to contain table of contents")
	}
	
	// Check for category sections
	if !strings.Contains(markdown, "## Messaging & Chat") {
		t.Error("Expected markdown to contain Messaging & Chat section")
	}
	
	// Check for service documentation
	if !strings.Contains(markdown, "### Discord") {
		t.Error("Expected markdown to contain Discord service documentation")
	}
	
	if !strings.Contains(markdown, "**Service ID:** `discord`") {
		t.Error("Expected markdown to contain Discord service ID")
	}
	
	// Check for parameters table
	if !strings.Contains(markdown, "| Name | Type | Required | Description | Default | Example |") {
		t.Error("Expected markdown to contain parameters table header")
	}
}

func TestDocumentationGenerator_ServiceReflection(t *testing.T) {
	dg := NewDocumentationGenerator()
	
	// Test reflection on Discord service
	discordInfo := dg.GetServiceByReflection("discord")
	if discordInfo == nil {
		t.Error("Expected Discord reflection info to be available")
	}
	
	if discordInfo["name"] != "DiscordService" {
		t.Errorf("Expected DiscordService name, got %v", discordInfo["name"])
	}
	
	// Check methods
	methods, ok := discordInfo["methods"].([]string)
	if !ok {
		t.Error("Expected methods to be string slice")
	}
	
	expectedMethods := []string{"Send", "ParseURL"}
	for _, expectedMethod := range expectedMethods {
		found := false
		for _, method := range methods {
			if method == expectedMethod {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected method %s not found in methods: %v", expectedMethod, methods)
		}
	}
}

func TestDocumentationGenerator_AllServicesCovered(t *testing.T) {
	dg := NewDocumentationGenerator()
	
	// Get all supported services
	supportedServices := GetSupportedServices()
	allDocs := dg.GetAllServiceDocumentation()
	
	// Check that major services have documentation
	majorServices := []string{"discord", "slack", "email", "twilio", "rich-mobile-push", "desktop-advanced"}
	
	for _, serviceID := range majorServices {
		if _, exists := allDocs[serviceID]; !exists {
			t.Errorf("Major service %s should have documentation", serviceID)
		}
	}
	
	// Count services with documentation vs total supported
	documentedCount := len(allDocs)
	supportedCount := len(supportedServices)
	
	coverage := float64(documentedCount) / float64(supportedCount)
	if coverage < 0.1 { // At least 10% coverage for now
		t.Errorf("Documentation coverage too low: %d/%d (%.1f%%)", documentedCount, supportedCount, coverage*100)
	}
	
	t.Logf("Documentation coverage: %d/%d services (%.1f%%)", documentedCount, supportedCount, coverage*100)
}

func TestDocumentationGenerator_ParameterValidation(t *testing.T) {
	dg := NewDocumentationGenerator()
	
	// Test Discord parameters
	discordDoc, _ := dg.GetServiceDocumentation("discord")
	
	// Find required parameters
	requiredParams := make([]ServiceParameter, 0)
	for _, param := range discordDoc.Parameters {
		if param.Required {
			requiredParams = append(requiredParams, param)
		}
	}
	
	if len(requiredParams) == 0 {
		t.Error("Expected Discord to have required parameters")
	}
	
	// Check that required parameters have proper documentation
	for _, param := range requiredParams {
		if param.Description == "" {
			t.Errorf("Required parameter %s should have description", param.Name)
		}
		
		if param.Example == "" {
			t.Errorf("Required parameter %s should have example", param.Name)
		}
	}
}

func TestDocumentationGenerator_ExampleValidation(t *testing.T) {
	dg := NewDocumentationGenerator()
	
	allDocs := dg.GetAllServiceDocumentation()
	
	for serviceID, doc := range allDocs {
		// Each service should have at least one example
		if len(doc.Examples) == 0 {
			t.Errorf("Service %s should have at least one example", serviceID)
			continue
		}
		
		// Each example should have description, URL, and code
		for i, example := range doc.Examples {
			if example.Description == "" {
				t.Errorf("Service %s example %d should have description", serviceID, i)
			}
			
			if example.URL == "" {
				t.Errorf("Service %s example %d should have URL", serviceID, i)
			}
			
			if example.Code == "" {
				t.Errorf("Service %s example %d should have code", serviceID, i)
			}
			
			// Code should contain the service ID
			if !strings.Contains(example.Code, serviceID) && !strings.Contains(example.URL, serviceID) {
				t.Errorf("Service %s example %d should reference the service", serviceID, i)
			}
		}
	}
}

func TestDocumentationGenerator_CategoryConsistency(t *testing.T) {
	dg := NewDocumentationGenerator()
	
	categories := dg.GetServiceCategories()
	allDocs := dg.GetAllServiceDocumentation()
	
	// Check that all services referenced in categories have documentation
	for categoryID, category := range categories {
		for _, serviceID := range category.Services {
			doc, exists := allDocs[serviceID]
			if !exists {
				t.Errorf("Category %s references service %s but no documentation exists", categoryID, serviceID)
				continue
			}
			
			// Check that service's category matches
			if doc.Category != categoryID {
				t.Errorf("Service %s is in category %s but documentation says %s", serviceID, categoryID, doc.Category)
			}
		}
	}
}

func TestDocumentationGenerator_MarkdownStructure(t *testing.T) {
	dg := NewDocumentationGenerator()
	
	markdown := dg.GenerateMarkdownDocumentation()
	lines := strings.Split(markdown, "\n")
	
	// Test structure elements
	hasMainHeader := false
	hasTOC := false
	hasCategorySection := false
	hasServiceSection := false
	hasParametersTable := false
	
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			hasMainHeader = true
		}
		if strings.Contains(line, "## Table of Contents") {
			hasTOC = true
		}
		if strings.HasPrefix(line, "## ") && strings.Contains(line, "&") {
			hasCategorySection = true
		}
		if strings.HasPrefix(line, "### ") {
			hasServiceSection = true
		}
		if strings.Contains(line, "| Name | Type | Required") {
			hasParametersTable = true
		}
	}
	
	if !hasMainHeader {
		t.Error("Markdown should have main header")
	}
	if !hasTOC {
		t.Error("Markdown should have table of contents")
	}
	if !hasCategorySection {
		t.Error("Markdown should have category sections")
	}
	if !hasServiceSection {
		t.Error("Markdown should have service sections")
	}
	if !hasParametersTable {
		t.Error("Markdown should have parameters tables")
	}
}

func BenchmarkDocumentationGeneration(b *testing.B) {
	dg := NewDocumentationGenerator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dg.GenerateMarkdownDocumentation()
	}
}

func BenchmarkServiceReflection(b *testing.B) {
	dg := NewDocumentationGenerator()
	services := []string{"discord", "slack", "email", "twilio", "desktop"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, serviceID := range services {
			_ = dg.GetServiceByReflection(serviceID)
		}
	}
}