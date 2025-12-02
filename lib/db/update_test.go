package db_test

import (
	"testing"

	convAuth "github.com/sofmon/convention/lib/auth"
	convCtx "github.com/sofmon/convention/lib/ctx"
)

func Test_Update(t *testing.T) {

	ctx := convCtx.New(
		convAuth.Claims{
			User: "Test_Update",
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

	if msg.Content == "Updated content" {
		t.Fatalf("Unexpected content: %v", msg.Content)
	}

	msg.Content = "Updated content"
	err = messagesDB.Tenant("test").Update(ctx, *msg)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	msg, err = messagesDB.Tenant("test").SelectByID(ctx, msgs[0].MessageID)
	if err != nil {
		t.Fatalf("SelectByID failed: %v", err)
	}

	if msg == nil {
		t.Fatalf("SelectByID failed: nil")
	}

	if msg.Content != "Updated content" {
		t.Fatalf("Unexpected content: %v", msg.Content)
	}

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Delete(ctx, msg.MessageID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	}
}

func Test_SageUpdate(t *testing.T) {
	t.Skip("No md5 function in sqlite")
}
