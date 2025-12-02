import 'dart:typed_data';

/// Abstract base class for storage providers.
///
/// Implementations include:
/// - [HttpStorageProvider]: Routes through backend proxy (for Flutter clients)
///
/// Future implementations may include direct cloud storage access for
/// server-side Dart or local filesystem storage.
abstract class StorageProvider {
  /// Saves data to the specified path.
  Future<void> save(String path, Uint8List data);

  /// Loads data from the specified path.
  ///
  /// Throws [StorageNotFoundException] if the file does not exist.
  Future<Uint8List> load(String path);

  /// Deletes data at the specified path.
  ///
  /// Returns normally if the file does not exist (idempotent).
  Future<void> delete(String path);

  /// Checks if data exists at the specified path.
  Future<bool> exists(String path);

  /// Provider identifier (e.g., "http", "local", "gcs").
  String get name;
}

/// Exception thrown by storage operations.
class StorageException implements Exception {
  /// Error message describing what went wrong.
  final String message;

  /// HTTP status code if applicable.
  final int? statusCode;

  StorageException(this.message, [this.statusCode]);

  @override
  String toString() => 'StorageException: $message';
}

/// Exception thrown when a file is not found.
class StorageNotFoundException extends StorageException {
  StorageNotFoundException(String path) : super('File not found: $path', 404);
}
