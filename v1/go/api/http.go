package api

import (
	"encoding/json"
	"net/http"

	convAuth "github.com/sofmon/convention/v1/go/auth"
	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

const (
	httpHeaderAuthorization = "Authorization"
	httpHeaderRequestID     = "Request-ID"
	httpHeaderApp           = "App"
)

func ServeJSON(w http.ResponseWriter, body any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(body)
}

func ReceiveJSON[T any](r *http.Request) (res T, err error) {
	err = json.NewDecoder(r.Body).Decode(&res)
	return
}

func setContextHttpHeaders(ctx convCtx.Context, r *http.Request) {

	inReq := ctx.Request()

	r.Header.Add(httpHeaderRequestID, string(ctx.RequestID()))
	r.Header.Add(httpHeaderApp, string(ctx.App()))

	if inReq == nil {
		systemClaim := convAuth.NewClaims(string(ctx.App()), false, true)
		convAuth.EncodeHTTPRequestClaims(r, systemClaim)
		return
	} else {
		authHeader := inReq.Header.Get(httpHeaderAuthorization)
		if authHeader != "" {
			r.Header.Add(httpHeaderAuthorization, authHeader)
		}
	}

}
