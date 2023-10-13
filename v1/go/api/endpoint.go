package api

import (
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

func (ep Endpoint[reqT, resT]) Get(ctx convCtx.Context, url string) (res resT, err error) {
	return
}

func (ep Endpoint[reqT, resT]) Put(ctx convCtx.Context, url string, req reqT) (res resT, err error) {
	return
}

func (ep Endpoint[reqT, resT]) Post(ctx convCtx.Context, url string, req reqT) (res resT, err error) {
	return
}
