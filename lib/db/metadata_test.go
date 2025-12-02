package db_test

import (
	"testing"

	convAuth "github.com/sofmon/convention/lib/auth"
	convCtx "github.com/sofmon/convention/lib/ctx"
)

func Test_metadata(t *testing.T) {

	ctx := convCtx.New(convAuth.Claims{User: "Test_select"})

	testMessages := generateTestMessages()

	for _, msg := range testMessages {
		err := messagesDB.Tenant("test").Insert(ctx, msg)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	msg, err := messagesDB.Tenant("test").SelectByID(ctx, testMessages[0].MessageID)
	if err != nil {
		t.Fatalf("SelectByID failed: %v", err)
	}

	if msg == nil {
		t.Fatalf("SelectByID failed: nil")
	}

	if msg.Content != testMessages[0].Content {
		t.Fatalf("Unexpected content: %v", msg.Content)
	}

	msgs, err := messagesDB.Tenant("test").SelectAll(ctx)
	if err != nil {
		t.Fatalf("SelectAll failed: %v", err)
	}

	if len(msgs) != len(testMessages) {
		t.Fatalf("Unexpected messages count: %v", len(msgs))
	}

	metadata, err := messagesDB.Tenant("test").Metadata(ctx, testMessages[0].MessageID)
	if err != nil {
		t.Fatalf("Metadata failed: %v", err)
	}

	if metadata == nil {
		t.Fatalf("Metadata failed: nil")
	}

	if metadata.CreatedBy != "Test_select" {
		t.Fatalf("Unexpected createdBy: %v", metadata.CreatedBy)
	}

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Delete(ctx, msg.MessageID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	}

}
