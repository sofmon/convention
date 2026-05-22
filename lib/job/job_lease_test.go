package job_test

import (
	"context"
	"errors"
	"testing"
	"time"

	convCtx "github.com/sofmon/convention/lib/ctx"
	convDB "github.com/sofmon/convention/lib/db"
	convJob "github.com/sofmon/convention/lib/job"
)

// The heartbeat classifier: only a confirmed lost lease is fatal; transient
// errors are retried.
func TestRenewOutcome(t *testing.T) {
	if !convJob.RenewOutcomeForTest(convDB.ErrLeaseLost) {
		t.Fatal("ErrLeaseLost must be fatal (stop the job)")
	}
	if convJob.RenewOutcomeForTest(errors.New("transient db blip")) {
		t.Fatal("a transient error must NOT be fatal (heartbeat must retry)")
	}
	if convJob.RenewOutcomeForTest(nil) {
		t.Fatal("nil must not be fatal")
	}
}

// Register is idempotent: a second registration over a nil-closure in-memory entry
// (as syncJobsFromDB would leave) re-attaches the closure instead of erroring.
func TestRegisterIdempotentReattachesClosure(t *testing.T) {
	ctx := newCtx()
	const jid convJob.JobID = "idem-job"
	defer func() { _ = convJob.Unregister(ctx, testTenant, jid) }()

	if err := convJob.InsertJobRowForTest(ctx, testTenant, jid, time.Now().Add(time.Hour), 5*time.Minute); err != nil {
		t.Fatalf("insert job row: %v", err)
	}
	convJob.InjectNilClosureJobForTest(testTenant, jid, time.Now().Add(time.Hour), 5*time.Minute)

	if present, hasClosure := convJob.MemJobClosureForTest(testTenant, jid); !present || hasClosure {
		t.Fatalf("precondition: want present nil-closure entry, got present=%v hasClosure=%v", present, hasClosure)
	}

	err := convJob.Register(ctx, testTenant, jid, time.Now().Add(time.Hour), 5*time.Minute, func(convCtx.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("idempotent Register should not error, got: %v", err)
	}

	if present, hasClosure := convJob.MemJobClosureForTest(testTenant, jid); !present || !hasClosure {
		t.Fatalf("after Register want closure attached, got present=%v hasClosure=%v", present, hasClosure)
	}
}

// When the lease is lost mid-execution, executeJob must NOT advance/persist
// next_run_at (the new owner is now responsible for scheduling).
func TestExecuteJobSkipsAdvanceOnLeaseLoss(t *testing.T) {
	base := newCtx()
	ctx, dump := capturingCtx(base)
	const jid convJob.JobID = "lease-loss-job"
	t0 := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)

	if err := convJob.InsertJobRowForTest(ctx, testTenant, jid, t0, time.Hour); err != nil {
		t.Fatalf("insert job row: %v", err)
	}
	defer func() { _ = convJob.Unregister(ctx, testTenant, jid) }()

	// Single in-memory connection so the heartbeat goroutine and the foreign steal
	// share one database (Postgres needs no such pinning).
	if err := convJob.PinSingleConnForTest(convDB.Vault(testVault), testTenant); err != nil {
		t.Fatalf("pin conn: %v", err)
	}

	restore := convJob.SetLeaseForTest(10*time.Second, 10*time.Millisecond)
	defer restore()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// The job body steals its own lock (as a foreign owner), then waits for
		// cancellation — which the heartbeat triggers once its Renew sees the steal.
		convJob.RunJobForTest(ctx, testTenant, jid, t0, time.Hour, func(jctx convCtx.Context) error {
			if err := convJob.StealJobLockForTest(ctx, testTenant, jid, 10*time.Second); err != nil {
				t.Errorf("steal: %v", err)
				return err
			}
			<-jctx.Done()
			return nil
		})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("executeJob did not return; lease-loss cancellation likely not wired")
	}

	got, ok, err := convJob.ReadJobRowNextRunForTest(ctx, testTenant, jid)
	if err != nil || !ok {
		t.Fatalf("read job row: ok=%v err=%v", ok, err)
	}
	if !got.Equal(t0) {
		t.Fatalf("next_run_at advanced despite lease loss: got %v, want %v", got, t0)
	}
	if logged(dump(), "job execution completed") {
		t.Fatalf("must NOT emit completion log when the lease was lost mid-run")
	}
}

