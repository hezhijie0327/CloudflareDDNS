#!/bin/sh
# bump-version.sh — bump CloudflareDDNS version (version.go + README badge).
# Usage:
#   sh scripts/bump-version.sh patch  # 1.6.0 → 1.6.1
#   sh scripts/bump-version.sh minor  # 1.6.0 → 1.7.0
#   sh scripts/bump-version.sh major  # 1.6.0 → 2.0.0
#
# Conventions (see CLAUDE.md §Conventions):
#   Z (patch) — bug fixes, perf improvements, refactors, linter fixes
#   Y (minor) — new features, new config options
#   X (major) — breaking config/schema/API changes

set -eu

BUMP="${1:-}"
if [ -z "$BUMP" ]; then
    echo "Usage: sh scripts/bump-version.sh <patch|minor|major>" >&2
    exit 1
fi

case "$BUMP" in
    patch|minor|major) ;;
    *) echo "bump must be patch, minor, or major" >&2; exit 1 ;;
esac

# ── Parse current version from version.go ────────────────────────────────
VERSION_FILE="cmd/cloudflareddns/version.go"
CURRENT=$(grep 'Version\s*=' "$VERSION_FILE" | head -1 | sed 's/.*"\(.*\)".*/\1/')
echo "Current version: $CURRENT"

MAJOR=$(echo "$CURRENT" | cut -d. -f1)
MINOR=$(echo "$CURRENT" | cut -d. -f2)
PATCH=$(echo "$CURRENT" | cut -d. -f3)

case "$BUMP" in
    major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
    minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
    patch) PATCH=$((PATCH + 1)) ;;
esac

NEW="$MAJOR.$MINOR.$PATCH"
echo "New version:     $NEW"

# ── Bump version.go ──────────────────────────────────────────────────────
# Use [[:space:]] instead of \s for portability (macOS sed lacks \s).
if [ "$(uname)" = "Darwin" ]; then
    sed -i '' "s/Version[[:space:]]*=[[:space:]]*\"$CURRENT\"/Version     = \"$NEW\"/" "$VERSION_FILE"
else
    sed -i "s/Version[[:space:]]*=[[:space:]]*\"$CURRENT\"/Version     = \"$NEW\"/" "$VERSION_FILE"
fi
echo "Bumped $VERSION_FILE"

# ── Bump README version badge ──────────────────────────────────────────────
README="README.md"
if [ "$(uname)" = "Darwin" ]; then
    sed -i '' "s/Version-[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*-/Version-${NEW}-/" "$README"
else
    sed -i "s/Version-[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*-/Version-${NEW}-/" "$README"
fi
echo "Bumped $README"
