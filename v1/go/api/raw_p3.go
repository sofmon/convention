package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

func NewRawP3[p1T, p2T, p3T ~string](fn func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, w http.ResponseWriter, r *http.Request)) RawP3[p1T, p2T, p3T] {
	return RawP3[p1T, p2T, p3T]{
		fn: fn,
	}
}

func (x RawP3[p1T, p2T, p3T]) WithPreCheck(check Check) RawP3[p1T, p2T, p3T] {
	return RawP3[p1T, p2T, p3T]{
		fn: func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, w http.ResponseWriter, r *http.Request) {
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

			x.fn(ctx, p1, p2, p3, w, r)
		},
	}
}

type RawP3[p1T, p2T, p3T ~string] struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, w http.ResponseWriter, r *http.Request)
}

func (x *RawP3[p1T, p2T, p3T]) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

	values, match := x.descriptor.match(r)
	if !match {
		return false
	}

	x.fn(
		ctx.WithRequest(r),
		p1T(values.GetByIndex(0)),
		p2T(values.GetByIndex(1)),
		p3T(values.GetByIndex(2)),
		w,
		r,
	)

	return true
}

func (x *RawP3[p1T, p2T, p3T]) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *RawP3[p1T, p2T, p3T]) getDescriptor() descriptor {
	return x.descriptor
}

func (x *RawP3[p1T, p2T, p3T]) getInOutTypes() (in, out reflect.Type) {
	return nil, nil
}

func (x *RawP3[p1T, p2T, p3T]) setEndpoints(eps endpoints) {}

func (x *RawP3[p1T, p2T, p3T]) Call(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, body io.Reader) (err error) {

	if !x.descriptor.isSet() {
		err = errors.New("api not initialized as client; user convAPI.NewClient to create client form api definition")
		return
	}

	values := values{
		{Name: "", Value: string(p1)},
		{Name: "", Value: string(p2)},
		{Name: "", Value: string(p3)},
	}

	req, err := x.descriptor.newRequest(values, body)
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

	err = errors.New("unexpected status code: " + res.Status)

	return
}
