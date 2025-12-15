# Storage Client Package - AI Agent Guide

This document provides context for AI agents working on the storage client package.

> IMPORTANT: AI agents must treat AGENTS.md and README.md as authoritative living documents. Any change to the implementation that affects behaviors must be mirrored in both files. The code and documentation must never drift apart. When the implementation changes, these documents must be updated immediately so they always reflect the current system.

> For Go backend implementation, see `lib/storage`.

## Package Overview

The storage client package is a **Flutter/Dart implementation** for client-side storage operations. It communicates with the Go backend (`lib/storage`) via HTTP for cloud storage access.

## File Responsibilities

| File | Purpose | Key Types |
|------|---------|-----------|
| `provider.dart` | Abstract provider interface | `StorageProvider`, `StorageException`, `StorageNotFoundException` |
| `http_provider.dart` | HTTP backend implementation | `HttpStorageProvider` (implements `StorageProvider`) |
| `storage.dart` | Main facade for Flutter | `Storage` |
| `drop_zone.dart` | Drag-and-drop widget | `StorageDropZone`, `StorageDropZoneState` |

## Codebase Patterns to Follow

1. **Callback-based auth**: Pass `getToken` callback, don't store tokens directly:
   ```dart
   Storage({
     required String baseUrl,
     required Future<String> Function() getToken,
   })
   ```

2. **Custom exceptions**: Define specific exception types extending a base `StorageException`.

3. **StatefulWidget pattern**: Use for widgets with internal state, expose methods via `GlobalKey<WidgetState>`.

4. **Async operations**: All storage operations are async and return `Future`.

## Adding a New Storage Provider

Example: Adding Local Storage support

1. Create `local_provider.dart`:
   ```dart
   import 'provider.dart';

   class LocalStorageProvider implements StorageProvider {
       final String basePath;
       LocalStorageProvider(this.basePath);

       @override
       String get name => 'local';

       @override
       Future<void> save(String path, Uint8List data) async { /* ... */ }

       @override
       Future<Uint8List> load(String path) async { /* ... */ }

       @override
       Future<void> delete(String path) async { /* ... */ }

       @override
       Future<bool> exists(String path) async { /* ... */ }
   }
   ```

2. Use with Storage:
   ```dart
   final storage = Storage.withProvider(LocalStorageProvider('/path'));
   ```

## Testing Considerations

- Use `Storage.withProvider()` with mock providers for unit tests
- Test widget state management with `GlobalKey<StorageDropZoneState>`
- Mock HTTP responses for `HttpStorageProvider` tests

## Common Modifications

### Add new Storage method (e.g., `list`)

1. Add to `StorageProvider` in `provider.dart`
2. Implement in `HttpStorageProvider` in `http_provider.dart`
3. Add wrapper in `Storage` in `storage.dart`

### Change base URL pattern

The base URL is configurable:
- Pass full URL as `basePath` to `Storage()` constructor (e.g., `basePath: 'https://api.example.com/myservice/v1/storage'`)

### Add upload progress tracking

Modify `HttpStorageProvider.save()` to use `http.StreamedRequest` and emit progress events.

## Dependencies

- `http` package - HTTP client
- `flutter/material.dart` - Widget framework
- `dart:typed_data` - For `Uint8List`

## Build Commands

```bash
flutter analyze lib/storage_client/
flutter test
```
