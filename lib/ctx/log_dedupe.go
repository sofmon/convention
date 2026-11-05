package ctx

import (
	"bytes"
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
// a line with a repeat, or one the scanner cannot vouch for, is scanned again
// in slow mode, which compares keys by their decoded value. The line is then
// written with "attr_" inserted into each repeated key and every other byte
// untouched, so values are never re-encoded.
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
	s := scanners.Get().(*scanner)
	defer func() {
		s.p = nil // never retain the caller's buffer
		scanners.Put(s)
	}()

	if s.scan(p, false) || !s.scan(p, true) || len(s.renames) == 0 {
		return d.w.Write(p)
	}

	buf := lineBuffers.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		if buf.Cap() <= 64<<10 {
			lineBuffers.Put(buf)
		}
	}()
	prev := 0
	for _, r := range s.renames {
		buf.Write(p[prev:r.at])
		for i := 0; i < r.depth; i++ {
			buf.WriteString("attr_")
		}
		prev = r.at
	}
	buf.Write(p[prev:])
	if _, err := d.w.Write(buf.Bytes()); err != nil {
		return 0, err
	}
	return len(p), nil
}
