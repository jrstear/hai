import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/models/lifelog.dart';
import 'package:pida/providers/contacts_provider.dart';
import 'package:pida/providers/filter_provider.dart';
import 'package:pida/providers/lifelog_provider.dart';
import 'package:pida/services/audio_service.dart';
import 'package:pida/services/api_client.dart';
import 'package:pida/widgets/contact_avatar.dart';
import 'package:pida/widgets/conversation_participants_display.dart';
import 'package:pida/widgets/participant_avatar_helper.dart';
import 'package:pida/widgets/error_widget.dart';
import 'package:pida/widgets/filter_bar.dart';
import 'package:pida/widgets/loading_widget.dart';
import 'package:pida/widgets/people_selector.dart';
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
class ConversationScreen extends ConsumerStatefulWidget {
  final String? lifelogId;
  final String? date; // Date in YYYY-MM-DD format

  const ConversationScreen({
    super.key,
    this.lifelogId,
    this.date,
  });

  @override
  ConsumerState<ConversationScreen> createState() => _ConversationScreenState();
}

class _ConversationScreenState extends ConsumerState<ConversationScreen> {
  String? _initializedLifelogId;
  
  @override
  void didUpdateWidget(ConversationScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    // Reset initialization flag if lifelogId changed
    if (oldWidget.lifelogId != widget.lifelogId) {
      _initializedLifelogId = null;
    }
  }


  /// Normalize a name for matching (lowercase, trim whitespace)
  String _normalizeName(String name) {
    return name.trim().toLowerCase();
  }

  /// Check if two names match (exact match or fuzzy match)
  bool _namesMatch(String name1, String name2) {
    // Exact match
    if (name1 == name2) return true;

    // Check if one name contains the other (for partial matches)
    if (name1.contains(name2) || name2.contains(name1)) {
      final shorter = name1.length < name2.length ? name1 : name2;
      if (shorter.length >= 3) {
        return true;
      }
    }

    return false;
  }

