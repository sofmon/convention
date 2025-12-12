# Storage Package - AI Agent Guide

This document provides context for AI agents working on the storage package.

> IMPORTANT: AI agents must treat AGENTS.md and README.md as authoritative living documents. Any change to the implementation that affects behaviors must be mirrored in both files. The code and documentation must never drift apart. When the implementation changes, these documents must be updated immediately so they always reflect the current system.

## Package Overview

The storage package is a **dual-language implementation** (Go + Flutter/Dart) for cloud storage operations. It follows the project's established patterns and conventions.

## File Responsibilities

### Go Files

| File | Purpose | Key Types |
|------|---------|-----------|
| `provider.go` | Provider interface and registry | `Provider`, `ProviderFactory`, `RegisterProvider()`, `NewProvider()` |
| `storage.go` | Main facade, config loading | `Storage`, `New()`, `NewWithCredentials()`, `NewWithProvider()`, `WithRootPath()` |
| `gcs.go` | Google Cloud Storage implementation | `gcsProvider` (implements `Provider`) |
| `handler.go` | HTTP handler for Flutter clients | `NewHandler(s, prefix)` returns `convAPI.Raw` |

### Dart Files

| File | Purpose | Key Types |
|------|---------|-----------|
| `provider.dart` | Abstract provider interface | `StorageProvider`, `StorageException`, `StorageNotFoundException` |
| `http_provider.dart` | HTTP backend implementation | `HttpStorageProvider` (implements `StorageProvider`) |
| `storage.dart` | Main facade for Flutter | `Storage` |
| `drop_zone.dart` | Drag-and-drop widget | `StorageDropZone`, `StorageDropZoneState` |

## Codebase Patterns to Follow

### Go Patterns

1. **Context pattern**: All significant functions take `ctx convCtx.Context` as first parameter:
   ```go
   func (s *Storage) Save(ctx convCtx.Context, path string, data []byte) (err error) {
       ctx = ctx.WithScope("storage.Save", "path", path, "size", len(data))
       defer ctx.Exit(&err)
       // ...
   }
   ```

2. **Named returns with defer**: Use named return values and `defer ctx.Exit(&err)` for error wrapping.

3. **Config loading**: Use `convCfg.String()`, `convCfg.Bytes()`, or `convCfg.Object[T]()` for configuration from `/etc/agent/`.

4. **HTTP handlers**: Use `convAPI.Raw` or `convAPI.RawP1[T]` with struct tags like `` `api:"PUT /path/{param}"` ``.

5. **Error responses**: Use `convAPI.ServeError(ctx, w, status, code, message, err)`.

6. **Provider registration**: Use `init()` function with `RegisterProvider("name", factoryFunc)`.

### Dart Patterns

1. **Callback-based auth**: Pass `getToken` callback, don't store tokens directly:
   ```dart
   Storage({
     required String baseUrl,
     required Future<String> Function() getToken,
   })
   ```

2. **Custom exceptions**: Define specific exception types extending a base `StorageException`.

3. **StatefulWidget pattern**: Use for widgets with internal state, expose methods via `GlobalKey<WidgetState>`.

## Root Path Configuration

Storage instances can have a root path prefix that is automatically prepended to all operations:

```go
// Create storage with root path
s, _ := storage.New()
tenant := s.WithRootPath("tenant-123/data")

// All operations now use the root path prefix
tenant.Save(ctx, "images/photo.jpg", data)   // stores at "tenant-123/data/images/photo.jpg"
tenant.Load(ctx, "images/photo.jpg")         // loads from "tenant-123/data/images/photo.jpg"
tenant.Delete(ctx, "images/photo.jpg")       // deletes "tenant-123/data/images/photo.jpg"
tenant.Exists(ctx, "images/photo.jpg")       // checks "tenant-123/data/images/photo.jpg"

// Get current root path
rootPath := tenant.RootPath()  // returns "tenant-123/data"

// Root paths can be chained
envStorage := s.WithRootPath("production")
tenantStorage := envStorage.WithRootPath("tenant-001")  // -> "production/tenant-001"
```

### Use Cases
- Multi-tenant storage isolation
- Environment separation (dev/staging/prod)
- Organizational prefixing

## Adding a New Storage Provider

### Go (e.g., S3)

1. Create `s3.go`:
   ```go
   package storage

   func init() {
       RegisterProvider("s3", newS3Provider)
   }

   type s3Provider struct {
       // S3-specific fields
   }

   // credentials parameter contains provider-specific auth data (e.g., JSON with access keys)
   func newS3Provider(bucket string, credentials []byte) (Provider, error) {
       // Parse credentials and initialize S3 client
   }

   func (p *s3Provider) Name() string { return "s3" }
   func (p *s3Provider) Save(ctx convCtx.Context, path string, data []byte) error { /* ... */ }
   func (p *s3Provider) Load(ctx convCtx.Context, path string) ([]byte, error) { /* ... */ }
   func (p *s3Provider) Delete(ctx convCtx.Context, path string) error { /* ... */ }
   func (p *s3Provider) Exists(ctx convCtx.Context, path string) (bool, error) { /* ... */ }
   ```

