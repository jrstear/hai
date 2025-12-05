import 'package:flutter/material.dart';

/// Todo screen
class TodoScreen extends StatelessWidget {
  const TodoScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Todo'),
      ),
      body: const Center(
        child: Text('Todo screen - Coming soon'),
      ),
    );
  }
}

