// Command localize shows two things at once: a complex struct with a nested
// sub-struct, and locale coherence — the same schema rendered per locale, where
// name, phone, address and card all agree with the chosen region.
package main

import (
	"fmt"
	"time"

	"github.com/bakhod1r/synth"
	"github.com/google/uuid"
)

// Address is a nested struct: Synth generates it recursively, and every field
// stays coherent with the record's locale.
type Address struct {
	Country  string
	Region   string
	City     string
	Postcode string
}

// Customer is the complex record — a scalar UUID key, a derived email, a
// locale-valid phone and card, and the nested Address above.
type Customer struct {
	ID        uuid.UUID `synth:"pk"`
	FirstName string
	Email     string `synth:"email,from=FirstName"`
	Phone     string
	Card      string `synth:"card"`
	Address   Address
	CreatedAt time.Time
}

func main() {
	for _, loc := range []string{"uz_UZ", "ja_JP", "de_DE"} {
		c := synth.Make[Customer](1, synth.WithSeed(42), synth.WithLocale(loc))[0]
		fmt.Printf("[%s]\n", loc)
		fmt.Printf("  %-10s %s <%s>\n", "name:", c.FirstName, c.Email)
		fmt.Printf("  %-10s %s\n", "phone:", c.Phone)
		fmt.Printf("  %-10s %s\n", "card:", c.Card)
		fmt.Printf("  %-10s %s, %s/%s %s\n\n", "address:",
			c.Address.Country, c.Address.Region, c.Address.City, c.Address.Postcode)
	}
}
