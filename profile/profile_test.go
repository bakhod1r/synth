package profile

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func fromCSV(t *testing.T, data string) *Result {
	t.Helper()
	r, err := FromCSV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("FromCSV: %v", err)
	}
	return r
}

func TestColumnOrderIsFileOrder(t *testing.T) {
	r := fromCSV(t, "z,a,m\n1,2,3\n")
	want := []string{"z", "a", "m"}
	for i, c := range want {
		if r.Order[i] != c {
			t.Fatalf("Order = %v, want %v", r.Order, want)
		}
	}
}

// Profiling exists to reproduce the shape of real data, so getting the type
// wrong is the whole failure. These are the formats a real export is full of.
func TestFormatDetection(t *testing.T) {
	// Enough rows for the identifier heuristic to be confident. Below twenty a
	// column with all-distinct values is genuinely ambiguous, and guessing
	// would be worse than waiting for evidence.
	lines := []string{"email,uuid,ip,url,when,n,amount,flag"}
	for i := 0; i < 40; i++ {
		lines = append(lines, fmt.Sprintf(
			"u%d@example.com,3f2504e0-4f89-11d3-9a0c-%012d,10.0.%d.%d,https://example.com/%d,2026-01-%02dT03:04:05Z,%d,%d.50,%v",
			i, i, i/256, i%256, i, i%28+1, i, i, i%2 == 0))
	}
	r := fromCSV(t, strings.Join(lines, "\n")+"\n")

	want := map[string]schema.Kind{
		"email": schema.KindEmail,
		"uuid":  schema.KindUUID,
		"ip":    schema.KindIPv4,
		"url":   schema.KindURL,
		"when":  schema.KindTime,
		"n":     schema.KindInt,
		"flag":  schema.KindBool,
	}
	for col, kind := range want {
		f := r.Schema.FieldByName(col)
		if f == nil {
			t.Errorf("no column %q", col)
			continue
		}
		if f.Kind != kind {
			t.Errorf("%s: kind = %q, want %q", col, f.Kind, kind)
		}
	}
	if f := r.Schema.FieldByName("amount"); f == nil || (f.Kind != schema.KindFloat && f.Kind != schema.KindAmount) {
		t.Errorf("amount: kind = %v, want a numeric kind", f)
	}
}

// A numeric range must come from the data. Generating outside it would produce
// rows the original table could not contain.
func TestNumericRange(t *testing.T) {
	r := fromCSV(t, "n\n10\n50\n30\n")
	s := r.Stats["n"]
	if s == nil {
		t.Fatal("no stats for n")
	}
	if s.Min != 10 || s.Max != 50 {
		t.Fatalf("range = %v..%v, want 10..50", s.Min, s.Max)
	}
}

// A low-cardinality column is reproduced as a weighted enum, so the generated
// data has the same category mix rather than a uniform one.
func TestCategoricalBecomesWeightedEnum(t *testing.T) {
	rows := []string{"status"}
	for i := 0; i < 80; i++ {
		rows = append(rows, "active")
	}
	for i := 0; i < 20; i++ {
		rows = append(rows, "closed")
	}
	r := fromCSV(t, strings.Join(rows, "\n")+"\n")

	f := r.Schema.FieldByName("status")
	if f == nil || f.Kind != schema.KindEnum {
		t.Fatalf("status: kind = %v, want enum", f)
	}
	if len(f.Choices) != 2 || len(f.Weights) != 2 {
		t.Fatalf("choices = %v, weights = %v", f.Choices, f.Weights)
	}
	// Weights are raw counts; weightedPick sums them, so they need not be
	// normalised. What matters is the ratio.
	byChoice := map[string]float64{}
	for i, c := range f.Choices {
		byChoice[c] = f.Weights[i]
	}
	ratio := byChoice["active"] / (byChoice["active"] + byChoice["closed"])
	if math.Abs(ratio-0.8) > 0.01 {
		t.Errorf("active share = %v, want ~0.8 (weights %v)", ratio, f.Weights)
	}
}

