package ctx

import "context"

func (ctx Context) WithAgentUser() Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyAgentUser,
			true,
		),
	}
}

func (ctx Context) MustUseAgentUser() bool {
	obj := ctx.Value(contextKeyAgentUser)
	if obj == nil {
		return false
	}
	return obj.(bool)
}
