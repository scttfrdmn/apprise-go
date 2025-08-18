#!/bin/bash

# Script to generate comprehensive Apprise-Go documentation
# This script creates all documentation files needed for the project

set -e

DOCS_DIR="docs"
TOOLS_DIR="/tmp"
BUILD_DIR="."

echo "🔧 Building documentation tools..."

# Build documentation tools
go build -o "$TOOLS_DIR/apprise-docs" ./cmd/apprise-docs/
go build -o "$TOOLS_DIR/apprise-migrate" ./cmd/apprise-migrate/

echo "📁 Creating documentation directory structure..."

# Create documentation directories
mkdir -p "$DOCS_DIR"
mkdir -p "$DOCS_DIR/services"
mkdir -p "$DOCS_DIR/migration"
mkdir -p "$DOCS_DIR/examples"

echo "📚 Generating service documentation..."

# Generate full service documentation
"$TOOLS_DIR/apprise-docs" -format markdown -output "$DOCS_DIR/services/README.md"

# Generate category-specific documentation
categories=("messaging" "email" "sms" "mobile" "desktop" "social")

for category in "${categories[@]}"; do
    echo "  📖 Generating $category documentation..."
    "$TOOLS_DIR/apprise-docs" -format markdown -category "$category" -output "$DOCS_DIR/services/$category.md" 2>/dev/null || echo "    ⚠️  No documentation available for $category category"
done

# Generate individual service documentation for major services
major_services=("discord" "slack" "email" "twilio" "rich-mobile-push" "desktop-advanced")

echo "📝 Generating individual service documentation..."

for service in "${major_services[@]}"; do
    echo "  📄 Generating $service documentation..."
    "$TOOLS_DIR/apprise-docs" -format markdown -service "$service" -output "$DOCS_DIR/services/$service.md" 2>/dev/null || echo "    ⚠️  No documentation available for $service"
done

echo "🔄 Generating migration documentation..."

# Generate migration guide
"$TOOLS_DIR/apprise-migrate" -output markdown > "$DOCS_DIR/migration/README.md"

# Generate individual service migration guides
migration_services=("discord" "slack" "email" "twilio")

for service in "${migration_services[@]}"; do
    echo "  🔄 Generating $service migration guide..."
    "$TOOLS_DIR/apprise-migrate" -guide "$service" -output markdown > "$DOCS_DIR/migration/$service.md" 2>/dev/null || echo "    ⚠️  No migration guide available for $service"
done

echo "🌐 Generating JSON documentation..."

# Generate JSON documentation for API reference
"$TOOLS_DIR/apprise-docs" -format json -output "$DOCS_DIR/services/api.json"

echo "🔗 Creating documentation index..."

# Create main documentation index
cat > "$DOCS_DIR/README.md" << EOF
# Apprise-Go Documentation

Welcome to the comprehensive documentation for Apprise-Go, a high-performance notification delivery library for Go.

## 📚 Documentation Sections

### Service Documentation
- [**Complete Service Reference**](services/README.md) - All supported notification services
- [**Service Categories**](services/) - Services organized by category
- [**API Reference**](services/api.json) - Machine-readable service documentation

### Category Documentation
$(for category in messaging email sms mobile desktop social; do
    if [[ -f "$DOCS_DIR/services/$category.md" ]]; then
        echo "- [$(echo $category | tr '[:lower:]' '[:upper:]')](services/$category.md)"
    fi
done)

### Individual Service Documentation
$(for service in discord slack email twilio rich-mobile-push desktop-advanced; do
    if [[ -f "$DOCS_DIR/services/$service.md" ]]; then
        echo "- [$service](services/$service.md)"
    fi
done)

### Migration Documentation
- [**Migration Guide**](migration/README.md) - Complete guide for migrating from Python Apprise
$(for service in discord slack email twilio; do
    if [[ -f "$DOCS_DIR/migration/$service.md" ]]; then
        echo "- [$service Migration](migration/$service.md)"
    fi
done)

## 🚀 Quick Start

### Installation
\`\`\`bash
go get github.com/scttfrdmn/apprise-go/apprise
\`\`\`

### Basic Usage
\`\`\`go
package main

import (
    "github.com/scttfrdmn/apprise-go/apprise"
)

func main() {
    // Create Apprise instance
    app := apprise.New()
    
    // Add notification services
    app.Add("discord://webhook_id/webhook_token")
    app.Add("slack://token/token/token/#general")
    
    // Send notification
    responses := app.Notify("Hello", "This is a test message", apprise.NotifyTypeInfo)
    
    // Check responses
    for _, response := range responses {
        if response.Error != nil {
            println("Error:", response.Error.Error())
        } else {
            println("Success:", response.Service)
        }
    }
}
\`\`\`

## 🎯 Key Features

- **60+ Notification Services** - Discord, Slack, Email, SMS, Push notifications, and more
- **High Performance** - Native Go concurrency and optimized delivery
- **Configuration Templating** - Environment-based configuration with templates  
- **Rich Mobile Push** - Advanced push notifications with actions and rich content
- **Migration Tools** - Easy migration from Python Apprise
- **Enterprise Ready** - Docker containers, Kubernetes manifests, and monitoring

## 📖 Documentation Tools

The documentation is generated using custom tools:

- \`cmd/apprise-docs\` - Generate service documentation
- \`cmd/apprise-migrate\` - Migration assistance and validation
- \`scripts/generate-docs.sh\` - Complete documentation generation

To regenerate documentation:
\`\`\`bash
./scripts/generate-docs.sh
\`\`\`

## 🤝 Contributing

See the main repository for contribution guidelines and development information.

## 📄 License

This project is licensed under the MIT License.
EOF

echo "📊 Generating documentation statistics..."

# Count documentation files and services
service_docs=$(find "$DOCS_DIR/services" -name "*.md" | wc -l | tr -d ' ')
migration_docs=$(find "$DOCS_DIR/migration" -name "*.md" | wc -l | tr -d ' ')
total_files=$(find "$DOCS_DIR" -name "*.md" -o -name "*.json" | wc -l | tr -d ' ')

echo "✅ Documentation generation complete!"
echo ""
echo "📈 Statistics:"
echo "  📄 Total files: $total_files"
echo "  🔧 Service docs: $service_docs"
echo "  🔄 Migration docs: $migration_docs"
echo "  📁 Output directory: $DOCS_DIR"
echo ""
echo "🔍 Generated files:"
find "$DOCS_DIR" -type f | sort

echo ""
echo "🌟 Next steps:"
echo "  1. Review generated documentation in $DOCS_DIR/"
echo "  2. Customize content as needed"
echo "  3. Add to version control"
echo "  4. Deploy to documentation site"