package convention

import "errors"

var (
	ErrNoAuthorizationHeader = errors.New("HTTP request has no valid Bearer authentication; expecting header like 'Authorization: Bearer <token>'")
	ErrInvalidToken          = errors.New("HTTP request has invalid bearer token'")
)
