package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func NewOut[outT any](fn func(ctx convCtx.Context) (outT, error)) Out[outT] {
	return Out[outT]{
		fn: fn,
	}
}

func (x Out[outT]) WithPreCheck(check Check) Out[outT] {
	return Out[outT]{
		fn: func(ctx convCtx.Context) (res outT, err error) {
			err = check(ctx)
			if err != nil {
				return
			}
			return x.fn(ctx)
		},
	}
}

func (x Out[outT]) WithPostCheck(check Check) Out[outT] {
	return Out[outT]{
		fn: func(ctx convCtx.Context) (res outT, err error) {
			res, err = x.fn(ctx)
			if err != nil {
				return
			}
			err = check(ctx)
			if err != nil {
				return
			}
			return
		},
	}
}

type Out[outT any] struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context) (outT, error)
}

func (x *Out[outT]) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

	_, match := x.descriptor.match(r)
	if !match {
		return false
	}

	out, err := x.fn(ctx.WithRequest(r))
	if err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) {
			serveError(w, *apiErr)
		} else {
			ServeError(w, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		}
	} else {
		ServeJSON(w, out)
	}

	return true
}

func (x *Out[outT]) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *Out[outT]) getDescriptor() descriptor {
	return x.descriptor
}

func (x *Out[outT]) getInOutTypes() (in, out reflect.Type) {
	return nil, reflect.TypeOf(new(outT))
}

func (x *Out[outT]) setEndpoints(eps endpoints) {}

func (x *Out[outT]) Call(ctx convCtx.Context) (out outT, err error) {

	if !x.descriptor.isSet() {
		err = errors.New("api not initialized as client; user convAPI.NewClient to create client form api definition")
		return
	}

	req, err := x.descriptor.newRequest(nil, nil)
	if err != nil {
		return
	}

	err = setContextHttpHeaders(ctx, req)
	if err != nil {
		return
	}

	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if res.StatusCode == http.StatusOK {
		err = json.NewDecoder(res.Body).Decode(&out)
		return
	}

	if res.StatusCode == http.StatusConflict {
		var eErr Error
		err = json.NewDecoder(res.Body).Decode(&eErr)
		if err == nil { // if we have error here, we leave it to the generic error below
			err = eErr
			return
		}
	}

	err = errors.New("unexpected status code: " + res.Status + " @ " + req.Method + " " + req.URL.String())

	return
}
