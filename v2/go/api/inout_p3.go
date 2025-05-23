package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func NewInOutP3[inT, outT any, p1T, p2T, p3T ~string](fn func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, in inT) (outT, error)) InOutP3[inT, outT, p1T, p2T, p3T] {
	return InOutP3[inT, outT, p1T, p2T, p3T]{
		fn: fn,
	}
}

func (x InOutP3[inT, outT, p1T, p2T, p3T]) WithPreCheck(check Check) InOutP3[inT, outT, p1T, p2T, p3T] {
	return InOutP3[inT, outT, p1T, p2T, p3T]{
		fn: func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, in inT) (res outT, err error) {
			err = check(ctx)
			if err != nil {
				return
			}
			return x.fn(ctx, p1, p2, p3, in)
		},
	}
}

func (x InOutP3[inT, outT, p1T, p2T, p3T]) WithPostCheck(check Check) InOutP3[inT, outT, p1T, p2T, p3T] {
	return InOutP3[inT, outT, p1T, p2T, p3T]{
		fn: func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, in inT) (res outT, err error) {
			res, err = x.fn(ctx, p1, p2, p3, in)
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

type InOutP3[inT, outT any, p1T, p2T, p3T ~string] struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, in inT) (outT, error)
}

func (x *InOutP3[inT, outT, p1T, p2T, p3T]) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

	values, match := x.descriptor.match(r)
	if !match {
		return false
	}

	var in inT
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		ServeError(ctx, w, http.StatusBadRequest, ErrorCodeBadRequest, "unable to decode http payload", err)
		return true
	}

	out, err := x.fn(
		ctx,
		p1T(values.GetByIndex(0)),
		p2T(values.GetByIndex(1)),
		p3T(values.GetByIndex(2)),
		in,
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

func (x *InOutP3[inT, outT, p1T, p2T, p3T]) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *InOutP3[inT, outT, p1T, p2T, p3T]) getDescriptor() descriptor {
	return x.descriptor
}

func (x *InOutP3[inT, outT, p1T, p2T, p3T]) getInOutTypes() (in, out reflect.Type) {
	return reflect.TypeOf(new(inT)), reflect.TypeOf(new(outT))
}

func (x *InOutP3[inT, outT, p1T, p2T, p3T]) setEndpoints(eps endpoints) {}

func (x *InOutP3[inT, outT, p1T, p2T, p3T]) Call(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, in inT) (out outT, err error) {

	if !x.descriptor.isSet() {
		err = errors.New("api not initialized as client; user convAPI.NewClient to create client form api definition")
		return
	}

	body, err := json.Marshal(in)
	if err != nil {
		return
	}

	values := values{
		{Name: "", Value: string(p1)},
		{Name: "", Value: string(p2)},
		{Name: "", Value: string(p3)},
	}

	req, err := x.descriptor.newRequest(values, bytes.NewReader(body))
	if err != nil {
		return
	}

	err = setContextHttpHeaders(ctx, req)
	if err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if res.StatusCode == http.StatusOK {
		err = json.NewDecoder(res.Body).Decode(&out)
		return
	}

	err = parseRemoteError(ctx, req, res)

	return
}
