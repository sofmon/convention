package storage

import (
	"fmt"

	convCtx "github.com/sofmon/convention/lib/ctx"
)

// Provider defines the interface for storage backends.
// Implementations include GCS, S3, Azure Blob, local filesystem, etc.
type Provider interface {
	// Save stores bytes at the specified path.
	Save(ctx convCtx.Context, path string, data []byte) (err error)

	// Load retrieves bytes from the specified path.
	Load(ctx convCtx.Context, path string) (data []byte, err error)

	// Delete removes the object at the specified path.
	// Returns nil if path does not exist (idempotent).
	Delete(ctx convCtx.Context, path string) (err error)

	// Exists checks if an object exists at the specified path.
	Exists(ctx convCtx.Context, path string) (exists bool, err error)

	// Name returns the provider identifier (e.g., "gcs", "s3", "azure", "local").
	Name() string
}

// ProviderFactory creates a provider from bucket name and credentials.
// The credentials parameter contains provider-specific authentication data
// (e.g., GCS service account JSON key, S3 access keys).
type ProviderFactory func(bucket string, credentials []byte) (Provider, error)

// registry holds registered provider factories
var registry = map[string]ProviderFactory{}

// RegisterProvider registers a provider factory by name.
// Called during init() of each provider implementation.
func RegisterProvider(name string, factory ProviderFactory) {
	registry[name] = factory
}

// NewProvider creates a provider by name using registered factory.
// The credentials parameter is provider-specific authentication data.
func NewProvider(name string, bucket string, credentials []byte) (Provider, error) {
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown storage provider: %s", name)
	}
	return factory(bucket, credentials)
}
