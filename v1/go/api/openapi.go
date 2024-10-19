package api

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	convCtx "github.com/sofmon/convention/v1/go/ctx"
)

type OpenAPI struct {
	descriptor descriptor
	endpoints  endpoints
}

func populateSchemas(res map[string]object, o *object) {
	if o == nil || o.Type.IsSimple() {
		return
	}

	if _, ok := res[o.Name]; ok {
		return
	}

	res[o.Name] = *o

	populateSchemas(res, o.Key)
	populateSchemas(res, o.Elem)

	for _, oo := range o.Fields {
		populateSchemas(res, oo)
	}
}

func snakeName(name string) string {
	sb := strings.Builder{}
	wasCapital := false
	for i, r := range name {
		isCapital := 'A' <= r && r <= 'Z'
		isLetter := 'a' <= r && r <= 'z' || r == '_' || isCapital
		if !isLetter {
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

func (x *OpenAPI) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {
	_, match := x.descriptor.match(r)
	if !match {
		return false
	}

	schemas := make(map[string]object)
	for _, ep := range x.endpoints {
		desc := ep.getDescriptor()
		populateSchemas(schemas, desc.in)
		populateSchemas(schemas, desc.out)
	}

	sb := strings.Builder{}

	sb.WriteString("openapi: 3.0.0\n")
	sb.WriteString("info:\n")
	sb.WriteString("  title: API\n")
	sb.WriteString("  version: 1.0.0\n")
	sb.WriteString("components:\n")
	sb.WriteString("  schemas:\n")
	for name, schema := range schemas {
		sb.WriteString(fmt.Sprintf("    %s:\n", snakeName(name)))
		sb.WriteString(fmt.Sprintf("      type: %s\n", schema.Type))
		if schema.Type == "object" {
			sb.WriteString("      properties:\n")
			for name, obj := range schema.Fields {
				sb.WriteString(fmt.Sprintf("        %s:\n", snakeName(name)))
				if obj.Type.IsSimple() {
					sb.WriteString(fmt.Sprintf("          type: %s\n", obj.Type))
				} else {
					sb.WriteString(fmt.Sprintf("          $ref: '#/components/schemas/%s'\n", snakeName(obj.Name)))
				}
				if obj.Mandatory {
					sb.WriteString("          required: true\n")
				} else {
					sb.WriteString("          required: false\n")
				}
			}
		}
	}

	epByPath := make(map[string]endpoints)
	for _, ep := range x.endpoints {
		desc := ep.getDescriptor()
		path := desc.path()
		epByPath[path] = append(epByPath[path], ep)
	}

	sb.WriteString("paths:\n")
	for path, eps := range epByPath {
		sb.WriteString(fmt.Sprintf("  %s:\n", path))
		sb.WriteString("    parameters:\n")
		desc := eps[0].getDescriptor()
		for _, p := range desc.parameters() {
			sb.WriteString(fmt.Sprintf("      - name: %s\n", p))
			sb.WriteString("        in: path\n")
			sb.WriteString("        required: true\n")
			sb.WriteString("        schema:\n")
			sb.WriteString("          type: string\n")
		}
		for _, ep := range eps {
			desc := ep.getDescriptor()
			sb.WriteString(fmt.Sprintf("    %s:\n", strings.ToLower(desc.method)))
			if desc.in != nil {
				sb.WriteString("      requestBody:\n")
				sb.WriteString("        content:\n")
				sb.WriteString("          application/json:\n")
				sb.WriteString("            schema:\n")
				sb.WriteString(fmt.Sprintf("              $ref: '#/components/schemas/%s'\n", snakeName(desc.in.Name)))
			}
			sb.WriteString("      responses:\n")
			sb.WriteString("        '200':\n")
			sb.WriteString("          description: OK\n")
			if desc.out != nil {
				sb.WriteString("          content:\n")
				sb.WriteString("            application/json:\n")
				sb.WriteString("              schema:\n")
				sb.WriteString(fmt.Sprintf("                $ref: '#/components/schemas/%s'\n", snakeName(desc.out.Name)))
			}
		}
	}

	w.Header().Set("Content-Type", "application/yaml")

	w.Write([]byte(sb.String()))

	return true
}

func (x *OpenAPI) setDescriptor(desc descriptor) {
	x.descriptor = desc
}

func (x *OpenAPI) getDescriptor() descriptor {
	return x.descriptor
}

func (x *OpenAPI) getInOutTypes() (in, out reflect.Type) {
	return nil, nil
}

func (x *OpenAPI) setEndpoints(eps endpoints) {
	x.endpoints = eps
}
