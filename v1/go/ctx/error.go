package ctx

import (
	"encoding/json"
)

const (
	ErrorCodeMissingAuthorizationHeader = "missing-authorization-header"
	ErrorCodeInvalidAuthorizationToken  = "invalid-authorization-token"
)

var (
	ErrMissingAuthorizationHeader = NewError(ErrorCodeMissingAuthorizationHeader, "HTTP request has no valid Bearer authentication; expecting header like 'Authorization: Bearer <token>'")
	ErrInvalidAuthorizationToken  = NewError(ErrorCodeInvalidAuthorizationToken, "HTTP request has invalid bearer token'")
)

type ErrorCode string

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
