package benchcmp_test

import (
	"testing"

	"github.com/bakhodir/synth"
	gofaker "github.com/go-faker/faker/v4"
	jaswdr "github.com/jaswdr/faker/v2"
)

// BenchUser is deliberately plain so all three libraries do comparable work:
// fill four string fields from a struct definition.
type BenchUser struct {
	Name  string
	Email string
	Phone string
	City  string
}

// go-faker uses its own tag namespace; a separate type keeps the comparison
// honest (each library gets the struct shape it expects).
type goFakerUser struct {
	Name  string `faker:"name"`
	Email string `faker:"email"`
	Phone string `faker:"phone_number"`
	City  string `faker:"word"`
}

func BenchmarkSynth_Struct(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = synth.Make[BenchUser](1, synth.WithSeed(uint64(i)))
	}
}

// Batch generation is Synth's normal mode: schema work happens once for the
// whole run instead of once per record.
func BenchmarkSynth_Batch1000(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = synth.Make[BenchUser](1000, synth.WithSeed(1))
	}
}

func BenchmarkGoFaker_Struct(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var u goFakerUser
		if err := gofaker.FakeData(&u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJaswdr_Fields(b *testing.B) {
	f := jaswdr.New()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var u BenchUser
		u.Name = f.Person().Name()
		u.Email = f.Internet().Email()
		u.Phone = f.Phone().Number()
		u.City = f.Address().City()
		_ = u
	}
}

// Synth's fluent generator is the closest match to jaswdr's per-field API.
func BenchmarkSynth_Fields(b *testing.B) {
	g := synth.New(synth.Config{Seed: 1})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var u BenchUser
		u.Name = g.Name()
		u.Email = g.Email()
		u.Phone = g.Phone()
		u.City = g.City()
		_ = u
	}
}
