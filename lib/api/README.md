# API Package

A type-safe, declarative HTTP API framework for Go built on generics. Define your API as a struct, and the framework handles routing, serialization, OpenAPI generation, and client creation automatically.

## Import Convention

This package follows the `conv` prefix naming convention for imports:

```go
import (
    convAPI "github.com/sofmon/convention/lib/api"
    convAuth "github.com/sofmon/convention/lib/auth"
    convCtx "github.com/sofmon/convention/lib/ctx"
)
```

## Quick Start

### 1. Define Your API

```go
package def

import (
    convAPI "github.com/sofmon/convention/lib/api"
    convAuth "github.com/sofmon/convention/lib/auth"
)

type API struct {
    GetHealth convAPI.Out[string] `api:"GET /health/"`

    GetUser    convAPI.OutP1[User, UserID]           `api:"GET /users/{user_id}"`
    CreateUser convAPI.InOut[CreateUserReq, User]    `api:"POST /users"`
    DeleteUser convAPI.TriggerP1[UserID]             `api:"DELETE /users/{user_id}"`
    UpdateUser convAPI.InP2[UpdateUserReq, convAuth.Tenant, UserID] `api:"PUT /tenants/{tenant}/users/{user_id}"`

    GetOpenAPI convAPI.OpenAPI `api:"GET /openapi.yaml"`
}

var Client = convAPI.NewClient[API]("api.example.com", 443)

func OpenAPI() convAPI.OpenAPI {
    return convAPI.NewOpenAPI().
        WithDescription("User management API").
        WithServers("https://api.example.com")
}
```

### 2. Implement Handlers

```go
package svc

import (
    convAPI "github.com/sofmon/convention/lib/api"
    convCtx "github.com/sofmon/convention/lib/ctx"

    "myapp/def"
)

func ListenAndServe(ctx convCtx.Context) error {
    svr, err := convAPI.NewServer(ctx, "", 443, authPolicy, &def.API{
        GetHealth:  convAPI.NewOut(handleGetHealth),
        GetUser:    convAPI.NewOutP1(handleGetUser),
        CreateUser: convAPI.NewInOut(handleCreateUser),
        DeleteUser: convAPI.NewTriggerP1(handleDeleteUser),
        UpdateUser: convAPI.NewInP2(handleUpdateUser),
        GetOpenAPI: def.OpenAPI(),
    })
    if err != nil {
        return err
    }
    return svr.ListenAndServe()
}

func handleGetHealth(ctx convCtx.Context) (string, error) {
    return "OK", nil
}

func handleGetUser(ctx convCtx.Context, id UserID) (User, error) {
    // Implementation
}

func handleCreateUser(ctx convCtx.Context, req CreateUserReq) (User, error) {
    // Implementation
}

func handleDeleteUser(ctx convCtx.Context, id UserID) error {
    // Implementation
}

func handleUpdateUser(ctx convCtx.Context, tenant convAuth.Tenant, id UserID, req UpdateUserReq) error {
    // Implementation
}
```

### 3. Use the Client

```go
client := def.Client

user, err := client.GetUser.Call(ctx, "user-123")
if err != nil {
    if convAPI.ErrorHasCode(err, convAPI.ErrorCodeNotFound) {
        // Handle not found
    }
    return err
}
```

## Handler Types

| Type | Description | Handler Signature | Client Call |
|------|-------------|-------------------|-------------|
| `Trigger` | No input or output | `func(ctx) error` | `Call(ctx) error` |
| `In[T]` | Input only | `func(ctx, in T) error` | `Call(ctx, in T) error` |
| `Out[T]` | Output only | `func(ctx) (T, error)` | `Call(ctx) (T, error)` |
| `InOut[I,O]` | Input and output | `func(ctx, in I) (O, error)` | `Call(ctx, in I) (O, error)` |
| `Raw` | Direct HTTP access | `func(ctx, w, r)` | `Call(ctx, body) error` |

### Path Parameters

Each handler type has variants supporting 1-5 path parameters (`P1` through `P5`):

```go
// Single parameter
GetUser convAPI.OutP1[User, UserID] `api:"GET /users/{user_id}"`

// Two parameters
GetUserPost convAPI.OutP2[Post, UserID, PostID] `api:"GET /users/{user_id}/posts/{post_id}"`

// Three parameters
GetComment convAPI.OutP3[Comment, Tenant, UserID, CommentID] `api:"GET /tenants/{t}/users/{u}/comments/{c}"`
```

