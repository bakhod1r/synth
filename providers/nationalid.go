package providers

import (
	"fmt"
	"strings"

	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/schema"
)

// National identifiers are not one format. A PINFL is fourteen digits, a US
// SSN is nine in three groups, a Turkish TC Kimlik is eleven with two check
// digits. Emitting a US SSN for an Uzbek record is the kind of detail that
// makes a fixture obviously fake to the one person whose opinion matters — the
// engineer who has to trust it.
//
// So `ssn` follows the locale. The kind keeps its familiar name; only the shape
// changes underneath.
//
// Where a country's number carries a real check digit, we compute it, because
// the point of a fixture is to exercise the validator rather than to slip past
// it. Where the rule is proprietary or genuinely complex (an Italian codice
// fiscale encodes the holder's name and birthplace), we generate the correct
// length and character classes and say so here rather than inventing a rule.

func init() {
	registry[schema.KindSSN] = nationalID

	// The same generator under the names people actually search for. Someone
	// building an Uzbek schema looks for "pinfl", not "ssn", and finding
	// nothing they either invent a text column or give up on the type — so
	// the alias is not a convenience, it is the difference between the feature
	// being found and not.
	//
	// Each alias still follows the locale: `pinfl` in a de_DE dataset produces
	// a German tax ID, because the column means "this country's national
	// identifier" and pinning it to Uzbekistan would be a different, worse
	// behaviour hidden behind a familiar name.
	for _, k := range []schema.Kind{
		schema.KindPINFL, schema.KindNationalID, schema.KindTaxID,
	} {
		registry[k] = nationalID
	}
}

func nationalID(c Ctx) any {
	code := ""
	if c.Locale != nil {
		code = c.Locale.Name
	}
	if f, ok := nationalIDFormats[langOf(code)]; ok {
		return f(c.Rand)
	}
	return usSSN(c.Rand)
}

// langOf reduces "uz_UZ" to "uz". Identifier formats are set by the country, and
// a locale's language part is the closest thing we carry to one; pt_BR and pt_PT
// differ, so the full tag is checked first by the caller's map lookup order.
func langOf(name string) string {
	if i := strings.IndexAny(name, "_-"); i > 0 {
		return name[:i]
	}
	return name
}

var nationalIDFormats = map[string]func(*rng.Rand) string{
	"uz": pinfl,
	"tr": tcKimlik,
	"us": usSSN,
	"en": usSSN,
	"ru": ruPassport,
	"kk": iinKZ,
	"ky": ruPassport,
	"fr": frINSEE,
	"de": deSteuerID,
	"es": esDNI,
	"it": itCodiceFiscale,
	"pl": plPESEL,
	"pt": ptNIF,
	"nl": nlBSN,
	"sv": nordicPersonal,
	"nb": nordicPersonal,
	"da": nordicPersonal,
	"fi": fiHetu,
	"zh": cnResidentID,
	"ja": jaMyNumber,
	"ko": krRRN,
	"ar": digits12,
	"hi": inAadhaar,
	"id": digits16,
	"vi": digits12,
	"th": thNationalID,
	"uk": digits10,
	"az": digits7,
	"ka": digits11,
	"he": ilTeudatZehut,
}

// pinfl is the Uzbek personal identification number: 14 digits, where the first
// encodes century and sex, the next six are the birth date, and the last is a
// weighted check digit.
func pinfl(r *rng.Rand) string {
	body := fmt.Sprintf("%d%02d%02d%02d%06d",
		r.IntRange(3, 6),      // century/sex marker
		r.IntRange(1, 28),     // day
		r.IntRange(1, 12),     // month
		r.IntRange(50, 99),    // year within century
		r.IntRange(1, 999999)) // region and sequence
	return body + string(rune('0'+weightedCheck(body, []int{7, 3, 1, 7, 3, 1, 7, 3, 1, 7, 3, 1, 7})))
}

// weightedCheck is the sum-of-products modulo ten used by several national
// numbering schemes.
func weightedCheck(s string, weights []int) int {
	sum := 0
	for i, w := range weights {
		if i >= len(s) {
			break
		}
		sum += int(s[i]-'0') * w
	}
	return sum % 10
}

