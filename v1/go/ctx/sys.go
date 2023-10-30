package ctx

import "context"

func (ctx Context) WithSystemUser() Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeySystemUser,
			true,
		),
	}
}

func (ctx Context) MustUseSystemUser() bool {
	obj := ctx.Value(contextKeySystemUser)
	if obj == nil {
		return false
	}
	return obj.(bool)
}
