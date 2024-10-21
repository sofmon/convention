package api

import (
	"net/http"
	"net/url"
	"testing"
)

func Test_descriptor(t *testing.T) {

	match := map[string]*http.Request{
		"GET /api/v1/p1":             {Method: "GET", URL: &url.URL{Path: "/api/v1/p1"}},
		"GET /api/v1/{any...}":       {Method: "GET", URL: &url.URL{Path: "/api/v1/p2"}},
		"GET /{any...}":              {Method: "GET", URL: &url.URL{Path: "/anything/else"}},
		"GET /api/{value1}/{any...}": {Method: "GET", URL: &url.URL{Path: "/api/v1/p2/p3/p4/p5"}},
		"GET /api/{value2}/{any...}": {Method: "GET", URL: &url.URL{Path: "/api/v1/"}},
	}

	notMatch := map[string]*http.Request{
		"GET /api/v1/any":            {Method: "POST", URL: &url.URL{Path: "/api/v1/any"}},
		"GET /api/v2/{any...}":       {Method: "GET", URL: &url.URL{Path: "/api/v1/p2"}},
		"GET /api/{any...}":          {Method: "GET", URL: &url.URL{Path: "/anything/else"}},
		"GET /api/{value2}/{any...}": {Method: "GET", URL: &url.URL{Path: "/api/"}},
	}

	for path, req := range match {
		t.Run(path, func(t *testing.T) {
			desc := newDescriptor("localhost", 443, path, nil, nil)

			if _, m := desc.match(req); !m {
				t.Errorf("expected match")
			}
		})
	}

	for path, req := range notMatch {
		t.Run(path, func(t *testing.T) {
			desc := newDescriptor("localhost", 443, path, nil, nil)

			if _, m := desc.match(req); m {
				t.Errorf("expected not match")
			}
		})
	}

}
