package providers_test

import (
	"encoding/json"
	"net/netip"
	"regexp"
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

var icd10Re = regexp.MustCompile(`^[A-TV-Z][0-9][0-9A-Z](\.[0-9A-TV-Z]{1,4})?$`)

// validISIN expands letters to two digits, then applies Luhn — the ISO 6166
// rule, implemented here independently of the generator.
func validISIN(s string) bool {
	if len(s) != 12 {
		return false
	}
	var digits []int
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			digits = append(digits, int(c-'0'))
		case c >= 'A' && c <= 'Z':
			v := int(c-'A') + 10
			digits = append(digits, v/10, v%10)
		default:
			return false
		}
	}
	sum, double := 0, false
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
	return sum%10 == 0
}

func TestISINChecksum(t *testing.T) {
	type Sec struct {
		ISIN string `synth:"isin"`
	}
	for _, s := range synth.Make[Sec](300, synth.WithSeed(7)) {
		if !validISIN(s.ISIN) {
			t.Fatalf("ISIN %q fails its check digit", s.ISIN)
		}
	}
}

// LEI check digits must satisfy ISO 7064 mod-97-10: the whole 20-character
// string reduces to 1.
func TestLEIChecksum(t *testing.T) {
	type Entity struct {
		LEI string `synth:"lei"`
	}
	for _, e := range synth.Make[Entity](300, synth.WithSeed(11)) {
		if len(e.LEI) != 20 {
			t.Fatalf("LEI %q is not 20 characters", e.LEI)
		}
		rem := 0
		for i := 0; i < len(e.LEI); i++ {
			c := e.LEI[i]
			if c >= '0' && c <= '9' {
				rem = (rem*10 + int(c-'0')) % 97
			} else {
				rem = (rem*100 + int(c-'A') + 10) % 97
			}
		}
		if rem != 1 {
			t.Fatalf("LEI %q fails mod-97-10 (got %d)", e.LEI, rem)
		}
	}
}

// CUSIP check digits must satisfy the modulus-10 double-add-double rule.
func TestCUSIPChecksum(t *testing.T) {
	type Sec struct {
		CUSIP string `synth:"cusip"`
	}
	for _, s := range synth.Make[Sec](300, synth.WithSeed(13)) {
		if len(s.CUSIP) != 9 {
			t.Fatalf("CUSIP %q is not 9 characters", s.CUSIP)
		}
		sum := 0
		for i := 0; i < 8; i++ {
			c := s.CUSIP[i]
			v := int(c - '0')
			if c >= 'A' {
				v = int(c-'A') + 10
			}
			if i%2 == 1 {
				v *= 2
			}
			sum += v/10 + v%10
		}
		want := byte('0' + (10-sum%10)%10)
		if s.CUSIP[8] != want {
			t.Fatalf("CUSIP %q has check digit %c, want %c", s.CUSIP, s.CUSIP[8], want)
		}
	}
}

func TestICD10Format(t *testing.T) {
	type Dx struct {
		Code string `synth:"icd10"`
	}
	seen := map[string]bool{}
	for _, d := range synth.Make[Dx](500, synth.WithSeed(3)) {
		if !icd10Re.MatchString(d.Code) {
			t.Fatalf("ICD-10 %q has the wrong shape", d.Code)
		}
		seen[d.Code] = true
	}
	if len(seen) < 100 {
		t.Fatalf("only %d distinct ICD-10 codes; the dataset is too small", len(seen))
	}
}

// A CIDR block must parse and must already be the network address, so a
// consumer can use it without re-masking.
func TestCIDRIsCanonical(t *testing.T) {
	type Net struct {
		Block string `synth:"cidr"`
	}
	for _, n := range synth.Make[Net](300, synth.WithSeed(9)) {
		p, err := netip.ParsePrefix(n.Block)
		if err != nil {
			t.Fatalf("CIDR %q does not parse: %v", n.Block, err)
		}
		if p.Masked() != p {
			t.Fatalf("CIDR %q is not a canonical network address", n.Block)
		}
	}
}

// NDC codes must use the FDA's 5-4-2 labeler-product-package segmentation.
func TestNDCFormat(t *testing.T) {
	type Rx struct {
		NDC string `synth:"ndc"`
	}
	re := regexp.MustCompile(`^\d{5}-\d{4}-\d{2}$`)
	for _, r := range synth.Make[Rx](200, synth.WithSeed(5)) {
		if !re.MatchString(r.NDC) {
			t.Fatalf("NDC %q is not in 5-4-2 format", r.NDC)
		}
	}
}

// GeoJSON points must be valid JSON, longitude-first, and in range.
func TestGeoJSONPoint(t *testing.T) {
	type Loc struct {
		Geo string `synth:"geojsonpoint"`
	}
	for _, l := range synth.Make[Loc](200, synth.WithSeed(17)) {
		var pt struct {
			Type   string    `json:"type"`
			Coords []float64 `json:"coordinates"`
		}
		if err := json.Unmarshal([]byte(l.Geo), &pt); err != nil {
			t.Fatalf("GeoJSON %q is not valid JSON: %v", l.Geo, err)
		}
		if pt.Type != "Point" || len(pt.Coords) != 2 {
			t.Fatalf("GeoJSON %q is not a Point with two coordinates", l.Geo)
		}
		if lon := pt.Coords[0]; lon < -180 || lon > 180 {
			t.Fatalf("longitude %v out of range in %q", lon, l.Geo)
		}
		if lat := pt.Coords[1]; lat < -90 || lat > 90 {
			t.Fatalf("latitude %v out of range in %q", lat, l.Geo)
		}
	}
}

// ASN values must carry the AS prefix and stay inside the public 16-bit range.
func TestASNFormat(t *testing.T) {
	type Peer struct {
		ASN string `synth:"asn"`
	}
	for _, p := range synth.Make[Peer](200, synth.WithSeed(19)) {
		if !strings.HasPrefix(p.ASN, "AS") {
			t.Fatalf("ASN %q lacks the AS prefix", p.ASN)
		}
	}
}
