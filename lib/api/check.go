package api

import (
	convCtx "github.com/sofmon/convention/lib/ctx"
)

type Check func(ctx convCtx.Context) error
