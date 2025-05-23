package api

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"

	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCfg "github.com/sofmon/convention/v2/go/cfg"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type server struct {
	httpServer *http.Server
}

func NewServer(ctx convCtx.Context, host string, port int, cfg convAuth.Config, svc any) (srv *server, err error) {

	if port == 0 {
		port = 443
	}

	check, err := convAuth.NewCheck(cfg)
	if err != nil {
		return
	}

	srv = &server{
		&http.Server{
			Addr:    fmt.Sprintf("%s:%d", host, port),
			Handler: NewHandler(ctx, host, port, check, svc),
		},
	}

	return
}

func (srv *server) ListenAndServe() (err error) {
	return srv.httpServer.ListenAndServeTLS(
		convCfg.FilePath("communication_certificate"), // following convention/v1
		convCfg.FilePath("communication_key"),         // following convention/v1
	)
}

func (srv *server) Shutdown(ctx convCtx.Context) (err error) {
	return srv.httpServer.Shutdown(ctx)
}

func NewHandler(ctx convCtx.Context, host string, port int, check convAuth.Check, svc any) http.Handler {
	return httpHandler{ctx, computeEndpoints(host, port, svc), check}
}

func computeEndpoints(host string, port int, api any) (eps endpoints) {

	for _, f := range reflect.VisibleFields(reflect.TypeOf(api).Elem()) {

		ep, ok := reflect.ValueOf(api).Elem().FieldByName(f.Name).Addr().Interface().(endpoint)
		if !ok {
			continue
		}

		apiTag := f.Tag.Get("api")
		in, out := ep.getInOutTypes()
		desc := newDescriptor(host, port, apiTag, in, out)
		ep.setDescriptor(desc)

		eps = append(eps, ep)
	}

	sort.Slice(
		eps,
		func(i, j int) bool {
			return eps[i].getDescriptor().weight > eps[j].getDescriptor().weight
		},
	)

	for _, ep := range eps {
		ep.setEndpoints(eps)
	}

	return
}

type httpHandler struct {
	ctx   convCtx.Context
	eps   endpoints
	check convAuth.Check
}

func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx.
		WithRequest(r)

	err := h.check(r)
	if err != nil {
		switch err {
		case convAuth.ErrMissingRequest:
			ServeError(ctx, w, http.StatusBadRequest, ErrorCodeBadRequest, "missing http request", err)
			return
		case convAuth.ErrForbidden,
			convAuth.ErrMissingAuthorizationHeader,
			convAuth.ErrInvalidAuthorizationToken:
			ServeError(ctx, w, http.StatusForbidden, ErrorCodeForbidden, "missing or wrong authentication token", err)
			return
		default:
			ServeError(ctx, w, http.StatusUnauthorized, ErrorCodeUnauthorized, "unexpected error", err)
			return
		}
	}

	ctx.LogTrace(w, r, func(w http.ResponseWriter, r *http.Request) {
		ok := false
		for _, ep := range h.eps {
			ok = ep.execIfMatch(ctx, w, r)
			if ok {
				break
			}
		}
		if !ok {
			ServeError(ctx, w, http.StatusNotFound, ErrorCodeNotFound, "Endpoint not found", nil)
		}
	})
}
