package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

func NewInOutP4[inT, outT any, p1T, p2T, p3T, p4T ~string](fn func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, p4 p4T, in inT) (outT, error)) InOutP4[inT, outT, p1T, p2T, p3T, p4T] {
	return InOutP4[inT, outT, p1T, p2T, p3T, p4T]{
		fn: fn,
	}
}

type InOutP4[inT, outT any, p1T, p2T, p3T, p4T ~string] struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, p4 p4T, in inT) (outT, error)
}

func (x *InOutP4[inT, outT, p1T, p2T, p3T, p4T]) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

	values, match := x.descriptor.match(r)
	if !match {
		return false
	}

	var in inT
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		ServeError(w, ErrorCodeBadRequest, err.Error())
		return true
	}

	out, err := x.fn(
		ctx.WithRequest(r),
		p1T(values.GetByIndex(0)),
		p2T(values.GetByIndex(1)),
		p3T(values.GetByIndex(2)),
		p4T(values.GetByIndex(3)),
		in,
	)
	if err != nil {
		if e, ok := err.(Error); ok {
			serveError(w, e)
		} else {
			ServeError(w, ErrorCodeInternalError, err.Error())
		}
	} else {
		ServeJSON(w, out)
	}

	return true
}

func (x *InOutP4[inT, outT, p1T, p2T, p3T, p4T]) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *InOutP4[inT, outT, p1T, p2T, p3T, p4T]) getDescriptor() descriptor {
	return x.descriptor
}

func (x *InOutP4[inT, outT, p1T, p2T, p3T, p4T]) Call(ctx convCtx.Context, p1 p1T, p2 p2T, p3 p3T, p4 p4T, in inT) (out outT, err error) {

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
