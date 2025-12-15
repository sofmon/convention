# Storage Client Package

A Flutter/Dart storage client that provides cloud storage operations through a Go backend. This package handles client-side storage logic including HTTP communication and UI widgets.

> For Go backend implementation, see `lib/storage`.

## Features

- **Simple API**: `save`, `load`, `delete`, `exists` operations
- **HTTP Provider**: Communicates with Go backend via HTTP
- **Extensible**: Provider interface for adding new backends
- **Drop Zone Widget**: Flutter widget for drag-and-drop file uploads
- **JWT Authentication**: Token-based authentication with the backend

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Flutter App    │────▶│   Go Backend    │────▶│  Cloud Storage  │
│  (this package) │ JWT │ (lib/storage)   │     │  (GCS/S3/etc)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

## Usage

### Storage Client

```dart
import 'package:convention/lib/storage_client/storage.dart';

final storage = Storage(
  basePath: 'https://api.example.com/asset/v1/storage',  // must match server API path
  getToken: () async => authService.currentToken,
);

// Save a file
await storage.save('images/photo.jpg', imageBytes);

// Load a file
final data = await storage.load('images/photo.jpg');

// Check if file exists
final exists = await storage.exists('documents/report.pdf');

// Delete a file
await storage.delete('temp/old-file.txt');
```

### Drop Zone Widget

```dart
import 'package:convention/lib/storage_client/storage.dart';
import 'package:convention/lib/storage_client/drop_zone.dart';

final dropZoneKey = GlobalKey<StorageDropZoneState>();

StorageDropZone(
  key: dropZoneKey,
  storage: storage,
  pathBuilder: (fileName) => 'uploads/${DateTime.now().millisecondsSinceEpoch}/$fileName',
  onUploadComplete: (path) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('Uploaded to: $path')),
    );
  },
  onError: (error) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('Upload failed: $error')),
    );
  },
  child: Container(
    width: 300,
    height: 200,
    decoration: BoxDecoration(
      border: Border.all(color: Colors.grey),
      borderRadius: BorderRadius.circular(8),
    ),
    child: const Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.cloud_upload, size: 48, color: Colors.grey),
          SizedBox(height: 8),
          Text('Drop files here or click to upload'),
        ],
      ),
    ),
  ),
)

// Upload programmatically (e.g., from file picker)
dropZoneKey.currentState?.uploadFile('photo.jpg', imageBytes);
```

## Error Handling

The package defines custom exceptions:

```dart
try {
  final data = await storage.load('path/to/file');
} on StorageNotFoundException {
  // File doesn't exist
} on StorageException catch (e) {
  // Other storage error
  print('Error: ${e.message}, status: ${e.statusCode}');
}
```

## File Structure

```
lib/storage_client/
├── provider.dart       # StorageProvider abstract class and exceptions
├── http_provider.dart  # HTTP provider implementation
├── storage.dart        # Storage class (main facade)
├── drop_zone.dart      # Drag-and-drop upload widget
├── README.md           # This file
└── AGENTS.md           # Documentation for AI agents
```
