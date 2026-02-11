# Job Package Implementation Details (for AI Agents)

This document provides implementation details for AI agents working on the `lib/job` package.

> IMPORTANT: AI agents must treat AGENTS.md and README.md as authoritative living documents. Any change to the implementation that affects behaviors must be mirrored in both files. The code and documentation must never drift apart. When the implementation changes, these documents must be updated immediately so they always reflect the current system.

## File Organization

- **[job.go](job.go)** - All types, state, public API, and background loop implementation

## Core Types

### Public Types (job.go:14-18)

```go
type JobID string                           // Unique identifier per tenant
type JobState string                        // Currently unused, reserved for future use
type JobFunc func(convCtx.Context) error    // Job function signature
```

### Internal Types (job.go:20-26)

```go
type job struct {
    ID          JobID         `json:"id"`
    NextRunAt   time.Time     `json:"next_run_at"`
    RepeatEvery time.Duration `json:"repeat_every"`
    f           JobFunc       `json:"-"`  // In-memory only, not serialized
}
```

The `f` field is critical -- it is `json:"-"` and therefore never persisted. When syncing from the database, the `f` from the in-memory map must be preserved. Jobs loaded from the database without a local `f` are tracked for scheduling awareness but cannot be executed by that instance.

### DBKey (job.go:28-33)

```go
func (x job) DBKey() convDB.Key[JobID, JobID]
```

Uses `JobID` for both ID and ShardKey. This means each job is its own shard key, ensuring lock operations target the correct database shard.

## Package State (job.go:35-41)

```go
var (
    jobs   map[convAuth.Tenant]map[JobID]job    // In-memory job registry
    mut    sync.Mutex                            // Protects all access to jobs map
    cancel context.CancelFunc                    // Cancels the background goroutine
    jobsDB convDB.ObjectSetReady[job, JobID, JobID]  // Database handle
    wakeUp chan struct{}                          // Buffered channel to wake background loop
)
```

All package state is global. The package is designed as a singleton scheduler -- only one `Initialise` call is allowed at a time.

## Public API

### Initialise (job.go:133-149)

```go
func Initialise(ctx convCtx.Context, vault convDB.Vault) error
```

1. Acquires mutex
2. Fails if already running (`cancel != nil`)
3. Creates `jobsDB` via `convDB.NewObjectSet`
4. Creates `wakeUp` channel (buffered, capacity 1)
5. Creates cancellable context from provided context
6. Launches `background()` goroutine

### Register (job.go:42-91)

```go
func Register(ctx convCtx.Context, tenant convAuth.Tenant, jid JobID, startAt time.Time, repeatEvery time.Duration, fn JobFunc) error
```

1. Fails if `jobsDB == nil` (not initialised)
2. Fails if job ID already exists in memory for tenant
3. Checks database for existing job (`SelectByID`)
4. If not in DB: creates new job and inserts
5. If in DB: attaches the function and updates schedule parameters
6. Stores in in-memory map
7. Sends a non-blocking signal on `wakeUp` channel to wake the background loop

### Unregister (job.go:93-116)

```go
func Unregister(ctx convCtx.Context, tenant convAuth.Tenant, jid JobID) error
```

1. Validates tenant and job exist in memory
2. Deletes from in-memory map
3. Deletes from database

### Cancel (job.go:118-131)

```go
func Cancel() error
```

Calls the cancel function and sets it to nil. The background goroutine detects cancellation via `ctx.Done()` and exits.

## Background Loop (job.go:151-192)

```go
func background(ctx convCtx.Context)
```

Main scheduling loop with three responsibilities:

### Loop Structure

```
syncInterval = 1 minute
lastSync = zero time (forces immediate first sync)

loop:
    if time since lastSync >= syncInterval:
        syncJobsFromDB(ctx)
        lastSync = now

    nextWakeUp = executeAndSchedule(ctx)

    sleepDuration = min(time until next sync, time until nextWakeUp)
    clamp to >= 1 second

    select:
        case <-ctx.Done(): return
        case <-wakeUp: stop timer, continue (new job registered)
        case <-timer.C: continue
```

Uses `time.Timer` (not `time.Ticker`) because sleep duration is dynamic. The 1-second minimum prevents tight loops on transient errors or when all jobs are due simultaneously.

## Helper Functions

### syncJobsFromDB (job.go:194-234)

```go
func syncJobsFromDB(ctx convCtx.Context)
```

Holds the mutex for the entire operation. For each tenant:

1. Calls `jobsDB.Tenant(tenant).SelectAll(ctx)`
2. Errors are logged and the tenant is skipped (retries next cycle)
3. Merges DB state into memory: updates schedule fields from DB while preserving `f`
4. Adds DB-only jobs with `f == nil` (tracked but not executable)

**Tenant discovery limitation**: Only syncs tenants already present in the `jobs` map. A tenant must have at least one local `Register()` call to be known. This is by design.

### executeAndSchedule (job.go:236-283)

```go
func executeAndSchedule(ctx convCtx.Context) (nextWakeUp time.Time)
```

