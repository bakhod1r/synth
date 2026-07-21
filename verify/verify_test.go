package verify_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/bakhodir/synth/verify"
)

// pad repeats a row often enough to clear the checks' minimum-sample guards.
func pad(base map[string]any, n int, vary func(i int, r map[string]any)) []map[string]any {
	out := make([]map[string]any, n)
	for i := 0; i < n; i++ {
		r := make(map[string]any, len(base))
		for k, v := range base {
			r[k] = v
		}
		if vary != nil {
			vary(i, r)
		}
		out[i] = r
	}
	return out
}

// A broken check digit must be reported, and the valid rows around it must not.
func TestDetectsLuhnFailure(t *testing.T) {
	rows := pad(map[string]any{"card": "4111111111111111"}, 50, func(i int, r map[string]any) {
		if i == 7 {
			r["card"] = "4111111111111112" // check digit off by one
		}
	})
	rep := verify.Run(rows, verify.Options{})

	var got *verify.Finding
	for i := range rep.Findings {
		if rep.Findings[i].Check == "luhn" {
			got = &rep.Findings[i]
		}
	}
	if got == nil {
		t.Fatalf("did not flag the invalid card; findings: %+v", rep.Findings)
	}
	if got.Row != 7 {
		t.Fatalf("flagged row %d, want 7", got.Row)
	}
	if rep.OK() {
		t.Fatal("a failed checksum must make the report not OK")
	}
}

// A column named "card" that clearly does not hold cards was misidentified;
// auditing it would report every row and drown the real findings.
func TestSkipsMisidentifiedColumn(t *testing.T) {
	rows := pad(map[string]any{"card": "loyalty-gold"}, 50, nil)
	for _, f := range verify.Run(rows, verify.Options{}).Findings {
		if f.Check == "luhn" {
			t.Fatalf("audited a column that does not hold card numbers: %+v", f)
		}
	}
}

// A child row pointing at a missing parent must be reported by value.
func TestDetectsBrokenForeignKey(t *testing.T) {
	parents := []map[string]any{{"id": "a"}, {"id": "b"}}
	children := []map[string]any{{"user_id": "a"}, {"user_id": "ghost"}, {"user_id": "b"}}

	rep := verify.Run(children, verify.Options{
		Refs: []verify.Ref{{Column: "user_id", Parent: parents, ParentKey: "id"}},
	})
	if len(rep.Findings) != 1 {
		t.Fatalf("want exactly one finding, got %+v", rep.Findings)
	}
	f := rep.Findings[0]
	if f.Check != "fk" || f.Row != 1 {
		t.Fatalf("wrong finding: %+v", f)
	}
	if !strings.Contains(f.Detail, "ghost") {
		t.Fatalf("finding does not name the bad value: %+v", f)
	}
}

// A null foreign key is a nullable relation, not a broken one.
func TestNullForeignKeyIsNotABreak(t *testing.T) {
	parents := []map[string]any{{"id": "a"}}
	children := []map[string]any{{"user_id": nil}, {"user_id": ""}, {"user_id": "a"}}

	rep := verify.Run(children, verify.Options{
		Refs: []verify.Ref{{Column: "user_id", Parent: parents, ParentKey: "id"}},
	})
	if len(rep.Findings) != 0 {
		t.Fatalf("null keys reported as broken: %+v", rep.Findings)
	}
}

// updated_at before created_at is an anomaly regardless of formatting.
func TestDetectsTemporalAnomaly(t *testing.T) {
	rows := pad(map[string]any{
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-02-01T00:00:00Z",
	}, 30, func(i int, r map[string]any) {
		if i == 3 {
			r["updated_at"] = "2025-01-01T00:00:00Z"
		}
	})
	rep := verify.Run(rows, verify.Options{})
	for _, f := range rep.Findings {
		if f.Check == "temporal" && f.Row == 3 {
			return
		}
	}
	t.Fatalf("did not flag updated_at before created_at; got %+v", rep.Findings)
}

