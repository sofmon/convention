package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

func NewInP4[inT any, p1T, p2T, p3T, p4T ~string](fn func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, p4 p4T, in inT) error) InP4[inT, p1T, p2T, p3T, p4T] {
	return InP4[inT, p1T, p2T, p3T, p4T]{
		fn: fn,
	}
}

func (x InP4[inT, p1T, p2T, p3T, p4T]) WithPreCheck(check Check) InP4[inT, p1T, p2T, p3T, p4T] {
	return InP4[inT, p1T, p2T, p3T, p4T]{
		fn: func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, p4 p4T, in inT) error {
			err := check(ctx)
			if err != nil {
				return err
			}
			return x.fn(ctx, p1, p2, p3, p4, in)
		},
	}
}

func (x InP4[inT, p1T, p2T, p3T, p4T]) WithPostCheck(check Check) InP4[inT, p1T, p2T, p3T, p4T] {
	return InP4[inT, p1T, p2T, p3T, p4T]{
		fn: func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, p4 p4T, in inT) error {

			err := x.fn(ctx, p1, p2, p3, p4, in)
			if err != nil {
				return err
			}
			return check(ctx)
		},
	}
}

type InP4[inT any, p1T, p2T, p3T, p4T ~string] struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, p4 p4T, in inT) error
}

func (x *InP4[inT, p1T, p2T, p3T, p4T]) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

	values, match := x.descriptor.match(r)
	if !match {
		return false
	}

	var in inT
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		ServeError(w, http.StatusBadRequest, ErrorCodeBadRequest, err.Error())
		return true
	}

	err = x.fn(
		ctx.WithRequest(r),
		p1T(values.GetByIndex(0)),
		p2T(values.GetByIndex(1)),
		p3T(values.GetByIndex(2)),
		p4T(values.GetByIndex(3)),
		in,
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

func (x *InP4[inT, p1T, p2T, p3T, p4T]) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *InP4[inT, p1T, p2T, p3T, p4T]) getDescriptor() descriptor {
	return x.descriptor
}

func (x *InP4[inT, p1T, p2T, p3T, p4T]) Call(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, p4 p4T, in inT) (err error) {

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
		{Name: "", Value: string(p4)},
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
