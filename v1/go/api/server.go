package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"sort"
	"strings"
	"time"

	convCfg "github.com/sofmon/convention/v1/go/cfg"
	convCtx "github.com/sofmon/convention/v1/go/ctx"
	convDB "github.com/sofmon/convention/v1/go/db"
)

type UserID string

type User struct {
	UserID    UserID
	Name      string
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
}

func (u User) Trail() convDB.Trail[UserID, UserID] {
	return convDB.Trail[UserID, UserID]{
		ID:        u.UserID,
		ShardKey:  u.UserID,
		CreatedAt: u.CreatedAt,
		CreatedBy: u.CreatedBy,
		UpdatedAt: u.UpdatedAt,
		UpdatedBy: u.UpdatedBy,
	}
}

const (
	httpHeaderAuthorization = "Authorization"
)

type Endpoints map[string]func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request, params ...string)

func computeHandles(eps Endpoints) (hls []handle) {

	hls = make([]handle, 0, len(eps))

	for pattern, handleFunc := range eps {

		hdl := handle{
			pattern:    pattern,
			handleFunc: handleFunc,
		}

		var segmentsSplit []string

		methodSplit := strings.Split(pattern, " ")

		hasMethodSpecific := len(methodSplit) > 1 && strings.HasPrefix(methodSplit[1], "/")

		if hasMethodSpecific {
			hdl.methods = append(hdl.methods, strings.Split(methodSplit[0], "|")...)
			segmentsSplit = strings.Split(strings.Trim(methodSplit[1], "/"), "/")
		} else {
			segmentsSplit = strings.Split(strings.Trim(pattern, "/"), "/")
		}

		weight := 0
		for _, s := range segmentsSplit {
			isParam := strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
			if isParam {
				s = strings.TrimLeft(s, "{")
				s = strings.TrimRight(s, "}")
				hdl.segments = append(hdl.segments, segment{s, true})
			} else {
				hdl.segments = append(hdl.segments, segment{s, false})
				weight++
			}
		}

		hls = append(hls, hdl)
	}

	sort.Slice(
		hls,
		func(i, j int) bool {
			return hls[i].weight > hls[j].weight
		},
	)

	return
}

type segment struct {
	value string
	param bool
}

type handle struct {
	pattern    string
	handleFunc func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request, params ...string)

	methods  []string
	segments []segment
	weight   int
}

func (e handle) Match(r *http.Request) (params []string, match bool) {

	if len(e.methods) > 0 {
		for _, method := range e.methods {
			if r.Method == method {
				match = true
			}
		}
		if !match {
			return
		}
	}

	urlSplit := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	match = len(urlSplit) == len(e.segments)
	if !match {
		return
	}

	for i, segment := range e.segments {
		if segment.param {
			params = append(params, urlSplit[i])
			continue
		}

		if segment.value != urlSplit[i] {
			match = false
			return
		}
	}

	return
}

type httpHandler struct {
	ctx convCtx.Context
	hls []handle
}

func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx.WithScope("convention.api.httpHandler.ServeHTTP")
	defer ctx.Exit(nil)

	for _, handler := range h.hls {
		if params, ok := handler.Match(r); ok {

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

			handler.handleFunc(ctx, rec, r, params...)

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
		httpHandler{ctx, computeHandles(eps)},
	)
}

func ServeJSON(w http.ResponseWriter, body any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(body)
}

func ReceiveJSON[T any](r *http.Request) (res T, err error) {
	err = json.NewDecoder(r.Body).Decode(&res)
	return
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
