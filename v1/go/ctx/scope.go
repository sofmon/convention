package ctx

import (
	"context"
	"fmt"
	"strings"
)

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
