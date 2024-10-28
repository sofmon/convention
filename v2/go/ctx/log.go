package ctx

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sofmon/convention/v2/go/auth"
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

type logEntry struct {
	Time     time.Time   `json:"time,omitempty"`
	Level    logLevel    `json:"level,omitempty"`
	Agent    Agent       `json:"agent,omitempty"`
	User     auth.User   `json:"user,omitempty"`
	Action   auth.Action `json:"action,omitempty"`
	Workflow Workflow    `json:"task,omitempty"`
	Scope    string      `json:"scope,omitempty"`
	Message  any         `json:"message,omitempty"`
}

func (ctx Context) logMsg(level logLevel, v any) {

	entry := logEntry{
		Time:     ctx.Now(),
		Level:    level,
		Agent:    ctx.Agent(),
		User:     ctx.User(),
		Workflow: ctx.Workflow(),
		Scope:    ctx.Scope(),
	}

	if r := ctx.Request(); r != nil {
		entry.Action = auth.Action(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
	}

	if err, ok := v.(error); ok {
		entry.Message = err.Error()
	} else {
		entry.Message = v
	}

	out, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("%v\n", entry)
	} else {
		fmt.Println(string(out))
	}
}
