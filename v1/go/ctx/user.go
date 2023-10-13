package ctx

func (ctx Context) User() string {
	claims := ctx.RequestClaims()
	if claims == nil {
		return ""
	}
	return claims.User()
}

func (ctx Context) IsAdmin() bool {
	claims := ctx.RequestClaims()
	if claims == nil {
		return false
	}
	return claims.IsAdmin()
}

func (ctx Context) IsSystem() bool {
	claims := ctx.RequestClaims()
	if claims == nil {
		return false
	}
	return claims.IsSystem()
}

func (ctx Context) IsAdminOrSystem() bool {
	return ctx.IsAdmin() || ctx.IsSystem()
}
