package synth_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bakhodir/synth"
	"github.com/bakhodir/synth/locale"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `synth:"pk"`
	FirstName string
	Email     string `synth:"email,from=FirstName"`
	Phone     string
	Region    string
	City      string
	Postcode  string
	Age       int
	IsActive  bool
	CreatedAt time.Time
}

type Order struct {
	ID     uuid.UUID `synth:"pk"`
	UserID uuid.UUID
	Amount float64 `synth:"float,min=1000,max=500000"`
}

func TestMakeBasic(t *testing.T) {
	users := synth.Make[User](100, synth.WithSeed(42), synth.WithLocale("uz_UZ"))
	if len(users) != 100 {
		t.Fatalf("want 100, got %d", len(users))
	}
	for _, u := range users {
		if u.FirstName == "" || u.Email == "" || u.Phone == "" {
			t.Fatalf("empty inferred field: %+v", u)
		}
		if !strings.HasPrefix(u.Phone, "+998") {
			t.Fatalf("uz phone expected, got %q", u.Phone)
		}
		if u.ID == uuid.Nil {
			t.Fatalf("uuid not filled")
		}
	}
}

// Coherence: postcode must belong to the record's city (same locale.Place).
func TestCoherenceCityPostcode(t *testing.T) {
	users := synth.Make[User](500, synth.WithSeed(7), synth.WithLocale("uz_UZ"))
	valid := map[string]string{}
	for _, p := range locale.Get("uz_UZ").Places {
		valid[p.City] = p.Postcode
	}
	for _, u := range users {
		if valid[u.City] != u.Postcode {
			t.Fatalf("city %q got postcode %q, want %q", u.City, u.Postcode, valid[u.City])
		}
	}
}

// Determinism: same seed → identical output.
func TestDeterminism(t *testing.T) {
	a := synth.Make[User](50, synth.WithSeed(99))
	b := synth.Make[User](50, synth.WithSeed(99))
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("record %d differs between runs", i)
		}
	}
}

// Referential integrity: every Order.UserID exists in users.
func TestRefIntegrity(t *testing.T) {
	users := synth.Make[User](20, synth.WithSeed(1))
	orders := synth.Make[Order](500, synth.WithSeed(2), synth.Ref(users, "UserID"))
	valid := map[uuid.UUID]bool{}
	for _, u := range users {
		valid[u.ID] = true
	}
	for _, o := range orders {
		if !valid[o.UserID] {
			t.Fatalf("order references non-existent user %v", o.UserID)
		}
	}
}

func TestTaglessInferenceAndWarnings(t *testing.T) {
	type Plain struct {
		Name  string
		Email string
		Junk  chan int // uninferable
	}
	w := synth.Warnings[Plain]()
	if len(w) != 1 || w[0].Field != "Junk" {
		t.Fatalf("expected one warning for Junk, got %+v", w)
	}
}

func TestCycleError(t *testing.T) {
	type Cyclic struct {
		A string `synth:"lorem,from=B"`
		B string `synth:"lorem,from=A"`
	}
	_, err := synth.TryMake[Cyclic](1)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestEncoders(t *testing.T) {
	dir := t.TempDir()
	users := synth.Make[User](10, synth.WithSeed(3))
	csvPath := filepath.Join(dir, "u.csv")
	if err := synth.WriteCSV(csvPath, users); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(csvPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 11 { // header + 10
		t.Fatalf("want 11 csv lines, got %d", len(lines))
	}
	sqlPath := filepath.Join(dir, "u.sql")
	if err := synth.WriteSQL(sqlPath, "users", users); err != nil {
		t.Fatal(err)
	}
	sql, _ := os.ReadFile(sqlPath)
	if !strings.Contains(string(sql), "INSERT INTO users") {
		t.Fatalf("sql missing insert: %s", sql)
	}
}

func TestStreamConstantMemory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.jsonl")
	if err := synth.Stream[User](5000, synth.WithSeed(5)).ToJSONL(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if n := strings.Count(string(data), "\n"); n != 5000 {
		t.Fatalf("want 5000 jsonl lines, got %d", n)
	}
}
