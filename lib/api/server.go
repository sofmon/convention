package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"reflect"
	"sort"
	"strings"

	convAuth "github.com/sofmon/convention/lib/auth"
	convCfg "github.com/sofmon/convention/lib/cfg"
	convCtx "github.com/sofmon/convention/lib/ctx"
)

type server struct {
	httpServer *http.Server
}

func NewServer(ctx convCtx.Context, host string, port int, policy convAuth.Policy, svc any) (srv *server, err error) {

	if port == 0 {
		port = 443
	}

	check, err := convAuth.NewCheck(policy)
	if err != nil {
		return
	}

	srv = &server{
		&http.Server{
			Addr:    fmt.Sprintf("%s:%d", host, port),
			Handler: NewHandler(ctx, host, port, check, svc, false),
		},
	}

	return
}

func (srv *server) EnableCallsLogging() {
	h, ok := srv.httpServer.Handler.(*httpHandler)
	if ok {
		h.logCalls = true
	}
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

func NewHandler(ctx convCtx.Context, host string, port int, check convAuth.Check, svc any, logCalls bool) http.Handler {
	return &httpHandler{ctx, computeEndpoints(host, port, svc), check, logCalls}
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
	ctx      convCtx.Context
	eps      endpoints
	check    convAuth.Check
	logCalls bool
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx.
		WithRequest(r)

	_, err := h.check(r)
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

	if h.logCalls {
		logCall(ctx, w, r, func(w http.ResponseWriter, r *http.Request) {
			execIfMatch(ctx, w, r, h.eps)
		})
	} else {
		execIfMatch(ctx, w, r, h.eps)
	}
}

func execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request, eps endpoints) {
	ok := false
	for _, ep := range eps {
		ok = ep.execIfMatch(ctx, w, r)
		if ok {
			break
		}
	}
	if !ok {
		ServeError(ctx, w, http.StatusNotFound, ErrorCodeNotFound, "Endpoint not found", nil)
	}
}

func logCall(ctx convCtx.Context, w http.ResponseWriter, r *http.Request, handle func(w http.ResponseWriter, r *http.Request)) {

	logger := ctx.Logger()
	if logger == nil {
		handle(w, r)
		return
	}

	rec := httptest.NewRecorder()

	// temporary hide the Authorization header while we dump the request
	authHeader := r.Header.Get(convAuth.HttpHeaderAuthorization)
	if authHeader != "" {
		l := len(authHeader) - 10
		if l < 0 {
			l = len(authHeader)
		}
		r.Header.Set(convAuth.HttpHeaderAuthorization, "..."+authHeader[l:])
	}

	contentHeader := r.Header.Get("Content-Type")
	logBody := contentHeader == "application/json" ||
		contentHeader == "application/xml" ||
		contentHeader == "application/yaml" ||
		contentHeader == "text/plain" ||
		contentHeader == "text/html"

	reqDump, err := httputil.DumpRequest(r, logBody)
	if err != nil {
		logger.Warn("error dumping request for logging", "error", err)
		return
	}

	// capture request headers while Authorization is still masked
	reqHeaderAttrs := headersToAttrs(r.Header)

	// restore the Authorization header
	if authHeader != "" {
		r.Header.Set(convAuth.HttpHeaderAuthorization, authHeader)
	}

	handle(rec, r)

	res := rec.Result()

	contentHeader = res.Header.Get("Content-Type")
	logBody = contentHeader == "application/json" ||
		contentHeader == "application/xml" ||
		contentHeader == "application/yaml" ||
		contentHeader == "text/plain" ||
		contentHeader == "text/html"

	resDump, err := httputil.DumpResponse(res, logBody)
	if err != nil {
		logger.Warn("error dumping response for logging", "error", err)
		return
	}

	for k, v := range res.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	w.WriteHeader(res.StatusCode)

	if res.Body != nil {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			logger.Warn("error reading response body after logging", "error", err)
			return
		}
		_, err = w.Write(resBody)
		if err != nil {
			logger.Warn("error writing response body after logging", "error", err)
			return
		}
	}

	resHeaderAttrs := headersToAttrs(res.Header)

	logger.
		With(
			"request", string(reqDump),
			"response", string(resDump),
			slog.Group("headers",
				slog.Group("request", reqHeaderAttrs...),
				slog.Group("response", resHeaderAttrs...),
			),
		).
		Info("API call")
}

func headersToAttrs(headers http.Header) []any {
	var attrs []any
	for name, values := range headers {
		attrs = append(attrs, name, strings.Join(values, ", "))
	}
	return attrs
}
