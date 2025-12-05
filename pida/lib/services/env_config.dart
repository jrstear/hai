/// Environment configuration service
/// 
/// Reads API key and device ID from environment variables at build time.
/// 
/// **NOTE**: This is a temporary solution. In the future:
/// - API key and device ID will be user-settable via settings screen
/// - Device ID will be stored in the database schema (recording_device_id field)
/// - This relates to future schema changes for multi-device support
/// 
/// Environment variables:
/// - PIDA_API_URL: Base URL for the Hai API server (default: http://localhost:8080)
/// - LIMITLESS_API_KEY: Limitless API key for audio streaming
/// - RECORDING_DEVICE_ID: Device identifier (e.g., pendant ID, Plaud device ID, etc.)
/// 
/// For Flutter, these are typically set via:
/// - Web: --dart-define=PIDA_API_URL=xxx --dart-define=LIMITLESS_API_KEY=yyy --dart-define=RECORDING_DEVICE_ID=zzz
/// - Mobile: Build-time environment variables or build config files

class EnvConfig {
  /// Get API base URL from environment
  /// Defaults to http://localhost:8080 if not set
  static String get apiBaseUrl {
    const baseUrl = String.fromEnvironment(
      'PIDA_API_URL',
      defaultValue: 'http://localhost:8080',
    );
    return baseUrl;
  }

  /// Get Limitless API key from environment
  /// Returns null if not set (will need to be configured by user later)
  static String? get limitlessApiKey {
    // In Flutter, use const String.fromEnvironment() for build-time values
    const apiKey = String.fromEnvironment(
      'LIMITLESS_API_KEY',
      defaultValue: '',
    );
    return apiKey.isEmpty ? null : apiKey;
  }

  /// Get recording device ID from environment
  /// 
  /// This is related to future schema changes where recordings will have
  /// a recording_device_id field to support multiple devices (Limitless pendant,
  /// Plaud, Bee, etc.) and multi-user scenarios.
  /// 
  /// Returns null if not set
  static String? get recordingDeviceId {
    const deviceId = String.fromEnvironment(
      'RECORDING_DEVICE_ID',
      defaultValue: '',
    );
    return deviceId.isEmpty ? null : deviceId;
  }

  /// Check if API key is configured
  static bool get hasApiKey => limitlessApiKey != null;

  /// Check if device ID is configured
  static bool get hasDeviceId => recordingDeviceId != null;

  /// Get all configuration values
  static Map<String, String?> get all => {
        'apiBaseUrl': apiBaseUrl,
        'limitlessApiKey': limitlessApiKey,
        'recordingDeviceId': recordingDeviceId,
      };
}

