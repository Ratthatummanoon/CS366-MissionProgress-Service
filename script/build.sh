#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$PROJECT_ROOT/src/backend"
BUILD_DIR="$PROJECT_ROOT/terraform/build"

echo "=== Building Lambda functions ==="

# Create build output directory
mkdir -p "$BUILD_DIR"

# List of Lambda functions to build
FUNCTIONS=("report-progress" "get-mission" "authorizer")

for func in "${FUNCTIONS[@]}"; do
    echo "--- Building $func ---"
    cd "$BACKEND_DIR"

    # Cross-compile for AWS Lambda (Linux AMD64)
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o "$BUILD_DIR/$func/bootstrap" "./cmd/$func/"

    # Create zip file
    cd "$BUILD_DIR/$func"
    zip -j "$BUILD_DIR/$func.zip" bootstrap

    # Clean up unzipped binary
    rm -rf "$BUILD_DIR/$func"

    echo "--- $func built successfully ---"
done

echo ""
echo "=== Build complete ==="
echo "Zip files in $BUILD_DIR:"
ls -la "$BUILD_DIR"/*.zip
