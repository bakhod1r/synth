package synth_test

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth"
)

func TestCatalog2Types(t *testing.T) {
	type Rec struct {
		ID            int
		MiddleName    string
		MaritalStatus string
		Education     string
		BankName      string
		PaymentMethod string
		Swift         string `synth:"swift"`
		VIN           string `synth:"vin"`
		MD5           string `synth:"md5"`
		SHA256        string `synth:"sha256"`
		JWT           string `synth:"jwt"`
		GitCommit     string `synth:"gitcommit"`
		Port          int    `synth:"port"`
		Rating        float64
		Season        string
		Element       string
		SKU           string `synth:"sku"`
	}
	for _, r := range synth.Make[Rec](200, synth.WithSeed(1)) {
		if len(r.MD5) != 32 {
			t.Fatalf("md5 length %d", len(r.MD5))
		}
		if len(r.SHA256) != 64 {
			t.Fatalf("sha256 length %d", len(r.SHA256))
		}
		if len(r.GitCommit) != 40 {
			t.Fatalf("gitcommit length %d", len(r.GitCommit))
		}
		if len(r.VIN) != 17 {
			t.Fatalf("vin length %d", len(r.VIN))
		}
		if strings.Count(r.JWT, ".") != 2 {
			t.Fatalf("jwt segments %q", r.JWT)
		}
		if r.Port < 1024 || r.Port > 65535 {
			t.Fatalf("port out of range %d", r.Port)
		}
		if r.Rating < 1.0 || r.Rating > 5.0 {
			t.Fatalf("rating out of range %v", r.Rating)
		}
		if r.MiddleName == "" || r.Season == "" || r.Element == "" || r.BankName == "" {
			t.Fatalf("empty catalog field: %+v", r)
		}
	}
}
