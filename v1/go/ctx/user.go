package ctx

func (ctx Context) User() string {
	claims := ctx.RequestClaims()
	if claims == nil {
		return ""
	}
	return claims.User()
}

func (ctx Context) UserIsAdmin() bool {
	claims := ctx.RequestClaims()
	if claims == nil {
		return false
	}
	return claims.IsAdmin()
}

func (ctx Context) UserIsSystem() bool {
	claims := ctx.RequestClaims()
	if claims == nil {
		return false
	}
	return claims.IsSystem()
}

func (ctx Context) UserIsAdminOrSystem() bool {
	return ctx.UserIsAdmin() || ctx.UserIsSystem()
}
