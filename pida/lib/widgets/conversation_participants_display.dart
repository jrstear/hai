import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/widgets/participant_avatar_helper.dart';

/// Conversation participants display widget
/// 
/// Displays conversation participants based on ordered distinct {speaker_name, contact_id} tuples.
/// Shows participants in order of first appearance in conversation.
/// 
/// Features:
/// - Contact avatars (if contact_id exists) with picture or initials
/// - Speaker name initials (if no contact_id)
/// - Special green border for auto-matched contacts (both contact.name and speaker_name non-null)
/// - "?" icon for "Unknown" speakers
/// - Right-justified, participants ordered left-to-right
/// - + button at far right to add people
class ConversationParticipantsDisplay extends ConsumerWidget {
  /// Lifelog ID for this conversation
  final String lifelogId;
  
  /// Ordered list of distinct {speaker_name, contact_id} tuples from blockquotes
  final List<({String speakerName, String? contactId})> orderedParticipants;
  
  /// List of contacts to lookup contact_id values
  final List<Contact> contacts;
  
  /// Optional callback when + button is tapped
  final VoidCallback? onAddTap;

  const ConversationParticipantsDisplay({
    super.key,
    required this.lifelogId,
    required this.orderedParticipants,
    required this.contacts,
    this.onAddTap,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final avatarWidgets = <Widget>[];
    
    // Limit to 3 avatars (current limit)
    final participantsToShow = orderedParticipants.take(3).toList();
    
    // Build avatars using shared helper function
    for (final participant in participantsToShow) {
      avatarWidgets.add(buildParticipantAvatar(
        participant: participant,
        contacts: contacts,
        size: 32,
      ));
    }
    
    // Build avatar row (right-aligned, participants in order left-to-right)
    return Align(
      alignment: Alignment.centerRight,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          // Participant avatars (left to right, in order of appearance)
          ...avatarWidgets,
          
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
  }
}