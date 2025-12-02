package ctx

import (
	convCfg "github.com/sofmon/convention/lib/cfg"
)

type Environment string

const (
	EnvironmentProduction Environment = "production"
)

func getEnv() Environment {
	envStr, err := convCfg.String("environment")
	if err != nil {
		// failed to get environment from config
		// it is safer to assuming 'production'
		return EnvironmentProduction
	}
	return Environment(envStr)
}

func (ctx Context) Environment() Environment {
	svc, _ := ctx.Value(contextKeyEnv).(Environment)
	return svc
}

func (ctx Context) IsProdEnv() bool {
	return ctx.Environment() == EnvironmentProduction
}
