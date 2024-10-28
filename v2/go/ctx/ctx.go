package ctx

import (
	"context"

	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCfg "github.com/sofmon/convention/v2/go/cfg"
)

type Context struct {
	context.Context
}

type contextKey int

const (
	contextKeyAgent contextKey = iota
	contextKeyRoles
	contextKeyEnv
	contextKeyRequest
	contextKeyRequestClaims
	contextKeyAction
	contextKeyWorkflow
	contextKeyScope
	contextKeyAgentUser
)

type Agent string

func New(agent Agent) (ctx Context) {
	return WrapContext(context.Background(), agent)
}

func WrapContext(parent context.Context, agent Agent) (ctx Context) {

	ctx.Context = context.WithValue(parent, contextKeyAgent, agent)
	ctx.Context = context.WithValue(ctx.Context, contextKeyEnv, getEnv())

	ctx = ctx.WithScope(string(agent)) // Use agent name as the initial scope

	return
}

func (ctx Context) WithRoles(roles ...convAuth.Role) Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyRoles,
			roles,
		),
	}
}

func getEnv() Environment {
	envStr, err := convCfg.String("environment")
	if err != nil {
		// failed to get environment from config
		// it is safer to assuming 'production'
		return EnvironmentProduction
	}
	return Environment(envStr)
}

func (ctx Context) Agent() Agent {
	return ctx.Value(contextKeyAgent).(Agent)
}

func (ctx Context) AgentRoles() convAuth.Roles {
	return ctx.Value(contextKeyRoles).(convAuth.Roles)
}
