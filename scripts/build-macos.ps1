param(
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "dist"
$appRoot = Join-Path $dist "Agent Feishu.app"
$macosDir = Join-Path $appRoot "Contents/MacOS"
$resourcesDir = Join-Path $appRoot "Contents/Resources"

if (Test-Path $appRoot) {
    Remove-Item -Recurse -Force $appRoot
}
New-Item -ItemType Directory -Force -Path $dist, $macosDir, $resourcesDir | Out-Null

$env:CGO_ENABLED = "0"
$env:GOOS = "darwin"

$env:GOARCH = "arm64"
& go build -trimpath -ldflags="-s -w -X main.version=$Version" -o (Join-Path $dist "agent-feishu-macos-arm64") ./cmd/agent-feishu

$env:GOARCH = "amd64"
& go build -trimpath -ldflags="-s -w -X main.version=$Version" -o (Join-Path $dist "agent-feishu-macos-amd64") ./cmd/agent-feishu

Copy-Item -Force (Join-Path $dist "agent-feishu-macos-arm64") (Join-Path $macosDir "agent-feishu-arm64")
Copy-Item -Force (Join-Path $dist "agent-feishu-macos-amd64") (Join-Path $macosDir "agent-feishu-amd64")

@'
#!/bin/sh
DIR="$(cd "$(dirname "$0")" && pwd)"
if [ "$(uname -m)" = "arm64" ]; then
  BIN="$DIR/agent-feishu-arm64"
else
  BIN="$DIR/agent-feishu-amd64"
fi
SCRIPT="/tmp/agent-feishu-setup.command"
cat > "$SCRIPT" <<EOF
#!/bin/sh
"$BIN" setup
echo
echo "You can close this Terminal window."
EOF
chmod +x "$SCRIPT"
exec open -a Terminal "$SCRIPT"
'@ | Set-Content -Encoding UTF8 (Join-Path $macosDir "launcher")

@'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "https://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>launcher</string>
  <key>CFBundleIdentifier</key>
  <string>com.agentfeishu.notifier</string>
  <key>CFBundleName</key>
  <string>Agent Feishu</string>
  <key>CFBundleDisplayName</key>
  <string>Agent Feishu</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>0.1.0</string>
  <key>CFBundleVersion</key>
  <string>1</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
</dict>
</plist>
'@ | Set-Content -Encoding UTF8 (Join-Path $appRoot "Contents/Info.plist")

$zip = Join-Path $dist "agent-feishu-macos.zip"
if (Test-Path $zip) {
    Remove-Item $zip -Force
}
Compress-Archive -Force -Path (Join-Path $dist "agent-feishu-macos-arm64"), (Join-Path $dist "agent-feishu-macos-amd64"), $appRoot -DestinationPath $zip

Write-Host "Built:"
Write-Host "  $zip"
Write-Host "  $(Join-Path $dist "agent-feishu-macos-arm64")"
Write-Host "  $(Join-Path $dist "agent-feishu-macos-amd64")"
