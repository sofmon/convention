package ctx

import (
	"github.com/sofmon/convention/v2/go/auth"
)

func (ctx Context) User() auth.User {
	return ctx.Claims().User
}
