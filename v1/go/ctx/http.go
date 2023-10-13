package ctx

import (
	"context"
	"net/http"

	convAuth "github.com/sofmon/convention/v1/go/auth"
)

type RequestID string

const (
	HttpHeaderAuthorization = convAuth.HttpHeaderAuthorization
	HttpHeaderRequestID     = "Request-Id"
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

	if rid := r.Header.Get(HttpHeaderRequestID); rid != "" {
		res = Context{
			context.WithValue(
				res.Context,
				contextKeyRequestID,
				RequestID(rid),
			),
		}
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

func (ctx Context) Request() (r *http.Request) {
	obj := ctx.Value(contextKeyRequest)
	if obj == nil {
		return
	}
	return obj.(*http.Request)
}

func (ctx Context) RequestID() (rid RequestID) {
	rid, _ = ctx.Value(contextKeyRequest).(RequestID)
	return
}

func (ctx Context) RequestClaims() (claims convAuth.Claims) {
	obj := ctx.Value(contextKeyRequestClaims)
	if obj == nil {
		return
	}
	return obj.(convAuth.Claims)
}
