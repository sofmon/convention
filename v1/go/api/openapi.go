package api

import (
	"fmt"
	"net/http"
	"reflect"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

type OpenAPI struct {
	descriptor descriptor
	endpoints  endpoints
}

func populateSchemas(res map[string]object, o *object) {
	if o == nil || o.Type.IsSimple() {
		return
	}

	res[o.Name] = *o

	populateSchemas(res, o.Key)
	populateSchemas(res, o.Elem)

	for _, oo := range o.Fields {
		populateSchemas(res, oo)
	}
}

func (x *OpenAPI) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {
	_, match := x.descriptor.match(r)
	if !match {
		return false
	}

	schemas := make(map[string]object)
	for _, ep := range x.endpoints {
		desc := ep.getDescriptor()
		populateSchemas(schemas, desc.in)
		populateSchemas(schemas, desc.out)
	}

	w.Write([]byte(fmt.Sprintf("%v", schemas)))

	return true
}

func (x *OpenAPI) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *OpenAPI) getDescriptor() descriptor {
	return descriptor{}
}

func (x *OpenAPI) getInOutTypes() (in, out reflect.Type) {
	return nil, nil
}

func (x *OpenAPI) setEndpoints(eps endpoints) {
	x.endpoints = eps
}
