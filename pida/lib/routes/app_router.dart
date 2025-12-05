import 'package:go_router/go_router.dart';
import 'package:pida/screens/people_screen.dart';
import 'package:pida/screens/conversation_screen.dart';
import 'package:pida/screens/calendar_screen.dart';
import 'package:pida/screens/todo_screen.dart';
import 'package:pida/widgets/app_navigation.dart';

/// App routes configuration
/// 
/// Navigation structure (left to right):
/// - People: Contacts page with three-section layout
/// - Calendar: Month/week/day views with conversation blocks (future: Google/Apple calendar integration)
/// - Conversation: Single conversation view (slides in from calendar day view)
/// - Todo: Task management (future feature)
class AppRoutes {
  static const String calendar = '/calendar';
  static const String conversation = '/conversation';
  static const String todo = '/todo';
  static const String people = '/people';

  // Route names for navigation
  static const String calendarName = 'calendar';
  static const String conversationName = 'conversation';
  static const String todoName = 'todo';
  static const String peopleName = 'people';
}

/// App router configuration with shell route for navigation
final appRouter = GoRouter(
  initialLocation: AppRoutes.people, // Start at People
  routes: [
    ShellRoute(
      builder: (context, state, child) {
        final selectedIndex = getNavigationIndex(state.uri.path);
        return AppNavigation(
          selectedIndex: selectedIndex,
          child: child,
        );
      },
      routes: [
        GoRoute(
          path: AppRoutes.calendar,
          name: AppRoutes.calendarName,
          builder: (context, state) => const CalendarScreen(),
        ),
        GoRoute(
          path: AppRoutes.conversation,
          name: AppRoutes.conversationName,
          builder: (context, state) {
            final lifelogId = state.uri.queryParameters['lifelog_id'];
            final date = state.uri.queryParameters['date'];
            return ConversationScreen(lifelogId: lifelogId, date: date);
          },
        ),
        GoRoute(
          path: AppRoutes.todo,
          name: AppRoutes.todoName,
          builder: (context, state) => const TodoScreen(),
        ),
        GoRoute(
          path: AppRoutes.people,
          name: AppRoutes.peopleName,
          builder: (context, state) => const PeopleScreen(),
        ),
      ],
    ),
  ],
);

