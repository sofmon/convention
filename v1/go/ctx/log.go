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

func (ctx Context) LogError(v any) {
	ctx.logMsg(logLevelError, v)
}

func (ctx Context) LogErrorf(format string, a ...any) {
	ctx.logMsg(logLevelError, fmt.Sprintf(format, a...))
}

func (ctx Context) LogWarn(v any) {
	ctx.logMsg(logLevelWarning, v)
}

func (ctx Context) LogWarnf(format string, a ...any) {
	ctx.logMsg(logLevelWarning, fmt.Sprintf(format, a...))
}

func (ctx Context) LogInfo(v any) {
	ctx.logMsg(logLevelInfo, v)
}

func (ctx Context) LogInfof(format string, a ...any) {
	ctx.logMsg(logLevelInfo, fmt.Sprintf(format, a...))
}

type metadata map[string]any

type logEntry struct {
	Time     time.Time `json:"time,omitempty"`
	Level    logLevel  `json:"level,omitempty"`
	App      App       `json:"app,omitempty"`
	User     string    `json:"user,omitempty"`
	Message  any       `json:"message,omitempty"`
	Metadata metadata  `json:"metadata,omitempty"`
}

func (ctx Context) logMsg(level logLevel, v any) {
	entry := logEntry{
		Time:  ctx.Now(),
		Level: level,
		App:   ctx.App(),
		User:  ctx.User(),
	}

	if err, ok := v.(error); ok {
		entry.Message = err.Error()
	} else {
		entry.Message = v
	}

	if r := ctx.Request(); r != nil {
		entry.Metadata = metadata{
			"request_path":   r.URL.Path,
			"request_method": r.Method,
			"request_id":     ctx.RequestID(),
		}
	}

	out, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("%v\n", entry)
	} else {
		fmt.Println(string(out))
	}
}
