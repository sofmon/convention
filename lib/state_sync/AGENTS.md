# StateSync - AI Agent Context

This document provides context for AI agents modifying or extending the StateSync feature.

> IMPORTANT: AI agents must treat AGENTS.md and README.md as authoritative living documents. Any change to the implementation that affects behaviors must be mirrored in both files. The code and documentation must never drift apart. When the implementation changes, these documents must be updated immediately so they always reflect the current system.

## Purpose

StateSync is a Flutter widget that provides automatic bidirectional state synchronization between a Flutter UI and a REST API backend. It handles the common pattern of periodically fetching data from a server, updating it, and keeping the UI in sync with minimal boilerplate.

## Architecture Overview

### Component Structure

```
lib/statesync/
├── ux/
│   └── main.dart          # Main implementation (3 classes)
├── README.md              # Developer documentation
└── AGENTS.md              # This file
```

### Three-Class Architecture

The implementation uses a standard Flutter pattern with three classes:

1. **`StateSync<T>` (StatefulWidget)**: Public API and configuration
2. **`_StateSyncState<T>` (State)**: Business logic and state management
3. **`_StateSyncInherited<T>` (InheritedWidget)**: Widget tree state propagation

### File: [lib/statesync/ux/main.dart](ux/main.dart)

**Lines 48-95: StateSync<T> class**
- Extends `StatefulWidget`
- Holds all configuration (callbacks, intervals, builders)
- Provides static `of<T>(context)` method for accessing state from widget tree
- Constructor parameters:
  - Optional: `getFn`, `setFn`, `initialState` (at least `getFn` or `initialState` required via assertion), `hashFn`
  - Required: `refreshInterval`, `builder`, `loadingBuilder`, `errorBuilder`
- Supports three modes: full sync (both functions), read-only (getFn only), write-only (setFn + initialState)

**Lines 97-221: _StateSyncState<T> class**
- Manages lifecycle (initState, dispose)
- Private fields:
  - `T? _cachedState`: Current state value
  - `int? _cachedHash`: Hash for change detection
  - `Timer? _refreshTimer`: Periodic refresh timer
  - `bool _isLoading`: Loading flag for initial load
  - `Object? _error`: Error from last operation
- Key methods:
  - `_fetchState()`: GET from backend (with null check), update cache if hash changed
  - `_updateState(T)`: Optimistic local update + PUT to backend (with null check)
  - `_computeHash(T)`: Hash computation using custom `hashFn` or fallback to `state.hashCode`
  - `_startRefreshTimer()`: Creates periodic timer (only if getFn available)
  - `build()`: Returns appropriate builder based on state
- `initState()` behavior:
  - If `getFn` provided: calls `_fetchState()` and `_startRefreshTimer()`
  - If only `initialState` provided: sets initial state without fetching
  - If neither: sets error (prevented by constructor assertion)

**Lines 223-240: _StateSyncInherited<T> class**
- Extends `InheritedWidget`
- Exposes `state` and `setState` down the widget tree
- `updateShouldNotify()` checks if state reference changed

**Lines 282-376: StateSyncExtension on BuildContext**
- Extension methods for convenient state access
- `getState<T>()`: Returns current state value (rebuilds on changes)
- `setState<T>(T)`: Updates state and syncs to backend
- Provides cleaner API than `StateSync.of<T>(context).state`

## Core Mechanisms

### 1. Hash-Based Caching (Lines 235-242)

```dart
int _computeHash(T state) {
  if (widget.hashFn != null) {
    return widget.hashFn!(state);
  }
  return state.hashCode;
}
```

**Why**: Prevents unnecessary rebuilds when backend returns identical data.

**How**:
- Use custom `hashFn` if provided (allows fine-tuned control)
- Otherwise fallback to `state.hashCode` (requires proper `hashCode` implementation in model)
- Compare with previous hash
- Only call `setState()` if different

**Critical**:
- If using default `hashCode`, models must override it to include all relevant fields
- Hash function must be deterministic (no timestamps, no random values)
- Custom `hashFn` can use `Object.hash()` for efficient multi-field hashing

### 2. Auto-Refresh Timer (Lines 218-226)

```dart
void _startRefreshTimer() {
  // Only start timer if getFn is available
  if (widget.getFn != null) {
    _refreshTimer = Timer.periodic(widget.refreshInterval, (_) {
      _fetchState();
    });
  }
}
```

**Why**: Keep UI synchronized with backend changes made elsewhere.

