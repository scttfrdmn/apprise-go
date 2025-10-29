#!/bin/bash
# Script to sync GitHub labels from labels.yml
# Usage: .github/sync-labels.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LABELS_FILE="$SCRIPT_DIR/labels.yml"

echo "🏷️  Syncing GitHub labels from $LABELS_FILE"
echo ""

# Parse YAML and create labels (requires yq for better parsing, but we'll use grep/sed for simplicity)
# Extract name, color, and description from labels.yml

while IFS= read -r line; do
    if [[ $line =~ ^-\ name:\ \"(.*)\" ]]; then
        name="${BASH_REMATCH[1]}"
        read -r line # color line
        if [[ $line =~ color:\ \"(.*)\" ]]; then
            color="${BASH_REMATCH[1]}"
        fi
        read -r line # description line
        if [[ $line =~ description:\ \"(.*)\" ]]; then
            description="${BASH_REMATCH[1]}"
        fi

        # Create or update label
        if gh label list | grep -q "^$name	"; then
            echo "  Updating label: $name"
            gh label edit "$name" --color "$color" --description "$description" 2>/dev/null || echo "    ⚠️  Failed to update $name"
        else
            echo "  Creating label: $name"
            gh label create "$name" --color "$color" --description "$description" 2>/dev/null || echo "    ⚠️  Failed to create $name"
        fi
    fi
done < "$LABELS_FILE"

echo ""
echo "✅ Label sync complete!"
echo ""
echo "To list all labels: gh label list"
