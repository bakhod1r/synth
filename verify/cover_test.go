package verify

import (
	"strings"
	"testing"
	"time"
)

type fakeConstraint struct{ ok bool }

func (f fakeConstraint) Holds(map[string]any) bool { return f.ok }
func (f fakeConstraint) String() string            { return "x <= y" }

func TestCheckFormatsAllValidators(t *testing.T) {
	// Each column is mostly valid with one broken value, so mostlyValid passes
	// and exactly one finding per column is produced.
	rows := []map[string]any{
		{"card": "4539578763621486", "iban": "GB82WEST12345698765432", "email": "a@b.com",
			"url": "https://x.com", "ip": "10.0.0.1", "uuid": "550e8400-e29b-41d4-a716-446655440000",
			"ean13": "4006381333931", "upc": "036000291452", "imei": "490154203237518"},
		{"card": "bad", "iban": "BAD", "email": "nope", "url": "not a url", "ip": "999.1.1.1",
			"uuid": "not-a-uuid", "ean13": "0000000000000", "upc": "000000000000", "imei": "123"},
	}
	rep := Run(rows, Options{})
	if rep.Errors() == 0 {
		t.Fatal("expected format errors")
	}
}

func TestCheckRefs(t *testing.T) {
	rows := []map[string]any{{"uid": "1"}, {"uid": "2"}, {"uid": ""}}
	parent := []map[string]any{{"id": "1"}}
	rep := Run(rows, Options{Refs: []Ref{{Column: "uid", Parent: parent, ParentKey: "id"}}})
	if rep.Errors() == 0 {
		t.Fatal("dangling FK should error")
	}
}

func TestCheckTemporal(t *testing.T) {
	rows := []map[string]any{
		{"created_at": "2026-02-01", "updated_at": "2026-01-01"}, // out of order
		{"created_at": "2026-01-01", "updated_at": "2026-02-01"}, // fine
	}
	rep := Run(rows, Options{})
	if rep.Errors() == 0 {
		t.Fatal("temporal violation should error")
	}
}

func TestCheckDistribution(t *testing.T) {
	var rows []map[string]any
	for i := 0; i < 25; i++ {
		val := "same"
		if i == 0 {
			val = "different" // degenerate: 96% one value
		}
		rows = append(rows, map[string]any{
			"const": "K",    // every row identical
			"deg":   val,    // one value dominates
			"num":   "3.14", // numeric, zero variance
		})
	}
	rep := Run(rows, Options{})
	warns := 0
	for _, f := range rep.Findings {
		if f.Check == "distribution" {
			warns++
		}
	}
	if warns == 0 {
		t.Fatal("expected distribution warnings")
	}
}

func TestCheckConstraints(t *testing.T) {
	rows := []map[string]any{{"a": 1}, {"a": 2}}
	rep := Run(rows, Options{Constraints: []Constraint{fakeConstraint{ok: false}}})
	if rep.Errors() == 0 {
		t.Fatal("violated constraint should error")
	}
	// A holding constraint produces nothing.
	rep2 := Run(rows, Options{Constraints: []Constraint{fakeConstraint{ok: true}}})
	for _, f := range rep2.Findings {
		if f.Check == "constraint" {
			t.Fatal("holding constraint should not fire")
		}
	}
}

func TestKAnonymity(t *testing.T) {
	rows := []map[string]any{
		{"zip": "111", "age": "30"},
		{"zip": "111", "age": "30"},
		{"zip": "222", "age": "40"}, // singleton group
	}
	rep := Run(rows, Options{KAnonymity: 2, QuasiIdentifiers: []string{"zip", "age"}})
	if rep.Errors() == 0 {
		t.Fatal("rare QI combo should error")
	}
	// A missing QI column is an error, and short-circuits grouping.
	rep2 := Run(rows, Options{KAnonymity: 2, QuasiIdentifiers: []string{"nosuch"}})
	if rep2.Errors() == 0 {
		t.Fatal("missing QI column should error")
	}
	// k<=1 disables the check.
	rep3 := Run(rows, Options{KAnonymity: 1, QuasiIdentifiers: []string{"zip"}})
	for _, f := range rep3.Findings {
		if f.Check == "k-anonymity" {
			t.Fatal("k<=1 should not run k-anonymity")
		}
	}
}

func TestKAnonymitySummaryCollapse(t *testing.T) {
	// Many singleton groups exceed MaxFindingsPerCheck, forcing the summary.
	var rows []map[string]any
	for i := 0; i < 30; i++ {
		rows = append(rows, map[string]any{"zip": string(rune('a' + i))})
	}
	rep := Run(rows, Options{KAnonymity: 2, QuasiIdentifiers: []string{"zip"}, MaxFindingsPerCheck: 2})
	if rep.Errors() == 0 {
		t.Fatal("expected k-anonymity violations")
	}
}

func TestCollectorSummaryBranch(t *testing.T) {
	// The column must be mostly valid or it is skipped, so include many valid
	// cards plus several invalid ones; max=2 then forces the summary.
	var rows []map[string]any
	for i := 0; i < 10; i++ {
		rows = append(rows, map[string]any{"card": "4539578763621486"})
	}
	for i := 0; i < 4; i++ {
		rows = append(rows, map[string]any{"card": "1111111111111111"})
	}
	rep := Run(rows, Options{MaxFindingsPerCheck: 2})
	summary := false
	for _, f := range rep.Findings {
		if strings.Contains(f.Detail, "rows affected in total") {
			summary = true
		}
	}
	if !summary {
		t.Fatal("expected a collapse summary finding")
	}
}

