package auth

import (
	"errors"
	"net/http"
	"strings"
)

var (
	ErrForbidden = errors.New("authenticated user has no permission to access the requested resource")
)

type allowedRoles map[Role]allowedActions

type allowedActions []allowedAction

type allowedAction struct {
	method          string
	path            allowedPath
	ignorePathAfter int
}

func expandConfig(cfg Config) (allowed allowedRoles, publicActions allowedActions, err error) {

	allowed = make(allowedRoles)

	for role, permissions := range cfg.Roles {
		for _, permission := range permissions {
			as, ok := cfg.Permissions[permission]
			if !ok {
				continue
			}

			for _, a := range as {
				var allowedAction allowedAction
				allowedAction, err = generateAllowedAction(a)
				if err != nil {
					return
				}
				allowed[role] = append(allowed[role], allowedAction)
			}
		}
	}

	for _, a := range cfg.Public {
		var allowedAction allowedAction
		allowedAction, err = generateAllowedAction(a)
		if err != nil {
			return
		}
		publicActions = append(publicActions, allowedAction)
	}

	return
}

type Check func(r *http.Request) error

func NewCheck(cfg Config) (check Check, err error) {

	allowedRoles, publicEndpoints, err := expandConfig(cfg)
	if err != nil {
		return
	}

	check = func(r *http.Request) error {

		if r == nil {
			return ErrMissingRequest
		}

		segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

		claims, err := DecodeHTTPRequestClaims(r)
		if err == ErrMissingAuthorizationHeader {
			// check only public endpoints
			if publicEndpoints.match(
				r.Method,
				segments,
				Claims{},
			) {
				return nil
			}
		}
		if err != nil {
			return ErrInvalidAuthorizationToken
		}

		if publicEndpoints.match(
			r.Method,
			segments,
			claims,
		) {
			return nil
		}

		if allowedRoles.match(
			r.Method,
			segments,
			claims,
		) {
			return nil
		}

		return ErrForbidden
	}

	return
}

func generateAllowedAction(a Action) (res allowedAction, err error) {

	method, path, err := a.MethodPath()
	if err != nil {
		return
	}

	segments := strings.Split(path, "/")

	ignorePathAfter := -1
	pathIsOpenEnded := strings.HasSuffix(path, "/{any...}")
	if pathIsOpenEnded {
		segments = segments[:len(segments)-1]
		ignorePathAfter = len(segments)
	}

	allowedPath := make(allowedPath, len(segments))

	for i, segment := range segments {
		switch segment {
		case "{any}":
			allowedPath[i] = allowedSegmentAny{}
		case "{user}":
			allowedPath[i] = allowedSegmentUser{}
		case "{tenant}":
			allowedPath[i] = allowedSegmentTenant{}
		case "{entity}":
			allowedPath[i] = allowedSegmentEntity{}
		default:
			allowedPath[i] = allowedSegmentFixed(segment)
		}
	}

	res = allowedAction{method, allowedPath, ignorePathAfter}

	return
}

func (allowedRoles allowedRoles) match(method string, segments []string, claims Claims) bool {

	for _, role := range claims.Roles {
		allowedActions, ok := allowedRoles[role]
		if !ok {
			continue
		}
		if allowedActions.match(method, segments, claims) {
			return true
		}
	}

	return false
}

func (as allowedActions) match(method string, segments []string, claims Claims) bool {
	for _, allowedAction := range as {
		if allowedAction.match(method, segments, claims) {
			return true
		}
	}
	return false
}

func (a allowedAction) match(method string, segments []string, claims Claims) bool {

	if a.method != method {
		return false
	}

	if a.ignorePathAfter >= 0 {
		return a.path.match(segments[:a.ignorePathAfter], claims)
	} else {
		return a.path.match(segments, claims)
	}
}

type allowedPath []allowedSegment

func (p allowedPath) match(segments []string, claims Claims) bool {
	if len(p) != len(segments) {
		return false
	}
	for i := range p {
		if !p[i].Match(segments[i], claims) {
			return false
		}
	}
	return true
}

type allowedSegment interface {
	Match(segment string, claims Claims) bool
}

type allowedSegmentFixed string

func (s allowedSegmentFixed) Match(segment string, claims Claims) bool {
	return string(s) == segment
}

type allowedSegmentAny struct{}

func (s allowedSegmentAny) Match(segment string, claims Claims) bool {
	return true
}

type allowedSegmentUser struct{}

func (s allowedSegmentUser) Match(segment string, claims Claims) bool {
	return claims.User == User(segment)
}

type allowedSegmentTenant struct{}

func (s allowedSegmentTenant) Match(segment string, claims Claims) bool {
	tenant := Tenant(segment)
	for _, t := range claims.Tenants {
		if t == tenant {
			return true
		}
	}
	return false
}

type allowedSegmentEntity struct{}

func (s allowedSegmentEntity) Match(segment string, claims Claims) bool {
	entity := Entity(segment)
	for _, e := range claims.Entities {
		if e == entity {
			return true
		}
	}
	return false
}
