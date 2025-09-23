# Go implementation of convention/v2

Reference [Go](https://go.dev) implementation of [convention/v2](../README.md) - a standard for containerized, multi-tenant, multi-entity architectures.

## Overview

This Go implementation provides a complete framework for building **agents** that comply with [convention/v2](../README.md) standards. It includes type-safe API definitions, authentication/authorization, multi-tenant database access, configuration management, and structured logging.

## Typical agent implementation
Below is a typical agent implementation following convention/v2/go practices

```txt
{{project_root}}            # Project root (example: "beluga")
│
├── go.mod                  # Go package file (package "beluga")
├── go.sum                  # Go dependency trace file
├── vendor/                 # Go vendor folder
│   └── ...                 # Saved dependencies
│
└── agents/
    └── {{agent_name}}/     # agent name
        │
        └── v1/             # agent version
            │
            ├── README.md   # agent description
            │
            ├── main.go     # agent entry point
            │
            ├── def         # object model definitions
            │   ├── api.go  # API definition
            │   └── ...     # file per main object group
            │
            ├── svc         # API endpoints implementation
            │   └── ...     # files per major endpoint handlers group
            │
            ├── bg          # implements the agent's background tasks
            │   └── ...     # files per major background tasks group
            │
            ├── tx          # implements the agent's business logic
            │   └── ...     # files per major business logic group
            │
            └── db          # database access layer
                └── db.go   # used to define ORM objects
                └── ...     # files with low level db access
```

### main.go
```Go
package main

import (
    // for clarity "conv..." prefix is preferred
    // for all convention related imports
	convCfg "github.com/sofmon/convention/v2/go/cfg"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
    convAuth "github.com/sofmon/convention/v2/go/auth"

	"beluga/agents/message/v1/svc"
)

func main() {
    // create main context that would trace the agent execution
	ctx := convCtx.New(
        // set the agent credentials
        convAuth.Claims{
            User:     "message-v1",
            Entities: convAuth.Entities{"system"},
            Tenants:  convAuth.Tenants{"main"},
            Roles:    "system",
	    },
    )

    // log a message letting us know that hte service has (re)started
	ctx.LogWarn("service restarted")

    // start listening for incoming calls (see "svc.go")
	err = svc.ListenAndServe(ctx)
	if err != nil {
		ctx.LogError(err)
	}
}
```

### api.go
```Go
package def

import (
	convAPI "github.com/sofmon/convention/v2/go/api"
	convAuth "github.com/sofmon/convention/v2/go/auth"
)

type MessageID string

type Message struct {
    MessageID MessageID `json:"message_id"`
    Text      string    `json:"text"`
}

type API struct {
	GetHealth convAPI.Out[string] `api:"GET /health/"`

	GetMessages convAPI.Out[[]Message] `api:"GET /message/v1/tenants/{tenant}/entities/{entity}/messages"`

    SendMessage convAPI.In[Message] `api:"PUT /message/v1/tenants/{tenant}/entities/{entity}/messages/{id}"`
}

var Client = convAPI.NewClient[API]("message-v2", 443)

func OpenAPI() convAPI.OpenAPI {
	return convAPI.NewOpenAPI().
		WithDescription(
			"Message service provides the ability to manage messages in the BRXS platform.",
		).
		WithServers(
			"https://api.brxs.com",
			"https://api.dev.brxs.com",
		).WithEnums(
		convAPI.NewEnum(
			ChannelNameCrisp,
			ChannelNameGChat,
		),
		convAPI.NewEnum(
			ConversationNameSupportNotes,
			ConversationNameSupportPrivate,
		),
		convAPI.NewEnum(
			MessageTypeText,
			MessageTypeNote,
		),
	)
}
```

## Packages

### `cfg` - Configuration Management
File-based configuration manager.

It is expected that all configurations would be securely stored and mounted to the file system by the environment runtime, like kubernetes, through deployment automation, like [sofmon/operations](github.com/sofmon/operations)

```Go
import convCfg "sofmon.com/convention/v2/go/cfg"

// Change where the configuration files are located
// Default value is "/etc/agent/"
convCfg.SetConfigLocation("./.secrets")

// get config value as string
value_as_string, err := convCfg.String("{{key}}")

// get config value as []byte
value_as_bytes, err := convCfg.Bytes("{{key}}")

// get an object from a JSON stored configuration
value_as_object, err := convCfg.Object[Messages]("{{key}}")

// When config location is certain you can use
// ...OrPanic versions of the methods, if error
// occurs the method will panic
value_as_string = convCfg.StringOrPanic("{{key}}")
```

### `auth` - Authentication & Authorization
JWT token handling and role-based access control.


- **Purpose**: JWT token handling and role-based access control
- **Key Types**: `Claims`, `Role`, `Permission`, `Action`, `Config`
- **Features**:
  - JWT token validation with `communication_secret`
  - Action template matching with placeholders (`{tenant}`, `{entity}`, `{user}`, `{any}`)
  - Multi-tenant and multi-entity access control

### `ctx` - Context Management
- **Purpose**: Convention/v2 compliant context with claims, workflow, and headers
- **Key Types**: `Context`, `Agent`, `Workflow`, `Environment`
- **Features**:
  - JWT claims integration
  - Workflow ID propagation
  - HTTP request/response tracing
  - Time simulation for non-production environments

### `api` - HTTP API Framework
- **Purpose**: Type-safe HTTP API definitions with automatic routing and OpenAPI generation
- **Key Types**: `Trigger`, `In`, `Out`, `InOut` with path parameter variants (P1-P4)
- **Features**:
  - Automatic endpoint discovery via struct tags
  - Type-safe client generation
  - Built-in authorization checking
  - OpenAPI 3.0 specification generation

### `db` - Database Access
- **Purpose**: Multi-tenant database connections with sharding support
- **Key Types**: `Vault`, `Engine` (postgres, sqlite3)
- **Features**:
  - Per-tenant database isolation
  - CRC32-based sharding across multiple databases
  - Vault-based database organization

## Usage Examples

### Creating an API Server

```go
package main

import (
    convAPI "github.com/sofmon/convention/v2/go/api"
    convAuth "github.com/sofmon/convention/v2/go/auth"
    convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type MessageAPI struct {
    GetMessages convAPI.Out[[]Message] `api:"GET /message/v1/tenants/{tenant}/entities/{entity}/messages"`
    SendMessage convAPI.In[Message]    `api:"PUT /message/v1/tenants/{tenant}/entities/{entity}/messages/{id}"`
    GetOpenAPI  convAPI.OpenAPI        `api:"GET /message/v1/openapi.yaml"`
}

func main() {
    authConfig := convAuth.Config{
        Roles: convAuth.RolePermissions{
            "user": {"read_messages", "write_messages"},
        },
        Permissions: convAuth.PermissionActions{
            "read_messages": {"GET /message/v1/tenants/{tenant}/entities/{entity}/messages"},
            "write_messages": {"PUT /message/v1/tenants/{tenant}/entities/{entity}/messages/{any}"},
        },
    }

    api := &MessageAPI{
        GetMessages: convAPI.NewOut(func(ctx convCtx.Context) ([]Message, error) {
            // Implementation
            return nil, nil
        }),
        SendMessage: convAPI.NewIn(func(ctx convCtx.Context, msg Message) error {
            // Implementation
            return nil
        }),
        GetOpenAPI: convAPI.NewOpenAPI(),
    }

    agentCtx := convCtx.New(convAuth.Claims{User: "message-v1"})

    server, err := convAPI.NewServer(agentCtx, "localhost", 443, authConfig, api)
    if err != nil {
        panic(err)
    }

    server.ListenAndServe()
}
```

### Creating an API Client

```go
client := convAPI.NewClient[MessageAPI]("api.example.com", 443)

ctx := convCtx.New(convAuth.Claims{
    User: "caller",
    Tenants: []convAuth.Tenant{"tenant1"},
    Entities: []convAuth.Entity{"entity1"},
    Roles: []convAuth.Role{"user"},
})

messages, err := client.GetMessages.Call(ctx)
if err != nil {
    // Handle error
}

err = client.SendMessage.Call(ctx, Message{Text: "Hello"})
```

### Database Access

```go
import convDB "github.com/sofmon/convention/v2/go/db"

// Get all databases for a vault/tenant
dbs, err := convDB.DBs("messages", "tenant1")

// Execute on specific shard
err = convDB.Insert(ctx, "messages", "tenant1").
    Shard("user123").
    Into("messages").
    Value("id", "msg1").
    Value("text", "Hello").
    Exec()

// Query across shards
results := convDB.Select(ctx, "messages", "tenant1").
    From("messages").
    Where("user_id", "user123").
    Query()
```

## Configuration

Place configuration files in `/etc/agent/`:

### `/etc/agent/environment`
```
production
```

### `/etc/agent/database`
```json
{
    "messages": {
        "tenant1": [
            {
                "engine": "postgres",
                "host": "db1.example.com",
                "port": 5432,
                "database": "messages_shard1",
                "username": "user",
                "password": "pass"
            }
        ]
    }
}
```

### `/etc/agent/communication_secret`
```
your-jwt-signing-secret
```

## API Endpoint Types

The framework provides several endpoint types:

- **`Trigger`**: No input/output (HEAD requests)
- **`In[T]`**: Input only (PUT requests)
- **`Out[T]`**: Output only (GET requests)
- **`InOut[T,U]`**: Input and output (POST requests)
- **`Raw`**: Raw HTTP request/response handling

Each type supports path parameters (P1, P2, P3, P4):
- **`TriggerP1[P1]`**: Single path parameter
- **`InP2[T,P1,P2]`**: Two path parameters with input
- **`OutP3[T,P1,P2,P3]`**: Three path parameters with output

## Authorization

Action templates support these placeholders:
- **`{any}`**: Match any value
- **`{any...}`**: Match any remaining path
- **`{user}`**: Must match authenticated user
- **`{tenant}`**: Must match user's allowed tenants
- **`{entity}`**: Must match user's allowed entities

## HTTP Headers

The implementation automatically handles Convention/v2 headers:
- **`Authorization`**: Bearer JWT token
- **`Workflow`**: Workflow identifier for request tracing
- **`Agent`**: Agent name
- **`Time-Now`**: Time simulation (non-production only)

## Testing

Run tests:
```bash
go test ./...
```

The implementation includes comprehensive tests demonstrating:
- Server/client communication
- Authorization enforcement
- Database operations
- Configuration loading

## Dependencies

- **Standard library**: `net/http`, `database/sql`, `context`, `encoding/json`
- **External**: `github.com/google/uuid` for workflow ID generation
- **Database drivers**: Import as needed (`github.com/lib/pq` for PostgreSQL, `github.com/mattn/go-sqlite3` for SQLite)

## Related Documentation

- [Convention/v2 Specification](../README.md)
- [Go Documentation](https://pkg.go.dev/github.com/sofmon/convention/v2/go)

