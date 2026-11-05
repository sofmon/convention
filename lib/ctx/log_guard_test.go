package ctx

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"

	convAuth "github.com/sofmon/convention/lib/auth"
)

// decodeStrict decodes one JSON object and fails on a repeated key at any level.
func decodeStrict(t *testing.T, line string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(line))
	v := readValue(t, dec)
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("not an object: %s", line)
	}
	return m
}

func readValue(t *testing.T, dec *json.Decoder) any {
	t.Helper()
	tok, err := dec.Token()
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return tok
	}
	switch d {
	case '{':
		m := map[string]any{}
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				t.Fatalf("decode key: %v", err)
			}
			k := kt.(string)
			if _, dup := m[k]; dup {
				t.Fatalf("repeated key %q", k)
			}
			m[k] = readValue(t, dec)
		}
		if _, err := dec.Token(); err != nil {
			t.Fatal(err)
		}
		return m
	case '[':
		var a []any
		for dec.More() {
			a = append(a, readValue(t, dec))
		}
		if _, err := dec.Token(); err != nil {
			t.Fatal(err)
		}
		return a
	}
	t.Fatalf("unexpected delim %v", d)
	return nil
}

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(newKeyGuard(slog.NewJSONHandler(buf, &slog.HandlerOptions{AddSource: true})))
}

func logOne(t *testing.T, build func(l *slog.Logger) *slog.Logger, log func(l *slog.Logger)) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	l := newTestLogger(&buf)
	if build != nil {
		l = build(l)
	}
	log(l)
	return decodeStrict(t, strings.TrimSpace(buf.String()))
}

func TestKeyGuard_BuiltInKeysRenamed(t *testing.T) {
	for _, k := range []string{"time", "level", "msg", "source"} {
		t.Run(k, func(t *testing.T) {
			m := logOne(t, nil, func(l *slog.Logger) { l.Info("m", k, "x") })
			if m["attr_"+k] != "x" {
				t.Fatalf("attr_%s = %v", k, m["attr_"+k])
			}
			switch k {
			case "source":
				if _, ok := m["source"].(map[string]any); !ok {
					t.Fatalf("source not the built-in object: %v", m["source"])
				}
			case "level":
				if m["level"] != "INFO" {
					t.Fatalf("level = %v", m["level"])
				}
			case "msg":
				if m["msg"] != "m" {
					t.Fatalf("msg = %v", m["msg"])
				}
			case "time":
				if s, _ := m["time"].(string); s == "x" {
					t.Fatal("time overwritten")
				}
			}
		})
	}
}

func TestKeyGuard_ConventionKeys(t *testing.T) {
	var buf bytes.Buffer
	c := New(convAuth.Claims{User: "u"}).
		WithLogger(newTestLogger(&buf)).
		WithScope("sc").
		WithAction(convAuth.Action("GET /x"))
	c.Logger().Info("m", "action", "put", "scope", "s")
	m := decodeStrict(t, strings.TrimSpace(buf.String()))
	if m["action"] != "GET /x" || m["attr_action"] != "put" {
		t.Fatalf("action: %v / %v", m["action"], m["attr_action"])
	}
	if m["scope"] == "s" || m["attr_scope"] != "s" {
		t.Fatalf("scope: %v / %v", m["scope"], m["attr_scope"])
	}
}

func TestKeyGuard_RepeatedKeyInRecord(t *testing.T) {
	m := logOne(t, nil, func(l *slog.Logger) { l.Info("m", "k", 1, "k", 2) })
	if m["k"] != float64(1) || m["attr_k"] != float64(2) {
		t.Fatalf("got %v", m)
	}
}

func TestKeyGuard_WithAttrsChainedRename(t *testing.T) {
	m := logOne(t,
		func(l *slog.Logger) *slog.Logger { return l.With("source", 1) },
		func(l *slog.Logger) { l.Info("m", "attr_source", 2, "source", 3) })
	if _, ok := m["source"].(map[string]any); !ok {
		t.Fatalf("source: %v", m["source"])
	}
	if m["attr_source"] != float64(1) {
		t.Fatalf("attr_source = %v", m["attr_source"])
	}
	if m["attr_attr_source"] != float64(2) {
		t.Fatalf("attr_attr_source = %v", m["attr_attr_source"])
	}
	if m["attr_attr_attr_source"] != float64(3) {
		t.Fatalf("attr_attr_attr_source = %v", m["attr_attr_attr_source"])
	}
}

func TestKeyGuard_WithAttrsDuplicate(t *testing.T) {
	m := logOne(t,
		func(l *slog.Logger) *slog.Logger { return l.With("k", 1).With("k", 2) },
		func(l *slog.Logger) { l.Info("m", "k", 3) })
	if m["k"] != float64(1) || m["attr_k"] != float64(2) || m["attr_attr_k"] != float64(3) {
		t.Fatalf("got %v", m)
	}
}

func TestKeyGuard_WithGroup(t *testing.T) {
	m := logOne(t,
		func(l *slog.Logger) *slog.Logger { return l.WithGroup("source") },
		func(l *slog.Logger) { l.Info("m", "a", 1) })
	g, ok := m["attr_source"].(map[string]any)
	if !ok || g["a"] != float64(1) {
		t.Fatalf("got %v", m)
	}
	if _, ok := m["source"].(map[string]any); !ok {
		t.Fatalf("built-in source missing: %v", m["source"])
	}
}

