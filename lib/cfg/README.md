# Config Package

A simple, file-based configuration management package for Go applications.

## Overview

The `cfg` package provides a straightforward way to read configuration values from files stored in a designated folder. It supports reading raw bytes, strings, and JSON-serialized objects.

## Default Configuration Location

By default, configuration files are read from `/etc/agent/`.

## Usage

### Basic String Configuration

```go
import "github.com/sofmon/convention/lib/cfg"

// Define a config key
const APIEndpoint cfg.ConfigKey = "api_endpoint.txt"

// Read as string (with error handling)
endpoint, err := cfg.String(APIEndpoint)
if err != nil {
    log.Fatal(err)
}

// Or use the panic variant
endpoint := cfg.StringOrPanic(APIEndpoint)
```

### Binary Data

```go
const CertFile cfg.ConfigKey = "cert.pem"

// Read raw bytes
certData, err := cfg.Bytes(CertFile)
if err != nil {
    log.Fatal(err)
}

// Or use the panic variant
certData := cfg.BytesOrPanic(CertFile)
```

### JSON Objects

```go
type DatabaseConfig struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Username string `json:"username"`
}

const DBConfig cfg.ConfigKey = "database.json"

// Read and unmarshal JSON
config, err := cfg.Object[DatabaseConfig](DBConfig)
if err != nil {
    log.Fatal(err)
}

// Or use the panic variant
config := cfg.ObjectOrPanic[DatabaseConfig](DBConfig)
```

### Custom Configuration Location

```go
// Change the configuration folder
err := cfg.SetConfigLocation("/var/app/config")
if err != nil {
    log.Fatal(err)
}

// Now all reads will use the new location
value := cfg.StringOrPanic("myconfig.txt")
```

### Get File Path

```go
// Get the full path to a config file
path := cfg.FilePath("myconfig.txt")
// Returns: "/etc/agent/myconfig.txt" (or custom location)
```

## Error Handling

The package provides two variants for each read operation:

- **Standard variant** (`Bytes`, `String`, `Object`): Returns values with errors, allowing you to handle failures gracefully.
- **Panic variant** (`BytesOrPanic`, `StringOrPanic`, `ObjectOrPanic`): Panics on any error, useful for initialization code where missing config should halt execution.

## Configuration File Format

- **String/Binary files**: Store content directly in the file
- **JSON files**: Must contain valid JSON that can be unmarshaled into the target type

## Example Directory Structure

```
/etc/agent/
├── api_endpoint.txt
├── database.json
├── cert.pem
└── features.json
```

## Best Practices

1. **Use typed constants** for config keys to avoid typos:
   ```go
   const (
       APIKey      cfg.ConfigKey = "api_key.txt"
       DatabaseCfg cfg.ConfigKey = "database.json"
   )
   ```

2. **Choose the right variant**:
   - Use panic variants in `main()` or `init()` for required configuration
   - Use error-returning variants in runtime code for optional configuration

3. **Set custom location early**: Call `SetConfigLocation` at the start of your application if you need a custom path

4. **Keep JSON simple**: Use flat structures when possible for easier maintenance

## Thread Safety

The package is not explicitly thread-safe. Call `SetConfigLocation` during initialization before any concurrent operations.
