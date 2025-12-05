import 'package:flutter/material.dart';

/// Speaker avatar widget
/// 
/// Displays a circular avatar for a speaker based on the display policy:
/// - "Unknown" (case-insensitive) -> circled question mark (?)
/// - Known name without picture -> circled initials (e.g., "FB" for "Foo Bar")
/// - Future: Known name with picture -> actual contact picture
class SpeakerAvatar extends StatelessWidget {
  final String speakerName;
  final double size;
  final Color? backgroundColor;

  const SpeakerAvatar({
    super.key,
    required this.speakerName,
    this.size = 32,
    this.backgroundColor,
  });

  @override
  Widget build(BuildContext context) {
    final isUnknown = speakerName.trim().toLowerCase() == 'unknown';
    final displayText = isUnknown ? '?' : _getInitials(speakerName);
    final bgColor = backgroundColor ?? _getColorForName(speakerName, context);

    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: bgColor,
      ),
      child: Center(
        child: Text(
          displayText,
          style: TextStyle(
            color: Colors.white,
            fontSize: size * 0.4,
            fontWeight: FontWeight.bold,
          ),
        ),
      ),
    );
  }

  /// Get initials from a name
  /// - "Foo Bar" -> "FB"
  /// - "John" -> "JO" (first two letters)
  /// - "A B C" -> "AB" (first and last word)
  /// - Empty/invalid -> "?"
  String _getInitials(String name) {
    final trimmed = name.trim();
    if (trimmed.isEmpty) return '?';

    final words = trimmed.split(RegExp(r'\s+'));
    if (words.length == 1) {
      // Single word: use first two letters
      return words[0].length >= 2
          ? words[0].substring(0, 2).toUpperCase()
          : words[0].toUpperCase();
    } else {
      // Multiple words: use first letter of first and last word
      final first = words[0].isNotEmpty ? words[0][0] : '';
      final last = words[words.length - 1].isNotEmpty
          ? words[words.length - 1][0]
          : '';
      if (first.isEmpty && last.isEmpty) return '?';
      return '${first.toUpperCase()}${last.toUpperCase()}';
    }
  }

  /// Get a consistent color for a name based on hash
  /// This ensures the same name always gets the same color
  Color _getColorForName(String name, BuildContext context) {
    // Generate a hash from the name
    int hash = 0;
    for (int i = 0; i < name.length; i++) {
      hash = name.codeUnitAt(i) + ((hash << 5) - hash);
    }
    
    // Use hash to pick a color from a predefined palette
    final colors = [
      Colors.blue,
      Colors.green,
      Colors.orange,
      Colors.purple,
      Colors.teal,
      Colors.pink,
      Colors.indigo,
      Colors.amber,
      Colors.cyan,
      Colors.deepOrange,
    ];
    
    return colors[hash.abs() % colors.length];
  }
}

