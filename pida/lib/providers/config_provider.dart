import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/services/env_config.dart';

/// User name provider
/// Gets user name from config for identifying "You" in conversations
/// Defaults to null (will display as "You" in UI)
final userNameProvider = Provider<String?>((ref) {
  return ref.watch(configProvider).userName;
});

/// App configuration provider
/// Manages API base URL and environment settings
class AppConfig {
  final String apiBaseUrl;
  final bool isDevelopment;
  final String? userName; // User's name (for displaying "You" in participant lists)

  const AppConfig({
    required this.apiBaseUrl,
    this.isDevelopment = true,
    this.userName,
  });

  AppConfig copyWith({
    String? apiBaseUrl,
    bool? isDevelopment,
    String? userName,
  }) {
    return AppConfig(
      apiBaseUrl: apiBaseUrl ?? this.apiBaseUrl,
      isDevelopment: isDevelopment ?? this.isDevelopment,
      userName: userName ?? this.userName,
    );
  }
}

/// Configuration provider
/// Reads API base URL from environment variable (PIDA_API_URL) or defaults to http://localhost:8080
final configProvider = StateProvider<AppConfig>((ref) {
  return AppConfig(
    apiBaseUrl: EnvConfig.apiBaseUrl,
    isDevelopment: true,
    userName: EnvConfig.userName,
  );
});

/// API base URL provider (derived from config)
final apiBaseUrlProvider = Provider<String>((ref) {
  return ref.watch(configProvider).apiBaseUrl;
});

