# Storage Package

A dual-language storage package for Go and Flutter/Dart that provides cloud storage operations with a simple, unified API.

## Features

- **Simple API**: `Save`, `Load`, `Delete`, `Exists` operations
- **Root Path Support**: Configure a root path prefix for all operations (multi-tenant, environment isolation)
- **Go**: Direct cloud storage access with service account authentication
- **Flutter/Dart**: Backend proxy for secure authentication (no private keys in client)
- **Extensible**: Provider interface for adding new storage backends (S3, Azure, etc.)
- **Drop Zone Widget**: Flutter widget for drag-and-drop file uploads
- **Configurable URL Prefix**: Strip service prefixes from incoming handler URLs

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Flutter App    │────▶│   Go Backend    │────▶│  Cloud Storage  │
│  (Storage)      │ JWT │   (Handler)     │     │  (GCS/S3/etc)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │
                              ▼
                        ┌─────────────────┐
                        │  Go Services    │
                        │  (Storage)      │
                        └─────────────────┘
```

## Go Usage

### Direct Storage Access

```go
import "github.com/sofmon/convention/lib/storage"

// Option 1: From config files (storage_bucket, storage_provider, storage_credentials)
s, err := storage.New()

// Option 2: Explicit credentials
credentials, _ := os.ReadFile("/path/to/service-account.json")
s, err := storage.NewWithCredentials(ctx, "gcs", "my-bucket", credentials)

// Save a file
err = s.Save(ctx, "images/photo.jpg", imageBytes)

// Load a file
data, err := s.Load(ctx, "images/photo.jpg")

// Check if file exists
exists, err := s.Exists(ctx, "documents/report.pdf")

// Delete a file
err = s.Delete(ctx, "temp/old-file.txt")
```

### Root Path (Multi-tenant)

Use root paths to isolate storage for different tenants, environments, or logical partitions:

```go
// Create base storage from config
s, err := storage.New()
if err != nil {
    return err
}

// Create tenant-specific storage instances
tenant1Storage := s.WithRootPath("tenants/tenant-001")
tenant2Storage := s.WithRootPath("tenants/tenant-002")

// Operations are automatically scoped to the tenant
tenant1Storage.Save(ctx, "data/file.txt", data)  // -> "tenants/tenant-001/data/file.txt"
tenant2Storage.Save(ctx, "data/file.txt", data)  // -> "tenants/tenant-002/data/file.txt"

// Root path chaining
envStorage := s.WithRootPath("production")
tenantStorage := envStorage.WithRootPath("tenant-001")  // -> "production/tenant-001"

// Check current root path
fmt.Println(tenantStorage.RootPath())  // "production/tenant-001"
```

### HTTP Handler for Flutter Clients

Include the storage handler in your service API to enable Flutter client access:

```go
import (
    convAPI "github.com/sofmon/convention/lib/api"
    "github.com/sofmon/convention/lib/storage"
)

type MyServiceAPI struct {
    // ... other endpoints
    Storage convAPI.Raw `api:"{any} /asset/v1/storage/{any...}"`
}

func NewMyServiceAPI(ctx convCtx.Context) (*MyServiceAPI, error) {
    s, err := storage.New()
    if err != nil {
        return nil, err
    }

    return &MyServiceAPI{
        // The prefix parameter strips the URL prefix from incoming requests
        // PUT /asset/v1/storage/images/photo.jpg -> stores to "images/photo.jpg"
        Storage: storage.NewHandler(s, "/asset/v1/storage"),
    }, nil
}
```

The handler dispatches to different operations based on HTTP method:

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/asset/v1/storage/{path...}` | Upload file |
| `GET` | `/asset/v1/storage/{path...}` | Download file |
| `DELETE` | `/asset/v1/storage/{path...}` | Delete file |
| `HEAD` | `/asset/v1/storage/{path...}` | Check if file exists |

The `prefix` parameter specifies the URL path prefix to strip from incoming requests.
Pass empty string `""` if no prefix stripping is needed.

All endpoints require JWT Bearer token in `Authorization` header.

## Flutter/Dart Usage

### Storage Client

```dart
import 'package:ingreed/lib/util/storage/storage.dart';

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
import 'package:ingreed/lib/util/storage/storage.dart';
import 'package:ingreed/lib/util/storage/drop_zone.dart';

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

## Configuration

Create these config files in `/etc/agent/`:

| File | Content | Description |
|------|---------|-------------|
| `storage_bucket` | `my-bucket-name` | GCS bucket name |
| `storage_provider` | `gcs` | Provider name (optional, defaults to "gcs") |
| `storage_credentials` | `{...}` | GCS service account JSON key (required) |

### GCS Credentials Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/) → IAM & Admin → Service Accounts
2. Create a service account or select an existing one
3. Click "Keys" → "Add Key" → "Create new key" → JSON
4. Download the JSON key file
5. Copy the **entire content** of the JSON file to `/etc/agent/storage_credentials`

Example `storage_credentials` content:
```json
{
  "type": "service_account",
  "project_id": "your-project-id",
  "private_key_id": "...",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "client_email": "storage@your-project-id.iam.gserviceaccount.com",
  "client_id": "...",
  ...
}
```

## Error Handling

### Go

All methods return errors following the standard Go pattern:

```go
data, err := s.Load(ctx, "path/to/file")
if err != nil {
    // Handle error - file not found, permission denied, etc.
}
```

### Flutter/Dart

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
lib/util/storage/
├── provider.go         # Provider interface and registry (Go)
├── storage.go          # Storage facade (Go)
├── gcs.go              # Google Cloud Storage provider (Go)
├── handler.go          # HTTP handler for Flutter proxy (Go)
├── provider.dart       # StorageProvider abstract class (Dart)
├── http_provider.dart  # HTTP provider implementation (Dart)
├── storage.dart        # Storage class (Dart)
├── drop_zone.dart      # Drag-and-drop upload widget (Dart)
├── README.md           # This file
└── AGENTS.md           # Documentation for AI agents
```
