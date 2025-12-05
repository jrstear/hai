import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/providers/contacts_provider.dart';
import 'package:pida/routes/app_router.dart';
import 'package:pida/widgets/contact_avatar.dart';
import 'package:pida/widgets/error_widget.dart' as error_widget;
import 'package:pida/widgets/loading_widget.dart';

/// People screen (Contacts page)
/// 
/// Displays a native-style contacts list with:
/// - Alphabetical section headers (A, B, C, etc.)
/// - Circular profile pictures with initials fallback
/// - Known status indicators (green check for known, red X for unknown)
/// - Search bar with real-time filtering
/// - Pull-to-refresh support
/// - Contact row tap to show recordings
class PeopleScreen extends ConsumerStatefulWidget {
  const PeopleScreen({super.key});

  @override
  ConsumerState<PeopleScreen> createState() => _PeopleScreenState();
}

class _PeopleScreenState extends ConsumerState<PeopleScreen> {
  final TextEditingController _searchController = TextEditingController();
  String _searchQuery = '';
  String _debouncedSearchQuery = '';
  bool? _knownFilter;
  ContactsFilter? _cachedFilter;
  Timer? _debounceTimer;

  @override
  void initState() {
    super.initState();
    _searchController.addListener(_onSearchChanged);
  }

  @override
  void dispose() {
    _debounceTimer?.cancel();
    _searchController.removeListener(_onSearchChanged);
    _searchController.dispose();
    super.dispose();
  }

  void _onSearchChanged() {
    final value = _searchController.text;
    if (value != _searchQuery) {
      setState(() {
        _searchQuery = value;
      });
      
      // Cancel previous timer
      _debounceTimer?.cancel();
      
      // Debounce: Update debounced query after 500ms of no typing
      _debounceTimer = Timer(const Duration(milliseconds: 500), () {
        if (mounted && _searchController.text == value) {
          setState(() {
            _debouncedSearchQuery = value;
            _cachedFilter = null; // Reset cache when search changes
          });
        }
      });
    }
  }

  ContactsFilter _getFilter() {
    // Use debounced search query for API calls
    final searchValue = _debouncedSearchQuery.isEmpty ? null : _debouncedSearchQuery;
    
    // Only create new filter if values actually changed
    if (_cachedFilter == null ||
        _cachedFilter!.known != _knownFilter ||
        _cachedFilter!.search != searchValue) {
      _cachedFilter = ContactsFilter(
        known: _knownFilter,
        search: searchValue,
      );
    }
    
    return _cachedFilter!;
  }

