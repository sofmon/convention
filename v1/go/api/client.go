package api

import (
	"reflect"
)

func NewClient[svcT any](host string, port int) (svc *svcT) {

	svc = new(svcT)

	for _, f := range reflect.VisibleFields(reflect.TypeOf(svc).Elem()) {

		ep, ok := reflect.ValueOf(svc).Elem().FieldByName(f.Name).Addr().Interface().(endpoint)
		if !ok {
			continue
		}

		apiTag := f.Tag.Get("api")
		desc := newDescriptor(host, port, apiTag)
		ep.setDescriptor(desc)
	}

	return
}
