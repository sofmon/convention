package ctx

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
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

// newDecoder decodes numbers as json.Number so an out-of-range number is not
// an error.
func newDecoder(data []byte) *json.Decoder {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return dec
}

// isObject reports whether data is valid JSON whose top level is an object.
func isObject(data []byte) bool {
	if !json.Valid(data) {
		return false
	}
	tok, err := newDecoder(data).Token()
	return err == nil && tok == json.Delim('{')
}

// repeatsKey is the oracle: it uses encoding/json to find a repeated key in
// any object of data, which must be a valid JSON object.
func repeatsKey(t *testing.T, data []byte) bool {
	t.Helper()
	dec := newDecoder(data)
	if _, err := dec.Token(); err != nil {
		t.Fatalf("oracle: %v", err)
	}
	return repeatsIn(t, dec)
}

// repeatsIn consumes the rest of an object whose '{' was already read.
func repeatsIn(t *testing.T, dec *json.Decoder) bool {
	t.Helper()
	rep := false
	seen := map[string]bool{}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			t.Fatalf("oracle: %v", err)
		}
		k := kt.(string)
		rep = rep || seen[k]
		seen[k] = true
		rep = repeatsValue(t, dec) || rep
	}
	if _, err := dec.Token(); err != nil { // }
		t.Fatalf("oracle: %v", err)
	}
	return rep
}

func repeatsValue(t *testing.T, dec *json.Decoder) bool {
	t.Helper()
	tok, err := dec.Token()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	switch tok {
	case json.Delim('{'):
		return repeatsIn(t, dec)
	case json.Delim('['):
		rep := false
		for dec.More() {
			rep = repeatsValue(t, dec) || rep
		}
		if _, err := dec.Token(); err != nil { // ]
			t.Fatalf("oracle: %v", err)
		}
		return rep
	}
	return false
}

type rawToken struct {
	gap   string // bytes between the previous token and this one
	raw   string
	isKey bool
}

// rawTokens returns every token of valid JSON with the bytes before it and
// whether it is an object key, plus the bytes after the last token. It uses
// the decoder's offsets, so it is independent of the scanner.
func rawTokens(t *testing.T, data []byte) (tokens []rawToken, tail string) {
	t.Helper()
	dec := newDecoder(data)
	type frame struct{ obj, key bool }
	var stack []frame
	prev := int64(0)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return tokens, string(data[prev:])
		}
		if err != nil {
			t.Fatalf("oracle: %v", err)
		}
		chunk := data[prev:dec.InputOffset()]
		raw := bytes.TrimLeft(chunk, " \t\r\n,:")
		gap := string(chunk[:len(chunk)-len(raw)])
		prev = dec.InputOffset()

		d, isDelim := tok.(json.Delim)
		switch {
		case isDelim && (d == '{' || d == '['):
			tokens = append(tokens, rawToken{gap, string(raw), false})
			stack = append(stack, frame{obj: d == '{', key: d == '{'})
		case isDelim:
			tokens = append(tokens, rawToken{gap, string(raw), false})
			stack = stack[:len(stack)-1]
			if n := len(stack); n > 0 && stack[n-1].obj {
				stack[n-1].key = true
			}
		default:
			top := &stack[len(stack)-1]
			isKey := top.obj && top.key
			tokens = append(tokens, rawToken{gap, string(raw), isKey})
			if top.obj {
				top.key = !isKey
			}
		}
	}
}

// isRenamedKey reports whether got is in with attr_ or attr<n>_ (n >= 2)
// inserted right after the opening quote.
func isRenamedKey(in, got string) bool {
	if !strings.HasSuffix(got, in[1:]) || len(got) <= len(in) {
		return false
	}
	prefix := got[1 : len(got)-len(in)+1]
	if prefix == "attr_" {
		return true
	}
	digits, ok := strings.CutPrefix(prefix, "attr")
	digits, ok2 := strings.CutSuffix(digits, "_")
	n, err := strconv.Atoi(digits)
	return ok && ok2 && err == nil && n >= 2 && strconv.Itoa(n) == digits
}

func FuzzDedupeWriter(f *testing.F) {
	for _, s := range []string{
		`{}`,
		`{"a":1}` + "\n",
		`{"a":1,"a":2}` + "\n",
		`{"a":1,"\u0061":2}`,
		`{"o":{"k":1,"k":2},"k":1,"l":[{"x":1,"x":2}]}`,
		`{"o":{},"k":1,"k":2}`,
		`{"a":1e999,"k":1,"k":2}`,
		`{"a":1,"attr_a":2,"a":3}`,
		`{"a":1,"a":2,"attr_a":3,"attr2_a":4,"a":5}`,
		"{\"\xff\":1,\"\xfe\":2}",
		`{"\ufffd":1,"` + "\xff" + `":2}`,
		`{"k":1,"k":2,"s":"\ud800"}`,
		"{\"k\":1,\"k\":2,\"s\":\"\xff\"}",
		`{"\ud83d\ude00":1,"` + "\U0001F600" + `":2}`,
		`{"\ud800":1,"\ud801":2}`,
		`{"a":"{\"a\":1,\"a\":2}","a":1.50e3}`,
		`{"":1,"":2}`,
		` {"a":1,"a":2} `,
		"\t{ \"a\" : 1 ,\n\"a\" : 2 }\r\n\n",
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
		if !isObject(data) {
			if !bytes.Equal(out.Bytes(), data) {
				t.Fatalf("changed input that is not one JSON object:\n in: %q\nout: %q", data, out.Bytes())
			}
			return
		}

		if !isObject(out.Bytes()) || repeatsKey(t, out.Bytes()) {
			t.Fatalf("output is not a repeat-free object:\n in: %q\nout: %q", data, out.Bytes())
		}
		if !repeatsKey(t, data) {
			if !bytes.Equal(out.Bytes(), data) {
				t.Fatalf("changed a line without a repeat:\n in: %q\nout: %q", data, out.Bytes())
			}
			return
		}

		in, inTail := rawTokens(t, data)
		got, gotTail := rawTokens(t, out.Bytes())
		if len(in) != len(got) || inTail != gotTail {
			t.Fatalf("tokens %d -> %d, tail %q -> %q:\n in: %q\nout: %q", len(in), len(got), inTail, gotTail, data, out.Bytes())
		}
		for i := range in {
			same := in[i] == got[i]
			renamed := in[i].isKey && got[i].isKey && in[i].gap == got[i].gap && isRenamedKey(in[i].raw, got[i].raw)
			if !same && !renamed {
				t.Fatalf("token %d changed %+v -> %+v:\n in: %q\nout: %q", i, in[i], got[i], data, out.Bytes())
			}
		}
	})
}
