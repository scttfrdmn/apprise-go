#!/bin/bash

# Script to check for upstream Apprise version updates
# This helps maintain version synchronization with the original Python project

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VERSION_FILE="$PROJECT_ROOT/VERSION"

echo "🔍 Checking upstream Apprise version..."

# Get current version from VERSION file
if [[ -f "$VERSION_FILE" ]]; then
    CURRENT_VERSION=$(cat "$VERSION_FILE")
    CURRENT_UPSTREAM=$(echo "$CURRENT_VERSION" | cut -d'-' -f1)
    CURRENT_PORT=$(echo "$CURRENT_VERSION" | cut -d'-' -f2)
    echo "📋 Current version: $CURRENT_VERSION"
    echo "   └─ Upstream: v$CURRENT_UPSTREAM"
    echo "   └─ Port revision: $CURRENT_PORT"
else
    echo "❌ VERSION file not found at $VERSION_FILE"
    exit 1
fi

# Check upstream version using GitHub API
echo ""
echo "🌐 Fetching latest upstream version..."

if command -v gh >/dev/null 2>&1; then
    # Use GitHub CLI if available
    UPSTREAM_VERSION=$(gh api repos/caronc/apprise/releases/latest --jq '.tag_name' | sed 's/^v//')
    UPSTREAM_URL=$(gh api repos/caronc/apprise/releases/latest --jq '.html_url')
    PUBLISHED_AT=$(gh api repos/caronc/apprise/releases/latest --jq '.published_at')
elif command -v curl >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
    # Fallback to curl + jq
    UPSTREAM_DATA=$(curl -s https://api.github.com/repos/caronc/apprise/releases/latest)
    UPSTREAM_VERSION=$(echo "$UPSTREAM_DATA" | jq -r '.tag_name' | sed 's/^v//')
    UPSTREAM_URL=$(echo "$UPSTREAM_DATA" | jq -r '.html_url')
    PUBLISHED_AT=$(echo "$UPSTREAM_DATA" | jq -r '.published_at')
elif command -v curl >/dev/null 2>&1; then
    # Basic curl fallback with sed
    UPSTREAM_DATA=$(curl -s https://api.github.com/repos/caronc/apprise/releases/latest)
    UPSTREAM_VERSION=$(echo "$UPSTREAM_DATA" | sed -n 's/.*"tag_name": *"v\?\([^"]*\)".*/\1/p' | head -1)
    UPSTREAM_URL=$(echo "$UPSTREAM_DATA" | sed -n 's/.*"html_url": *"\([^"]*\)".*/\1/p' | head -1)
    PUBLISHED_AT=$(echo "$UPSTREAM_DATA" | sed -n 's/.*"published_at": *"\([^"]*\)".*/\1/p' | head -1)
else
    echo "❌ Neither 'gh' nor 'curl' found. Install GitHub CLI or curl to check upstream versions."
    exit 1
fi

if [[ -z "$UPSTREAM_VERSION" ]]; then
    echo "❌ Failed to fetch upstream version"
    exit 1
fi

echo "🎯 Latest upstream: v$UPSTREAM_VERSION"
echo "   └─ Published: $PUBLISHED_AT"
echo "   └─ URL: $UPSTREAM_URL"

# Compare versions
echo ""
if [[ "$CURRENT_UPSTREAM" == "$UPSTREAM_VERSION" ]]; then
    echo "✅ Up to date! Current version $CURRENT_VERSION tracks latest upstream v$UPSTREAM_VERSION"
    
    echo ""
    echo "📊 Port revision history:"
    if [[ -f "$PROJECT_ROOT/CHANGELOG.md" ]]; then
        echo "   (See CHANGELOG.md for revision details)"
    else
        echo "   Current port revision: $CURRENT_PORT"
    fi
    
elif [[ "$CURRENT_UPSTREAM" < "$UPSTREAM_VERSION" ]]; then
    echo "📦 New upstream version available!"
    echo "   Current: v$CURRENT_UPSTREAM (port revision $CURRENT_PORT)"
    echo "   Latest:  v$UPSTREAM_VERSION"
    echo ""
    echo "🔄 To update to the new upstream version:"
    echo "   1. Review upstream changes: $UPSTREAM_URL"
    echo "   2. Update VERSION file: echo '${UPSTREAM_VERSION}-1' > VERSION"
    echo "   3. Update version constants in apprise/version.go"
    echo "   4. Update go.mod comments"
    echo "   5. Update README.md references"
    echo "   6. Test all functionality and update port-specific features"
    echo "   7. Update CHANGELOG.md with upstream and port changes"
    echo ""
    echo "💡 Quick update commands:"
    echo "   echo '${UPSTREAM_VERSION}-1' > VERSION"
    echo "   sed -i '' 's/UpstreamVersion = \"[^\"]*\"/UpstreamVersion = \"$UPSTREAM_VERSION\"/' apprise/version.go"
    echo "   sed -i '' 's/Version = \"[^\"]*\"/Version = \"${UPSTREAM_VERSION}-1\"/' apprise/version.go"
    
else
    echo "⚠️  Current version ($CURRENT_UPSTREAM) is newer than latest upstream ($UPSTREAM_VERSION)"
    echo "   This might indicate a pre-release or development version."
fi

echo ""
echo "🔗 Useful links:"
echo "   • Upstream releases: https://github.com/caronc/apprise/releases"
echo "   • Upstream changelog: https://github.com/caronc/apprise/blob/master/CHANGELOG.md"
echo "   • Compare changes: https://github.com/caronc/apprise/compare/v$CURRENT_UPSTREAM...v$UPSTREAM_VERSION"