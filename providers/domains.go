package providers

import (
	"fmt"
	"net/netip"

	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/schema"
)

// Regulated-domain identifiers: healthcare, finance and networking. Every
// code here either comes from a real published list or carries a check digit
// computed by the same algorithm the real issuer uses — never a hardcoded
// checksum, so the values validate in downstream systems.

func init() {
	set(schema.KindICD10, icd10Codes)
	set(schema.KindDrugName, drugNames)
	set(schema.KindMACVendor, macVendors)

	registry[schema.KindISIN] = func(c Ctx) any { return isin(c.Rand) }
	registry[schema.KindLEI] = func(c Ctx) any { return lei(c.Rand) }
	registry[schema.KindCUSIP] = func(c Ctx) any { return cusip(c.Rand) }
	registry[schema.KindCIDR] = func(c Ctx) any { return cidr(c.Rand) }
	registry[schema.KindASN] = func(c Ctx) any { return fmt.Sprintf("AS%d", c.Rand.IntRange(1, 64495)) }
	registry[schema.KindNDC] = func(c Ctx) any { return ndc(c.Rand) }
	registry[schema.KindGeoJSONPoint] = func(c Ctx) any { return geoJSONPoint(c.Rand) }
}

// isin builds a valid ISIN: a real numbering-agency country prefix, a
// nine-character NSIN, and the ISO 6166 check digit.
func isin(r *rng.Rand) string {
	body := pick(r, isinPrefixes) + upperAlnum(r, 9)
	return body + string(rune('0'+isinCheck(body)))
}

// isinCheck expands each letter to its two-digit value, then applies Luhn
// over the resulting digit string.
func isinCheck(body string) int {
	var digits []int
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if ch >= '0' && ch <= '9' {
			digits = append(digits, int(ch-'0'))
			continue
		}
		v := int(ch-'A') + 10
		digits = append(digits, v/10, v%10)
	}
	sum, double := 0, true
	for i := len(digits) - 1; i >= 0; i-- {
		d := digits[i]
		if double {
			if d *= 2; d > 9 {
				d -= 9
			}
		}
		double = !double
		sum += d
	}
	return (10 - sum%10) % 10
}

// lei builds a valid ISO 17442 legal entity identifier: an LOU prefix, two
// reserved zeros, a twelve-character entity part, and mod-97-10 check digits.
func lei(r *rng.Rand) string {
	body := upperAlnum(r, 4) + "00" + upperAlnum(r, 12)
	return body + fmt.Sprintf("%02d", mod97Check(body))
}

// mod97Check computes the two ISO 7064 mod-97-10 check digits for body.
func mod97Check(body string) int {
	rem := 0
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if ch >= '0' && ch <= '9' {
			rem = (rem*10 + int(ch-'0')) % 97
		} else {
			rem = (rem*100 + int(ch-'A') + 10) % 97
		}
	}
	return 98 - (rem*100)%97
}

// cusip builds a valid nine-character CUSIP: an eight-character issue
// identifier plus the modulus-10 "double add double" check digit.
func cusip(r *rng.Rand) string {
	body := upperAlnum(r, 8)
	return body + string(rune('0'+cusipCheck(body)))
}

func cusipCheck(body string) int {
	sum := 0
	for i := 0; i < len(body); i++ {
		ch := body[i]
		var v int
		if ch >= '0' && ch <= '9' {
			v = int(ch - '0')
		} else {
			v = int(ch-'A') + 10
		}
		if i%2 == 1 {
			v *= 2
		}
		sum += v/10 + v%10
	}
	return (10 - sum%10) % 10
}

// ndc returns a National Drug Code in the 5-4-2 labeler-product-package
// format the FDA directory publishes.
func ndc(r *rng.Rand) string {
	return fmt.Sprintf("%s-%s-%s", r.Digits(5), r.Digits(4), r.Digits(2))
}

// cidr returns a canonical private-range network block. Host bits are
// cleared, so the address is always the network address and the string
// round-trips through netip.ParsePrefix unchanged.
func cidr(r *rng.Rand) string {
	bits := r.IntRange(8, 24)
	addr := netip.AddrFrom4([4]byte{
		10, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)),
	})
	return netip.PrefixFrom(addr, bits).Masked().String()
}

// geoJSONPoint returns an RFC 7946 Point geometry. GeoJSON orders
// coordinates longitude-first, which is the usual source of bugs.
func geoJSONPoint(r *rng.Rand) string {
	lon := r.Float64()*360 - 180
	lat := r.Float64()*180 - 90
	return fmt.Sprintf(`{"type":"Point","coordinates":[%.6f,%.6f]}`, lon, lat)
}

// isinPrefixes are ISO 3166-1 alpha-2 codes of real national numbering
// agencies that issue ISINs.
var isinPrefixes = []string{
	"US", "GB", "DE", "FR", "JP", "CH", "NL", "CA", "AU", "IT",
	"ES", "SE", "NO", "DK", "FI", "BE", "AT", "IE", "LU", "SG",
	"HK", "KR", "BR", "MX", "ZA", "IN", "CN", "PL", "PT", "GR",
}
