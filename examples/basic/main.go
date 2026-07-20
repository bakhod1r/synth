// Command basic demonstrates Synth as a pure data provider: structs in,
// coherent records out — to memory, a file, and a stream.
package main

import (
	"fmt"
	"time"

	"github.com/bakhodir/synth"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `synth:"pk"`
	FirstName string
	Email     string `synth:"email,from=FirstName"` // email derives from the name
	Phone     string
	Country   string
	Region    string
	City      string
	Postcode  string // stays coherent with Country/Region/City
	Card      string `synth:"card"` // Luhn-valid HUMO/UZCARD
	CreatedAt time.Time
}

type Order struct {
	ID     uuid.UUID `synth:"pk"`
	UserID uuid.UUID // filled via Ref → always a real user
	Amount float64   `synth:"float,min=1000,max=500000"`
}

func main() {
	users := synth.Make[User](5, synth.WithSeed(42), synth.WithLocale("uz_UZ"))
	for _, u := range users {
		fmt.Printf("%s | %s | %s | %s, %s/%s %s | %s\n",
			u.FirstName, u.Email, u.Phone, u.Country, u.Region, u.City, u.Postcode, u.Card)
	}

	// Referential integrity: orders point at real users.
	orders := synth.Make[Order](10, synth.WithSeed(7), synth.Ref(users, "UserID"))
	fmt.Printf("\ngenerated %d orders, first UserID=%v\n", len(orders), orders[0].UserID)

	// Stream a million rows to CSV in constant memory (writes to /tmp).
	if err := synth.Stream[User](1_000_000, synth.WithLocale("uz_UZ")).ToCSV("/tmp/synth_users.csv"); err != nil {
		panic(err)
	}
	fmt.Println("\nstreamed 1,000,000 users to /tmp/synth_users.csv")
}
