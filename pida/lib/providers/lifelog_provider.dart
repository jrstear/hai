import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/models/lifelog.dart';
import 'package:pida/services/api_client.dart';
import 'package:pida/utils/loading_state.dart';

/// Lifelog data provider for a specific date
final lifelogProvider = FutureProvider.family<LifelogResponse, String>(
  (ref, date) async {
    final apiClient = ref.watch(apiClientProvider);
    return apiClient.getLifelogs(date);
  },
);

/// Lifelog loading state provider (for manual state management if needed)
final lifelogLoadingStateProvider =
    StateNotifierProvider.family<LifelogLoadingStateNotifier, LoadingState<LifelogResponse>, String>(
  (ref, date) => LifelogLoadingStateNotifier(ref, date),
);

class LifelogLoadingStateNotifier extends StateNotifier<LoadingState<LifelogResponse>> {
  final Ref _ref;
  final String _date;

  LifelogLoadingStateNotifier(this._ref, this._date) : super(const LoadingState.initial()) {
    loadLifelog();
  }

  Future<void> loadLifelog() async {
    state = const LoadingState.loading();
    try {
      final apiClient = _ref.read(apiClientProvider);
      final response = await apiClient.getLifelogs(_date);
      state = LoadingState.success(response);
    } catch (e) {
      state = LoadingState.error(e.toString());
    }
  }

  void reset() {
    state = const LoadingState.initial();
  }
}

/// Conversation summary model for day view
class ConversationSummary {
  final String lifelogId;
  final String? title;
  final String startTime; // HH:MM format
  final List<String> participantNames; // Speaker names (fallback)
  final List<String> participantContactIds; // Contact IDs (when associated)
  final bool hasUser; // Whether "You" is a participant
  final String summary; // First few words or title
  final ConversationTiming? timing;

  ConversationSummary({
    required this.lifelogId,
    this.title,
    required this.startTime,
    required this.participantNames,
    required this.participantContactIds,
    required this.hasUser,
    required this.summary,
    this.timing,
  });
}

/// Helper to extract conversation summaries from LifelogResponse
List<ConversationSummary> extractConversationSummaries(LifelogResponse response) {
  final summaries = <ConversationSummary>[];
  
  // Group blockquotes by lifelog_id
  final grouped = response.grouped;
  
  for (final entry in grouped.entries) {
    final lifelogId = entry.key;
    final blockquotes = entry.value;
    
    if (blockquotes.isEmpty) continue;
    
    // Sort by start time to get first blockquote
    blockquotes.sort((a, b) => a.startTime.compareTo(b.startTime));
    final firstBlockquote = blockquotes.first;
    
    // Extract start time and convert to 12-hour AM/PM format in local time
    final startTime = _formatTimeTo12Hour(firstBlockquote.startTime, firstBlockquote.startOffsetMs);
    
    // Get unique participant contact IDs (when blockquotes are associated with contacts)
    final participantContactIds = blockquotes
        .where((bq) => bq.contactId != null && bq.contactId!.isNotEmpty)
        .map((bq) => bq.contactId!)
        .toSet()
        .toList();
    
    // Get unique participant names (speaker names, fallback when no contact ID)
    final participantNames = blockquotes
        .map((bq) => bq.speakerName)
        .toSet()
        .toList();
    
    // Check if "You" is a participant (speaker_name is "You" or speakerIdentifier is "user")
    final hasUser = blockquotes.any((bq) {
      final speakerName = bq.speakerName.toLowerCase().trim();
      return speakerName == 'you';
    });
    
    // Get summary (title or first few words of first blockquote)
    final summary = firstBlockquote.lifelogTitle ?? 
        (firstBlockquote.content.length > 50 
            ? '${firstBlockquote.content.substring(0, 50)}...'
            : firstBlockquote.content);
    
    // Get timing if available
    final timing = response.conversationTimings[lifelogId];
    
    summaries.add(ConversationSummary(
      lifelogId: lifelogId,
      title: firstBlockquote.lifelogTitle,
      startTime: startTime,
      participantNames: participantNames,
      participantContactIds: participantContactIds,
      hasUser: hasUser,
      summary: summary,
      timing: timing,
    ));
  }
  
  // Sort by start time (chronological) - use original timestamp for sorting
  summaries.sort((a, b) {
    final aTime = a.timing?.startMs ?? 0;
    final bTime = b.timing?.startMs ?? 0;
    return aTime.compareTo(bTime);
  });
  
  return summaries;
}

/// Format time to 12-hour AM/PM format in local timezone
/// 
/// Takes the time string (HH:MM:SS) and the Unix timestamp in milliseconds
/// to convert to local time and format as 12-hour with AM/PM
String _formatTimeTo12Hour(String timeStr, int timestampMs) {
  // Parse the time string to get hours/minutes
  final parts = timeStr.split(':');
  if (parts.length < 2) return timeStr;
  
  // Use the timestamp to get the actual date/time in local timezone
  final dateTime = DateTime.fromMillisecondsSinceEpoch(timestampMs, isUtc: true).toLocal();
  
  // Format as 12-hour AM/PM
  final hour = dateTime.hour;
  final minute = dateTime.minute;
  
  final period = hour >= 12 ? 'PM' : 'AM';
  final hour12 = hour == 0 ? 12 : (hour > 12 ? hour - 12 : hour);
  
  return '${hour12}:${minute.toString().padLeft(2, '0')} $period';
}

