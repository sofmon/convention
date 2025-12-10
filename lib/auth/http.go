package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"

	convCfg "github.com/sofmon/convention/lib/cfg"
)

const (
	HttpHeaderAuthorization = "Authorization"
)

var (
	hmacSecret []byte

	ErrMissingRequest             = errors.New("HTTP request is nil")
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

func DecodeHTTPRequestClaims(r *http.Request) (res Claims, err error) {

	authHeader := r.Header.Get(HttpHeaderAuthorization)
	if authHeader == "" {
		err = ErrMissingAuthorizationHeader
		return
	}

	authSplit := strings.Split(authHeader, " ")
	if len(authSplit) != 2 || authSplit[0] != "Bearer" {
		err = ErrMissingAuthorizationHeader
		return
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

func GenerateToken(claims Claims) (string, error) {

	hmac, err := getHmacSecret()
	if err != nil {
		return "", err
	}

	rawClaim := make(map[string]any)

	for k, v := range claims.Additions {
		rawClaim[k] = v
	}

	// fixed claims overwrite any additions with the same key
	rawClaim[claimUser] = string(claims.User)
	rawClaim[claimEntities] = claims.Entities
	rawClaim[claimTenants] = claims.Tenants
	rawClaim[claimRoles] = claims.Roles

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(rawClaim))
	tokenString, err := token.SignedString(hmac)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func DecodeToken(tokenString string) (res Claims, err error) {

	hmac, err := getHmacSecret()
	if err != nil {
		return
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
		return
	}

	if !token.Valid {
		err = ErrInvalidAuthorizationToken
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims == nil {
		err = ErrInvalidAuthorizationToken
		return
	}

	if user, ok := claims[claimUser].(string); ok {
		res.User = User(user)
	}

	if entities, ok := claims[claimEntities].(map[string]any); ok {
		res.Entities = make(RolesPerEntity)
		for entityKey, rolesVal := range entities {
			entity := Entity(entityKey)
			if rolesSlice, ok := rolesVal.([]any); ok {
				roles := make(Roles, len(rolesSlice))
				for i, r := range rolesSlice {
					roles[i] = Role(r.(string))
				}
				res.Entities[entity] = roles
			} else {
				res.Entities[entity] = Roles{}
			}
		}
	}

	if tenants, ok := claims[claimTenants].([]any); ok {
		res.Tenants = make(Tenants, len(tenants))
		for i, tenant := range tenants {
			res.Tenants[i] = Tenant(tenant.(string))
		}
	}

	if roles, ok := claims[claimRoles].([]any); ok {
		res.Roles = make(Roles, len(roles))
		for i, role := range roles {
			res.Roles[i] = Role(role.(string))
		}
	}

	return
}
