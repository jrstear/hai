import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/providers/contacts_provider.dart';
import 'package:pida/providers/filter_provider.dart';
import 'package:pida/providers/lifelog_provider.dart';
import 'package:pida/routes/app_router.dart';
import 'package:pida/services/api_client.dart';
import 'package:pida/widgets/error_widget.dart';
import 'package:pida/widgets/filter_bar.dart';
import 'package:pida/widgets/loading_widget.dart';
import 'package:pida/widgets/people_filter_display.dart';
import 'package:pida/widgets/contact_avatar.dart';
import 'package:pida/widgets/people_selector.dart';
import 'package:pida/widgets/speaker_avatar.dart';
import 'package:pida/widgets/time_filter.dart';
import 'package:pida/providers/config_provider.dart';

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
      // Only scroll to bottom if scroll position hasn't been set yet (initial load)
      // Check if position is at or near the top (within 100 pixels)
      if (_scrollController.position.pixels <= 100) {
        _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
        _hasScrolledToBottom = true;
      }
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

  /// Build "You" avatar for the app user
  Widget _buildYouAvatar(BuildContext context, String? userName) {
    final displayName = userName ?? 'You';
    return Container(
      padding: const EdgeInsets.all(2), // Border width
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        border: Border.all(
          color: Theme.of(context).colorScheme.primary,
          width: 2,
        ),
      ),
      child: ContactAvatar(
        name: displayName,
        size: 32,
      ),
    );
  }

  /// Build contact avatar with optional highlighting border when name matches speaker name
  Widget _buildContactAvatarWithHighlight({
    required Contact contact,
    required bool highlight,
    required double size,
  }) {
    final avatar = ContactAvatar(
      name: contact.name,
      pictureUrl: contact.pictureUrl,
      favoriteColor: contact.favoriteColor,
      size: size,
    );
    
    if (highlight) {
      return Container(
        padding: const EdgeInsets.all(2), // Border width
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          border: Border.all(
            color: Colors.green, // Or use theme color - could be configurable
            width: 2,
          ),
        ),
        child: avatar,
      );
    }
    
    return avatar;
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
      body: Column(
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
    );
  }


  Widget _buildConversationsList(
      BuildContext context, List<ConversationSummary> summaries) {
    return ListView.builder(
      controller: _scrollController,
      reverse: false, // Chronological order (earliest first)
      physics: const AlwaysScrollableScrollPhysics(), // Ensure scrolling is always enabled
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
      onTap: () async {
        // Navigate to Conversation screen with lifelog_id and date
        final selectedDate = ref.read(calendarDateFilterProvider) ?? DateTime.now();
        await context.push(
            '${AppRoutes.conversation}?lifelog_id=${summary.lifelogId}&date=${_formatDate(selectedDate)}');
        // After returning from conversation, refresh participants for this conversation
        // This ensures contact associations made in conversation are reflected in calendar view
        // Uses optimized endpoint that only fetches participants, not all blockquotes
        if (mounted) {
          try {
            final apiClient = ref.read(apiClientProvider);
            final participants = await apiClient.getLifelogParticipants(summary.lifelogId);
            // Update conversation participants provider with fresh data
            ref.read(conversationParticipantsProvider(summary.lifelogId).notifier).state = participants;
            
            // Also invalidate the lifelog provider so the summary is regenerated with updated contact IDs
            // This ensures the summary.participantContactIds reflects the current state
            final dateStr = _formatDate(selectedDate);
            ref.invalidate(lifelogProvider(dateStr));
          } catch (e) {
            // Log error but don't block UI - participant data is optional for display
            // ignore: avoid_print
            print('Failed to refresh participants for conversation ${summary.lifelogId}: $e');
          }
        }
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
              child: Builder(
                builder: (context) {
                      // Check if we have updated participants from the provider
                      final providerParticipants = ref.watch(conversationParticipantsProvider(summary.lifelogId));
                      final contactIdsToUse = providerParticipants.isNotEmpty 
                          ? providerParticipants 
                          : summary.participantContactIds;
                      
                      // Build list of avatars to display
                      final contactsAsync = ref.watch(contactsProvider);
                      final userName = ref.watch(userNameProvider);
                      
                      return contactsAsync.when(
                        data: (contactListResponse) {
                          final avatarWidgets = <Widget>[];
                          
                          // Add "You" avatar if user is a participant
                          if (summary.hasUser) {
                            avatarWidgets.add(
                              Container(
                                margin: const EdgeInsets.only(right: 4),
                                child: _buildYouAvatar(context, userName),
                              ),
                            );
                          }
                          
                          // Add contact avatars for associated contacts
                          if (contactIdsToUse.isNotEmpty) {
                            final participantContacts = contactIdsToUse
                                .take(3 - (summary.hasUser ? 1 : 0)) // Reserve space for "You"
                                .map((id) => contactListResponse.contacts.firstWhere(
                                      (contact) => contact.id == id,
                                      orElse: () => Contact(id: id, name: 'Unknown'),
                                    ))
                                .toList();
                            
                            for (final contact in participantContacts) {
                              // Check if contact name matches any speaker name (for highlighting)
                              final nameMatches = summary.participantNames.any((speakerName) =>
                                  _namesMatch(_normalizeName(speakerName), _normalizeName(contact.name)));
                              
                              avatarWidgets.add(
                                Container(
                                  margin: const EdgeInsets.only(right: 4),
                                  child: _buildContactAvatarWithHighlight(
                                    contact: contact,
                                    highlight: nameMatches,
                                    size: 32,
                                  ),
                                ),
                              );
                            }
                          }
                          
                          // Add speaker avatars for non-associated speakers (fallback)
                          if (avatarWidgets.length < 3 && summary.participantNames.isNotEmpty) {
                            final remainingSlots = 3 - avatarWidgets.length;
                            final speakerNamesToShow = summary.participantNames
                                .where((name) {
                                  final normalized = _normalizeName(name);
                                  return normalized != 'you' && normalized != 'unknown';
                                })
                                .take(remainingSlots)
                                .toList();
                            
                            for (final name in speakerNamesToShow) {
                              avatarWidgets.add(
                                Container(
                                  margin: const EdgeInsets.only(right: 4),
                                  child: SpeakerAvatar(
                                    speakerName: name,
                                    size: 32,
                                  ),
                                ),
                              );
                            }
                          }
                          
                          final totalParticipants = 
                              (summary.hasUser ? 1 : 0) + 
                              contactIdsToUse.length + 
                              summary.participantNames.where((n) {
                                final normalized = _normalizeName(n);
                                return normalized != 'you' && normalized != 'unknown';
                              }).length;
                          
                          return Row(
                            children: [
                              ...avatarWidgets,
                              if (totalParticipants > 3)
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
                                      '+${totalParticipants - 3}',
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
                          );
                        },
                        loading: () => Row(
                            children: [
                              ...summary.participantNames.take(3).map((name) {
                                return Container(
                                  margin: const EdgeInsets.only(right: 4),
                                  child: SpeakerAvatar(
                                    speakerName: name,
                                    size: 32,
                                  ),
                                );
                              }),
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
                        error: (error, stack) => Row(
                            children: [
                              ...summary.participantNames.take(3).map((name) {
                                return Container(
                                  margin: const EdgeInsets.only(right: 4),
                                  child: SpeakerAvatar(
                                    speakerName: name,
                                    size: 32,
                                  ),
                                );
                              }),
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
                        );
                    },
                  ),
            ),
          ],
        ),
      ),
    );
  }
}

