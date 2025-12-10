package auth

import (
	"fmt"
	"strings"
)

type User string

type Users []User

type Entity string

type Entities []Entity

type RolesPerEntity map[Entity]Roles

type Tenant string

type Tenants []Tenant

type Role string

type Roles []Role

type Permission string

type Permissions []Permission

type Action string

func (a Action) MethodPath() (method, resource string, err error) {
	methodPath := strings.SplitN(string(a), " ", 2)
	if len(methodPath) != 2 {
		err = fmt.Errorf("invalid action: %s", a)
		return
	}

	method = methodPath[0]
	resource = strings.Trim(methodPath[1], "/")

	return
}

type Actions []Action

type RolePermissions map[Role]Permissions

type PermissionActions map[Permission]Actions

type Policy struct {
	Roles       RolePermissions   `json:"roles"`
	Permissions PermissionActions `json:"permissions"`
	Public      Actions           `json:"public"`
}
