# AGENTS.md - AI Agent Guidelines for ctx Package

## Package Purpose

The `ctx` package provides a structured context wrapper around Go's `context.Context` for consistent logging, error handling, scope tracking, and request-scoped data propagation.

> IMPORTANT: AI agents must treat AGENTS.md and README.md as authoritative living documents. Any change to the implementation that affects behaviors must be mirrored in both files. The code and documentation must never drift apart. When the implementation changes, these documents must be updated immediately so they always reflect the current system.

## Critical Patterns

### Function Signature Pattern

Every significant function MUST follow this pattern:

```go
func functionName(ctx ctx.Context, param1 Type1, param2 Type2) (result ResultType, err error) {
    ctx = ctx.WithScope("functionName", "param1", param1, "param2", param2)
    defer ctx.Exit(&err)

    // function body

    return
}
```

**Requirements:**
- `ctx ctx.Context` is ALWAYS the first parameter
- Named error return `err error` to work with `defer ctx.Exit(&err)`
- `WithScope` called immediately with function name and key parameters
- `defer ctx.Exit(&err)` called immediately after WithScope

### Functions Without Error Returns

For functions that don't return errors but still benefit from scope tracking:

```go
func processItem(ctx ctx.Context, item Item) {
    ctx = ctx.WithScope("processItem", "itemID", item.ID)
    defer ctx.Exit(nil)

    // function body
}
```

### When NOT to Use This Pattern

- Simple getters/setters (one-liners)
- Pure utility functions with no side effects

## File Structure

| File | Purpose |
|------|---------|
| `ctx.go` | Core Context struct, New(), WrapContext(), Agent() |
| `scope.go` | WithScope(), WithScopef(), Scope(), Exit(), wrapErr() |
| `workflow.go` | Workflow type, WithWorkflow(), WithNewWorkflow() |
| `log.go` | Logger(), WithLogger(), defaultLogger() |
| `env.go` | Environment type, Environment(), IsProdEnv() |
| `http.go` | HTTP integration, WithRequest(), Request(), headers |
| `claims.go` | WithClaims(), Claims(), User() |
| `action.go` | WithAction(), Action() |
| `now.go` | WithNow(), Now() |
| `sys.go` | WithAgentClaims(), mustUseAgentClaims() |

## Key Types

```go
type Context struct {
    context.Context
}

type Agent string
type Workflow string
type Environment string
```

## Context Keys (internal)

- `contextKeyAgent` - Agent name
- `contextKeyAgentClaims` - Original agent claims
- `contextKeyUseAgentClaims` - Flag to use agent claims
- `contextKeyEnv` - Environment
- `contextKeyRequest` - HTTP request
- `contextKeyClaims` - Current user claims
- `contextKeyAction` - Current action
- `contextKeyWorkflow` - Workflow ID
- `contextKeyScope` - Scope chain string
- `contextKeyNow` - Time override
- `contextKeyLogger` - slog.Logger instance

## Logger Keys (for structured logs)

- `env`, `agent`, `workflow`, `user`, `scope`, `action`, `now`, `use_agent_claims`

## Code Generation Rules

### Creating New Context Methods

When adding a new context value:

1. Add context key constant to `ctx.go`:
   ```go
   contextKeyNewThing contextKey = iota
   ```

2. Add logger key if needed (in `ctx.go`):
   ```go
   loggerKeyNewThing = "new_thing"
   ```

3. Create getter and setter in a new or appropriate file:
   ```go
   func (ctx Context) WithNewThing(value NewType) Context {
       return Context{
           context.WithValue(ctx.Context, contextKeyNewThing, value),
       }
   }

   func (ctx Context) NewThing() NewType {
       obj := ctx.Value(contextKeyNewThing)
       if obj == nil {
           return defaultValue
       }
       return obj.(NewType)
   }
   ```

4. If you want the value to appear in log output, add it to `Logger()` in `log.go`:
   ```go
   if newThing, ok := ctx.Value(contextKeyNewThing).(NewType); ok {
       attrs = append(attrs, loggerKeyNewThing, newThing)
   }
   ```

**Note:** The logger is built lazily from context values when `Logger()` is called. Do NOT call `WithLogger()` in `With*` methods - this prevents duplicate keys in log output.

### Error Handling

- Errors are wrapped with scope prefix: `✘ scope → subscope: error message`
- Use `Exit(&err, exceptedErrors...)` to exclude specific errors from wrapping
- Never wrap already-wrapped errors (checked by prefix)

### Scope String Format

- Scopes chain with ` → ` separator
- Arguments formatted as `{key=value key2=value2}`
- Example: `agent → processOrder → validateItems {itemID=123}`

## Dependencies

- `github.com/sofmon/convention/lib/auth` - Claims, User, Action types
- `github.com/sofmon/convention/lib/cfg` - Configuration for environment
- `github.com/google/uuid` - Workflow ID generation
- `log/slog` - Structured logging

## Testing Considerations

- Use `WithNow()` to inject fixed time for deterministic tests
- Workflow IDs are UUIDs; mock if needed for reproducible tests
- Environment defaults to "production" if config unavailable

## Common Mistakes to Avoid

1. **Forgetting `defer ctx.Exit(&err)`** - Errors won't be wrapped with scope
2. **Not passing ctx to called functions** - Breaks scope chain
3. **Storing context in structs** - Contexts are request-scoped
4. **Using unnamed error return** - `Exit()` requires pointer to named error
5. **Calling WithScope without function name** - Scope becomes unclear
6. **Not including key parameters in WithScope** - Loses debugging context
