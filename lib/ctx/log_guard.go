package ctx

import (
	"context"
	"log/slog"
)

// keyGuard keeps every JSON object a wrapped handler writes free of repeated
// keys. slog does not deduplicate, and downstream log agents drop or corrupt
// lines that repeat a key.
//
// A colliding key is renamed to attr_<key> (attr_attr_<key> if that is taken
// too), so the earlier value, such as the built-in source, always survives.
type keyGuard struct {
	inner slog.Handler
	// levels[i] holds the keys already written at object depth i; the last is
	// the current level. Sets are never mutated once stored.
	levels []keySet
}

type keySet map[string]struct{}

func newKeyGuard(inner slog.Handler) slog.Handler {
	return &keyGuard{
		inner: inner,
		levels: []keySet{{
			slog.TimeKey:    {},
			slog.LevelKey:   {},
			slog.MessageKey: {},
			slog.SourceKey:  {},
		}},
	}
}

func (g *keyGuard) Enabled(ctx context.Context, l slog.Level) bool {
	return g.inner.Enabled(ctx, l)
}

func (g *keyGuard) current() keySet { return g.levels[len(g.levels)-1] }

// withCurrent returns the levels with the current set replaced by cur.
func (g *keyGuard) withCurrent(cur keySet, extra ...keySet) []keySet {
	levels := make([]keySet, 0, len(g.levels)+len(extra))
	levels = append(levels, g.levels[:len(g.levels)-1]...)
	levels = append(levels, cur)
	return append(levels, extra...)
}

func (g *keyGuard) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return g
	}
	added := keySet{}
	out, _ := renameAttrs(attrs, g.current(), added)
	return &keyGuard{
		inner:  g.inner.WithAttrs(out),
		levels: g.withCurrent(union(g.current(), added)),
	}
}

func (g *keyGuard) WithGroup(name string) slog.Handler {
	if name == "" {
		return g
	}
	added := keySet{}
	name = uniqueKey(name, g.current(), added)
	return &keyGuard{
		inner:  g.inner.WithGroup(name),
		levels: g.withCurrent(union(g.current(), added), keySet{}),
	}
}

func (g *keyGuard) Handle(ctx context.Context, r slog.Record) error {
	attrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	out, changed := renameAttrs(attrs, g.current(), keySet{})
	if !changed {
		return g.inner.Handle(ctx, r)
	}
	nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	nr.AddAttrs(out...)
	return g.inner.Handle(ctx, nr)
}

// renameAttrs renames attrs whose key is in taken or in added, the keys used
// earlier in the same batch, and records the final keys in added. Nested
// groups are objects of their own and start from an empty set. It returns
// attrs itself when nothing was renamed.
func renameAttrs(attrs []slog.Attr, taken, added keySet) (out []slog.Attr, changed bool) {
	out = make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		a.Value = a.Value.Resolve()
		isGroup := a.Value.Kind() == slog.KindGroup
		switch {
		case isGroup && a.Key == "":
			members, c := renameAttrs(a.Value.Group(), taken, added)
			if c {
				a = slog.Attr{Key: "", Value: slog.GroupValue(members...)}
				changed = true
			}
		case a.Key == "":
			// slog drops it.
		default:
			key := uniqueKey(a.Key, taken, added)
			if isGroup {
				members, c := renameAttrs(a.Value.Group(), nil, keySet{})
				if c {
					a.Value = slog.GroupValue(members...)
					changed = true
				}
			}
			if key != a.Key {
				a.Key = key
				changed = true
			}
		}
		out[i] = a
	}
	if !changed {
		return attrs, false
	}
	return out, true
}

// uniqueKey returns key, prefixed with attr_ until it is in neither taken nor
// added, and records the result in added.
func uniqueKey(key string, taken, added keySet) string {
	for {
		_, inTaken := taken[key]
		_, inAdded := added[key]
		if !inTaken && !inAdded {
			break
		}
		key = "attr_" + key
	}
	added[key] = struct{}{}
	return key
}

func union(a, b keySet) keySet {
	u := make(keySet, len(a)+len(b))
	for k := range a {
		u[k] = struct{}{}
	}
	for k := range b {
		u[k] = struct{}{}
	}
	return u
}