2. Add dependency: `go get github.com/aws/aws-sdk-go-v2/...`

3. Run: `go mod tidy && go mod vendor`

### Dart (e.g., Local Storage)

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
       // ... other methods
   }
   ```

2. Use with Storage:
   ```dart
   final storage = Storage.withProvider(LocalStorageProvider('/path'));
   ```

## HTTP Endpoint Structure

The handler is a single `convAPI.Raw` that dispatches based on HTTP method.
The path prefix is defined by the service embedding the handler.

### Handler Usage

```go
type API struct {
    Storage convAPI.Raw `api:"* /asset/v1/storage/{any...}"`
}

// The prefix parameter specifies the URL prefix to strip from incoming requests
// PUT /asset/v1/storage/images/photo.jpg -> stores to "images/photo.jpg"
api := &API{Storage: storage.NewHandler(s, "/asset/v1/storage")}
```

The `*` method in the api tag matches any HTTP method (wildcard).
The handler internally routes by method:
- PUT → Save
- GET → Load
- DELETE → Delete
- HEAD → Exists

The `prefix` parameter:
- Specifies the URL path prefix to strip from incoming requests
- Pass empty string `""` if no prefix stripping is needed
- Handles leading/trailing slashes automatically

### Combining Root Path with Handler Prefix

For multi-tenant scenarios where you want both features:

```go
func NewMyServiceAPI(ctx convCtx.Context, tenantID string) (*MyServiceAPI, error) {
    s, err := storage.New()
    if err != nil {
        return nil, err
    }

    // Create tenant-specific storage
    tenantStorage := s.WithRootPath("tenants/" + tenantID)

    return &MyServiceAPI{
        // Handler strips URL prefix, Storage adds root path
        // PUT /asset/v1/storage/file.txt -> stores to "tenants/{tenantID}/file.txt"
        Storage: storage.NewHandler(tenantStorage, "/asset/v1/storage"),
    }, nil
}
```

## Testing Considerations

- **Go**: Test providers in isolation using mock storage clients
- **Dart**: Use `Storage.withProvider()` with mock providers for unit tests
- **Integration**: Test HTTP handler with actual Storage instance

## Common Modifications

### Add new Storage method (e.g., `List`)

1. Add to `Provider` interface in `provider.go`
2. Implement in `gcsProvider` in `gcs.go`
3. Add wrapper in `Storage` in `storage.go`
4. Add HTTP endpoint in `handler.go` (if needed for Flutter)
5. Add to `StorageProvider` in `provider.dart`
6. Implement in `HttpStorageProvider` in `http_provider.dart`
7. Add wrapper in `Storage` in `storage.dart`

### Change HTTP path prefix

The path prefix is now configurable:
- **Go**: Define in your service API struct tag (e.g., `api:"* /myservice/v1/storage/{any...}"`)
- **Dart**: Pass full URL as `basePath` to `Storage()` constructor (e.g., `basePath: 'https://api.example.com/myservice/v1/storage'`)

The handler extracts the file path by looking for the `/storage/` segment in the URL.

### Add upload progress tracking (Dart)

Modify `HttpStorageProvider.save()` to use `http.StreamedRequest` and emit progress events.

## Configuration

Config files in `/etc/agent/`:

| File | Description |
|------|-------------|
| `storage_bucket` | Bucket/container name (string) |
| `storage_provider` | Provider name, defaults to "gcs" (string) |
| `storage_credentials` | Provider-specific credentials (bytes/JSON) |

### Credentials Format by Provider

**GCS** (`storage_credentials`): Service account JSON key downloaded from GCP Console
```json
{
  "type": "service_account",
  "project_id": "...",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...",
  ...
}
```

**S3** (future): Could use JSON with access keys
```json
{
  "access_key_id": "AKIA...",
  "secret_access_key": "...",
  "region": "us-east-1"
}
```

## Dependencies

### Go
- `cloud.google.com/go/storage` - GCS client
- `google.golang.org/api/option` - Client options (credentials)
- `ingreed/lib/util/ctx` - Context with scope tracking
- `ingreed/lib/util/cfg` - Configuration loading
- `ingreed/lib/util/api` - HTTP handler patterns

### Dart
- `http` package - HTTP client
- `flutter/material.dart` - Widget framework

## Build Commands

```bash
# Go
go build ./lib/util/storage/...
go test ./lib/util/storage/...

# Dart
flutter analyze lib/util/storage/
flutter test
```
