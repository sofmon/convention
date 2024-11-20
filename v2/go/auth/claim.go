package auth

type Claims struct {
	User     User
	Entities Entities
	Tenants  Tenants
	Roles    Roles
}

const (
	claimUser     = "user"
	claimEntities = "entities"
	claimTenants  = "tenants"
	claimRoles    = "roles"
)
