package ctx

import (
	"context"

	convCfg "github.com/sofmon/convention/v1/go/cfg"
)

type Context struct {
	context.Context
}

type contextKey int

const (
	contextKeyApp contextKey = iota
	contextKeyEnv
	contextKeyRequest
	contextKeyRequestID
	contextKeyRequestClaims
	contextKeyScope
	contextKeySystemUser
)

type App string

func NewContext(app App) (ctx Context) {
	return WrapContext(context.Background(), app)
}

func WrapContext(parent context.Context, app App) (ctx Context) {
	return wrapWithEnv(
		context.WithValue(
			parent,
			contextKeyApp,
			app,
		),
	).WithScope(string(app)) // Use app name as the initial scope
}

func wrapWithEnv(parent context.Context) (ctx Context) {

	var env Environment

	envStr, err := convCfg.String("environment")
	if err != nil {
		// failed to get environment from config, assuming 'production'
		env = EnvironmentProduction
	} else {
		env = Environment(envStr)
	}

	ctx = Context{
		Context: context.WithValue(
			parent,
			contextKeyEnv,
			env,
		),
	}

	return
}

func (ctx Context) App() App {
	app, _ := ctx.Value(contextKeyApp).(App)
	return app
}
