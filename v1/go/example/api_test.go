package example

import (
	"net/http"
	"testing"
	"time"

	convAPI "github.com/sofmon/convention/v1/go/api"
	convCfg "github.com/sofmon/convention/v1/go/cfg"
	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

type UserID string

type User struct {
	UserID UserID `json:"user_id"`
	Name   string `json:"name"`
}

// The API specification
type API struct {
	GetHealth convAPI.Out[string] `api:"GET /health"`

	PutUser convAPI.InP1[User, UserID]  `api:"PUT /users/{user_id}" description:"Create new user"`
	GetUser convAPI.OutP1[User, UserID] `api:"GET /users/{user_id}" description:"Get user"`

	PostUserDisable convAPI.InOutP1[User, User, UserID] `api:"POST /users/{user_id}/@block" description:"block user"`
}

func Test_server_and_client(t *testing.T) {

	convCfg.SetConfigLocation("./etc")

	ctx := convCtx.NewContext("Test_server_and_client")

	userDB := make(map[UserID]User)

	// Create a service based on the API specification
	service := &API{
		GetHealth: convAPI.NewOut(
			func(ctx convCtx.Context) (string, error) {
				return "OK", nil
			},
		),
		PutUser: convAPI.NewInP1(
			func(ctx convCtx.Context, uid UserID, user User) error {
				if user.UserID != uid {
					return convAPI.NewError(http.StatusBadRequest, "bad_request", "mismatch between user_id in path and body")
				}
				userDB[user.UserID] = user
				return nil
			},
		),
		GetUser: convAPI.NewOutP1(
			func(ctx convCtx.Context, uid UserID) (User, error) {
				user, ok := userDB[uid]
				if !ok {
					return User{}, convAPI.NewError(http.StatusNotFound, "not_found", "user not found")
				}
				return user, nil
			},
		),
	}

	go func() {
		err := convAPI.ListenAndServe(ctx, "localhost", 12345, service)
		if err != nil {
			t.Errorf("ListenAndServe() = %v; want nil", err)
		}
	}()

	time.Sleep(10 * time.Millisecond) // give time for server to start

	// Create new client based on the same API specification
	client := convAPI.NewClient[API]("localhost", 12345)

	res, err := client.GetHealth.Call(ctx)
	if err != nil {
		t.Errorf("GetHealth.Call() = %v; want nil", err)
	}
	if res != "OK" {
		t.Errorf("GetHealth.Call() = %v; want %v", res, "OK")
	}

	err = client.PutUser.Call(
		ctx,
		"123",
		User{
			UserID: "123",
			Name:   "John Doe",
		},
	)
	if err != nil {
		t.Errorf("PutUser.Call() = %v; want nil", err)
	}

	u, err := client.GetUser.Call(ctx, "123")
	if err != nil {
		t.Errorf("GetUser.Call() = %v; want nil", err)
	}
	if u.UserID != "123" {
		t.Errorf("GetUser.Call() = %v; want %v", u.UserID, "123")
	}

}
