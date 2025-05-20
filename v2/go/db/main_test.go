package db_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	_ "github.com/mattn/go-sqlite3"

	convCfg "github.com/sofmon/convention/v2/go/cfg"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
	convDB "github.com/sofmon/convention/v2/go/db"
)

type MessageID string

type Message struct {
	MessageID MessageID `json:"message_id"`
	Content   string    `json:"content"`

	CreatedAt time.Time `json:"created_at"` // to test compute
	UpdatedAt time.Time `json:"updated_at"` // to test compute
}

func (m Message) DBKey() convDB.Key[MessageID, MessageID] {
	return convDB.Key[MessageID, MessageID]{
		ID:       m.MessageID,
		ShardKey: m.MessageID,
	}
}

func generateTestMessages() (res []Message) {
	for i := 0; i < 100; i++ {
		res = append(res, Message{
			MessageID: MessageID(uuid.NewString()),
			Content:   uuid.NewString(),
		})
	}
	return
}

var messagesDB = convDB.NewObjectSet[Message]("messages").
	WithCompute(
		func(ctx convCtx.Context, md convDB.Metadata, obj *Message) error {
			obj.CreatedAt = md.CreatedAt
			obj.UpdatedAt = md.UpdatedAt
			return nil
		},
	).
	Ready()

func TestMain(m *testing.M) {

	err := convCfg.SetConfigLocation("../../../.secret")
	if err != nil {
		err = fmt.Errorf("SetConfigLocation failed: %w", err)
		panic(err)
	}

	m.Run()
}
