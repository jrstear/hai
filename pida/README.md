# Pida - Flutter App

User-facing Flutter application for the Hai audio lifelog processing system.

## Overview

Pida is a cross-platform Flutter application (web, Android, iOS) that provides a user interface for viewing and managing audio lifelogs, contacts, and conversations.

## Features

- **People**: Contact management
- **Calendar**: Day/week/month views of conversations
- **Conversation**: Detailed view of individual conversations with audio playback
- **Todo**: Task management (coming soon)

## Prerequisites

- Flutter SDK 3.0.0+
- Dart SDK 3.0.0+
- Hai API server running (default: http://localhost:8080)

## Setup

1. Install dependencies:
```bash
flutter pub get
```

2. Generate code (for freezed and json_serializable):
```bash
flutter pub run build_runner build --delete-conflicting-outputs
```

Or use the Taskfile:
```bash
task setup
```

## Environment Variables

The app uses build-time environment variables for configuration:

- `PIDA_API_URL`: Base URL for the Hai API server (default: `http://localhost:8080`)
- `LIMITLESS_API_KEY`: Limitless API key for audio streaming
- `RECORDING_DEVICE_ID`: Device identifier (e.g., pendant ID, Plaud device ID)

These are set via `--dart-define` flags when running or building:

```bash
flutter run -d chrome \
  --dart-define=PIDA_API_URL=http://localhost:8080 \
  --dart-define=LIMITLESS_API_KEY=your_key_here \
  --dart-define=RECORDING_DEVICE_ID=your_device_id
```

**Note**: In the future, these will be user-settable via a settings screen with secure storage.

## Running

### Web

```bash
task run
```

Or manually:
```bash
flutter run -d chrome
```

### Android

```bash
task run-android
```

### iOS

```bash
task run-ios
```

## Building

### Web

```bash
task build
```

Output: `build/web/`

### Android

```bash
task build-android
```

Output: `build/app/outputs/flutter-apk/`

### iOS

```bash
task build-ios
```

Output: `build/ios/`

## Development

### Code Generation

When models change, regenerate code:

```bash
task generate
```

Or watch for changes:

```bash
task generate-watch
```

### Formatting

```bash
task format
```

### Linting

```bash
task analyze
```

### Testing

```bash
task test
```

## Project Structure

```
lib/
├── main.dart                 # App entry point
├── models/                   # Data models (Contact, Speaker, Recording, Lifelog)
├── providers/                # Riverpod providers (state management)
├── routes/                   # GoRouter configuration
├── screens/                  # Screen widgets (People, Calendar, Conversation, Todo)
├── services/                 # API client, audio service, env config
├── utils/                    # Utilities (error handling, loading states)
└── widgets/                  # Reusable widgets (navigation, avatars, etc.)
```

## License

AGPL-3.0 (same as parent repository)

