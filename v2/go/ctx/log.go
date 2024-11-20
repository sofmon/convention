package ctx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"time"

	"github.com/sofmon/convention/v2/go/auth"
	convAuth "github.com/sofmon/convention/v2/go/auth"
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

func (ctx Context) LogTrace(w http.ResponseWriter, r *http.Request, handle func(w http.ResponseWriter, r *http.Request)) {

	rec := httptest.NewRecorder()

	// temporary hide the Authorization header while we dump the request
	authHeader := r.Header.Get(convAuth.HttpHeaderAuthorization)
	if authHeader != "" {
		l := len(authHeader) - 10
		if l < 0 {
			l = len(authHeader)
		}
		r.Header.Set(convAuth.HttpHeaderAuthorization, "..."+authHeader[l:])
	}

	contentHeader := r.Header.Get("Content-Type")
	logBody := contentHeader == "application/json" ||
		contentHeader == "application/xml" ||
		contentHeader == "application/yaml" ||
		contentHeader == "text/plain" ||
		contentHeader == "text/html"

	reqDump, err := httputil.DumpRequest(r, logBody)
	if err != nil {
		ctx.LogWarn(err)
		return
	}

	// restore the Authorization header
	if authHeader != "" {
		r.Header.Set(convAuth.HttpHeaderAuthorization, authHeader)
	}

	handle(rec, r)

	res := rec.Result()

	contentHeader = res.Header.Get("Content-Type")
	logBody = contentHeader == "application/json" ||
		contentHeader == "application/xml" ||
		contentHeader == "application/yaml" ||
		contentHeader == "text/plain" ||
		contentHeader == "text/html"

	resDump, err := httputil.DumpResponse(res, logBody)
	if err != nil {
		ctx.LogWarn(err)
		return
	}

	for k, v := range res.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	w.WriteHeader(res.StatusCode)

	if res.Body != nil {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			ctx.LogWarn(err)
			return
		}
		_, err = w.Write(resBody)
		if err != nil {
			ctx.LogWarn(err)
			return
		}
	}

	entry := logEntry{
		Time:     ctx.Now(),
		Level:    logLevelTrace,
		Agent:    ctx.Agent(),
		User:     ctx.User(),
		Workflow: ctx.Workflow(),
		Action:   auth.Action(fmt.Sprintf("%s %s", r.Method, r.URL.Path)),
		Request:  string(reqDump),
		Response: string(resDump),
	}

	out, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("%v\n", entry)
	} else {
		fmt.Println(string(out))
	}
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
	Request  string      `json:"request,omitempty"`
	Response string      `json:"response,omitempty"`
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
