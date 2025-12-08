# Android Setup Phase 4: Test App on Android

## Status: ✅ BUILD SUCCESSFUL

### Step 4.0: Fix Build Issues ✅

**Issue:** `file_picker` plugin v6.1.1 was incompatible with Flutter's v2 Android embedding.

**Fix:** Updated `file_picker` to version 8.1.6 in `pubspec.yaml`:
```yaml
file_picker: ^8.1.6  # Updated from ^6.1.1
```

**Result:** Build now succeeds without errors.

### Step 4.1: Test Basic Build ✅

**Status:** Build successful!

Test that the Android app can build successfully:

```bash
cd pida
flutter build apk --debug
```

**Result:** ✅ Build completed successfully
- APK created at: `build/app/outputs/flutter-apk/app-debug.apk`
- Build time: ~74 seconds
- No errors (only warnings about obsolete Java source/target values, which are harmless)

### Step 4.2: Run App on Emulator

Once emulator is running and detected:

```bash
cd pida
# Default uses http://10.0.2.2:8080 for emulator
task run-android
```

**Or with explicit API URL:**
```bash
task run-android PIDA_API_URL=http://10.0.2.2:8080
```

**Note:** The Taskfile automatically uses `http://10.0.2.2:8080` for emulator (maps to host's localhost:8080).

### Step 4.3: Test Key Features

Once app is running on emulator, verify:

- [ ] App launches successfully
- [ ] Navigation works (People, Calendar, Conversation screens)
- [ ] API calls work (contacts, lifelogs load from API)
- [ ] Network connectivity works (emulator can reach API at 10.0.2.2:8080)
- [ ] UI renders correctly (Material Design)
- [ ] Touch interactions work
- [ ] Participant avatars display correctly
- [ ] Highlighting works (purple borders for auto-matched contacts)

### Step 4.4: Test Build Variants

**Debug Build:**
```bash
cd pida
flutter build apk --debug
```
Output: `build/app/outputs/flutter-apk/app-debug.apk`

**Release Build:**
```bash
cd pida
flutter build apk --release
```
Output: `build/app/outputs/flutter-apk/app-release.apk`

**App Bundle (for Play Store):**
```bash
cd pida
flutter build appbundle --release
```
Output: `build/app/outputs/bundle/release/app-release.aab`

## Troubleshooting

### Emulator Not Detected
- Wait 30-60 seconds for emulator to fully boot
- Check: `flutter devices`
- Verify emulator is running: Check for emulator window

### Build Fails
- Check: `flutter doctor -v` for Android toolchain issues
- Clean build: `cd pida/android && ./gradlew clean`
- Check Java version: Should be JDK 17+

### API Connection Fails
- Verify API server is running on localhost:8080
- Check emulator uses `10.0.2.2:8080` (not `localhost:8080`)
- Verify network permissions in AndroidManifest.xml

### App Crashes on Launch
- Check logs: `flutter logs` or `adb logcat`
- Verify all dependencies are installed: `flutter pub get`
- Check for missing environment variables

## Next Steps

After successful testing:
- ✅ App builds successfully
- ✅ App runs on emulator
- ✅ Key features work
- ✅ Ready for development and testing

Then proceed to **Phase 5: Documentation** (optional)
