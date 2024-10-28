package api

import (
	"encoding/json"
	"net/http"

	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

const (
	httpHeaderAuthorization = "Authorization"
	httpHeaderAgent         = "Agent"
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

func setContextHttpHeaders(ctx convCtx.Context, r *http.Request) (err error) {

	r.Header.Add(convCtx.HttpHeaderWorkflow, string(ctx.Workflow()))
	r.Header.Add(httpHeaderAgent, string(ctx.Agent()))

	inReq := ctx.Request()

	if inReq == nil || ctx.MustUseAgentUser() {
		agentClaims := convAuth.Claims{
			User:  convAuth.User(ctx.Agent()),
			Roles: ctx.AgentRoles(),
		}
		err = convAuth.EncodeHTTPRequestClaims(r, agentClaims)
		if err != nil {
			return
		}
	} else {
		authHeader := inReq.Header.Get(httpHeaderAuthorization)
		if authHeader != "" {
			r.Header.Add(httpHeaderAuthorization, authHeader)
		}
	}

	return
}
