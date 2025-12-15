# Storage Package - AI Agent Guide

This document provides context for AI agents working on the storage package.

> IMPORTANT: AI agents must treat AGENTS.md and README.md as authoritative living documents. Any change to the implementation that affects behaviors must be mirrored in both files. The code and documentation must never drift apart. When the implementation changes, these documents must be updated immediately so they always reflect the current system.

> For Flutter/Dart client implementation, see `lib/storage_client`.

## Package Overview

The storage package is a **Go implementation** for server-side cloud storage operations. It provides direct cloud access for Go services and HTTP handlers for client applications.

## File Responsibilities

| File | Purpose | Key Types |
|------|---------|-----------|
| `provider.go` | Provider interface and registry | `Provider`, `ProviderFactory`, `RegisterProvider()`, `NewProvider()` |
| `storage.go` | Main facade, config loading | `Storage`, `New()`, `NewWithCredentials()`, `NewWithProvider()`, `WithRootPath()` |
| `gcs.go` | Google Cloud Storage implementation | `gcsProvider` (implements `Provider`) |
| `handler.go` | HTTP handler for client proxy | `NewHandler(s, prefix)` returns `convAPI.Raw` |

## Codebase Patterns to Follow

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

Example: Adding S3 support

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

- Test providers in isolation using mock storage clients
- Integration test HTTP handler with actual Storage instance

## Common Modifications

### Add new Storage method (e.g., `List`)

1. Add to `Provider` interface in `provider.go`
2. Implement in `gcsProvider` in `gcs.go`
3. Add wrapper in `Storage` in `storage.go`
4. Add HTTP endpoint in `handler.go` (if needed for clients)

### Change HTTP path prefix

The path prefix is configurable:
- Define in your service API struct tag (e.g., `api:"* /myservice/v1/storage/{any...}"`)

The handler extracts the file path by looking for the `/storage/` segment in the URL.

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

- `cloud.google.com/go/storage` - GCS client
- `google.golang.org/api/option` - Client options (credentials)
- `github.com/sofmon/convention/lib/ctx` - Context with scope tracking
- `github.com/sofmon/convention/lib/cfg` - Configuration loading
- `github.com/sofmon/convention/lib/api` - HTTP handler patterns

## Build Commands

```bash
go build ./lib/storage/...
go test ./lib/storage/...
```
