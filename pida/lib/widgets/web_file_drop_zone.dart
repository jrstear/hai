import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:pida/utils/web_file_helper.dart' if (dart.library.io) 'package:pida/utils/web_file_helper_stub.dart';

/// Web-specific file drop zone widget
/// Handles HTML5 drag and drop for file uploads
class WebFileDropZone extends StatefulWidget {
  final bool isDragging;
  final bool isUploading;
  final Function(dynamic) onFileDropped;
  final Function(bool) onDragStateChanged;
  final VoidCallback onTap;

  const WebFileDropZone({
    super.key,
    required this.isDragging,
    required this.isUploading,
    required this.onFileDropped,
    required this.onDragStateChanged,
    required this.onTap,
  });

  @override
  State<WebFileDropZone> createState() => _WebFileDropZoneState();
}

class _WebFileDropZoneState extends State<WebFileDropZone> {
  final GlobalKey _key = GlobalKey();
  bool _isListening = false;
  bool _mouseOverDropZone = false;
  bool _draggingValidFile = false;

  @override
  void initState() {
    super.initState();
    if (kIsWeb) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _attachDragHandlers();
      });
    }
  }


  @override
  void dispose() {
    if (kIsWeb && _isListening) {
      _removeDragHandlers();
    }
    super.dispose();
  }

  void _attachDragHandlers() {
    if (!kIsWeb || _isListening) return;

    try {
      // Add document-level listeners for drag and drop
      final document = _getDocument();
      if (document == null) return;

      // Prevent default on document to stop downloads, but allow our handlers to process
      document.addEventListener('dragover', _handleDragOver, true); // Use capture phase
      document.addEventListener('dragleave', _handleDragLeave, true);
      document.addEventListener('drop', _handleDrop, true); // Use capture phase
      
      _isListening = true;
    } catch (e) {
      // Ignore errors if html library not available
    }
  }

  void _removeDragHandlers() {
    if (!kIsWeb || !_isListening) return;

    try {
      final document = _getDocument();
      if (document == null) return;

      document.removeEventListener('dragover', _handleDragOver, true);
      document.removeEventListener('dragleave', _handleDragLeave, true);
      document.removeEventListener('drop', _handleDrop, true);
      
      _isListening = false;
    } catch (e) {
      // Ignore errors
    }
  }

  dynamic _getDocument() {
    if (!kIsWeb) return null;
    // Use the helper function which handles conditional imports
    return getDocument();
  }

  void _handleDragOver(dynamic e) {
    e.preventDefault();
    e.stopPropagation();
    
    // Check if files are valid vCard files
    final files = e.dataTransfer?.files;
    bool hasValidFile = false;
    if (files != null && files.length > 0) {
      final file = files[0];
      final fileName = (file.name as String?)?.toLowerCase() ?? '';
      hasValidFile = fileName.endsWith('.vcf') || fileName.endsWith('.vcard');
    }
    
    // Update dragging state
    if (hasValidFile != _draggingValidFile) {
      _draggingValidFile = hasValidFile;
    }
    
    // Check if we're over the drop zone and update highlight
    final isOver = _checkIfOverDropZone(e);
    if (isOver != _mouseOverDropZone) {
      _mouseOverDropZone = isOver;
    }
    
    _updateHighlightState();
  }

  void _handleDragLeave(dynamic e) {
    e.preventDefault();
    e.stopPropagation();
    // Clear dragging state when drag leaves
    if (_draggingValidFile) {
      _draggingValidFile = false;
      _mouseOverDropZone = false;
      _updateHighlightState();
    }
  }

  void _updateHighlightState() {
    // Highlight only if dragging a valid file AND mouse is over drop zone
    final shouldHighlight = _draggingValidFile && _mouseOverDropZone;
    if (shouldHighlight != widget.isDragging) {
      widget.onDragStateChanged(shouldHighlight);
    }
  }

  /// Check if drag event is over the drop zone using simple bounds check
  bool _checkIfOverDropZone(dynamic e) {
    try {
      if (!mounted) return false;
      
      final renderObject = _key.currentContext?.findRenderObject();
      if (renderObject == null || renderObject is! RenderBox) return false;

      // Get mouse position from event (viewport coordinates)
      final clientX = (e.clientX as num?)?.toDouble() ?? 0;
      final clientY = (e.clientY as num?)?.toDouble() ?? 0;

      // Get widget bounds in screen coordinates
      final box = renderObject;
      final position = box.localToGlobal(Offset.zero);
      final size = box.size;

      // Get viewport scroll to convert screen to viewport coordinates
      final scrollX = (e.view?.scrollX as num?)?.toDouble() ?? 0;
      final scrollY = (e.view?.scrollY as num?)?.toDouble() ?? 0;
      
      // Convert widget position from screen to viewport coordinates
      final viewportX = position.dx - scrollX;
      final viewportY = position.dy - scrollY;

      // Check if mouse is within widget bounds (in viewport coordinates)
      return clientX >= viewportX &&
          clientX <= viewportX + size.width &&
          clientY >= viewportY &&
          clientY <= viewportY + size.height;
    } catch (e) {
      return false;
    }
  }

  void _handleDrop(dynamic e) {
    e.preventDefault();
    e.stopPropagation();
    
    // Clear dragging state on drop
    _draggingValidFile = false;
    _updateHighlightState();

    try {
      final files = e.dataTransfer?.files;
      if (files != null && files.length > 0) {
        final file = files[0];
        final fileName = (file.name as String?)?.toLowerCase() ?? '';
        if (fileName.endsWith('.vcf') || fileName.endsWith('.vcard')) {
          // Process the drop - upload the file
          widget.onFileDropped(file);
        }
      }
    } catch (error) {
      // Error handling - the callback should handle showing errors to user
      print('Error handling dropped file: $error');
    }
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: widget.onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          key: _key,
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          decoration: BoxDecoration(
            border: Border.all(
              color: widget.isDragging
                  ? Theme.of(context).colorScheme.primary
                  : Theme.of(context).dividerColor,
              width: widget.isDragging ? 2 : 1,
            ),
            borderRadius: BorderRadius.circular(8),
            color: widget.isDragging
                ? Theme.of(context).colorScheme.primaryContainer.withValues(alpha: 0.3)
                : Theme.of(context).colorScheme.surfaceContainerHighest,
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.upload_file,
                size: 18,
                color: widget.isDragging
                    ? Theme.of(context).colorScheme.primary
                    : Theme.of(context).colorScheme.onSurface,
              ),
              const SizedBox(width: 6),
              Text(
                widget.isUploading
                    ? 'Uploading...'
                    : 'Drop vCard or click',
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      fontSize: 14,
                      color: widget.isDragging
                          ? Theme.of(context).colorScheme.primary
                          : Theme.of(context).colorScheme.onSurface,
                    ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
