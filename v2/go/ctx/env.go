package ctx

type Environment string

const (
	EnvironmentProduction Environment = "production"
)

func (ctx Context) Environment() Environment {
	svc, _ := ctx.Value(contextKeyEnv).(Environment)
	return svc
}

func (ctx Context) IsProdEnv() bool {
	return ctx.Environment() == EnvironmentProduction
}