Two-phase operation:

**Phase 1 (under mutex)**: Snapshot all due jobs (`NextRunAt <= now` and `f != nil`) into a local slice, then release the mutex.

**Execution (no mutex)**: Execute each due job via `executeJob()`. This allows `Register`/`Unregister` to proceed concurrently.

**Phase 2 (under mutex)**: Re-scan all jobs to find the earliest future `NextRunAt`. Return it as `nextWakeUp` (zero value if no future jobs exist).

### executeJob (job.go:285-370)

```go
func executeJob(ctx convCtx.Context, tenant convAuth.Tenant, j job)
```

Execution flow:

1. **Guard**: Skip if `RepeatEvery <= 0` (prevents infinite loop in schedule advancement)
2. **Lock**: `jobsDB.Tenant(tenant).Lock(ctx, j, ...)` -- non-blocking. If `lock == nil`, another instance holds it; skip silently.
3. **Defer unlock**: `lock.Unlock()` is deferred immediately after acquisition
4. **Execute**: Run `j.f(ctx)` inside a nested func with `recover()` for panic safety. Errors and panics are logged.
5. **Advance schedule**: Compute `nextRunAt = NextRunAt + RepeatEvery`, advancing past any missed intervals via a `for` loop
6. **Update memory**: Acquire mutex, update `NextRunAt` in the in-memory map
7. **Update database**: Call `jobsDB.Tenant(tenant).Update(ctx, updatedJob)` to persist the new schedule

**Error policy**: `NextRunAt` is advanced even when the job function returns an error. This prevents a permanently failing job from retrying in a tight loop. The error is logged for observability.

## Database Interaction

### Tables

The `convDB.NewObjectSet[job, JobID, JobID](vault).Ready()` call creates three tables (see [db AGENTS.md](../db/AGENTS.md)):

- `job` -- Runtime table with JSONB object column
- `job_history` -- History tracking
- `job_lock` -- Pessimistic lock table

### Operations Used

| Operation | Where Used | Purpose |
|---|---|---|
| `SelectAll` | `syncJobsFromDB` | Fetch all jobs for a tenant |
| `SelectByID` | `Register` | Check if job already exists in DB |
| `Insert` | `Register` | Persist new job |
| `Delete` | `Unregister` | Remove job from DB |
| `Lock` | `executeJob` | Acquire execution lock |
| `Unlock` | `executeJob` | Release execution lock |
| `Update` | `executeJob` | Persist updated `NextRunAt` |

### Lock Mechanism

Uses `convDB` pessimistic locking ([lock.go](../db/lock.go)):

```sql
INSERT INTO "job_lock" ("id","created_at","description")
VALUES($1,$2,$3)
ON CONFLICT ("id") DO NOTHING;
```

- Non-blocking: returns immediately
- `RowsAffected == 1` means lock acquired
- `RowsAffected == 0` means another instance holds the lock
- Lock persists until `DELETE FROM "job_lock" WHERE "id"=$1`

## Concurrency Model

### Mutex Scope

The `mut` mutex protects all reads and writes to the `jobs` map. It is held:

- During `Register` and `Unregister` (entire function)
- During `syncJobsFromDB` (entire function, including DB calls)
- During `executeAndSchedule` Phase 1 (snapshot) and Phase 2 (re-scan)
- During `executeJob` in-memory update (brief, just map write)

The mutex is intentionally **not** held during job execution or database lock/update operations in `executeJob`.

### Cross-Instance Coordination

Database locks (not the in-memory mutex) provide cross-instance mutual exclusion. The mutex only protects the local `jobs` map.

## Known Limitations

1. **Tenant discovery is local**: Only tenants with at least one locally registered job are synced from the database.
2. **Sequential job execution**: Due jobs within a single loop iteration are executed sequentially, not in parallel.
3. **Mutex held during DB sync**: `syncJobsFromDB` holds the mutex while calling `SelectAll`, which blocks `Register`/`Unregister` during sync. For small job counts this is acceptable.
4. **No job execution timeout**: Job functions can run indefinitely. The context has cancellation but no deadline per job.
5. **`JobState` type is unused**: Defined but not referenced anywhere.

## Common Modification Scenarios

### Adding Job Execution Timeout

1. In `executeJob`, wrap `j.f(ctx)` call with a context deadline:
   ```go
   timeoutCtx, cancel := context.WithTimeout(ctx.Context, timeout)
   defer cancel()
   ```
2. Consider making timeout configurable per job (add field to `job` struct)

### Adding Parallel Job Execution

1. In `executeAndSchedule`, replace the sequential `for` loop with goroutines and `sync.WaitGroup`
2. Be careful with the mutex -- `executeJob` already acquires it briefly for the in-memory update

### Adding One-Shot Jobs (No Repeat)

1. Allow `RepeatEvery == 0` to mean "run once"
2. In `executeJob`, after execution: if `RepeatEvery == 0`, delete the job instead of advancing schedule
3. Update the `RepeatEvery <= 0` guard to only reject negative values
