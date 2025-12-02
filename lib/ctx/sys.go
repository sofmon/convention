package ctx

import "context"

func (ctx Context) WithAgentClaims() Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyUseAgentClaims,
			true,
		),
	}.WithLogger(
		ctx.Logger().With(loggerKeyUseAgentClaims, true),
	)
}

func (ctx Context) mustUseAgentClaims() bool {
	obj := ctx.Value(contextKeyUseAgentClaims)
	if obj == nil {
		return false
	}
	return obj.(bool)
}
