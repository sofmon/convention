package auth

import (
	"errors"
	"net/http"
	"strings"
)

var (
	ErrForbidden = errors.New("authenticated user has no permission to access the requested resource")
)

type allowedActions []allowedAction

type allowedAction struct {
	method  string
	path    allowedPath
	openEnd bool
}

// actionSource tracks which role an action came from
type actionSource struct {
	action allowedAction
	role   Role
}

type allowedActionSources []actionSource

func expandConfig(policy Policy) (actions allowedActionSources, publicActions allowedActions, err error) {

	actions = make(allowedActionSources, 0)

	for role, permissions := range policy.Roles {
		for _, permission := range permissions {
			as, ok := policy.Permissions[permission]
			if !ok {
				continue
			}

			for _, a := range as {
				var aa allowedAction
				aa, err = generateAllowedAction(a)
				if err != nil {
					return
				}
				actions = append(actions, actionSource{
					action: aa,
					role:   role,
				})
			}
		}
	}

	for _, a := range policy.Public {
		var aa allowedAction
		aa, err = generateAllowedAction(a)
		if err != nil {
			return
		}
		publicActions = append(publicActions, aa)
	}

	return
}

type Target struct {
	Tenant Tenant
	User   User
	Entity Entity
}

type Check func(r *http.Request) (Target, error)

func NewCheck(policy Policy) (check Check, err error) {

	actionSources, publicEndpoints, err := expandConfig(policy)
	if err != nil {
		return
	}

	check = func(r *http.Request) (Target, error) {

		var target Target

		if r == nil {
			return target, ErrMissingRequest
		}

		segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

		if publicEndpoints.match(
			r.Method,
			segments,
			Claims{},
			&target,
		) {
			return target, nil
		}

		claims, err := DecodeHTTPRequestClaims(r)
		if err != nil {
			return Target{}, err
		}

		if actionSources.match(
			r.Method,
			segments,
			claims,
			&target,
		) {
			return target, nil
		}

		return target, ErrForbidden
	}

	return
}

func generateAllowedAction(a Action) (res allowedAction, err error) {

	method, path, err := a.MethodPath()
	if err != nil {
		return
	}

	segments := strings.Split(path, "/")

	openEnd := strings.HasSuffix(path, "{any...}")
	if openEnd {
		segments = segments[:len(segments)-1]
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

	res = allowedAction{method, allowedPath, openEnd}

	return
}

func (sources allowedActionSources) match(method string, segments []string, claims Claims, target *Target) bool {
	for _, src := range sources {
		// Try to match this action
		tempTarget := Target{}
		if !src.action.match(method, segments, claims, &tempTarget) {
			continue
		}

		// Action matched! Now validate the role is allowed for this context
		if isRoleAllowed(src.role, tempTarget.Entity, claims) {
			*target = tempTarget
			return true
		}
	}
	return false
}

// isRoleAllowed checks if the role is valid for the matched entity context
func isRoleAllowed(role Role, matchedEntity Entity, claims Claims) bool {
	// Check if role is in user's base roles
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}

	// If no entity was matched in the path, only base roles apply
	if matchedEntity == "" {
		return false
	}

	// Check if role is in the matched entity's roles
	if entityRoles, ok := claims.Entities[matchedEntity]; ok {
		for _, r := range entityRoles {
			if r == role {
				return true
			}
		}
	}

	return false
}

func (as allowedActions) match(method string, segments []string, claims Claims, target *Target) bool {
	for _, allowedAction := range as {
		if allowedAction.match(method, segments, claims, target) {
			return true
		}
	}
	return false
}

func (a allowedAction) match(method string, segments []string, claims Claims, target *Target) bool {

	if a.method != method {
		return false
	}

	if a.openEnd {
		if len(a.path) > len(segments) {
			return false
		}
		return a.path.match(segments[:len(a.path)], claims, target)
	} else {
		return a.path.match(segments, claims, target)
	}
}

type allowedPath []allowedSegment

func (p allowedPath) match(segments []string, claims Claims, target *Target) bool {
	if len(p) != len(segments) {
		return false
	}
	for i := range p {
		if !p[i].Match(segments[i], claims, target) {
			return false
		}
	}
	return true
}

type allowedSegment interface {
	Match(segment string, claims Claims, target *Target) bool
}

type allowedSegmentFixed string

func (s allowedSegmentFixed) Match(segment string, claims Claims, target *Target) bool {
	return string(s) == segment
}

type allowedSegmentAny struct{}

func (s allowedSegmentAny) Match(segment string, claims Claims, target *Target) bool {
	return true
}

type allowedSegmentUser struct{}

func (s allowedSegmentUser) Match(segment string, claims Claims, target *Target) bool {
	if claims.User == User(segment) {
		target.User = User(segment)
		return true
	}
	return false
}

type allowedSegmentTenant struct{}

func (s allowedSegmentTenant) Match(segment string, claims Claims, target *Target) bool {
	tenant := Tenant(segment)
	for _, t := range claims.Tenants {
		if t == tenant {
			target.Tenant = tenant
			return true
		}
	}
	return false
}

type allowedSegmentEntity struct{}

func (s allowedSegmentEntity) Match(segment string, claims Claims, target *Target) bool {
	entity := Entity(segment)

	// Check if entity exists in the Entities map
	if _, exists := claims.Entities[entity]; exists {
		target.Entity = entity
		return true
	}

	return false
}
