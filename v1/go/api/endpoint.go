package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	convAuth "github.com/sofmon/convention/v1/go/auth"
	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

type Endpoint[reqT, resT any] struct {
	urlTemplate string
}

func NewEndpoint[reqT any, resT any](urlTemplate string) (ep *Endpoint[reqT, resT]) {
	ep = &Endpoint[reqT, resT]{
		urlTemplate: urlTemplate,
	}
	return
}

func (ep Endpoint[reqT, resT]) Get(ctx convCtx.Context, params ...any) (res resT, err error) {

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(ep.urlTemplate, params...), nil)
	if err != nil {
		return
	}

	err = passHeaders(ctx, req)
	if err != nil {
		return
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return
	}

	if rsp.StatusCode == http.StatusConflict {
		var apiErr Error
		if e := json.Unmarshal(body, &apiErr); e == nil {
			err = apiErr
			return
		}
	}

	if rsp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code: %d, body: %s", rsp.StatusCode, string(body))
		return
	}

	if len(body) > 0 {
		err = json.Unmarshal(body, &res)
		if err != nil {
			return
		}
	}

	return
}

func (ep Endpoint[reqT, resT]) Put(ctx convCtx.Context, obj reqT, params ...any) (err error) {

	body, err := json.Marshal(obj)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf(ep.urlTemplate, params...), bytes.NewReader(body))
	if err != nil {
		return
	}

	err = passHeaders(ctx, req)
	if err != nil {
		return
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if rsp.StatusCode == http.StatusOK {
		return // no body to process
	}

	body, err = io.ReadAll(rsp.Body)
	if err != nil {
		return
	}

	if rsp.StatusCode == http.StatusConflict {
		var apiErr Error
		if e := json.Unmarshal(body, &apiErr); e == nil {
			err = apiErr
			return
		}
	}

	err = fmt.Errorf("unexpected status code: %d, body: %s", rsp.StatusCode, string(body))
	return
}

func (ep Endpoint[reqT, resT]) Post(ctx convCtx.Context, obj reqT, params ...any) (res resT, err error) {

	body, err := json.Marshal(obj)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(ep.urlTemplate, params...), bytes.NewReader(body))
	if err != nil {
		return
	}

	err = passHeaders(ctx, req)
	if err != nil {
		return
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	body, err = io.ReadAll(rsp.Body)
	if err != nil {
		return
	}

	if rsp.StatusCode == http.StatusConflict {
		var apiErr Error
		if e := json.Unmarshal(body, &apiErr); e == nil {
			err = apiErr
			return
		}
	}

	if rsp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code: %d, body: %s", rsp.StatusCode, string(body))
		return
	}

	if len(body) > 0 {
		err = json.Unmarshal(body, &res)
		if err != nil {
			return
		}
	}

	return
}

func (ep Endpoint[reqT, resT]) Delete(ctx convCtx.Context, params ...any) (err error) {

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf(ep.urlTemplate, params...), nil)
	if err != nil {
		return
	}

	err = passHeaders(ctx, req)
	if err != nil {
		return
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if rsp.StatusCode == http.StatusOK {
		return // no body to process
	}

	var body []byte
	body, err = io.ReadAll(rsp.Body)
	if err != nil {
		return
	}

	if rsp.StatusCode == http.StatusConflict {
		var apiErr Error
		if e := json.Unmarshal(body, &apiErr); e == nil {
			err = apiErr
			return
		}
	}

	err = fmt.Errorf("unexpected status code: %d, body: %s", rsp.StatusCode, string(body))
	return
}

func passHeaders(ctx convCtx.Context, req *http.Request) (err error) {

	if priviesReq := ctx.Request(); priviesReq != nil {

		rid := priviesReq.Header.Get(convCtx.HttpHeaderRequestID)
		if rid != "" {
			req.Header.Set(convCtx.HttpHeaderRequestID, rid)
		}

		auth := priviesReq.Header.Get(convCtx.HttpHeaderAuthorization)
		if auth != "" {
			req.Header.Set(convCtx.HttpHeaderAuthorization, auth)
		}

		if !ctx.IsProdEnv() {
			now := priviesReq.Header.Get(convCtx.HTTPHeaderTimeNow)
			if now != "" {
				req.Header.Set(convCtx.HTTPHeaderTimeNow, now)
			}
		}

	}

	if ctx.MustUseSystemAccount() {

		err = convAuth.EncodeHTTPRequestClaims(
			req,
			convAuth.NewClaims(string(ctx.App()), false, true),
		)
		if err != nil {
			return
		}

	}

	return
}
