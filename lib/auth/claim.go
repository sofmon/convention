package auth

type Claims struct {
	User      User
	Entities  Entities
	Tenants   Tenants
	Roles     Roles
	Additions map[string]any
}

const (
	claimUser     = "user"
	claimEntities = "entities"
	claimTenants  = "tenants"
	claimRoles    = "roles"
)
