package api

import (
	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

type Check func(ctx convCtx.Context) error
