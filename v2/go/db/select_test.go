package db_test

import (
	"testing"
	"time"

	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
	convDB "github.com/sofmon/convention/v2/go/db"
)

func Test_select(t *testing.T) {

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

	msgs, err = messagesDB.Tenant("test").Select(ctx,
		convDB.Where().
			Noop().
			And().
			Key("content").Equals().Value(testMessages[1].Content).
			And().
			CreatedBetween(time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(time.Hour)).
			And().
			CreatedBy("Test_select").
			And().
			UpdatedBetween(time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(time.Hour)).
			And().
			UpdatedBy("Test_select").
			And().
			Expression(
				convDB.Where().
					Noop().
					Or().
					UpdatedBy("unknown"),
			),
	)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("Unexpected messages count: %v", len(msgs))
	}

	for _, msg := range msgs {
		err := messagesDB.Tenant("test").Delete(ctx, msg.MessageID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	}

}
