#!/bin/bash
set -euo pipefail

# Sign + notarize the Wails AgentDesk.app and DMG for a given arch.
# Usage: ./scripts/sign-and-notarize.sh [arm64|amd64]
#
# Required environment variables:
#   SIGNING_IDENTITY — e.g., "Developer ID Application: Name (TEAMID)"
#   APPLE_ID         — Apple ID email
#   APP_PASSWORD     — app-specific password from appleid.apple.com
#   TEAM_ID          — Apple Developer Team ID

ARCH="${1:-arm64}"
APP_NAME="AgentDesk"
DMG_NAME="AgentDesk-macOS-${ARCH}"
BUILD_DIR="build/bin"
DMG_DIR="build/dmg"
APP_PATH="${BUILD_DIR}/${APP_NAME}.app"
DMG_PATH="${DMG_DIR}/${DMG_NAME}.dmg"
ENTITLEMENTS="build/darwin/entitlements.plist"

: "${SIGNING_IDENTITY:?SIGNING_IDENTITY is required}"
: "${APPLE_ID:?APPLE_ID is required}"
: "${APP_PASSWORD:?APP_PASSWORD is required}"
: "${TEAM_ID:?TEAM_ID is required}"

if [ ! -d "${APP_PATH}" ]; then
    echo "ERROR: ${APP_PATH} not found. Run 'wails build' first."
    exit 1
fi

echo "=== Code signing ${APP_NAME}.app (${ARCH}) ==="
codesign --force --deep --options runtime \
    --sign "${SIGNING_IDENTITY}" \
    --timestamp \
    --entitlements "${ENTITLEMENTS}" \
    "${APP_PATH}"
echo "  ${APP_NAME}.app signed"

codesign --verify --deep --strict --verbose=2 "${APP_PATH}" 2>&1 | tail -3
echo "  Signature verified"

echo "=== Creating DMG ==="
rm -rf "${DMG_DIR}"
mkdir -p "${DMG_DIR}"

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
    "${DMG_PATH}" \
    "${APP_PATH}" || true

if [ ! -f "${DMG_PATH}" ]; then
    echo "ERROR: DMG not created at ${DMG_PATH}"
    exit 1
fi

echo "=== Signing DMG ==="
codesign --force --sign "${SIGNING_IDENTITY}" --timestamp "${DMG_PATH}"
echo "  DMG signed"

echo "=== Notarizing ==="
xcrun notarytool submit "${DMG_PATH}" \
    --apple-id "${APPLE_ID}" \
    --password "${APP_PASSWORD}" \
    --team-id "${TEAM_ID}" \
    --wait

echo "=== Stapling ==="
xcrun stapler staple "${DMG_PATH}"
echo "  Notarization complete"

echo ""
echo "=== Done ==="
ls -lh "${DMG_PATH}"
