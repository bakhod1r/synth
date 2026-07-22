package mask

import (
	"regexp"
	"strings"

	"github.com/bakhod1r/synth/infer"
	"github.com/bakhod1r/synth/schema"
)

// personalKind decides whether a column holds personal data and, if so, what
// kind of value to synthesize. It checks the column NAME first (authoritative
// when it matches a known personal field), then the VALUE's format — so a
// column called "notes" holding an email is still caught.
func personalKind(column, value string) (schema.Kind, bool) {
	if k, matched := infer.Kind(column, ""); matched && personalKinds[k] {
		return k, true
	}
	if k, ok := formatKind(value); ok {
		return k, true
	}
	// Column names that signal personal data even without a synonym match.
	lower := strings.ToLower(column)
	for _, hint := range personalNameHints {
		if strings.Contains(lower, hint) {
			return schema.KindLorem, true
		}
	}
	return "", false
}

// personalKinds are the kinds that carry personal or sensitive data and must
// never survive into an anonymized dump.
var personalKinds = map[schema.Kind]bool{
	schema.KindName: true, schema.KindFirstName: true, schema.KindLastName: true,
	schema.KindMiddleName: true, schema.KindNickname: true, schema.KindUsername: true,
	schema.KindEmail: true, schema.KindPhone: true, schema.KindStreet: true,
	schema.KindPostcode: true, schema.KindIBAN: true, schema.KindCard: true,
	schema.KindPassport: true, schema.KindSSN: true, schema.KindEIN: true,
	schema.KindIPv4: true, schema.KindIPv6: true, schema.KindMAC: true,
	schema.KindBloodType: true, schema.KindMaritalStatus: true, schema.KindGender: true,
	schema.KindSwift: true, schema.KindAccountNumber: true, schema.KindRoutingNumber: true,
	schema.KindVIN: true, schema.KindLicensePlate: true, schema.KindPassword: true,
	schema.KindLatitude: true, schema.KindLongitude: true, schema.KindImageURL: true,
}

var personalNameHints = []string{
	"birth", "dob", "salary", "passport", "national", "tax", "insurance",
	"address", "contact", "secret", "token", "credential",
}

var (
	reEmailV = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	rePhoneV = regexp.MustCompile(`^\+?\d[\d\s\-()]{6,}$`)
	reIPv4V  = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}$`)
	reIBANV  = regexp.MustCompile(`^[A-Z]{2}\d{2}[A-Z0-9]{10,30}$`)
	reCardV  = regexp.MustCompile(`^\d{13,19}$`)
	reSSNV   = regexp.MustCompile(`^\d{3}-\d{2}-\d{4}$`)
	reMACV   = regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)
)

// formatKind recognizes personal data by the shape of the value itself.
func formatKind(v string) (schema.Kind, bool) {
	switch {
	case reEmailV.MatchString(v):
		return schema.KindEmail, true
	case reSSNV.MatchString(v):
		return schema.KindSSN, true
	case reMACV.MatchString(v):
		return schema.KindMAC, true
	case reIPv4V.MatchString(v):
		return schema.KindIPv4, true
	case reIBANV.MatchString(v):
		return schema.KindIBAN, true
	case reCardV.MatchString(v) && luhnValid(v):
		return schema.KindCard, true
	case rePhoneV.MatchString(v):
		return schema.KindPhone, true
	}
	return "", false
}

// embeddedPII matches identifiers that can appear INSIDE free text, where
// formatKind (which anchors on the whole value) would miss them.
var embeddedPII = []struct {
	re   *regexp.Regexp
	kind schema.Kind
}{
	{regexp.MustCompile(`[^\s,;:<>()"']+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`), schema.KindEmail},
	{regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), schema.KindSSN},
	{regexp.MustCompile(`\b(?:\d[ -]?){12,18}\d\b`), schema.KindCard},
	{regexp.MustCompile(`\+\d[\d\s\-()]{7,}\d`), schema.KindPhone},
	{regexp.MustCompile(`\b\d{1,3}(\.\d{1,3}){3}\b`), schema.KindIPv4},
}

// scrubEmbedded replaces PII found inside an otherwise non-personal value,
// reusing the same consistent replacement machinery.
func (m *Masker) scrubEmbedded(column, value string) string {
	out := value
	for _, p := range embeddedPII {
		out = p.re.ReplaceAllStringFunc(out, func(match string) string {
			return m.fake(column+":"+string(p.kind), match, p.kind)
		})
	}
	return out
}

// luhnValid distinguishes a real card number from an arbitrary long digit
// string, so order IDs are not mistaken for payment data.
func luhnValid(s string) bool {
	sum, alt := 0, false
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}
	return sum%10 == 0
}
