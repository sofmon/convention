package storage

import (
	convCfg "github.com/sofmon/convention/lib/cfg"
	convCtx "github.com/sofmon/convention/lib/ctx"
)

const configKeyBucket convCfg.ConfigKey = "storage_bucket"
const configKeyProvider convCfg.ConfigKey = "storage_provider"
const configKeyCredentials convCfg.ConfigKey = "storage_credentials"

// Storage provides a simple interface for storing and retrieving files.
// It wraps a Provider and adds context-aware logging and error handling.
type Storage struct {
	provider Provider
}

// New creates a Storage instance using configuration from config files.
// Reads "storage_bucket", "storage_provider" (defaults to "gcs"), and
// "storage_credentials" (JSON key file content) from config.
func New() (s *Storage, err error) {
	bucket, err := convCfg.String(configKeyBucket)
	if err != nil {
		return
	}

	providerName, err := convCfg.String(configKeyProvider)
	if err != nil {
		providerName = "gcs" // default provider
	}

	credentials, err := convCfg.Bytes(configKeyCredentials)
	if err != nil {
		return
	}

	provider, err := NewProvider(providerName, bucket, credentials)
	if err != nil {
		return
	}

	s = &Storage{provider: provider}
	return
}

// NewWithCredentials creates a Storage instance with explicit provider, bucket, and credentials.
// This allows overriding the config-based initialization.
// The credentials parameter should contain the provider-specific authentication data
// (e.g., GCS service account JSON key content).
func NewWithCredentials(ctx convCtx.Context, providerName, bucket string, credentials []byte) (s *Storage, err error) {
	ctx = ctx.WithScope("NewWithCredentials", "provider", providerName, "bucket", bucket)
	defer ctx.Exit(&err)

	provider, err := NewProvider(providerName, bucket, credentials)
	if err != nil {
		return
	}

	s = &Storage{provider: provider}
	return
}

// NewWithProvider creates a Storage instance with a custom provider.
// Useful for testing or when using a non-standard provider.
func NewWithProvider(provider Provider) *Storage {
	return &Storage{provider: provider}
}

// Save stores data at the specified path.
func (s *Storage) Save(ctx convCtx.Context, path string, data []byte) (err error) {
	ctx = ctx.WithScope("storage.Save", "path", path, "size", len(data))
	defer ctx.Exit(&err)

	err = s.provider.Save(ctx, path, data)
	return
}

// Load retrieves data from the specified path.
func (s *Storage) Load(ctx convCtx.Context, path string) (data []byte, err error) {
	ctx = ctx.WithScope("storage.Load", "path", path)
	defer ctx.Exit(&err)

	data, err = s.provider.Load(ctx, path)
	return
}

// Delete removes data at the specified path.
// Returns nil if path does not exist (idempotent).
func (s *Storage) Delete(ctx convCtx.Context, path string) (err error) {
	ctx = ctx.WithScope("storage.Delete", "path", path)
	defer ctx.Exit(&err)

	err = s.provider.Delete(ctx, path)
	return
}

// Exists checks if data exists at the specified path.
func (s *Storage) Exists(ctx convCtx.Context, path string) (exists bool, err error) {
	ctx = ctx.WithScope("storage.Exists", "path", path)
	defer ctx.Exit(&err)

	exists, err = s.provider.Exists(ctx, path)
	return
}

// Provider returns the underlying provider.
func (s *Storage) Provider() Provider {
	return s.provider
}
