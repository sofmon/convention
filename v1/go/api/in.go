package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

func NewIn[inT any](fn func(ctx convCtx.Context, in inT, values Values) error) In[inT] {
	return In[inT]{
		fn: fn,
	}
}

type In[inT any] struct {
	descriptor descriptor
	fn         func(ctx convCtx.Context, in inT, values Values) error
}

func (x *In[inT]) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {

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

	err = x.fn(ctx.WithRequest(r), in, values)
	if err != nil {
		if e, ok := err.(Error); ok {
			serveError(w, e)
			return true
		} else {
			ServeError(w, ErrorCodeInternalError, err.Error())
			return true
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}

	return true
}

func (x *In[inT]) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *In[inT]) getDescriptor() descriptor {
	return x.descriptor
}

func (x *In[inT]) Call(ctx convCtx.Context, values Values, in inT) (err error) {
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
