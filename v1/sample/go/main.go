package main

import (
	"net/http"

	convAPI "github.com/sofmon/convention/v1/go/api"
	convCtx "github.com/sofmon/convention/v1/go/ctx"
	convDB "github.com/sofmon/convention/v1/go/db"
)

var (
	users = convDB.NewObjectSet[User]()
)

func main() {

	ctx := convCtx.NewContext("sample-v1")

	// Ensure the postgresql docker container is running as described in the README.md file:
	// $ docker-compose up -d postgresql
	err := convDB.Open("v1")
	if err != nil {
		ctx.LogError(err)
		return
	}
	defer convDB.Close()

	err = convDB.RegisterObject[User](true)
	if err != nil {
		ctx.LogError(err)
		return
	}

	err = convAPI.ListenAndServe(
		ctx,
		convAPI.Endpoints{
			"/users/%s/": func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request, params ...string) {

				// ctx is now populated with request information

				return
			},
		},
	)
	if err != nil {
		ctx.LogError(err)
		return
	}
}

var (
	// Endpoint to the v1/sample/go/communication/endpoint.go
	endpoint = convAPI.NewEndpoint[any, any]("https://localhost/sample/v1/")
)
