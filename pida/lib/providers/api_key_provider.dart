import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/services/env_config.dart';

/// API key provider
/// 
/// Currently reads from build-time environment variables.
/// In the future, this will read from secure storage (user-settable).
/// 
/// Related to future schema changes: recording_device_id field in recordings table
/// for multi-device and multi-user support.
final apiKeyProvider = Provider<String?>((ref) {
  // For now, read from environment variables at build time
  // TODO: Read from secure storage when user-settable settings are implemented
  return EnvConfig.limitlessApiKey;
});

/// Recording device ID provider
/// 
/// Currently reads from build-time environment variables.
/// In the future, this will be:
/// - Stored in database schema (recording_device_id field)
/// - User-selectable if they have multiple devices
/// - Used for multi-user data isolation
final recordingDeviceIdProvider = Provider<String?>((ref) {
  // For now, read from environment variables at build time
  // TODO: Read from database/user settings when schema is updated
  return EnvConfig.recordingDeviceId;
});

/// Provider that checks if API key is configured
final hasApiKeyProvider = Provider<bool>((ref) {
  return ref.watch(apiKeyProvider) != null;
});

/// Provider that checks if device ID is configured
final hasDeviceIdProvider = Provider<bool>((ref) {
  return ref.watch(recordingDeviceIdProvider) != null;
});

