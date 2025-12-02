package ctx

import (
	"context"

	"github.com/google/uuid"
	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCfg "github.com/sofmon/convention/v2/go/cfg"
)

type Context struct {
	context.Context
}

type contextKey int

const (
	contextKeyAgent contextKey = iota
	contextKeyAgentClaims
	contextKeyUseAgentClaims
	contextKeyEnv
	contextKeyRequest
	contextKeyClaims
	contextKeyAction
	contextKeyWorkflow
	contextKeyScope
)

type Agent string

func New(claims convAuth.Claims) (ctx Context) {
	return WrapContext(context.Background(), claims)
}

func WrapContext(parent context.Context, claims convAuth.Claims) (ctx Context) {

	agent := Agent(claims.User)

	ctx.Context = context.WithValue(parent, contextKeyEnv, getEnv())
	ctx.Context = context.WithValue(ctx.Context, contextKeyAgent, agent)
	ctx.Context = context.WithValue(ctx.Context, contextKeyAgentClaims, claims)
	ctx.Context = context.WithValue(ctx.Context, contextKeyClaims, claims)
	ctx.Context = context.WithValue(ctx.Context, contextKeyWorkflow, Workflow(uuid.NewString()))
	ctx.Context = context.WithValue(ctx.Context, contextKeyScope, string(agent))

	return
}

func (ctx Context) WithClaims(claims convAuth.Claims) Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyClaims,
			claims,
		),
	}
}

func (ctx Context) Claims() (claims convAuth.Claims) {

	if ctx.mustUseAgentClaims() {
		obj := ctx.Value(contextKeyAgentClaims)
		if obj == nil {
			return
		}
		return obj.(convAuth.Claims)
	}

	obj := ctx.Value(contextKeyClaims)
	if obj == nil {
		return
	}
	return obj.(convAuth.Claims)
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
	obj := ctx.Value(contextKeyAgent)
	if obj == nil {
		return ""
	}
	return obj.(Agent)
}
