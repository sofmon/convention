package ctx

import "net/http"

type RequestID string

const (
	httpHeaderAuthorization = "Authorization"
	httpHeaderRequestID     = "Request-Id"

	HTTPHeaderTimeNow = "Time-Now"
)

type HttpHandler struct {
	ctx Context
}

func (h HttpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	ctx := h.ctx.WithScope("convention.HttpHandler.ServeHTTP")
	defer ctx.Exit(nil)

}
