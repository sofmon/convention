package convention

import "net/http"

type RequestID string

const (
	httpHeaderAuthorization = "Authorization"
	httpHeaderRequestID     = "Request-Id"

	HTTPHeaderWithNowTimeAs = "With-Now-Time-As"
)

type HttpHandler struct {
	ctx Context
}

func (h HttpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	ctx := h.ctx.WithScope("convention.HttpHandler.ServeHTTP")
	defer ctx.Exit(nil)

}
