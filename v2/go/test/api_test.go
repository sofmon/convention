package test

import (
	"testing"
	"time"

	api "github.com/sofmon/convention/v2/go/api"
	auth "github.com/sofmon/convention/v2/go/auth"
	cfg "github.com/sofmon/convention/v2/go/cfg"
	ctx "github.com/sofmon/convention/v2/go/ctx"
)

func Test_server_and_client(t *testing.T) {

	type MessageID string

	type Message struct {
		MessageID MessageID `json:"message_id"`
	}

	type API struct {
		GetOpenAPI api.OpenAPI `api:"GET /openapi.yaml"`

		PutUserMessage api.InP3[Message, auth.Tenant, auth.Entity, MessageID]  `api:"PUT /message/v1/tenants/{tenant}/entity/{entity}/message/{message_id}"`
		GetUserMessage api.OutP3[Message, auth.Tenant, auth.Entity, MessageID] `api:"GET /message/v1/tenants/{tenant}/entity/{entity}/message/{message_id}"`
	}

	cfg.SetConfigLocation("./.secret")

	const (
		roleUser  auth.Role = "user"
		roleAgent auth.Role = "agent"

		permissionReadOwnMessage  auth.Permission = "read_own_messages"
		permissionWriteOwnMessage auth.Permission = "write_own_messages"

		permissionReadAnyMessage  auth.Permission = "read_any_messages"
		permissionWriteAnyMessage auth.Permission = "write_any_messages"
	)

	authCfg := auth.Config{
		Roles: auth.RolePermissions{
			roleUser: auth.Permissions{
				permissionReadOwnMessage,
				permissionWriteOwnMessage,
			},
			roleAgent: auth.Permissions{
				permissionReadAnyMessage,
				permissionWriteAnyMessage,
			},
		},
		Permissions: auth.PermissionActions{
			permissionReadOwnMessage: auth.Actions{
				"GET /message/v1/tenant/{tenant}/entity/{entity}/message/{any}",
			},
			permissionWriteOwnMessage: auth.Actions{
				"PUT /message/v1/tenant/{tenant}/entity/{entity}/message/{any}",
			},
			permissionReadAnyMessage: auth.Actions{
				"GET /message/v1/tenant/{tenant}/entity/{entity}/message/{any}",
			},
			permissionWriteAnyMessage: auth.Actions{
				"PUT /message/v1/tenant/{tenant}/entity/{entity}/message/{any}",
			},
		},
	}

	agentAPI := &API{
		PutUserMessage: api.NewInP3(
			func(ctx ctx.Context, ten auth.Tenant, ent auth.Entity, mid MessageID, msg Message) (err error) {
				return
			},
		),
		GetUserMessage: api.NewOutP3(
			func(ctx ctx.Context, ten auth.Tenant, ent auth.Entity, mid MessageID) (msg Message, err error) {
				return
			},
		),
	}

	agentCtx := ctx.New("message-v1").
		WithRoles(roleAgent)

	go func() {
		err := api.ListenAndServe(agentCtx, "localhost", 12345, agentAPI, authCfg)
		if err != nil {
			t.Errorf("ListenAndServe() = %v; want nil", err)
		}
	}()

	time.Sleep(10 * time.Millisecond) // give time for the agent api to start

	// Create new client based on the same API specification
	// client := api.NewClient[API]("localhost", 12345)

	// clientCtx := ctx.New("user").
	// 	WithRoles(roleUser)

	//client.GetUserMessage.Call(clientCtx, "t1", "user", "e1", "m1")

	// r, _ := http.Get("https://localhost:12345/openapi.yaml")
	// b, _ := io.ReadAll(r.Body)
	// fmt.Println("res", string(b))

	// res, err := client.GetHealth.Call(ctx)
	// if err != nil {
	// 	t.Errorf("GetHealth.Call() = %v; want nil", err)
	// }
	// if res != "OK" {
	// 	t.Errorf("GetHealth.Call() = %v; want %v", res, "OK")
	// }

	// err = client.PutUser.Call(
	// 	ctx,
	// 	"123",
	// 	User{
	// 		UserID: "123",
	// 		Name:   "John Doe",
	// 	},
	// )
	// if err != nil {
	// 	t.Errorf("PutUser.Call() = %v; want nil", err)
	// }

	// u, err := client.GetUser.Call(ctx, "123")
	// if err != nil {
	// 	t.Errorf("GetUser.Call() = %v; want nil", err)
	// }
	// if u.UserID != "123" {
	// 	t.Errorf("GetUser.Call() = %v; want %v", u.UserID, "123")
	// }

}
