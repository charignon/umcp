#!/bin/bash
set -e

VERSION="$1"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 1.0.0"
    exit 1
fi

# Update version in main.go
sed -i '' "s/var version = .*/var version = \"$VERSION\"/" main.go

# Run tests
make test

# Commit version bump
git add main.go
git commit -m "Bump version to v$VERSION"

# Create and push tag
git tag "v$VERSION"
git push origin master
git push origin "v$VERSION"

echo "✅ Released v$VERSION"
echo ""
echo "Next steps:"
echo "1. Go to https://github.com/charignon/umcp/releases/new"
echo "2. Create release for tag v$VERSION"
echo "3. Calculate SHA:"
echo "   curl -sL https://github.com/charignon/umcp/archive/refs/tags/v$VERSION.tar.gz | shasum -a 256"
echo "4. Update umcp.rb with the SHA256"
