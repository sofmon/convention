package ctx

import (
	"context"

	"github.com/sofmon/convention/lib/auth"
	convAuth "github.com/sofmon/convention/lib/auth"
)

func (ctx Context) WithClaims(claims convAuth.Claims) Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyClaims,
			claims,
		),
	}.WithLogger(
		ctx.Logger().With(loggerKeyUser, claims.User),
	)
}

func (ctx Context) Claims() (claims convAuth.Claims) {

	if ctx.mustUseAgentClaims() {
		obj := ctx.Value(contextKeyAgentClaims)
		if obj == nil {
			return
		}
		return obj.(convAuth.Claims)
	}

	obj := ctx.Value(contextKeyClaims)
	if obj == nil {
		return
	}
	return obj.(convAuth.Claims)
}

func (ctx Context) User() auth.User {
	return ctx.Claims().User
}
