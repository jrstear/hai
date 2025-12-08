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

/// Normalize a name for matching (lowercase, trim whitespace)
String _normalizeNameForMatching(String name) {
  return name.trim().toLowerCase();
}

/// Check if two names match (exact match or fuzzy match)
bool _namesMatchForHighlighting(String name1, String name2) {
  final normalized1 = _normalizeNameForMatching(name1);
  final normalized2 = _normalizeNameForMatching(name2);
  
  // Exact match
  if (normalized1 == normalized2) {
    return true;
  }

  // Check if one name contains the other (for partial matches)
  if (normalized1.contains(normalized2) || normalized2.contains(normalized1)) {
    final shorter = normalized1.length < normalized2.length ? normalized1 : normalized2;
    // Only allow if the shorter name is at least 3 characters
    // to avoid false matches like "a" matching "alice"
    if (shorter.length >= 3) {
      return true;
    }
  }

  return false;
}

/// Build a widget for a single participant avatar
/// 
/// Rules:
/// - If contact_id exists and contact found → show contact avatar (picture if available, else initials)
/// - If speaker_name is "Unknown" and no contact_id → show "?" icon
/// - Otherwise → show speaker name initials
/// - Purple border if contactId is non-null/non-empty AND speakerName (from blockquote) matches contact.name
///   This indicates auto-matched from Limitless/lifelog onboarding
///   Manual assignments (where speaker_name doesn't match contact.name) are NOT highlighted
Widget buildParticipantAvatar({
  required ({String speakerName, String? contactId}) participant,
  required List<Contact> contacts,
  double size = 32,
}) {
  final speakerName = participant.speakerName;
  final contactId = participant.contactId;
  
  // If contact_id exists, try to find the contact FIRST
  // This ensures that even if speaker_name is "Unknown", we show the contact avatar if contact_id is set
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
    
    // Show purple border ONLY if contactId is non-null/non-empty AND speakerName matches contact.name
    // This indicates auto-matched from Limitless/lifelog onboarding
    // Manual assignments (where speaker_name is "Unknown" or doesn't match) should NOT be highlighted
    // Note: contactId is already checked above (contact != null), so we just need to check name match
    // Ensure both names are non-empty before matching (empty speakerName shouldn't highlight)
    if (contact.name.isNotEmpty && 
        speakerName.isNotEmpty) {
      final namesMatch = _namesMatchForHighlighting(contact.name, speakerName);
      if (namesMatch) {
        showSpecialBorder = true;
      }
    }
  } else {
    // No contact_id or contact not found - check if it's "Unknown" speaker
    final isUnknown = speakerName.trim().toLowerCase() == 'unknown';
    if (isUnknown) {
      // Special case: "Unknown" speakers without contact_id show "?" icon
      avatar = SpeakerAvatar(
        speakerName: 'Unknown',
        size: size,
      );
    } else {
      // Show speaker name initials
      avatar = SpeakerAvatar(
        speakerName: speakerName,
        size: size,
      );
    }
  }
  
  // Wrap with border if needed
  Widget avatarWidget = avatar;
  if (showSpecialBorder) {
    // Use purple color matching Limitless logo (#667eea or similar purple shade)
    avatarWidget = Container(
      padding: const EdgeInsets.all(2), // Border width
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        border: Border.all(
          color: const Color(0xFF667EEA), // Purple matching Limitless logo
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
