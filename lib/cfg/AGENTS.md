# Config Package - Implementation Details

This document provides implementation details for AI agents working on the cfg package.

> IMPORTANT: AI agents must treat AGENTS.md and README.md as authoritative living documents. Any change to the implementation that affects behaviors must be mirrored in both files. The code and documentation must never drift apart. When the implementation changes, these documents must be updated immediately so they always reflect the current system.

## Package Structure

**Location**: `lib/util/cfg/config.go`
**Package**: `cfg`

## Architecture

This is a simple file-based configuration loader with no external dependencies beyond the Go standard library.

### Core Components

1. **ConfigKey type** ([config.go:10](config.go#L10))
   - Simple string alias for type safety
   - Used to identify configuration files

2. **Global state** ([config.go:12](config.go#L12))
   - `configLocation` variable stores the current config directory
   - Default: `/etc/agent/`
   - Mutable via `SetConfigLocation`

## Functions

### Configuration Management

#### `SetConfigLocation(folder string) error` ([config.go:14-30](config.go#L14-L30))

Changes the configuration directory.

**Behavior**:
- Validates folder exists using `os.Stat`
- Checks if path is a directory
- Appends trailing slash if missing
- Updates global `configLocation` variable

**Error cases**:
- Folder does not exist: `fmt.Errorf("folder '%s' does not exists", folder)`
- Cannot read folder: `fmt.Errorf("error reading folder '%s': %w", folder, err)`
- Path is not a directory: `fmt.Errorf("config location '%s' must be a folder", folder)`

**Thread safety**: ⚠️ Not thread-safe. Should be called during initialization.

#### `FilePath(key ConfigKey) string` ([config.go:32-34](config.go#L32-L34))

Returns the full file path for a config key.

**Implementation**: Simple string concatenation: `configLocation + string(key)`

### Read Functions

#### `Bytes(key ConfigKey) ([]byte, error)` ([config.go:36-43](config.go#L36-L43))

Reads raw file contents.

**Implementation**:
- Constructs path: `configLocation + string(key)`
- Uses `os.ReadFile`
- Wraps errors with context

**Error format**: `fmt.Errorf("error reading config file '%s': %w", file, err)`

#### `BytesOrPanic(key ConfigKey) []byte` ([config.go:45-51](config.go#L45-L51))

Panic variant of `Bytes`.

**Implementation**: Calls `Bytes`, panics on error

#### `String(key ConfigKey) (string, error)` ([config.go:53-60](config.go#L53-L60))

Reads file contents as string.

**Implementation**:
- Calls `Bytes`
- Converts to string
- Propagates errors

#### `StringOrPanic(key ConfigKey) string` ([config.go:62-68](config.go#L62-L68))

Panic variant of `String`.

**Implementation**: Calls `String`, panics on error

#### `Object[T any](key ConfigKey) (T, error)` ([config.go:70-80](config.go#L70-L80))

Generic function to read and unmarshal JSON.

**Implementation**:
1. Calls `Bytes` to read file
2. Uses `json.Unmarshal` to deserialize
3. Returns zero value of T on error (Go default behavior)

**Error handling**: Returns errors from both file read and JSON unmarshal

#### `ObjectOrPanic[T any](key ConfigKey) T` ([config.go:82-92](config.go#L82-L92))

Panic variant of `Object`.

**Implementation**:
- Calls `Bytes`, panics on error
- Calls `json.Unmarshal`, panics on error
- Note: Duplicates `Bytes` call instead of using `Object`

## Design Patterns

### Panic vs Error Returns

The package follows a dual-interface pattern:
- **Error-returning functions**: For runtime configuration where graceful degradation is possible
- **Panic functions**: For initialization-time configuration where missing values should halt execution

### Type Safety

- `ConfigKey` type prevents accidental string mixing
- Generic `Object[T]` function provides compile-time type checking for deserialization

### Simplicity Trade-offs

**What's included**:
- File-based storage
- JSON deserialization
- Basic error wrapping

**What's NOT included**:
- No caching (reads from disk every time)
- No watching/reloading
- No environment variable fallback
- No default values
- No validation
- No thread safety
- No logging

## Modification Guidelines

### Adding Features

1. **Caching**: Consider adding a cache layer in `Bytes` to avoid repeated disk reads
2. **Default values**: Add `OrDefault` variants that accept fallback values
3. **Environment variables**: Add `FromEnvOr` functions for 12-factor app support
4. **Validation**: Add schema validation for Object types
5. **Hot reload**: Add file watchers and reload callbacks

### Backward Compatibility

When modifying:
- Keep the global `configLocation` variable (existing code depends on it)
- Maintain panic behavior in `*OrPanic` functions
- Preserve error message format (logging systems may parse it)
- Keep `ConfigKey` as a string alias (allows string literals)

### Testing Considerations

Test cases should cover:
1. Missing files
2. Invalid JSON in `Object` functions
3. Non-directory paths in `SetConfigLocation`
4. Paths with/without trailing slashes
5. Binary vs text file contents
6. Empty files
7. Very large files (memory implications)

### Common Issues

1. **Race conditions**: If `SetConfigLocation` is called after goroutines start reading
2. **Path traversal**: No validation that keys don't contain `..` or absolute paths
3. **Memory**: Large files are fully loaded into memory
4. **File handles**: No explicit cleanup (relies on OS GC)

## Dependencies

Standard library only:
- `encoding/json` - JSON unmarshaling
- `fmt` - Error formatting
- `os` - File I/O and stat
- `strings` - String manipulation

## Usage Patterns in Codebase

When you encounter code using this package:

```go
// Initialization pattern (use panic variants)
func init() {
    apiKey := cfg.StringOrPanic("api_key.txt")
}

// Runtime pattern (use error variants)
func handleRequest() {
    config, err := cfg.Object[Config]("feature.json")
    if err != nil {
        // Handle gracefully
    }
}

// Custom location pattern
func main() {
    if customPath := os.Getenv("CONFIG_PATH"); customPath != "" {
        if err := cfg.SetConfigLocation(customPath); err != nil {
            log.Fatal(err)
        }
    }
}
```

## Security Considerations

1. **No path sanitization**: ConfigKey values are used directly in file paths
2. **No permission checks**: Relies on OS-level file permissions
3. **No encryption**: All files stored in plaintext
4. **No secrets management**: Not suitable for sensitive credentials without external encryption
