package ctx

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
)

// dedupeWriter guarantees that no JSON object in a log line repeats a key.
// slog does not deduplicate, and log agents such as fluent-bit drop the line
// or corrupt the value when a key repeats. Working on the encoded line, not on
// slog attributes, sees exactly what the handler wrote, whatever LogValuers,
// ReplaceAttr, raw JSON values or invalid UTF-8 keys turned into.
//
// Every line is first checked by an allocation-free scanner (log_scan.go). Only
// a line with a repeat, or one the scanner cannot vouch for, is decoded with
// encoding/json and rebuilt.
//
// slog.JSONHandler writes each record with one Write call holding one JSON
// object and a trailing newline, under a mutex shared by all derived handlers.
// The writer is stateless, so it is safe for concurrent use, and it does not
// retain p (the handler reuses its buffer after Write returns).
type dedupeWriter struct {
	w io.Writer
}

func newDedupeWriter(w io.Writer) io.Writer { return dedupeWriter{w} }

var lineBuffers = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func (d dedupeWriter) Write(p []byte) (int, error) {
	if cleanLine(p) {
		return d.w.Write(p)
	}
	buf := lineBuffers.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		if buf.Cap() <= 64<<10 {
			lineBuffers.Put(buf)
		}
	}()

	if changed, ok := rewriteLine(p, buf); ok && changed {
		if _, err := d.w.Write(buf.Bytes()); err != nil {
			return 0, err
		}
		return len(p), nil
	}
	return d.w.Write(p)
}

// rewriteLine copies the single JSON object in p to out, renaming each later
// repeat of a key within an object. Key order and values are kept and number
// text is preserved. ok is false if p is not exactly one JSON object.
func rewriteLine(p []byte, out *bytes.Buffer) (changed, ok bool) {
	dec := json.NewDecoder(bytes.NewReader(p))
	dec.UseNumber()
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	rw := rewriter{dec: dec, out: out, enc: enc}

	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return false, false
	}
	if rw.object() != nil {
		return false, false
	}
	if _, err := dec.Token(); err != io.EOF {
		return false, false
	}
	out.WriteByte('\n')
	return rw.changed, true
}

type rewriter struct {
	dec     *json.Decoder
	out     *bytes.Buffer
	enc     *json.Encoder
	changed bool
}

func (r *rewriter) string(s string) error {
	if err := r.enc.Encode(s); err != nil {
		return err
	}
	r.out.Truncate(r.out.Len() - 1) // Encode appends a newline
	return nil
}

// object copies the members of an object whose '{' was already consumed.
func (r *rewriter) object() error {
	r.out.WriteByte('{')
	seen := map[string]struct{}{}
	for first := true; r.dec.More(); first = false {
		kt, err := r.dec.Token()
		if err != nil {
			return err
		}
		key := kt.(string)
		for {
			if _, taken := seen[key]; !taken {
				break
			}
			key = "attr_" + key
			r.changed = true
		}
		seen[key] = struct{}{}

		if !first {
			r.out.WriteByte(',')
		}
		if err := r.string(key); err != nil {
			return err
		}
		r.out.WriteByte(':')
		if err := r.value(); err != nil {
			return err
		}
	}
	if _, err := r.dec.Token(); err != nil {
		return err
	}
	r.out.WriteByte('}')
	return nil
}

func (r *rewriter) array() error {
	r.out.WriteByte('[')
	for first := true; r.dec.More(); first = false {
		if !first {
			r.out.WriteByte(',')
		}
		if err := r.value(); err != nil {
			return err
		}
	}
	if _, err := r.dec.Token(); err != nil {
		return err
	}
	r.out.WriteByte(']')
	return nil
}

func (r *rewriter) value() error {
	tok, err := r.dec.Token()
	if err != nil {
		return err
	}
	switch v := tok.(type) {
	case json.Delim:
		if v == '{' {
			return r.object()
		}
		return r.array()
	case string:
		return r.string(v)
	case json.Number:
		r.out.WriteString(v.String())
	case bool:
		if v {
			r.out.WriteString("true")
		} else {
			r.out.WriteString("false")
		}
	default:
		r.out.WriteString("null")
	}
	return nil
}
