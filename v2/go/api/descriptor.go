package api

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

func newDescriptor(host string, port int, pattern string, in, out reflect.Type) (desc descriptor) {

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
	for i, s := range segmentsSplit {

		if s == "{any...}" && i == len(segmentsSplit)-1 {
			desc.open = true
			break
		}

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
	desc.in = objectFromType(in)
	desc.out = objectFromType(out)

	return
}

type descriptor struct {
	host     string
	port     int
	method   string
	segments []urlSegment
	weight   int
	open     bool

	in, out *object
}

func (desc *descriptor) path() string {
	sb := strings.Builder{}
	for _, segment := range desc.segments {
		sb.WriteRune('/')
		if segment.param {
			sb.WriteRune('{')
			sb.WriteString(segment.value)
			sb.WriteRune('}')
		} else {
			sb.WriteString(segment.value)
		}
	}
	return sb.String()
}

func (desc *descriptor) parameters() (params []string) {
	for _, segment := range desc.segments {
		if segment.param {
			params = append(params, segment.value)
		}
	}
	return
}

type objectType string

func (o objectType) IsSimple() bool {
	switch o {
	case objectTypeString,
		objectTypeInteger,
		objectTypeNumber,
		objectTypeBoolean:
		return true
	default:
		return false
	}
}

const (
	objectTypeString  objectType = "string"
	objectTypeInteger objectType = "integer"
	objectTypeNumber  objectType = "number"
	objectTypeBoolean objectType = "boolean"
	objectTypeArray   objectType = "array"
	objectTypeMap     objectType = "map"
	objectTypeObject  objectType = "object"
	objectTypeInvalid objectType = "invalid"
)

type object struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Type      objectType         `json:"type"`
	Mandatory bool               `json:"mandatory"`
	Elem      *object            `json:"elem"`
	Key       *object            `json:"key"`
	Fields    map[string]*object `json:"fields"`
}

func snakeName(name string) string {
	sb := strings.Builder{}
	wasCapital := false
	for i, r := range name {
		isCapital := 'A' <= r && r <= 'Z'
		isNumber := '0' <= r && r <= '9'
		isLetter := ('a' <= r && r <= 'z') || r == '_' || isCapital
		isAllowed := isNumber || isLetter
		if !isAllowed {
			continue
		}
		if i == 0 {
			wasCapital = isCapital
		}
		if isCapital && !wasCapital {
			sb.WriteRune('_')
		}
		sb.WriteRune(r)
		wasCapital = isCapital
	}
	return strings.ToLower(sb.String())
}

// Remove package name from a fully qualified type name
func friendlyName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Array, reflect.Slice:
		return "list_of_" + friendlyName(t.Elem())
	case reflect.Map:
		return "map_by_" + friendlyName(t.Key()) + "_of_" + friendlyName(t.Elem())
	default:
		name := t.String()
		if dotIndex := strings.LastIndex(name, "."); dotIndex != -1 {
			name = name[dotIndex+1:]
		}
		return snakeName(name)
	}
}

func objectFromType(t reflect.Type, knownObjects ...*object) (o *object) {

	if t == nil {
		return nil
	}

	o = &object{}

	o.Mandatory = t.Kind() != reflect.Pointer

	if !o.Mandatory {
		t = t.Elem() // Dereference if it's a pointer
	}

	o.Name = friendlyName(t)
	o.ID = t.PkgPath() + "/" + o.Name

	for _, known := range knownObjects {
		if known.ID == o.ID {
			return known
		}
	}

	knownObjects = append(knownObjects, o)

	switch t.Kind() {

	case reflect.Bool:
		o.Type = objectTypeBoolean

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		o.Type = objectTypeInteger

	case reflect.Float32, reflect.Float64:
		o.Type = objectTypeNumber

	case reflect.String:
		o.Type = objectTypeString

	case reflect.Array, reflect.Slice:
		o.Type = objectTypeArray
		o.Elem = objectFromType(t.Elem(), knownObjects...)

	case reflect.Map:
		o.Type = objectTypeMap
		o.Key, o.Elem = objectFromType(t.Key(), knownObjects...), objectFromType(t.Elem(), knownObjects...)

	case reflect.Struct:
		o.Type = objectTypeObject
		o.Fields = make(map[string]*object)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue // Skip unexported fields
			}
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue // Skip fields with json tag "-"
			}
			if jsonTag != "" {
				jsonTag = strings.Split(jsonTag, ",")[0]
			}
			if jsonTag == "" {
				jsonTag = field.Name
			}
			o.Fields[jsonTag] = objectFromType(field.Type, knownObjects...)
		}

	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Interface,
		reflect.UnsafePointer:
		o.Type = objectTypeInvalid
	}

	return
}

func (desc *descriptor) isSet() bool {
	return desc != nil && desc.host != ""
}

func (desc *descriptor) match(r *http.Request) (values values, match bool) {

	if desc.method != r.Method {
		match = false
		return
	}

	urlSplit := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(urlSplit) < len(desc.segments) {
		match = false
		return
	}

	segmentsCountMatch := len(urlSplit) == len(desc.segments)

	if !desc.open && !segmentsCountMatch {
		match = false
		return
	}

	for i, segment := range desc.segments {
		if segment.param {
			values.Add(segment.value, urlSplit[i])
			continue
		}
		if segment.value != urlSplit[i] {
			match = false
			return
		}
	}

	match = segmentsCountMatch || desc.open

	return

}

func (desc *descriptor) newRequest(vls values, body io.Reader) (*http.Request, error) {

	sb := strings.Builder{}

	sb.WriteString("https://")
	sb.WriteString(fmt.Sprintf("%s:%d", desc.host, desc.port))
	sb.WriteRune('/')

	vi := 0

	for i, segment := range desc.segments {

		if segment.param {
			val := vls.GetByIndex(vi)
			if val == "" {
				return nil, fmt.Errorf("missing value '%s'", segment.value)
			}
			sb.WriteString(val)
			vi++
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