**How**:
- `Timer.periodic` calls `_fetchState()` at regular intervals
- Timer created in `initState()` only if `getFn` is provided
- Timer cancelled in `dispose()` to prevent memory leaks
- Guards against null `getFn` to support write-only mode

**Note**: Timer runs even when data hasn't changed (hash comparison prevents rebuild)

### 3. Optimistic Updates (Lines 194-226)

```dart
Future<void> _updateState(T newState) async {
  // Guard against null setFn
  if (widget.setFn == null) {
    setState(() {
      _error = Exception('setFn is not configured for this StateSync instance. Cannot update state.');
    });
    return;
  }

  try {
    // Update local cache immediately (optimistic update)
    final newHash = _computeHash(newState);
    setState(() {
      _cachedState = newState;
      _cachedHash = newHash;
      _error = null;
    });

    // Push to backend
    await widget.setFn!(newState);
  } catch (e) {
    setState(() {
      _error = e;
    });
  }
}
```

**Why**: Better UX - UI updates immediately without waiting for network

**How**:
1. Check if `setFn` is available (supports read-only mode)
2. Update local state first
3. Then call backend
4. If backend fails, set error (but keep local state)
5. Next auto-refresh will restore backend truth

**Trade-off**: Temporary inconsistency if backend rejects the update

**Read-only Mode**: If `setFn` is null, throws clear error message to user

### 4. State Branching (Lines 198-220)

The `build()` method handles three states:

```dart
if (_error != null) {
  return widget.errorBuilder(context, _error!);
}

if (_isLoading && _cachedState == null) {
  return widget.loadingBuilder(context);
}

if (_cachedState != null) {
  return _StateSyncInherited<T>(
    state: _cachedState!,
    setState: _updateState,
    child: widget.builder(context, _cachedState!),
  );
}

return widget.loadingBuilder(context);
```

**Order matters**:
1. Error takes precedence (always show errors)
2. Loading only on initial load (no cached state yet)
3. Normal state shows cached data (even during background refresh)
4. Fallback to loading (should never reach in practice)

## Operational Modes

StateSync supports three distinct operational modes based on which parameters are provided:

### 1. Full Sync Mode (Read-Write)
**Parameters**: Both `getFn` and `setFn` provided
**Behavior**:
- Fetches initial state from backend via `getFn`
- Starts auto-refresh timer
- Allows updates via `setState()` which call `setFn`
- Full bidirectional synchronization

**Use Case**: Complete CRUD operations on a resource

### 2. Read-Only Mode
**Parameters**: Only `getFn` provided, `setFn` is null
**Behavior**:
- Fetches initial state from backend via `getFn`
- Starts auto-refresh timer
- Calling `setState()` throws error: "setFn is not configured"
- Display-only, no backend modifications

**Use Case**: Viewing data without edit permissions, dashboards, monitoring

### 3. Write-Only Mode
**Parameters**: Only `setFn` and `initialState` provided, `getFn` is null
**Behavior**:
- Uses `initialState` as starting state (no fetch)
- No auto-refresh timer started
- Allows updates via `setState()` which call `setFn`
- Calling a refresh would throw error: "getFn is not configured"
- Write operations only

**Use Case**: Creating new resources, form submissions, one-way data push

### Mode Validation
The constructor includes an assertion:
```dart
assert(
  getFn != null || initialState != null,
  'Either getFn or initialState must be provided',
)
```

This ensures at least one data source exists, preventing invalid states.

## BuildContext Extension Methods

StateSync provides extension methods on `BuildContext` for cleaner, more concise state access.

### Extension API

**Location**: Lines 282-376 in `main.dart`

```dart
extension StateSyncExtension on BuildContext {
  T getState<T>();
  Future<void> setState<T>(T newState);
}
```

### Why Extension Methods?

**Problem with Traditional API**:
```dart
// Verbose and repetitive
StateSync.of<User>(context).state.name
StateSync.of<User>(context).setState(updated)
```

**Solution with Extensions**:
```dart
// Concise and idiomatic
context.getState<User>().name
context.setState<User>(updated)
```

**Benefits**:
1. **More concise**: Shorter, easier to read
2. **Idiomatic Dart**: Extension methods are common pattern
3. **Consistent with Provider**: Similar to `context.watch<T>()` and `context.read<T>()`
4. **Backward compatible**: Original `StateSync.of<T>()` still works
5. **Type-safe**: Generic type parameter ensures compile-time safety

