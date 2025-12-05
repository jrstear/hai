import 'package:flutter_riverpod/flutter_riverpod.dart';

/// App settings model
class AppSettings {
  final String apiBaseUrl;
  final String? limitlessApiKey;
  final String? recordingDeviceId;

  AppSettings({
    required this.apiBaseUrl,
    this.limitlessApiKey,
    this.recordingDeviceId,
  });

  AppSettings copyWith({
    String? apiBaseUrl,
    String? limitlessApiKey,
    String? recordingDeviceId,
  }) {
    return AppSettings(
      apiBaseUrl: apiBaseUrl ?? this.apiBaseUrl,
      limitlessApiKey: limitlessApiKey ?? this.limitlessApiKey,
      recordingDeviceId: recordingDeviceId ?? this.recordingDeviceId,
    );
  }
}

/// Settings provider
/// 
/// TODO: Persist to secure storage (flutter_secure_storage for mobile, shared_preferences for web)
final settingsProvider = StateProvider<AppSettings>((ref) {
  return AppSettings(
    apiBaseUrl: 'http://localhost:8080',
  );
});

