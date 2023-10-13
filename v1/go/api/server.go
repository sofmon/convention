package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"time"

	convCfg "github.com/sofmon/convention/v1/go/cfg"
	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

const (
	httpHeaderAuthorization = "Authorization"
)

type Endpoints map[string]func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request, params ...string)

type httpHandler struct {
	ctx convCtx.Context
	eps Endpoints
}

func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx.WithScope("convention.api.httpHandler.ServeHTTP")
	defer ctx.Exit(nil)

	for path, handler := range h.eps {
		if params, ok := requestMatch(r, path); ok {

			ctx := ctx.WithRequest(r)

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

			handler(ctx, rec, r, params...)

			res := rec.Result()

			hasBody = res.StatusCode != http.StatusNoContent

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

			return
		}
	}

	ServeError(w, ErrorCodeNotFound, "Endpoint not found")
}

func ListenAndServe(ctx convCtx.Context, eps Endpoints) (err error) {
	ctx = ctx.WithScope("convention.api.ListenAndServe")
	defer ctx.Exit(&err)

	return http.ListenAndServeTLS(":443",
		convCfg.FilePath("communication_certificate"),
		convCfg.FilePath("communication_key"),
		httpHandler{ctx, eps},
	)
}

func requestMatch(r *http.Request, path string) ([]string, bool) {

	urlSplit := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	matchSplit := strings.Split(strings.Trim(path, "/"), "/")

	if len(urlSplit) != len(matchSplit) {
		return nil, false
	}

	var params []string

	pCnt := 0
	for i := 0; i < len(urlSplit); i++ {
		if matchSplit[i] == "%s" {
			params = append(params, urlSplit[i])
			pCnt++
			continue
		}

		if matchSplit[i] != urlSplit[i] {
			return nil, false
		}
	}

	return params, true
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
