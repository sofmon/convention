package ctx

import (
	"encoding/json"
	"fmt"
	"time"
)

type logLevel string

const (
	logLevelTrace   logLevel = "trace"
	logLevelInfo    logLevel = "info"
	logLevelWarning logLevel = "warning"
	logLevelError   logLevel = "error"
)

type metadata map[string]any

type logEntry struct {
	Time     time.Time `json:"time,omitempty"`
	Level    logLevel  `json:"level,omitempty"`
	App      App       `json:"app,omitempty"`
	User     string    `json:"user,omitempty"`
	Message  any       `json:"message,omitempty"`
	Metadata metadata  `json:"metadata,omitempty"`
}

func log(e logEntry) {
	out, err := json.Marshal(e)
	if err != nil {
		fmt.Printf("%v\n", e)
	} else {
		fmt.Println(string(out))
	}
}
