import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/providers/filter_provider.dart';

/// People filter display widget
/// 
/// Displays the selected people (contacts) as avatars in alphabetical order.
/// Currently a placeholder - will be fully implemented in Phase 4 with:
/// - Contact avatars (using ContactAvatar widget)
/// - Alphabetical sorting
/// - Add button (+)
/// - Smart wrapping logic
/// - Click handlers for adding/removing people
class PeopleFilterDisplay extends ConsumerWidget {
  /// Optional callback when + button is tapped
  final VoidCallback? onAddTap;

  const PeopleFilterDisplay({
    super.key,
    this.onAddTap,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectedPeople = ref.watch(calendarPeopleFilterProvider);

    // Placeholder: Show count and + button
    // Phase 4 will replace this with actual contact avatars
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        // Selected people count (placeholder)
        if (selectedPeople.isNotEmpty)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.primaryContainer,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Text(
              '${selectedPeople.length}',
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Theme.of(context).colorScheme.onPrimaryContainer,
                    fontWeight: FontWeight.bold,
                  ),
            ),
          ),
        
        // Add button
        IconButton(
          icon: const Icon(Icons.add_circle_outline),
          onPressed: onAddTap,
          tooltip: 'Add people to filter',
        ),
      ],
    );
  }
}

