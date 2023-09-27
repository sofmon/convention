package convention

import (
	"fmt"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v4"
)

var (
	hmacSampleSecret []byte
)

func init() {
	var err error
	hmacSampleSecret, err = ConfigBytes(configKeyCommSecret)
	if err != nil {
		panic(err)
	}
}

func DecodeHTTPRequestClaims(r *http.Request) (Claims, error) {

	authHeader := r.Header.Get(httpHeaderAuthorization)
	if authHeader == "" {
		return nil, ErrNoAuthorizationHeader
	}

	authSplit := strings.Split(authHeader, " ")
	if len(authSplit) != 2 || authSplit[0] != "Bearer" {
		return nil, ErrNoAuthorizationHeader
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
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	tokenString, err := token.SignedString(hmacSampleSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func decodeToken(tokenString string) (Claims, error) {

	token, err := jwt.Parse(tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return hmacSampleSecret, nil
		},
	)
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims == nil {
		return nil, ErrInvalidToken
	}

	return Claims(claims), nil
}

type Claims map[string]any

const (
	claimUser      = "user"
	claimIsAdmin   = "admin"
	claimIsService = "service"
)

func NewClaims(user string, isAdmin, isService bool) Claims {
	return Claims{
		claimUser:      user,
		claimIsAdmin:   isAdmin,
		claimIsService: isService,
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

func (c Claims) IsService() bool {
	serviceUserAny, ok := c[claimIsService]
	if !ok {
		return false
	}

	serviceUser, ok := serviceUserAny.(bool)
	if !ok {
		return false
	}

	return serviceUser
}

func (c Claims) IsAdminOrService() bool {
	return c.IsAdmin() || c.IsService()
}
