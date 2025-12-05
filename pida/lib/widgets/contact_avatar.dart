import 'package:flutter/material.dart';

/// Contact avatar widget
/// 
/// Displays a circular avatar for a contact:
/// - With picture_url -> displays the contact picture
/// - Without picture -> displays initials (e.g., "FB" for "Foo Bar")
/// - Uses favorite_color if available, otherwise generates color from name
class ContactAvatar extends StatelessWidget {
  final String name;
  final String? pictureUrl;
  final String? favoriteColor;
  final double size;

  const ContactAvatar({
    super.key,
    required this.name,
    this.pictureUrl,
    this.favoriteColor,
    this.size = 40,
  });

  @override
  Widget build(BuildContext context) {
    // Try to use picture if available
    if (pictureUrl != null && pictureUrl!.isNotEmpty) {
      return Container(
        width: size,
        height: size,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: Colors.grey[300],
        ),
        child: ClipOval(
          child: Image.network(
            pictureUrl!,
            width: size,
            height: size,
            fit: BoxFit.cover,
            errorBuilder: (context, error, stackTrace) {
              // Fall back to initials if image fails to load
              return _buildInitialsAvatar(context);
            },
            loadingBuilder: (context, child, loadingProgress) {
              if (loadingProgress == null) return child;
              return _buildInitialsAvatar(context);
            },
          ),
        ),
      );
    }

    // No picture URL, use initials
    return _buildInitialsAvatar(context);
  }

  Widget _buildInitialsAvatar(BuildContext context) {
    final initials = _getInitials(name);
    final bgColor = _getBackgroundColor(context);

    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: bgColor,
      ),
      child: Center(
        child: Text(
          initials,
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

  /// Get background color for initials avatar
  /// Uses favorite_color if available, otherwise generates color from name
  Color _getBackgroundColor(BuildContext context) {
    // Use favorite_color if available
    if (favoriteColor != null && favoriteColor!.isNotEmpty) {
      try {
        // Try to parse hex color (with or without #)
        var hex = favoriteColor!.replaceFirst('#', '');
        if (hex.length == 6) {
          return Color(int.parse('FF$hex', radix: 16));
        }
      } catch (e) {
        // Fall through to generated color
      }
    }

    // Generate a consistent color from the name
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
      Colors.red,
      Colors.brown,
    ];

    return colors[hash.abs() % colors.length];
  }
}

