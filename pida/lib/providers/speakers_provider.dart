import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/models/speaker.dart';
import 'package:pida/services/api_client.dart';

/// Unassociated speakers provider
final unassociatedSpeakersProvider = FutureProvider<List<Speaker>>((ref) async {
  final apiClient = ref.watch(apiClientProvider);
  return apiClient.getUnassociatedSpeakers();
});

