import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/providers/contacts_provider.dart';
import 'package:pida/providers/filter_provider.dart';
import 'package:pida/widgets/contact_avatar.dart';

/// People selector drawer widget
/// 
/// Slides in from the right (1/3 screen width, partial screen, never full).
/// Used for selecting people to add to filters or conversation participants.
/// 
/// **Sections (context-dependent):**
/// - **On Calendar/Todo pages:** Two sections
///   1. "People in current filter" (top, sorted, pre-checked)
///   2. "All other people" (bottom, sorted)
/// 
/// **Interactions:**
/// - Checkbox click: Add/remove from filter immediately (does NOT close drawer)
/// - Row click: Add to filter + close drawer (slide out to right)
/// - Search bar: Filter list by name substring (real-time)
/// - Close button: Close drawer (slide out to right)
class PeopleSelector extends ConsumerStatefulWidget {
  /// Optional list of contact IDs already selected (for calendar/todo filters)
  /// If provided, these will appear in the "People in filter" section
  final List<String>? selectedContactIds;

  /// Optional list of contact IDs for conversation participants
  /// If provided, these will appear in the "Conversation participants" section
  final List<String>? participantContactIds;

  /// Context: 'filter' for calendar/todo pages, 'conversation' for conversation page
  final String context;

  /// Callback when a contact is selected (checkbox or row click)
  final void Function(String contactId, bool isSelected)? onContactSelected;

  /// Callback when drawer should close
  final VoidCallback? onClose;

  const PeopleSelector({
    super.key,
    this.selectedContactIds,
    this.participantContactIds,
    this.context = 'filter',
    this.onContactSelected,
    this.onClose,
  });

  @override
  ConsumerState<PeopleSelector> createState() => _PeopleSelectorState();
}

