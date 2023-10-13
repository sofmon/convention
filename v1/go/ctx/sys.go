package ctx

import "context"

func (ctx Context) WithSystemAccount() Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeySystemAccount,
			true,
		),
	}
}

func (ctx Context) MustUseSystemAccount() bool {
	obj := ctx.Value(contextKeySystemAccount)
	if obj == nil {
		return false
	}
	return obj.(bool)
}
