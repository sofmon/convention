package db_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	_ "github.com/mattn/go-sqlite3"

	convCfg "github.com/sofmon/convention/lib/cfg"
	convCtx "github.com/sofmon/convention/lib/ctx"
	convDB "github.com/sofmon/convention/lib/db"
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

type ComplexID string

type ComplexNested struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type ComplexObject struct {
	ComplexID ComplexID         `json:"complex_id"`
	Title     string            `json:"title"`
	Nested    ComplexNested     `json:"nested"`
	Tags      []string          `json:"tags"`
	Attrs     map[string]string `json:"attrs"`

	// failOn is a test hook used by transaction_rollback_on_marshal_failure.
	// When set, MarshalJSON increments the counter and returns an error once
	// it reaches failOn. encoding/json skips unexported fields, so the hook
	// has no effect on the serialised bytes.
	marshalCount *int
	failOn       int
}

var errMarshalInjected = errors.New("injected marshal failure")

func (c ComplexObject) MarshalJSON() ([]byte, error) {
	if c.marshalCount != nil {
		*c.marshalCount++
		if *c.marshalCount >= c.failOn {
			return nil, errMarshalInjected
		}
	}
	type alias ComplexObject
	return json.Marshal(alias(c))
}

func (c ComplexObject) DBKey() convDB.Key[ComplexID, ComplexID] {
	return convDB.Key[ComplexID, ComplexID]{
		ID:       c.ComplexID,
		ShardKey: c.ComplexID,
	}
}

var complexDB = convDB.NewObjectSet[ComplexObject]("complex").Ready()

type SplitID string
type SplitShard string

type SplitKeyObject struct {
	SplitID    SplitID    `json:"split_id"`
	SplitShard SplitShard `json:"split_shard"`
	Payload    string     `json:"payload"`
}

func (s SplitKeyObject) DBKey() convDB.Key[SplitID, SplitShard] {
	return convDB.Key[SplitID, SplitShard]{
		ID:       s.SplitID,
		ShardKey: s.SplitShard,
	}
}

var splitKeyDB = convDB.NewObjectSet[SplitKeyObject]("complex").Ready()

func TestMain(m *testing.M) {

	err := convCfg.SetConfigLocation("../../.secret")
	if err != nil {
		err = fmt.Errorf("SetConfigLocation failed: %w", err)
		panic(err)
	}

	m.Run()
}