// A high-cardinality column is not an enum: reproducing ten thousand distinct
// ids as a choice list would copy the original values into the spec.
func TestHighCardinalityIsNotAnEnum(t *testing.T) {
	rows := []string{"id"}
	for i := 0; i < 500; i++ {
		rows = append(rows, "value-"+strings.Repeat("x", i%7)+string(rune('a'+i%26))+itoa(i))
	}
	r := fromCSV(t, strings.Join(rows, "\n")+"\n")
	if f := r.Schema.FieldByName("id"); f != nil && f.Kind == schema.KindEnum {
		t.Fatalf("a 500-value column became an enum with %d choices", len(f.Choices))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// Null rates must be observed, so a column that is empty a fifth of the time
// generates that way instead of always being populated.
func TestNullsAreCounted(t *testing.T) {
	// A blank line is skipped by encoding/csv rather than read as an empty
	// value, so the file needs a second column to hold the gaps.
	r := fromCSV(t, "a,b\n1,x\n,x\n3,x\n,x\n5,x\n")
	s := r.Stats["a"]
	if s == nil {
		t.Fatal("no stats")
	}
	if s.Nulls != 2 || s.NonNull != 3 {
		t.Fatalf("nulls = %d, non-null = %d, want 2 and 3", s.Nulls, s.NonNull)
	}
}

func TestJSONL(t *testing.T) {
	r, err := FromJSONL(strings.NewReader(
		`{"email":"a@example.com","n":1}` + "\n" + `{"email":"b@example.com","n":2}` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Rows != 2 {
		t.Fatalf("rows = %d, want 2", r.Rows)
	}
	if f := r.Schema.FieldByName("email"); f == nil || f.Kind != schema.KindEmail {
		t.Fatalf("email: %v", f)
	}
}

// A ragged CSV is what a hand-edited export looks like. It must not panic or
// silently drop the rest of the file.
func TestRaggedRows(t *testing.T) {
	r := fromCSV(t, "a,b,c\n1,2,3\n4,5\n6,7,8,9\n")
	if r.Rows == 0 {
		t.Fatal("a ragged file produced no rows at all")
	}
	if len(r.Order) != 3 {
		t.Fatalf("columns = %v, want the three from the header", r.Order)
	}
}

func TestEmptyInput(t *testing.T) {
	for _, data := range []string{"", "\n"} {
		r, err := FromCSV(strings.NewReader(data))
		if err != nil {
			continue // refusing an empty file is a fine answer
		}
		if len(r.Order) > 0 {
			t.Errorf("%q produced columns %v", data, r.Order)
		}
	}
	// A whitespace-only line is a header naming one blank column. That is what
	// the file literally says, and inventing a rule to reject it would also
	// reject a real export whose first column has no name.
	if r, err := FromCSV(strings.NewReader("   ")); err == nil && r.Rows != 0 {
		t.Errorf("a header-only file reported %d rows", r.Rows)
	}
}

func TestHeaderOnly(t *testing.T) {
	r := fromCSV(t, "a,b\n")
	if r.Rows != 0 {
		t.Fatalf("rows = %d, want 0", r.Rows)
	}
	if len(r.Order) != 2 {
		t.Fatalf("columns = %v, want two", r.Order)
	}
}

// Numbers too large for a float must not become an infinity that later renders
// as "+Inf" in a spec nobody can use.
func TestOutOfRangeNumbers(t *testing.T) {
	r := fromCSV(t, "n\n1e400\n-1e400\n1\n")
	s := r.Stats["n"]
	if s == nil {
		return // treating the column as text is a valid answer
	}
	if math.IsInf(s.Min, 0) || math.IsInf(s.Max, 0) || math.IsNaN(s.Min) || math.IsNaN(s.Max) {
		t.Fatalf("range = %v..%v, which cannot be written to a spec", s.Min, s.Max)
	}
}

func TestDuplicateHeaders(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a duplicate header panicked: %v", r)
		}
	}()
	r := fromCSV(t, "a,a,b\n1,2,3\n")
	if len(r.Order) == 0 {
		t.Fatal("no columns")
	}
}

// A true/false column must generate booleans, not the strings "true" and
// "false". An enum would reproduce the ratio faithfully and still hand a JSON
// consumer a string where the source had a boolean.
func TestBooleanColumn(t *testing.T) {
	for _, spelling := range [][2]string{
		{"true", "false"}, {"t", "f"}, {"TRUE", "FALSE"}, {"yes", "no"},
	} {
		rows := []string{"flag"}
		for i := 0; i < 90; i++ {
			rows = append(rows, spelling[0])
		}
		for i := 0; i < 10; i++ {
			rows = append(rows, spelling[1])
		}
		r := fromCSV(t, strings.Join(rows, "\n")+"\n")
		f := r.Schema.FieldByName("flag")
		if f == nil || f.Kind != schema.KindBool {
			t.Fatalf("%v: kind = %v, want bool", spelling, f)
		}
		// The observed ratio must survive, or profiling a 90%-true column
		// produces an even split and the generated data has a different shape.
		share, err := strconv.ParseFloat(f.Params["true"], 64)
		if err != nil {
			t.Fatalf("%v: true share %q: %v", spelling, f.Params["true"], err)
		}
		if math.Abs(share-0.9) > 0.01 {
			t.Errorf("%v: true share = %v, want ~0.9", spelling, share)
		}
	}
}

// A column of only 1 and 0 stays an integer. An integer column that happens to
// hold only zeroes and ones is far more common than a boolean written that way,
// and guessing wrong turns a count into a flag.
func TestOnesAndZeroesStayNumeric(t *testing.T) {
	rows := []string{"n"}
	for i := 0; i < 50; i++ {
		rows = append(rows, "1", "0")
	}
	r := fromCSV(t, strings.Join(rows, "\n")+"\n")
	if f := r.Schema.FieldByName("n"); f == nil || f.Kind == schema.KindBool {
		t.Fatalf("a 1/0 column became %v", f)
	}
}
