package ctx

import (
	"bytes"
	"strings"
	"sync"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	maxScanDepth = 512   // fast mode; deeper lines go to the slow mode
	maxDepth     = 10000 // slow mode, the limit of encoding/json
	maxScanKeys  = 64    // per object in fast mode; bigger objects go to the slow mode
)

// scanner walks one JSON object without decoding values, in one of two modes.
//
// Fast mode is allocation-free: it answers whether the line is valid and
// repeats no key, comparing raw key bytes, and answers false whenever it cannot
// be sure (keys holding an escape or invalid UTF-8, big or deep objects,
// malformed input).
//
// Slow mode compares keys by their decoded value, as encoding/json does, and
// records where each repeated key needs an attr_ prefix. It never changes a
// value, and a key keeps its own escapes.
type scanner struct {
	p        []byte
	i        int
	slow     bool
	maxDepth int
	keys     [][2]int // fast: spans into p of the keys of the open objects
	renames  []rename // slow: in document order
	esc      bool     // the last string held a backslash
	hiBit    bool     // the last string held a byte >= 0x80
}

// rename says: insert depth repetitions of "attr_" at offset at, just inside
// the opening quote of a key.
type rename struct{ at, depth int }

var scanners = sync.Pool{New: func() any { return new(scanner) }}

func (s *scanner) reset(p []byte, slow bool) {
	s.p, s.i, s.slow = p, 0, slow
	s.keys, s.renames = s.keys[:0], s.renames[:0]
	s.maxDepth = maxScanDepth
	if slow {
		s.maxDepth = maxDepth
	}
}

// scan reports whether p is exactly one JSON object, with optional surrounding
// whitespace. In fast mode it also requires that no object repeats a key.
func (s *scanner) scan(p []byte, slow bool) bool {
	s.reset(p, slow)
	return s.skipSpace() && s.p[s.i] == '{' && s.object(0) && !s.skipSpace()
}

// cleanLine reports whether p is exactly one JSON object and no object in it
// repeats a key, as far as the fast scan can tell.
func cleanLine(p []byte) bool {
	s := scanners.Get().(*scanner)
	ok := s.scan(p, false)
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
	if depth > s.maxDepth {
		return false
	}
	s.i++ // {
	base := len(s.keys)
	var names keyNames

	if s.skipSpace() && s.p[s.i] == '}' {
		s.i++
		return true
	}
	for {
		if !s.skipSpace() || s.p[s.i] != '"' {
			return false
		}
		start, end, ok := s.str()
		if !ok {
			return false
		}
		if s.slow {
			if depth := names.add(decodeKey(s.p[start:end])); depth > 0 {
				s.renames = append(s.renames, rename{start, depth})
			}
		} else {
			if s.esc || (s.hiBit && !utf8.Valid(s.p[start:end])) || len(s.keys)-base >= maxScanKeys {
				return false
			}
			key := s.p[start:end]
			for _, k := range s.keys[base:] {
				if bytes.Equal(key, s.p[k[0]:k[1]]) {
					return false
				}
			}
			s.keys = append(s.keys, [2]int{start, end})
		}

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
	if depth > s.maxDepth {
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

// keyNames tracks the decoded keys of one object in slow mode.
type keyNames struct {
	seen map[string]struct{}
	// next maps a decoded key to the first attr_ depth not yet known to be
	// taken, so each repeat costs one candidate check however many came before.
	next map[string]int
}

// add records key and returns how many attr_ prefixes it needs to be unique
// among the keys of the object (0 for none).
func (n *keyNames) add(key string) int {
	if n.seen == nil {
		n.seen, n.next = map[string]struct{}{}, map[string]int{}
	}
	if _, taken := n.seen[key]; !taken {
		n.seen[key] = struct{}{}
		return 0
	}
	depth := n.next[key]
	if depth == 0 {
		depth = 1
	}
	for {
		candidate := strings.Repeat("attr_", depth) + key
		if _, taken := n.seen[candidate]; !taken {
			n.seen[candidate] = struct{}{}
			n.next[key] = depth + 1
			return depth
		}
		depth++
	}
}

// decodeKey decodes the inside of a JSON string exactly as encoding/json does:
// invalid UTF-8 and lone surrogates become U+FFFD. raw must be a string the
// scanner accepted, so its escapes are well formed.
func decodeKey(raw []byte) string {
	if bytes.IndexByte(raw, '\\') < 0 && utf8.Valid(raw) {
		return string(raw)
	}
	b := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); {
		c := raw[i]
		switch {
		case c == '\\' && raw[i+1] == 'u':
			r := hex4(raw[i+2 : i+6])
			i += 6
			if utf16.IsSurrogate(r) {
				paired := false
				if i+6 <= len(raw) && raw[i] == '\\' && raw[i+1] == 'u' {
					if dec := utf16.DecodeRune(r, hex4(raw[i+2:i+6])); dec != unicode.ReplacementChar {
						r, paired = dec, true
						i += 6
					}
				}
				if !paired {
					r = unicode.ReplacementChar
				}
			}
			b = utf8.AppendRune(b, r)
		case c == '\\':
			b = append(b, unescape(raw[i+1]))
			i += 2
		case c < utf8.RuneSelf:
			b = append(b, c)
			i++
		default:
			r, size := utf8.DecodeRune(raw[i:])
			b = utf8.AppendRune(b, r)
			i += size
		}
	}
	return string(b)
}

func unescape(c byte) byte {
	switch c {
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	}
	return c // " \ /
}

func hex4(h []byte) rune {
	var r rune
	for _, c := range h {
		switch {
		case c >= 'a':
			c -= 'a' - 10
		case c >= 'A':
			c -= 'A' - 10
		default:
			c -= '0'
		}
		r = r<<4 | rune(c)
	}
	return r
}
