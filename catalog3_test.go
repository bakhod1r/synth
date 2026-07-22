package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

func TestCatalog3Types(t *testing.T) {
	type Rec struct {
		ID          int
		OrderStatus string
		Coupon      string `synth:"couponcode"`
		Nickname    string
		LogLevel    string
		Environment string
		AWSRegion   string `synth:"awsregion"`
		IMEI        string `synth:"imei"`
		UPC         string `synth:"upc"`
		Cron        string `synth:"cron"`
		GitTag      string `synth:"gittag"`
		Priority    string
		Medal       string
	}
	for _, r := range synth.Make[Rec](200, synth.WithSeed(1)) {
		if len(r.IMEI) != 15 || !luhnOK(r.IMEI) {
			t.Fatalf("bad imei %q", r.IMEI)
		}
		if len(r.UPC) != 12 {
			t.Fatalf("bad upc %q", r.UPC)
		}
		if !strings.HasPrefix(r.GitTag, "v") {
			t.Fatalf("bad git tag %q", r.GitTag)
		}
		if strings.Count(r.Cron, " ") != 4 {
			t.Fatalf("bad cron %q", r.Cron)
		}
		if r.OrderStatus == "" || r.Nickname == "" || r.LogLevel == "" || r.Priority == "" {
			t.Fatalf("empty catalog3 field: %+v", r)
		}
	}
}

func luhnOK(s string) bool {
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
