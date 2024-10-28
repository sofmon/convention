package ctx

import (
	"context"
	"fmt"
	"net/http"

	convAuth "github.com/sofmon/convention/v2/go/auth"
)

type Workflow string

const (
	HttpHeaderAuthorization = convAuth.HttpHeaderAuthorization
	HttpHeaderWorkflow      = "Workflow"
	HTTPHeaderTimeNow       = "Time-Now"
)

func (ctx Context) WithRequest(r *http.Request) (res Context) {

	res = Context{
		context.WithValue(
			ctx.Context,
			contextKeyRequest,
			r,
		),
	}

	if wid := r.Header.Get(HttpHeaderWorkflow); wid != "" {
		res = Context{
			context.WithValue(
				res.Context,
				contextKeyWorkflow,
				Workflow(wid),
			),
		}
	}

	res = Context{
		context.WithValue(
			res.Context,
			contextKeyAction,
			fmt.Sprintf("%s %s", r.Method, r.URL.Path),
		),
	}

	if claims, err := convAuth.DecodeHTTPRequestClaims(r); err == nil {
		res = Context{
			context.WithValue(
				res.Context,
				contextKeyRequestClaims,
				claims,
			),
		}
	}

	return
}

func (ctx Context) Workflow() (wid Workflow) {
	return ctx.Value(contextKeyRequest).(Workflow)
}

func (ctx Context) Action() (act convAuth.Action) {
	return ctx.Value(contextKeyAction).(convAuth.Action)
}

func (ctx Context) Request() (r *http.Request) {
	obj := ctx.Value(contextKeyRequest)
	if obj == nil {
		return
	}
	return obj.(*http.Request)
}

func (ctx Context) RequestClaims() (claims convAuth.Claims) {
	obj := ctx.Value(contextKeyRequestClaims)
	if obj == nil {
		return
	}
	return obj.(convAuth.Claims)
}
