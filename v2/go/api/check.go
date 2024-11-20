package api

import (
	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type Check func(ctx convCtx.Context) error