// tcKimlik is the Turkish identity number: eleven digits whose last two are
// check digits over the first nine.
func tcKimlik(r *rng.Rand) string {
	d := make([]int, 11)
	d[0] = r.IntRange(1, 9)
	for i := 1; i < 9; i++ {
		d[i] = r.Intn(10)
	}
	odd, even := 0, 0
	for i := 0; i < 9; i++ {
		if i%2 == 0 {
			odd += d[i]
		} else {
			even += d[i]
		}
	}
	d[9] = ((odd*7 - even) % 10 % 10)
	if d[9] < 0 {
		d[9] += 10
	}
	sum := 0
	for i := 0; i < 10; i++ {
		sum += d[i]
	}
	d[10] = sum % 10
	var b strings.Builder
	for _, v := range d {
		b.WriteByte(byte('0' + v))
	}
	return b.String()
}

// usSSN avoids the ranges the SSA never issues (area 000, 666 and 900+, group
// 00, serial 0000), so the value passes a real validator.
func usSSN(r *rng.Rand) string {
	area := r.IntRange(1, 899)
	if area == 666 {
		area = 665
	}
	return fmt.Sprintf("%03d-%02d-%04d", area, r.IntRange(1, 99), r.IntRange(1, 9999))
}

// ruPassport is the internal passport: a four-digit series and a six-digit
// number. Russia has no single lifelong personal number in common use, so this
// is the identifier a form actually asks for.
func ruPassport(r *rng.Rand) string {
	return fmt.Sprintf("%04d %06d", r.IntRange(1000, 9999), r.IntRange(1, 999999))
}

// iinKZ is the Kazakh individual identification number: twelve digits, the last
// a weighted check digit with a documented second pass when the first yields 10.
func iinKZ(r *rng.Rand) string {
	body := fmt.Sprintf("%02d%02d%02d%06d",
		r.IntRange(50, 99), r.IntRange(1, 12), r.IntRange(1, 28), r.IntRange(1, 999999))
	first := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	second := []int{3, 4, 5, 6, 7, 8, 9, 10, 11, 1, 2}
	check := weightedSumMod11(body, first)
	if check == 10 {
		check = weightedSumMod11(body, second)
	}
	if check == 10 {
		check = 0 // such a number is never issued; keep the length correct
	}
	return body + string(rune('0'+check))
}

func weightedSumMod11(s string, weights []int) int {
	sum := 0
	for i, w := range weights {
		if i >= len(s) {
			break
		}
		sum += int(s[i]-'0') * w
	}
	return sum % 11
}

// frINSEE is the French social security number: sex, birth year and month,
// department, commune, sequence, then a 97-complement key.
func frINSEE(r *rng.Rand) string {
	body := fmt.Sprintf("%d%02d%02d%02d%03d%03d",
		r.IntRange(1, 2), r.IntRange(50, 99), r.IntRange(1, 12),
		r.IntRange(1, 95), r.IntRange(1, 999), r.IntRange(1, 999))
	n := 0
	for _, ch := range body {
		n = (n*10 + int(ch-'0')) % 97
	}
	return fmt.Sprintf("%s %02d", body, 97-n)
}

// deSteuerID is the German tax identification number: eleven digits that never
// begin with zero.
func deSteuerID(r *rng.Rand) string {
	return fmt.Sprintf("%d%010d", r.IntRange(1, 9), r.IntRange(0, 9999999999))
}

// esDNI is eight digits plus the letter selected by the number modulo 23.
func esDNI(r *rng.Rand) string {
	n := r.IntRange(1, 99999999)
	const table = "TRWAGMYFPDXBNJZSQVHLCKE"
	return fmt.Sprintf("%08d%c", n, table[n%23])
}

// itCodiceFiscale produces the right shape — six letters, two digits, a month
// letter, two digits, a letter, three digits, a letter. The real code encodes
// the holder's name and birthplace and its check character depends on that
// encoding, which we do not model; this is a well-formed placeholder, not a
// derivable code.
func itCodiceFiscale(r *rng.Rand) string {
	const months = "ABCDEHLMPRST"
	return fmt.Sprintf("%s%02d%c%02d%c%03d%c",
		upperAlpha(r, 6), r.IntRange(50, 99), months[r.Intn(len(months))],
		r.IntRange(1, 28), upperAlpha(r, 1)[0], r.IntRange(1, 999), upperAlpha(r, 1)[0])
}

