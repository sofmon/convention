package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func NewRaw(fn func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request)) Raw {
	return Raw{
		fn: fn,
	}
}

func (x Raw) WithPreCheck(check Check) Raw {
	return Raw{
		fn: func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) {
			err := check(ctx)
			if err != nil {
				var apiErr *Error
				if errors.As(err, &apiErr) {
					serveError(w, *apiErr)
				} else {
					ServeError(w, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
				}
				return
			}

			x.fn(ctx, w, r)
		},
	}
}

type Raw struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request)
}

func (x *Raw) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

	_, match := x.descriptor.match(r)
	if !match {
		return false
	}

	x.fn(
		ctx.WithRequest(r),
		w,
		r,
	)

	return true
}

func (x *Raw) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *Raw) getDescriptor() descriptor {
	return x.descriptor
}

func (x *Raw) getInOutTypes() (in, out reflect.Type) {
	return nil, nil
}

func (x *Raw) setEndpoints(eps endpoints) {}

func (x *Raw) Call(ctx convCtx.Context, body io.Reader) (err error) {

	if !x.descriptor.isSet() {
		err = errors.New("api not initialized as client; user convAPI.NewClient to create client form api definition")
		return
	}

	req, err := x.descriptor.newRequest(nil, body)
	if err != nil {
		return
	}

	err = setContextHttpHeaders(ctx, req)
	if err != nil {
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if res.StatusCode == http.StatusOK {
		return
	}

	if res.StatusCode == http.StatusConflict {
		var eErr Error
		err = json.NewDecoder(res.Body).Decode(&eErr)
		if err == nil { // if we have error here, we leave it to the generic error below
			return eErr
		}
	}

	err = errors.New("unexpected status code: " + res.Status + " @ " + req.Method + " " + req.URL.String())

	return
}
