import 'dart:typed_data';
import 'provider.dart';
import 'http_provider.dart';

/// Simple storage interface for Flutter applications.
///
/// Provides [save], [load], [delete], and [exists] operations that route
/// through the configured provider. Default provider is [HttpStorageProvider]
/// for backend proxy.
///
/// Example usage:
/// ```dart
/// final storage = Storage(
///   basePath: 'https://api.example.com/asset/v1/storage',
///   getToken: () async => authService.currentToken,
/// );
///
/// // Save a file
/// await storage.save('images/photo.jpg', imageBytes);
///
/// // Load a file
/// final data = await storage.load('images/photo.jpg');
///
/// // Check if file exists
/// final exists = await storage.exists('images/photo.jpg');
///
/// // Delete a file
/// await storage.delete('images/photo.jpg');
/// ```
class Storage {
  final StorageProvider _provider;

  /// Creates a Storage instance with HTTP backend provider.
  ///
  /// [basePath] - Full URL to storage endpoint (e.g., 'https://api.example.com/asset/v1/storage')
  /// [getToken] - Callback to retrieve the current JWT token
  Storage({
    required String basePath,
    required Future<String> Function() getToken,
  }) : _provider = HttpStorageProvider(
          basePath: basePath,
          getToken: getToken,
        );

  /// Creates a Storage instance with a custom provider.
  ///
  /// Useful for testing or when using a non-standard provider.
  Storage.withProvider(this._provider);

  /// Saves data to the specified path.
  ///
  /// Overwrites any existing file at the path.
  Future<void> save(String path, Uint8List data) => _provider.save(path, data);

  /// Loads data from the specified path.
  ///
  /// Throws [StorageNotFoundException] if the file does not exist.
  Future<Uint8List> load(String path) => _provider.load(path);

  /// Deletes data at the specified path.
  ///
  /// Returns normally if the file does not exist (idempotent).
  Future<void> delete(String path) => _provider.delete(path);

  /// Checks if data exists at the specified path.
  Future<bool> exists(String path) => _provider.exists(path);

  /// Returns the underlying provider.
  StorageProvider get provider => _provider;
}
