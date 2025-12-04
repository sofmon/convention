# StateSync Widget

A Flutter widget that synchronizes state between your UI and a REST API backend with intelligent caching, automatic refresh, and optimistic updates.

## Features

- **Automatic Synchronization**: Periodically fetches fresh data from your backend
- **Smart Caching**: Uses hash-based change detection to prevent unnecessary rebuilds
- **Optimistic Updates**: UI updates immediately while syncing to backend
- **Type-Safe**: Fully generic implementation with type safety for your models
- **State Management**: Handles loading, error, and success states automatically
- **Configurable**: Customize refresh intervals, serialization, and HTTP operations per instance
- **Flexible Modes**: Supports read-only, write-only, or full read-write synchronization

## Installation

Add the http dependency to your `pubspec.yaml`:

```yaml
dependencies:
  http: ^1.2.0
```

## Basic Usage

```dart
import 'package:lib/statesync/ux/main.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';

// Your model class
class User {
  final String id;
  final String name;
  final String email;

  User({required this.id, required this.name, required this.email});

  factory User.fromJson(Map<String, dynamic> json) => User(
    id: json['id'],
    name: json['name'],
    email: json['email'],
  );

  Map<String, dynamic> toJson() => {
    'id': id,
    'name': name,
    'email': email,
  };

  User copyWith({String? name, String? email}) => User(
    id: id,
    name: name ?? this.name,
    email: email ?? this.email,
  );

  // Override hashCode for StateSync change detection
  @override
  int get hashCode => Object.hash(id, name, email);

  // Override == operator for proper equality comparison
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is User &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          name == other.name &&
          email == other.email;
}

// Using StateSync
class UserProfile extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return StateSync<User>(
      // Fetch function (HTTP GET)
      getFn: () async {
        final response = await http.get(
          Uri.parse('https://api.example.com/user/123'),
        );
        if (response.statusCode == 200) {
          return User.fromJson(jsonDecode(response.body));
        }
        throw Exception('Failed to load user');
      },

      // Update function (HTTP PUT)
      setFn: (user) async {
        final response = await http.put(
          Uri.parse('https://api.example.com/user/123'),
          headers: {'Content-Type': 'application/json'},
          body: jsonEncode(user.toJson()),
        );
        if (response.statusCode != 200) {
          throw Exception('Failed to update user');
        }
      },

      // Auto-refresh every 30 seconds
      refreshInterval: Duration(seconds: 30),

      // Loading state
      loadingBuilder: (context) => Center(
        child: CircularProgressIndicator(),
      ),

      // Error state
      errorBuilder: (context, error) => Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.error, color: Colors.red, size: 48),
            SizedBox(height: 16),
            Text('Error: $error'),
          ],
        ),
      ),

      // Success state
      builder: (context, user) {
        return Column(
          children: [
            Text('Name: ${user.name}'),
            Text('Email: ${user.email}'),
            SizedBox(height: 20),
            ElevatedButton(
              onPressed: () {
                // Update state - will sync to backend automatically
                final updatedUser = user.copyWith(
                  name: 'Updated Name',
                );
                StateSync.of<User>(context).setState(updatedUser);
              },
              child: Text('Update Name'),
            ),
          ],
        );
      },
    );
  }
}
```

## Accessing State

StateSync provides state access through an InheritedWidget pattern with two convenient APIs:

### Extension Methods (Recommended)

The most concise way to access and update state using BuildContext extensions:

```dart
// Reading state
builder: (context, user) {
  return SomeChildWidget(
    child: Builder(
      builder: (context) {
        // Access the current state with extension method
        final currentUser = context.getState<User>();
        return Text('Hello ${currentUser.name}');
      },
    ),
  );
}

// Updating state
ElevatedButton(
  onPressed: () {
    final user = context.getState<User>();
    final updatedUser = user.copyWith(name: 'New Name');

    // This will:
    // 1. Update the UI immediately (optimistic update)
    // 2. Call setFn to sync to backend
    // 3. Handle errors if the backend call fails
    context.setState<User>(updatedUser);
  },
  child: Text('Update'),
)

// Awaiting updates for confirmation
ElevatedButton(
  onPressed: () async {
    try {
      final user = context.getState<User>();
      await context.setState<User>(user.copyWith(name: 'New Name'));
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Updated successfully')),
      );
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Update failed: $e')),
      );
    }
  },
  child: Text('Update with Confirmation'),
)
```

### Traditional API

You can also use the traditional `StateSync.of<T>()` approach:

```dart
// Reading state
final currentUser = StateSync.of<User>(context).state;

// Updating state
final syncState = StateSync.of<User>(context);
syncState.setState(updatedUser);
```

## How It Works

### 1. Initialization
When StateSync is created:
- Immediately calls `getFn()` to fetch initial state
- Starts a periodic timer to auto-refresh based on `refreshInterval`
- Shows `loadingBuilder` until first fetch completes

