import 'package:flutter/material.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/models/lifelog.dart';
import 'package:pida/widgets/contact_avatar.dart';
import 'package:pida/widgets/speaker_avatar.dart';

/// Helper functions for building participant avatars from blockquote data
/// Shared between calendar and conversation pages

/// Build ordered list of distinct {speaker_name, contact_id} tuples from blockquotes
/// Order by first appearance in conversation (chronological)
List<({String speakerName, String? contactId})> buildOrderedParticipantTuples(List<Blockquote> blockquotes) {
  final participantTuples = <({String speakerName, String? contactId})>{};
  final orderedParticipants = <({String speakerName, String? contactId})>[];
  
  // Sort blockquotes by start time to process in chronological order
  final sortedBlockquotes = List<Blockquote>.from(blockquotes)
    ..sort((a, b) => a.startTime.compareTo(b.startTime));
  
  for (final blockquote in sortedBlockquotes) {
    final tuple = (
      speakerName: blockquote.speakerName,
      contactId: blockquote.contactId,
    );
    if (!participantTuples.contains(tuple)) {
      participantTuples.add(tuple);
      orderedParticipants.add(tuple);
    }
  }
  
  return orderedParticipants;
}

/// Build a widget for a single participant avatar
/// 
/// Rules:
/// - If speaker_name is "Unknown" → show "?" icon
/// - If contact_id exists and contact found → show contact avatar (picture if available, else initials)
/// - Otherwise → show speaker name initials
/// - Green border if both contact.name and speaker_name are non-null (auto-matched from Limitless)
Widget buildParticipantAvatar({
  required ({String speakerName, String? contactId}) participant,
  required List<Contact> contacts,
  double size = 32,
}) {
  final speakerName = participant.speakerName;
  final contactId = participant.contactId;
  
  // Special case: "Unknown" speakers show "?" icon
  if (speakerName.trim().toLowerCase() == 'unknown') {
    return Container(
      margin: const EdgeInsets.only(right: 4),
      child: const SpeakerAvatar(
        speakerName: 'Unknown',
        size: 32,
      ),
    );
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
      size: size,
    );
    
    // Show green border if both contact.name and speaker_name are non-null/non-empty
    // This indicates auto-matched from Limitless/lifelog onboarding
    if (contact.name.isNotEmpty && speakerName.isNotEmpty) {
      showSpecialBorder = true;
    }
  } else {
    // No contact_id or contact not found - show speaker name initials
    avatar = SpeakerAvatar(
      speakerName: speakerName,
      size: size,
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
  }
  
  // Add margin
  return Container(
    margin: const EdgeInsets.only(right: 4),
    child: avatarWidget,
  );
}
