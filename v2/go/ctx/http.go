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
				contextKeyClaims,
				claims,
			),
		}
	}

	return
}

func (ctx Context) Workflow() Workflow {
	obj := ctx.Value(contextKeyWorkflow)
	if obj == nil {
		return ""
	}
	return obj.(Workflow)
}

func (ctx Context) Action() convAuth.Action {
	obj := ctx.Value(contextKeyAction)
	if obj == nil {
		return ""
	}
	return obj.(convAuth.Action)
}

func (ctx Context) Request() (r *http.Request) {
	obj := ctx.Value(contextKeyRequest)
	if obj == nil {
		return
	}
	return obj.(*http.Request)
}
