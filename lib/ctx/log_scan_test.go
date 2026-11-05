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
// any object of data, which must be exactly one JSON object.
func repeatsKey(data []byte) (repeats, isObject bool) {
	if !json.Valid(data) {
		return false, false
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return false, false
	}
	return repeatsIn(dec), true
}

// repeatsIn consumes the rest of an object whose '{' was already read.
func repeatsIn(dec *json.Decoder) bool {
	rep := false
	seen := map[string]bool{}
	for dec.More() {
		kt, _ := dec.Token()
		k := kt.(string)
		rep = rep || seen[k]
		seen[k] = true
		rep = repeatsValue(dec) || rep
	}
	dec.Token() // }
	return rep
}

func repeatsValue(dec *json.Decoder) bool {
	tok, _ := dec.Token()
	switch tok {
	case json.Delim('{'):
		return repeatsIn(dec)
	case json.Delim('['):
		rep := false
		for dec.More() {
			rep = repeatsValue(dec) || rep
		}
		dec.Token() // ]
		return rep
	}
	return false
}

type rawToken struct {
	raw   string
	isKey bool
}

// rawTokens returns the raw bytes of every token of valid JSON, with object
// keys marked, using the decoder's offsets.
func rawTokens(data []byte) []rawToken {
	dec := json.NewDecoder(bytes.NewReader(data))
	var out []rawToken
	type frame struct{ obj, key bool }
	var stack []frame
	prev := int64(0)
	for {
		tok, err := dec.Token()
		if err != nil {
			return out
		}
		off := dec.InputOffset()
		raw := string(bytes.Trim(data[prev:off], " \t\r\n,:"))
		prev = off
		isKey := len(stack) > 0 && stack[len(stack)-1].obj && stack[len(stack)-1].key
		out = append(out, rawToken{raw, isKey})
		d, isDelim := tok.(json.Delim)
		if isDelim && (d == '{' || d == '[') {
			stack = append(stack, frame{obj: d == '{', key: d == '{'})
			continue
		}
		if isDelim { // closing delimiter
			stack = stack[:len(stack)-1]
		}
		if n := len(stack); n > 0 && stack[n-1].obj {
			stack[n-1].key = !isKey
		}
	}
}

func FuzzDedupeWriter(f *testing.F) {
	for _, s := range []string{
		`{}`,
		`{"a":1}` + "\n",
		`{"a":1,"a":2}` + "\n",
		`{"a":1,"\u0061":2}`,
		`{"o":{"k":1,"k":2},"k":1,"l":[{"x":1,"x":2}]}`,
		`{"a":1,"attr_a":2,"a":3}`,
		"{\"\xff\":1,\"\xfe\":2}",
		`{"\ufffd":1,"` + "\xff" + `":2}`,
		`{"k":1,"k":2,"s":"\ud800"}`,
		"{\"k\":1,\"k\":2,\"s\":\"\xff\"}",
		`{"\ud83d\ude00":1,"` + "\U0001F600" + `":2}`,
		`{"\ud800":1,"\ud801":2}`,
		`{"a":"{\"a\":1,\"a\":2}","a":1.50e3}`,
		`{"":1,"":2}`,
		` {"a":1,"a":2} `,
		`[1,2]`,
		`{"a":1}{"a":2}`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var out bytes.Buffer
		n, err := newDedupeWriter(&out).Write(data)
		if err != nil || n != len(data) {
			t.Fatalf("Write = %d, %v", n, err)
		}

		// The strict form: one JSON object, optionally followed by one newline.
		body := bytes.TrimSuffix(data, []byte("\n"))
		strict := len(body) > 0 && body[0] == '{' && body[len(body)-1] == '}' && json.Valid(body)
		if !json.Valid(data) {
			if !bytes.Equal(out.Bytes(), data) {
				t.Fatalf("changed malformed input:\n in: %q\nout: %q", data, out.Bytes())
			}
			return
		}
		repeats, isObject := repeatsKey(data)
		if !isObject {
			if !bytes.Equal(out.Bytes(), data) {
				t.Fatalf("changed a non-object:\n in: %q\nout: %q", data, out.Bytes())
			}
			return
		}
		if !strict {
			// Padded with whitespace: only the guarantee is checked.
			if again, _ := repeatsKey(out.Bytes()); again {
				t.Fatalf("repeat left:\n in: %q\nout: %q", data, out.Bytes())
			}
			return
		}
		if !repeats {
			if !bytes.Equal(out.Bytes(), data) {
				t.Fatalf("changed a line without a repeat:\n in: %q\nout: %q", data, out.Bytes())
			}
			return
		}
		if again, isObject := repeatsKey(out.Bytes()); !isObject || again {
			t.Fatalf("output is not a repeat-free object:\n in: %q\nout: %q", data, out.Bytes())
		}
		if bytes.HasSuffix(out.Bytes(), []byte("\n")) != bytes.HasSuffix(data, []byte("\n")) {
			t.Fatalf("trailing newline changed: %q -> %q", data, out.Bytes())
		}
		in, got := rawTokens(data), rawTokens(out.Bytes())
		if len(in) != len(got) {
			t.Fatalf("token count %d -> %d:\n in: %q\nout: %q", len(in), len(got), data, out.Bytes())
		}
		for i := range in {
			if in[i] == got[i] {
				continue
			}
			renamed := in[i].isKey && got[i].isKey && strings.HasPrefix(got[i].raw, `"attr_`) &&
				strings.HasSuffix(got[i].raw, in[i].raw[1:])
			if renamed {
				prefix := got[i].raw[1 : len(got[i].raw)-len(in[i].raw)+1]
				renamed = prefix != "" && prefix == strings.Repeat("attr_", len(prefix)/len("attr_"))
			}
			if !renamed {
				t.Fatalf("token %d changed %q -> %q:\n in: %q\nout: %q", i, in[i].raw, got[i].raw, data, out.Bytes())
			}
		}
	})
}
