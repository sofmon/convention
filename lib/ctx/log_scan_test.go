package ctx

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCleanLine(t *testing.T) {
	deep := func(n int) string {
		return strings.Repeat(`{"k":`, n) + "1" + strings.Repeat("}", n)
	}
	manyKeys := func(n int) string {
		var b strings.Builder
		b.WriteString("{")
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(`"k` + strings.Repeat("x", i) + `":1`)
		}
		b.WriteString("}")
		return b.String()
	}

	clean := []string{
		`{}`,
		"{}\n",
		" \t\r\n{ \"a\" : 1 , \"b\" : { } }\n",
		`{"a":1,"b":{"a":2}}`,
		`{"x":{"k":1},"y":{"k":2}}`,
		`{"l":[{"k":1},{"k":2},[{"k":3},{"k":4}]]}`,
		`{"a":"{}\":\\:,[]","b":"x"}`,
		`{"a":"é\n\/","b":"é"}`,
		`{"n":[0,-0,1.5,-0.5e+10,2E-3,10],"t":true,"f":false,"z":null}`,
		`{"é":1,"ü":2}`,
		`{"a":[],"b":[ ],"c":{}}`,
		deep(100),
		manyKeys(maxScanKeys),
	}
	for _, in := range clean {
		if !cleanLine([]byte(in)) {
			t.Errorf("want clean: %.80q", in)
		}
	}

	notClean := []string{
		// repeats
		`{"a":1,"a":2}`,
		`{"a":{"a":1},"a":2}`,
		`{"o":{"k":1,"k":2}}`,
		`{"l":[{"k":1,"k":2}]}`,
		`{"a":1,"b":2,"c":3,"a":4}`,
		// equal when decoded, spelled differently or not decidable
		`{"a":1,"a":2}`,
		`{"\"":1,"\"":2}`,
		`{"\xff":1,"\xfe":2}`,
		"{\"\xff\":1,\"\xfe\":2}",
		// too big or too deep for the fast path
		manyKeys(maxScanKeys + 1),
		deep(maxScanDepth + 2),
		// malformed or not a single object
		``,
		` `,
		`[]`,
		`1`,
		`"a"`,
		`{`,
		`{"a"`,
		`{"a":`,
		`{"a"}`,
		`{"a":}`,
		`{"a":1,}`,
		`{,"a":1}`,
		`{"a":1 "b":2}`,
		`{a:1}`,
		`{"a":01}`,
		`{"a":1.}`,
		`{"a":-}`,
		`{"a":1e}`,
		`{"a":tru}`,
		`{"a":nul}`,
		`{"a":"x}`,
		`{"a":"\q"}`,
		`{"a":"\u12"}`,
		"{\"a\":\"x\ny\"}",
		`{"a":[1,]}`,
		`{"a":[1 2]}`,
		`{"a":1}{"b":2}`,
		`{"a":1} x`,
		`{"a":1}}`,
	}
	for _, in := range notClean {
		if cleanLine([]byte(in)) {
			t.Errorf("want not clean: %.80q", in)
		}
	}
}

func TestCleanLine_DoesNotAllocate(t *testing.T) {
	p := []byte(`{"time":"t","level":"INFO","source":{"function":"f","file":"x.go","line":1},"msg":"m","a":[1,{"k":2}],"b":{"k":3}}` + "\n")
	cleanLine(p) // warm the pool
	if n := testing.AllocsPerRun(100, func() { cleanLine(p) }); n != 0 {
		t.Fatalf("allocs = %v", n)
	}
}

// repeatsKey is the oracle: it uses encoding/json to find a repeated key in
// any object of a document that decodes as one JSON object.
func repeatsKey(data []byte) (repeats, isObject bool) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' || !json.Valid(data) {
		return false, false
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	var walk func() bool
	walk = func() bool {
		tok, err := dec.Token()
		if err != nil {
			return false
		}
		d, ok := tok.(json.Delim)
		if !ok {
			return false
		}
		rep := false
		switch d {
		case '{':
			seen := map[string]bool{}
			for dec.More() {
				kt, _ := dec.Token()
				k := kt.(string)
				rep = rep || seen[k]
				seen[k] = true
				rep = walk() || rep
			}
		case '[':
			for dec.More() {
				rep = walk() || rep
			}
		}
		dec.Token()
		return rep
	}
	return walk(), true
}

func FuzzDedupeWriter(f *testing.F) {
	for _, s := range []string{
		`{}`,
		`{"a":1}` + "\n",
		`{"a":1,"a":2}` + "\n",
		`{"a":1,"a":2}`,
		`{"o":{"k":1,"k":2},"k":1,"l":[{"x":1,"x":2}]}`,
		`{"a":1,"attr_a":2,"a":3}`,
		"{\"\xff\":1,\"\xfe\":2}",
		`{"�":1,"` + "\xff" + `":2}`,
		`{"a":"{\"a\":1,\"a\":2}","a":1.50e3}`,
		`{"":1,"":2}`,
		`[1,2]`,
		`{"a":1}{"a":2}`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		repeats, isObject := repeatsKey(data)
		var out bytes.Buffer
		n, err := newDedupeWriter(&out).Write(data)
		if err != nil || n != len(data) {
			t.Fatalf("Write = %d, %v", n, err)
		}
		if !isObject || !repeats {
			if !bytes.Equal(out.Bytes(), data) {
				t.Fatalf("changed a line without a repeat:\n in: %q\nout: %q", data, out.Bytes())
			}
			return
		}
		again, isObject := repeatsKey(out.Bytes())
		if !isObject || again {
			t.Fatalf("output is not a repeat-free object:\n in: %q\nout: %q", data, out.Bytes())
		}
		if !bytes.HasSuffix(out.Bytes(), []byte("\n")) {
			t.Fatalf("missing newline: %q", out.Bytes())
		}
	})
}
