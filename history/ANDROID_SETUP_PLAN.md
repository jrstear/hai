# Android Build and Testing Setup Plan

## Overview

This document outlines the steps needed to set up Android build and testing for the Pida Flutter app.

## Current Status

### ✅ Already Configured
- Android SDK installed (version 35.0.1) at `/Users/jrstear/Library/Android/sdk`
- Android emulator available (version 35.5.10.0)
- Flutter Android support enabled
- Basic Android project structure exists
- Taskfile tasks exist (`run-android`, `build-android`)
- `local.properties` configured with SDK path

### ❌ Missing/Incomplete
- cmdline-tools component not installed
- Android SDK licenses not accepted
- Network configuration for emulator (API URL)
- AndroidManifest.xml permissions verification
- Testing documentation

## Step-by-Step Plan

### Phase 1: Fix Android Toolchain Setup

#### Step 1.1: Install Android Command-Line Tools
**Options:**
- **Option A (Recommended):** Install via Android Studio
  1. Open Android Studio
  2. Tools > SDK Manager
  3. SDK Tools tab
  4. Check "Android SDK Command-line Tools (latest)"
  5. Apply and install

- **Option B:** Install standalone command-line tools
  1. Download from: https://developer.android.com/studio#command-line-tools-only
  2. Extract to `$ANDROID_HOME/cmdline-tools/latest/`
  3. Add to PATH if needed

**Verification:**
```bash
flutter doctor -v
# Should show cmdline-tools as installed
```

#### Step 1.2: Accept Android SDK Licenses
```bash
flutter doctor --android-licenses
# Press 'y' to accept each license
```

**Verification:**
```bash
flutter doctor
# Android toolchain should show no errors
```

#### Step 1.3: Verify Complete Setup
```bash
cd pida
flutter doctor -v
# Check Android toolchain section
```

**Expected Output:**
```
[✓] Android toolchain - develop for Android devices (Android SDK version 35.0.1)
    • Android SDK at /Users/jrstear/Library/Android/sdk
    • Platform android-35, build-tools 35.0.1
    • Java binary at: ...
    • Java version: ...
    • ANDROID_HOME = /Users/jrstear/Library/Android/sdk
    • cmdline-tools component is installed
    • All Android licenses accepted
```

### Phase 2: Configure Android Project

#### Step 2.1: Verify AndroidManifest.xml
**Location:** `pida/android/app/src/main/AndroidManifest.xml`

**Check for:**
- Internet permission (required for API calls)
- Audio playback permissions (if needed)
- Network security config (for HTTP if not using HTTPS)

**Required Permissions:**
```xml
<uses-permission android:name="android.permission.INTERNET"/>
```

#### Step 2.2: Configure Network for Emulator
**Issue:** Android emulator can't access `localhost:8080` directly

**Solutions:**
1. **Use emulator's host alias:** `http://10.0.2.2:8080` (recommended for localhost)
2. **Use machine's IP address:** `http://192.168.x.x:8080` (for network access)
3. **Use actual device:** Connect physical Android device via USB

**Update Taskfile or create Android-specific config:**
- Add `ANDROID_API_URL` variable
- Default to `10.0.2.2:8080` for emulator
- Allow override for physical devices

#### Step 2.3: Verify Build Configuration
**Check:** `pida/android/app/build.gradle` (if exists)
- Min SDK version
- Target SDK version
- Compile SDK version
- Application ID

### Phase 3: Set Up Testing Environment

#### Step 3.1: Create/Start Android Emulator
**Option A: Via Android Studio**
1. Open Android Studio
2. Tools > Device Manager
3. Create Virtual Device (if none exists)
4. Select device (e.g., Pixel 5)
5. Select system image (API 33+ recommended)
6. Start emulator

**Option B: Via Command Line**
```bash
# List available emulators
emulator -list-avds

# Start emulator
emulator -avd <avd_name>
```

#### Step 3.2: Verify Device Connection
```bash
cd pida
flutter devices
# Should show Android device/emulator
```

**Expected Output:**
```
2 connected devices:
sdk gphone64 arm64 (mobile) • emulator-5554 • android-arm64  • Android 13 (API 33)
Chrome (web)                • chrome        • web-javascript • Google Chrome
```

#### Step 3.3: Test Basic Build
```bash
cd pida
flutter build apk --debug
# Should complete without errors
```

### Phase 4: Test App on Android

#### Step 4.1: Run App on Emulator/Device
```bash
cd pida
# Set API URL for emulator
task run-android PIDA_API_URL=http://10.0.2.2:8080
```

**Or manually:**
```bash
flutter run -d android \
  --dart-define=PIDA_API_URL=http://10.0.2.2:8080 \
  --dart-define=LIMITLESS_API_KEY=your_key \
  --dart-define=RECORDING_DEVICE_ID=your_device_id
```

