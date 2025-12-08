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
        // Left content (time/date filter) - takes only space it needs
        if (leftContent != null)
          leftContent!,
        
        // Spacer pushes right content to the edge
        if (leftContent != null && rightContent != null)
          const Spacer(),
        
        // Right content (people filter) - at right edge
        if (rightContent != null)
          rightContent!,
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

    // Remove right padding so + button can reach the edge
    final adjustedPadding = EdgeInsets.only(
      left: padding.left,
      right: 0,
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