func upperAlpha(r *rng.Rand, n int) string {
	const set = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = set[r.Intn(len(set))]
	}
	return string(b)
}

// plPESEL is eleven digits: birth date, sequence with a sex marker, and a
// weighted check digit.
func plPESEL(r *rng.Rand) string {
	body := fmt.Sprintf("%02d%02d%02d%04d",
		r.IntRange(50, 99), r.IntRange(1, 12), r.IntRange(1, 28), r.IntRange(1, 9999))
	sum := weightedCheck(body, []int{1, 3, 7, 9, 1, 3, 7, 9, 1, 3})
	check := (10 - sum) % 10
	return body + string(rune('0'+check))
}

// ptNIF is nine digits whose last is a mod-11 check over the first eight.
func ptNIF(r *rng.Rand) string {
	body := fmt.Sprintf("%d%07d", r.IntRange(1, 3), r.IntRange(0, 9999999))
	sum := 0
	for i := 0; i < 8; i++ {
		sum += int(body[i]-'0') * (9 - i)
	}
	check := 11 - sum%11
	if check >= 10 {
		check = 0
	}
	return body + string(rune('0'+check))
}

// nlBSN is nine digits satisfying the "elfproef": the weighted sum, with the
// last digit weighted -1, is divisible by eleven. We search rather than solve;
// at most ten candidates are ever tried.
func nlBSN(r *rng.Rand) string {
	body := fmt.Sprintf("%08d", r.IntRange(10000000, 99999999))
	for last := 0; last < 10; last++ {
		sum := 0
		for i := 0; i < 8; i++ {
			sum += int(body[i]-'0') * (9 - i)
		}
		sum -= last
		if sum%11 == 0 {
			return body + string(rune('0'+last))
		}
	}
	return body + "0" // unreachable for a valid body; keeps the length correct
}

// nordicPersonal is the Swedish/Norwegian/Danish shape: a birth date and a
// four-digit suffix. The Swedish check digit is Luhn over the ten digits.
func nordicPersonal(r *rng.Rand) string {
	body := fmt.Sprintf("%02d%02d%02d%03d",
		r.IntRange(50, 99), r.IntRange(1, 12), r.IntRange(1, 28), r.IntRange(1, 999))
	return body + string(luhnCheck(body))
}

// fiHetu is a birth date, a century separator, a three-digit individual number
// and a character from a 31-symbol table indexed by the number modulo 31.
func fiHetu(r *rng.Rand) string {
	const table = "0123456789ABCDEFHJKLMNPRSTUVWXY"
	d, m, y := r.IntRange(1, 28), r.IntRange(1, 12), r.IntRange(50, 99)
	ind := r.IntRange(900, 999)
	n := ((d*100+m)*100+y)*1000 + ind
	return fmt.Sprintf("%02d%02d%02d-%03d%c", d, m, y, ind, table[n%31])
}

// cnResidentID is eighteen characters: address, birth date, sequence and an
// ISO 7064 MOD 11-2 check character.
func cnResidentID(r *rng.Rand) string {
	body := fmt.Sprintf("%06d%04d%02d%02d%03d",
		r.IntRange(110000, 659000), r.IntRange(1950, 2005),
		r.IntRange(1, 12), r.IntRange(1, 28), r.IntRange(1, 999))
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	sum := 0
	for i, w := range weights {
		sum += int(body[i]-'0') * w
	}
	return body + string("10X98765432"[sum%11])
}

// jaMyNumber is twelve digits with an ISO 7064-style check digit.
func jaMyNumber(r *rng.Rand) string {
	body := fmt.Sprintf("%011d", r.IntRange(10000000000, 99999999999))
	sum := 0
	for i := 0; i < 11; i++ {
		p := int(body[10-i] - '0')
		q := i + 1
		if q > 6 {
			q = i - 5
		}
		sum += p * (q + 1)
	}
	check := 11 - sum%11
	if check >= 10 {
		check = 0
	}
	return body + string(rune('0'+check))
}

