package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

func NewRaw(fn func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request, values Values)) Raw {
	return Raw{
		fn: fn,
	}
}

type Raw struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request, values Values)
}

func (x *Raw) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

	values, match := x.descriptor.match(r)
	if !match {
		return false
	}

	x.fn(ctx.WithRequest(r), w, r, values)

	return true
}

func (x *Raw) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *Raw) getDescriptor() descriptor {
	return x.descriptor
}

func (x *Raw) Call(ctx convCtx.Context, values Values, body io.Reader) (err error) {
	if !x.descriptor.isSet() {
		err = errors.New("api not initialized as client; user convAPI.NewClient to create client form api definition")
		return
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
