package api

import (
	"net/http"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

type Check func(ctx convCtx.Context, req http.Request) error
