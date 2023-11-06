package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"reflect"
	"sort"
	"time"

	convCfg "github.com/sofmon/convention/v1/go/cfg"
	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

func ListenAndServe(ctx convCtx.Context, host string, port int, svc any) (err error) {
	ctx = ctx.WithScope("convention.api.ListenAndServe")
	defer ctx.Exit(&err)

	if port == 0 {
		port = 443
	}

	return http.ListenAndServeTLS(
		fmt.Sprintf("%s:%d", host, port),              // following convention/v1
		convCfg.FilePath("communication_certificate"), // following convention/v1
		convCfg.FilePath("communication_key"),         // following convention/v1
		NewHandler(ctx, host, port, svc),
	)
}

func NewHandler(ctx convCtx.Context, host string, port int, svc any) http.Handler {
	return httpHandler{ctx, computeEndpoints(host, port, svc)}
}

func computeEndpoints(host string, port int, svc any) (eps endpoints) {

	for _, f := range reflect.VisibleFields(reflect.TypeOf(svc).Elem()) {

		ep, ok := reflect.ValueOf(svc).Elem().FieldByName(f.Name).Addr().Interface().(endpoint)
		if !ok {
			continue
		}

		apiTag := f.Tag.Get("api")
		desc := newDescriptor(host, port, apiTag)
		ep.setDescriptor(desc)

		eps = append(eps, ep)
	}

	sort.Slice(
		eps,
		func(i, j int) bool {
			return eps[i].getDescriptor().weight > eps[j].getDescriptor().weight
		},
	)

	return
}

type httpHandler struct {
	ctx convCtx.Context
	eps endpoints
}

func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx.WithScope("convention.api.httpHandler.serveHTTP")
	defer ctx.Exit(nil)

	rec := httptest.NewRecorder()

	authHeader := r.Header.Get(httpHeaderAuthorization)
	if authHeader != "" {
		l := len(authHeader) - 10
		if l < 0 {
			l = len(authHeader)
		}
		r.Header.Set(httpHeaderAuthorization, "..."+authHeader[l:])
	}

	hasBody := r.Method == http.MethodPost || r.Method == http.MethodPut

	reqDump, err := httputil.DumpRequest(r, hasBody)
	if err != nil {
		ctx.LogWarn(err)
		return
	}

	if authHeader != "" {
		r.Header.Set("Authorization", authHeader)
	}

	ok := false
	for _, ep := range h.eps {
		ok = ep.execIfMatch(ctx, rec, r)
		if ok {
			break
		}
	}

	if !ok {
		ServeError(rec, ErrorCodeNotFound, "Endpoint not found")
	}

	res := rec.Result()

	hasBody = res.Body != nil

	resDump, err := httputil.DumpResponse(res, hasBody)
	if err != nil {
		ctx.LogWarn(err)
		return
	}

	for k, v := range res.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	w.WriteHeader(res.StatusCode)

	if hasBody {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			ctx.LogWarn(err)
			return
		}
		_, err = w.Write(resBody)
		if err != nil {
			ctx.LogWarn(err)
			return
		}
	}

	trace(
		traceEntry{
			Time:       ctx.Now(),
			App:        ctx.App(),
			User:       ctx.User(),
			RequestID:  ctx.RequestID(),
			Method:     r.Method,
			Path:       r.URL.Path,
			StatusCode: res.StatusCode,
			Request:    string(reqDump),
			Response:   string(resDump),
		},
	)
}

type traceEntry struct {
	Time       time.Time         `json:"time,omitempty"`
	App        convCtx.App       `json:"app,omitempty"`
	User       string            `json:"user,omitempty"`
	RequestID  convCtx.RequestID `json:"request_id,omitempty"`
	Method     string            `json:"method,omitempty"`
	Path       string            `json:"path,omitempty"`
	StatusCode int               `json:"status_code,omitempty"`
	Request    string            `json:"request,omitempty"`
	Response   string            `json:"response,omitempty"`
}

func trace(e traceEntry) {
	out, err := json.Marshal(e)
	if err != nil {
		fmt.Printf("%v\n", e)
	} else {
		fmt.Println(string(out))
	}
}