  @override
  Widget build(BuildContext context) {
    // Use memoized filter to avoid recreating on every build
    final filter = _getFilter();
    final contactsAsync = ref.watch(contactsFilteredProvider(filter));

    return Scaffold(
      appBar: AppBar(
        title: const Text('People'),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(60),
          child: _buildSearchBar(),
        ),
      ),
      body: contactsAsync.when(
        data: (response) {
          if (response.contacts.isEmpty) {
            return _buildEmptyState();
          }

          // Group contacts by first letter
          final groupedContacts = _groupContactsByLetter(response.contacts);

          return RefreshIndicator(
            onRefresh: () async {
              ref.invalidate(contactsFilteredProvider(filter));
              await Future.delayed(const Duration(milliseconds: 500));
            },
            child: _buildContactsList(groupedContacts),
          );
        },
        loading: () => const LoadingWidget(message: 'Loading contacts...'),
        error: (error, stack) => error_widget.ErrorDisplayWidget(
          message: 'Failed to load contacts: ${error.toString()}',
          onRetry: () => ref.invalidate(contactsFilteredProvider(filter)),
        ),
      ),
    );
  }

  Widget _buildSearchBar() {
    return Padding(
      padding: const EdgeInsets.all(8.0),
      child: Column(
        children: [
          TextField(
            controller: _searchController,
            decoration: InputDecoration(
              hintText: 'Search contacts...',
              prefixIcon: const Icon(Icons.search),
              suffixIcon: _searchQuery.isNotEmpty
                  ? IconButton(
                      icon: const Icon(Icons.clear),
                      onPressed: () {
                        _searchController.clear();
                        setState(() {
                          _searchQuery = '';
                          _debouncedSearchQuery = '';
                          _cachedFilter = null; // Reset cache
                        });
                      },
                    )
                  : null,
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
              ),
              contentPadding: const EdgeInsets.symmetric(
                horizontal: 16,
                vertical: 12,
              ),
            ),
            // Search is handled by controller listener with debouncing
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              FilterChip(
                label: const Text('All'),
                selected: _knownFilter == null,
                onSelected: (selected) {
                  setState(() {
                    _knownFilter = null;
                    _cachedFilter = null; // Reset cache when filter changes
                  });
                },
              ),
              const SizedBox(width: 8),
              FilterChip(
                label: const Text('Known'),
                selected: _knownFilter == true,
                onSelected: (selected) {
                  setState(() {
                    _knownFilter = selected ? true : null;
                    _cachedFilter = null; // Reset cache when filter changes
                  });
                },
              ),
              const SizedBox(width: 8),
              FilterChip(
                label: const Text('Unknown'),
                selected: _knownFilter == false,
                onSelected: (selected) {
                  setState(() {
                    _knownFilter = selected ? false : null;
                    _cachedFilter = null; // Reset cache when filter changes
                  });
                },
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.people_outline,
            size: 64,
            color: Colors.grey[400],
          ),
          const SizedBox(height: 16),
          Text(
            'No contacts found',
            style: Theme.of(context).textTheme.titleLarge?.copyWith(
                  color: Colors.grey[600],
                ),
          ),
          const SizedBox(height: 8),
          Text(
            _searchQuery.isNotEmpty
                ? 'Try adjusting your search'
                : 'Contacts will appear here',
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: Colors.grey[500],
                ),
          ),
        ],
      ),
    );
  }

  Map<String, List<Contact>> _groupContactsByLetter(List<Contact> contacts) {
    final Map<String, List<Contact>> grouped = {};

    for (final contact in contacts) {
      final firstLetter = contact.name.isNotEmpty
          ? contact.name[0].toUpperCase()
          : '#';

      // Group non-alphabetic characters under '#'
      final sectionKey = RegExp(r'[A-Z]').hasMatch(firstLetter)
          ? firstLetter
          : '#';

      grouped.putIfAbsent(sectionKey, () => []).add(contact);
    }

    // Sort contacts within each section
    for (final key in grouped.keys) {
      grouped[key]!.sort((a, b) => a.name.compareTo(b.name));
    }

    return grouped;
  }

  Widget _buildContactsList(Map<String, List<Contact>> groupedContacts) {
    // Sort section keys: letters first, then '#'
    final sortedKeys = groupedContacts.keys.toList()
      ..sort((a, b) {
        if (a == '#') return 1;
        if (b == '#') return -1;
        return a.compareTo(b);
      });

    return ListView.builder(
      itemCount: sortedKeys.fold<int>(
        0,
        (sum, key) => sum + groupedContacts[key]!.length + 1, // +1 for header
      ),
      itemBuilder: (context, index) {
        // Find which section this index belongs to
        int currentIndex = 0;
        String? currentSection;
        int contactIndex = 0;

        for (final key in sortedKeys) {
          final sectionLength = groupedContacts[key]!.length + 1;
          if (index < currentIndex + sectionLength) {
            currentSection = key;
            contactIndex = index - currentIndex - 1; // -1 for header
            break;
          }
          currentIndex += sectionLength;
        }

        if (currentSection == null) return const SizedBox.shrink();

        final contacts = groupedContacts[currentSection]!;

        // First item in section is the header
        if (contactIndex == -1) {
          return _buildSectionHeader(currentSection);
        }

        // Then the contact items
        final contact = contacts[contactIndex];
        return _buildContactRow(contact);
      },
    );
  }

  Widget _buildSectionHeader(String letter) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      color: Colors.grey[200],
      child: Text(
        letter,
        style: Theme.of(context).textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.bold,
              color: Colors.grey[700],
            ),
      ),
    );
  }

  Widget _buildContactRow(Contact contact) {
    return ListTile(
      leading: ContactAvatar(
        name: contact.name,
        pictureUrl: contact.pictureUrl,
        favoriteColor: contact.favoriteColor,
        size: 48,
      ),
      title: Text(contact.name),
      subtitle: contact.email != null && contact.email!.isNotEmpty
          ? Text(contact.email!)
          : contact.phone != null && contact.phone!.isNotEmpty
              ? Text(contact.phone!)
              : null,
      trailing: Icon(
        contact.known ? Icons.check_circle : Icons.cancel,
        color: contact.known ? Colors.green : Colors.red,
        size: 24,
      ),
      onTap: () {
        // Navigate to calendar screen
        // TODO: When recordings screen is implemented (hai-uev), navigate there instead
        // or add contact filter parameter to calendar screen
        context.go(AppRoutes.calendar);
        
        // Show a brief message about the contact
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Showing calendar for ${contact.name}'),
            duration: const Duration(seconds: 2),
          ),
        );
      },
    );
  }
}
