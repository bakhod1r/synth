package synth_test

import (
	"testing"

	"github.com/bakhod1r/devicex"
	"github.com/bakhod1r/synth"
)

type session struct {
	Code  string `synth:"device_code"`
	Brand string `synth:"device_brand,from=Code"`
	Name  string `synth:"device_name,from=Code"`
}

// The point of the catalogue: the three fields describe one handset, not three.
func TestDeviceFieldsAgreeWithinRecord(t *testing.T) {
	for _, s := range synth.Make[session](200, synth.WithSeed(1)) {
		d, ok := devicex.Lookup(s.Code)
		if !ok {
			t.Fatalf("generated code %q is not in the catalogue", s.Code)
		}
		if s.Brand != d.Brand || s.Name != d.Name {
			t.Errorf("code %q: got %q/%q, catalogue says %q/%q",
				s.Code, s.Brand, s.Name, d.Brand, d.Name)
		}
	}
}

func TestDeviceSameSeedSameDevices(t *testing.T) {
	a := synth.Make[session](50, synth.WithSeed(7))
	b := synth.Make[session](50, synth.WithSeed(7))
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("record %d differs between runs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestDeviceDifferentSeedsDiffer(t *testing.T) {
	a := synth.Make[session](50, synth.WithSeed(1))
	b := synth.Make[session](50, synth.WithSeed(2))
	same := 0
	for i := range a {
		if a[i] == b[i] {
			same++
		}
	}
	if same == len(a) {
		t.Fatal("two seeds produced identical devices for every record")
	}
}

// A brand field with no from= has no code to agree with, so it draws its own
// device rather than returning nothing.
func TestDeviceBrandWithoutFromStillGenerates(t *testing.T) {
	type loose struct {
		Brand string `synth:"device_brand"`
	}
	for _, r := range synth.Make[loose](20, synth.WithSeed(3)) {
		if r.Brand == "" {
			t.Fatal("device_brand without from= produced an empty brand")
		}
	}
}

// The catalogue is the only source: an unknown code is never given a name.
func TestDeviceCodeStaysInCatalogue(t *testing.T) {
	for _, r := range synth.Make[session](100, synth.WithSeed(11)) {
		if _, ok := devicex.Lookup(r.Code); !ok {
			t.Fatalf("code %q left the catalogue", r.Code)
		}
	}
}

func BenchmarkDeviceRecord(b *testing.B) {
	synth.Make[session](1, synth.WithSeed(1)) // warm the flattened catalogue
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		synth.Make[session](1, synth.WithSeed(uint64(i)))
	}
}
