import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:pida/routes/app_router.dart';

/// Responsive navigation widget
/// 
/// Uses BottomNavigationBar on mobile and NavigationRail on web/desktop
class AppNavigation extends StatelessWidget {
  final int selectedIndex;
  final Widget child;

  const AppNavigation({
    super.key,
    required this.selectedIndex,
    required this.child,
  });

  @override
  Widget build(BuildContext context) {
    final isMobile = MediaQuery.of(context).size.width < 600;

    if (isMobile) {
      return Scaffold(
        body: child,
        bottomNavigationBar: _buildBottomNavigationBar(context),
      );
    } else {
      return Scaffold(
        body: Row(
          children: [
            _buildNavigationRail(context),
            Expanded(child: child),
          ],
        ),
      );
    }
  }

  Widget _buildBottomNavigationBar(BuildContext context) {
    return BottomNavigationBar(
      currentIndex: selectedIndex,
      onTap: (index) => _navigateToIndex(context, index),
      type: BottomNavigationBarType.fixed,
      items: const [
        BottomNavigationBarItem(
          icon: Icon(Icons.people_outline),
          activeIcon: Icon(Icons.people),
          label: 'People',
        ),
        BottomNavigationBarItem(
          icon: Icon(Icons.calendar_today_outlined),
          activeIcon: Icon(Icons.calendar_today),
          label: 'Calendar',
        ),
        BottomNavigationBarItem(
          icon: Icon(Icons.chat_bubble_outline),
          activeIcon: Icon(Icons.chat_bubble),
          label: 'Conversation',
        ),
        BottomNavigationBarItem(
          icon: Icon(Icons.check_circle_outline),
          activeIcon: Icon(Icons.check_circle),
          label: 'Todo',
        ),
      ],
    );
  }

  Widget _buildNavigationRail(BuildContext context) {
    return NavigationRail(
      selectedIndex: selectedIndex,
      onDestinationSelected: (index) => _navigateToIndex(context, index),
      labelType: NavigationRailLabelType.all,
      destinations: const [
        NavigationRailDestination(
          icon: Icon(Icons.people_outline),
          selectedIcon: Icon(Icons.people),
          label: Text('People'),
        ),
        NavigationRailDestination(
          icon: Icon(Icons.calendar_today_outlined),
          selectedIcon: Icon(Icons.calendar_today),
          label: Text('Calendar'),
        ),
        NavigationRailDestination(
          icon: Icon(Icons.chat_bubble_outline),
          selectedIcon: Icon(Icons.chat_bubble),
          label: Text('Conversation'),
        ),
        NavigationRailDestination(
          icon: Icon(Icons.check_circle_outline),
          selectedIcon: Icon(Icons.check_circle),
          label: Text('Todo'),
        ),
      ],
    );
  }

  void _navigateToIndex(BuildContext context, int index) {
    switch (index) {
      case 0:
        context.go(AppRoutes.people);
        break;
      case 1:
        context.go(AppRoutes.calendar);
        break;
      case 2:
        context.go(AppRoutes.conversation);
        break;
      case 3:
        context.go(AppRoutes.todo);
        break;
    }
  }
}

/// Get navigation index from route path
int getNavigationIndex(String path) {
  if (path.startsWith(AppRoutes.people)) return 0;
  if (path.startsWith(AppRoutes.calendar)) return 1;
  if (path.startsWith(AppRoutes.conversation)) return 2;
  if (path.startsWith(AppRoutes.todo)) return 3;
  return 0; // Default to People
}

