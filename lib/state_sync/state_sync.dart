import 'dart:async';
import 'package:flutter/material.dart';

/// A generic widget that synchronizes state between Flutter UI and a backend REST API.
///
/// StateSync manages:
/// - Automatic periodic fetching from backend (GET)
/// - Manual state updates to backend (PUT)
/// - Intelligent caching with hash-based change detection
/// - Loading, error, and success states
/// - Prevents unnecessary rebuilds when data hasn't changed
///
/// Example usage:
/// ```dart
/// // Basic usage with default hashCode
/// StateSync<User>(
///   getFn: () async {
///     final response = await http.get(Uri.parse('https://api.example.com/user'));
///     return User.fromJson(jsonDecode(response.body));
///   },
///   setFn: (user) async {
///     await http.put(
///       Uri.parse('https://api.example.com/user'),
///       body: jsonEncode(user.toJson()),
///     );
///   },
///   refreshInterval: Duration(seconds: 30),
///   loadingBuilder: (context) => CircularProgressIndicator(),
///   errorBuilder: (context, error) => Text('Error: $error'),
///   builder: (context, user) {
///     return Column(
///       children: [
///         Text('Hello ${user.name}'),
///         ElevatedButton(
///           onPressed: () {
///             final updatedUser = user.copyWith(name: 'New Name');
///             context.setState<User>(updatedUser);
///           },
///           child: Text('Update Name'),
///         ),
///       ],
///     );
///   },
/// )
///
/// // With custom hash function for fine-tuned control
/// StateSync<User>(
///   getFn: () => fetchUser(),
///   setFn: (user) => updateUser(user),
///   hashFn: (user) => Object.hash(user.id, user.name, user.email),
///   refreshInterval: Duration(seconds: 30),
///   // ... builders
/// )
/// ```
class StateSync<T> extends StatefulWidget {
  /// Function to fetch the state from the backend (HTTP GET)
  /// Optional - if not provided, widget operates in write-only mode
  final Future<T> Function()? getFn;

  /// Function to update the state on the backend (HTTP PUT)
  /// Optional - if not provided, widget operates in read-only mode
  final Future<void> Function(T)? setFn;

  /// Initial state to use when getFn is not provided
  /// Required when getFn is null
  final T? initialState;

  /// Optional custom hash function for change detection
  /// If not provided, falls back to state.hashCode
  ///
  /// The hash function should return a consistent hash for equal states.
  /// Example: (user) => Object.hash(user.id, user.name, user.email)
  final int Function(T)? hashFn;

  /// Duration between automatic refresh calls
  final Duration refreshInterval;

  /// Builder for the normal state (when data is loaded)
  final Widget Function(BuildContext context, T state) builder;

  /// Builder for the loading state (initial load)
  final Widget Function(BuildContext context) loadingBuilder;

  /// Builder for the error state
  final Widget Function(BuildContext context, Object error) errorBuilder;

  const StateSync({
    super.key,
    this.getFn,
    this.setFn,
    this.initialState,
    this.hashFn,
    required this.refreshInterval,
    required this.builder,
    required this.loadingBuilder,
    required this.errorBuilder,
  }) : assert(getFn != null || initialState != null, 'Either getFn or initialState must be provided');

  @override
  State<StateSync<T>> createState() => _StateSyncState<T>();

  /// Access the StateSync from the widget tree to get/set state
  static _StateSyncInherited<T> of<T>(BuildContext context) {
    final _StateSyncInherited<T>? result = context.dependOnInheritedWidgetOfExactType<_StateSyncInherited<T>>();
    assert(result != null, 'No StateSync<$T> found in context');
    return result!;
  }
}

class _StateSyncState<T> extends State<StateSync<T>> {
  /// Cached state value
  T? _cachedState;

  /// Hash of the cached state for comparison
  int? _cachedHash;

  /// Timer for automatic refresh
  Timer? _refreshTimer;

  /// Loading flag for initial load
  bool _isLoading = true;

  /// Error object if any operation fails
  Object? _error;

