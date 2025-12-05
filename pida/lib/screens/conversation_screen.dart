import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:pida/models/lifelog.dart';
import 'package:pida/providers/lifelog_provider.dart';
import 'package:pida/services/audio_service.dart';
import 'package:pida/widgets/error_widget.dart';
import 'package:pida/widgets/filter_bar.dart';
import 'package:pida/widgets/loading_widget.dart';
import 'package:pida/widgets/speaker_avatar.dart';

/// Conversation screen (Single conversation view)
/// 
/// Shows a SINGLE conversation corresponding to one lifelog conversation.
/// 
/// **Navigation Flow:**
/// - User clicks on a conversation in Calendar day view
/// - Conversation screen slides in from right to left
/// - Date/time filter shows the conversation's start date/time
/// - User can swipe left-to-right to return to calendar day view
/// 
/// **Content:**
/// - Conversation title/header
/// - Date and time of conversation start
/// - List of participants (contact pictures)
/// - Blockquotes/transcripts in chronological order
/// - Audio playback controls for each segment
class ConversationScreen extends ConsumerWidget {
  final String? lifelogId;
  final String? date; // Date in YYYY-MM-DD format

  const ConversationScreen({
    super.key,
    this.lifelogId,
    this.date,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (lifelogId == null) {
      return Scaffold(
        appBar: AppBar(
          title: const Text('Conversation'),
        ),
        body: const ErrorDisplayWidget(
          message: 'No conversation ID provided',
        ),
      );
    }

    // If date is provided, use it; otherwise default to today
    final dateStr = date ?? DateFormat('yyyy-MM-dd').format(DateTime.now());
    final lifelogAsync = ref.watch(lifelogProvider(dateStr));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Conversation'),
      ),
      body: lifelogAsync.when(
        data: (response) {
          // Find the conversation with matching lifelogId
          final blockquotes = response.grouped[lifelogId!];

          if (blockquotes == null || blockquotes.isEmpty) {
            return ErrorDisplayWidget(
              message: 'Conversation not found',
              onRetry: () => ref.invalidate(lifelogProvider(dateStr)),
            );
          }

          // Sort blockquotes by start time
          final sortedBlockquotes = List<Blockquote>.from(blockquotes)
            ..sort((a, b) => a.startTime.compareTo(b.startTime));

          final firstBlockquote = sortedBlockquotes.first;
          final timing = response.conversationTimings[lifelogId!];

          // Get unique participant names
          final participantNames = blockquotes
              .map((bq) => bq.speakerName)
              .toSet()
              .toList();

          return _buildConversationContent(
            context,
            ref,
            firstBlockquote,
            sortedBlockquotes,
            participantNames,
            timing,
          );
        },
        loading: () => const LoadingWidget(message: 'Loading conversation...'),
        error: (error, stack) => ErrorDisplayWidget(
          message: 'Failed to load conversation: ${error.toString()}',
          onRetry: () => ref.invalidate(lifelogProvider(dateStr)),
        ),
      ),
    );
  }

  Widget _buildConversationContent(
    BuildContext context,
    WidgetRef ref,
    Blockquote firstBlockquote,
    List<Blockquote> blockquotes,
    List<String> participantNames,
    ConversationTiming? timing,
  ) {
    return Column(
      children: [
        // Header with title and participants
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surface,
            border: Border(
              bottom: BorderSide(
                color: Theme.of(context).dividerColor,
                width: 1,
              ),
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (firstBlockquote.lifelogTitle != null)
                Text(
                  firstBlockquote.lifelogTitle!,
                  style: Theme.of(context).textTheme.titleLarge?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Text(
                    _formatTimeTo12Hour(
                        firstBlockquote.startTime, firstBlockquote.startOffsetMs),
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: Theme.of(context).colorScheme.primary,
                          fontWeight: FontWeight.w500,
                        ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    softWrap: false,
                  ),
                  const SizedBox(width: 16),
                  ...participantNames.map((name) {
                    return Padding(
                      padding: const EdgeInsets.only(right: 8),
                      child: SpeakerAvatar(
                        speakerName: name,
                        size: 32,
                      ),
                    );
                  }),
                ],
              ),
            ],
          ),
        ),

        // Blockquotes/transcripts
        Expanded(
          child: ListView.builder(
            padding: const EdgeInsets.all(16),
            itemCount: blockquotes.length,
            itemBuilder: (context, index) {
              final blockquote = blockquotes[index];
              return _buildBlockquote(context, blockquote, ref);
            },
          ),
        ),
      ],
    );
  }

  Widget _buildBlockquote(
      BuildContext context, Blockquote blockquote, WidgetRef ref) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              SpeakerAvatar(
                speakerName: blockquote.speakerName,
                size: 24,
              ),
              const SizedBox(width: 8),
              Text(
                blockquote.speakerName,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
              ),
              const SizedBox(width: 8),
              Text(
                _formatTimeTo12Hour(
                    blockquote.startTime, blockquote.startOffsetMs),
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color:
                          Theme.of(context).colorScheme.onSurface.withOpacity(0.6),
                    ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                softWrap: false,
              ),
              const SizedBox(width: 8),
              // Play/pause button
              _buildPlayPauseButton(context, blockquote, ref),
            ],
          ),
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(
              blockquote.content,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
        ],
      ),
    );
  }

  /// Format time to 12-hour AM/PM format in local timezone
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

  /// Build play/pause button for a blockquote
  Widget _buildPlayPauseButton(
    BuildContext context,
    Blockquote blockquote,
    WidgetRef ref,
  ) {
    final audioState = ref.watch(audioPlaybackStateProvider);
    final audioNotifier = ref.read(audioPlaybackStateProvider.notifier);
    final isPlaying = audioNotifier.isPlayingSegment(
      blockquote.startOffsetMs,
      blockquote.endOffsetMs,
    );

    IconData icon;
    String tooltip;
    VoidCallback? onPressed;

    if (audioState == AudioPlaybackState.loading &&
        audioNotifier.isPlayingSegment(
          blockquote.startOffsetMs,
          blockquote.endOffsetMs,
        )) {
      icon = Icons.hourglass_empty;
      tooltip = 'Loading...';
      onPressed = null;
    } else if (isPlaying && audioState == AudioPlaybackState.playing) {
      icon = Icons.pause;
      tooltip = 'Pause';
      onPressed = () async {
        await audioNotifier.pause();
      };
    } else if (isPlaying && audioState == AudioPlaybackState.paused) {
      icon = Icons.play_arrow;
      tooltip = 'Resume';
      onPressed = () async {
        await audioNotifier.resume();
      };
    } else {
      icon = Icons.play_arrow;
      tooltip = 'Play audio';
      onPressed = () async {
        // Stop any currently playing audio
        await audioNotifier.stop();
        // Play this blockquote's audio
        await audioNotifier.play(
          startMs: blockquote.startOffsetMs,
          endMs: blockquote.endOffsetMs,
        );
      };
    }

    return IconButton(
      icon: Icon(icon, size: 20),
      onPressed: onPressed,
      padding: EdgeInsets.zero,
      constraints: const BoxConstraints(
        minWidth: 32,
        minHeight: 32,
      ),
      tooltip: tooltip,
    );
  }
}

