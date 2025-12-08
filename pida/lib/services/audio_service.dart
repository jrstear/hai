import 'package:audioplayers/audioplayers.dart';
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
/// Uses the local API proxy endpoint (/api/limitless/audio) which handles
/// authentication with the Limitless API (API key in headers).
/// This works consistently across all platforms (web, Android, iOS).
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
    // Configure player for better OGG/streaming support on Android
    // Use MediaPlayer mode (not LowLatency) for better format support
    // This is async but we can't await in constructor, so we'll set it before playing
    _player.setPlayerMode(PlayerMode.mediaPlayer);
    
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

    // Listen for player errors
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

  /// Play audio from Limitless API via local proxy
  ///
  /// [startMs] and [endMs] are Unix milliseconds (absolute timestamps)
  /// These are used to construct the local API proxy URL with query parameters
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

      // Construct audio URL - always use proxy through local API server
      // This ensures consistent behavior across platforms and allows the local API
      // to handle authentication with Limitless API (API key in headers)
      // The local API endpoint is: /api/limitless/audio?startMs=X&endMs=Y
      final url = _buildProxyUrl(startMs: startMs, endMs: endMs);
      _currentUrl = url;

      // Ensure player is in MediaPlayer mode for OGG streaming support
      await _player.setPlayerMode(PlayerMode.mediaPlayer);
      
      // Play the audio
      final source = UrlSource(url);
      print('Playing audio from URL: $url');
      await _player.play(source, volume: 1.0);
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

  /// Build proxy URL through our API server (for all platforms)
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
final audioPlaybackStateProvider =
    StateNotifierProvider<AudioPlaybackStateNotifier, AudioPlaybackState>(
        (ref) {
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
