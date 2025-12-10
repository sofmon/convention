package auth

type Claims struct {
	User      User
	Entities  RolesPerEntity
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
