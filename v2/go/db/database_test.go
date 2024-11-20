package db_test

import (
	"testing"

	"github.com/sofmon/convention/v2/go/auth"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
	convDB "github.com/sofmon/convention/v2/go/db"
)

type MessageID string

type Message struct {
	MessageID MessageID
	Content   string
}

func (m Message) DBKey() convDB.Key[MessageID, MessageID] {
	return convDB.Key[MessageID, MessageID]{
		ID:       m.MessageID,
		ShardKey: m.MessageID,
	}
}

func Test_Insert_And_Select(t *testing.T) {

	err := convDB.Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	messagesDB, err := convDB.NewObjectSet[Message]("messages", false)
	if err != nil {
		t.Fatalf("NewObjectSet failed: %v", err)
	}

	ctx := convCtx.New(
		auth.Claims{
			User: "test",
		},
	)

	err = messagesDB.Tenant("test").Insert(
		ctx,
		Message{
			MessageID: "1",
			Content:   "Hello, World!",
		},
	)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	msg, err := messagesDB.Tenant("test").SelectByID(ctx, "1")
	if err != nil {
		t.Fatalf("SelectByID failed: %v", err)
	}

	if msg == nil {
		t.Fatalf("SelectByID failed: nil")
	}

	if msg.Content != "Hello, World!" {
		t.Fatalf("Unexpected content: %v", msg.Content)
	}

}
