import 'dart:typed_data';
import 'package:http/http.dart' as http;
import 'provider.dart';

/// Storage provider that routes through the backend HTTP proxy.
///
/// This is the default provider for Flutter clients since they cannot
/// use private keys directly for cloud storage authentication.
///
/// Example usage:
/// ```dart
/// final provider = HttpStorageProvider(
///   basePath: 'https://api.example.com/asset/v1/storage',
///   getToken: () async => authService.currentToken,
/// );
///
/// await provider.save('images/photo.jpg', imageBytes);
/// final data = await provider.load('images/photo.jpg');
/// ```
class HttpStorageProvider implements StorageProvider {
  /// Full base path including protocol, domain, and API path
  /// (e.g., 'https://api.example.com/asset/v1/storage').
  final String basePath;

  /// Callback to retrieve the current JWT token.
  final Future<String> Function() getToken;

  /// Creates an HTTP storage provider.
  ///
  /// [basePath] - Full URL path to storage endpoint (e.g., 'https://api.example.com/asset/v1/storage')
  /// [getToken] - Callback to retrieve the current JWT token
  HttpStorageProvider({
    required this.basePath,
    required this.getToken,
  });

  @override
  String get name => 'http';

  Future<Map<String, String>> _headers() async {
    final token = await getToken();
    return {
      'Authorization': 'Bearer $token',
    };
  }

  @override
  Future<void> save(String path, Uint8List data) async {
    final headers = await _headers();
    headers['Content-Type'] = 'application/octet-stream';

    final response = await http.put(
      Uri.parse('$basePath/$path'),
      headers: headers,
      body: data,
    );

    if (response.statusCode != 200) {
      throw StorageException(
        'Failed to save file: ${response.statusCode}',
        response.statusCode,
      );
    }
  }

  @override
  Future<Uint8List> load(String path) async {
    final headers = await _headers();

    final response = await http.get(
      Uri.parse('$basePath/$path'),
      headers: headers,
    );

    if (response.statusCode == 404) {
      throw StorageNotFoundException(path);
    }

    if (response.statusCode != 200) {
      throw StorageException(
        'Failed to load file: ${response.statusCode}',
        response.statusCode,
      );
    }

    return response.bodyBytes;
  }

  @override
  Future<void> delete(String path) async {
    final headers = await _headers();

    final response = await http.delete(
      Uri.parse('$basePath/$path'),
      headers: headers,
    );

    // 200 = deleted, 404 = already doesn't exist (idempotent)
    if (response.statusCode != 200 && response.statusCode != 404) {
      throw StorageException(
        'Failed to delete file: ${response.statusCode}',
        response.statusCode,
      );
    }
  }

  @override
  Future<bool> exists(String path) async {
    final headers = await _headers();

    final response = await http.head(
      Uri.parse('$basePath/$path'),
      headers: headers,
    );

    return response.statusCode == 200;
  }
}
