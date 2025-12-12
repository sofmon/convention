package storage

import (
	"strings"

	convCfg "github.com/sofmon/convention/lib/cfg"
	convCtx "github.com/sofmon/convention/lib/ctx"
)

const configKeyBucket convCfg.ConfigKey = "storage_bucket"
const configKeyProvider convCfg.ConfigKey = "storage_provider"
const configKeyCredentials convCfg.ConfigKey = "storage_credentials"

// joinPath combines root and path, handling edge cases with slashes.
// Returns path unchanged if root is empty.
func joinPath(root, path string) string {
	root = strings.Trim(root, "/")
	path = strings.TrimLeft(path, "/")
	if root == "" {
		return path
	}
	return root + "/" + path
}

// Storage provides a simple interface for storing and retrieving files.
// It wraps a Provider and adds context-aware logging and error handling.
type Storage struct {
	provider Provider
	rootPath string // prepended to all paths
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

// WithRootPath returns a new Storage instance with the specified root path.
// The root path is prepended to all storage paths.
// Example: storage.WithRootPath("tenant-123/data") causes Save(ctx, "file.txt", data)
// to store at "tenant-123/data/file.txt".
func (s *Storage) WithRootPath(rootPath string) *Storage {
	return &Storage{
		provider: s.provider,
		rootPath: joinPath(s.rootPath, rootPath),
	}
}

// RootPath returns the current root path prefix.
func (s *Storage) RootPath() string {
	return s.rootPath
}

// Save stores data at the specified path.
func (s *Storage) Save(ctx convCtx.Context, path string, data []byte) (err error) {
	fullPath := joinPath(s.rootPath, path)
	ctx = ctx.WithScope("storage.Save", "path", fullPath, "size", len(data))
	defer ctx.Exit(&err)

	err = s.provider.Save(ctx, fullPath, data)
	return
}

// Load retrieves data from the specified path.
func (s *Storage) Load(ctx convCtx.Context, path string) (data []byte, err error) {
	fullPath := joinPath(s.rootPath, path)
	ctx = ctx.WithScope("storage.Load", "path", fullPath)
	defer ctx.Exit(&err)

	data, err = s.provider.Load(ctx, fullPath)
	return
}

// Delete removes data at the specified path.
// Returns nil if path does not exist (idempotent).
func (s *Storage) Delete(ctx convCtx.Context, path string) (err error) {
	fullPath := joinPath(s.rootPath, path)
	ctx = ctx.WithScope("storage.Delete", "path", fullPath)
	defer ctx.Exit(&err)

	err = s.provider.Delete(ctx, fullPath)
	return
}

// Exists checks if data exists at the specified path.
func (s *Storage) Exists(ctx convCtx.Context, path string) (exists bool, err error) {
	fullPath := joinPath(s.rootPath, path)
	ctx = ctx.WithScope("storage.Exists", "path", fullPath)
	defer ctx.Exit(&err)

	exists, err = s.provider.Exists(ctx, fullPath)
	return
}

// Provider returns the underlying provider.
func (s *Storage) Provider() Provider {
	return s.provider
}
