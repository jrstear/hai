import 'package:flutter/material.dart';

/// Filter bar widget - Framework for displaying filters at the top of screens
/// 
/// This is a flexible container that displays:
/// - Left content (typically time/date filter)
/// - Right content (typically people filter)
/// 
/// Supports smart wrapping for different screen sizes and content lengths.
/// Used across Calendar, Conversation, Todo, and other screens.
class FilterBar extends StatelessWidget {
  /// Widget to display on the left side (e.g., date selector)
  final Widget? leftContent;
  
  /// Widget to display on the right side (e.g., people filter)
  final Widget? rightContent;
  
  /// Optional callback for horizontal swipe gestures (left/right)
  final void Function(DragEndDetails)? onHorizontalDragEnd;
  
  /// Padding around the filter bar content
  final EdgeInsets padding;
  
  /// Whether to show bottom border
  final bool showBorder;

  const FilterBar({
    super.key,
    this.leftContent,
    this.rightContent,
    this.onHorizontalDragEnd,
    this.padding = const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
    this.showBorder = true,
  });

  @override
  Widget build(BuildContext context) {
    Widget content = Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        // Left content (time/date filter) - takes available space, but allows right content to reach edge
        if (leftContent != null)
          Flexible(
            child: leftContent!,
            flex: 2,
          ),
        
        // Spacer between left and right content
        if (leftContent != null && rightContent != null)
          const SizedBox(width: 8),
        
        // Right content (people filter) - right-aligned to edge
        if (rightContent != null)
          Expanded(
            child: Align(
              alignment: Alignment.centerRight,
              child: rightContent!,
            ),
          ),
      ],
    );

    // Wrap with GestureDetector if swipe gestures are needed
    if (onHorizontalDragEnd != null) {
      content = GestureDetector(
        behavior: HitTestBehavior.opaque,
        onHorizontalDragEnd: onHorizontalDragEnd,
        child: content,
      );
    }

    // Adjust padding to allow right content to reach edge
    final adjustedPadding = EdgeInsets.only(
      left: padding.left,
      right: 0, // No right padding so + button can reach edge
      top: padding.top,
      bottom: padding.bottom,
    );
    
    return Container(
      padding: adjustedPadding,
      decoration: showBorder
          ? BoxDecoration(
              color: Theme.of(context).colorScheme.surface,
              border: Border(
                bottom: BorderSide(
                  color: Theme.of(context).dividerColor,
                  width: 1,
                ),
              ),
            )
          : BoxDecoration(
              color: Theme.of(context).colorScheme.surface,
            ),
      child: content,
    );
  }
}

