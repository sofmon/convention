package db_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	_ "github.com/mattn/go-sqlite3"

	convCfg "github.com/sofmon/convention/v2/go/cfg"
	convDB "github.com/sofmon/convention/v2/go/db"
)

type MessageID string

type Message struct {
	MessageID MessageID `json:"message_id"`
	Content   string    `json:"content"`
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

var messagesDB = convDB.NewObjectSet[Message]("messages")

func TestMain(m *testing.M) {

	err := convCfg.SetConfigLocation("../../../.secret")
	if err != nil {
		err = fmt.Errorf("SetConfigLocation failed: %w", err)
		panic(err)
	}

	m.Run()
}
