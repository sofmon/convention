package ctx

import (
	"bytes"
	"sync"
	"unicode/utf8"
)

const (
	maxScanDepth = 512
	maxScanKeys  = 64 // per object; more keys are left to the slow path
)

// scanner is the allocation-free fast check for dedupeWriter. It walks one JSON
// object and reports whether it is valid and repeats no key. It answers false
// whenever it cannot be sure, and the caller then takes the slow path, which
// decides for real: keys holding an escape or invalid UTF-8 (equal keys can be
// spelled differently), big objects, deep nesting, and any malformed input.
type scanner struct {
	p     []byte
	i     int
	keys  [][2]int // spans into p of the keys of the open objects
	esc   bool     // the last string held a backslash
	hiBit bool     // the last string held a byte >= 0x80
}

var scanners = sync.Pool{New: func() any { return new(scanner) }}

// cleanLine reports whether p is exactly one JSON object, with optional
// surrounding whitespace, and no object in it repeats a key.
func cleanLine(p []byte) bool {
	s := scanners.Get().(*scanner)
	s.p, s.i, s.keys = p, 0, s.keys[:0]
	ok := s.skipSpace() && s.p[s.i] == '{' && s.object(0) && !s.skipSpace()
	s.p = nil // never retain the caller's buffer
	scanners.Put(s)
	return ok
}

func (s *scanner) peek() byte {
	if s.i < len(s.p) {
		return s.p[s.i]
	}
	return 0
}

// skipSpace advances over whitespace and reports whether input remains.
func (s *scanner) skipSpace() bool {
	for s.i < len(s.p) {
		switch s.p[s.i] {
		case ' ', '\t', '\r', '\n':
			s.i++
		default:
			return true
		}
	}
	return false
}

func (s *scanner) expect(c byte) bool {
	if s.skipSpace() && s.p[s.i] == c {
		s.i++
		return true
	}
	return false
}

func (s *scanner) value(depth int) bool {
	if !s.skipSpace() {
		return false
	}
	switch c := s.p[s.i]; {
	case c == '{':
		return s.object(depth + 1)
	case c == '[':
		return s.array(depth + 1)
	case c == '"':
		_, _, ok := s.str()
		return ok
	case c == 't':
		return s.literal("true")
	case c == 'f':
		return s.literal("false")
	case c == 'n':
		return s.literal("null")
	default:
		return s.number()
	}
}

func (s *scanner) object(depth int) bool {
	if depth > maxScanDepth {
		return false
	}
	s.i++ // {
	base := len(s.keys)

	if s.skipSpace() && s.p[s.i] == '}' {
		s.i++
		return true
	}
	for {
		if !s.skipSpace() || s.p[s.i] != '"' {
			return false
		}
		start, end, ok := s.str()
		if !ok || s.esc || (s.hiBit && !utf8.Valid(s.p[start:end])) || len(s.keys)-base >= maxScanKeys {
			return false
		}
		key := s.p[start:end]
		for _, k := range s.keys[base:] {
			if bytes.Equal(key, s.p[k[0]:k[1]]) {
				return false
			}
		}
		s.keys = append(s.keys, [2]int{start, end})

		if !s.expect(':') || !s.value(depth) {
			return false
		}
		if !s.skipSpace() {
			return false
		}
		c := s.p[s.i]
		s.i++
		if c == '}' {
			s.keys = s.keys[:base]
			return true
		}
		if c != ',' {
			return false
		}
	}
}

func (s *scanner) array(depth int) bool {
	if depth > maxScanDepth {
		return false
	}
	s.i++ // [
	if s.skipSpace() && s.p[s.i] == ']' {
		s.i++
		return true
	}
	for {
		if !s.value(depth) || !s.skipSpace() {
			return false
		}
		c := s.p[s.i]
		s.i++
		if c == ']' {
			return true
		}
		if c != ',' {
			return false
		}
	}
}

// str scans the string at s.i and returns the span between its quotes.
func (s *scanner) str() (start, end int, ok bool) {
	s.esc, s.hiBit = false, false
	s.i++ // opening quote
	start = s.i
	for s.i < len(s.p) {
		switch c := s.p[s.i]; {
		case c == '"':
			end = s.i
			s.i++
			return start, end, true
		case c == '\\':
			s.esc = true
			if s.i+1 >= len(s.p) {
				return 0, 0, false
			}
			switch s.p[s.i+1] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				s.i += 2
			case 'u':
				if s.i+6 > len(s.p) {
					return 0, 0, false
				}
				for _, h := range s.p[s.i+2 : s.i+6] {
					if !isHex(h) {
						return 0, 0, false
					}
				}
				s.i += 6
			default:
				return 0, 0, false
			}
		case c < 0x20:
			return 0, 0, false
		default:
			if c >= 0x80 {
				s.hiBit = true
			}
			s.i++
		}
	}
	return 0, 0, false
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func (s *scanner) literal(lit string) bool {
	if len(s.p)-s.i >= len(lit) && string(s.p[s.i:s.i+len(lit)]) == lit {
		s.i += len(lit)
		return true
	}
	return false
}

// number scans -?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?
func (s *scanner) number() bool {
	if s.peek() == '-' {
		s.i++
	}
	switch c := s.peek(); {
	case c == '0':
		s.i++
	case c >= '1' && c <= '9':
		s.digits()
	default:
		return false
	}
	if s.peek() == '.' {
		s.i++
		if !s.digits() {
			return false
		}
	}
	if c := s.peek(); c == 'e' || c == 'E' {
		s.i++
		if c := s.peek(); c == '+' || c == '-' {
			s.i++
		}
		if !s.digits() {
			return false
		}
	}
	return true
}

func (s *scanner) digits() bool {
	start := s.i
	for s.i < len(s.p) && s.p[s.i] >= '0' && s.p[s.i] <= '9' {
		s.i++
	}
	return s.i > start
}
