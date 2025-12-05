import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/providers/theme_provider.dart';
import 'package:pida/routes/app_router.dart';

void main() {
  runApp(
    const ProviderScope(
      child: PidaApp(),
    ),
  );
}

class PidaApp extends ConsumerWidget {
  const PidaApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final themeMode = ref.watch(themeModeProvider);
    final lightTheme = ref.watch(lightThemeProvider);
    final darkTheme = ref.watch(darkThemeProvider);

    return MaterialApp.router(
      title: 'Pida',
      theme: lightTheme,
      darkTheme: darkTheme,
      themeMode: themeMode,
      routerConfig: appRouter,
    );
  }
}