### 2. Caching & Change Detection
StateSync uses hash-based caching:
- Computes hash of state objects using `hashCode` or custom `hashFn`
- Only triggers rebuild if hash changes
- Prevents unnecessary rebuilds when data is identical

```dart
// Internal implementation
int _computeHash(T state) {
  if (widget.hashFn != null) {
    return widget.hashFn!(state);  // Use custom hash function
  }
  return state.hashCode;  // Fall back to object's hashCode
}
```

**Important**: Make sure your model classes properly override `hashCode` and `==` operators to include all relevant fields. Alternatively, provide a custom `hashFn` when creating StateSync.

### 3. Auto-Refresh
- Timer periodically calls `getFn()` in the background
- If data hasn't changed (same hash), no rebuild occurs
- If data changed, updates cache and rebuilds UI
- Timer is cancelled when widget is disposed

### 4. Optimistic Updates
When you call `setState()`:
1. UI updates immediately with new state
2. `setFn()` is called to sync to backend
3. If backend call fails, error is set (but state remains)
4. Next auto-refresh will restore correct backend state

## Usage Modes

StateSync supports three different modes of operation:

### 1. Full Sync Mode (Read-Write)

The standard mode with both `getFn` and `setFn`:

```dart
StateSync<User>(
  getFn: () async {
    final response = await http.get(Uri.parse('https://api.example.com/user'));
    return User.fromJson(jsonDecode(response.body));
  },
  setFn: (user) async {
    await http.put(
      Uri.parse('https://api.example.com/user'),
      body: jsonEncode(user.toJson()),
    );
  },
  // Optional: provide custom hash function for better control
  hashFn: (user) => Object.hash(user.id, user.name, user.email),
  refreshInterval: Duration(seconds: 30),
  loadingBuilder: (context) => CircularProgressIndicator(),
  errorBuilder: (context, error) => Text('Error: $error'),
  builder: (context, user) => Text('Hello ${user.name}'),
)
```

### 2. Read-Only Mode

Omit `setFn` when you only need to display data without updates:

```dart
StateSync<User>(
  // Only getFn - no setFn
  getFn: () async {
    final response = await http.get(Uri.parse('https://api.example.com/user'));
    return User.fromJson(jsonDecode(response.body));
  },
  refreshInterval: Duration(seconds: 30),
  loadingBuilder: (context) => CircularProgressIndicator(),
  errorBuilder: (context, error) => Text('Error: $error'),
  builder: (context, user) {
    return Column(
      children: [
        Text('Name: ${user.name}'),
        // Attempting to call setState() will throw an error
        // context.setState<User>(updated); // ❌ Will error
      ],
    );
  },
)
```

**Note**: If you try to call `setState()` in read-only mode, you'll get an error: `"setFn is not configured for this StateSync instance. Cannot update state."`

### 3. Write-Only Mode

Omit `getFn` and provide `initialState` when you want to update data without fetching:

```dart
StateSync<User>(
  // No getFn - provide initialState instead
  initialState: User(id: '123', name: 'John', email: 'john@example.com'),
  setFn: (user) async {
    await http.put(
      Uri.parse('https://api.example.com/user'),
      body: jsonEncode(user.toJson()),
    );
  },
  refreshInterval: Duration(seconds: 30), // Timer won't start without getFn
  loadingBuilder: (context) => CircularProgressIndicator(),
  errorBuilder: (context, error) => Text('Error: $error'),
  builder: (context, user) {
    return Column(
      children: [
        Text('Name: ${user.name}'),
        ElevatedButton(
          onPressed: () {
            final updated = user.copyWith(name: 'Updated Name');
            context.setState<User>(updated); // ✅ Will work
          },
          child: Text('Update'),
        ),
      ],
    );
  },
)
```

**Important**:
- When using write-only mode, you must provide `initialState`
- The refresh timer will not start (no automatic fetching)
- You can update state via `setState()`, but cannot trigger fetches

## Advanced Usage

### Custom Error Recovery

```dart
StateSync<User>(
  getFn: _fetchUser,
  setFn: _updateUser,
  refreshInterval: Duration(seconds: 30),

  errorBuilder: (context, error) {
    return Column(
      children: [
        Text('Error: $error'),
        ElevatedButton(
          onPressed: () {
            // Manually trigger a retry
            // The next auto-refresh will attempt again
          },
          child: Text('Retry'),
        ),
      ],
    );
  },

  // ... other builders
);
```

### Different Refresh Intervals

```dart
// Fast refresh for critical data
StateSync<StockPrice>(
  refreshInterval: Duration(seconds: 5),
  // ...
)

// Slow refresh for static data
StateSync<UserSettings>(
  refreshInterval: Duration(minutes: 5),
  // ...
)
```

### Nested StateSync Widgets

You can use multiple StateSync widgets for different data types:

```dart
Column(
  children: [
    StateSync<User>(
      // User data syncing
    ),
    StateSync<Settings>(
      // Settings data syncing
    ),
    StateSync<Analytics>(
      // Analytics data syncing
    ),
  ],
)
```

Each instance maintains its own state, cache, and refresh timer independently.

## Best Practices

### 1. Implement copyWith() for Your Models

This makes updates cleaner:

```dart
class User {
  final String name;
  final String email;

  User copyWith({String? name, String? email}) => User(
    name: name ?? this.name,
    email: email ?? this.email,
  );
}

// Usage
final updated = user.copyWith(name: 'New Name');
StateSync.of<User>(context).setState(updated);
```

### 2. Handle HTTP Errors Properly

```dart
getFn: () async {
  final response = await http.get(uri);

  if (response.statusCode == 200) {
    return User.fromJson(jsonDecode(response.body));
  } else if (response.statusCode == 404) {
    throw Exception('User not found');
  } else if (response.statusCode >= 500) {
    throw Exception('Server error');
  } else {
    throw Exception('Request failed: ${response.statusCode}');
  }
}
```

### 3. Use Appropriate Refresh Intervals

- Critical real-time data: 5-10 seconds
- User-facing data: 30-60 seconds
- Background data: 2-5 minutes
- Static configuration: 10+ minutes

### 4. Implement Proper Equality

For StateSync's change detection to work correctly, ensure your models properly override `hashCode` and `==`:

```dart
class User {
  final String id;
  final String name;
  final String email;

  // Override hashCode to include all relevant fields
  @override
  int get hashCode => Object.hash(id, name, email);

  // Override == for proper equality comparison
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is User &&
          id == other.id &&
          name == other.name &&
          email == other.email;
}
```

**Alternatively**, provide a custom `hashFn`:

```dart
StateSync<User>(
  hashFn: (user) => Object.hash(user.id, user.name, user.email),
  // ... other parameters
)
```

## Performance Considerations

- **Hash Computation**: Uses `hashCode` by default (very fast) or custom `hashFn` if provided
- **Memory**: Keeps one copy of state in memory
- **Network**: Makes periodic GET requests based on `refreshInterval`
- **Rebuilds**: Only rebuilds when hash changes, not on every refresh

**Performance Tips**:
- Default `hashCode` is fastest for simple models
- Use custom `hashFn` with `Object.hash()` for fine-tuned control
- Avoid expensive computations in your hash function

## Troubleshooting

### Widget doesn't update after setState()

Check that your model's `hashCode` includes all relevant fields. If fields are missing from the hash calculation, changes won't be detected.

```dart
// Bad - only hashes id
@override
int get hashCode => id.hashCode;

// Good - hashes all fields
@override
int get hashCode => Object.hash(id, name, email);
```

### Too many rebuilds

Ensure your `hashCode` implementation is deterministic. Don't include timestamps, random values, or computed properties that change between calls.

### "No StateSync<T> found in context" error

Make sure you're calling `StateSync.of<T>(context)` from within the `builder` callback or a descendant widget.

### Backend updates not persisting

Check your `setFn()` implementation. Ensure it's actually making the HTTP PUT/POST request and handling errors.

## API Reference

### StateSync Constructor Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `getFn` | `Future<T> Function()?` | No* | Fetches state from backend (HTTP GET) |
| `setFn` | `Future<void> Function(T)?` | No | Updates state on backend (HTTP PUT) |
| `initialState` | `T?` | No* | Initial state when getFn is not provided |
| `hashFn` | `int Function(T)?` | No | Custom hash function for change detection. Falls back to `state.hashCode` if not provided |
| `refreshInterval` | `Duration` | Yes | Time between automatic refreshes |
| `builder` | `Widget Function(BuildContext, T)` | Yes | Builds UI with loaded state |
| `loadingBuilder` | `Widget Function(BuildContext)` | Yes | Builds UI during initial load |
| `errorBuilder` | `Widget Function(BuildContext, Object)` | Yes | Builds UI when error occurs |
| `key` | `Key?` | No | Widget key |

\* **Note**: Either `getFn` or `initialState` must be provided (at least one is required)

### Extension Methods on BuildContext

**`T context.getState<T>()`**
- Gets the current state value for type T
- Widget rebuilds when state changes
- Throws AssertionError if StateSync<T> not found

**`Future<void> context.setState<T>(T newState)`**
- Updates state locally (optimistic) and syncs to backend
- Returns Future that completes when backend sync finishes
- Throws AssertionError if StateSync<T> not found
- Throws Exception if in read-only mode (setFn is null)

### Traditional API: StateSync.of<T>(context)

Returns an `_StateSyncInherited<T>` object with:

- `T state`: Current state value
- `Future<void> setState(T)`: Updates state locally and syncs to backend

## License

Part of the convention project.
