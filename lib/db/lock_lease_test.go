package db_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	convAuth "github.com/sofmon/convention/lib/auth"
	convCtx "github.com/sofmon/convention/lib/ctx"
	convDB "github.com/sofmon/convention/lib/db"
)

const testLease = 60 * time.Second

// leaseCtx returns a context whose Now() is pinned, so lease arithmetic is fully
// deterministic (no sleeps).
func leaseCtx(user string, now time.Time) convCtx.Context {
	return convCtx.New(convAuth.Claims{User: convAuth.User(user)}).WithNow(now)
}

func leaseMsg() Message {
	return Message{MessageID: MessageID("lease-" + uuid.NewString())}
}

// A stale lease lock is stolen on acquire, and the steal is reported.
func Test_Lock_StealAfterExpiry(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	msg := leaseMsg()

	lockA, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerA", base), msg, "ownerA", convDB.WithLease(testLease))
	if err != nil || lockA == nil {
		t.Fatalf("ownerA acquire: lock=%v err=%v", lockA, err)
	}
	if lockA.Stolen() {
		t.Fatalf("a fresh acquire must not be marked stolen")
	}

	// Before expiry, a second owner cannot steal.
	early, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerB", base.Add(testLease/2)), msg, "ownerB", convDB.WithLease(testLease))
	if err != nil {
		t.Fatalf("ownerB early acquire err: %v", err)
	}
	if early != nil {
		t.Fatalf("ownerB must not steal a live lock")
	}

	// After expiry, the second owner steals and learns the previous owner.
	lockB, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerB", base.Add(testLease+time.Second)), msg, "ownerB", convDB.WithLease(testLease))
	if err != nil || lockB == nil {
		t.Fatalf("ownerB steal: lock=%v err=%v", lockB, err)
	}
	if !lockB.Stolen() {
		t.Fatalf("expected Stolen()=true after taking over an expired lock")
	}
	if lockB.PreviousOwner() != "ownerA" {
		t.Fatalf("PreviousOwner=%q, want ownerA", lockB.PreviousOwner())
	}

	if err := lockB.Unlock(); err != nil {
		t.Fatalf("ownerB unlock: %v", err)
	}
}

// Renewing keeps the lease alive so a would-be thief sees it as live (RowsAffected==0).
func Test_Lock_RenewPreventsSteal(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	msg := leaseMsg()

	lockA, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerA", base), msg, "ownerA", convDB.WithLease(testLease))
	if err != nil || lockA == nil {
		t.Fatalf("ownerA acquire: lock=%v err=%v", lockA, err)
	}

	// Heartbeat at base+lease/2 moves created_at forward.
	if err := lockA.Renew(leaseCtx("ownerA", base.Add(testLease/2))); err != nil {
		t.Fatalf("renew: %v", err)
	}

	// At base+lease the cutoff is base; the renewed created_at (base+lease/2) is
	// newer, so the thief must fail.
	thief, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerB", base.Add(testLease)), msg, "ownerB", convDB.WithLease(testLease))
	if err != nil {
		t.Fatalf("thief acquire err: %v", err)
	}
	if thief != nil {
		t.Fatalf("renew should have kept the lease alive; thief must not steal")
	}

	if err := lockA.Unlock(); err != nil {
		t.Fatalf("ownerA unlock: %v", err)
	}
}

// A stale owner's Unlock returns ErrLeaseLost and must not remove the new owner's row.
func Test_Lock_OwnerSafeUnlock(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	msg := leaseMsg()

	lockA, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerA", base), msg, "ownerA", convDB.WithLease(testLease))
	if err != nil || lockA == nil {
		t.Fatalf("ownerA acquire: lock=%v err=%v", lockA, err)
	}

	lockB, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerB", base.Add(testLease+time.Second)), msg, "ownerB", convDB.WithLease(testLease))
	if err != nil || lockB == nil {
		t.Fatalf("ownerB steal: lock=%v err=%v", lockB, err)
	}

	// ownerA lost the lease — its Unlock reports it and leaves ownerB's row intact.
	if err := lockA.Unlock(); !errors.Is(err, convDB.ErrLeaseLost) {
		t.Fatalf("stale owner Unlock: got %v, want ErrLeaseLost", err)
	}

	// Probe: ownerB's lock is still live (a fresh thief just past the steal time fails).
	probe, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerC", base.Add(testLease+2*time.Second)), msg, "ownerC", convDB.WithLease(testLease))
	if err != nil {
		t.Fatalf("probe err: %v", err)
	}
	if probe != nil {
		t.Fatalf("ownerA.Unlock must not have removed ownerB's live lock")
	}

	if err := lockB.Unlock(); err != nil {
		t.Fatalf("ownerB unlock: %v", err)
	}
}

