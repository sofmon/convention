package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type ErrorCode string

const (
	ErrorCodeInternalError        ErrorCode = "internal_error"
	ErrorCodeNotFound             ErrorCode = "not_found"
	ErrorCodeBadRequest           ErrorCode = "bad_request"
	ErrorCodeForbidden            ErrorCode = "forbidden"
	ErrorCodeUnauthorized         ErrorCode = "unauthorized"
	ErrorCodeUnexpectedStatusCode ErrorCode = "unexpected_status_code"
)

func NewError(ctx convCtx.Context, status int, code ErrorCode, message string, inner error) (err Error) {
	err.Status = status
	err.Code = code
	err.Message = message

	err.Scope = ctx.Scope()

	r := ctx.Request()
	if r != nil {
		err.Method = r.Method
		err.URL = r.URL.String()
	}
	if inner != nil {
		if apiErr, ok := inner.(*Error); ok {
			err.Inner = apiErr
		} else {
			err.Message += " → " + inner.Error()
		}
	}

	return
}

type Error struct {
	URL     string    `json:"url,omitempty"`
	Method  string    `json:"method,omitempty"`
	Status  int       `json:"status,omitempty"`
	Code    ErrorCode `json:"code,omitempty"`
	Scope   string    `json:"scope,omitempty"`
	Message string    `json:"message,omitempty"`
	Inner   *Error    `json:"inner,omitempty"`
}

func (e Error) Error() string {
	sb := strings.Builder{}
	sb.WriteString("✘ ")
	sb.WriteString(e.Method)
	sb.WriteRune(' ')
	sb.WriteString(e.URL)
	sb.WriteString(" → ")
	sb.WriteString(strconv.Itoa(e.Status))
	sb.WriteRune(' ')
	sb.WriteString(string(e.Code))
	sb.WriteString(" → ")
	sb.WriteString(e.Message)
	if e.Inner != nil {
		sb.WriteString(" → ")
		sb.WriteString(e.Inner.Error())
	}
	return sb.String()
}

func ServeError(ctx convCtx.Context, w http.ResponseWriter, status int, code ErrorCode, message string, inner error) {
	serveError(w, NewError(ctx, status, code, message, inner))
}

func serveError(w http.ResponseWriter, err Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(err)
}

func parseRemoteError(ctx convCtx.Context, req *http.Request, res *http.Response) (err Error) {

	targetUrl := req.URL.String()
	targetMethod := req.Method

	var (
		inner *Error
	)
	inner = &Error{} // reserve memory for inner error
	if e := json.NewDecoder(res.Body).Decode(inner); e != nil {
		inner = nil // no inner error
	}

	if inner != nil &&
		(inner.URL == "" || inner.Method == "" || inner.Status == 0) { // inner is not complete
		inner = nil // no inner error
	}

	if inner != nil && inner.URL != targetUrl && inner.Method != targetMethod {
		err = *inner // if URL and method matches return the inner error directly, no need to wrap it again
		return
	}

	var code ErrorCode
	if inner != nil {
		code = inner.Code
	} else {
		code = ErrorCodeUnexpectedStatusCode
	}

	err = Error{
		URL:     req.URL.String(),
		Method:  req.Method,
		Status:  res.StatusCode,
		Code:    code,
		Scope:   ctx.Scope(),
		Message: "unexpected status code: " + res.Status,
		Inner:   inner,
	}

	return
}
