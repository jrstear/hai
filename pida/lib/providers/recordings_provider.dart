import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/models/recording.dart';
import 'package:pida/services/api_client.dart';

/// Speaker recordings provider
final speakerRecordingsProvider = FutureProvider.family<List<Segment>, String>(
  (ref, speakerId) async {
    final apiClient = ref.watch(apiClientProvider);
    return apiClient.getSpeakerRecordings(speakerId);
  },
);

