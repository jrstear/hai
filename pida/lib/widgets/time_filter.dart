import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:pida/providers/filter_provider.dart';

/// Time/Date filter widget
/// 
/// Displays the selected date with navigation controls:
/// - Date display with < > buttons for previous/next day
/// - Swipe gestures support (handled at filter bar level via FilterBar)
/// - Click date to open date picker
/// - Connects to calendarDateFilterProvider for persistent state
/// 
/// Used in the filter bar on calendar screen to navigate between days.
class TimeFilter extends ConsumerWidget {
  /// Optional callback when date is changed via date picker
  final void Function(DateTime)? onDateChanged;

  const TimeFilter({
    super.key,
    this.onDateChanged,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectedDate = ref.watch(calendarDateFilterProvider) ?? DateTime.now();

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        // Previous day button
        IconButton(
          icon: const Icon(Icons.arrow_back_ios),
          onPressed: () {
            goToPreviousCalendarDay(ref);
          },
          tooltip: 'Previous day',
          constraints: const BoxConstraints(),
          padding: const EdgeInsets.all(8),
        ),
        
        // Date display with date picker - only takes space it needs, left-aligned
        InkWell(
          onTap: () async {
            final DateTime? picked = await showDatePicker(
              context: context,
              initialDate: selectedDate,
              firstDate: DateTime(2000),
              lastDate: DateTime(2100),
            );
            if (picked != null) {
              setCalendarDate(ref, picked);
              onDateChanged?.call(picked);
            }
          },
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                DateFormat('EEE MM/dd').format(selectedDate),
                style: Theme.of(context).textTheme.titleMedium,
                overflow: TextOverflow.ellipsis,
                maxLines: 1,
              ),
              const SizedBox(width: 4),
              const Icon(Icons.calendar_today, size: 18),
            ],
          ),
        ),
        
        // Next day button
        IconButton(
          icon: const Icon(Icons.arrow_forward_ios),
          onPressed: () {
            goToNextCalendarDay(ref);
          },
          tooltip: 'Next day',
          constraints: const BoxConstraints(),
          padding: const EdgeInsets.all(8),
        ),
      ],
    );
  }
}