func TestReportOutput(t *testing.T) {
	// Empty dataset: OK and a clean text line.
	empty := Run(nil, Options{})
	if !empty.OK() {
		t.Fatal("empty report should be OK")
	}
	var b strings.Builder
	if err := empty.Text(&b); err != nil || !strings.Contains(b.String(), "no problems") {
		t.Fatalf("clean text = %q err=%v", b.String(), err)
	}

	rep := Run([]map[string]any{
		{"card": "4539578763621486"}, {"card": "4539578763621486"},
		{"card": "4539578763621486"}, {"card": "1111111111111111"},
	}, Options{})
	var tb strings.Builder
	if err := rep.Text(&tb); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tb.String(), "findings") {
		t.Fatalf("text report = %q", tb.String())
	}
	var jb strings.Builder
	if err := rep.JSON(&jb); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(jb.String(), "findings") {
		t.Fatal("json missing findings")
	}
}

func TestValueHelpers(t *testing.T) {
	if _, ok := stringValue(nil); ok {
		t.Fatal("nil -> false")
	}
	if s, _ := stringValue(3.0); s != "3" {
		t.Fatalf("float int render = %q", s)
	}
	if s, _ := stringValue(3.5); s != "3.5" {
		t.Fatalf("float render = %q", s)
	}
	if s, _ := stringValue(true); s != "true" {
		t.Fatalf("bool render = %q", s)
	}
	if _, ok := timeValue(time.Now()); !ok {
		t.Fatal("time.Time should parse")
	}
	if _, ok := timeValue("2026-01-02"); !ok {
		t.Fatal("date string should parse")
	}
	if _, ok := timeValue("nonsense"); ok {
		t.Fatal("nonsense should not parse")
	}
	if _, ok := timeValue(nil); ok {
		t.Fatal("nil time -> false")
	}
}

type failW struct{ n, at int }

func (f *failW) Write(p []byte) (int, error) {
	f.n++
	if f.n >= f.at {
		return 0, errW
	}
	return len(p), nil
}

type errWT struct{}

func (errWT) Error() string { return "w" }

var errW = errWT{}

func TestFormatEdgeBranches(t *testing.T) {
	// A card column, mostly valid, with an empty and a nil cell (skipped), plus
	// an IBAN with a special char that passes length but fails the alphabet, and
	// an EAN13 of the wrong digit count.
	rows := []map[string]any{
		{"card": "4539578763621486", "iban": "GB82WEST12345698765432", "ean13": "4006381333931"},
		{"card": "4539578763621486", "iban": "GB82WEST12345698765432", "ean13": "4006381333931"},
		{"card": "", "iban": "GB82-WEST-1234-5698-7654", "ean13": "123"},
		{"card": nil},
	}
	_ = Run(rows, Options{})
}

func TestFormatColumnMisidentified(t *testing.T) {
	// An "email" column that is mostly NOT email is skipped (mostlyValid false).
	rows := []map[string]any{
		{"email": "not an email"}, {"email": "also bad"}, {"email": "nope"},
	}
	rep := Run(rows, Options{})
	for _, f := range rep.Findings {
		if f.Check == "email" {
			t.Fatal("misidentified email column should be skipped")
		}
	}
}

func TestTemporalNonTimeSkipped(t *testing.T) {
	rows := []map[string]any{
		{"created_at": "not a date", "updated_at": "also not"},
		{"created_at": "2026-02-01", "updated_at": "2026-01-01"},
	}
	_ = Run(rows, Options{})
}

func TestDistributionZeroVarianceDistinctStrings(t *testing.T) {
	// Two distinct string spellings of the same number: >1 distinct value but
	// zero numeric variance.
	var rows []map[string]any
	for i := 0; i < 25; i++ {
		v := "3.14"
		if i%2 == 0 {
			v = "3.140"
		}
		rows = append(rows, map[string]any{"num": v})
	}
	rep := Run(rows, Options{})
	found := false
	for _, f := range rep.Findings {
		if strings.Contains(f.Detail, "zero variance") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected zero-variance finding")
	}
}

func TestStringValueTime(t *testing.T) {
	if s, ok := stringValue(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); !ok || s == "" {
		t.Fatalf("time stringValue = %q,%v", s, ok)
	}
}

func TestCheckKAnonymityDirectDefaultMax(t *testing.T) {
	// Direct call with max=0 exercises the internal default.
	rows := []map[string]any{{"z": "a"}, {"z": "b"}}
	out := checkKAnonymity(rows, []string{"z"}, Options{KAnonymity: 2, QuasiIdentifiers: []string{"z"}})
	if len(out) == 0 {
		t.Fatal("expected violations")
	}
}

func TestTextWriteErrors(t *testing.T) {
	rep := Run([]map[string]any{
		{"card": "4539578763621486"}, {"card": "4539578763621486"},
		{"card": "4539578763621486"}, {"card": "1111111111111111"},
	}, Options{})
	// Fail at various write points to hit each error return in Text.
	for at := 1; at <= 3; at++ {
		if err := rep.Text(&failW{at: at}); err == nil {
			t.Fatalf("Text should surface a write error at %d", at)
		}
	}
	// No-findings clean line write error.
	if err := Run(nil, Options{}).Text(&failW{at: 1}); err == nil {
		t.Fatal("clean Text write error should surface")
	}
}

func TestFindColumnAmbiguous(t *testing.T) {
	if findColumn([]string{"created_at", "recreated_at"}, "created") != "" {
		t.Fatal("ambiguous fragment should return empty")
	}
	if findColumn([]string{"created_at"}, "created") != "created_at" {
		t.Fatal("single match should return the column")
	}
}
