package ctx

import (
	"context"
	"log/slog"
	"os"
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
	logger, _ := ctx.Value(contextKeyLogger).(*slog.Logger)
	if logger == nil {
		logger = defaultLogger()
	}
	return logger
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
