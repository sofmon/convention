package auth_test

import (
	"net/http"
	"net/url"
	"testing"

	convAuth "github.com/sofmon/convention/lib/auth"
)

var (
	fullPolicy = convAuth.Policy{
		Roles: convAuth.RolePermissions{
			"manage_own_assets": convAuth.Permissions{
				"read_own_assets",
				"write_own_assets",
			},
			"manage_all_assets": convAuth.Permissions{
				"read_all_assets",
				"write_all_assets",
			},
			"manage_all_assets_all_tenants": convAuth.Permissions{
				"read_all_assets_all_tenants",
				"write_all_assets_all_tenants",
			},
			"access_anything": convAuth.Permissions{
				"access_anything",
			},
		},
		Permissions: convAuth.PermissionActions{
			"read_own_assets": convAuth.Actions{
				"GET /tenants/{tenant}/users/{user}/assets",
				"GET /tenants/{tenant}/users/{user}/assets/{any}",
				"GET /tenants/{tenant}/users/{user}/open/{any...}",
			},
			"write_own_assets": convAuth.Actions{
				"PUT /tenants/{tenant}/users/{user}/assets/{any}",
				"PUT /tenants/{tenant}/users/{user}/open/{any...}",
			},
			"read_all_assets": convAuth.Actions{
				"GET /tenants/{tenant}/users/{any}/assets",
				"GET /tenants/{tenant}/users/{any}/assets/{any}",
				"GET /tenants/{tenant}/users/{any}/open/{any...}",
			},
			"write_all_assets": convAuth.Actions{
				"PUT /tenants/{tenant}/users/{any}/assets/{any}",
				"PUT /tenants/{tenant}/users/{any}/open/{any...}",
			},
			"read_all_assets_all_tenants": convAuth.Actions{
				"GET /tenants/{any}/users/{any}/assets",
				"GET /tenants/{any}/users/{any}/assets/{any}",
				"GET /tenants/{any}/users/{any}/open/{any...}",
			},
			"write_all_assets_all_tenants": convAuth.Actions{
				"PUT /tenants/{any}/users/{any}/assets/{any}",
				"PUT /tenants/{any}/users/{any}/open/{any...}",
			},
			"access_anything": convAuth.Actions{
				"GET /{any...}",
			},
		},
		Public: convAuth.Actions{
			"GET /public/{any...}",
		},
	}

	entityRolePolicy = convAuth.Policy{
		Roles: convAuth.RolePermissions{
			"basic_user":   convAuth.Permissions{"read_entity_data"},
			"entity_admin": convAuth.Permissions{"read_entity_data", "write_entity_data"},
		},
		Permissions: convAuth.PermissionActions{
			"read_entity_data":  convAuth.Actions{"GET /entities/{entity}/data", "GET /entities/{entity}/data/{any}"},
			"write_entity_data": convAuth.Actions{"PUT /entities/{entity}/data/{any}", "DELETE /entities/{entity}/data/{any}"},
		},
	}

	testData = []struct {
		name     string                  // test case name
		policy   convAuth.Policy         // access control configuration
		user     convAuth.User           // authenticated user
		tenants  convAuth.Tenants        // authenticated user assigned tenants
		roles    convAuth.Roles          // authenticated user assigned roles
		entities convAuth.RolesPerEntity // authenticated user assigned entities with their roles
		pass     []*http.Request         // requests that should pass the access check
		block    []*http.Request         // requests that should be blocked by the access check
	}{
		{
			name:    "test access assets based on assigned user and tenant",
			policy:  fullPolicy,
			user:    "user1",
			tenants: convAuth.Tenants{"tenant1"},
			roles:   convAuth.Roles{"manage_own_assets"},
			pass: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets"}},
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user1/open/something/else/that/is/whatever"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant1/users/user1/open/something/else/that/is/whatever"}},
			},
			block: []*http.Request{
				// disallowed methods
				{Method: "POST", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},
				{Method: "DELETE", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},

				// disallowed paths
				{Method: "GET", URL: &url.URL{Path: "/asdf/tenant1/users/user1/11/asset1"}},
				{Method: "PUT", URL: &url.URL{Path: "/asdf/tenant1/users/user1/11/asset1"}},

				// disallowed tenants
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant2/users/user1/assets/asset1"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant2/users/user1/assets/asset1"}},

				// disallowed users
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user2/assets/asset1"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant1/users/user2/assets/asset1"}},

				// random paths
				{Method: "GET", URL: &url.URL{Path: "/something/else/that/is/whatever/"}},
				{Method: "PUT", URL: &url.URL{Path: "/something/else/that/is/whatever/"}},
			},
		},
		{
			name:    "test access assets based for all users in assigned tenant",
			policy:  fullPolicy,
			user:    "user1",
			tenants: convAuth.Tenants{"tenant1"},
			roles:   convAuth.Roles{"manage_all_assets"},
			pass: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets"}},
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user1/open/something/else/that/is/whatever"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant1/users/user1/open/something/else/that/is/whatever"}},

				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user2/assets/asset1"}},
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user2/open/something/else/that/is/whatever"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant1/users/user2/assets/asset1"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant1/users/user2/open/something/else/that/is/whatever"}},
			},
			block: []*http.Request{
				// disallowed methods
				{Method: "POST", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},
				{Method: "DELETE", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},

				// disallowed tenants
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant2/users/user1/assets/asset1"}},
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant2/users/user1/open/something/else/that/is/whatever"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant2/users/user1/assets/asset1"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant2/users/user1/open/something/else/that/is/whatever"}},

				// random paths
				{Method: "GET", URL: &url.URL{Path: "/something/else/that/is/whatever/"}},
				{Method: "PUT", URL: &url.URL{Path: "/something/else/that/is/whatever/"}},
			},
		},
		{
			name:    "test access assets based for all users and all tenant",
			policy:  fullPolicy,
			user:    "user1",
			tenants: convAuth.Tenants{"tenant1"},
			roles:   convAuth.Roles{"manage_all_assets_all_tenants"},
			pass: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets"}},
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},

				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user2/assets/asset1"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant1/users/user2/assets/asset1"}},

				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant2/users/user1/assets/asset1"}},
				{Method: "PUT", URL: &url.URL{Path: "/tenants/tenant2/users/user1/assets/asset1"}},
			},
			block: []*http.Request{
				// disallowed methods
				{Method: "POST", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},
				{Method: "DELETE", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets/asset1"}},
			},
		},
		{
			name:    "test access anything with {any...} as root",
			policy:  fullPolicy,
			user:    "user1",
			tenants: convAuth.Tenants{"tenant1"},
			roles:   convAuth.Roles{"access_anything"},
			pass: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/"}},
				{Method: "GET", URL: &url.URL{Path: "/anything"}},
				{Method: "GET", URL: &url.URL{Path: "/anything/else"}},
				{Method: "GET", URL: &url.URL{Path: "/anything/else/that/"}},
				{Method: "GET", URL: &url.URL{Path: "/anything/else/that/is/"}},
				{Method: "GET", URL: &url.URL{Path: "/anything/else/that/is/whatever"}},
			},
		},
		{
			name:    "test public access",
			policy:  fullPolicy,
			user:    "xxx",
			tenants: convAuth.Tenants{"xxx"},
			roles:   convAuth.Roles{"xxx"},
			pass: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/public/something/else/that/is/whatever"}},
			},
		},
		{
			name:   "test entity-specific roles grant additional permissions",
			policy: entityRolePolicy,
			user:   "user1",
			roles:  convAuth.Roles{"basic_user"},
			entities: convAuth.RolesPerEntity{
				"entity1": convAuth.Roles{"entity_admin"}, // extra roles for entity1
				"entity2": convAuth.Roles{},               // no extra roles for entity2
			},
			pass: []*http.Request{
				// Can read both entities (basic_user base role)
				{Method: "GET", URL: &url.URL{Path: "/entities/entity1/data"}},
				{Method: "GET", URL: &url.URL{Path: "/entities/entity2/data"}},
				// Can write to entity1 (entity_admin from Entities map)
				{Method: "PUT", URL: &url.URL{Path: "/entities/entity1/data/item1"}},
			},
			block: []*http.Request{
				// Cannot write to entity2 (no entity_admin role for entity2)
				{Method: "PUT", URL: &url.URL{Path: "/entities/entity2/data/item1"}},
				// Cannot access entity3 (not in Entities map)
				{Method: "GET", URL: &url.URL{Path: "/entities/entity3/data"}},
			},
		},
		{
			name:   "test entity-specific roles without base roles",
			policy: entityRolePolicy,
			user:   "user1",
			roles:  convAuth.Roles{}, // No base roles
			entities: convAuth.RolesPerEntity{
				"entity1": convAuth.Roles{"entity_admin"},
			},
			pass: []*http.Request{
				// Can access entity1 via entity-specific roles
				{Method: "GET", URL: &url.URL{Path: "/entities/entity1/data"}},
				{Method: "PUT", URL: &url.URL{Path: "/entities/entity1/data/item1"}},
			},
			block: []*http.Request{
				// Cannot access entity2 (not in Entities map)
				{Method: "GET", URL: &url.URL{Path: "/entities/entity2/data"}},
			},
		},
		{
			name:    "test base roles work for non-entity paths",
			policy:  fullPolicy,
			user:    "user1",
			tenants: convAuth.Tenants{"tenant1"},
			roles:   convAuth.Roles{"manage_own_assets"},
			entities: convAuth.RolesPerEntity{
				"entity1": convAuth.Roles{"some_role"},
			},
			pass: []*http.Request{
				// Base roles work for paths without {entity}
				{Method: "GET", URL: &url.URL{Path: "/tenants/tenant1/users/user1/assets"}},
			},
		},
		{
			name: "test specificity ordering - specific path matched before wildcard",
			policy: convAuth.Policy{
				Roles: convAuth.RolePermissions{
					"specific_role": convAuth.Permissions{"specific_action"},
					"wildcard_role": convAuth.Permissions{"wildcard_action"},
				},
				Permissions: convAuth.PermissionActions{
					"specific_action": convAuth.Actions{"GET /path/to/specific"},
					"wildcard_action": convAuth.Actions{"GET /path/to/{any}"},
				},
			},
			user:  "user1",
			roles: convAuth.Roles{"specific_role"}, // Only has specific_role
			pass: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/path/to/specific"}},
			},
			block: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/path/to/other"}}, // No wildcard_role
			},
		},
		{
			name: "test specificity ordering - user path before any path",
			policy: convAuth.Policy{
				Roles: convAuth.RolePermissions{
					"user_role": convAuth.Permissions{"user_action"},
					"any_role":  convAuth.Permissions{"any_action"},
				},
				Permissions: convAuth.PermissionActions{
					"user_action": convAuth.Actions{"GET /users/{user}/data"},
					"any_action":  convAuth.Actions{"GET /users/{any}/data"},
				},
			},
			user:  "user1",
			roles: convAuth.Roles{"user_role"}, // Only has user_role
			pass: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/users/user1/data"}},
			},
			block: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/users/other/data"}}, // No any_role
			},
		},
		{
			name: "test specificity ordering - fixed path before open end",
			policy: convAuth.Policy{
				Roles: convAuth.RolePermissions{
					"fixed_role":   convAuth.Permissions{"fixed_action"},
					"openend_role": convAuth.Permissions{"openend_action"},
				},
				Permissions: convAuth.PermissionActions{
					"fixed_action":   convAuth.Actions{"GET /api/v1/users"},
					"openend_action": convAuth.Actions{"GET /api/{any...}"},
				},
			},
			user:  "user1",
			roles: convAuth.Roles{"fixed_role"}, // Only has fixed_role
			pass: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/api/v1/users"}},
			},
			block: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/api/v1/other"}}, // No openend_role
			},
		},
	}
)

func TestCheck(t *testing.T) {

	for _, td := range testData {

		check, err := convAuth.NewCheck(td.policy)
		if err != nil {
			t.Fatalf("NewCheck failed: %v", err)
		}

		claim := convAuth.Claims{
			User:     td.user,
			Tenants:  td.tenants,
			Roles:    td.roles,
			Entities: td.entities,
		}

		for _, req := range td.pass {
			req.Header = make(http.Header)
			err = convAuth.EncodeHTTPRequestClaims(req, claim)
			if err != nil {
				t.Fatalf("EncodeHTTPRequestClaims failed: %v", err)
			}
			_, err = check(req)
			if err != nil {
				t.Fatalf("%s\n%s %s: endpoint blocked: %v", td.name, req.Method, req.URL.Path, err)
			}
		}

		for _, req := range td.block {
			req.Header = make(http.Header)
			err = convAuth.EncodeHTTPRequestClaims(req, claim)
			if err != nil {
				t.Fatalf("EncodeHTTPRequestClaims failed: %v", err)
			}
			_, err = check(req)
			if err == nil {
				t.Fatalf("%s\n%s %s: endpoint allowed", td.name, req.Method, req.URL.Path)
			}
		}
	}

}