Parameter types must be based on `string` (using Go's type constraint `~string`):

```go
type UserID string
type PostID string
```

Handler signature with parameters:
```go
func handleGetUserPost(ctx convCtx.Context, userId UserID, postId PostID) (Post, error) {
    // Parameters are extracted from URL path in order
}
```

## API Tag Format

```
`api:"METHOD /path/to/endpoint?query=type|description"`
```

- **METHOD** (optional): HTTP method (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`). Defaults to `GET`.
- **Path**: Static segments and `{param}` placeholders.
- **Query params** (optional): `?name=type|description` for OpenAPI documentation.

Examples:
```go
`api:"GET /users"`                           // Simple GET
`api:"POST /users"`                          // POST with body
`api:"/users/{id}"`                          // GET is default
`api:"GET /search?q=string|Search query"`    // With query param docs
`api:"GET /items?limit=integer|Max results&offset=integer|Skip count"`
```

## OpenAPI Generation

The package auto-generates OpenAPI 3.0 YAML documentation:

```go
func OpenAPI() convAPI.OpenAPI {
    return convAPI.NewOpenAPI().
        WithDescription("API description").
        WithServers(
            "https://api.example.com",
            "https://api.dev.example.com",
        ).
        WithTypeSubstitutions(
            // Map types with custom marshaling to their JSON structure
            convAPI.NewTypeSubstitution[
                Money,           // Original type
                struct {         // How it appears in JSON
                    Amount   float64 `json:"amount"`
                    Currency string  `json:"currency"`
                },
            ](),
        ).
        WithEnums(
            // Document enum values for string-based types
            convAPI.NewEnum(StatusDraft, StatusActive, StatusArchived),
            convAPI.NewEnum(RoleAdmin, RoleUser, RoleGuest),
        )
}
```

## Pre/Post Checks

Add authorization or validation logic that runs before or after handlers:

```go
handler := convAPI.NewOutP1(handleGetUser).
    WithPreCheck(func(ctx convCtx.Context) error {
        // Runs before handler - e.g., permission check
        if !hasPermission(ctx) {
            return convAPI.NewError(ctx, 403, convAPI.ErrorCodeForbidden, "access denied", nil)
        }
        return nil
    }).
    WithPostCheck(func(ctx convCtx.Context) error {
        // Runs after handler - e.g., audit logging
        return nil
    })
```

## Error Handling

### Creating Errors

```go
return convAPI.NewError(ctx, http.StatusNotFound, convAPI.ErrorCodeNotFound, "user not found", nil)
```

### Error Codes

| Code | Description |
|------|-------------|
| `ErrorCodeInternalError` | Server-side error (500) |
| `ErrorCodeNotFound` | Resource not found (404) |
| `ErrorCodeBadRequest` | Invalid request (400) |
| `ErrorCodeForbidden` | Authentication failed (403) |
| `ErrorCodeUnauthorized` | Authorization failed (401) |

### Checking Errors (Client-side)

```go
user, err := client.GetUser.Call(ctx, id)
if err != nil {
    if convAPI.ErrorHasCode(err, convAPI.ErrorCodeNotFound) {
        // Handle 404
    }
    return err
}
```

## Server Creation

### Full Server with TLS

```go
svr, err := convAPI.NewServer(ctx, "0.0.0.0", 443, authPolicy, &API{...})
if err != nil {
    return err
}
return svr.ListenAndServe() // Uses TLS certificates from config
```

### HTTP Handler Only

For integration with existing servers or custom TLS setup:

```go
handler := convAPI.NewHandler(ctx, "api.example.com", 443, authCheck, &API{...})
http.Handle("/", handler)
```

## Helper Functions

```go
// Receive JSON body in Raw handlers
req, err := convAPI.ReceiveJSON[CreateUserReq](r)

// Serve JSON response in Raw handlers
convAPI.ServeJSON(w, response)

// Serve error response
convAPI.ServeError(ctx, w, http.StatusBadRequest, convAPI.ErrorCodeBadRequest, "invalid input", err)
```

## Type Mapping

Go types are automatically converted to OpenAPI schemas:

| Go Type | OpenAPI Type |
|---------|--------------|
| `string`, `~string` | `string` |
| `int`, `int64`, etc. | `integer` |
| `float32`, `float64` | `number` |
| `bool` | `boolean` |
| `time.Time` | `string` (format: date-time) |
| `[]T` | `array` |
| `map[K]V` | `object` (additionalProperties) |
| `struct` | `object` |
| `*T` | Same as T, but optional |

Field naming follows these rules:
- Uses `json` tag if present
- Otherwise converts to snake_case (`UserID` → `user_id`)
- `omitempty` makes fields optional in the schema
