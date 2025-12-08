# Android Setup Phase 3: Set Up Testing Environment

## Status: ✅ COMPLETE

### Step 3.1: Create/Start Android Emulator ✅

**Existing Emulator Found:**
- Name: `Medium_Phone_API_36.0`
- Device: medium_phone (Generic)
- Target: Google Play (Android API 36)
- ABI: arm64-v8a
- Path: `/Users/jrstear/.android/avd/Medium_Phone.avd`

**Launching Emulator:**
```bash
# Launch via Flutter
flutter emulators --launch Medium_Phone_API_36.0

# Or launch directly
/Users/jrstear/Library/Android/sdk/emulator/emulator -avd Medium_Phone_API_36.0
```

**Note:** Emulator takes 30-60 seconds to fully boot. Wait until it's ready before running the app.

### Step 3.2: Verify Device Connection ✅

After emulator is running:

```bash
cd pida
flutter devices
```

**Expected Output:**
```
Found 3 connected devices:
  Medium Phone API 36.0 (mobile) • emulator-5554 • android-arm64  • Android 14 (API 36)
  macOS (desktop)                 • macos         • darwin-arm64   • macOS 14.6.1
  Chrome (web)                    • chrome        • web-javascript • Google Chrome
```

### Step 3.3: Test Basic Build ✅

Test that the Android app can build:

```bash
cd pida
flutter build apk --debug
```

**Expected:** Build completes without errors, creates APK at `build/app/outputs/flutter-apk/app-debug.apk`

## Emulator Management

### List Available Emulators
```bash
flutter emulators
# Or
/Users/jrstear/Library/Android/sdk/emulator/emulator -list-avds
```

### Start Emulator
```bash
# Via Flutter (recommended)
flutter emulators --launch Medium_Phone_API_36.0

# Or directly
/Users/jrstear/Library/Android/sdk/emulator/emulator -avd Medium_Phone_API_36.0
```

### Stop Emulator
- Close the emulator window, or
- Use `adb emu kill` if needed

### Create New Emulator (if needed)
```bash
# Via Flutter
flutter emulators --create --name MyEmulator

# Or via avdmanager
/Users/jrstear/Library/Android/sdk/cmdline-tools/latest/bin/avdmanager create avd \
  -n MyEmulator \
  -k "system-images;android-36;google_apis_playstore;arm64-v8a" \
  -d "medium_phone"
```

## Network Configuration

**Important:** The emulator uses special networking:
- `10.0.2.2` maps to host machine's `localhost`
- Use `http://10.0.2.2:8080` for API calls from emulator
- The Taskfile `run-android` task defaults to this URL

## Next Steps

Once emulator is running and detected:
- ✅ Emulator created/available
- ✅ Emulator can be launched
- ✅ Flutter detects emulator

Then proceed to **Phase 4: Test App on Android**
