import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Calendar page date filter provider
/// 
/// Stores the selected date for the calendar day view.
/// Defaults to today's date.
/// State persists across navigation (Riverpod StateProvider survives navigation).
final calendarDateFilterProvider = StateProvider<DateTime?>((ref) {
  // Default to today's date
  final now = DateTime.now();
  return DateTime(now.year, now.month, now.day);
});

/// Calendar page people filter provider
/// 
/// Stores a list of contact IDs for filtering conversations.
/// The list should be sorted alphabetically by name when displayed.
/// State persists across navigation (Riverpod StateProvider survives navigation).
final calendarPeopleFilterProvider = StateProvider<List<String>>((ref) => []);

/// Helper function to add a person to the calendar people filter
/// 
/// If the contact ID is already in the filter, this is a no-op.
/// The filter should be sorted alphabetically when displayed (handled in UI).
void addPersonToCalendarFilter(WidgetRef ref, String contactId) {
  final currentFilter = ref.read(calendarPeopleFilterProvider);
  if (!currentFilter.contains(contactId)) {
    ref.read(calendarPeopleFilterProvider.notifier).state = [
      ...currentFilter,
      contactId,
    ];
  }
}

/// Helper function to remove a person from the calendar people filter
/// 
/// If the contact ID is not in the filter, this is a no-op.
void removePersonFromCalendarFilter(WidgetRef ref, String contactId) {
  final currentFilter = ref.read(calendarPeopleFilterProvider);
  if (currentFilter.contains(contactId)) {
    ref.read(calendarPeopleFilterProvider.notifier).state = 
        currentFilter.where((id) => id != contactId).toList();
  }
}

/// Helper function to set the calendar date filter
void setCalendarDate(WidgetRef ref, DateTime date) {
  // Normalize to just the date part (no time)
  final normalizedDate = DateTime(date.year, date.month, date.day);
  ref.read(calendarDateFilterProvider.notifier).state = normalizedDate;
}

/// Helper function to clear all calendar filters
void clearCalendarFilters(WidgetRef ref) {
  ref.read(calendarDateFilterProvider.notifier).state = null;
  ref.read(calendarPeopleFilterProvider.notifier).state = [];
}

/// Helper function to get previous day from current calendar date filter
DateTime? getPreviousCalendarDate(WidgetRef ref) {
  final currentDate = ref.read(calendarDateFilterProvider);
  if (currentDate == null) return null;
  return currentDate.subtract(const Duration(days: 1));
}

/// Helper function to get next day from current calendar date filter
DateTime? getNextCalendarDate(WidgetRef ref) {
  final currentDate = ref.read(calendarDateFilterProvider);
  if (currentDate == null) return null;
  return currentDate.add(const Duration(days: 1));
}

/// Helper function to go to previous day in calendar
void goToPreviousCalendarDay(WidgetRef ref) {
  final previousDate = getPreviousCalendarDate(ref);
  if (previousDate != null) {
    setCalendarDate(ref, previousDate);
  }
}

/// Helper function to go to next day in calendar
void goToNextCalendarDay(WidgetRef ref) {
  final nextDate = getNextCalendarDate(ref);
  if (nextDate != null) {
    setCalendarDate(ref, nextDate);
  }
}

// TODO: Future providers for Todo page filters
// final todoDateFilterProvider = StateProvider<DateTime?>((ref) => null);
// final todoPeopleFilterProvider = StateProvider<List<String>>((ref) => []);

/// Conversation participants provider (keyed by lifelogId)
/// 
/// Stores a list of contact IDs for each conversation.
/// This is conversation-specific data (not a filter).
/// Used to track which contacts are associated with a conversation.
/// State persists across navigation (Riverpod StateProvider survives navigation).
final conversationParticipantsProvider = StateProvider.family<List<String>, String>(
  (ref, lifelogId) => [],
);

/// Helper function to add a person to conversation participants
void addPersonToConversationParticipants(WidgetRef ref, String lifelogId, String contactId) {
  final currentParticipants = ref.read(conversationParticipantsProvider(lifelogId));
  if (!currentParticipants.contains(contactId)) {
    ref.read(conversationParticipantsProvider(lifelogId).notifier).state = [
      ...currentParticipants,
      contactId,
    ];
  }
}

/// Helper function to remove a person from conversation participants
void removePersonFromConversationParticipants(WidgetRef ref, String lifelogId, String contactId) {
  final currentParticipants = ref.read(conversationParticipantsProvider(lifelogId));
  if (currentParticipants.contains(contactId)) {
    ref.read(conversationParticipantsProvider(lifelogId).notifier).state =
        currentParticipants.where((id) => id != contactId).toList();
  }
}

/// Blockquote-to-contact association provider
/// 
/// Tracks which blockquotes are associated with which contacts.
/// Key: blockquote_id, Value: contact_id
/// Used to display contact avatars for blockquotes that have been associated.
final blockquoteContactAssociationProvider = StateProvider<Map<String, String>>((ref) => {});

/// Helper function to associate a blockquote with a contact
void associateBlockquoteWithContact(WidgetRef ref, String blockquoteId, String contactId) {
  final currentAssociations = ref.read(blockquoteContactAssociationProvider);
  ref.read(blockquoteContactAssociationProvider.notifier).state = {
    ...currentAssociations,
    blockquoteId: contactId,
  };
}

/// Helper function to get the contact ID associated with a blockquote
String? getBlockquoteContactId(WidgetRef ref, String blockquoteId) {
  final associations = ref.read(blockquoteContactAssociationProvider);
  return associations[blockquoteId];
}

