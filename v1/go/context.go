package convention

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"time"
)

type Context struct {
	context.Context
}

type contextKey int

const (
	contextKeyService contextKey = iota
	contextKeyEnvironment
	contextKeyRequest
	contextKeyRequestID
	contextKeyRequestClaims
	contextKeyScope
)

type Service string

func NewContext(svc string) (ctx Context) {
	return WrapContext(context.Background(), svc)
}

func WrapContext(parent context.Context, service string) (ctx Context) {
	return wrapWithEnv(
		context.WithValue(
			parent,
			contextKeyService,
			Service(service),
		),
	)
}

func wrapWithEnv(parent context.Context) (ctx Context) {

	var env Environment

	envStr, err := ConfigString("environment")
	if err != nil {
		// failed to get environment from config, assuming 'production'
		env = EnvironmentProduction
	} else {
		env = Environment(envStr)
	}

	ctx = Context{
		Context: context.WithValue(
			parent,
			contextKeyEnvironment,
			env,
		),
	}

	return
}

func (ctx Context) Service() Service {
	svc, _ := ctx.Value(contextKeyService).(Service)
	return svc
}

/*
	Environment
*/

func (ctx Context) Environment() Environment {
	svc, _ := ctx.Value(contextKeyEnvironment).(Environment)
	return svc
}

func (ctx Context) IsProdEnv() bool {
	return ctx.Environment() == EnvironmentProduction
}

/*
	HTTP
*/

func (ctx Context) WithRequest(r *http.Request) (res Context) {

	res = Context{
		context.WithValue(
			ctx.Context,
			contextKeyRequest,
			r,
		),
	}

	if rid := r.Header.Get(httpHeaderRequestID); rid != "" {
		res = Context{
			context.WithValue(
				res.Context,
				contextKeyRequestID,
				RequestID(rid),
			),
		}
	}

	if claims, err := DecodeHTTPRequestClaims(r); err != nil {
		res = Context{
			context.WithValue(
				res.Context,
				contextKeyRequestClaims,
				claims,
			),
		}
	}

	return
}

func (ctx Context) Request() (r *http.Request) {
	obj := ctx.Value(contextKeyRequest)
	if obj == nil {
		return
	}
	return obj.(*http.Request)
}

func (ctx Context) RequestID() (rid RequestID) {
	rid, _ = ctx.Value(contextKeyRequest).(RequestID)
	return
}

func (ctx Context) RequestClaims() (claims Claims) {
	obj := ctx.Value(contextKeyRequestClaims)
	if obj == nil {
		return
	}
	return obj.(Claims)
}

func (ctx Context) HandleFunc(f func(c Context, w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := ctx.WithRequest(r)

		rec := httptest.NewRecorder()

		authHeader := r.Header.Get(httpHeaderAuthorization)
		if authHeader != "" {
			l := len(authHeader) - 10
			if l < 0 {
				l = len(authHeader)
			}
			r.Header.Set(httpHeaderAuthorization, "..."+authHeader[l:])
		}

		hasBody := r.Method == http.MethodPost || r.Method == http.MethodPut

		reqDump, err := httputil.DumpRequest(r, hasBody)
		if err != nil {
			ctx.LogWarn(err)
			return
		}

		if authHeader != "" {
			r.Header.Set("Authorization", authHeader)
		}

		f(ctx, rec, r)

		res := rec.Result()

		hasBody = res.StatusCode != http.StatusNoContent

		resDump, err := httputil.DumpResponse(res, hasBody)
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

		if hasBody {
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

		trace(
			traceEntry{
				Time:       ctx.Now(),
				Service:    ctx.Service(),
				User:       ctx.User(),
				RequestID:  ctx.RequestID(),
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: res.StatusCode,
				Request:    string(reqDump),
				Response:   string(resDump),
			},
		)
	}
}

/*
	Scope
*/

func (ctx Context) WithScope(scope string) Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyScope,
			ctx.Scope()+" → "+scope,
		),
	}
}

func (ctx Context) WithScopef(format string, a ...any) Context {
	return ctx.WithScope(fmt.Sprintf(format, a...))
}

func (ctx Context) Scope() string {
	scope, _ := ctx.Value(contextKeyScope).(string)
	return scope
}

func (ctx Context) wrapErr(err error) error {

	if err == nil {
		return nil
	}

	prefix := "✘ " + ctx.Scope()

	if strings.HasPrefix(err.Error(), prefix) {
		// no need to wrap the error as it already has the scope prefix
		// it is most probably a wrap call from parent function
		return err
	}

	return fmt.Errorf("%s: %w", prefix, err)
}

// Indicate the current context exits and wrapped eventual error with the current scope
func (ctx Context) Exit(errPtr *error) {
	if errPtr == nil || *errPtr == nil {
		return
	}
	*errPtr = ctx.wrapErr(*errPtr)
}

/*
	Time
*/

func (ctx Context) Now() time.Time {

	if !ctx.IsProdEnv() {

		r := ctx.Request()

		if r != nil {

			nowStr := r.Header.Get(HTTPHeaderWithNowTimeAs)

			if nowStr != "" {

				now, err := time.Parse(time.RFC3339, nowStr)
				if err != nil {
					ctx.LogWarnf("failed to parse %s header: %s", HTTPHeaderWithNowTimeAs, err.Error())
					return time.Now().UTC()
				}

				return now.UTC()
			}

		}

	}

	return time.Now().UTC()
}

/*
	User
*/

func (ctx Context) User() string {
	claims := ctx.RequestClaims()
	if claims == nil {
		return ""
	}
	return claims.User()
}

func (ctx Context) IsAdmin() bool {
	claims := ctx.RequestClaims()
	if claims == nil {
		return false
	}
	return claims.IsAdmin()
}

func (ctx Context) IsService() bool {
	claims := ctx.RequestClaims()
	if claims == nil {
		return false
	}
	return claims.IsService()
}

func (ctx Context) IsAdminOrService() bool {
	return ctx.IsAdmin() || ctx.IsService()
}

/*
	Log
*/

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

func (ctx Context) logMsg(level logLevel, v any) {
	entry := logEntry{
		Time:    ctx.Now(),
		Level:   level,
		Service: ctx.Service(),
		User:    ctx.User(),
		Message: v,
	}
	if r := ctx.Request(); r != nil {
		entry.Metadata = metadata{
			"request_path":   r.URL.Path,
			"request_method": r.Method,
			"request_id":     ctx.RequestID(),
		}
	}
	log(entry)
}