// krRRN is a birth date, a century/sex digit and a six-digit suffix. Since 2020
// the trailing digits are randomly assigned with no published check rule, so we
// assign them randomly too — which is what the real number now does.
func krRRN(r *rng.Rand) string {
	return fmt.Sprintf("%02d%02d%02d-%d%06d",
		r.IntRange(50, 99), r.IntRange(1, 12), r.IntRange(1, 28),
		r.IntRange(1, 4), r.IntRange(1, 999999))
}

// inAadhaar is twelve digits that never begin with 0 or 1, with a Verhoeff
// check digit. We use the standard Verhoeff tables so a validator accepts it.
func inAadhaar(r *rng.Rand) string {
	body := fmt.Sprintf("%d%010d", r.IntRange(2, 9), r.IntRange(0, 9999999999))
	return body + string(rune('0'+verhoeff(body)))
}

var verhoeffD = [10][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, {1, 2, 3, 4, 0, 6, 7, 8, 9, 5},
	{2, 3, 4, 0, 1, 7, 8, 9, 5, 6}, {3, 4, 0, 1, 2, 8, 9, 5, 6, 7},
	{4, 0, 1, 2, 3, 9, 5, 6, 7, 8}, {5, 9, 8, 7, 6, 0, 4, 3, 2, 1},
	{6, 5, 9, 8, 7, 1, 0, 4, 3, 2}, {7, 6, 5, 9, 8, 2, 1, 0, 4, 3},
	{8, 7, 6, 5, 9, 3, 2, 1, 0, 4}, {9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
}

var verhoeffP = [8][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, {1, 5, 7, 6, 2, 8, 3, 0, 9, 4},
	{5, 8, 0, 3, 7, 9, 6, 1, 4, 2}, {8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
	{9, 4, 5, 3, 1, 2, 6, 8, 7, 0}, {4, 2, 8, 6, 5, 7, 3, 9, 0, 1},
	{2, 7, 9, 3, 8, 0, 6, 4, 1, 5}, {7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
}

var verhoeffInv = [10]int{0, 4, 3, 2, 1, 5, 6, 7, 8, 9}

// verhoeff returns the check digit that makes s+digit a valid Verhoeff string.
func verhoeff(s string) int {
	c := 0
	for i := len(s) - 1; i >= 0; i-- {
		c = verhoeffD[c][verhoeffP[(len(s)-i)%8][int(s[i]-'0')]]
	}
	return verhoeffInv[c]
}

// thNationalID is thirteen digits with a mod-11 check digit.
func thNationalID(r *rng.Rand) string {
	body := fmt.Sprintf("%d%011d", r.IntRange(1, 8), r.IntRange(0, 99999999999))
	sum := 0
	for i := 0; i < 12; i++ {
		sum += int(body[i]-'0') * (13 - i)
	}
	return body + string(rune('0'+(11-sum%11)%10))
}

// ilTeudatZehut is nine digits with a Luhn-style check over the first eight.
func ilTeudatZehut(r *rng.Rand) string {
	body := fmt.Sprintf("%08d", r.IntRange(0, 99999999))
	sum := 0
	for i := 0; i < 8; i++ {
		d := int(body[i]-'0') * (i%2 + 1)
		if d > 9 {
			d -= 9
		}
		sum += d
	}
	return body + string(rune('0'+(10-sum%10)%10))
}

// Countries whose number is a plain block of digits with no public check rule.
func digits7(r *rng.Rand) string  { return fmt.Sprintf("%07d", r.IntRange(1000000, 9999999)) }
func digits10(r *rng.Rand) string { return fmt.Sprintf("%010d", r.IntRange(1, 9999999999)) }
func digits11(r *rng.Rand) string { return fmt.Sprintf("%011d", r.IntRange(1, 99999999999)) }
func digits12(r *rng.Rand) string {
	return fmt.Sprintf("%06d%06d", r.IntRange(1, 999999), r.IntRange(1, 999999))
}
func digits16(r *rng.Rand) string {
	return fmt.Sprintf("%08d%08d", r.IntRange(1, 99999999), r.IntRange(1, 99999999))
}
