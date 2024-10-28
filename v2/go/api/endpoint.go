package api

import (
	"net/http"
	"reflect"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type endpoint interface {
	execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool
	setDescriptor(desc descriptor)
	getDescriptor() descriptor
	getInOutTypes() (in, out reflect.Type)
	setEndpoints(eps endpoints)
}

type endpoints []endpoint
