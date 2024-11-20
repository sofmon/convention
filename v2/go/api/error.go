package api

import (
	"encoding/json"
	"net/http"
)

type ErrorCode string

const (
	ErrorCodeInternalError ErrorCode = "internal_error"
	ErrorCodeNotFound      ErrorCode = "not_found"
	ErrorCodeBadRequest    ErrorCode = "bad_request"
	ErrorCodeForbidden     ErrorCode = "forbidden"
	ErrorCodeUnauthorized  ErrorCode = "unauthorized"
)

func NewError(status int, code ErrorCode, message string) (err error) {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

type Error struct {
	Status  int       `json:"http_status,omitempty"`
	Code    ErrorCode `json:"code,omitempty"`
	Message string    `json:"message,omitempty"`
}

func (e Error) Error() string {
	bytes, _ := json.Marshal(e)
	return string(bytes)
}

func ServeError(w http.ResponseWriter, status int, code ErrorCode, message string) {
	serveError(w, Error{status, code, message})
}

func serveError(w http.ResponseWriter, err Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(err)
}
