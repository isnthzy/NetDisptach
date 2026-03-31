#!/bin/bash

# Release script for NetDispatch
# Usage: ./release.sh v1.0.0
# This script builds and creates a GitHub release

set -e

VERSION=${1}

if [ -z "$VERSION" ]; then
    echo "Usage: ./release.sh <version>"
    echo "Example: ./release.sh v1.0.0"
    exit 1
fi

# Remove 'v' prefix if present for version variable
VERSION_NUM=${VERSION#v}

echo "=== Releasing NetDispatch ${VERSION} ==="

# Build the executable
./build.sh ${VERSION_NUM}

# Commit any changes
git add -A
git commit -m "chore: release ${VERSION}" || echo "No changes to commit"

# Tag the release
git tag -a ${VERSION} -m "Release ${VERSION}"
git push origin main --tags

# Create GitHub release
gh release create ${VERSION} ./netdispatch.exe \
    --title "NetDispatch ${VERSION}" \
    --notes "## NetDispatch ${VERSION}

### 更新日志
-

### 默认端口
- HTTP/HTTPS 代理: 8009
- SOCKS5 代理: 8010
- API/Web 控制台: 9090

### 安装
下载 \`netdispatch.exe\` 并运行即可。"

echo "=== Release ${VERSION} created successfully ==="
echo "https://github.com/isnthzy/NetDisptach/releases/tag/${VERSION}"
