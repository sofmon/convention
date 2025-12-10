package ctx

import (
	"context"
	"log/slog"
	"os"
	"time"

	convAuth "github.com/sofmon/convention/lib/auth"
)

func defaultLogger() *slog.Logger {
	return slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level:     slog.LevelInfo,
				AddSource: true,
			},
		),
	)
}

func (ctx Context) Logger() *slog.Logger {
	// Get base logger (user-provided or default)
	baseLogger, _ := ctx.Value(contextKeyLogger).(*slog.Logger)
	if baseLogger == nil {
		baseLogger = defaultLogger()
	}

	// Build key-value pairs from context values (lazy building)
	var attrs []any

	if env, ok := ctx.Value(contextKeyEnv).(Environment); ok {
		attrs = append(attrs, loggerKeyEnv, env)
	}
	if agent, ok := ctx.Value(contextKeyAgent).(Agent); ok {
		attrs = append(attrs, loggerKeyAgent, agent)
	}
	if workflow, ok := ctx.Value(contextKeyWorkflow).(Workflow); ok {
		attrs = append(attrs, loggerKeyWorkflow, workflow)
	}
	if claims, ok := ctx.Value(contextKeyClaims).(convAuth.Claims); ok {
		attrs = append(attrs, loggerKeyUser, claims.User)
	}
	if scope, ok := ctx.Value(contextKeyScope).(string); ok {
		attrs = append(attrs, loggerKeyScope, scope)
	}
	if action, ok := ctx.Value(contextKeyAction).(convAuth.Action); ok {
		attrs = append(attrs, loggerKeyAction, action)
	}
	if now, ok := ctx.Value(contextKeyNow).(time.Time); ok {
		attrs = append(attrs, loggerKeyNow, now)
	}
	if useAgentClaims, ok := ctx.Value(contextKeyUseAgentClaims).(bool); ok && useAgentClaims {
		attrs = append(attrs, loggerKeyUseAgentClaims, true)
	}

	if len(attrs) == 0 {
		return baseLogger
	}
	return baseLogger.With(attrs...)
}

func (ctx Context) WithLogger(logger *slog.Logger) Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyLogger,
			logger,
		),
	}
}