// Ownership is per-acquisition, not per-description: two acquisitions that pass the SAME
// human-readable description must still be distinct owners, so a stale holder cannot renew
// or unlock a lock a new holder took over with the identical description.
func Test_Lock_SameDescriptionDistinctOwners(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	msg := leaseMsg()

	lockA, err := messagesDB.Tenant("test").Lock(leaseCtx("dup", base), msg, "dup", convDB.WithLease(testLease))
	if err != nil || lockA == nil {
		t.Fatalf("first acquire: lock=%v err=%v", lockA, err)
	}

	// A second caller takes over after expiry using the SAME description.
	lockB, err := messagesDB.Tenant("test").Lock(leaseCtx("dup", base.Add(testLease+time.Second)), msg, "dup", convDB.WithLease(testLease))
	if err != nil || lockB == nil {
		t.Fatalf("second acquire (steal): lock=%v err=%v", lockB, err)
	}

	// The stale first holder must NOT be able to renew or unlock lockB's row despite the
	// identical description — otherwise it would silently keep/steal a lock it no longer owns.
	if err := lockA.Renew(leaseCtx("dup", base.Add(testLease+2*time.Second))); !errors.Is(err, convDB.ErrLeaseLost) {
		t.Fatalf("stale holder Renew with identical description must report ErrLeaseLost, got %v", err)
	}
	if err := lockA.Unlock(); !errors.Is(err, convDB.ErrLeaseLost) {
		t.Fatalf("stale holder Unlock with identical description must report ErrLeaseLost, got %v", err)
	}

	// lockB still genuinely owns it.
	if err := lockB.Renew(leaseCtx("dup", base.Add(testLease+3*time.Second))); err != nil {
		t.Fatalf("current holder Renew should succeed: %v", err)
	}
	if err := lockB.Unlock(); err != nil {
		t.Fatalf("current holder Unlock: %v", err)
	}
}

// UpdateGuarded persists only while the caller still holds a live lease: the write and the
// ownership check are one atomic statement, so a stolen holder cannot persist a stale write.
func Test_Lock_UpdateGuarded(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := leaseCtx("owner", base)
	msg := leaseMsg()
	msg.Content = "v0"
	if err := messagesDB.Tenant("test").Insert(ctx, msg); err != nil {
		t.Fatalf("insert: %v", err)
	}
	defer func() { _ = messagesDB.Tenant("test").Delete(leaseCtx("owner", base), msg.MessageID) }()

	lockA, err := messagesDB.Tenant("test").Lock(ctx, msg, "ownerA", convDB.WithLease(testLease))
	if err != nil || lockA == nil {
		t.Fatalf("acquire: lock=%v err=%v", lockA, err)
	}

	// A live owner persists.
	upd := msg
	upd.Content = "v1"
	if err := lockA.UpdateGuarded(leaseCtx("ownerA", base.Add(time.Second)), upd); err != nil {
		t.Fatalf("live owner UpdateGuarded should succeed: %v", err)
	}
	if got, _ := messagesDB.Tenant("test").SelectByID(ctx, msg.MessageID); got == nil || got.Content != "v1" {
		t.Fatalf("UpdateGuarded did not persist: %+v", got)
	}

	// The UpdateGuarded above renewed the lease (created_at = base+1s), so the steal must be
	// at least one lease past that. A thief takes over; the stale owner's guarded write must
	// NOT land.
	lockB, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerB", base.Add(testLease+2*time.Second)), msg, "ownerB", convDB.WithLease(testLease))
	if err != nil || lockB == nil {
		t.Fatalf("steal: lock=%v err=%v", lockB, err)
	}
	stale := msg
	stale.Content = "v2-must-not-persist"
	if err := lockA.UpdateGuarded(leaseCtx("ownerA", base.Add(testLease+3*time.Second)), stale); !errors.Is(err, convDB.ErrLeaseLost) {
		t.Fatalf("stale owner UpdateGuarded must report ErrLeaseLost, got %v", err)
	}
	if got, _ := messagesDB.Tenant("test").SelectByID(ctx, msg.MessageID); got == nil || got.Content != "v1" {
		t.Fatalf("stale UpdateGuarded must not have persisted: %+v", got)
	}

	// The current owner can still persist.
	win := msg
	win.Content = "v3"
	if err := lockB.UpdateGuarded(leaseCtx("ownerB", base.Add(testLease+4*time.Second)), win); err != nil {
		t.Fatalf("current owner UpdateGuarded: %v", err)
	}
	if err := lockB.Unlock(); err != nil {
		t.Fatalf("ownerB unlock: %v", err)
	}
}

