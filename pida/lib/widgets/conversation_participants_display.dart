import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/providers/contacts_provider.dart';
import 'package:pida/providers/config_provider.dart';
import 'package:pida/widgets/contact_avatar.dart';
import 'package:pida/widgets/speaker_avatar.dart';

/// Conversation participants display widget
/// 
/// Displays conversation participants as avatars in alphabetical order.
/// This shows conversation data (participants), not a filter.
/// 
/// Features:
/// - Contact avatars (using ContactAvatar widget)
/// - Alphabetical sorting by name
/// - Right-justified, expands leftward
/// - + button at far right to add people
/// - Supports smart wrapping with title
class ConversationParticipantsDisplay extends ConsumerWidget {
  /// Lifelog ID for this conversation
  final String lifelogId;
  
  /// List of participant contact IDs
  final List<String> participantContactIds;
  
  /// Whether "You" (the app user) is a participant
  final bool hasUser;
  
  /// Whether "Unknown" speakers are present in the conversation
  final bool hasUnknown;
  
  /// Optional callback when + button is tapped
  final VoidCallback? onAddTap;

  const ConversationParticipantsDisplay({
    super.key,
    required this.lifelogId,
    required this.participantContactIds,
    this.hasUser = false,
    this.hasUnknown = false,
    this.onAddTap,
  });

  /// Build "You" avatar for the app user
  Widget _buildYouAvatar(BuildContext context, String? userName) {
    final displayName = userName ?? 'You';
    return Container(
      padding: const EdgeInsets.all(2), // Border width
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        border: Border.all(
          color: Theme.of(context).colorScheme.primary,
          width: 2,
        ),
      ),
      child: ContactAvatar(
        name: displayName,
        size: 32,
      ),
    );
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final contactsAsync = ref.watch(contactsProvider);
    final userName = ref.watch(userNameProvider);

    return contactsAsync.when(
      data: (contactListResponse) {
        // Get contact details for participant IDs
        final participantContacts = participantContactIds
            .map((id) => contactListResponse.contacts.firstWhere(
                  (contact) => contact.id == id,
                  orElse: () => Contact(
                    id: id,
                    name: 'Unknown',
                  ),
                ))
            .toList();

        // Sort alphabetically by name
        participantContacts.sort((a, b) {
          return a.name.toLowerCase().compareTo(b.name.toLowerCase());
        });

        // Build avatar row (right-aligned, expanding leftward)
        return Align(
          alignment: Alignment.centerRight,
          child: Row(
            mainAxisSize: MainAxisSize.min,
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              // Avatars (right to left, so reverse the list)
              ...participantContacts.reversed.map((contact) {
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
              
              // Add "You" avatar if user is a participant
              if (hasUser)
                Container(
                  margin: const EdgeInsets.only(right: 4),
                  child: _buildYouAvatar(context, userName),
                ),
              
              // Add "?" icon for Unknown participants (consistent with calendar page)
              if (hasUnknown)
                Container(
                  margin: const EdgeInsets.only(right: 4),
                  child: const SpeakerAvatar(
                    speakerName: 'Unknown',
                    size: 32,
                  ),
                ),
              
              // Add button (always at far right)
              IconButton(
                icon: const Icon(Icons.add_circle_outline),
                onPressed: onAddTap,
                tooltip: 'Add people to conversation',
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
            if (participantContactIds.isNotEmpty)
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.primaryContainer,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  '${participantContactIds.length}',
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: Theme.of(context).colorScheme.onPrimaryContainer,
                        fontWeight: FontWeight.bold,
                      ),
                ),
              ),
            IconButton(
              icon: const Icon(Icons.add_circle_outline),
              onPressed: onAddTap,
              tooltip: 'Add people to conversation',
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
            if (participantContactIds.isNotEmpty)
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.errorContainer,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  '${participantContactIds.length}',
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: Theme.of(context).colorScheme.onErrorContainer,
                        fontWeight: FontWeight.bold,
                      ),
                ),
              ),
            IconButton(
              icon: const Icon(Icons.add_circle_outline),
              onPressed: onAddTap,
              tooltip: 'Add people to conversation',
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(),
            ),
          ],
        ),
      ),
    );
  }
}

