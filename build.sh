#!/bin/bash

# Build script for NetDispatch
# Usage: ./build.sh [version]
# Example: ./build.sh v1.0.0

set -e

# Get version from argument or use default
VERSION=${1:-dev}

# Get git info
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Building NetDispatch ${VERSION}"
echo "  Git Commit: ${GIT_COMMIT}"
echo "  Build Date: ${BUILD_DATE}"

# Build frontend
echo "Building frontend..."
cd web
npm run build
cd ..

# Build backend with version info
echo "Building backend..."
go build -ldflags "-s -w \
    -X netdispatch/pkg/version.Version=${VERSION} \
    -X netdispatch/pkg/version.GitCommit=${GIT_COMMIT} \
    -X netdispatch/pkg/version.BuildDate=${BUILD_DATE}" \
    -o netdispatch.exe ./cmd/netdispatch

echo "Build complete: netdispatch.exe"
ls -la netdispatch.exe
