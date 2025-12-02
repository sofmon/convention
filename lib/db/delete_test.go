package db_test

import (
	"testing"

	convAuth "github.com/sofmon/convention/lib/auth"
	convCtx "github.com/sofmon/convention/lib/ctx"
)

func Test_Delete(t *testing.T) {

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

	msg, err := messagesDB.Tenant("test").SelectByID(ctx, msgs[0].MessageID)
	if err != nil {
		t.Fatalf("SelectByID failed: %v", err)
	}

	if msg == nil {
		t.Fatalf("SelectByID failed: nil")
	}

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Delete(ctx, msg.MessageID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	}

	msg, err = messagesDB.Tenant("test").SelectByID(ctx, msgs[0].MessageID)
	if err != nil {
		t.Fatalf("SelectByID failed: %v", err)
	}

	if msg != nil {
		t.Fatalf("SelectByID failed: not nil")
	}
}
