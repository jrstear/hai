import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/providers/contacts_provider.dart';
import 'package:pida/providers/filter_provider.dart';
import 'package:pida/providers/lifelog_provider.dart';
import 'package:pida/routes/app_router.dart';
import 'package:pida/widgets/error_widget.dart';
import 'package:pida/widgets/filter_bar.dart';
import 'package:pida/widgets/loading_widget.dart';
import 'package:pida/widgets/people_filter_display.dart';
import 'package:pida/widgets/people_selector.dart';
import 'package:pida/widgets/speaker_avatar.dart';
import 'package:pida/widgets/time_filter.dart';

/// Calendar screen - Day View
/// 
/// Shows conversations for a selected date with:
/// - Date selector (previous/next day, date picker)
/// - Swipe gestures for date navigation
/// - List of conversations in chronological order
/// - Each conversation shows time, participant pictures, and a summary
/// - Scrolls to the latest conversation on load
class CalendarScreen extends ConsumerStatefulWidget {
  const CalendarScreen({super.key});

  @override
  ConsumerState<CalendarScreen> createState() => _CalendarScreenState();
}

class _CalendarScreenState extends ConsumerState<CalendarScreen> {
  final ScrollController _scrollController = ScrollController();
  bool _hasScrolledToBottom = false;

  @override
  void initState() {
    super.initState();
    // Listen for scroll events to detect if user has scrolled up
    _scrollController.addListener(() {
      if (_scrollController.position.pixels <
              _scrollController.position.maxScrollExtent &&
          _hasScrolledToBottom) {
        setState(() {
          _hasScrolledToBottom = false;
        });
      }
    });
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _scrollToBottom() {
    if (!_hasScrolledToBottom && _scrollController.hasClients) {
      _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
      _hasScrolledToBottom = true;
    }
  }

  void _onDateChanged(DateTime date) {
    setState(() {
      _hasScrolledToBottom = false;
    });
  }

  /// Open people selector drawer
  void _openPeopleSelector(BuildContext context, WidgetRef ref) {
    final selectedPeopleIds = ref.read(calendarPeopleFilterProvider);

    showPeopleSelector(
      context: context,
      selectedContactIds: selectedPeopleIds,
      contextType: 'filter',
      onContactSelected: (contactId, isSelected) {
        if (isSelected) {
          addPersonToCalendarFilter(ref, contactId);
        } else {
          removePersonFromCalendarFilter(ref, contactId);
        }
      },
    );
  }

  String _formatDate(DateTime date) {
    return DateFormat('yyyy-MM-dd').format(date);
  }

  /// Filter conversations by selected people (local filtering)
  /// 
  /// Matches speaker names to contact IDs and filters conversations
  /// to only include those where at least one participant matches
  /// a selected contact (OR logic).
  List<ConversationSummary> _filterConversationsByPeople(
    List<ConversationSummary> summaries,
    List<String> selectedPeopleIds,
    List<Contact> contacts,
  ) {
    // If no people selected, return all conversations
    if (selectedPeopleIds.isEmpty) {
      return summaries;
    }
    
    // Build mapping: contact ID -> contact name (normalized for matching)
    final contactIdToName = <String, String>{};
    for (final contact in contacts) {
      if (selectedPeopleIds.contains(contact.id)) {
        contactIdToName[contact.id] = _normalizeName(contact.name);
      }
    }

    // Filter conversations: include if any participant matches a selected contact
    return summaries.where((summary) {
      // Check if any participant name matches any selected contact name
      for (final participantName in summary.participantNames) {
        final normalizedParticipant = _normalizeName(participantName);
        
        // Skip "You" and "Unknown" - they don't match contacts
        if (normalizedParticipant == 'you' || normalizedParticipant == 'unknown') {
          continue;
        }

        // Check if this participant name matches any selected contact
        for (final contactName in contactIdToName.values) {
          if (_namesMatch(normalizedParticipant, contactName)) {
            return true; // Match found, include this conversation
          }
        }
      }
      return false; // No match found, exclude this conversation
    }).toList();
  }

  /// Normalize a name for matching (lowercase, trim whitespace)
  String _normalizeName(String name) {
    return name.trim().toLowerCase();
  }

  /// Check if two names match (exact match or fuzzy match)
  /// 
  /// Currently does exact normalized match. Can be enhanced later
  /// for fuzzy matching (e.g., "Jon" vs "Jonathan").
  bool _namesMatch(String name1, String name2) {
    // Exact match
    if (name1 == name2) return true;

    // Check if one name contains the other (for partial matches)
    // e.g., "jon stearley" contains "jon"
    if (name1.contains(name2) || name2.contains(name1)) {
      // Only allow if the shorter name is at least 3 characters
      // to avoid false matches like "a" matching "alice"
      final shorter = name1.length < name2.length ? name1 : name2;
      if (shorter.length >= 3) {
        return true;
      }
    }

    return false;
  }

  // Format time to 12-hour AM/PM format in local timezone
  String _formatTimeTo12Hour(String timeStr, int timestampMs) {
    // Use the timestamp to get the actual date/time in local timezone
    final dateTime =
        DateTime.fromMillisecondsSinceEpoch(timestampMs, isUtc: true).toLocal();

    // Format as 12-hour AM/PM
    final hour = dateTime.hour;
    final minute = dateTime.minute;

    final period = hour >= 12 ? 'PM' : 'AM';
    final hour12 = hour == 0 ? 12 : (hour > 12 ? hour - 12 : hour);

    return '$hour12:${minute.toString().padLeft(2, '0')} $period';
  }

  @override
  Widget build(BuildContext context) {
    final selectedDate = ref.watch(calendarDateFilterProvider) ?? DateTime.now();
    final dateStr = _formatDate(selectedDate);
    final lifelogAsync = ref.watch(lifelogProvider(dateStr));
    final contactsAsync = ref.watch(contactsProvider);
    final selectedPeopleIds = ref.watch(calendarPeopleFilterProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Calendar'),
      ),
      body: GestureDetector(
        // Per-page swipe handling: Capture horizontal swipes to change date
        // This prevents browser back/forward navigation from intercepting swipes
        behavior: HitTestBehavior.opaque,
        onHorizontalDragStart: (_) {
          // Start tracking to participate in gesture arena
        },
        onHorizontalDragUpdate: (details) {
          // Track horizontal drag to ensure we win the gesture arena
          // This prevents browser navigation from handling the swipe
        },
        onHorizontalDragEnd: (details) {
          // Only handle if there's significant velocity (actual swipe, not just a tap)
          if (details.primaryVelocity != null && details.primaryVelocity!.abs() > 200) {
            // User expectation:
            // Swipe left to right -> previous date (same as < button)
            // Swipe right to left -> next date (same as > button)
            if (details.primaryVelocity! > 0) {
              // Left to right swipe -> previous date
              goToPreviousCalendarDay(ref);
              setState(() {
                _hasScrolledToBottom = false;
              });
            } else if (details.primaryVelocity! < 0) {
              // Right to left swipe -> next date
              goToNextCalendarDay(ref);
              setState(() {
                _hasScrolledToBottom = false;
              });
            }
          }
        },
        child: Column(
          children: [
            // Filter bar with time and people filters
            FilterBar(
              leftContent: TimeFilter(
                onDateChanged: _onDateChanged,
              ),
              rightContent: PeopleFilterDisplay(
                onAddTap: () => _openPeopleSelector(context, ref),
              ),
            ),

            // Conversations list
            Expanded(
              child: lifelogAsync.when(
                data: (response) {
                  final summaries = extractConversationSummaries(response);
                  
                  // Apply people filter (local filtering)
                  // Filter conversations based on selected people
                  final filteredSummaries = contactsAsync.when(
                    data: (contactListResponse) {
                      return _filterConversationsByPeople(
                        summaries,
                        selectedPeopleIds,
                        contactListResponse.contacts,
                      );
                    },
                    loading: () => summaries, // Show all while contacts load
                    error: (error, stack) => summaries, // Show all if contacts fail to load
                  );

                  if (filteredSummaries.isEmpty) {
                    if (selectedPeopleIds.isNotEmpty) {
                      return EmptyStateWidget(
                        message: 'No conversations found for this date with selected people',
                        icon: Icons.chat_bubble_outline,
                      );
                    } else {
                      return const EmptyStateWidget(
                        message: 'No conversations found for this date',
                        icon: Icons.chat_bubble_outline,
                      );
                    }
                  }

                  // Scroll to bottom after data loads (snap to latest conversation)
                  WidgetsBinding.instance.addPostFrameCallback((_) {
                    _scrollToBottom();
                  });

                  return _buildConversationsList(context, filteredSummaries);
                },
                loading: () =>
                    const LoadingWidget(message: 'Loading conversations...'),
                error: (error, stack) => ErrorDisplayWidget(
                  message: 'Failed to load conversations: ${error.toString()}',
                  onRetry: () => ref.invalidate(lifelogProvider(dateStr)),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }


  Widget _buildConversationsList(
      BuildContext context, List<ConversationSummary> summaries) {
    return ListView.builder(
      controller: _scrollController,
      reverse: false, // Chronological order (earliest first)
      itemCount: summaries.length,
      itemBuilder: (context, index) {
        final summary = summaries[index];
        return _buildConversationRow(context, summary);
      },
    );
  }

  Widget _buildConversationRow(
      BuildContext context, ConversationSummary summary) {
    return InkWell(
      onTap: () {
        // Navigate to Conversation screen with lifelog_id and date
        final selectedDate = ref.read(calendarDateFilterProvider) ?? DateTime.now();
        context.push(
            '${AppRoutes.conversation}?lifelog_id=${summary.lifelogId}&date=${_formatDate(selectedDate)}');
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          border: Border(
            bottom: BorderSide(
              color: Theme.of(context).dividerColor,
              width: 0.5,
            ),
          ),
        ),
        child: Row(
          children: [
            // Time
            SizedBox(
              width: 70, // Increased width to prevent AM/PM wrapping
              child: Text(
                _formatTimeTo12Hour(summary.startTime,
                    summary.timing?.startMs ?? 0),
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      fontWeight: FontWeight.w500,
                      color: Theme.of(context).colorScheme.primary,
                    ),
                maxLines: 1, // Prevent wrapping
                overflow: TextOverflow.ellipsis, // Handle overflow
                softWrap: false, // Prevent soft wrapping
              ),
            ),

            const SizedBox(width: 12),

            // Participant pictures (circular avatars)
            Expanded(
              child: Row(
                children: [
                  // Placeholder for participant pictures
                  // TODO: Load actual contact pictures when contacts are associated
                  ...summary.participantNames.take(3).map((name) {
                    return Container(
                      margin: const EdgeInsets.only(right: 4),
                      child: SpeakerAvatar(
                        speakerName: name,
                        size: 32,
                      ),
                    );
                  }),
                  if (summary.participantNames.length > 3)
                    Container(
                      margin: const EdgeInsets.only(right: 4),
                      width: 32,
                      height: 32,
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: Theme.of(context).colorScheme.surfaceContainerHighest,
                      ),
                      child: Center(
                        child: Text(
                          '+${summary.participantNames.length - 3}',
                          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                                color: Theme.of(context).colorScheme.onSurface,
                              ),
                        ),
                      ),
                    ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Text(
                      summary.summary,
                      style: Theme.of(context).textTheme.bodyMedium,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