func TestKeyGuard_AttrsInsideGroupAreNotTopLevel(t *testing.T) {
	m := logOne(t,
		func(l *slog.Logger) *slog.Logger { return l.WithGroup("g") },
		func(l *slog.Logger) { l.Info("m", "source", 1, "msg", 2) })
	g := m["g"].(map[string]any)
	if g["source"] != float64(1) || g["msg"] != float64(2) {
		t.Fatalf("got %v", m)
	}
}

func TestKeyGuard_WithAttrsInsideGroup(t *testing.T) {
	m := logOne(t,
		func(l *slog.Logger) *slog.Logger { return l.WithGroup("g").With("k", 1) },
		func(l *slog.Logger) { l.Info("m", "k", 2) })
	g := m["g"].(map[string]any)
	if g["k"] != float64(1) || g["attr_k"] != float64(2) {
		t.Fatalf("got %v", m)
	}
}

func TestKeyGuard_NestedGroup(t *testing.T) {
	m := logOne(t, nil, func(l *slog.Logger) {
		l.Info("m", slog.Group("g", "source", 1))
	})
	if m["g"].(map[string]any)["source"] != float64(1) {
		t.Fatalf("nested renamed: %v", m)
	}

	m = logOne(t, nil, func(l *slog.Logger) {
		l.Info("m", slog.Group("g", "k", 1, "k", 2))
	})
	g := m["g"].(map[string]any)
	if g["k"] != float64(1) || g["attr_k"] != float64(2) {
		t.Fatalf("nested dup: %v", m)
	}

	m = logOne(t, nil, func(l *slog.Logger) {
		l.Info("m", "g", 1, slog.Group("g", "a", 1))
	})
	if m["g"] != float64(1) || m["attr_g"].(map[string]any)["a"] != float64(1) {
		t.Fatalf("group key collision: %v", m)
	}
}

func TestKeyGuard_InlinedEmptyKeyGroup(t *testing.T) {
	m := logOne(t, nil, func(l *slog.Logger) {
		l.Info("m", slog.Group("", "source", 1, "k", 2), "k", 3)
	})
	if _, ok := m["source"].(map[string]any); !ok {
		t.Fatalf("source: %v", m["source"])
	}
	if m["attr_source"] != float64(1) || m["k"] != float64(2) || m["attr_k"] != float64(3) {
		t.Fatalf("got %v", m)
	}
}

type groupValuer struct{}

func (groupValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.Int("source", 1), slog.Int("source", 2))
}

func TestKeyGuard_LogValuerGroup(t *testing.T) {
	m := logOne(t, nil, func(l *slog.Logger) { l.Info("m", slog.Any("", groupValuer{})) })
	if m["attr_source"] != float64(1) || m["attr_attr_source"] != float64(2) {
		t.Fatalf("inlined valuer: %v", m)
	}
	m = logOne(t, nil, func(l *slog.Logger) { l.Info("m", "v", groupValuer{}) })
	v := m["v"].(map[string]any)
	if v["source"] != float64(1) || v["attr_source"] != float64(2) {
		t.Fatalf("nested valuer: %v", m)
	}
}

func TestKeyGuard_NoCollisionIsByteIdentical(t *testing.T) {
	drop := func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) == 0 && a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	opts := &slog.HandlerOptions{AddSource: true, ReplaceAttr: drop}
	emit := func(h slog.Handler) {
		l := slog.New(h).With("a", 1).WithGroup("g").With("b", 2)
		l.Info("m", "c", 3, slog.Group("n", "d", 4), slog.Group("", "e", 5))
	}
	var plain, guarded bytes.Buffer
	emit(slog.NewJSONHandler(&plain, opts))
	emit(newKeyGuard(slog.NewJSONHandler(&guarded, opts)))
	if plain.String() != guarded.String() {
		t.Fatalf("differs:\n%s\n%s", plain.String(), guarded.String())
	}
}

func TestKeyGuard_EnabledDelegates(t *testing.T) {
	h := newKeyGuard(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if h.Enabled(context.Background(), slog.LevelInfo) || !h.Enabled(context.Background(), slog.LevelError) {
		t.Fatal("Enabled does not delegate")
	}
}

func TestKeyGuard_WithGroupEmptyReturnsReceiver(t *testing.T) {
	h := newKeyGuard(slog.NewJSONHandler(io.Discard, nil))
	if h.WithGroup("") != h {
		t.Fatal("WithGroup(\"\") must return the receiver")
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func TestKeyGuardConcurrency(t *testing.T) {
	var out lockedBuffer
	base := slog.New(newKeyGuard(slog.NewJSONHandler(&out, &slog.HandlerOptions{AddSource: true})))
	withA := base.With("source", 0)
	withG := base.WithGroup("g").With("k", 0)

	const workers, per = 32, 50
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			own := withA.With("k", i).WithGroup("source")
			for j := 0; j < per; j++ {
				withA.Info("m", "source", j, "k", j, "k", j)
				withG.Info("m", "k", j, "msg", j)
				own.Info("m", "k", j, slog.Group("x", "k", j, "k", j))
			}
		}()
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(out.buf.String()), "\n")
	if len(lines) != workers*per*3 {
		t.Fatalf("got %d lines", len(lines))
	}
	for _, line := range lines {
		decodeStrict(t, line)
	}
}

func TestLoggerKeys(t *testing.T) {
	want := []string{"env", "agent", "workflow", "user", "scope", "action", "now", "use_agent_claims"}
	got := LoggerKeys()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range got {
		got[i] = "mutated"
	}
	if !reflect.DeepEqual(LoggerKeys(), want) {
		t.Fatal("mutating the result affected the next call")
	}
}
