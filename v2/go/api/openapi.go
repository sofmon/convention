package api

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type OpenAPI struct {
	descriptor    descriptor
	endpoints     endpoints
	yaml          string
	substitutions map[string]*object
	servers       []string
	description   string
}

func NewOpenAPI() OpenAPI {
	return OpenAPI{}
}

type typeSubstitution struct {
	from, to *object
}

func NewTypeSubstitution[fromT any, toT any]() (ts typeSubstitution) {
	ts.from = objectFromType(reflect.TypeOf(new(fromT)))
	ts.to = objectFromType(reflect.TypeOf(new(toT)))
	ts.to.Name = ts.from.Name // keep the name
	ts.to.ID = ts.from.ID     // keep the ID
	return
}

func (o OpenAPI) WithDescription(desc string) OpenAPI {
	o.description = desc
	return o
}

func (o OpenAPI) WithServers(svs ...string) OpenAPI {
	o.servers = append(o.servers, svs...)
	return o
}

func (o OpenAPI) WithTypeSubstitutions(subs ...typeSubstitution) OpenAPI {
	if o.substitutions == nil {
		o.substitutions = make(map[string]*object)
	}
	for _, sub := range subs {
		o.substitutions[sub.from.ID] = sub.to
	}
	return o
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

func (x *OpenAPI) objOrSub(o *object) *object {
	if o == nil {
		return nil
	}
	if x.substitutions == nil {
		return o
	}
	if sub, ok := x.substitutions[o.ID]; ok {
		return sub
	}
	return o
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
		populateSchemas(schemas, x.objOrSub(desc.in))
		populateSchemas(schemas, x.objOrSub(desc.out))
	}

	var uniqueNames = make(map[string]int)
	var knownNames = make(map[string]string)
	uniqueName := func(o object) string {
		if name, op := knownNames[o.ID]; op {
			return name
		}
		name := snakeName(o.Name)
		if _, ok := uniqueNames[name]; !ok {
			uniqueNames[name] = 0
			knownNames[o.ID] = name
			return name
		}
		uniqueNames[name]++
		return fmt.Sprintf("%s_%d", name, uniqueNames[name])
	}

	sb := strings.Builder{}

	sb.WriteString("openapi: 3.0.0\n")
	sb.WriteString("info:\n")
	sb.WriteString("  title: API\n")
	sb.WriteString("  version: 1.0.0\n")
	if x.description != "" {
		sb.WriteString(fmt.Sprintf("  description: %s\n", x.description))
	}
	if len(x.servers) > 0 {
		sb.WriteString("servers:\n")
		for _, sv := range x.servers {
			sb.WriteString(fmt.Sprintf("  - url: %s\n", sv))
		}
	}
	sb.WriteString("components:\n")
	sb.WriteString("  schemas:\n")
	for _, schema := range schemas {
		sb.WriteString(fmt.Sprintf("    %s:\n", uniqueName(schema)))
		sb.WriteString(fmt.Sprintf("      type: %s\n", schema.Type))
		switch schema.Type {
		case objectTypeArray:
			sb.WriteString("      items:\n")
			if schema.Elem.Type.IsSimple() {
				sb.WriteString(fmt.Sprintf("        type: %s\n", schema.Elem.Type))
			} else {
				sb.WriteString(fmt.Sprintf("        $ref: '#/components/schemas/%s'\n", uniqueName(*schema.Elem)))
			}
		case objectTypeMap:
			sb.WriteString("      additionalProperties:\n")
			if schema.Elem.Type.IsSimple() {
				sb.WriteString(fmt.Sprintf("        type: %s\n", schema.Elem.Type))
			} else {
				sb.WriteString(fmt.Sprintf("        $ref: '#/components/schemas/%s'\n", uniqueName(*schema.Elem)))
			}
		case objectTypeObject:
			if len(schema.Fields) > 0 {
				required := make([]string, 0, len(schema.Fields))
				sb.WriteString("      properties:\n")
				for name, obj := range schema.Fields {
					name = snakeName(name)
					sb.WriteString(fmt.Sprintf("        %s:\n", name))
					if obj.Type.IsSimple() {
						sb.WriteString(fmt.Sprintf("          type: %s\n", obj.Type))
					} else {
						sb.WriteString(fmt.Sprintf("          $ref: '#/components/schemas/%s'\n", uniqueName(*obj)))
					}
					if obj.Mandatory {
						sb.WriteString("          nullable: false\n")
						required = append(required, name)
					} else {
						sb.WriteString("          nullable: true\n")
					}
				}
				if len(required) > 0 {
					sb.WriteString("      required:\n")
					for _, name := range required {
						sb.WriteString(fmt.Sprintf("        - %s\n", name))
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
				sb.WriteString(fmt.Sprintf("              $ref: '#/components/schemas/%s'\n", snakeName(x.objOrSub(desc.in).Name)))
			}
			sb.WriteString("      responses:\n")
			sb.WriteString("        '200':\n")
			sb.WriteString("          description: OK\n")
			if desc.out != nil {
				sb.WriteString("          content:\n")
				sb.WriteString("            application/json:\n")
				sb.WriteString("              schema:\n")
				sb.WriteString(fmt.Sprintf("                $ref: '#/components/schemas/%s'\n", snakeName(x.objOrSub(desc.out).Name)))
			}
		}
	}

	x.yaml = sb.String()

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
