package mask

import (
	"bytes"
	"strings"
	"testing"
)

// A DP rule on a non-numeric value is a spec error, not a row error: the column
// was declared numeric and it is not. The Masker records the first one and
// keeps it, so a later run on the same Masker still reports it rather than
// quietly noising numbers under a rule that has already been proven wrong.
func TestJSONLReportsDPErrorOnANumberAfterAPoisonedRun(t *testing.T) {
	m := New("poison-key", "en_US")
	m.Rule(Rule{Column: "amount", Strategy: DP, Epsilon: 1, Sensitivity: 1})

	var out bytes.Buffer
	if _, err := m.JSONL(strings.NewReader(`{"amount":"not a number"}`), &out); err == nil {
		t.Fatal("JSONL = nil error, want one for a DP rule on a non-numeric value")
	}

	out.Reset()
	// The value here is a JSON number, which the number branch routes through
	// the masker as text — and it finds the Masker already carrying the error.
	if _, err := m.JSONL(strings.NewReader(`{"amount":12.5}`), &out); err == nil {
		t.Fatal("JSONL = nil error on reuse, want the recorded error to persist")
	}
}
