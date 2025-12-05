import 'package:flutter/material.dart';
import 'package:pida/utils/loading_state.dart';
import 'package:pida/widgets/error_widget.dart';
import 'package:pida/widgets/loading_widget.dart';

/// Widget builder for LoadingState
class LoadingStateBuilder<T> extends StatelessWidget {
  final LoadingState<T> state;
  final Widget Function(T data) builder;
  final Widget? loading;
  final Widget Function(String error)? error;
  final Widget? empty;

  const LoadingStateBuilder({
    super.key,
    required this.state,
    required this.builder,
    this.loading,
    this.error,
    this.empty,
  });

  @override
  Widget build(BuildContext context) {
    return state.when(
      initial: () => empty ?? const SizedBox.shrink(),
      loading: () => loading ?? const LoadingWidget(),
      success: (data) => builder(data),
      error: (message) => error != null
          ? error!(message)
          : ErrorDisplayWidget(message: message),
    );
  }
}

