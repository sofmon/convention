package db_test

import (
	"testing"

	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
	convDB "github.com/sofmon/convention/v2/go/db"
)

func Test_Process(t *testing.T) {

	ctx := convCtx.New(
		convAuth.Claims{
			User: "Test_Insert",
		},
	)

	msgs := generateTestMessages()

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Insert(ctx, msg)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	count, err := messagesDB.Tenant("test").Process(
		ctx,
		convDB.Where().Noop(),
		func(ctx convCtx.Context, obj Message) error {
			// do nothing
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if count != len(msgs) {
		t.Fatalf("Unexpected count: %v", count)
	}

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Delete(ctx, msg.MessageID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	}

}

func Test_ProcessWithMetadata(t *testing.T) {

	ctx := convCtx.New(
		convAuth.Claims{
			User: "Test_ProcessWithMetadata",
		},
	)

	msgs := generateTestMessages()

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Insert(ctx, msg)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	count, err := messagesDB.Tenant("test").ProcessWithMetadata(
		ctx,
		convDB.Where().Noop(),
		func(ctx convCtx.Context, obj convDB.ObjectWithMetadata[Message]) error {
			// do nothing
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if count != len(msgs) {
		t.Fatalf("Unexpected count: %v", count)
	}

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Delete(ctx, msg.MessageID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	}

}
