package ctx

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	convCfg "github.com/sofmon/convention/v1/go/cfg"

	jwt "github.com/golang-jwt/jwt/v4"
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

	authHeader := r.Header.Get(httpHeaderAuthorization)
	if authHeader == "" {
		return nil, ErrMissingAuthorizationHeader
	}

	authSplit := strings.Split(authHeader, " ")
	if len(authSplit) != 2 || authSplit[0] != "Bearer" {
		return nil, ErrMissingAuthorizationHeader
	}

	return decodeToken(authSplit[1])
}

func EncodeHTTPRequestClaims(r *http.Request, claims Claims) error {

	token, err := generateToken(claims)
	if err != nil {
		return err
	}

	r.Header.Set(httpHeaderAuthorization, "Bearer "+token)

	return nil
}

func generateToken(claims map[string]any) (string, error) {

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

func decodeToken(tokenString string) (Claims, error) {

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

type Claims map[string]any

const (
	claimUser     = "user"
	claimIsAdmin  = "admin"
	claimIsSystem = "system"
)

func NewClaims(user string, isAdmin, isSystem bool) Claims {
	return Claims{
		claimUser:     user,
		claimIsAdmin:  isAdmin,
		claimIsSystem: isSystem,
	}
}

func (c Claims) User() string {
	userAny, ok := c[claimUser]
	if !ok {
		return ""
	}

	user, ok := userAny.(string)
	if !ok {
		return ""
	}

	return user
}

func (c Claims) IsAdmin() bool {
	adminUserAny, ok := c[claimIsAdmin]
	if !ok {
		return false
	}

	adminUser, ok := adminUserAny.(bool)
	if !ok {
		return false
	}

	return adminUser
}

func (c Claims) IsSystem() bool {
	serviceUserAny, ok := c[claimIsSystem]
	if !ok {
		return false
	}

	serviceUser, ok := serviceUserAny.(bool)
	if !ok {
		return false
	}

	return serviceUser
}

func (c Claims) IsAdminOrSystem() bool {
	return c.IsAdmin() || c.IsSystem()
}
