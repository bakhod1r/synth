package synth_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/bakhodir/synth"
)

func TestFakerExtraTypes(t *testing.T) {
	type Rec struct {
		ID        int
		Word      string
		Sentence  string
		Paragraph string `synth:"paragraph"`
		IPv6      string `synth:"ipv6"`
		Domain    string
		Latitude  float64
		Longitude float64
		Weekday   string
		BloodType string
		UserAgent string
		SSN       string
		Timezone  string
		ImageURL  string `synth:"imageurl"`
	}
	for _, r := range synth.Make[Rec](200, synth.WithSeed(1)) {
		if r.Word == "" || r.Sentence == "" || r.Paragraph == "" {
			t.Fatal("empty lorem field")
		}
		if strings.Count(r.IPv6, ":") != 7 {
			t.Fatalf("bad ipv6 %q", r.IPv6)
		}
		if r.Latitude < -90 || r.Latitude > 90 {
			t.Fatalf("latitude out of range: %v", r.Latitude)
		}
		if r.Longitude < -180 || r.Longitude > 180 {
			t.Fatalf("longitude out of range: %v", r.Longitude)
		}
		if len(strings.Split(r.SSN, "-")) != 3 {
			t.Fatalf("bad ssn %q", r.SSN)
		}
		if !strings.HasPrefix(r.ImageURL, "https://") {
			t.Fatalf("bad image url %q", r.ImageURL)
		}
		if !strings.Contains(r.Domain, ".") {
			t.Fatalf("bad domain %q", r.Domain)
		}
	}
}

// IPs must fall in the locale's allocated first-octet blocks.
func TestCountryMatchingIP(t *testing.T) {
	type Rec struct {
		ID int
		IP string
	}
	uzBlocks := map[string]bool{"84": true, "213": true, "195": true, "217": true}
	for _, r := range synth.Make[Rec](300, synth.WithSeed(2), synth.WithLocale("uz_UZ")) {
		first := strings.SplitN(r.IP, ".", 2)[0]
		if _, err := strconv.Atoi(first); err != nil {
			t.Fatalf("bad ip %q", r.IP)
		}
		if !uzBlocks[first] {
			t.Fatalf("uz IP %q not in allocated blocks", r.IP)
		}
	}
}