### Why No Watch/Read Separation?

Unlike Provider, StateSync doesn't need separate `watch` vs `read` methods because:

1. **Scoped Rebuild Boundary**: `_StateSyncInherited` only wraps the `builder` child (line 259):
   ```dart
   return _StateSyncInherited<T>(
     state: _cachedState!,
     setState: _updateState,
     child: widget.builder(context, _cachedState!),
   );
   ```

2. **No External Access**: Any widget that can call `context.getState<T>()` is already inside the rebuild boundary

3. **All Descendants Rebuild Together**: When state changes, all descendants of the builder rebuild via `updateShouldNotify()` returning true

4. **No Performance Difference**: Using `getInheritedWidgetOfExactType()` vs `dependOnInheritedWidgetOfExactType()` makes no practical difference since everything rebuilds together

Therefore, a single `getState<T>()` method is sufficient.

### Implementation Details

#### getState<T>()

```dart
T getState<T>() {
  final inherited = dependOnInheritedWidgetOfExactType<_StateSyncInherited<T>>();
  assert(inherited != null, 'No StateSync<$T> found in context');
  return inherited!.state;
}
```

- Uses `dependOnInheritedWidgetOfExactType()` for consistency with existing API
- Returns just the state value (`T`), not the InheritedWidget
- Asserts if StateSync<T> not found (fails fast in debug mode)
- Widget rebuilds when state changes

#### setState<T>(T newState)

```dart
Future<void> setState<T>(T newState) {
  final inherited = dependOnInheritedWidgetOfExactType<_StateSyncInherited<T>>();
  assert(inherited != null, 'No StateSync<$T> found in context');
  return inherited!.setState(newState);
}
```

- Accesses the `setState` function from `_StateSyncInherited<T>`
- Calls `_updateState(T)` which performs optimistic update + backend sync
- Returns `Future<void>` that can be awaited
- Throws exception if in read-only mode (setFn is null)

### Usage Patterns

#### Pattern 1: Simple Read

```dart
builder: (context, user) {
  return Text('Hello ${context.getState<User>().name}');
}
```

#### Pattern 2: Simple Update

```dart
ElevatedButton(
  onPressed: () {
    final user = context.getState<User>();
    context.setState<User>(user.copyWith(name: 'New Name'));
  },
  child: Text('Update'),
)
```

#### Pattern 3: Update with Confirmation

```dart
ElevatedButton(
  onPressed: () async {
    try {
      final user = context.getState<User>();
      await context.setState<User>(user.copyWith(name: 'New Name'));
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Updated successfully')),
      );
    } catch (e) {
      print('Update failed: $e');
    }
  },
  child: Text('Update with Confirmation'),
)
```

#### Pattern 4: Nested Widget Access

```dart
builder: (context, user) {
  return Column(
    children: [
      Text('User: ${user.name}'),
      Builder(
        builder: (nestedContext) {
          // Can access state from deeply nested widgets
          final currentUser = nestedContext.getState<User>();
          return Text('Email: ${currentUser.email}');
        },
      ),
    ],
  );
}
```

### Migration Guide

**From Traditional API:**
```dart
// Old
final user = StateSync.of<User>(context).state;
StateSync.of<User>(context).setState(updated);

// New (recommended)
final user = context.getState<User>();
context.setState<User>(updated);
```

Both APIs work and will continue to be supported. The extension methods are recommended for new code due to improved readability.

## Design Decisions

### Why Generic Type Parameter T?

Allows type-safe usage with any serializable model:
```dart
StateSync<User>(...)    // T = User
StateSync<Settings>(...) // T = Settings
StateSync<Product>(...)  // T = Product
```

Each instance is type-checked at compile time.

### Why Callback-Based HTTP?

Instead of hardcoding HTTP logic, accepts `getFn` and `setFn`:

**Pros**:
- Flexible: works with any HTTP client (http, dio, etc.)
- Testable: can inject mock functions
- Reusable: same widget for different endpoints
- Configurable: can add headers, auth, etc. per instance

**Cons**:
- More boilerplate for users
- Must provide functions for each instance

**Alternative considered**: Accepting URL strings and handling HTTP internally
**Rejected because**: Less flexible, assumes REST conventions, hard to customize

### Why Optional hashFn Callback?

Instead of requiring models to implement an interface or using JSON serialization:

