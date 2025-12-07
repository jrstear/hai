import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/widgets/contact_avatar.dart';
import 'package:pida/widgets/speaker_avatar.dart';

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
/// - Right-justified, expands leftward
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
    
    for (final participant in participantsToShow) {
      final speakerName = participant.speakerName;
      final contactId = participant.contactId;
      
      // Special case: "Unknown" speakers show "?" icon
      if (speakerName.trim().toLowerCase() == 'unknown') {
        avatarWidgets.add(
          Container(
            margin: const EdgeInsets.only(right: 4),
            child: const SpeakerAvatar(
              speakerName: 'Unknown',
              size: 32,
            ),
          ),
        );
        continue;
      }
      
      // If contact_id exists, try to find the contact
      Contact? contact;
      if (contactId != null && contactId.isNotEmpty) {
        try {
          contact = contacts.firstWhere((c) => c.id == contactId);
        } catch (e) {
          // Contact not found, will fall back to speaker name initials
          contact = null;
        }
      }
      
      Widget avatar;
      bool showSpecialBorder = false;
      
      if (contact != null) {
        // Contact found - show contact avatar (picture if available, else initials)
        avatar = ContactAvatar(
          name: contact.name,
          pictureUrl: contact.pictureUrl,
          favoriteColor: contact.favoriteColor,
          size: 32,
        );
        
        // Show green border if both contact.name and speaker_name are non-null
        // This indicates auto-matched from Limitless/lifelog onboarding
        if (contact.name.isNotEmpty && speakerName.isNotEmpty) {
          showSpecialBorder = true;
        }
      } else {
        // No contact_id or contact not found - show speaker name initials
        avatar = SpeakerAvatar(
          speakerName: speakerName,
          size: 32,
        );
      }
      
      // Wrap with border if needed
      Widget avatarWidget = avatar;
      if (showSpecialBorder) {
        avatarWidget = Container(
          padding: const EdgeInsets.all(2), // Border width
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            border: Border.all(
              color: Colors.green,
              width: 2,
            ),
          ),
          child: avatar,
        );
      } else {
        avatarWidget = Container(
          margin: const EdgeInsets.only(right: 4),
          child: avatar,
        );
      }
      
      // Add margin if we added border (border container doesn't have margin)
      if (showSpecialBorder) {
        avatarWidget = Container(
          margin: const EdgeInsets.only(right: 4),
          child: avatarWidget,
        );
      }
      
      avatarWidgets.add(avatarWidget);
    }
    
    // Build avatar row (right-aligned, expanding leftward)
    return Align(
      alignment: Alignment.centerRight,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          // Participant avatars (right to left)
          ...avatarWidgets.reversed,
          
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