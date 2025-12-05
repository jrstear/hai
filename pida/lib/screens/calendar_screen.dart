import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:pida/providers/filter_provider.dart';
import 'package:pida/providers/lifelog_provider.dart';
import 'package:pida/routes/app_router.dart';
import 'package:pida/widgets/error_widget.dart';
import 'package:pida/widgets/filter_bar.dart';
import 'package:pida/widgets/loading_widget.dart';
import 'package:pida/widgets/people_filter_display.dart';
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

  String _formatDate(DateTime date) {
    return DateFormat('yyyy-MM-dd').format(date);
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

    return '${hour12}:${minute.toString().padLeft(2, '0')} $period';
  }

  @override
  Widget build(BuildContext context) {
    final selectedDate = ref.watch(calendarDateFilterProvider) ?? DateTime.now();
    final dateStr = _formatDate(selectedDate);
    final lifelogAsync = ref.watch(lifelogProvider(dateStr));

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
              rightContent: const PeopleFilterDisplay(),
            ),

            // Conversations list
            Expanded(
              child: lifelogAsync.when(
                data: (response) {
                  final summaries = extractConversationSummaries(response);

                  if (summaries.isEmpty) {
                    return const EmptyStateWidget(
                      message: 'No conversations found for this date',
                      icon: Icons.chat_bubble_outline,
                    );
                  }

                  // Scroll to bottom after data loads (snap to latest conversation)
                  WidgetsBinding.instance.addPostFrameCallback((_) {
                    _scrollToBottom();
                  });

                  return _buildConversationsList(context, summaries);
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

