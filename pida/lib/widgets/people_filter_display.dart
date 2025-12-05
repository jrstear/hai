import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/providers/contacts_provider.dart';
import 'package:pida/providers/filter_provider.dart';
import 'package:pida/widgets/contact_avatar.dart';

/// People filter display widget
/// 
/// Displays the selected people (contacts) as avatars in alphabetical order.
/// - Contact avatars (using ContactAvatar widget)
/// - Alphabetical sorting by name (first,family name)
/// - Starts at right edge, expands leftward
/// - Always shows + button at far right
/// - Handles wrapping for different screen sizes
class PeopleFilterDisplay extends ConsumerWidget {
  /// Optional callback when + button is tapped
  final VoidCallback? onAddTap;

  const PeopleFilterDisplay({
    super.key,
    this.onAddTap,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectedPeopleIds = ref.watch(calendarPeopleFilterProvider);
    final contactsAsync = ref.watch(contactsProvider);

    return contactsAsync.when(
      data: (contactListResponse) {
        // Get contact details for selected IDs
        final selectedContacts = selectedPeopleIds
            .map((id) => contactListResponse.contacts.firstWhere(
                  (contact) => contact.id == id,
                  orElse: () => Contact(
                    id: id,
                    name: 'Unknown',
                  ),
                ))
            .toList();

        // Sort alphabetically by name (first,family name)
        selectedContacts.sort((a, b) {
          // Compare by full name
          return a.name.toLowerCase().compareTo(b.name.toLowerCase());
        });

        // Build avatar row (right-aligned, expanding leftward)
        return Align(
          alignment: Alignment.centerRight,
          child: Row(
            mainAxisSize: MainAxisSize.min,
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              // Clear filter button (X) - only shown when there are selected people
              if (selectedContacts.isNotEmpty)
                IconButton(
                  icon: const Icon(Icons.close),
                  onPressed: () {
                    ref.read(calendarPeopleFilterProvider.notifier).state = [];
                  },
                  tooltip: 'Clear filter',
                  padding: EdgeInsets.zero,
                  constraints: const BoxConstraints(),
                ),
              
              // Avatars (right to left, so reverse the list)
              ...selectedContacts.reversed.map((contact) {
                return Container(
                  margin: const EdgeInsets.only(right: 4),
                  child: ContactAvatar(
                    name: contact.name,
                    pictureUrl: contact.pictureUrl,
                    favoriteColor: contact.favoriteColor,
                    size: 32,
                  ),
                );
              }),
              
              // Add button (always at far right)
              IconButton(
                icon: const Icon(Icons.add_circle_outline),
                onPressed: onAddTap,
                tooltip: 'Add people to filter',
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(),
              ),
            ],
          ),
        );
      },
      loading: () => Align(
        alignment: Alignment.centerRight,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          mainAxisAlignment: MainAxisAlignment.end,
          children: [
            if (selectedPeopleIds.isNotEmpty) ...[
              IconButton(
                icon: const Icon(Icons.close),
                onPressed: () {
                  ref.read(calendarPeopleFilterProvider.notifier).state = [];
                },
                tooltip: 'Clear filter',
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.primaryContainer,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  '${selectedPeopleIds.length}',
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: Theme.of(context).colorScheme.onPrimaryContainer,
                        fontWeight: FontWeight.bold,
                      ),
                ),
              ),
            ],
            IconButton(
              icon: const Icon(Icons.add_circle_outline),
              onPressed: onAddTap,
              tooltip: 'Add people to filter',
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(),
            ),
          ],
        ),
      ),
      error: (error, stack) => Align(
        alignment: Alignment.centerRight,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          mainAxisAlignment: MainAxisAlignment.end,
          children: [
            if (selectedPeopleIds.isNotEmpty) ...[
              IconButton(
                icon: const Icon(Icons.close),
                onPressed: () {
                  ref.read(calendarPeopleFilterProvider.notifier).state = [];
                },
                tooltip: 'Clear filter',
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.errorContainer,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  '${selectedPeopleIds.length}',
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: Theme.of(context).colorScheme.onErrorContainer,
                        fontWeight: FontWeight.bold,
                      ),
                ),
              ),
            ],
            IconButton(
              icon: const Icon(Icons.add_circle_outline),
              onPressed: onAddTap,
              tooltip: 'Add people to filter',
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(),
            ),
          ],
        ),
      ),
    );
  }
}

