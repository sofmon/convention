package api

import (
	"encoding/json"
	"errors"
	"net/http"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

func NewOutP1[outT any, p1T ~string](fn func(ctx convCtx.Context, p1 p1T) (outT, error)) OutP1[outT, p1T] {
	return OutP1[outT, p1T]{
		fn: fn,
	}
}

type OutP1[outT any, p1T ~string] struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, p1 p1T) (outT, error)
}

func (x *OutP1[outT, p1T]) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

	values, match := x.descriptor.match(r)
	if !match {
		return false
	}

	out, err := x.fn(
		ctx.WithRequest(r),
		p1T(values.GetByIndex(0)),
	)
	if err != nil {
		var apiErr Error
		if errors.As(err, &apiErr) {
			serveError(w, apiErr)
		} else {
			ServeError(w, ErrorCodeInternalError, err.Error())
		}
	} else {
		ServeJSON(w, out)
	}

	return true
}

func (x *OutP1[outT, p1T]) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *OutP1[outT, p1T]) getDescriptor() descriptor {
	return x.descriptor
}

func (x *OutP1[outT, p1T]) Call(ctx convCtx.Context, p1 p1T) (out outT, err error) {

	if !x.descriptor.isSet() {
		err = errors.New("api not initialized as client; user convAPI.NewClient to create client form api definition")
		return
	}

	values := values{
		{Name: "", Value: string(p1)},
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

	err = errors.New("unexpected status code: " + res.Status)

	return
}
