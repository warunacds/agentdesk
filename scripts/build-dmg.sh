#!/bin/bash
set -euo pipefail

# Build a .dmg installer for Agent Desk
# Usage: ./scripts/build-dmg.sh [arm64|amd64]

ARCH="${1:-arm64}"
APP_NAME="AgentDesk"
DMG_NAME="AgentDesk-macOS-${ARCH}"
BUILD_DIR="build/bin"
DMG_DIR="build/dmg"

if [ ! -d "${BUILD_DIR}/${APP_NAME}.app" ]; then
  echo "Error: ${BUILD_DIR}/${APP_NAME}.app not found. Run 'wails build' first."
  exit 1
fi

# Clean previous DMG artifacts
rm -rf "${DMG_DIR}"
mkdir -p "${DMG_DIR}"

# Create DMG with drag-to-Applications
create-dmg \
  --volname "${APP_NAME}" \
  --volicon "build/appicon.png" \
  --window-pos 200 120 \
  --window-size 600 400 \
  --icon-size 100 \
  --icon "${APP_NAME}.app" 150 185 \
  --hide-extension "${APP_NAME}.app" \
  --app-drop-link 450 185 \
  --no-internet-enable \
  "${DMG_DIR}/${DMG_NAME}.dmg" \
  "${BUILD_DIR}/${APP_NAME}.app"

echo ""
echo "DMG created: ${DMG_DIR}/${DMG_NAME}.dmg"
ls -lh "${DMG_DIR}/${DMG_NAME}.dmg"
