import 'package:freezed_annotation/freezed_annotation.dart';

part 'loading_state.freezed.dart';

/// Generic loading state for async operations
@freezed
class LoadingState<T> with _$LoadingState<T> {
  const factory LoadingState.initial() = _Initial<T>;
  const factory LoadingState.loading() = _Loading<T>;
  const factory LoadingState.success(T data) = _Success<T>;
  const factory LoadingState.error(String message) = _Error<T>;
}

/// Extension methods for LoadingState
extension LoadingStateExtension<T> on LoadingState<T> {
  /// Check if state is initial
  bool get isInitial => this is _Initial<T>;

  /// Check if state is loading
  bool get isLoading => this is _Loading<T>;

  /// Check if state is success
  bool get isSuccess => this is _Success<T>;

  /// Check if state is error
  bool get isError => this is _Error<T>;

  /// Get data if success, null otherwise
  T? get dataOrNull => maybeWhen(
        success: (data) => data,
        orElse: () => null,
      );

  /// Get error message if error, null otherwise
  String? get errorOrNull => maybeWhen(
        error: (message) => message,
        orElse: () => null,
      );
}

