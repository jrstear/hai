import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:pida/models/lifelog.dart';
import 'package:pida/providers/lifelog_provider.dart';
import 'package:pida/routes/app_router.dart';
import 'package:pida/widgets/error_widget.dart';
import 'package:pida/widgets/loading_widget.dart';
import 'package:pida/widgets/speaker_avatar.dart';

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
  DateTime _selectedDate = DateTime.now();
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

  void _goToPreviousDay() {
    setState(() {
      _selectedDate = _selectedDate.subtract(const Duration(days: 1));
      _hasScrolledToBottom = false;
    });
  }

  void _goToNextDay() {
    setState(() {
      _selectedDate = _selectedDate.add(const Duration(days: 1));
      _hasScrolledToBottom = false;
    });
  }

  void _onDateChanged(DateTime date) {
    setState(() {
      _selectedDate = date;
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
    final dateStr = _formatDate(_selectedDate);
    final lifelogAsync = ref.watch(lifelogProvider(dateStr));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Calendar'),
      ),
      body: Column(
        children: [
          // Date selector with swipe gestures
          _buildDateSelector(context),

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
    );
  }

  Widget _buildDateSelector(BuildContext context) {
    return GestureDetector(
      onHorizontalDragEnd: (details) {
        if (details.primaryVelocity! > 0) {
          _goToPreviousDay();
        } else if (details.primaryVelocity! < 0) {
          _goToNextDay();
        }
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surface,
          border: Border(
            bottom: BorderSide(
              color: Theme.of(context).dividerColor,
              width: 1,
            ),
          ),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            IconButton(
              icon: const Icon(Icons.arrow_back_ios),
              onPressed: _goToPreviousDay,
            ),
            Expanded(
              child: InkWell(
                onTap: () async {
                  final DateTime? picked = await showDatePicker(
                    context: context,
                    initialDate: _selectedDate,
                    firstDate: DateTime(2000),
                    lastDate: DateTime(2100),
                  );
                  if (picked != null && picked != _selectedDate) {
                    _onDateChanged(picked);
                  }
                },
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Text(
                      DateFormat('EEEE, MMMM d').format(_selectedDate),
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(width: 8),
                    const Icon(Icons.calendar_today, size: 18),
                  ],
                ),
              ),
            ),
            IconButton(
              icon: const Icon(Icons.arrow_forward_ios),
              onPressed: _goToNextDay,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildConversationsList(
      BuildContext context, List<ConversationSummary> summaries) {
    return GestureDetector(
      onHorizontalDragEnd: (details) {
        if (details.primaryVelocity! > 0) {
          _goToPreviousDay();
        } else if (details.primaryVelocity! < 0) {
          _goToNextDay();
        }
      },
      child: ListView.builder(
        controller: _scrollController,
        reverse: false, // Chronological order (earliest first)
        itemCount: summaries.length,
        itemBuilder: (context, index) {
          final summary = summaries[index];
          return _buildConversationRow(context, summary);
        },
      ),
    );
  }

  Widget _buildConversationRow(
      BuildContext context, ConversationSummary summary) {
    return InkWell(
      onTap: () {
        // Navigate to Conversation screen with lifelog_id and date
        context.push(
            '${AppRoutes.conversation}?lifelog_id=${summary.lifelogId}&date=${_formatDate(_selectedDate)}');
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

