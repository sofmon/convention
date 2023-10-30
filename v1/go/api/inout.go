package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

func NewInOut[inT, outT any](fn func(ctx convCtx.Context, in inT, values Values) (outT, error)) InOut[inT, outT] {
	return InOut[inT, outT]{
		fn: fn,
	}
}

type InOut[inT, outT any] struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, in inT, values Values) (outT, error)
}

func (x InOut[inT, outT]) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

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

	out, err := x.fn(ctx.WithRequest(r), in, values)
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

func (x *InOut[inT, outT]) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *InOut[inT, outT]) getDescriptor() descriptor {
	return x.descriptor
}

func (x *InOut[inT, outT]) Call(ctx convCtx.Context, values Values, in inT) (out outT, err error) {
	if !x.descriptor.isSet() {
		err = errors.New("api not initialized as client; user convAPI.NewClient to create client form api definition")
		return
	}

	body, err := json.Marshal(in)
	if err != nil {
		return
	}

	req, err := x.descriptor.newRequest(values, bytes.NewReader(body))
	if err != nil {
		return
	}

	setContextHttpHeaders(ctx, req)
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