class _PeopleSelectorState extends ConsumerState<PeopleSelector>
    with SingleTickerProviderStateMixin {
  final TextEditingController _searchController = TextEditingController();
  String _searchQuery = '';
  late AnimationController _animationController;
  late Animation<Offset> _slideAnimation;

  @override
  void initState() {
    super.initState();
    _animationController = AnimationController(
      duration: const Duration(milliseconds: 300),
      vsync: this,
    );
    _slideAnimation = Tween<Offset>(
      begin: const Offset(1.0, 0.0), // Start off-screen to the right
      end: Offset.zero, // End at normal position
    ).animate(CurvedAnimation(
      parent: _animationController,
      curve: Curves.easeOut,
    ));
    _animationController.forward();
    _searchController.addListener(_onSearchChanged);
  }

  @override
  void dispose() {
    _searchController.removeListener(_onSearchChanged);
    _searchController.dispose();
    _animationController.dispose();
    super.dispose();
  }

  void _onSearchChanged() {
    setState(() {
      _searchQuery = _searchController.text.trim().toLowerCase();
    });
  }

  Future<void> _handleClose() async {
    await _animationController.reverse();
    widget.onClose?.call();
  }

  void _handleContactToggle(String contactId, bool isCurrentlySelected) {
    final newSelection = !isCurrentlySelected;
    widget.onContactSelected?.call(contactId, newSelection);
  }

  void _handleContactRowTap(String contactId, bool isCurrentlySelected) {
    // Add if not selected, then close
    if (!isCurrentlySelected) {
      widget.onContactSelected?.call(contactId, true);
    }
    _handleClose();
  }

  @override
  Widget build(BuildContext context) {
    final screenWidth = MediaQuery.of(context).size.width;
    final drawerWidth = screenWidth / 3; // 1/3 screen width

    return SlideTransition(
      position: _slideAnimation,
      child: Container(
        width: drawerWidth,
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surface,
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.2),
              blurRadius: 10,
              spreadRadius: 2,
            ),
          ],
        ),
        child: Column(
          children: [
            // Top bar: Close button (left) + Search bar (right)
            _buildTopBar(context),

            // Divider
            const Divider(height: 1),

            // Scrollable list of people
            Expanded(
              child: _buildPeopleList(context),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTopBar(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        children: [
          // Close button (left)
          IconButton(
            icon: const Icon(Icons.close),
            onPressed: _handleClose,
            tooltip: 'Close',
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(),
          ),

          const SizedBox(width: 8),

          // Search bar (right, expands to fill)
          Expanded(
            child: TextField(
              controller: _searchController,
              decoration: InputDecoration(
                hintText: 'Search people...',
                prefixIcon: const Icon(Icons.search, size: 20),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: BorderSide(
                    color: Theme.of(context).dividerColor,
                  ),
                ),
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 8,
                ),
                isDense: true,
              ),
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPeopleList(BuildContext context) {
    final contactsAsync = ref.watch(contactsProvider);

    return contactsAsync.when(
      data: (contactListResponse) {
        final allContacts = contactListResponse.contacts;

        // Determine which contacts are in the "top section"
        List<String> topSectionIds;
        String topSectionTitle;
        if (widget.context == 'conversation' && widget.participantContactIds != null) {
          topSectionIds = widget.participantContactIds!;
          topSectionTitle = 'Conversation participants';
        } else if (widget.context == 'filter' && widget.selectedContactIds != null) {
          topSectionIds = widget.selectedContactIds!;
          topSectionTitle = 'People in current filter';
        } else {
          topSectionIds = [];
          topSectionTitle = 'People in current filter';
        }

        // Filter contacts by search query
        final filteredContacts = allContacts.where((contact) {
          if (_searchQuery.isEmpty) return true;
          return contact.name.toLowerCase().contains(_searchQuery);
        }).toList();

        // Split into top section and bottom section
        final topSectionContacts = filteredContacts
            .where((contact) => topSectionIds.contains(contact.id))
            .toList();
        final bottomSectionContacts = filteredContacts
            .where((contact) => !topSectionIds.contains(contact.id))
            .toList();

        // Sort both sections alphabetically
        topSectionContacts.sort((a, b) =>
            a.name.toLowerCase().compareTo(b.name.toLowerCase()));
        bottomSectionContacts.sort((a, b) =>
            a.name.toLowerCase().compareTo(b.name.toLowerCase()));

        return ListView(
          padding: const EdgeInsets.all(8),
          children: [
            // Top section: People in filter/participants
            if (topSectionContacts.isNotEmpty) ...[
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
                child: Text(
                  topSectionTitle,
                  style: Theme.of(context).textTheme.titleSmall?.copyWith(
                        color: Theme.of(context).colorScheme.onSurface.withOpacity(0.7),
                        fontWeight: FontWeight.w600,
                      ),
                ),
              ),
              ...topSectionContacts.map((contact) =>
                  _buildContactRow(context, contact, true)),
              const SizedBox(height: 8),
              const Divider(height: 1),
              const SizedBox(height: 8),
            ],

            // Bottom section: All other people
            if (bottomSectionContacts.isNotEmpty) ...[
              if (topSectionContacts.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
                  child: Text(
                    'All other people',
                    style: Theme.of(context).textTheme.titleSmall?.copyWith(
                          color: Theme.of(context).colorScheme.onSurface.withOpacity(0.7),
                          fontWeight: FontWeight.w600,
                        ),
                  ),
                ),
              ...bottomSectionContacts.map((contact) =>
                  _buildContactRow(context, contact, false)),
            ],

            // Empty state
            if (filteredContacts.isEmpty)
              Padding(
                padding: const EdgeInsets.all(32),
                child: Center(
                  child: Text(
                    _searchQuery.isEmpty
                        ? 'No contacts found'
                        : 'No contacts match "$_searchQuery"',
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: Theme.of(context).colorScheme.onSurface.withOpacity(0.6),
                        ),
                  ),
                ),
              ),
          ],
        );
      },
      loading: () => const Center(
        child: CircularProgressIndicator(),
      ),
      error: (error, stack) => Center(
        child: Text(
          'Failed to load contacts: ${error.toString()}',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: Theme.of(context).colorScheme.error,
              ),
        ),
      ),
    );
  }

  Widget _buildContactRow(
    BuildContext context,
    Contact contact,
    bool isInTopSection,
  ) {
    // Watch the actual filter state if we're in filter context
    // This ensures checkbox updates immediately when filter changes
    final isSelected = widget.context == 'filter'
        ? ref.watch(calendarPeopleFilterProvider).contains(contact.id)
        : (widget.selectedContactIds?.contains(contact.id) ?? false);

    return InkWell(
      onTap: () => _handleContactRowTap(contact.id, isSelected),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
        child: Row(
          children: [
            // Checkbox column
            Checkbox(
              value: isSelected,
              onChanged: (value) {
                _handleContactToggle(contact.id, isSelected);
              },
            ),

            const SizedBox(width: 8),

            // Picture column
            ContactAvatar(
              name: contact.name,
              pictureUrl: contact.pictureUrl,
              favoriteColor: contact.favoriteColor,
              size: 40,
            ),

            const SizedBox(width: 12),

            // Name column (expands to fill)
            Expanded(
              child: Text(
                contact.name,
                style: Theme.of(context).textTheme.bodyMedium,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Show people selector as an overlay drawer
/// 
/// Creates an overlay that shows the people selector drawer sliding in from the right.
/// The drawer takes up 1/3 of the screen width.
Future<void> showPeopleSelector({
  required BuildContext context,
  List<String>? selectedContactIds,
  List<String>? participantContactIds,
  String contextType = 'filter',
  void Function(String contactId, bool isSelected)? onContactSelected,
}) async {
  final overlayState = Overlay.of(context);
  late OverlayEntry overlayEntry;
  bool isRemoving = false;

  void close() {
    if (isRemoving) return;
    isRemoving = true;
    // Small delay to allow animation to start, then remove overlay
    Future.delayed(const Duration(milliseconds: 300), () {
      overlayEntry.remove();
    });
  }

  overlayEntry = OverlayEntry(
    builder: (context) {
      // Calculate top offset to position drawer below filter bar
      // AppBar height + FilterBar height (approximately 56px + 56px = 112px)
      final mediaQuery = MediaQuery.of(context);
      final appBarHeight = kToolbarHeight; // Standard AppBar height
      final filterBarHeight = 56.0; // Approximate FilterBar height (padding + content)
      final topOffset = appBarHeight + filterBarHeight;
      
      return Stack(
        children: [
          // Semi-transparent backdrop (only below filter bar)
          Positioned(
            left: 0,
            right: 0,
            top: topOffset,
            bottom: 0,
            child: GestureDetector(
              onTap: close,
              child: Container(
                color: Colors.black.withOpacity(0.3),
              ),
            ),
          ),

          // People selector drawer (right side, below filter bar)
          Positioned(
            right: 0,
            top: topOffset,
            bottom: 0,
            child: PeopleSelector(
              selectedContactIds: selectedContactIds,
              participantContactIds: participantContactIds,
              context: contextType,
              onContactSelected: onContactSelected,
              onClose: close,
            ),
          ),
        ],
      );
    },
  );

  overlayState.insert(overlayEntry);
}

