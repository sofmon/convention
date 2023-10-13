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

type traceEntry struct {
	Time       time.Time `json:"time,omitempty"`
	App        App       `json:"app,omitempty"`
	User       string    `json:"user,omitempty"`
	RequestID  RequestID `json:"request_id,omitempty"`
	Method     string    `json:"method,omitempty"`
	Path       string    `json:"path,omitempty"`
	StatusCode int       `json:"status_code,omitempty"`
	Request    string    `json:"request,omitempty"`
	Response   string    `json:"response,omitempty"`
}

func trace(e traceEntry) {
	out, err := json.Marshal(e)
	if err != nil {
		fmt.Printf("%v\n", e)
	} else {
		fmt.Println(string(out))
	}
}
