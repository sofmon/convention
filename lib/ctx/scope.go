package ctx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (ctx Context) WithScope(scope string, args ...any) Context {

	scope = ctx.Scope() + " → " + scope

	if len(args) > 0 {
		scope += " {" + formatArgs(args...) + "}"
	}

	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyScope,
			scope,
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
// unless the error is nil or is in the except list
func (ctx Context) Exit(errPtr *error, except ...error) {
	if errPtr == nil || *errPtr == nil {
		return
	}
	for _, ex := range except {
		if errors.Is(*errPtr, ex) {
			return
		}
	}
	*errPtr = ctx.wrapErr(*errPtr)
	ctx.Logger().Debug("exiting scope", "error", (*errPtr).Error())
}

func formatArgs(args ...any) string {
	type kv struct {
		key   string
		value any
	}

	const (
		badKey = "!BAD_KEY!"
	)

	var (
		kvs []kv
	)

	for len(args) > 0 {
		switch k := args[0].(type) {
		case string:
			// string is a key
			if len(args) == 1 {
				// no value, record as special pair
				kvs = append(kvs, kv{badKey, k})
				args = args[1:]
				continue
			}
			kvs = append(kvs, kv{k, args[1]})
			args = args[2:]
		default:
			// no key, record as special pair
			kvs = append(kvs, kv{badKey, k})
			args = args[1:]
		}
	}

	var buf bytes.Buffer
	for i, p := range kvs {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(escapeKey(p.key))
		buf.WriteByte('=')
		buf.WriteString(formatValue(p.value))
	}

	return buf.String()
}

func escapeKey(key string) string {
	// keep it simple, if key has bad chars, quote it
	if key == "" || strings.ContainsAny(key, " \t\r\n=") {
		return strconv.Quote(key)
	}
	return key
}

func formatValue(v any) string {
	if v == nil {
		return "null"
	}
	s := fmt.Sprint(v)

	// no spaces, no quotes, no equals, print bare
	if s != "" && !strings.ContainsAny(s, " \t\r\n\"=") {
		return s
	}
	return strconv.Quote(s)
}
