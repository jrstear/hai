import 'package:audioplayers/audioplayers.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/providers/api_key_provider.dart';
import 'package:pida/providers/config_provider.dart';

/// Audio playback state
enum AudioPlaybackState {
  stopped,
  loading,
  playing,
  paused,
  error,
}

/// Audio playback service for Limitless API
/// 
/// Handles streaming audio from Limitless API using the audioplayers package.
/// Supports time range playback using start_ms and end_ms parameters.
/// 
/// For web platform, uses API proxy endpoint to handle API key headers.
/// For mobile platforms, can call Limitless API directly (if headers are supported).
class AudioService {
  final AudioPlayer _player = AudioPlayer();
  final String? apiKey;
  final String? apiBaseUrl;
  
  String? _currentUrl;
  int? _startMs;
  int? _endMs;
  AudioPlaybackState _state = AudioPlaybackState.stopped;
  String? _errorMessage;
  
  // Callback for state changes (for notifying listeners)
  void Function()? _onStateChanged;

  AudioService({this.apiKey, this.apiBaseUrl}) {
    // Set up player event listeners
    _player.onPlayerStateChanged.listen((PlayerState playerState) {
      if (playerState == PlayerState.playing) {
        _state = AudioPlaybackState.playing;
      } else if (playerState == PlayerState.paused) {
        _state = AudioPlaybackState.paused;
      } else if (playerState == PlayerState.completed) {
        _state = AudioPlaybackState.stopped;
      }
      _onStateChanged?.call();
    });

    _player.onLog.listen((message) {
      // Log audio player messages for debugging
      print('AudioPlayer: $message');
    });
  }

  /// Register a callback to be notified when the audio state changes
  void registerStateChangeListener(void Function() listener) {
    _onStateChanged = listener;
  }

  /// Unregister the state change listener
  void unregisterStateChangeListener() {
    _onStateChanged = null;
  }

  /// Get current playback state
  AudioPlaybackState get state => _state;
  
  /// Get error message if state is error
  String? get errorMessage => _errorMessage;
  
  /// Get current playback position in milliseconds
  Future<int?> get position async {
    final pos = await _player.getCurrentPosition();
    return pos?.inMilliseconds;
  }
  
  /// Get total duration in milliseconds
  Future<Duration?> get duration async {
    return await _player.getDuration();
  }

  /// Play audio from Limitless API
  /// 
  /// [startMs] and [endMs] are Unix milliseconds (absolute timestamps)
  /// These are used to construct the Limitless API URL with query parameters
  Future<void> play({
    required int startMs,
    required int endMs,
  }) async {
    if (apiKey == null || apiKey!.isEmpty) {
      _state = AudioPlaybackState.error;
      _errorMessage = 'API key not configured';
      return;
    }

    try {
      _state = AudioPlaybackState.loading;
      _errorMessage = null;
      _startMs = startMs;
      _endMs = endMs;

      // Construct audio URL
      // For web, use our API proxy endpoint (handles API key server-side)
      // For mobile, we can try direct Limitless API (if headers are supported)
      final url = kIsWeb
          ? _buildProxyUrl(startMs: startMs, endMs: endMs)
          : _buildLimitlessApiUrl(startMs: startMs, endMs: endMs);
      _currentUrl = url;

      // Play the audio
      final source = UrlSource(url);
      await _player.play(source);
      _state = AudioPlaybackState.playing;
      _onStateChanged?.call();
    } catch (e) {
      _state = AudioPlaybackState.error;
      _errorMessage = e.toString();
      print('Audio playback error: $e');
      _onStateChanged?.call();
    }
  }

  /// Pause audio playback
  Future<void> pause() async {
    try {
      await _player.pause();
      _state = AudioPlaybackState.paused;
      _onStateChanged?.call();
    } catch (e) {
      _state = AudioPlaybackState.error;
      _errorMessage = e.toString();
      _onStateChanged?.call();
    }
  }

  /// Resume audio playback
  Future<void> resume() async {
    try {
      await _player.resume();
      _state = AudioPlaybackState.playing;
      _onStateChanged?.call();
    } catch (e) {
      _state = AudioPlaybackState.error;
      _errorMessage = e.toString();
      _onStateChanged?.call();
    }
  }

