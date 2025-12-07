import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:file_picker/file_picker.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/providers/contacts_provider.dart';
import 'package:pida/providers/config_provider.dart';
import 'package:pida/routes/app_router.dart';
import 'package:pida/services/api_client.dart';
import 'package:pida/widgets/contact_avatar.dart';
import 'package:pida/widgets/error_widget.dart' as error_widget;
import 'package:pida/widgets/loading_widget.dart';
import 'package:pida/widgets/web_file_drop_zone.dart';
import 'package:pida/utils/web_file_helper.dart' if (dart.library.io) 'package:pida/utils/web_file_helper_stub.dart';

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
  final TextEditingController _userNameController = TextEditingController();
  String _searchQuery = '';
  String _debouncedSearchQuery = '';
  ContactsFilter? _cachedFilter;
  Timer? _debounceTimer;
  bool _isUploading = false;
  bool _isDragging = false;

  @override
  void initState() {
    super.initState();
    _searchController.addListener(_onSearchChanged);
    _loadUserName();
  }

  Future<void> _loadUserName() async {
    final prefs = await SharedPreferences.getInstance();
    final userName = prefs.getString('user_name');
    if (userName != null && mounted) {
      _userNameController.text = userName;
      // Update config provider
      ref.read(configProvider.notifier).state = ref.read(configProvider).copyWith(
        userName: userName,
      );
    }
  }

  Future<void> _saveUserName(String userName) async {
    final prefs = await SharedPreferences.getInstance();
    if (userName.isEmpty) {
      await prefs.remove('user_name');
    } else {
      await prefs.setString('user_name', userName);
    }
    // Update config provider
    ref.read(configProvider.notifier).state = ref.read(configProvider).copyWith(
      userName: userName.isEmpty ? null : userName,
    );
  }

  @override
  void dispose() {
    _debounceTimer?.cancel();
    _searchController.removeListener(_onSearchChanged);
    _searchController.dispose();
    _userNameController.dispose();
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
        _cachedFilter!.search != searchValue) {
      _cachedFilter = ContactsFilter(
        known: null, // No filter by known status
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
          preferredSize: const Size.fromHeight(120),
          child: Column(
            children: [
              _buildUserAndVCardRow(),
              _buildSearchBar(),
            ],
          ),
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

  Widget _buildUserAndVCardRow() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8.0, vertical: 8.0),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // "You" are: text field
          Expanded(
            child: Row(
              children: [
                const Text('"You" are:'),
                const SizedBox(width: 8),
                Expanded(
                  child: TextField(
                    controller: _userNameController,
                    decoration: InputDecoration(
                      hintText: 'Your name',
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                      ),
                      contentPadding: const EdgeInsets.symmetric(
                        horizontal: 12,
                        vertical: 8,
                      ),
                      isDense: true,
                    ),
                    onChanged: (value) {
                      _saveUserName(value);
                    },
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          // vCard drop/select area - don't expand, let it size itself
          Flexible(
            child: _buildVCardDropZone(),
          ),
        ],
      ),
    );
  }

  Widget _buildVCardDropZone() {
    if (kIsWeb) {
      return _buildWebDropZone();
    } else {
      return _buildMobileDropZone();
    }
  }

  Widget _buildWebDropZone() {
    return WebFileDropZone(
      isDragging: _isDragging,
      isUploading: _isUploading,
      onFileDropped: (file) async {
        if (mounted) {
          await _handleWebFile(file);
        }
      },
      onDragStateChanged: (dragging) {
        if (mounted) {
          setState(() {
            _isDragging = dragging;
          });
        }
      },
      onTap: _selectVCardFile,
    );
  }

  Widget _buildMobileDropZone() {
    return GestureDetector(
      onTap: _selectVCardFile,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          border: Border.all(
            color: Theme.of(context).dividerColor,
            width: 1,
          ),
          borderRadius: BorderRadius.circular(8),
          color: Theme.of(context).colorScheme.surfaceContainerHighest,
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.upload_file,
              size: 20,
              color: Theme.of(context).colorScheme.onSurface,
            ),
            const SizedBox(width: 8),
            Text(
              _isUploading ? 'Uploading...' : 'Select vCard',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _handleWebFile(dynamic file) async {
    if (!kIsWeb) return;
    
    setState(() {
      _isUploading = true;
    });

    try {
      // On web, read file using web helper
      final bytes = await readWebFileAsBytes(file);
      final fileName = getWebFileName(file);
      final apiClient = ref.read(apiClientProvider);
      final result = await apiClient.uploadVCardFromBytes(bytes, fileName);

      final imported = result['imported'] as int? ?? 0;

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('✓ Imported $imported contact(s)'),
            backgroundColor: Colors.green,
            duration: const Duration(seconds: 3),
          ),
        );

        // Refresh contacts list
        ref.invalidate(contactsFilteredProvider(_getFilter()));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Error uploading vCard: $e'),
            backgroundColor: Colors.red,
            duration: const Duration(seconds: 5),
          ),
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          _isUploading = false;
        });
      }
    }
  }

  Future<void> _selectVCardFile() async {
    try {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.custom,
        allowedExtensions: ['vcf', 'vcard'],
        withData: kIsWeb, // On web, we need bytes, not path
      );

      if (result != null && result.files.isNotEmpty) {
        final file = result.files.single;
        // On web, check for bytes; on other platforms, check for path
        if (kIsWeb) {
          if (file.bytes != null) {
            await _uploadVCardFile(file);
          } else {
            throw Exception('File bytes not available');
          }
        } else {
          // On non-web, path should be available
          if (file.path != null) {
            await _uploadVCardFile(file);
          } else {
            throw Exception('File path not available');
          }
        }
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Error selecting file: $e'),
            backgroundColor: Colors.red,
            duration: const Duration(seconds: 5),
          ),
        );
      }
    }
  }

  Future<void> _uploadVCardFile(PlatformFile file) async {
    setState(() {
      _isUploading = true;
    });

    try {
      final apiClient = ref.read(apiClientProvider);
      Map<String, dynamic> result;

      if (kIsWeb) {
        // Web: must use bytes, path is not available
        if (file.bytes == null) {
          throw Exception('File bytes not available on web');
        }
        result = await apiClient.uploadVCardFromBytes(
          file.bytes!,
          file.name,
        );
      } else {
        // Mobile/Desktop: use file path
        // Only access path on non-web platforms to avoid exceptions
        if (file.path == null) {
          throw Exception('File path not available');
        }
        result = await apiClient.uploadVCard(file.path!);
      }

      final imported = result['imported'] as int? ?? 0;

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('✓ Imported $imported contact(s)'),
            backgroundColor: Colors.green,
            duration: const Duration(seconds: 3),
          ),
        );

        // Refresh contacts list
        ref.invalidate(contactsFilteredProvider(_getFilter()));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Error uploading vCard: $e'),
            backgroundColor: Colors.red,
            duration: const Duration(seconds: 5),
          ),
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          _isUploading = false;
        });
      }
    }
  }

  Widget _buildSearchBar() {
    return Padding(
      padding: const EdgeInsets.all(8.0),
      child: TextField(
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
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          Text(
            letter,
            style: Theme.of(context).textTheme.titleSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                  color: Theme.of(context).colorScheme.onSurface,
                ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Container(
              height: 1,
              color: Theme.of(context).colorScheme.onSurface,
            ),
          ),
        ],
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
