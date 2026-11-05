package ctx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	convAuth "github.com/sofmon/convention/lib/auth"
)

// decodeStrict decodes one JSON object and fails on a repeated key at any level.
func decodeStrict(t *testing.T, line string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(line))
	m, ok := readValue(t, dec).(map[string]any)
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

// logOne logs one line through newDefaultLogger and decodes it strictly.
func logOne(t *testing.T, build func(l *slog.Logger) *slog.Logger, log func(l *slog.Logger)) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	l := newDefaultLogger(&buf)
	if build != nil {
		l = build(l)
	}
	log(l)
	if strings.Count(buf.String(), "\n") != 1 {
		t.Fatalf("want exactly one line, got %q", buf.String())
	}
	return decodeStrict(t, strings.TrimSpace(buf.String()))
}

func TestDedupe_BuiltInKeysRenamed(t *testing.T) {
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
				if m["time"] == "x" {
					t.Fatal("time overwritten")
				}
			}
		})
	}
}

func TestDedupe_ContextLoggerKeys(t *testing.T) {
	var buf bytes.Buffer
	c := New(convAuth.Claims{User: "u"}).
		WithLogger(newDefaultLogger(&buf)).
		WithScope("sc").
		WithAction(convAuth.Action("GET /x"))
	c.Logger().Info("m", "action", "put", "scope", "s", "user", "other", "env", "e")
	m := decodeStrict(t, strings.TrimSpace(buf.String()))
	if m["action"] != "GET /x" || m["attr_action"] != "put" {
		t.Fatalf("action: %v / %v", m["action"], m["attr_action"])
	}
	if m["scope"] == "s" || m["attr_scope"] != "s" {
		t.Fatalf("scope: %v / %v", m["scope"], m["attr_scope"])
	}
	if m["user"] != "u" || m["attr_user"] != "other" || m["attr_env"] != "e" {
		t.Fatalf("user/env: %v", m)
	}
}

func TestDedupe_DefaultLoggerIsDeduplicated(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defaultLogger().Info("m", "source", "x", "k", 1, "k", 2)
	os.Stdout = orig
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	m := decodeStrict(t, strings.TrimSpace(string(out)))
	if m["attr_source"] != "x" || m["attr_k"] != float64(2) {
		t.Fatalf("got %v", m)
	}
}

func TestDedupe_RepeatedKeyInRecord(t *testing.T) {
	m := logOne(t, nil, func(l *slog.Logger) { l.Info("m", "k", 1, "k", 2) })
	if m["k"] != float64(1) || m["attr_k"] != float64(2) {
		t.Fatalf("got %v", m)
	}
}

func TestDedupe_WithChains(t *testing.T) {
	m := logOne(t,
		func(l *slog.Logger) *slog.Logger { return l.With("source", 1) },
		func(l *slog.Logger) { l.Info("m", "attr_source", 2, "source", 3) })
	if _, ok := m["source"].(map[string]any); !ok {
		t.Fatalf("source: %v", m["source"])
	}
	if m["attr_source"] != float64(1) || m["attr_attr_source"] != float64(2) || m["attr_attr_attr_source"] != float64(3) {
		t.Fatalf("got %v", m)
	}

	m = logOne(t,
		func(l *slog.Logger) *slog.Logger { return l.With("k", 1).With("k", 2) },
		func(l *slog.Logger) { l.Info("m", "k", 3) })
	if m["k"] != float64(1) || m["attr_k"] != float64(2) || m["attr_attr_k"] != float64(3) {
		t.Fatalf("got %v", m)
	}
}

func TestDedupe_WithGroup(t *testing.T) {
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

	m = logOne(t,
		func(l *slog.Logger) *slog.Logger { return l.WithGroup("g") },
		func(l *slog.Logger) { l.Info("m", "source", 1, "msg", 2) })
	g = m["g"].(map[string]any)
	if g["source"] != float64(1) || g["msg"] != float64(2) {
		t.Fatalf("attributes inside a group are not top level: %v", m)
	}

	m = logOne(t,
		func(l *slog.Logger) *slog.Logger { return l.WithGroup("g").With("k", 1) },
		func(l *slog.Logger) { l.Info("m", "k", 2) })
	g = m["g"].(map[string]any)
	if g["k"] != float64(1) || g["attr_k"] != float64(2) {
		t.Fatalf("got %v", m)
	}
}

func TestDedupe_NestedGroups(t *testing.T) {
	m := logOne(t, nil, func(l *slog.Logger) { l.Info("m", slog.Group("g", "source", 1)) })
	if m["g"].(map[string]any)["source"] != float64(1) {
		t.Fatalf("nested key renamed: %v", m)
	}

	m = logOne(t, nil, func(l *slog.Logger) { l.Info("m", slog.Group("g", "k", 1, "k", 2)) })
	g := m["g"].(map[string]any)
	if g["k"] != float64(1) || g["attr_k"] != float64(2) {
		t.Fatalf("nested dup: %v", m)
	}

	m = logOne(t, nil, func(l *slog.Logger) { l.Info("m", "g", 1, slog.Group("g", "a", 1)) })
	if m["g"] != float64(1) || m["attr_g"].(map[string]any)["a"] != float64(1) {
		t.Fatalf("group key collision: %v", m)
	}
}