// A column that is 99% one value is worth a warning, but is not an error:
// real data legitimately looks like this.
func TestFlagsDegenerateDistributionAsWarning(t *testing.T) {
	rows := pad(map[string]any{"status": "same"}, 1000, func(i int, r map[string]any) {
		if i == 0 {
			r["status"] = "different"
		}
	})
	rep := verify.Run(rows, verify.Options{})
	found := false
	for _, f := range rep.Findings {
		if f.Check == "distribution" && f.Column == "status" {
			found = true
			if f.Severity != verify.SevWarn {
				t.Errorf("distribution finding should be a warning, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("did not warn about a degenerate status column: %+v", rep.Findings)
	}
	if !rep.OK() {
		t.Fatal("warnings alone must not make a report fail")
	}
}

// A systematically broken column must report once, not a million times.
func TestFindingsAreCapped(t *testing.T) {
	rows := pad(map[string]any{"user_id": "ghost"}, 5000, nil)
	rep := verify.Run(rows, verify.Options{
		Refs: []verify.Ref{{Column: "user_id", Parent: []map[string]any{{"id": "a"}}, ParentKey: "id"}},
	})
	var fk []verify.Finding
	for _, f := range rep.Findings {
		if f.Check == "fk" {
			fk = append(fk, f)
		}
	}
	if len(fk) > verify.DefaultMaxFindings+1 {
		t.Fatalf("emitted %d findings for one broken column", len(fk))
	}
	summary := fk[len(fk)-1]
	if !strings.Contains(summary.Detail, "5000") {
		t.Fatalf("summary finding does not report the true total: %+v", summary)
	}
	if summary.Row != -1 {
		t.Fatalf("the summary should be dataset-wide, got row %d", summary.Row)
	}
}

// The whole design rests on this: correct data must produce nothing.
func TestCleanDatasetIsSilent(t *testing.T) {
	rows := []map[string]any{}
	cards := []string{"4111111111111111", "5555555555554444", "378282246310005"}
	for i := 0; i < 100; i++ {
		rows = append(rows, map[string]any{
			"id":         uuidAt(i),
			"card":       cards[i%len(cards)],
			"email":      "user" + string(rune('a'+i%26)) + "@example.com",
			"url":        fmt.Sprintf("https://example.com/page/%d", i),
			"ip":         fmt.Sprintf("192.0.2.%d", 1+i%200),
			"created_at": fmt.Sprintf("2026-01-%02dT00:00:00Z", 1+i%28),
			"updated_at": fmt.Sprintf("2026-02-%02dT00:00:00Z", 1+i%28),
			"status":     []string{"active", "pending", "closed"}[i%3],
			"amount":     float64(i) * 1.5,
		})
	}
	rep := verify.Run(rows, verify.Options{})
	if len(rep.Findings) != 0 {
		t.Fatalf("clean data produced findings — these are false positives: %+v", rep.Findings)
	}
}

func uuidAt(i int) string {
	const hex = "0123456789abcdef"
	b := []byte("00000000-0000-4000-8000-000000000000")
	b[len(b)-1] = hex[i%16]
	b[len(b)-2] = hex[(i/16)%16]
	return string(b)
}

// A mined invariant must be checkable against a suspect dataset.
func TestChecksConstraints(t *testing.T) {
	rows := []map[string]any{
		{"subtotal": 10.0, "tax": 1.0, "total": 11.0},
		{"subtotal": 10.0, "tax": 1.0, "total": 99.0}, // wrong
	}
	rep := verify.Run(rows, verify.Options{Constraints: []verify.Constraint{sumRule{}}})
	for _, f := range rep.Findings {
		if f.Check == "constraint" && f.Row == 1 {
			return
		}
	}
	t.Fatalf("did not flag the row violating the invariant: %+v", rep.Findings)
}

type sumRule struct{}

func (sumRule) String() string { return "subtotal + tax = total" }
func (sumRule) Holds(rec map[string]any) bool {
	return rec["subtotal"].(float64)+rec["tax"].(float64) == rec["total"].(float64)
}

// An empty dataset is not a failure.
func TestEmptyDataset(t *testing.T) {
	rep := verify.Run(nil, verify.Options{})
	if !rep.OK() || len(rep.Findings) != 0 {
		t.Fatalf("empty dataset produced findings: %+v", rep.Findings)
	}
}

// Both report formats must render without losing the findings.
func TestReportRendering(t *testing.T) {
	rows := []map[string]any{{"user_id": "ghost"}}
	rep := verify.Run(rows, verify.Options{
		Refs: []verify.Ref{{Column: "user_id", Parent: []map[string]any{{"id": "a"}}, ParentKey: "id"}},
	})

	var text bytes.Buffer
	if err := rep.Text(&text); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "ghost") {
		t.Fatalf("text report lost the detail:\n%s", text.String())
	}

	var js bytes.Buffer
	if err := rep.JSON(&js); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(js.String(), `"check": "fk"`) {
		t.Fatalf("json report lost the finding:\n%s", js.String())
	}
}

// A clean report should say so rather than print nothing.
func TestCleanReportText(t *testing.T) {
	var buf bytes.Buffer
	if err := (verify.Report{Rows: 10}).Text(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "no problems") {
		t.Fatalf("clean report is unclear: %q", buf.String())
	}
}
