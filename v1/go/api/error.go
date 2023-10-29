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
)

func NewError(code ErrorCode, message string) (err error) {
	return &Error{
		Code:    code,
		Message: message,
	}
}

type Error struct {
	Code    ErrorCode `json:"code,omitempty"`
	Message string    `json:"message,omitempty"`
}

func (e Error) Error() string {
	bytes, _ := json.Marshal(e)
	return string(bytes)
}

func ServeError(w http.ResponseWriter, code ErrorCode, message string) {
	serveError(w, Error{code, message})
}

func serveError(w http.ResponseWriter, err Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(err)
}
