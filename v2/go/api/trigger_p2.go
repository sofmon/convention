package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func NewTriggerP2[p1T, p2T ~string](fn func(ctx convCtx.Context, p1 p1T, p2 p2T) error) TriggerP2[p1T, p2T] {
	return TriggerP2[p1T, p2T]{
		fn: fn,
	}
}

func (x TriggerP2[p1T, p2T]) WithPreCheck(check Check) TriggerP2[p1T, p2T] {
	return TriggerP2[p1T, p2T]{
		fn: func(ctx convCtx.Context, p1 p1T, p2 p2T) error {
			err := check(ctx)
			if err != nil {
				return err
			}
			return x.fn(ctx, p1, p2)
		},
	}
}

func (x TriggerP2[p1T, p2T]) WithPostCheck(check Check) TriggerP2[p1T, p2T] {
	return TriggerP2[p1T, p2T]{
		fn: func(ctx convCtx.Context, p1 p1T, p2 p2T) error {
			err := x.fn(ctx, p1, p2)
			if err != nil {
				return err
			}
			return check(ctx)
		},
	}
}

type TriggerP2[p1T, p2T ~string] struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, p1 p1T, p2 p2T) error
}

func (x *TriggerP2[p1T, p2T]) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

	values, match := x.descriptor.match(r)
	if !match {
		return false
	}

	err := x.fn(
		ctx.WithRequest(r),
		p1T(values.GetByIndex(0)),
		p2T(values.GetByIndex(1)),
	)
	if err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) {
			serveError(w, *apiErr)
		} else {
			ServeError(w, http.StatusInternalServerError, ErrorCodeInternalError, err.Error())
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}

	return true
}

func (x *TriggerP2[p1T, p2T]) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *TriggerP2[p1T, p2T]) getDescriptor() descriptor {
	return x.descriptor
}

func (x *TriggerP2[p1T, p2T]) getInOutTypes() (in, out reflect.Type) {
	return nil, nil
}

func (x *TriggerP2[p1T, p2T]) setEndpoints(eps endpoints) {}

func (x *TriggerP2[p1T, p2T]) Call(ctx convCtx.Context, p1 p1T, p2 p2T) (err error) {

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
