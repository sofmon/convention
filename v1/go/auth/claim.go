package auth

type Claims map[string]any

const (
	claimUser     = "user"
	claimIsAdmin  = "admin"
	claimIsSystem = "system"
)

func NewClaims(user string, isAdmin, isSystem bool) Claims {
	return Claims{
		claimUser:     user,
		claimIsAdmin:  isAdmin,
		claimIsSystem: isSystem,
	}
}

func (c Claims) User() string {
	userAny, ok := c[claimUser]
	if !ok {
		return ""
	}

	user, ok := userAny.(string)
	if !ok {
		return ""
	}

	return user
}

func (c Claims) IsAdmin() bool {
	adminUserAny, ok := c[claimIsAdmin]
	if !ok {
		return false
	}

	adminUser, ok := adminUserAny.(bool)
	if !ok {
		return false
	}

	return adminUser
}

func (c Claims) IsSystem() bool {
	serviceUserAny, ok := c[claimIsSystem]
	if !ok {
		return false
	}

	serviceUser, ok := serviceUserAny.(bool)
	if !ok {
		return false
	}

	return serviceUser
}

func (c Claims) IsAdminOrSystem() bool {
	return c.IsAdmin() || c.IsSystem()
}