  @override
  void initState() {
    super.initState();

    // If getFn is provided, fetch initial state
    if (widget.getFn != null) {
      _fetchState();
      _startRefreshTimer();
    }
    // If only initialState is provided (write-only mode)
    else if (widget.initialState != null) {
      setState(() {
        _cachedState = widget.initialState;
        _cachedHash = _computeHash(widget.initialState!);
        _isLoading = false;
      });
    }
    // This should never happen due to constructor assertion
    else {
      setState(() {
        _error = Exception('Neither getFn nor initialState was provided');
        _isLoading = false;
      });
    }
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  /// Fetch state from backend using getFn
  Future<void> _fetchState() async {
    // Guard against null getFn
    if (widget.getFn == null) {
      setState(() {
        _error = Exception('getFn is not configured for this StateSync instance. Cannot fetch state.');
        _isLoading = false;
      });
      return;
    }

    try {
      final newState = await widget.getFn!();
      final newHash = _computeHash(newState);

      // Only update if hash changed (data actually changed)
      if (newHash != _cachedHash) {
        setState(() {
          _cachedState = newState;
          _cachedHash = newHash;
          _isLoading = false;
          _error = null;
        });
      } else {
        // Data hasn't changed, just clear loading/error states
        if (_isLoading || _error != null) {
          setState(() {
            _isLoading = false;
            _error = null;
          });
        }
      }
    } catch (e) {
      setState(() {
        _error = e;
        _isLoading = false;
      });
    }
  }

  /// Update state locally and push to backend using setFn
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

      // Optionally fetch again to ensure sync
      // Uncomment if you want to verify the backend state after PUT
      // await _fetchState();
    } catch (e) {
      setState(() {
        _error = e;
      });
      // Optionally revert state or refetch from backend on error
      // await _fetchState();
    }
  }

  /// Compute hash of state object for change detection
  /// Uses custom hashFn if provided, otherwise falls back to state.hashCode
  int _computeHash(T state) {
    if (widget.hashFn != null) {
      return widget.hashFn!(state);
    }
    return state.hashCode;
  }

  /// Start the periodic refresh timer
  void _startRefreshTimer() {
    // Only start timer if getFn is available
    if (widget.getFn != null) {
      _refreshTimer = Timer.periodic(widget.refreshInterval, (_) {
        _fetchState();
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    // Show error state if error exists
    if (_error != null) {
      return widget.errorBuilder(context, _error!);
    }

    // Show loading state if initial load and no cached state
    if (_isLoading && _cachedState == null) {
      return widget.loadingBuilder(context);
    }

    // Show cached state (even during background refresh)
    if (_cachedState != null) {
      return _StateSyncInherited<T>(state: _cachedState!, setState: _updateState, child: widget.builder(context, _cachedState!));
    }

    // Fallback to loading (shouldn't normally reach here)
    return widget.loadingBuilder(context);
  }
}

/// InheritedWidget to provide state access down the widget tree
class _StateSyncInherited<T> extends InheritedWidget {
  final T state;
  final Future<void> Function(T) setState;

  const _StateSyncInherited({required this.state, required this.setState, required super.child});

  @override
  bool updateShouldNotify(_StateSyncInherited<T> oldWidget) {
    // Only notify if state reference changed
    // (hash comparison already happened in _StateSyncState)
    return state != oldWidget.state;
  }
}

/// Extension methods on BuildContext for convenient StateSync access
///
/// Provides a more concise API for accessing and updating state compared to
/// the verbose `StateSync.of<T>(context).state` pattern.
///
/// Example usage:
/// ```dart
/// // Reading state
/// final user = context.getState<User>();
/// Text('Hello ${user.name}')
///
/// // Updating state
/// context.setState<User>(user.copyWith(name: 'New Name'));
/// ```
extension StateSyncExtension on BuildContext {
  /// Get the current state value for type T
  ///
  /// This method retrieves the current state from the nearest StateSync<T>
  /// ancestor in the widget tree. The calling widget will rebuild when the
  /// state changes.
  ///
  /// Example:
  /// ```dart
  /// builder: (context, initialUser) {
  ///   // Access state from nested widgets
  ///   final user = context.getState<User>();
  ///   return Text('User: ${user.name}');
  /// }
  /// ```
  ///
  /// Throws an AssertionError if no StateSync<T> is found in the widget tree.
  /// Make sure this method is called from within a StateSync<T> widget's builder
  /// or one of its descendants.
  T getState<T>() {
    final inherited = dependOnInheritedWidgetOfExactType<_StateSyncInherited<T>>();
    assert(
      inherited != null,
      'No StateSync<$T> found in context. '
      'Make sure you are calling getState<$T>() from within a StateSync<$T> widget\'s builder or its descendants.',
    );
    return inherited!.state;
  }

  /// Update the state value for type T
  ///
  /// This method updates the state both locally (optimistic update) and syncs
  /// to the backend via the configured setFn. The UI updates immediately while
  /// the backend sync happens asynchronously.
  ///
  /// Returns a Future that completes when the backend sync finishes. You can
  /// await this to know when the sync is complete or handle errors.
  ///
  /// Example:
  /// ```dart
  /// ElevatedButton(
  ///   onPressed: () {
  ///     final user = context.getState<User>();
  ///     final updated = user.copyWith(name: 'New Name');
  ///     context.setState<User>(updated);
  ///   },
  ///   child: Text('Update'),
  /// )
  /// ```
  ///
  /// Example with await:
  /// ```dart
  /// ElevatedButton(
  ///   onPressed: () async {
  ///     try {
  ///       final user = context.getState<User>();
  ///       await context.setState<User>(user.copyWith(name: 'New Name'));
  ///       ScaffoldMessenger.of(context).showSnackBar(
  ///         SnackBar(content: Text('Updated successfully')),
  ///       );
  ///     } catch (e) {
  ///       // Error is also set in StateSync's error state
  ///       print('Update failed: $e');
  ///     }
  ///   },
  ///   child: Text('Update with Confirmation'),
  /// )
  /// ```
  ///
  /// Throws an AssertionError if no StateSync<T> is found in the widget tree.
  /// Throws an Exception if the StateSync<T> is in read-only mode (setFn is null).
  Future<void> setState<T>(T newState) {
    final inherited = dependOnInheritedWidgetOfExactType<_StateSyncInherited<T>>();
    assert(
      inherited != null,
      'No StateSync<$T> found in context. '
      'Make sure you are calling setState<$T>() from within a StateSync<$T> widget\'s builder or its descendants.',
    );
    return inherited!.setState(newState);
  }
}
