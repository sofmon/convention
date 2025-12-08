# Auth Package Implementation Details

This document provides implementation details for AI agents working on the auth package.

> IMPORTANT: AI agents must treat AGENTS.md and README.md as authoritative living documents. Any change to the implementation that affects behaviors must be mirrored in both files. The code and documentation must never drift apart. When the implementation changes, these documents must be updated immediately so they always reflect the current system.

## Architecture Overview

The auth package implements a two-phase authorization system:

1. **Configuration Expansion** ([eval.go:23-55](eval.go)) - Converts high-level `Config` into efficient runtime structures
2. **Request Matching** ([eval.go:137-235](eval.go)) - Evaluates HTTP requests against expanded configuration

## File Structure

- **[auth.go](auth.go)** - Core type definitions (User, Role, Permission, Action, Config)
- **[claim.go](claim.go)** - JWT claims structure and constants
- **[http.go](http.go)** - JWT encoding/decoding and HTTP integration
- **[eval.go](eval.go)** - Authorization evaluation engine
- **[eval_test.go](eval_test.go)** - Comprehensive test cases
- **[main_test.go](main_test.go)** - Test setup (currently minimal)

## Core Types

### Basic Types (auth.go:8-43)

All basic types are type aliases over `string` or `[]string`:

```go
type User string
type Users []User
type Entity string
type Entities []Entity
type Tenant string
type Tenants []Tenant
type Role string
type Roles []Role
type Permission string
type Permissions []Permission
type Action string
type Actions []Action
```

### Configuration Types (auth.go:45-53)

```go
type RolePermissions map[Role]Permissions      // Role -> list of permissions
type PermissionActions map[Permission]Actions  // Permission -> list of actions
type Config struct {
    Roles       RolePermissions
    Permissions PermissionActions
    Public      Actions  // Unauthenticated access
}
```

### Claims (claim.go:3-16)

```go
type Claims struct {
    User      User
    Entities  Entities
    Tenants   Tenants
    Roles     Roles
    Additions map[string]any  // Custom claims
}
```

JWT claim keys are defined as package constants to avoid typos.

## Authorization Flow

### 1. Configuration Expansion (eval.go:23-55)

`expandConfig()` transforms the declarative `Config` into optimized runtime structures:

```go
func expandConfig(cfg Config) (allowed allowedRoles, publicActions allowedActions, err error)
```

**Process:**
- Iterates through each role's permissions
- Looks up each permission's actions
- Converts each action string into `allowedAction` struct
- Builds two indexes: `allowedRoles` (authenticated) and `publicActions` (unauthenticated)

**Key transformation:** `Action` string → `allowedAction` struct via `generateAllowedAction()` (eval.go:101-135)

### 2. Action Parsing (eval.go:101-135)

`generateAllowedAction()` parses action strings like `"GET /tenants/{tenant}/users/{user}/data"`:

```go
func generateAllowedAction(a Action) (res allowedAction, err error)
```

**Steps:**
1. Split into method and path using `Action.MethodPath()` (auth.go:30-41)
2. Split path into segments by `/`
3. Detect open-end wildcard (`{any...}` suffix)
4. Convert each segment into typed `allowedSegment`:
   - `{any}` → `allowedSegmentAny{}`
   - `{user}` → `allowedSegmentUser{}`
   - `{tenant}` → `allowedSegmentTenant{}`
   - `{entity}` → `allowedSegmentEntity{}`
   - Otherwise → `allowedSegmentFixed(segment)`

### 3. Runtime Structures (eval.go:13-21, 177-235)

```go
type allowedRoles map[Role]allowedActions
type allowedActions []allowedAction
type allowedAction struct {
    method  string
    path    allowedPath
    openEnd bool  // true if path ends with {any...}
}
type allowedPath []allowedSegment
```

**Segment matchers:** Each implements the `allowedSegment` interface:

```go
type allowedSegment interface {
    Match(segment string, claims Claims) bool
}
```

Implementations:
- `allowedSegmentFixed` (eval.go:195-199) - Exact string match
- `allowedSegmentAny` (eval.go:201-205) - Always matches
- `allowedSegmentUser` (eval.go:207-211) - Matches `claims.User`
- `allowedSegmentTenant` (eval.go:213-223) - Matches any tenant in `claims.Tenants`
- `allowedSegmentEntity` (eval.go:225-235) - Matches any entity in `claims.Entities`

### 4. Request Evaluation (eval.go:59-99)

`NewCheck()` returns a closure that evaluates HTTP requests:

```go
check := func(r *http.Request) error {
    segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

    // 1. Check public endpoints (no auth required)
    if publicEndpoints.match(r.Method, segments, Claims{}) {
        return nil
    }

    // 2. Decode JWT claims from Authorization header
    claims, err := DecodeHTTPRequestClaims(r)
    if err != nil {
        return err
    }

    // 3. Check user's roles against allowed actions
    if allowedRoles.match(r.Method, segments, claims) {
        return nil
    }

    return ErrForbidden
}
```

### 5. Matching Algorithm (eval.go:137-189)

