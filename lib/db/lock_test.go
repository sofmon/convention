package db_test

import (
	"testing"

	convAuth "github.com/sofmon/convention/lib/auth"
	convCtx "github.com/sofmon/convention/lib/ctx"
)

func Test_Lock(t *testing.T) {

	ctx := convCtx.New(
		convAuth.Claims{
			User: "Test_Lock",
		},
	)

	msgs := generateTestMessages()

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Insert(ctx, msg)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	msg, err := messagesDB.Tenant("test").SelectByID(ctx, msgs[0].MessageID)
	if err != nil {
		t.Fatalf("SelectByID failed: %v", err)
	}

	if msg == nil {
		t.Fatalf("SelectByID failed: nil")
	}

	lock, err := messagesDB.Tenant("test").Lock(ctx, *msg, "test lock")
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}

	if lock == nil {
		t.Fatalf("Lock failed: nil")
	}

	lock2, err := messagesDB.Tenant("test").Lock(ctx, *msg, "test lock")
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}

	if lock2 != nil {
		t.Fatalf("Lock failed: %v", lock2)
	}

	err = lock.Unlock()
	if err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Delete(ctx, msg.MessageID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	}
}

func Test_SelectByIDAndLock(t *testing.T) {

	ctx := convCtx.New(
		convAuth.Claims{
			User: "Test_Upsert",
		},
	)

	msgs := generateTestMessages()

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Upsert(ctx, msg)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	msg, lock, err := messagesDB.Tenant("test").SelectByIDAndLock(ctx, msgs[0].MessageID, "test lock")
	if err != nil {
		t.Fatalf("SelectByID failed: %v", err)
	}

	if msg == nil {
		t.Fatalf("SelectByID failed: nil")
	}

	if lock == nil {
		t.Fatalf("SelectByID failed: nil")
	}

	msg2, lock2, err := messagesDB.Tenant("test").SelectByIDAndLock(ctx, msgs[0].MessageID, "test lock 2")
	if err != nil {
		t.Fatalf("SelectByID failed: %v", err)
	}

	if msg2 != nil {
		t.Fatalf("SelectByID failed: %v", msg2)
	}

	if lock2 != nil {
		t.Fatalf("SelectByID failed: %v", lock2)
	}

	err = lock.Unlock()
	if err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	msg2, lock2, err = messagesDB.Tenant("test").SelectByIDAndLock(ctx, msgs[0].MessageID, "test lock 2")
	if err != nil {
		t.Fatalf("SelectByID failed: %v", err)
	}

	if msg2 == nil {
		t.Fatalf("SelectByID failed: nil")
	}

	if lock2 == nil {
		t.Fatalf("SelectByID failed: nil")
	}

	err = lock2.Unlock()
	if err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Delete(ctx, msg.MessageID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	}
}
