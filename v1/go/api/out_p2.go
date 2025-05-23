package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

func NewOutP2[outT any, p1T, p2T ~string](fn func(ctx convCtx.Context, p1 p1T, p2 p2T) (outT, error)) OutP2[outT, p1T, p2T] {
	return OutP2[outT, p1T, p2T]{
		fn: fn,
	}
}

func (x OutP2[outT, p1T, p2T]) WithPreCheck(check Check) OutP2[outT, p1T, p2T] {
	return OutP2[outT, p1T, p2T]{
		fn: func(ctx convCtx.Context, p1 p1T, p2 p2T) (res outT, err error) {
			err = check(ctx)
			if err != nil {
				return
			}
			return x.fn(ctx, p1, p2)
		},
	}
}

func (x OutP2[outT, p1T, p2T]) WithPostCheck(check Check) OutP2[outT, p1T, p2T] {
	return OutP2[outT, p1T, p2T]{
		fn: func(ctx convCtx.Context, p1 p1T, p2 p2T) (res outT, err error) {
			res, err = x.fn(ctx, p1, p2)
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

type OutP2[outT any, p1T, p2T ~string] struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, p1 p1T, p2 p2T) (outT, error)
}

func (x *OutP2[outT, p1T, p2T]) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

	values, match := x.descriptor.match(r)
	if !match {
		return false
	}

	out, err := x.fn(
		ctx,
		p1T(values.GetByIndex(0)),
		p2T(values.GetByIndex(1)),
	)
	if err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) {
			serveError(w, *apiErr)
		} else {
			ServeError(ctx, w, http.StatusInternalServerError, ErrorCodeInternalError, "unexpected error", err)
		}
	} else {
		ServeJSON(w, out)
	}

	return true
}

func (x *OutP2[outT, p1T, p2T]) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *OutP2[outT, p1T, p2T]) getDescriptor() descriptor {
	return x.descriptor
}

func (x *OutP2[outT, p1T, p2T]) getInOutTypes() (in, out reflect.Type) {
	return nil, reflect.TypeOf(new(outT))
}

func (x *OutP2[outT, p1T, p2T]) setEndpoints(eps endpoints) {}

func (x *OutP2[outT, p1T, p2T]) Call(ctx convCtx.Context, p1 p1T, p2 p2T) (out outT, err error) {

	if !x.descriptor.isSet() {
		err = errors.New("api not initialized as client; user convAPI.NewClient to create client form api definition")
		return
	}

	values := values{
		{Name: "", Value: string(p1)},
		{Name: "", Value: string(p2)},
	}

	req, err := x.descriptor.newRequest(values, nil)
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