// A steal can land in the window between the heartbeat's last poll and the job
// returning, leaving leaseLost false even though ownership is already gone. executeJob
// must re-check ownership before advancing the schedule; otherwise it double-advances
// next_run_at and emits a false completion (the deferred Unlock then reports
// ErrLeaseLost). A long renew interval keeps the heartbeat from ever polling during the
// fast job body, isolating exactly that race (distinct from the heartbeat-detected case
// above).
func TestExecuteJobRechecksLeaseBeforeAdvance(t *testing.T) {
	base := newCtx()
	ctx, dump := capturingCtx(base)
	const jid convJob.JobID = "lease-recheck-job"
	t0 := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)

	if err := convJob.InsertJobRowForTest(ctx, testTenant, jid, t0, time.Hour); err != nil {
		t.Fatalf("insert job row: %v", err)
	}
	defer func() { _ = convJob.Unregister(ctx, testTenant, jid) }()

	if err := convJob.PinSingleConnForTest(convDB.Vault(testVault), testTenant); err != nil {
		t.Fatalf("pin conn: %v", err)
	}

	// Long renew interval: the heartbeat never ticks during the fast job body, so
	// leaseLost stays false and only the final ownership re-check can catch the steal.
	restore := convJob.SetLeaseForTest(10*time.Second, 10*time.Minute)
	defer restore()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// The job body steals its own lock (as a foreign owner) and returns at once,
		// never waiting for cancellation — so the heartbeat has no chance to observe it.
		convJob.RunJobForTest(ctx, testTenant, jid, t0, time.Hour, func(convCtx.Context) error {
			if err := convJob.StealJobLockForTest(ctx, testTenant, jid, 10*time.Second); err != nil {
				t.Errorf("steal: %v", err)
			}
			return nil
		})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("executeJob did not return")
	}

	got, ok, err := convJob.ReadJobRowNextRunForTest(ctx, testTenant, jid)
	if err != nil || !ok {
		t.Fatalf("read job row: ok=%v err=%v", ok, err)
	}
	if !got.Equal(t0) {
		t.Fatalf("next_run_at advanced despite a steal the heartbeat never observed: got %v, want %v", got, t0)
	}
	if logged(dump(), "job execution completed") {
		t.Fatalf("must NOT emit completion log for a run whose lease was stolen")
	}
}

// Scheduler shutdown (Cancel()) cancels the parent context. A job that ignores
// cancellation can run past its lease and be stolen; the final ownership check must
// still reach the DB. If it used the cancelled parent context it would get
// context.Canceled (treated as transient) before the DB call, and — because the schedule
// Update does not honour ctx cancellation — would advance next_run_at and log completion
// for a run it no longer owns.
func TestExecuteJobRechecksLeaseDespiteSchedulerCancel(t *testing.T) {
	base := newCtx()
	capCtx, dump := capturingCtx(base)
	ctx := capCtx
	var cancelFn context.CancelFunc
	ctx.Context, cancelFn = context.WithCancel(capCtx.Context)
	defer cancelFn()

	const jid convJob.JobID = "lease-cancel-job"
	t0 := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)

	if err := convJob.InsertJobRowForTest(ctx, testTenant, jid, t0, time.Hour); err != nil {
		t.Fatalf("insert job row: %v", err)
	}
	defer func() { _ = convJob.Unregister(base, testTenant, jid) }()

	if err := convJob.PinSingleConnForTest(convDB.Vault(testVault), testTenant); err != nil {
		t.Fatalf("pin conn: %v", err)
	}

	// Long renew interval: the heartbeat never polls during the fast job body, so it never
	// flips leaseLost — only the final ownership re-check can catch the steal.
	restore := convJob.SetLeaseForTest(10*time.Second, 10*time.Minute)
	defer restore()

	done := make(chan struct{})
	go func() {
		defer close(done)
		convJob.RunJobForTest(ctx, testTenant, jid, t0, time.Hour, func(convCtx.Context) error {
			// Steal while the parent ctx is still live, then simulate Cancel() landing
			// before executeJob's final ownership check.
			if err := convJob.StealJobLockForTest(ctx, testTenant, jid, 10*time.Second); err != nil {
				t.Errorf("steal: %v", err)
			}
			cancelFn()
			return nil
		})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("executeJob did not return")
	}

	// Read with the uncancelled base ctx (the test ctx is now cancelled).
	got, ok, err := convJob.ReadJobRowNextRunForTest(base, testTenant, jid)
	if err != nil || !ok {
		t.Fatalf("read job row: ok=%v err=%v", ok, err)
	}
	if !got.Equal(t0) {
		t.Fatalf("next_run_at advanced after scheduler cancel despite a steal: got %v, want %v", got, t0)
	}
	if logged(dump(), "job execution completed") {
		t.Fatalf("must NOT emit completion log for a stolen run during scheduler shutdown")
	}
}