  /// Open people selector for conversation
  void _openPeopleSelector(
    BuildContext context,
    WidgetRef ref,
    String lifelogId,
    List<({String speakerName, String? contactId})> orderedParticipants,
  ) {
    // Extract contact IDs from ordered participants (exclude nulls)
    final participantContactIds = orderedParticipants
        .where((p) => p.contactId != null && p.contactId!.isNotEmpty)
        .map((p) => p.contactId!)
        .toSet()
        .toList();

    showPeopleSelector(
      context: context,
      participantContactIds: participantContactIds,
      contextType: 'conversation',
      onContactSelected: (contactId, isSelected) {
        if (isSelected) {
          addPersonToConversationParticipants(ref, lifelogId, contactId);
        } else {
          removePersonFromConversationParticipants(ref, lifelogId, contactId);
        }
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    if (widget.lifelogId == null) {
      return Scaffold(
        appBar: AppBar(
          title: const Text('Conversation'),
        ),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(16.0),
            child: Text(
              'Use the Calendar to select a conversation.',
              style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                    color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.7),
                  ),
              textAlign: TextAlign.center,
            ),
          ),
        ),
      );
    }

    // If date is provided, use it; otherwise default to today
    final dateStr = widget.date ?? DateFormat('yyyy-MM-dd').format(DateTime.now());
    final lifelogAsync = ref.watch(lifelogProvider(dateStr));
    final contactsAsync = ref.watch(contactsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Conversation'),
      ),
      body: lifelogAsync.when(
        data: (response) {
          // Find the conversation with matching lifelogId
          final blockquotes = response.grouped[widget.lifelogId!];

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
          final timing = response.conversationTimings[widget.lifelogId!];

          // Build ordered list of distinct {speaker_name, contact_id} pairs from blockquotes
          // Order by first appearance in conversation (chronological)
          final participantTuples = <({String speakerName, String? contactId})>{};
          final orderedParticipants = <({String speakerName, String? contactId})>[];
          
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

          return contactsAsync.when(
            data: (contactListResponse) {
              return _buildConversationContent(
                context,
                ref,
                widget.lifelogId!,
                firstBlockquote,
                sortedBlockquotes,
                orderedParticipants,
                contactListResponse.contacts,
                timing,
              );
            },
            loading: () => const LoadingWidget(message: 'Loading contacts...'),
            error: (error, stack) => _buildConversationContent(
              context,
              ref,
              widget.lifelogId!,
              firstBlockquote,
              sortedBlockquotes,
              orderedParticipants,
              <Contact>[],
              timing,
            ),
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
    String lifelogId,
    Blockquote firstBlockquote,
    List<Blockquote> blockquotes,
    List<({String speakerName, String? contactId})> orderedParticipants,
    List<Contact> contacts,
    ConversationTiming? timing,
  ) {
    final conversationTitle = firstBlockquote.lifelogTitle ?? 'Conversation';

    return Column(
      children: [
        // Filter bar with title and participants
        FilterBar(
          leftContent: _buildConversationTitle(context, conversationTitle),
          rightContent: ConversationParticipantsDisplay(
            lifelogId: lifelogId,
            orderedParticipants: orderedParticipants,
            contacts: contacts,
            onAddTap: () => _openPeopleSelector(context, ref, lifelogId, orderedParticipants),
          ),
          showBorder: true,
        ),

        // Blockquotes/transcripts
        Expanded(
          child: ListView.builder(
            padding: const EdgeInsets.all(16),
            itemCount: blockquotes.length,
            itemBuilder: (context, index) {
              final blockquote = blockquotes[index];
              return _buildBlockquote(context, blockquote, ref, lifelogId, orderedParticipants);
            },
          ),
        ),
      ],
    );
  }

  /// Check if a speaker name is unknown
  bool _isUnknownSpeaker(String speakerName) {
    return speakerName.trim().toLowerCase() == 'unknown';
  }

  /// Handle unknown speaker click - open people selector to associate
  void _handleUnknownSpeakerClick(
    BuildContext context,
    WidgetRef ref,
    Blockquote blockquote,
    String lifelogId,
    List<({String speakerName, String? contactId})> orderedParticipants,
  ) {
    // Extract contact IDs from ordered participants (exclude nulls)
    final participantContactIds = orderedParticipants
        .where((p) => p.contactId != null && p.contactId!.isNotEmpty)
        .map((p) => p.contactId!)
        .toSet()
        .toList();

    showPeopleSelector(
      context: context,
      participantContactIds: participantContactIds,
      contextType: 'conversation',
      onContactSelected: (contactId, isSelected) async {
        if (isSelected) {
          // Add to conversation participants if not already there
          addPersonToConversationParticipants(ref, lifelogId, contactId);
          
          // Track association locally immediately (for UI update)
          associateBlockquoteWithContact(ref, blockquote.id, contactId);
          
          // Store association in Elasticsearch via API
          await _associateBlockquoteWithContact(ref, context, blockquote, contactId, lifelogId);
        }
      },
    );
  }

  /// Associate blockquote with a contact via API (stores contact_id in Elasticsearch)
  Future<void> _associateBlockquoteWithContact(
    WidgetRef ref,
    BuildContext context,
    Blockquote blockquote,
    String contactId,
    String lifelogId,
  ) async {
    try {
      final apiClient = ref.read(apiClientProvider);
      await apiClient.updateBlockquoteContact(blockquote.id, contactId);
      
      // Refresh lifelog data to get updated blockquotes with contact_id
      final dateStr = widget.date ?? DateFormat('yyyy-MM-dd').format(DateTime.now());
      ref.invalidate(lifelogProvider(dateStr));
      
      // No success message needed - UI already shows success (icon updated, person added)
    } catch (error) {
      // Show error message
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Failed to associate blockquote: ${error.toString()}'),
            duration: const Duration(seconds: 4),
          ),
        );
      }
    }
  }

  Widget _buildBlockquote(
      BuildContext context, 
      Blockquote blockquote, 
      WidgetRef ref, 
      String lifelogId,
      List<({String speakerName, String? contactId})> orderedParticipants) {
    final isUnknown = _isUnknownSpeaker(blockquote.speakerName);
    // Use contact_id from API first, fall back to local state if not yet persisted
    final associatedContactId = blockquote.contactId ?? getBlockquoteContactId(ref, blockquote.id);
    
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              // Show contact avatar if associated, otherwise show speaker avatar
              _buildBlockquoteAvatar(
                context,
                ref,
                blockquote,
                lifelogId,
                isUnknown,
                associatedContactId,
                orderedParticipants,
              ),
              const SizedBox(width: 8),
              // Show contact name if associated, otherwise show speaker name
              Builder(
                builder: (context) {
                  if (associatedContactId != null) {
                    final contactsAsync = ref.watch(contactsProvider);
                    return contactsAsync.when(
                      data: (contactListResponse) {
                        final contact = contactListResponse.contacts.firstWhere(
                          (c) => c.id == associatedContactId,
                          orElse: () => Contact(id: associatedContactId, name: 'Unknown'),
                        );
                        
                        // Name is not highlighted - only the icon is highlighted when names match
                        return Text(
                          contact.name,
                          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                                fontWeight: FontWeight.w600,
                              ),
                        );
                      },
                      loading: () => Text(
                        blockquote.speakerName,
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              fontWeight: FontWeight.w600,
                            ),
                      ),
                      error: (error, stack) => Text(
                        blockquote.speakerName,
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              fontWeight: FontWeight.w600,
                            ),
                      ),
                    );
                  }
                  return Text(
                    blockquote.speakerName,
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                  );
                },
              ),
              const SizedBox(width: 8),
              Text(
                _formatTimeTo12Hour(
                    blockquote.startTime, blockquote.startOffsetMs),
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color:
                          Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
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

  /// Build the avatar for a blockquote (contact avatar if associated, speaker avatar otherwise)
  /// Treats everyone the same - no special "You" logic
  Widget _buildBlockquoteAvatar(
    BuildContext context,
    WidgetRef ref,
    Blockquote blockquote,
    String lifelogId,
    bool isUnknown,
    String? associatedContactId,
    List<({String speakerName, String? contactId})> orderedParticipants,
  ) {
    // If blockquote is associated with a contact, show contact avatar
    if (associatedContactId != null) {
      final contactsAsync = ref.watch(contactsProvider);
      return contactsAsync.when(
        data: (contactListResponse) {
          final contact = contactListResponse.contacts.firstWhere(
            (c) => c.id == associatedContactId,
            orElse: () => Contact(id: associatedContactId, name: 'Unknown'),
          );
          
          // Check if contact name matches speaker name (for highlighting)
          final nameMatches = _namesMatch(
            _normalizeName(blockquote.speakerName),
            _normalizeName(contact.name),
          );
          
          final avatar = ContactAvatar(
            name: contact.name,
            pictureUrl: contact.pictureUrl,
            favoriteColor: contact.favoriteColor,
            size: 24,
          );
          
          // Add green border if name matches speaker name
          if (nameMatches) {
            return Container(
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
          
          return avatar;
        },
        loading: () => const SizedBox(
          width: 24,
          height: 24,
          child: CircularProgressIndicator(strokeWidth: 2),
        ),
        error: (error, stack) => SpeakerAvatar(
          speakerName: blockquote.speakerName,
          size: 24,
        ),
      );
    }

    // If unknown and not associated, make clickable
    if (isUnknown) {
      return InkWell(
        onTap: () => _handleUnknownSpeakerClick(
          context,
          ref,
          blockquote,
          lifelogId,
          orderedParticipants,
        ),
        borderRadius: BorderRadius.circular(12),
        child: SpeakerAvatar(
          speakerName: blockquote.speakerName,
          size: 24,
        ),
      );
    }

    // Known speaker, show speaker avatar
    return SpeakerAvatar(
      speakerName: blockquote.speakerName,
      size: 24,
    );
  }


  /// Build conversation title widget for filter bar
  Widget _buildConversationTitle(BuildContext context, String title) {
    return Text(
      title,
      style: Theme.of(context).textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
      maxLines: 2,
      overflow: TextOverflow.ellipsis,
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

    return '$hour12:${minute.toString().padLeft(2, '0')} $period';
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