#### Step 4.2: Test Key Features
- [ ] App launches successfully
- [ ] Navigation works (People, Calendar, Conversation screens)
- [ ] API calls work (contacts, lifelogs load)
- [ ] Audio playback works (if implemented)
- [ ] File picker works (vCard upload)
- [ ] UI renders correctly (Material Design)
- [ ] Touch interactions work

#### Step 4.3: Test Build Variants
**Debug Build:**
```bash
flutter build apk --debug
```

**Release Build:**
```bash
flutter build apk --release
```

**App Bundle (for Play Store):**
```bash
flutter build appbundle --release
```

### Phase 5: Documentation and Testing

#### Step 5.1: Create Android Testing Guide
**File:** `pida/ANDROID_TESTING.md`

**Contents:**
- Prerequisites
- Setup steps
- Running on emulator vs physical device
- Network configuration
- Common issues and solutions
- Build commands
- Testing checklist

#### Step 5.2: Update README
**File:** `pida/README.md`

**Add:**
- Android-specific setup instructions
- Network configuration notes
- Emulator vs physical device differences
- Troubleshooting section

#### Step 5.3: Create Testing Checklist
**Checklist items:**
- [ ] App installs successfully
- [ ] App launches without crashes
- [ ] All screens accessible
- [ ] API connectivity works
- [ ] Audio playback works
- [ ] File upload works
- [ ] Navigation works
- [ ] UI responsive and correct
- [ ] Performance acceptable

### Phase 6: CI/CD Considerations (Future)

#### Step 6.1: GitHub Actions (if needed)
- Set up Android build in CI
- Run tests on Android
- Build APK/App Bundle artifacts

#### Step 6.2: Signing Configuration (for release)
- Set up keystore for app signing
- Configure signing in `build.gradle`
- Document signing process

## Network Configuration Details

### Emulator Network Setup

**Problem:** Android emulator can't access `localhost:8080` on host machine.

**Solution 1: Use 10.0.2.2 (Recommended)**
- `10.0.2.2` is the emulator's special IP that maps to host's `localhost`
- Works for localhost services
- No network configuration needed

**Solution 2: Use Host Machine IP**
- Find machine's IP: `ifconfig | grep "inet "`
- Use that IP instead of localhost
- Requires API server to bind to 0.0.0.0 (not just 127.0.0.1)

**Solution 3: Use Physical Device**
- Connect Android device via USB
- Enable USB debugging
- Use actual IP address or localhost (if on same network)

### Recommended Configuration

**For Development:**
- Emulator: Use `http://10.0.2.2:8080`
- Physical device: Use machine's IP or configure port forwarding

**For Production:**
- Use actual server URL (e.g., `https://api.example.com`)

## Build Commands Reference

### Development
```bash
# Run on Android
task run-android PIDA_API_URL=http://10.0.2.2:8080

# Build debug APK
flutter build apk --debug

# Build release APK
flutter build apk --release
```

### Production
```bash
# Build App Bundle (for Play Store)
flutter build appbundle --release \
  --dart-define=PIDA_API_URL=https://api.example.com \
  --dart-define=LIMITLESS_API_KEY=... \
  --dart-define=RECORDING_DEVICE_ID=...
```

## Troubleshooting

### Common Issues

1. **"cmdline-tools component is missing"**
   - Install via Android Studio SDK Manager
   - Or download standalone command-line tools

2. **"Android license status unknown"**
   - Run: `flutter doctor --android-licenses`
   - Accept all licenses

3. **"No devices found"**
   - Start emulator: Android Studio > Device Manager
   - Or connect physical device with USB debugging enabled
   - Verify: `flutter devices`

4. **"Connection refused" or API calls fail**
   - Check API URL (use `10.0.2.2` for emulator)
   - Verify API server is running
   - Check network permissions in AndroidManifest.xml

5. **Build fails with Gradle errors**
   - Run: `cd android && ./gradlew clean`
   - Check `build.gradle` configuration
   - Verify Java/JDK version compatibility

## Next Steps

1. Complete Phase 1 (Fix toolchain setup)
2. Complete Phase 2 (Configure project)
3. Complete Phase 3 (Set up testing environment)
4. Complete Phase 4 (Test app)
5. Complete Phase 5 (Documentation)
6. Optional: Phase 6 (CI/CD)

## Related Files

- `pida/android/` - Android project files
- `pida/Taskfile.yml` - Build tasks
- `pida/README.md` - Main documentation
- `pida/pubspec.yaml` - Flutter dependencies

## References

- [Flutter Android Setup](https://docs.flutter.dev/get-started/install/macos#android-setup)
- [Android Emulator Networking](https://developer.android.com/studio/run/emulator-networking)
- [Flutter Build Modes](https://docs.flutter.dev/testing/build-modes)
