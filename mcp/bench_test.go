package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// These benchmarks measure the MCP layer, not the engine.
//
// The engine's own throughput is covered by the root package's benchmarks. What
// matters here is the overhead this module adds on top: JSON-RPC decoding,
// argument binding, and encoding the result back out. If that overhead is a
// large fraction of the total, the wrapper is the problem rather than the
// generator, and the number is the only way to know.

var benchRPC = mustJSON(map[string]any{
	"jsonrpc": "2.0",
	"id":      1,
	"method":  "tools/call",
	"params": map[string]any{
		"name":      "generate",
		"arguments": map[string]any{"preset": "user", "rows": 100, "seed": 1},
	},
})

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// BenchmarkGenerateHandler is the handler alone: engine plus argument checks.
func BenchmarkGenerateHandler(b *testing.B) {
	args := generateArgs{Preset: "user", Rows: 100, Seed: 1}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := handleGenerate(args); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateOverRPC is the same work through the protocol. The gap
// between this and BenchmarkGenerateHandler is what MCP costs.
func BenchmarkGenerateOverRPC(b *testing.B) {
	s := New()
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if s.HandleMessage(ctx, benchRPC) == nil {
			b.Fatal("no response")
		}
	}
}

// Server construction happens once per client session, but it registers seven
// tools and builds their schemas — worth knowing it is not seconds.
func BenchmarkNewServer(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		New()
	}
}

// The user preset costs about 19ms per 100 rows, and 96% of that is one column:
// password_hash runs PBKDF2 at 1000 iterations, which is a key derivation
// function and is meant to be slow. The same shape without it generates in
// 194µs — roughly 500,000 rows per second.
//
// These two sit next to each other so nobody reads the preset number as Synth's
// throughput and concludes the engine is slow.
func BenchmarkPresetWithPasswordHash(b *testing.B) {
	args := generateArgs{Preset: "user", Rows: 100, Seed: 1}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := handleGenerate(args); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSameShapeWithoutPasswordHash(b *testing.B) {
	spec := "name: t\nfields:\n" +
		"  id: { kind: uuid }\n" +
		"  full_name: { kind: name }\n" +
		"  email: { kind: email }\n" +
		"  phone: { kind: phone }\n" +
		"  city: { kind: city }\n" +
		"  country: { kind: country }\n"
	args := generateArgs{Spec: spec, Rows: 100, Seed: 1}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := handleGenerate(args); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateRows(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(rowLabel(n), func(b *testing.B) {
			// A cheap spec on purpose: this measures how the engine scales with
			// row count, and a KDF column would drown that out entirely.
			args := generateArgs{Spec: cheapSpec, Rows: n, Seed: 1}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := handleGenerate(args); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// cheapSpec has no key-derivation column, so a benchmark using it measures the
// engine rather than PBKDF2.
const cheapSpec = "name: t\nfields:\n" +
	"  id: { kind: uuid }\n" +
	"  full_name: { kind: name }\n" +
	"  email: { kind: email }\n" +
	"  city: { kind: city }\n"

func rowLabel(n int) string {
	switch n {
	case 1:
		return "1row"
	case 10:
		return "10rows"
	case 100:
		return "100rows"
	default:
		return "1000rows"
	}
}

// The catalog is around 250 entries and is returned whole. If listing it were
// expensive, a model calling it before every generate would pay for it every
// time.
func BenchmarkListTypes(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := handleListTypes(listTypesArgs{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	data := benchCSV(1000)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		if _, err := handleVerify(verifyArgs{Data: data}); err != nil {
			b.Fatal(err)
		}
	}
}

// Profile is the slowest tool: it makes two passes over the data, one for
// statistics and one for constraint mining. Worth measuring so the cost is a
// known number rather than a surprise on a large paste.
func BenchmarkProfile(b *testing.B) {
	data := benchCSV(1000)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		if _, err := handleProfile(profileArgs{Data: data}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMask(b *testing.B) {
	data := benchCSV(1000)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		if _, err := handleMask(maskArgs{Data: data, Key: "k"}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSnapshotAt(b *testing.B) {
	args := snapshotArgs{Spec: snapSpec, Rows: 1000, Seed: 1, At: "2026-07-01"}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := handleSnapshot(args); err != nil {
			b.Fatal(err)
		}
	}
}

// benchCSV builds n rows of realistic-looking data to feed the reading tools.
func benchCSV(n int) string {
	rows, err := handleGenerate(generateArgs{Preset: "user", Rows: n, Seed: 1})
	if err != nil {
		panic(err)
	}
	cols := []string{"id", "full_name", "email", "phone", "city", "country"}
	var b strings.Builder
	b.WriteString(strings.Join(cols, ","))
	b.WriteByte('\n')
	for _, r := range rows.([]map[string]any) {
		for i, c := range cols {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strings.ReplaceAll(toString(r[c]), ",", " "))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return strings.Trim(string(b), `"`)
}
