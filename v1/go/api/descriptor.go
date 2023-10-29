package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func newDescriptor(host string, port int, pattern string) (desc descriptor) {

	var segmentsSplit []string

	methodSplit := strings.Split(pattern, " ")

	hasMethodSpecific := len(methodSplit) > 1 && strings.HasPrefix(methodSplit[1], "/")

	if hasMethodSpecific {
		desc.method = methodSplit[0]
		segmentsSplit = strings.Split(strings.Trim(methodSplit[1], "/"), "/")
	} else {
		segmentsSplit = strings.Split(strings.Trim(pattern, "/"), "/")
	}

	weight := 0
	for _, s := range segmentsSplit {
		isParam := strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
		if isParam {
			s = strings.TrimLeft(s, "{")
			s = strings.TrimRight(s, "}")
			desc.segments = append(desc.segments, urlSegment{s, true})
		} else {
			desc.segments = append(desc.segments, urlSegment{s, false})
			weight++
		}
	}

	desc.weight = weight
	desc.host = host
	desc.port = port

	return
}

type descriptor struct {
	host     string
	port     int
	method   string
	segments []urlSegment
	weight   int
}

func (desc *descriptor) isSet() bool {
	return desc != nil && desc.host != ""
}

func (desc *descriptor) match(r *http.Request) (values Values, match bool) {

	// if desc.host != "" && desc.host != r.Host {
	// 	match = false
	// 	return
	// }

	values = Values{}

	if desc.method != r.Method {
		match = false
		return
	}

	urlSplit := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	match = len(urlSplit) == len(desc.segments)
	if !match {
		return
	}

	for i, segment := range desc.segments {
		if segment.param {
			values.Set(segment.value, urlSplit[i])
			continue
		}

		if segment.value != urlSplit[i] {
			match = false
			return
		}
	}

	return

}

func (desc *descriptor) newRequest(values Values, body io.Reader) (*http.Request, error) {

	sb := strings.Builder{}

	sb.WriteString("https://")
	sb.WriteString(fmt.Sprintf("%s:%d", desc.host, desc.port))
	sb.WriteRune('/')

	for i, segment := range desc.segments {

		if segment.param {
			val := values.Get(segment.value)
			if val == "" {
				return nil, fmt.Errorf("missing value '%s'", segment.value)
			}
			sb.WriteString(values.Get(segment.value))
		} else {
			sb.WriteString(segment.value)
		}

		if i < len(desc.segments)-1 {
			sb.WriteRune('/')
		}
	}

	return http.NewRequest(desc.method, sb.String(), body)
}

type urlSegment struct {
	value string
	param bool
}
