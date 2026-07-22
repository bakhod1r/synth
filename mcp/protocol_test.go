package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// These tests drive the server the way a client does — through HandleMessage
// with real JSON-RPC — rather than calling the handlers directly.
//
// The handler tests check the logic. These check the wiring: that arguments
// survive JSON binding with their types intact, that a tool error reaches the
// caller as a readable message rather than a dropped connection, and that a
// result encodes to something a model can actually read.

// call sends one tools/call and returns the decoded result.
func call(t *testing.T, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": name, "arguments": args},
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	msg := New().HandleMessage(context.Background(), body)
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Result *mcp.CallToolResult `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("cannot decode the response: %v\n%s", err, raw)
	}
	if envelope.Error != nil {
		t.Fatalf("the call failed at the transport level: %s", envelope.Error.Message)
	}
	if envelope.Result == nil {
		t.Fatalf("no result in the response:\n%s", raw)
	}
	return envelope.Result
}

func callText(t *testing.T, name string, args map[string]any) string {
	t.Helper()
	return resultText(t, call(t, name, args))
}

// A number arriving as JSON is a float64. If it is bound to an int field
// carelessly, rows:1000 becomes 0 and the caller silently gets the default.
func TestNumericArgumentsSurviveBinding(t *testing.T) {
	text := callText(t, "generate", map[string]any{
		"preset": "user", "rows": 25, "seed": 12345,
	})
	var rows []map[string]any
	if err := json.Unmarshal([]byte(text), &rows); err != nil {
		t.Fatalf("cannot decode the rows: %v\n%s", err, text)
	}
	if len(rows) != 25 {
		t.Fatalf("got %d rows, want the 25 asked for over the wire", len(rows))
	}
}

// A seed is uint64 and JSON numbers are float64, which loses precision above
// 2^53. A large seed must either arrive intact or be rejected — never be
// silently rounded to a different seed that still generates something.
func TestLargeSeedIsNotSilentlyRounded(t *testing.T) {
	const seed = 1 << 53
	a := callText(t, "generate", map[string]any{"preset": "user", "rows": 3, "seed": seed})
	b := callText(t, "generate", map[string]any{"preset": "user", "rows": 3, "seed": seed})
	if a != b {
		t.Fatal("the same large seed gave different rows twice")
	}
}

// A boolean must survive too: unmasked=true arriving as a string "true" and
// being ignored would leak raw card numbers while looking like it worked.
func TestBooleanArgumentSurvivesBinding(t *testing.T) {
	masked := callText(t, "generate", map[string]any{"preset": "transaction", "rows": 5, "seed": 1})
	if !strings.Contains(masked, "*") {
		t.Fatal("the default over the wire is not masked")
	}
	raw := callText(t, "generate", map[string]any{
		"preset": "transaction", "rows": 5, "seed": 1, "unmasked": true,
	})
	var rows []map[string]any
	json.Unmarshal([]byte(raw), &rows)
	for i, r := range rows {
		if card, _ := r["card_number"].(string); strings.Contains(card, "*") {
			t.Fatalf("row %d: unmasked=true was ignored over the wire: %q", i, card)
		}
	}
}

// A tool error must come back as a result the model can read, not as a
// transport error that drops the connection over a typo.
func TestToolErrorIsAReadableResult(t *testing.T) {
	for _, tc := range []struct {
		name string
		args map[string]any
		want string
	}{
		{"generate", map[string]any{"preset": "nope"}, "list_presets"},
		{"generate", map[string]any{}, "exactly one"},
		{"generate", map[string]any{"preset": "user", "rows": 100000}, "synth gen"},
		{"verify", map[string]any{"data": ""}, "not a file path"},
		{"mask", map[string]any{"data": "a,b\n1,2\n"}, "key is required"},
		{"snapshot", map[string]any{"spec": snapSpec, "at": "someday"}, "cannot read"},
	} {
		res := call(t, tc.name, tc.args)
		if !res.IsError {
			t.Errorf("%s%v: not reported as an error", tc.name, tc.args)
			continue
		}
		if got := resultText(t, res); !strings.Contains(got, tc.want) {
			t.Errorf("%s%v: error %q does not mention %q", tc.name, tc.args, got, tc.want)
		}
	}
}

// An unknown tool must be refused rather than silently doing nothing.
func TestUnknownToolIsRefused(t *testing.T) {
	req, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": "delete_everything", "arguments": map[string]any{}},
	})
	raw, _ := json.Marshal(New().HandleMessage(context.Background(), req))
	if !strings.Contains(string(raw), "delete_everything") {
		t.Fatalf("an unknown tool produced no useful response:\n%s", raw)
	}
}

// Every tool must survive being called with no arguments at all. A model will
// do this, and a panic in a handler takes the whole server down with it.
func TestNoToolPanicsOnEmptyArguments(t *testing.T) {
	for name := range listTools(t) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked on empty arguments: %v", name, r)
				}
			}()
			call(t, name, map[string]any{})
		}()
	}
}

// Junk in every string field must not panic either.
func TestNoToolPanicsOnJunkArguments(t *testing.T) {
	junk := map[string]any{
		"preset": "\x00\xff", "spec": "\x00not yaml", "data": "\x00\xff\xfe",
		"format": "../../etc", "key": "", "locale": "zz_ZZ", "name": "\n\n",
		"at": "0000-00-00", "from": "-", "to": "-", "window": "-1",
		"rows": -1, "seed": -1,
	}
	for name := range listTools(t) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked on junk arguments: %v", name, r)
				}
			}()
			call(t, name, junk)
		}()
	}
}

// The full workflow has to hold together: generate, verify what came out,
// profile it, and generate again from the inferred spec.
func TestGenerateVerifyProfileRoundTrip(t *testing.T) {
	spec := "name: t\ncount: 50\nfields:\n" +
		"  id: { kind: uuid }\n" +
		"  email: { kind: email }\n" +
		"  card: { kind: card }\n"

	generated := callText(t, "generate", map[string]any{"spec": spec, "rows": 50, "seed": 4})
	var rows []map[string]any
	if err := json.Unmarshal([]byte(generated), &rows); err != nil {
		t.Fatal(err)
	}

	// Turn the rows into JSONL and check them.
	var jsonl strings.Builder
	for _, r := range rows {
		line, _ := json.Marshal(r)
		jsonl.Write(line)
		jsonl.WriteByte('\n')
	}
	report := callText(t, "verify", map[string]any{"data": jsonl.String(), "format": "jsonl"})
	var rep struct {
		Findings []any `json:"findings"`
	}
	json.Unmarshal([]byte(report), &rep)
	if len(rep.Findings) != 0 {
		t.Fatalf("generated data failed its own verify: %s", report)
	}

	// Profile it and generate from what came back.
	profiled := callText(t, "profile", map[string]any{"data": jsonl.String(), "format": "jsonl"})
	var pr struct {
		Spec string `json:"spec"`
	}
	json.Unmarshal([]byte(profiled), &pr)
	if pr.Spec == "" {
		t.Fatalf("profile returned no spec: %s", profiled)
	}
	again := call(t, "generate", map[string]any{"spec": pr.Spec, "rows": 5, "seed": 5})
	if again.IsError {
		t.Fatalf("the profiled spec does not generate: %s\n%s", resultText(t, again), pr.Spec)
	}
}

// Masked output must still verify: a fixture that fails its own checks is not
// a fixture. This is the property that would break if a mask stopped producing
// format-valid values.
func TestMaskedOutputStillVerifies(t *testing.T) {
	data := "email,card\n" +
		"a@example.com,4539578763621486\n" +
		"b@example.com,4556737586899855\n" +
		"c@example.com,4532015112830366\n"

	masked := callText(t, "mask", map[string]any{"data": data, "key": "k"})
	var mr struct {
		Data string `json:"data"`
	}
	json.Unmarshal([]byte(masked), &mr)
	if strings.Contains(mr.Data, "a@example.com") {
		t.Fatal("the original email survived")
	}

	report := callText(t, "verify", map[string]any{"data": mr.Data})
	var rep struct {
		Findings []struct {
			Column string `json:"column"`
			Detail string `json:"detail"`
		} `json:"findings"`
	}
	json.Unmarshal([]byte(report), &rep)
	if len(rep.Findings) != 0 {
		t.Fatalf("masked output no longer passes verify: %+v\n%s", rep.Findings, mr.Data)
	}
}

// A result must be readable as text, because that is what the model gets.
func TestEveryToolReturnsReadableText(t *testing.T) {
	calls := []struct {
		name string
		args map[string]any
	}{
		{"generate", map[string]any{"preset": "user", "rows": 2}},
		{"list_types", map[string]any{"search": "email"}},
		{"list_presets", map[string]any{}},
		{"verify", map[string]any{"data": goodCSV}},
		{"profile", map[string]any{"data": sampleCSV}},
		{"mask", map[string]any{"data": personalCSV, "key": "k"}},
		{"snapshot", map[string]any{"spec": snapSpec, "rows": 5, "at": "2026-07-01"}},
	}
	for _, c := range calls {
		res := call(t, c.name, c.args)
		if res.IsError {
			t.Errorf("%s failed: %s", c.name, resultText(t, res))
			continue
		}
		text := resultText(t, res)
		if len(text) < 10 {
			t.Errorf("%s returned almost nothing: %q", c.name, text)
		}
		var v any
		if err := json.Unmarshal([]byte(text), &v); err != nil {
			t.Errorf("%s returned text that is not JSON: %v", c.name, err)
		}
	}
}