**Pros**:
- Flexible: users can provide custom hash function or rely on default `hashCode`
- No JSON serialization overhead for hash computation (much faster)
- Works with any model class
- Users can optimize hashing for their specific use case
- No code generation dependencies

**Cons**:
- Users must either implement `hashCode`/`==` properly or provide custom `hashFn`
- Runtime errors if hash function is non-deterministic

**How to Use**:

**Option 1: Override hashCode in model (recommended for most cases)**
```dart
class User {
  final String id;
  final String name;

  @override
  int get hashCode => Object.hash(id, name);

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is User && id == other.id && name == other.name;
}

StateSync<User>(/* no hashFn needed */);
```

**Option 2: Provide custom hashFn**
```dart
StateSync<User>(
  hashFn: (user) => Object.hash(user.id, user.name),
  // ...
)
```

**Previous Design (v1.2 and earlier)**:
- Required `toJson`/`fromJson` callbacks for JSON-based hashing
- Serialized entire object to JSON string for hash computation
- More overhead but worked without model modifications
- `fromJson` was never actually used (dead code)

**Why Changed**:
- JSON serialization was expensive (O(n) where n = object size)
- `fromJson` was never used, only required for API symmetry
- Native Dart `hashCode` is faster and more idiomatic
- Custom `hashFn` provides flexibility when needed

### Why Three Separate Builders?

Instead of a single builder with state parameter:

**Pros**:
- Clear separation of UI states
- Forces developers to handle all cases
- Better type safety (error builder gets error type)

**Cons**:
- More verbose constructor
- Cannot share widgets between states easily

**Alternative considered**: Single builder with `AsyncSnapshot<T>` pattern
**Rejected because**: Less explicit, easy to miss error handling

### Why InheritedWidget Instead of Provider?

**Decision**: Use vanilla Flutter `InheritedWidget` pattern

**Reasons**:
1. No external dependencies (project uses no state management library)
2. Simpler for single-widget state propagation
3. Standard Flutter pattern developers understand
4. Lighter weight than Provider

**Trade-off**: Less powerful than Provider (no multi-provider, no lazy loading)

### Why Optimistic Updates?

**Decision**: Update UI immediately, sync to backend after

**Reasons**:
1. Better perceived performance
2. Common pattern in modern apps
3. Auto-refresh corrects inconsistencies

**Trade-off**: Temporary inconsistency if backend rejects update

**Note**: Commented code at lines 172-173 allows fetching after PUT for verification (disabled by default for performance)

## Dependencies

### Required
- `dart:async`: For `Timer` class
- `package:flutter/material.dart`: For Flutter widgets
- `package:http` (in pubspec.yaml): For user's HTTP calls (not directly imported in StateSync)

### Not Used (Intentionally)
- No state management libraries (Provider, Riverpod, BLoC, etc.)
- No code generation (json_serializable, freezed, etc.)
- No HTTP client dependency (users provide their own functions)

## Extension Points

### Adding New Features

**1. Pause/Resume Refresh**

Add to `_StateSyncState`:
```dart
void pauseRefresh() {
  _refreshTimer?.cancel();
}

void resumeRefresh() {
  _startRefreshTimer();
}
```

Expose via `StateSync.of<T>(context).pauseRefresh()`

**2. Manual Refresh**

Add method to `_StateSyncInherited`:
```dart
final Future<void> Function() refresh;
```

Pass `_fetchState` from `_StateSyncState`

**3. Loading Indicator During Background Refresh**

Add `bool _isRefreshing` field to track background vs initial load

Modify builder signature:
```dart
Widget Function(BuildContext, T, bool isRefreshing) builder;
```

**4. Retry on Error**

Add callback to error builder:
```dart
Widget Function(BuildContext, Object, VoidCallback retry) errorBuilder;
```

Pass `_fetchState` as retry function

**5. Custom Cache Strategy**

Add optional parameter:
```dart
final bool Function(T oldState, T newState)? shouldUpdate;
```

Use in `_fetchState()` instead of hash comparison

**6. WebSocket Support**

Add optional parameter:
```dart
final Stream<T>? updateStream;
```

Listen to stream in `initState()`, update cache on events

**7. Offline Support**

Add optional parameter:
```dart
final Future<void> Function(T)? cacheFn;
final Future<T?> Function()? getCacheFn;
```

Call `cacheFn` when state updates, use `getCacheFn` as fallback in `_fetchState()`

## Common Modifications

### Changing Refresh Behavior

