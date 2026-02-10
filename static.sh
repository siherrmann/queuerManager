#!/bin/bash
set -e

# Download package queuerManager for copying static assets for the frontend
cd "$(dirname "$0")"
MODULE="github.com/siherrmann/queuerManager"
go mod download

# Get module path
MOD_PATH=$(go list -m -f '{{.Dir}}' $MODULE)
if [ -z "$MOD_PATH" ] || [ ! -d "$MOD_PATH" ]; then
    CACHE_DIR=$(go env GOMODCACHE)
    VERSION=$(go list -m -f '{{.Version}}' $MODULE)
    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine module version."
        exit 1
    fi

    ESCAPED_MODULE="github.com/siherrmann/queuer!manager"
    MOD_PATH="$CACHE_DIR/$ESCAPED_MODULE@$VERSION"
fi

if [ -z "$MOD_PATH" ] || [ ! -d "$MOD_PATH" ]; then
    echo "Error: Could not resolve module path for $MODULE"
    exit 1
fi

echo "Module path found: $MOD_PATH"

# Copy static assets from module to manager's static directory
TARGET_DIR="./view/static"
SOURCE_DIR="$MOD_PATH/view/static"
if [ ! -d "$SOURCE_DIR" ]; then
    echo "Error: 'view/static' directory not found in module at: $SOURCE_DIR"
    exit 1
fi
mkdir -p "$TARGET_DIR"

if [ -d "$TARGET_DIR" ]; then
    chmod -R u+w "$TARGET_DIR"
fi
cp -Rf "$SOURCE_DIR/"* "$TARGET_DIR/" || true
chmod -R u+w "$TARGET_DIR"

echo "Successfully copied static assets."