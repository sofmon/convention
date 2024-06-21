package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v4"

	convCfg "github.com/sofmon/convention/v1/go/cfg"
)

const (
	HttpHeaderAuthorization = "Authorization"
)

var (
	hmacSecret []byte

	ErrMissingAuthorizationHeader = errors.New("HTTP request has no valid Bearer authentication; expecting header like 'Authorization: Bearer <token>'")
	ErrInvalidAuthorizationToken  = errors.New("HTTP request has invalid bearer token'")
)

func getHmacSecret() ([]byte, error) {

	if hmacSecret != nil {
		return hmacSecret, nil
	}

	var err error
	hmacSecret, err = convCfg.Bytes("communication_secret")
	if err != nil {
		return nil, err
	}

	return hmacSecret, nil
}

func DecodeHTTPRequestClaims(r *http.Request) (Claims, error) {

	authHeader := r.Header.Get(HttpHeaderAuthorization)
	if authHeader == "" {
		return nil, ErrMissingAuthorizationHeader
	}

	authSplit := strings.Split(authHeader, " ")
	if len(authSplit) != 2 || authSplit[0] != "Bearer" {
		return nil, ErrMissingAuthorizationHeader
	}

	return DecodeToken(authSplit[1])
}

func EncodeHTTPRequestClaims(r *http.Request, claims Claims) error {

	token, err := GenerateToken(claims)
	if err != nil {
		return err
	}

	r.Header.Set(HttpHeaderAuthorization, "Bearer "+token)

	return nil
}

func GenerateToken(claims map[string]any) (string, error) {

	hmac, err := getHmacSecret()
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	tokenString, err := token.SignedString(hmac)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func DecodeToken(tokenString string) (Claims, error) {

	hmac, err := getHmacSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return hmac, nil
		},
	)
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, ErrInvalidAuthorizationToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims == nil {
		return nil, ErrInvalidAuthorizationToken
	}

	return Claims(claims), nil
}
