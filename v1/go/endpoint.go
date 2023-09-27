package convention

import "encoding/json"

type Endpoint[reqT, resT any] struct {
	urlTemplate string
}

func NewEndpoint[reqT any, resT any](urlTemplate string) (ep *Endpoint[reqT, resT]) {
	ep = &Endpoint[reqT, resT]{
		urlTemplate: urlTemplate,
	}
	return
}

func (ep Endpoint[reqT, resT]) Get(ctx Context, url string) (res resT, err error) {
	return
}

func (ep Endpoint[reqT, resT]) Put(ctx Context, url string, req reqT) (res resT, err error) {
	return
}

func (ep Endpoint[reqT, resT]) Post(ctx Context, url string, req reqT) (res resT, err error) {
	return
}

type ErrorCode string

type JSONError struct {
	Code    ErrorCode `json:"code,omitempty"`
	Message string    `json:"message,omitempty"`
}

func (e JSONError) Error() string {
	bytes, _ := json.Marshal(e)
	return string(bytes)
}