Hierarchical matching with short-circuit evaluation:

```go
allowedRoles.match() → // Iterates user's roles
  allowedActions.match() → // Iterates role's actions
    allowedAction.match() → // Checks method and path
      allowedPath.match() → // Validates each segment
```

**Key behaviors:**
- Method must match exactly
- For `openEnd=true`: path can be longer than template
- For `openEnd=false`: path must match template length exactly
- Each segment validated left-to-right with early termination

## JWT Integration

### Token Generation (http.go:70-95)

```go
func GenerateToken(claims Claims) (string, error)
```

**Process:**
1. Load HMAC secret from config (cached after first load)
2. Create `jwt.MapClaims` with standard claims (user, entities, tenants, roles)
3. Add any custom claims from `Claims.Additions`
4. Standard claims overwrite additions if keys conflict
5. Sign with HS256 algorithm

### Token Decoding (http.go:97-153)

```go
func DecodeToken(tokenString string) (Claims, error)
```

**Process:**
1. Load HMAC secret from config
2. Parse and validate token (checks signature and format)
3. Extract standard claims with type assertions
4. Convert `[]any` arrays back to typed slices

**Note:** Type assertions use `.(string)` which will panic on type mismatch. Consider error handling improvements.

### HTTP Request Integration (http.go:41-68)

```go
func DecodeHTTPRequestClaims(r *http.Request) (Claims, error)
func EncodeHTTPRequestClaims(r *http.Request, claims Claims) error
```

- Expects `Authorization: Bearer <token>` header format
- Uses constant `HttpHeaderAuthorization = "Authorization"`
- Split on space, validate "Bearer" prefix

## Configuration Management

### HMAC Secret (http.go:26-39)

```go
func getHmacSecret() ([]byte, error)
```

- Lazy loads from config on first use
- Cached in package variable `hmacSecret`
- Loaded via `github.com/sofmon/convention/lib/cfg.Bytes("communication_secret")`
- Not thread-safe (assumes single-threaded initialization)

## Testing Strategy

### Test Data Structure (eval_test.go:67-195)

Comprehensive test cases in `testData` slice:

```go
var testData = []struct {
    name     string
    cfg      convAuth.Config
    user     convAuth.User
    tenants  convAuth.Tenants
    roles    convAuth.Roles
    entities convAuth.Entities
    pass     []*http.Request  // Should succeed
    block    []*http.Request  // Should fail
}
```

### Test Coverage (eval_test.go:197-238)

Single test function `TestCheck()` iterates all test cases:
1. Create checker from config
2. Build claims from test data
3. Verify all `pass` requests succeed
4. Verify all `block` requests fail

**Test scenarios:**
- User-specific access (`{user}` template)
- Tenant-scoped access (`{tenant}` template)
- Cross-tenant admin access (`{any}` template)
- Wildcard suffixes (`{any...}`)
- Public endpoints (empty claims)

## Implementation Patterns

### Type Safety

- All domain types (User, Role, etc.) are distinct types despite being strings
- Prevents accidental mixing (e.g., passing Role where User expected)
- Enables future behavior addition without breaking changes

### Interface-Based Matching

- `allowedSegment` interface enables polymorphic matching
- Each matcher type has single responsibility
- Easily extensible for new segment types

### Functional Options

- `NewCheck()` returns configured function, not struct
- Captures configuration in closure
- Immutable after creation

### Error Handling

- Distinct error types for different failure modes
- Enables appropriate HTTP status code selection
- Errors returned, not logged (caller decides logging)

## Potential Improvements

### Security
- Add JWT expiration support (exp claim)
- Add token refresh mechanism
- Consider blacklist/revocation support
- Add rate limiting per user/tenant

### Performance
- Thread-safe secret initialization (sync.Once)
- Cache expanded configurations
- Optimize string splitting (reuse buffers)

### Robustness
- Better type assertion error handling in DecodeToken
- Validate configuration on load (detect cycles, missing permissions)
- Add structured logging hooks

### Features
- Support for custom segment matchers
- Regex-based path matching
- Conditional permissions (time-based, IP-based)
- Permission inheritance/hierarchies

## Common Modification Scenarios

### Adding New Segment Type

1. Define new type implementing `allowedSegment` (eval.go)
2. Add case in `generateAllowedAction()` switch statement
3. Add validation logic in `Match()` method
4. Add test cases in eval_test.go

### Adding Custom Claims

1. Add fields to `Claims` struct (claim.go)
2. Add claim key constant (claim.go)
3. Update `GenerateToken()` to serialize (http.go)
4. Update `DecodeToken()` to deserialize (http.go)
5. Use in segment matchers if needed (eval.go)

### Changing JWT Algorithm

1. Update `GenerateToken()` signing method (http.go:89)
2. Update `DecodeToken()` validation (http.go:106)
3. Ensure secret format matches algorithm requirements

### Adding Middleware Features

1. Extend `Check` function signature or create new wrapper
2. Keep existing `Check` function for compatibility
3. Consider context.Context for request-scoped data
