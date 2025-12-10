package ctx

import (
	"context"

	convAuth "github.com/sofmon/convention/lib/auth"
)

func (ctx Context) WithAction(action convAuth.Action) Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyAction,
			action,
		),
	}
}

func (ctx Context) Action() convAuth.Action {
	obj := ctx.Value(contextKeyAction)
	if obj == nil {
		return ""
	}
	return obj.(convAuth.Action)
}
