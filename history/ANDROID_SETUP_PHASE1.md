# Android Setup Phase 1: Fix Toolchain Setup

## Current Status

✅ **Installed:**
- Android Studio is installed
- Android SDK at `/Users/jrstear/Library/Android/sdk`
- Android SDK version 35.0.1
- Emulator version 35.5.10.0
- Other SDK components (build-tools, platform-tools, platforms, etc.)

❌ **Missing:**
- cmdline-tools component (required for `sdkmanager` and license acceptance)
- Android licenses not accepted (can't proceed without cmdline-tools)

## Step 1.1: Install Android Command-Line Tools

**Status:** Manual download attempted but had extraction issues. **Recommended:** Use Android Studio GUI method.

### Option A: Via Android Studio (Recommended - Most Reliable)

1. **Open Android Studio**
   ```bash
   open -a "Android Studio"
   ```

2. **Open SDK Manager**
   - Click "More Actions" or "Configure" (bottom right)
   - Select "SDK Manager"
   - Or: Tools > SDK Manager

3. **Install Command-Line Tools**
   - Go to "SDK Tools" tab
   - Check "Android SDK Command-line Tools (latest)"
   - Click "Apply" and "OK"
   - Wait for installation to complete

4. **Verify Installation**
   ```bash
   ls -la /Users/jrstear/Library/Android/sdk/cmdline-tools/
   # Should show a directory (e.g., "latest" or version number)
   ```

### Option B: Manual Installation (Alternative - May Have Issues)

**Note:** Manual download/installation can have zip extraction issues. Android Studio method is more reliable.

If you prefer command-line installation:

1. **Download Command-Line Tools**
   ```bash
   cd /tmp
   curl -L -o cmdline-tools.zip https://dl.google.com/android/repository/commandlinetools-mac-11076708_latest.zip
   # Verify download (should be ~146MB)
   ls -lh cmdline-tools.zip
   ```

2. **Extract and Install**
   ```bash
   mkdir -p /Users/jrstear/Library/Android/sdk/cmdline-tools
   # Extract (may show warnings but should work)
   unzip -q cmdline-tools.zip -d /Users/jrstear/Library/Android/sdk/cmdline-tools
   
   # Check what was extracted
   ls -la /Users/jrstear/Library/Android/sdk/cmdline-tools/
   
   # If there's a "cmdline-tools" subdirectory, rename it to "latest"
   if [ -d "/Users/jrstear/Library/Android/sdk/cmdline-tools/cmdline-tools" ]; then
     mv /Users/jrstear/Library/Android/sdk/cmdline-tools/cmdline-tools /Users/jrstear/Library/Android/sdk/cmdline-tools/latest
   fi
   ```

3. **Verify Installation**
   ```bash
   /Users/jrstear/Library/Android/sdk/cmdline-tools/latest/bin/sdkmanager --version
   # Should show version number
   ```

**If manual installation fails:** Use Android Studio method (Option A) instead.

## Step 1.2: Accept Android SDK Licenses

Once cmdline-tools are installed:

```bash
cd /Users/jrstear/mine/git/hai/pida
flutter doctor --android-licenses
```

**Expected behavior:**
- Will show each license
- Type `y` and press Enter for each license
- Should accept all licenses automatically if using `-y` flag (if supported)

**Alternative (if interactive prompts are problematic):**
```bash
# Accept all licenses non-interactively
yes | flutter doctor --android-licenses
```

## Step 1.3: Install Required Android SDK Components

After accepting licenses, install the Android SDK platform and build tools that Flutter requires:

```bash
# Install Android SDK Platform 36
/Users/jrstear/Library/Android/sdk/cmdline-tools/latest/bin/sdkmanager "platforms;android-36"

# Install Android BuildTools (Flutter may require specific version)
/Users/jrstear/Library/Android/sdk/cmdline-tools/latest/bin/sdkmanager "build-tools;28.0.3"
```

**Note:** Flutter may also use newer build-tools versions. The sdkmanager will install what's needed.

## Step 1.4: Verify Complete Setup

After installing cmdline-tools, accepting licenses, and installing SDK components:

```bash
cd /Users/jrstear/mine/git/hai/pida
flutter doctor -v
```

**Expected Output:**
```
[✓] Android toolchain - develop for Android devices (Android SDK version 36.0.0)
    • Android SDK at /Users/jrstear/Library/Android/sdk
    • Platform android-36, build-tools 36.0.0
    • Java binary at: /path/to/java
    • Java version: OpenJDK Runtime Environment (build 17+ or 21+)
    • ANDROID_HOME = /Users/jrstear/Library/Android/sdk
    • cmdline-tools component is installed
    • All Android licenses accepted
```

**If still showing errors:**
- Verify cmdline-tools path: `ls -la $ANDROID_HOME/cmdline-tools/`
- Check ANDROID_HOME is set: `echo $ANDROID_HOME`
- Try restarting terminal/IDE after installation

## Verification Commands

```bash
# Check cmdline-tools installation
ls -la /Users/jrstear/Library/Android/sdk/cmdline-tools/

# Check sdkmanager is available
/Users/jrstear/Library/Android/sdk/cmdline-tools/latest/bin/sdkmanager --version

# Check Flutter Android setup
cd /Users/jrstear/mine/git/hai/pida
flutter doctor -v | grep -A 10 "Android toolchain"

# List available Android devices/emulators
flutter devices
```

## Status: ✅ COMPLETE

Phase 1 is complete:
- ✅ cmdline-tools installed
- ✅ Licenses accepted
- ✅ Android SDK 36 and BuildTools installed
- ✅ Flutter doctor shows Android toolchain as fully configured

**Completed on:** 2025-12-07

## Next Steps

Proceed to **Phase 3: Set Up Testing Environment** (Phase 2 is already complete - Android project configured)
