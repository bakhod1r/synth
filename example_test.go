package synth_test

import (
	"fmt"

	"github.com/bakhodir/synth"
	"github.com/google/uuid"
)

func ExampleMake() {
	type User struct {
		ID        uuid.UUID `synth:"pk"`
		FirstName string
		Email     string `synth:"email,from=FirstName"`
	}
	users := synth.Make[User](3, synth.WithSeed(42), synth.WithLocale("uz_UZ"))
	for _, u := range users {
		fmt.Println(u.FirstName, u.Email)
	}
	// Output:
	// Malika malika.abdullayeva51@example.uz
	// Azizbek azizbek.tursunov61@mail.uz
	// Shahnoza shahnoza.rashidova83@example.uz
}

func ExampleNew() {
	g := synth.New(synth.Config{Seed: 1, Locale: "uz_UZ"})
	fmt.Println(g.Currency())
	// Output: UZS
}

func ExampleRegisterSet() {
	synth.RegisterSet("planetpick", "Mars", "Venus", "Earth")
	type Row struct {
		Planet string `synth:"planetpick"`
	}
	r := synth.Make[Row](1, synth.WithSeed(7))
	_ = r
	fmt.Println("registered")
	// Output: registered
}