  /// Stop audio playback
  Future<void> stop() async {
    try {
      await _player.stop();
      _state = AudioPlaybackState.stopped;
      _currentUrl = null;
      _startMs = null;
      _endMs = null;
      _onStateChanged?.call();
    } catch (e) {
      _state = AudioPlaybackState.error;
      _errorMessage = e.toString();
      _onStateChanged?.call();
    }
  }

  /// Seek to a specific position
  /// 
  /// [position] is in milliseconds
  Future<void> seek(Duration position) async {
    try {
      await _player.seek(position);
    } catch (e) {
      _state = AudioPlaybackState.error;
      _errorMessage = e.toString();
    }
  }

  /// Check if currently playing the specified audio segment
  bool isPlayingSegment(int startMs, int endMs) {
    return _state == AudioPlaybackState.playing &&
        _startMs == startMs &&
        _endMs == endMs;
  }

  /// Build Limitless API URL with query parameters (for mobile)
  String _buildLimitlessApiUrl({
    required int startMs,
    required int endMs,
  }) {
    const baseUrl = 'https://api.limitless.ai/v1/download-audio';
    final url = Uri.parse(baseUrl).replace(
      queryParameters: {
        'startMs': startMs.toString(),
        'endMs': endMs.toString(),
      },
    );
    
    return url.toString();
  }

  /// Build proxy URL through our API server (for web)
  String _buildProxyUrl({
    required int startMs,
    required int endMs,
  }) {
    final baseUrl = apiBaseUrl ?? 'http://localhost:8080';
    final url = Uri.parse('$baseUrl/api/limitless/audio').replace(
      queryParameters: {
        'startMs': startMs.toString(),
        'endMs': endMs.toString(),
      },
    );
    
    return url.toString();
  }

  /// Dispose resources
  void dispose() {
    _player.dispose();
  }
}

/// Provider for audio service instance
final audioServiceProvider = Provider<AudioService>((ref) {
  final apiKey = ref.watch(apiKeyProvider);
  final apiBaseUrl = ref.watch(apiBaseUrlProvider);
  final service = AudioService(apiKey: apiKey, apiBaseUrl: apiBaseUrl);
  
  // Dispose when provider is disposed
  ref.onDispose(() {
    service.dispose();
  });
  
  return service;
});

/// Provider for audio playback state (reactive)
/// 
/// This provider watches the audio service and notifies listeners of state changes
final audioPlaybackStateProvider = StateNotifierProvider<AudioPlaybackStateNotifier, AudioPlaybackState>((ref) {
  final notifier = AudioPlaybackStateNotifier(ref);
  
  // Set up state change callback
  final service = ref.read(audioServiceProvider);
  service.registerStateChangeListener(() {
    notifier._updateState();
  });
  
  // Clean up listener when provider is disposed
  ref.onDispose(() {
    service.unregisterStateChangeListener();
  });
  
  return notifier;
});

class AudioPlaybackStateNotifier extends StateNotifier<AudioPlaybackState> {
  final Ref _ref;

  AudioPlaybackStateNotifier(this._ref) : super(AudioPlaybackState.stopped);

  void _updateState() {
    final service = _ref.read(audioServiceProvider);
    state = service.state;
  }

  Future<void> play({required int startMs, required int endMs}) async {
    final service = _ref.read(audioServiceProvider);
    await service.play(startMs: startMs, endMs: endMs);
    _updateState();
  }

  Future<void> pause() async {
    final service = _ref.read(audioServiceProvider);
    await service.pause();
    _updateState();
  }

  Future<void> resume() async {
    final service = _ref.read(audioServiceProvider);
    await service.resume();
    _updateState();
  }

  Future<void> stop() async {
    final service = _ref.read(audioServiceProvider);
    await service.stop();
    _updateState();
  }

  bool isPlayingSegment(int startMs, int endMs) {
    final service = _ref.read(audioServiceProvider);
    return service.isPlayingSegment(startMs, endMs);
  }
}
