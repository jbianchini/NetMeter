#!/usr/bin/env bash
set -euo pipefail

APP_NAME="NetMeter"
BUNDLE="${APP_NAME}.app"
BIN="${APP_NAME}"

rm -rf "$BUNDLE"
mkdir -p "$BUNDLE/Contents/MacOS"
mkdir -p "$BUNDLE/Contents/Resources"

CGO_ENABLED=1 GOOS=darwin go build -o "$BUNDLE/Contents/MacOS/$BIN" .

cat > "$BUNDLE/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>${BIN}</string>
  <key>CFBundleIdentifier</key>
  <string>com.juanbianchini.NetMeter.menubar</string>
  <key>CFBundleName</key>
  <string>${APP_NAME}</string>
  <key>CFBundleDisplayName</key>
  <string>${APP_NAME}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>1.1</string>
  <key>CFBundleVersion</key>
  <string>2</string>
  <key>LSMinimumSystemVersion</key>
  <string>12.0</string>
  <key>LSUIElement</key>
  <true/>
</dict>
</plist>
PLIST

# Remove quarantine if this folder came from a downloaded zip.
xattr -dr com.apple.quarantine "$BUNDLE" 2>/dev/null || true

echo "OK: created $BUNDLE"
echo "Open it with: open $BUNDLE"