func TestDedupe_InlinedEmptyKeyGroup(t *testing.T) {
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

type countingValuer struct{ calls *int }

func (v countingValuer) LogValue() slog.Value {
	*v.calls++
	return slog.StringValue("v")
}

func TestDedupe_LogValuerRunsOncePerLine(t *testing.T) {
	var plainCalls, ourCalls int
	var plain bytes.Buffer
	plainLogger := slog.New(slog.NewJSONHandler(&plain, &slog.HandlerOptions{AddSource: true}))
	plainLogger.Info("m", "v", countingValuer{&plainCalls})

	m := logOne(t, nil, func(l *slog.Logger) { l.Info("m", "v", countingValuer{&ourCalls}) })
	if plainCalls != 1 || ourCalls != plainCalls {
		t.Fatalf("calls: plain=%d ours=%d", plainCalls, ourCalls)
	}
	if m["v"] != "v" {
		t.Fatalf("got %v", m)
	}
}

// changingValuer returns a clean group on its first call and a group with a
// repeated key on later ones.
type changingValuer struct{ calls *int }

func (v changingValuer) LogValue() slog.Value {
	*v.calls++
	if *v.calls == 1 {
		return slog.GroupValue(slog.Int("a", 1))
	}
	return slog.GroupValue(slog.Int("a", 1), slog.Int("a", 2))
}

func TestDedupe_LogValuerChangingResult(t *testing.T) {
	var calls int
	m := logOne(t, nil, func(l *slog.Logger) { l.Info("m", "v", changingValuer{&calls}) })
	if calls != 1 {
		t.Fatalf("valuer ran %d times", calls)
	}
	if m["v"].(map[string]any)["a"] != float64(1) {
		t.Fatalf("got %v", m)
	}
}

type groupValuer struct{}

func (groupValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.Int("source", 1), slog.Int("source", 2))
}

func TestDedupe_LogValuerGroup(t *testing.T) {
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

func TestDedupe_EmptyKeyIsNotDropped(t *testing.T) {
	m := logOne(t,
		func(l *slog.Logger) *slog.Logger { return l.With("", 1) },
		func(l *slog.Logger) { l.Info("m", "", 2) })
	if m[""] != float64(1) || m["attr_"] != float64(2) {
		t.Fatalf("got %v", m)
	}
}

func TestDedupe_EmptyKeySourceValueIsInlined(t *testing.T) {
	m := logOne(t, nil, func(l *slog.Logger) {
		l.Info("m", slog.Any("", &slog.Source{Function: "f"}), "function", "x")
	})
	if m["function"] != "f" || m["attr_function"] != "x" {
		t.Fatalf("got %v", m)
	}
}

func TestDedupe_InvalidUTF8KeysCollide(t *testing.T) {
	m := logOne(t, nil, func(l *slog.Logger) { l.Info("m", "\xff", 1, "\xfe", 2) })
	if m["�"] != float64(1) || m["attr_�"] != float64(2) {
		t.Fatalf("got %v", m)
	}
}

func TestDedupe_RawMessageWithRepeatedKey(t *testing.T) {
	m := logOne(t, nil, func(l *slog.Logger) {
		l.Info("m", "r", json.RawMessage(`{"k":1,"k":2}`))
	})
	r := m["r"].(map[string]any)
	if r["k"] != float64(1) || r["attr_k"] != float64(2) {
		t.Fatalf("got %v", m)
	}
}

func TestDedupe_AttributesThatEmitNothingDoNotReserveKeys(t *testing.T) {
	m := logOne(t, nil, func(l *slog.Logger) { l.Info("m", slog.Group("g"), "g", 1) })
	if m["g"] != float64(1) {
		t.Fatalf("got %v", m)
	}
	if _, ok := m["attr_g"]; ok {
		t.Fatalf("empty group reserved its key: %v", m)
	}

	var buf bytes.Buffer
	l := newDefaultLogger(&buf)
	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "m", 0)
	r.AddAttrs(slog.String("time", "x"))
	if err := l.Handler().Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	m = decodeStrict(t, strings.TrimSpace(buf.String()))
	if m["time"] != "x" {
		t.Fatalf("omitted zero time reserved its key: %v", m)
	}
	if _, ok := m["attr_time"]; ok {
		t.Fatalf("got %v", m)
	}
}