**Location**: Lines 191-195 (`_startRefreshTimer()`)

**Example**: Add pause when app in background
```dart
@override
void didChangeAppLifecycleState(AppLifecycleState state) {
  if (state == AppLifecycleState.paused) {
    _refreshTimer?.cancel();
  } else if (state == AppLifecycleState.resumed) {
    _startRefreshTimer();
  }
}
```

### Modifying Cache Strategy

**Location**: Lines 127-155 (`_fetchState()`)

**Example**: Always update (disable hash check)
```dart
setState(() {
  _cachedState = newState;
  _isLoading = false;
  _error = null;
});
```

### Changing Update Strategy

**Location**: Lines 158-181 (`_updateState()`)

**Example**: Pessimistic update (wait for backend before updating UI)
```dart
Future<void> _updateState(T newState) async {
  try {
    await widget.setFn(newState);  // Backend first
    setState(() {
      _cachedState = newState;     // Then UI
      _cachedHash = _computeHash(newState);
      _error = null;
    });
  } catch (e) {
    setState(() {
      _error = e;
    });
  }
}
```

## Testing Considerations

### Unit Testing `_computeHash()`

Verify hash consistency with proper `hashCode` implementation:
```dart
test('hash unchanged for identical objects', () {
  final user1 = User(id: '1', name: 'John');
  final user2 = User(id: '1', name: 'John');
  expect(user1.hashCode, equals(user2.hashCode));
});

test('hash changes when objects differ', () {
  final user1 = User(id: '1', name: 'John');
  final user2 = User(id: '1', name: 'Jane');
  expect(user1.hashCode, isNot(equals(user2.hashCode)));
});
```

### Widget Testing with Mock Functions

```dart
testWidgets('shows loading then data', (tester) async {
  await tester.pumpWidget(
    StateSync<User>(
      getFn: () async {
        await Future.delayed(Duration(milliseconds: 100));
        return User(name: 'Test');
      },
      setFn: (_) async {},
      // ... other params
    ),
  );

  expect(find.byType(CircularProgressIndicator), findsOneWidget);
  await tester.pump(Duration(milliseconds: 100));
  expect(find.text('Test'), findsOneWidget);
});
```

### Integration Testing

Test with real HTTP calls using mock server:
```dart
final mockServer = MockWebServer();
await mockServer.start();

StateSync<User>(
  getFn: () async {
    final response = await http.get(mockServer.url);
    return User.fromJson(jsonDecode(response.body));
  },
  // ...
)
```

## Performance Characteristics

### Time Complexity
- `_fetchState()`: O(1) for hash comparison (if using default hashCode or simple custom hashFn)
- `_updateState()`: O(1) for hash computation
- `_computeHash()`: O(1) with default hashCode, O(k) with custom hashFn where k = number of fields hashed
- `build()`: O(1) (just conditionals)

### Space Complexity
- O(n) where n = size of state object T
- One copy in `_cachedState`
- One integer for `_cachedHash`

### Network Usage
- GET request every `refreshInterval`
- PUT request on each `setState()` call
- No request batching or debouncing

### Rebuild Optimization
- Only rebuilds when hash changes
- Background refreshes don't trigger rebuild if data identical
- InheritedWidget prevents rebuilding unaffected widgets

## Known Limitations

1. **No request cancellation**: If widget disposed during fetch, request continues
2. **No retry logic**: Failed requests wait until next refresh (or throw error in write-only mode)
3. **No exponential backoff**: Refresh interval is constant
4. **No request deduplication**: Multiple instances fetch independently
5. **No conflict resolution**: Last write wins on concurrent updates
6. **No offline queue**: Updates fail immediately without network
7. **No partial updates**: Must send entire object on PUT
8. **No optimistic update rollback**: Failed updates keep optimistic state
9. **Mode-specific limitations**:
   - Read-only mode: Cannot programmatically trigger updates
   - Write-only mode: Cannot fetch latest state, relies on provided `initialState`
   - Write-only mode: Refresh timer never starts (no periodic updates)

## Future Considerations

### Breaking Changes to Avoid

- Changing generic type constraint (would break existing code)
- Removing required parameters (would break existing code)
- Changing `StateSync.of<T>()` return type (would break existing code)

### Safe Additions

- Adding optional parameters with defaults
- Adding new methods to `_StateSyncInherited`
- Adding new private methods to `_StateSyncState`
- Adding new lifecycle hooks

