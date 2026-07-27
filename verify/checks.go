package verify

import (
	"fmt"
	"math"
	"net/mail"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// checkFormats validates values whose column name says what they should be.
//
// The column name is the only signal available, so a check runs only when the
// name is unambiguous and the column's values already look like the right
// shape. A column called "id" full of order numbers must not be audited as
// payment cards.
func checkFormats(rows []map[string]any, cols []string, opts Options) []Finding {
	var out []Finding
	for _, col := range cols {
		validator, name := validatorFor(col)
		if validator == nil {
			continue
		}
		// Only audit a column whose values are mostly the expected shape.
		// Otherwise the column name misled us and every row would "fail".
		if !mostlyValid(rows, col, validator) {
			continue
		}
		c := &collector{check: name, severity: SevError, column: col, max: opts.MaxFindingsPerCheck}
		for i, r := range rows {
			v, ok := stringValue(r[col])
			if !ok || v == "" {
				continue
			}
			if !validator(v) {
				c.add(i, fmt.Sprintf("%q is not a valid %s", v, name), v)
			}
		}
		out = append(out, c.result()...)
	}
	return out
}

// mostlyValid reports whether enough of a column passes to believe the column
// really holds that kind of value. The threshold is deliberately low: a
// genuinely broken export should still be audited, but a column that fails
// almost everywhere was misidentified.
func mostlyValid(rows []map[string]any, col string, valid func(string) bool) bool {
	present, ok := 0, 0
	for _, r := range rows {
		v, isStr := stringValue(r[col])
		if !isStr || v == "" {
			continue
		}
		present++
		if valid(v) {
			ok++
		}
	}
	return present > 0 && float64(ok)/float64(present) >= 0.5
}

// validatorFor maps a column name to a format check.
func validatorFor(col string) (func(string) bool, string) {
	switch normalizeCol(col) {
	case "card", "cardnumber", "creditcard", "pan":
		return validLuhn, "luhn"
	case "imei":
		return validLuhn, "luhn"
	case "iban":
		return validIBAN, "iban"
	case "email", "emailaddress", "mail":
		return validEmail, "email"
	case "url", "website", "link":
		return validURL, "url"
	case "ip", "ipaddress", "ipv4", "ipv6":
		return validIP, "ip"
	case "uuid", "guid":
		return validUUID, "uuid"
	case "ean", "ean13", "barcode":
		return validEAN13, "ean13"
	case "upc":
		return validUPC, "upc"
	}
	return nil, ""
}

func normalizeCol(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func validLuhn(s string) bool {
	d := digitsOnly(s)
	if len(d) < 12 || len(d) > 19 {
		return false
	}
	sum, alt := 0, false
	for i := len(d) - 1; i >= 0; i-- {
		n := int(d[i] - '0')
		if alt {
			if n *= 2; n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}

// validIBAN applies the ISO 7064 mod-97-10 rule: move the first four
// characters to the end, expand letters, and require a remainder of 1.
func validIBAN(s string) bool {
	t := strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	if len(t) < 15 || len(t) > 34 {
		return false
	}
	rearranged := t[4:] + t[:4]
	rem := 0
	for i := 0; i < len(rearranged); i++ {
		ch := rearranged[i]
		switch {
		case ch >= '0' && ch <= '9':
			rem = (rem*10 + int(ch-'0')) % 97
		case ch >= 'A' && ch <= 'Z':
			rem = (rem*100 + int(ch-'A') + 10) % 97
		default:
			return false
		}
	}
	return rem == 1
}

func validEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return false
	}
	at := strings.LastIndex(addr.Address, "@")
	return at > 0 && strings.Contains(addr.Address[at:], ".")
}

func validURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func validIP(s string) bool {
	_, err := netip.ParseAddr(s)
	return err == nil
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func validUUID(s string) bool { return uuidRe.MatchString(s) }

func validEAN13(s string) bool { return validGTIN(s, 13) }
func validUPC(s string) bool   { return validGTIN(s, 12) }

// validGTIN checks the alternating 3/1-weighted check digit shared by EAN-13
// and UPC-A.
func validGTIN(s string, length int) bool {
	d := digitsOnly(s)
	if len(d) != length {
		return false
	}
	sum := 0
	for i := 0; i < length-1; i++ {
		n := int(d[i] - '0')
		// The weight-3 positions alternate from the right, so which parity
		// gets the 3 depends on the code's length.
		if (length-i)%2 == 0 {
			n *= 3
		}
		sum += n
	}
	return (10-sum%10)%10 == int(d[length-1]-'0')
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// checkRefs resolves foreign keys against parent rows loaded from their own
// file.
func checkRefs(rows []map[string]any, opts Options) []Finding {
	var out []Finding
	for _, ref := range opts.Refs {
		keys := make(map[string]bool, len(ref.Parent))
		for _, p := range ref.Parent {
			if v, ok := stringValue(p[ref.ParentKey]); ok {
				keys[v] = true
			}
		}
		c := &collector{check: "fk", severity: SevError, column: ref.Column, max: opts.MaxFindingsPerCheck}
		for i, r := range rows {
			v, ok := stringValue(r[ref.Column])
			if !ok || v == "" {
				continue // a null foreign key is a nullable relation, not a break
			}
			if !keys[v] {
				c.add(i, fmt.Sprintf("%q has no matching %s", v, ref.ParentKey), v)
			}
		}
		out = append(out, c.result()...)
	}
	return out
}

// temporalPairs are column-name fragments that imply an ordering. The first
// element must not come after the second.
var temporalPairs = [][2]string{
	{"created", "updated"},
	{"created", "deleted"},
	{"start", "end"},
	{"started", "finished"},
	{"started", "ended"},
	{"opened", "closed"},
	{"begin", "end"},
	{"from", "to"},
	{"issued", "expires"},
	{"issued", "expiry"},
	{"birth", "death"},
	{"order", "ship"},
	{"ordered", "shipped"},
	{"shipped", "delivered"},
}

// checkTemporal flags timestamp pairs whose names imply an order the data
// does not respect.
func checkTemporal(rows []map[string]any, cols []string, opts Options) []Finding {
	var out []Finding
	for _, pair := range temporalPairs {
		early := findColumn(cols, pair[0])
		late := findColumn(cols, pair[1])
		if early == "" || late == "" || early == late {
			continue
		}
		c := &collector{
			check: "temporal", severity: SevError,
			column: early + "/" + late, max: opts.MaxFindingsPerCheck,
		}
		for i, r := range rows {
			a, okA := timeValue(r[early])
			b, okB := timeValue(r[late])
			if !okA || !okB {
				continue
			}
			if a.After(b) {
				c.add(i, fmt.Sprintf("%s (%s) is after %s (%s)",
					early, a.Format(time.RFC3339), late, b.Format(time.RFC3339)), "")
			}
		}
		out = append(out, c.result()...)
	}
	return out
}

// findColumn returns the column whose name contains frag, if exactly one does.
// An ambiguous match is skipped rather than guessed at.
func findColumn(cols []string, frag string) string {
	var hit string
	for _, c := range cols {
		if strings.Contains(normalizeCol(c), frag) {
			if hit != "" {
				return "" // ambiguous
			}
			hit = c
		}
	}
	return hit
}

// degenerateFrac is how dominant a single value must be before a column is
// called degenerate. Real categorical columns are skewed; this is for columns
// that carry effectively no information.
const degenerateFrac = 0.95

// checkDistribution warns about columns that carry no information: one value
// almost everywhere, or a numeric column that never varies. These are warnings
// because a real dataset may legitimately look like this.
func checkDistribution(rows []map[string]any, cols []string) []Finding {
	var out []Finding
	for _, col := range cols {
		counts := map[string]int{}
		present := 0
		var nums []float64
		for _, r := range rows {
			v, ok := stringValue(r[col])
			if !ok || v == "" {
				continue
			}
			present++
			counts[v]++
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				nums = append(nums, f)
			}
		}
		if present < 20 {
			continue // too few rows to call anything degenerate
		}
		top, topVal := 0, ""
		for v, n := range counts {
			if n > top {
				top, topVal = n, v
			}
		}
		if float64(top)/float64(present) >= degenerateFrac && len(counts) > 1 {
			out = append(out, Finding{
				Check: "distribution", Severity: SevWarn, Column: col, Row: -1,
				Detail: fmt.Sprintf("%.1f%% of rows share one value",
					100*float64(top)/float64(present)),
				Sample: []string{topVal},
			})
			continue
		}
		if len(counts) == 1 {
			out = append(out, Finding{
				Check: "distribution", Severity: SevWarn, Column: col, Row: -1,
				Detail: "every row has the same value",
				Sample: []string{topVal},
			})
			continue
		}
		if len(nums) == present && len(nums) > 0 && variance(nums) == 0 {
			out = append(out, Finding{
				Check: "distribution", Severity: SevWarn, Column: col, Row: -1,
				Detail: "numeric column has zero variance",
			})
		}
	}
	return out
}

func variance(xs []float64) float64 {
	mean := 0.0
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	v := 0.0
	for _, x := range xs {
		v += (x - mean) * (x - mean)
	}
	return v / float64(len(xs))
}

// checkConstraints re-checks mined invariants against this dataset. Mining a
// trusted sample and verifying a suspect one is how a drifted export shows up.
func checkConstraints(rows []map[string]any, opts Options) []Finding {
	var out []Finding
	for _, c := range opts.Constraints {
		col := &collector{
			check: "constraint", severity: SevError,
			column: c.String(), max: opts.MaxFindingsPerCheck,
		}
		for i, r := range rows {
			if !c.Holds(r) {
				col.add(i, "violates "+c.String(), "")
			}
		}
		out = append(out, col.result()...)
	}
	return out
}

// stringValue renders a value for comparison. A nil or absent value reports
// false so checks can skip it rather than validating "<nil>".
func stringValue(v any) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "", false
	case string:
		return x, true
	case time.Time:
		return x.Format(time.RFC3339), true
	case float64:
		// JSON numbers arrive as float64; render integers without a decimal
		// point so they compare equal to the same value from a CSV.
		if x == math.Trunc(x) && math.Abs(x) < 1e15 {
			return strconv.FormatInt(int64(x), 10), true
		}
		return strconv.FormatFloat(x, 'g', -1, 64), true
	default:
		return fmt.Sprint(x), true
	}
}

var timeLayouts = []string{
	time.RFC3339Nano, time.RFC3339,
	"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02",
}

func timeValue(v any) (time.Time, bool) {
	if t, ok := v.(time.Time); ok {
		return t, true
	}
	s, ok := stringValue(v)
	if !ok || s == "" {
		return time.Time{}, false
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// checkKAnonymity requires every combination of quasi-identifier columns to be
// shared by at least k rows. A rarer combination re-identifies an individual
// even after the direct identifiers are gone — a 1985-born person in a small
// ZIP is often unique — so it is the measure of whether "anonymized" data is.
//
// It runs only when the caller asks for it (k > 1 and at least one QI column).
// A named column that is not present is an error rather than a silent pass, so
// a typo cannot read as "no violations".
func checkKAnonymity(rows []map[string]any, cols []string, opts Options) []Finding {
	if opts.KAnonymity <= 1 || len(opts.QuasiIdentifiers) == 0 {
		return nil
	}
	present := map[string]bool{}
	for _, c := range cols {
		present[c] = true
	}
	var out []Finding
	for _, qi := range opts.QuasiIdentifiers {
		if !present[qi] {
			out = append(out, Finding{
				Check: "k-anonymity", Severity: SevError, Column: qi, Row: -1,
				Detail: fmt.Sprintf("quasi-identifier column %q is not in the data", qi),
			})
		}
	}
	if len(out) > 0 {
		return out // do not report bogus groups against a mistyped column set
	}

	// Group rows by the tuple of QI values. The joined key uses a NUL separator
	// so ("a","bc") and ("ab","c") do not collide.
	groups := map[string]int{}
	sample := map[string][]string{}
	for _, row := range rows {
		vals := make([]string, len(opts.QuasiIdentifiers))
		for i, qi := range opts.QuasiIdentifiers {
			vals[i], _ = stringValue(row[qi])
		}
		key := strings.Join(vals, "\x00")
		groups[key]++
		if _, ok := sample[key]; !ok {
			sample[key] = vals
		}
	}

	max := opts.MaxFindingsPerCheck
	if max <= 0 {
		max = DefaultMaxFindings
	}
	shown, violations := 0, 0
	for key, n := range groups {
		if n >= opts.KAnonymity {
			continue
		}
		violations++
		if shown < max {
			shown++
			out = append(out, Finding{
				Check: "k-anonymity", Severity: SevError, Row: -1,
				Detail: fmt.Sprintf("%d row(s) share this quasi-identifier, need %d",
					n, opts.KAnonymity),
				Sample: labelTuple(opts.QuasiIdentifiers, sample[key]),
			})
		}
	}
	if violations > shown {
		out = append(out, Finding{
			Check: "k-anonymity", Severity: SevError, Row: -1,
			Detail: fmt.Sprintf("%d quasi-identifier groups fall below k=%d (%d shown)",
				violations, opts.KAnonymity, shown),
		})
	}
	return out
}

// labelTuple renders a QI group as col=value pairs for a readable finding.
func labelTuple(cols, vals []string) []string {
	out := make([]string, len(cols))
	for i := range cols {
		v := ""
		if i < len(vals) {
			v = vals[i]
		}
		out[i] = cols[i] + "=" + v
	}
	return out
}
