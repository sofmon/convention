package ctx

import (
	"github.com/sofmon/convention/v2/go/auth"
)

func (ctx Context) User() auth.User {
	if ctx.MustUseAgentUser() {
		return auth.User(ctx.Agent())
	}
	return ctx.RequestClaims().User
}
