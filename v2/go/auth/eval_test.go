package auth_test

import (
	"net/http"
	"net/url"
	"testing"

	convAuth "github.com/sofmon/convention/v2/go/auth"
)

var (
	fullConfig = convAuth.Config{
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

	testData = []struct {
		name     string            // test case name
		cfg      convAuth.Config   // access control configuration
		user     convAuth.User     // authenticated user
		tenants  convAuth.Tenants  // authenticated user assigned tenants
		roles    convAuth.Roles    // authenticated user assigned roles
		entities convAuth.Entities // authenticated user assigned entities
		pass     []*http.Request   // requests that should pass the access check
		block    []*http.Request   // requests that should be blocked by the access check
	}{
		{
			name:    "test access assets based on assigned user and tenant",
			cfg:     fullConfig,
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
			cfg:     fullConfig,
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
			cfg:     fullConfig,
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
			cfg:     fullConfig,
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
			cfg:     fullConfig,
			user:    "xxx",
			tenants: convAuth.Tenants{"xxx"},
			roles:   convAuth.Roles{"xxx"},
			pass: []*http.Request{
				{Method: "GET", URL: &url.URL{Path: "/public/something/else/that/is/whatever"}},
			},
		},
	}
)

func TestCheck(t *testing.T) {

	for _, td := range testData {

		check, err := convAuth.NewCheck(td.cfg)
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
			err = check(req)
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
			err = check(req)
			if err == nil {
				t.Fatalf("%s\n%s %s: endpoint allowed", td.name, req.Method, req.URL.Path)
			}
		}
	}

}
