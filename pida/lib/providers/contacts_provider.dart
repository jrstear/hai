import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/services/api_client.dart';

/// Contacts list provider
final contactsProvider = FutureProvider<ContactListResponse>((ref) async {
  final apiClient = ref.watch(apiClientProvider);
  return apiClient.getContacts();
});

/// Contacts list provider with filters
final contactsFilteredProvider = FutureProvider.family<ContactListResponse, ContactsFilter>(
  (ref, filter) async {
    final apiClient = ref.watch(apiClientProvider);
    return apiClient.getContacts(
      known: filter.known,
      search: filter.search,
    );
  },
);

/// Filter parameters for contacts
class ContactsFilter {
  final bool? known;
  final String? search;

  ContactsFilter({
    this.known,
    this.search,
  });
}

/// Single contact provider
final contactProvider = FutureProvider.family<Contact, String>(
  (ref, contactId) async {
    final apiClient = ref.watch(apiClientProvider);
    return apiClient.getContact(contactId);
  },
);

