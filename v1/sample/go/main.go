package main

import (
	"net/http"

	conv "github.com/sofmon/convention/v1/go"
)

func main() {

	ctx := conv.NewContext("sample-v1")

	// Ensure the postgresql docker container is running as described in the README.md file:
	// $ docker-compose up -d postgresql
	err := conv.DBOpen("v1")
	if err != nil {
		ctx.LogError(err)
		return
	}
	defer conv.DBClose()

	err = conv.DBRegisterObject[User](true)
	if err != nil {
		ctx.LogError(err)
		return
	}

	http.HandleFunc("/",
		ctx.HandleFunc(
			func(ctx conv.Context, w http.ResponseWriter, r *http.Request) {

				// ctx is now populated with request information

				return
			},
		),
	)

	err = http.ListenAndServeTLS(":443",
		conv.ConfigFilePath("communication_certificate"),
		conv.ConfigFilePath("communication_key"),
		nil,
	)
	if err != nil {
		ctx.LogError(err)
		return
	}

}

var (
	// Endpoint to the v1/sample/go/communication/endpoint.go
	endpoint = conv.NewEndpoint[any, any]("https://localhost/sample/v1/")
)
