package mask

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// Masking a JSONL export has to leave the file loadable: same line count, same
// keys, same JSON types — only the values change.
func TestJSONLKeepsShapeAndTypes(t *testing.T) {
	in := strings.Join([]string{
		`{"id":1,"email":"real.person@example.com","salary":52000,"note":null,"active":true}`,
		`{"id":2,"email":"other@example.com","salary":61000,"note":"ok","active":false}`,
	}, "\n")

	m := New("k1", "en_US")
	var out bytes.Buffer
	rep, err := m.JSONL(strings.NewReader(in), &out)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Rows != 2 {
		t.Fatalf("Rows = %d, want 2", rep.Rows)
	}

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	for i, l := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(l), &obj); err != nil {
			t.Fatalf("line %d: %v", i+1, err)
		}
		if _, ok := obj["id"].(float64); !ok {
			t.Fatalf("line %d: id is no longer a number: %#v", i+1, obj["id"])
		}
		if _, ok := obj["active"].(bool); !ok {
			t.Fatalf("line %d: active is no longer a bool", i+1)
		}
		email, _ := obj["email"].(string)
		if strings.Contains(email, "real.person") {
			t.Fatalf("line %d: the real address survived: %q", i+1, email)
		}
		if !strings.Contains(email, "@") {
			t.Fatalf("line %d: the replacement is not an address: %q", i+1, email)
		}
	}
	if rep.Masked["email"] != 2 {
		t.Fatalf("Masked[email] = %d, want 2", rep.Masked["email"])
	}
	// A number with no DP rule is not identity-bearing, so it passes through.
	if rep.Masked["salary"] != 0 {
		t.Fatalf("salary was masked without a rule")
	}
}

// A DP rule exists to noise a number, so JSONL must route numeric columns
// through the masker and keep them numeric.
func TestJSONLAppliesDPToNumbers(t *testing.T) {
	m := New("k2", "en_US")
	m.Rule(Rule{Column: "salary", Strategy: DP, Epsilon: 1.0})

	in := `{"salary":50000}
{"salary":50000}
{"salary":50000}`
	var out bytes.Buffer
	rep, err := m.JSONL(strings.NewReader(in), &out)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Masked["salary"] != 3 {
		t.Fatalf("Masked[salary] = %d, want 3", rep.Masked["salary"])
	}
	for i, l := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n") {
		var obj map[string]any
		if err := json.Unmarshal([]byte(l), &obj); err != nil {
			t.Fatal(err)
		}
		if _, ok := obj["salary"].(float64); !ok {
			t.Fatalf("line %d: DP turned a number into %#v", i+1, obj["salary"])
		}
	}
}

func TestJSONLReportsMalformedLine(t *testing.T) {
	m := New("k3", "en_US")
	var out bytes.Buffer
	_, err := m.JSONL(strings.NewReader("{\"a\":1}\n{not json\n"), &out)
	if err == nil {
		t.Fatal("expected a parse error")
	}
	if !strings.Contains(err.Error(), "row 2") {
		t.Fatalf("err = %v, want the failing row number", err)
	}
}

// The same key must produce the same replacement, so related dumps still join;
// a different key must make the two unlinkable.
func TestSameKeyJoinsDifferentKeyDoesNot(t *testing.T) {
	const row = `{"user_id":"u-77","email":"a@b.com"}`

	mask := func(key string) map[string]any {
		var out bytes.Buffer
		if _, err := New(key, "en_US").JSONL(strings.NewReader(row), &out); err != nil {
			t.Fatal(err)
		}
		var obj map[string]any
		if err := json.Unmarshal(out.Bytes(), &obj); err != nil {
			t.Fatal(err)
		}
		return obj
	}

	a, b, c := mask("same"), mask("same"), mask("different")
	if a["email"] != b["email"] {
		t.Fatalf("the same key gave two different replacements: %v vs %v", a["email"], b["email"])
	}
	if a["email"] == c["email"] {
		t.Fatal("a different key produced a linkable replacement")
	}
}
