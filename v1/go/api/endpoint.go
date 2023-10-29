package api

import (
	"net/http"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

type endpoint interface {
	execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool
	setDescriptor(desc descriptor)
	getDescriptor() descriptor
}

type endpoints []endpoint