// Expiry alone does not block UpdateGuarded: a holder whose lease lapsed but was NOT stolen
// is still the sole owner, so its owner-checked renew succeeds and it persists. (Only an
// actual steal blocks it — see Test_Lock_UpdateGuarded.) This avoids wasting a completed run
// when the holder merely paused, e.g. a GC stall, without anyone taking over.
func Test_Lock_UpdateGuardedExpiredButUnstolen(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := leaseCtx("owner", base)
	msg := leaseMsg()
	msg.Content = "v0"
	if err := messagesDB.Tenant("test").Insert(ctx, msg); err != nil {
		t.Fatalf("insert: %v", err)
	}
	defer func() { _ = messagesDB.Tenant("test").Delete(ctx, msg.MessageID) }()

	lock, err := messagesDB.Tenant("test").Lock(ctx, msg, "owner", convDB.WithLease(testLease))
	if err != nil || lock == nil {
		t.Fatalf("acquire: lock=%v err=%v", lock, err)
	}

	// No renew and well past the lease, but nobody stole it — the owner-checked renew still
	// matches this owner, so the write lands.
	upd := msg
	upd.Content = "v1"
	if err := lock.UpdateGuarded(leaseCtx("owner", base.Add(testLease+time.Second)), upd); err != nil {
		t.Fatalf("expired-but-unstolen lease should still persist (no steal happened): %v", err)
	}
	if got, _ := messagesDB.Tenant("test").SelectByID(ctx, msg.MessageID); got == nil || got.Content != "v1" {
		t.Fatalf("expired-but-unstolen UpdateGuarded should have persisted: %+v", got)
	}
}

// After a steal, the original owner's Renew reports ErrLeaseLost (the trigger the
// scheduler uses to stop advancing the schedule).
func Test_Lock_RenewAfterStealReturnsErrLeaseLost(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	msg := leaseMsg()

	lockA, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerA", base), msg, "ownerA", convDB.WithLease(testLease))
	if err != nil || lockA == nil {
		t.Fatalf("ownerA acquire: lock=%v err=%v", lockA, err)
	}

	lockB, err := messagesDB.Tenant("test").Lock(leaseCtx("ownerB", base.Add(testLease+time.Second)), msg, "ownerB", convDB.WithLease(testLease))
	if err != nil || lockB == nil {
		t.Fatalf("ownerB steal: lock=%v err=%v", lockB, err)
	}

	if err := lockA.Renew(leaseCtx("ownerA", base.Add(testLease+2*time.Second))); !errors.Is(err, convDB.ErrLeaseLost) {
		t.Fatalf("stale Renew: got %v, want ErrLeaseLost", err)
	}

	if err := lockB.Unlock(); err != nil {
		t.Fatalf("ownerB unlock: %v", err)
	}
}
