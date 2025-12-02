package ctx

import (
	"context"
	"time"
)

func (ctx Context) WithNow(now time.Time) Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyNow,
			now,
		),
	}.WithLogger(
		ctx.Logger().With(loggerKeyNow, now),
	)
}

func (ctx Context) Now() time.Time {
	if now, ok := ctx.Value(contextKeyNow).(time.Time); ok {
		return now
	}
	return time.Now().UTC()
}
