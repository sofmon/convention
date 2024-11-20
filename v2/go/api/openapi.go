package api

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type OpenAPI struct {
	descriptor descriptor
	endpoints  endpoints
	yaml       string
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
	name = strings.Replace(name, "[]", "list_of_", 1)                               // handle array and slice
	name = strings.ReplaceAll(strings.ReplaceAll(name, "map[", "map_of_"), "]", "") // handle map
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

func (x *OpenAPI) execIfMatch(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) bool {
	_, match := x.descriptor.match(r)
	if !match {
		return false
	}

	if x.yaml != "" {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(x.yaml))
		return true
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
		switch schema.Type {
		case objectTypeArray:
			sb.WriteString("      items:\n")
			if schema.Elem.Type.IsSimple() {
				sb.WriteString(fmt.Sprintf("        type: %s\n", schema.Elem.Type))
			} else {
				sb.WriteString(fmt.Sprintf("        $ref: '#/components/schemas/%s'\n", snakeName(schema.Elem.Name)))
			}
		case objectTypeMap:
			sb.WriteString("      additionalProperties:\n")
			if schema.Elem.Type.IsSimple() {
				sb.WriteString(fmt.Sprintf("        type: %s\n", schema.Elem.Type))
			} else {
				sb.WriteString(fmt.Sprintf("        $ref: '#/components/schemas/%s'\n", snakeName(schema.Elem.Name)))
			}
		case objectTypeObject:
			if len(schema.Fields) > 0 {
				sb.WriteString("      properties:\n")
				for name, obj := range schema.Fields {
					sb.WriteString(fmt.Sprintf("        %s:\n", snakeName(name)))
					if obj.Type.IsSimple() {
						sb.WriteString(fmt.Sprintf("          type: %s\n", obj.Type))
					} else {
						sb.WriteString(fmt.Sprintf("          $ref: '#/components/schemas/%s'\n", snakeName(obj.Name)))
					}
					if obj.Mandatory {
						sb.WriteString("          nullable: false\n")
					} else {
						sb.WriteString("          nullable: true\n")
					}
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
		desc := eps[0].getDescriptor()
		params := desc.parameters()
		if len(params) > 0 {
			sb.WriteString("    parameters:\n")
			for _, p := range desc.parameters() {
				sb.WriteString(fmt.Sprintf("      - name: %s\n", p))
				sb.WriteString("        required: true\n")
				sb.WriteString("        in: path\n")
				sb.WriteString("        schema:\n")
				sb.WriteString("          type: string\n")
			}
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

	x.yaml = sb.String()

	fmt.Println(x.yaml)

	w.Header().Set("Content-Type", "application/yaml")
	w.Write([]byte(x.yaml))
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