func TestDedupe_CollisionFreeIsByteIdentical(t *testing.T) {
	var calls int
	emit := func(h slog.Handler) {
		r := slog.NewRecord(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), slog.LevelInfo, "m <&>", 0)
		r.AddAttrs(
			slog.Int("c", 3),
			slog.Group("n", "d", 4),
			slog.Group("", "e", 5),
			slog.Any("v", countingValuer{&calls}),
			slog.Float64("f", 1.50),
			slog.String("html", "<a href='x'>&</a>"),
		)
		l := slog.New(h).With("a", 1).WithGroup("g").With("b", 2)
		if err := l.Handler().Handle(context.Background(), r); err != nil {
			t.Fatal(err)
		}
	}
	var plain, ours bytes.Buffer
	emit(slog.NewJSONHandler(&plain, &slog.HandlerOptions{Level: slog.LevelInfo, AddSource: true}))
	emit(newDefaultLogger(&ours).Handler())
	if plain.String() != ours.String() {
		t.Fatalf("differs:\n%s\n%s", plain.String(), ours.String())
	}
}

func TestDedupeWriter(t *testing.T) {
	write := func(in string) (string, int, error) {
		var out bytes.Buffer
		n, err := newDedupeWriter(&out).Write([]byte(in))
		return out.String(), n, err
	}
	unchanged := []string{
		"not json\n",
		"",
		"[1,2]\n",
		`{"a":1}{"b":2}` + "\n",
		`{"a":1,"a":2` + "\n",
		`{"a":1,"a":2} trailing` + "\n",
		`{"a":1,"b":{"c":1,"d":1},"e":[{"f":1},{"f":2}]}` + "\n",
	}
	for _, in := range unchanged {
		got, n, err := write(in)
		if got != in || n != len(in) || err != nil {
			t.Errorf("%q: got %q n=%d err=%v", in, got, n, err)
		}
	}

	rewritten := map[string]string{
		`{"a":1,"a":2}` + "\n":                                           `{"a":1,"attr_a":2}` + "\n",
		`{"a":1,"attr_a":2,"a":3}` + "\n":                                `{"a":1,"attr_a":2,"attr_attr_a":3}` + "\n",
		`{"a":1,"a":2,"attr_a":3}` + "\n":                                `{"a":1,"attr_a":2,"attr_attr_a":3}` + "\n",
		`{"o":{"k":1,"k":2},"k":1}` + "\n":                               `{"o":{"k":1,"attr_k":2},"k":1}` + "\n",
		`{"l":[{"k":1,"k":2},[{"k":3,"k":4}]],"n":null}` + "\n":          `{"l":[{"k":1,"attr_k":2},[{"k":3,"attr_k":4}]],"n":null}` + "\n",
		`{"a":12345678901234567890,"a":1.50,"b":true,"c":-0.0e5}` + "\n": `{"a":12345678901234567890,"attr_a":1.50,"b":true,"c":-0.0e5}` + "\n",
		`{"h":"<&>","h":"é\n"}` + "\n":                                   `{"h":"<&>","attr_h":"é\n"}` + "\n",
	}
	for in, want := range rewritten {
		got, n, err := write(in)
		if got != want || n != len(in) || err != nil {
			t.Errorf("%q: got %q n=%d err=%v, want %q", in, got, n, err, want)
		}
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestDedupeWriter_PropagatesDestinationErrors(t *testing.T) {
	boom := errors.New("boom")
	for _, in := range []string{`{"a":1}` + "\n", `{"a":1,"a":2}` + "\n"} {
		if _, err := newDedupeWriter(failingWriter{boom}).Write([]byte(in)); !errors.Is(err, boom) {
			t.Errorf("%q: err = %v", in, err)
		}
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

func TestDedupeConcurrency(t *testing.T) {
	var out lockedBuffer
	base := newDefaultLogger(&out)
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

func benchRecord() slog.Record {
	r := slog.NewRecord(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), slog.LevelInfo, "request handled", 0)
	r.AddAttrs(slog.String("path", "/entity/v1/x"), slog.Int("status", 200), slog.Duration("took", time.Millisecond), slog.Group("req", "id", "abc", "size", 12))
	return r
}

func benchHandler(b *testing.B, h slog.Handler, r slog.Record) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		if err := h.Handle(ctx, r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDefaultLoggerNoCollision(b *testing.B) {
	h := newDefaultLogger(io.Discard).With("env", "e", "agent", "a").Handler()
	benchHandler(b, h, benchRecord())
}

func BenchmarkPlainJSONHandlerNoCollision(b *testing.B) {
	h := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo, AddSource: true}).
		WithAttrs([]slog.Attr{slog.String("env", "e"), slog.String("agent", "a")})
	benchHandler(b, h, benchRecord())
}

func BenchmarkDefaultLoggerCollision(b *testing.B) {
	h := newDefaultLogger(io.Discard).With("env", "e", "agent", "a").Handler()
	r := benchRecord()
	r.AddAttrs(slog.String("source", "x"), slog.String("env", "y"))
	benchHandler(b, h, r)
}