### Migration Path for Major Changes

If breaking changes needed:
1. Create `StateSync2<T>` as new class
2. Deprecate `StateSync<T>` with warnings
3. Provide migration guide in README
4. Keep both for at least one major version

## Related Patterns

### Similar Flutter Patterns
- `FutureBuilder`: Single async operation, no auto-refresh
- `StreamBuilder`: Continuous stream, no REST integration
- `Provider`: State management, no built-in sync

### When to Use StateSync
- REST API with periodic polling
- CRUD operations on single resources
- Real-time data with acceptable latency (seconds)

### When NOT to Use StateSync
- WebSocket/SSE real-time updates (use StreamBuilder)
- One-time fetch (use FutureBuilder)
- Complex state management (use Provider/BLoC)
- List pagination (StateSync is for single objects)
- GraphQL subscriptions (use GraphQL client)

## Debugging Tips

### Common Issues

**Issue**: Widget rebuilds too often
- **Check**: `hashCode` implementation is deterministic (doesn't include timestamps, computed values, etc.)
- **Debug**: Print hash values in `_computeHash()` to verify consistency

**Issue**: State not updating
- **Check**: `setFn()` is actually making backend call
- **Debug**: Add logging in `_updateState()`

**Issue**: "No StateSync found in context"
- **Check**: Calling `StateSync.of<T>()` from within builder or descendant
- **Debug**: Print widget tree with `debugDumpApp()`

**Issue**: Memory leak / timer not stopping
- **Check**: `dispose()` is being called
- **Debug**: Add print in `dispose()`, check it's called

### Debug Logging

Add to `_StateSyncState`:
```dart
void _log(String message) {
  if (kDebugMode) {
    print('[StateSync<$T>] $message');
  }
}
```

Then log in key methods:
```dart
Future<void> _fetchState() async {
  _log('Fetching state...');
  try {
    final newState = await widget.getFn();
    final newHash = _computeHash(newState);
    _log('Fetched state with hash: $newHash');
    // ...
  } catch (e) {
    _log('Error: $e');
    // ...
  }
}
```

## Version History

- **v1.3** (2025-01-19): Replaced JSON serialization with native hashCode
  - **BREAKING CHANGE**: Removed `fromJson` and `toJson` required parameters
  - Added optional `hashFn` parameter for custom hash functions
  - `_computeHash()` now uses `hashFn` if provided, otherwise falls back to `state.hashCode`
  - Removed `dart:convert` dependency (no longer needed for hashing)
  - **Performance improvement**: Much faster hash computation (O(1) vs O(n) for JSON serialization)
  - **Removed dead code**: `fromJson` was never used internally
  - **Migration guide**:
    - Remove `fromJson` and `toJson` parameters from StateSync constructor
    - Either implement `hashCode` and `==` operators in your models, or provide custom `hashFn`
    - Models still need `toJson()`/`fromJson()` for HTTP serialization, but not for StateSync
  - Updated README.md and AGENTS.md with new patterns and examples

- **v1.2** (2025-01-19): BuildContext extension methods
  - Added `StateSyncExtension` on `BuildContext`
  - Added `context.getState<T>()` method for concise state access
  - Added `context.setState<T>(T)` method for concise state updates
  - Extension methods provide cleaner API than `StateSync.of<T>(context).state`
  - No watch/read separation needed (all descendants rebuild together)
  - Backward compatible - traditional `StateSync.of<T>()` still supported

- **v1.1** (2025-01-19): Optional getFn/setFn support
  - Made `getFn` and `setFn` optional parameters
  - Added `initialState` parameter for write-only mode
  - Added constructor assertion requiring `getFn` or `initialState`
  - Added null guards in `_fetchState()` and `_updateState()`
  - Updated `_startRefreshTimer()` to check for `getFn` availability
  - Updated `initState()` to handle three operational modes
  - **Breaking Change**: `getFn` and `setFn` are now nullable types
  - Supports three modes: full sync, read-only, write-only

- **v1.0** (2025-01-19): Initial implementation
  - Generic StateSync<T> widget
  - Hash-based caching
  - Auto-refresh timer
  - Optimistic updates
  - Three-builder pattern (loading/error/success)
  - InheritedWidget for state access

## Contact & Maintenance

Part of the Ingreed project by Sofmon.

For modifications:
- Ensure backward compatibility or provide migration path
- Update both README.md and AGENTS.md
- Add tests for new functionality
- Update version history in this file
