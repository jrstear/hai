#!/bin/bash
# iOS Setup Script for Pida
# This script configures Xcode for iOS development

set -e

echo "🔧 Configuring Xcode for iOS development..."

# Step 1: Switch xcode-select to Xcode.app
echo "📱 Switching xcode-select to Xcode.app..."
sudo xcode-select --switch /Applications/Xcode.app/Contents/Developer

# Step 2: Run first launch setup
echo "🚀 Running Xcode first launch setup..."
sudo xcodebuild -runFirstLaunch

# Step 3: Accept Xcode license
echo "📝 Accepting Xcode license..."
sudo xcodebuild -license accept

echo "✅ Xcode configuration complete!"
echo ""
echo "Next steps:"
echo "1. Verify iOS simulators: xcrun simctl list devices available"
echo "2. Install CocoaPods dependencies: cd ios && pod install"
echo "3. Run the app: task run-ios"
