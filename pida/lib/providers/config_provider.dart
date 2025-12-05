import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/services/env_config.dart';

/// App configuration provider
/// Manages API base URL and environment settings
class AppConfig {
  final String apiBaseUrl;
  final bool isDevelopment;

  const AppConfig({
    required this.apiBaseUrl,
    this.isDevelopment = true,
  });

  AppConfig copyWith({
    String? apiBaseUrl,
    bool? isDevelopment,
  }) {
    return AppConfig(
      apiBaseUrl: apiBaseUrl ?? this.apiBaseUrl,
      isDevelopment: isDevelopment ?? this.isDevelopment,
    );
  }
}

/// Configuration provider
/// Reads API base URL from environment variable (PIDA_API_URL) or defaults to http://localhost:8080
final configProvider = StateProvider<AppConfig>((ref) {
  return AppConfig(
    apiBaseUrl: EnvConfig.apiBaseUrl,
    isDevelopment: true,
  );
});

/// API base URL provider (derived from config)
final apiBaseUrlProvider = Provider<String>((ref) {
  return ref.watch(configProvider).apiBaseUrl;
});

