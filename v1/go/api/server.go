package api

import (
	"net/http"
	"strings"

	convCfg "github.com/sofmon/convention/v1/go/cfg"
	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

type Endpoints map[string]func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request, params ...string)

type httpHandler struct {
	ctx convCtx.Context
	eps Endpoints
}

func (h httpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	ctx := h.ctx.WithScope("convention.api.httpHandler.ServeHTTP")
	defer ctx.Exit(nil)

	for path, handler := range h.eps {
		if params, ok := requestMatch(r, path); ok {
			handler(ctx, rw, r, params...)
			return
		}
	}
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
