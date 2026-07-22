# Truth, Depth, CLI — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use `- [ ]` checkboxes.

**Goal:** Make the docs true, close the catalog/locale gaps, and expose every capability through the CLI.

**Architecture:** Tasks 1–5 need no architectural change: Task 1 is a docs-only correction, Tasks 2–4 add providers/locale data, Task 5 extends `cmd/synth`. Tasks 6–9 add four new packages — `constraint` (mines invariants from a sample and enforces them during generation), `verify` (the inverse product: audits an existing dataset), `snapshot` (materializes any point in time from one seed), and `ui` (a localhost-only browser workbench).

**Tech Stack:** Go 1.25, existing `providers`, `locale`, `mask`, `cdc`, `sink/parquet` packages.

## Global Constraints

- **No database connections, no network calls.** Synth is a pure provider: it reads and writes files only. This is a hard user boundary and applies to every task.
- Core module dependencies stay at exactly two: `google/uuid`, `yaml.v3`. Anything heavier goes in a submodule (as `sink/parquet` does).
- `go.mod` stays `go 1.25` (CI pins 1.25).
- All datasets must be **real, curated values** — no combinatorial synthesis, no placeholder examples.
- Every locale name bank is gender-split: male first names pair with male surnames, female with female.
- `gofmt`, `go vet ./...`, and `go test -race ./...` must pass before each commit.
- Do not push docs.

---

### Task 1: Make the README true

The README advertises sinks that do not exist and contradicts itself. `README.md:9`, `:19`, `:98` claim "Kafka, Postgres (batched `COPY`), CSV, JSONL out of the box"; `README.md:385` says "Network sinks (Kafka, Postgres) are intentionally **out of scope**". The code has only `WriteCSV`, `WriteJSONL`, `WriteSQL` (`encode.go:15,46,67`), `WriteCDC`, and `sink/parquet`. Under the project's no-network boundary, `:385` is the correct statement — the marketing lines are wrong.

**Files:**
- Modify: `README.md:9`, `README.md:19`, `README.md:98`, `README.md:265`

- [x] **Step 1: Fix the opening claim (`README.md:9`)**

Replace "It streams millions of records straight to Kafka, Postgres, CSV, or JSONL with minimal memory usage" with:

```
It streams millions of records to CSV, JSONL, SQL `INSERT` files, or Parquet
with minimal memory usage, and can produce valid request payloads from OpenAPI
schemas.
```

- [x] **Step 2: Fix the comparison table row (`README.md:19`)**

Change the Output row's right-hand cell from `Kafka, Postgres, CSV, JSONL sinks` to:

```
CSV, JSONL, SQL, Parquet, CDC files
```

- [x] **Step 3: Rewrite the sinks section (`README.md:98`)**

Replace the "Kafka, Postgres (batched `COPY`), CSV, and JSONL out of the box, behind one interface" sentence with:

```
CSV, JSONL, SQL `INSERT` files, Parquet, and Debezium-shaped CDC events —
every one of them a file. Synth never opens a database or network connection;
handing the file to your loader is the last step, and it is yours. See
[Scope](#scope) for why.
```

- [x] **Step 4: Verify no contradiction remains**

Run: `grep -n "Kafka\|COPY\|Postgres" README.md`
Expected: only the `:265` line ("no database, no Kafka, just a file") and the
`:385` scope paragraph — both of which *deny* network sinks. No line may claim
Synth writes to Kafka or Postgres.

- [x] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: correct sink claims to match the code"
```

---

### Task 2: Domain type catalog — healthcare, finance, network, geo

The catalog is broad but shallow in regulated domains. Add four groups of real, checksum-valid identifiers.

**Files:**
- Create: `providers/domains.go`
- Create: `providers/domains_test.go`
- Modify: `schema/schema.go` (new `Kind` constants)
- Modify: `providers/providers.go` (register the new kinds)
- Modify: `register.go` (field-name synonyms, e.g. `icd10`, `isin`, `cidr`)

**Interfaces:**
- Consumes: `providers.Ctx{Rand, Locale, Params, Field, Place, Gender, Sibling}`
- Produces: providers registered under `schema.KindICD10`, `KindNDC`, `KindDrugName`, `KindISIN`, `KindLEI`, `KindCUSIP`, `KindCIDR`, `KindASN`, `KindMACVendor`, `KindTimezoneCoord`, `KindGeoJSONPoint`

- [x] **Step 1: Write the failing test**

```go
package providers_test

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
)

// ISIN check digits must validate under the Luhn-on-expanded-alphanumeric rule.
func TestISINChecksum(t *testing.T) {
	type Sec struct {
		ISIN string `synth:"isin"`
	}
	for _, s := range synth.Make[Sec](200, synth.WithSeed(7)) {
		if len(s.ISIN) != 12 {
			t.Fatalf("ISIN %q is not 12 characters", s.ISIN)
		}
		if !validISIN(s.ISIN) {
			t.Fatalf("ISIN %q fails its check digit", s.ISIN)
		}
	}
}

// ICD-10 codes must match the real format: letter, two digits, optional
// dot plus one to four alphanumerics.
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

// A CIDR block must parse and its address must be the network address.
func TestCIDRIsCanonical(t *testing.T) {
	type Net struct {
		Block string `synth:"cidr"`
	}
	for _, n := range synth.Make[Net](200, synth.WithSeed(9)) {
		p, err := netip.ParsePrefix(n.Block)
		if err != nil {
			t.Fatalf("CIDR %q does not parse: %v", n.Block, err)
		}
		if p.Masked() != p {
			t.Fatalf("CIDR %q is not a canonical network address", n.Block)
		}
	}
}

// A timezone must be geographically consistent with its coordinates.
func TestTimezoneMatchesCoordinates(t *testing.T) {
	type Loc struct {
		TZ  string  `synth:"timezone"`
		Lat float64 `synth:"latitude"`
		Lon float64 `synth:"longitude"`
	}
	for _, l := range synth.Make[Loc](200, synth.WithSeed(4)) {
		if !strings.Contains(l.TZ, "/") {
			t.Fatalf("timezone %q is not an IANA name", l.TZ)
		}
		if l.Lat < -90 || l.Lat > 90 || l.Lon < -180 || l.Lon > 180 {
			t.Fatalf("coordinates out of range: %v, %v", l.Lat, l.Lon)
		}
	}
}
```

Add at the top of the test file:

```go
var icd10Re = regexp.MustCompile(`^[A-TV-Z][0-9][0-9AB](\.[0-9A-TV-Z]{1,4})?$`)

// validISIN expands letters to two-digit numbers, then applies Luhn.
func validISIN(s string) bool {
	var digits []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			digits = append(digits, c-'0')
		case c >= 'A' && c <= 'Z':
			v := int(c-'A') + 10
			digits = append(digits, byte(v/10), byte(v%10))
		default:
			return false
		}
	}
	sum, double := 0, true
	for i := len(digits) - 2; i >= 0; i-- {
		d := int(digits[i])
		if double {
			if d *= 2; d > 9 {
				d -= 9
			}
		}
		double = !double
		sum += d
	}
	return (10-sum%10)%10 == int(digits[len(digits)-1])
}
```

- [x] **Step 2: Run the test and confirm it fails**

Run: `go test ./providers/ -run 'ISIN|ICD10|CIDR|Timezone' -v`
Expected: FAIL — the fields come back empty because the kinds are not registered.

- [x] **Step 3: Add the Kind constants**

In `schema/schema.go`, extend the `Kind` block following the existing style:

```go
	KindICD10
	KindNDC
	KindDrugName
	KindISIN
	KindLEI
	KindCUSIP
	KindCIDR
	KindASN
	KindMACVendor
	KindTimezoneCoord
	KindGeoJSONPoint
```

- [x] **Step 4: Implement the providers**

Create `providers/domains.go`. Datasets are curated real values; check digits are computed, never hardcoded.

```go
package providers

// icd10Codes holds real ICD-10-CM codes with their official descriptions.
// Curated from the CDC's published tabular list.
var icd10Codes = []string{
	"E11.9", "I10", "J45.909", "M54.5", "F41.1", "K21.9", "N39.0", "R51.9",
	// ... continue to at least 1000 real codes
}

// isinCountryPrefixes are the ISO 3166-1 alpha-2 codes used by real
// numbering agencies.
var isinCountryPrefixes = []string{"US", "GB", "DE", "FR", "JP", "CH", "NL", "CA"}

// isin builds a syntactically valid ISIN: country prefix, nine-character
// NSIN, then the computed check digit.
func isin(c Ctx) string {
	prefix := pick(c.Rand, isinCountryPrefixes)
	nsin := randAlnum(c.Rand, 9)
	body := prefix + nsin
	return body + string(rune('0'+isinCheckDigit(body)))
}

// isinCheckDigit expands letters to two digits, then applies Luhn.
func isinCheckDigit(body string) int {
	var digits []int
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if ch >= '0' && ch <= '9' {
			digits = append(digits, int(ch-'0'))
			continue
		}
		v := int(ch-'A') + 10
		digits = append(digits, v/10, v%10)
	}
	sum, double := 0, true
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
	return (10 - sum%10) % 10
}

// cidr returns a canonical private-range network block. The host bits are
// cleared so the address is always the network address.
func cidr(c Ctx) string {
	bits := 8 + c.Rand.IntN(17) // /8 .. /24
	base := netip.AddrFrom4([4]byte{
		10, byte(c.Rand.IntN(256)), byte(c.Rand.IntN(256)), byte(c.Rand.IntN(256)),
	})
	return netip.PrefixFrom(base, bits).Masked().String()
}
```

Implement `ndc`, `drugName`, `lei`, `cusip`, `asn`, `macVendor`, `timezoneCoord`
and `geoJSONPoint` in the same file, in the same style. `timezoneCoord` must draw
the timezone and the coordinates from a single shared table row so they agree;
store the chosen row on `Ctx.Sibling` the way `locale.Place` is shared.

- [x] **Step 5: Register the providers and name synonyms**

In `providers/providers.go`, add to the dispatch table:

```go
	schema.KindICD10:         func(c Ctx) any { return pick(c.Rand, icd10Codes) },
	schema.KindISIN:          func(c Ctx) any { return isin(c) },
	schema.KindCIDR:          func(c Ctx) any { return cidr(c) },
	// ... one line per new kind
```

In `register.go`, add field-name synonyms so untagged structs infer correctly:
`"icd10"`, `"diagnosiscode"` → `KindICD10`; `"isin"` → `KindISIN`;
`"cidr"`, `"subnet"` → `KindCIDR`; `"asn"` → `KindASN`.

- [x] **Step 6: Run the tests**

Run: `go test ./providers/ -run 'ISIN|ICD10|CIDR|Timezone' -v`
Expected: PASS.

Then run the full suite: `go test -race ./...`
Expected: PASS — in particular `TestParallelMatchesSerial` and the cardinality
tests, which will now cover the new kinds.

- [x] **Step 7: Commit**

```bash
git add schema/schema.go providers/domains.go providers/domains_test.go providers/providers.go register.go
git commit -m "feat: add healthcare, finance, network and geo identifier types"
```

---

### Task 3: Localize the hardcoded English catalog

Entries such as Superhero and Cocktail return English strings regardless of locale, so `WithLocale("uz_UZ")` silently yields English data. Make locale-sensitive catalog entries fall back explicitly rather than pretending.

**Files:**
- Create: `providers/catalog_locale.go`
- Create: `providers/catalog_locale_test.go`
- Modify: `providers/catalog2.go`, `providers/catalog3.go` (route through the lookup)

**Interfaces:**
- Produces: `func localized(c Ctx, key string, fallback []string) string` — returns a locale-specific dataset when one exists for `c.Locale.Code`, otherwise `fallback`.

- [x] **Step 1: Write the failing test**

```go
// Locale-aware catalog types must return locale-specific values where a
// dataset exists, and must not silently return English for a locale that
// claims full coverage.
func TestCatalogRespectsLocale(t *testing.T) {
	type Row struct {
		Food  string `synth:"food"`
		Color string `synth:"color"`
	}
	uz := synth.Make[Row](200, synth.WithSeed(1), synth.WithLocale("uz_UZ"))
	en := synth.Make[Row](200, synth.WithSeed(1), synth.WithLocale("en_US"))

	same := 0
	for i := range uz {
		if uz[i].Food == en[i].Food {
			same++
		}
	}
	if same > len(uz)/10 {
		t.Fatalf("uz_UZ food matched en_US in %d/%d rows — not localized", same, len(uz))
	}
}

// Types with no locale dataset must be reported by Warnings, not returned
// as silent English.
func TestUnlocalizedTypeWarns(t *testing.T) {
	type Row struct {
		Hero string `synth:"superhero"`
	}
	g := synth.New[Row](synth.WithLocale("uz_UZ"))
	g.Make(10)
	found := false
	for _, w := range g.Warnings() {
		if strings.Contains(w, "superhero") && strings.Contains(w, "uz_UZ") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a warning about superhero having no uz_UZ dataset, got %v", g.Warnings())
	}
}
```

- [x] **Step 2: Run it and confirm it fails**

Run: `go test ./providers/ -run Catalog -v`
Expected: FAIL — `uz_UZ` food equals `en_US` food on every row, and no warning is emitted.

- [x] **Step 3: Implement the lookup and the warning**

Create `providers/catalog_locale.go`:

```go
package providers

// localeCatalog holds per-locale datasets for catalog types whose values are
// culturally specific. Missing entries are a known gap, not a bug: callers
// are warned rather than being handed English data unannounced.
var localeCatalog = map[string]map[string][]string{
	"uz_UZ": {
		"food":  {"osh", "somsa", "manti", "lag'mon", "shashlik", /* ...1000 */},
		"color": {"qizil", "ko'k", "yashil", "sariq", "oq", "qora", /* ... */},
	},
	"ru_RU": {
		"food":  {"борщ", "пельмени", "блины", "окрошка", /* ... */},
		"color": {"красный", "синий", "зелёный", "жёлтый", /* ... */},
	},
	// ... de, fr, es, it, ja, tr, pt_BR, nl, pl, ko, uk
}

// localized returns a value from the locale's dataset for key. When the
// locale has no dataset it records a warning and falls back to English, so
// the gap is visible instead of silent.
func localized(c Ctx, key string, fallback []string) string {
	if byKey, ok := localeCatalog[c.Locale.Code]; ok {
		if vals, ok := byKey[key]; ok && len(vals) > 0 {
			return pick(c.Rand, vals)
		}
	}
	if c.Locale.Code != "en_US" {
		c.Warnf("%s has no %s dataset; using en_US values", key, c.Locale.Code)
	}
	return pick(c.Rand, fallback)
}
```

Add a `Warnf(format string, args ...any)` method to `Ctx` that appends to the
engine's existing warnings slice — the same one `synth.Warnings()` returns.

Route the culturally specific catalog entries in `catalog2.go` and `catalog3.go`
through `localized`, e.g.:

```go
	schema.KindFood: func(c Ctx) any { return localized(c, "food", enFoods) },
```

- [x] **Step 4: Run the tests**

Run: `go test ./providers/ -run Catalog -v`
Expected: PASS.

- [x] **Step 5: Document the coverage honestly**

In `README.md`, add a row to the locale table stating which locales have
localized catalog datasets versus name/address data only. Do not claim
coverage the code does not have.

- [x] **Step 6: Commit**

```bash
git add providers/catalog_locale.go providers/catalog_locale_test.go providers/catalog2.go providers/catalog3.go README.md
git commit -m "feat: localize culturally specific catalog types and warn on gaps"
```

---

### Task 4: Complete the remaining locale name banks

14 of 52 locales have gendered 1k-combination name banks. The rest fall back to a thin list, so `uz_UZ`-grade quality is not what most locales deliver.

**Files:**
- Modify: `locale/locales_ext.go`
- Modify: `locale/locale_test.go`

- [x] **Step 1: Write the failing test**

```go
// Every locale must reach at least 1000 distinct full-name combinations with
// gender-consistent first and last names.
func TestAllLocalesReachThousandNames(t *testing.T) {
	for _, code := range locale.Codes() {
		l := locale.Get(code)
		for _, g := range []string{"male", "female"} {
			first, last := l.FirstNamesFor(g), l.LastNamesFor(g)
			if n := len(first) * len(last); n < 1000 {
				t.Errorf("%s %s: only %d name combinations (%d x %d)",
					code, g, n, len(first), len(last))
			}
		}
	}
}
```

- [x] **Step 2: Run it and record the gap**

Run: `go test ./locale/ -run Thousand -v`
Expected: FAIL, listing each locale still short. That list is this task's worklist.

- [x] **Step 3: Fill each locale's banks**

For every failing locale, extend its `seed` entry in `locales_ext.go` to at
least 24 male first names, 24 female first names, and 24 surnames per gender —
24 × 42 exceeds 1000. Names must be real and orthographically correct for the
language. Where a language inflects surnames by gender (Polish, Ukrainian,
Czech, Slovak, Latvian, Lithuanian), supply both forms:

```go
	{code: "cs_CZ",
		maleFirst:  []string{"Jakub", "Jan", "Tomáš", /* 24 */},
		femaleFirst: []string{"Eliška", "Tereza", "Anna", /* 24 */},
		maleLast:   []string{"Novák", "Svoboda", "Novotný", /* 42 */},
		femaleLast: []string{"Nováková", "Svobodová", "Novotná", /* 42 */},
	},
```

Work through the list in batches, re-running the test after each batch so the
remaining worklist stays visible.

- [x] **Step 4: Run the tests**

Run: `go test ./locale/ -run Thousand -v`
Expected: PASS with no locale reported.

Run: `go test -race ./...`
Expected: PASS — the gender-coherence and cardinality suites must still hold.

- [x] **Step 5: Commit**

```bash
git add locale/locales_ext.go locale/locale_test.go
git commit -m "feat: complete gendered 1k name banks for all locales"
```

---

### Task 5: Expose every capability through the CLI

`cmd/synth` supports only `gen` with `csv|jsonl|sql`. Profiling, masking, CDC, Protobuf and Parquet are library-only.

**Files:**
- Modify: `cmd/synth/main.go`
- Create: `cmd/synth/main_test.go`

**Interfaces:**
- Consumes: `synth.YAMLFile`, `synth.Profile`, `synth.MaskFile`, `synth.CDC`, `synth.ProtoFile`, `synth.DDLFile`
- Produces: subcommands `gen`, `profile`, `mask`, `cdc`

- [x] **Step 1: Write the failing test**

```go
package main_test

// Each subcommand must run end to end and write its output file.
func TestSubcommands(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "synth")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.yaml")
	os.WriteFile(spec, []byte("name: users\ncount: 50\nfields:\n  id: {kind: uuid}\n  email: {kind: email}\n"), 0o644)

	csvOut := filepath.Join(dir, "users.csv")
	run(t, bin, "gen", "-s", spec, "-o", csvOut, "-f", "csv")
	mustExist(t, csvOut)

	// profile reads the exported file and emits a schema — no DB involved.
	profOut := filepath.Join(dir, "inferred.yaml")
	run(t, bin, "profile", "-i", csvOut, "-o", profOut)
	mustExist(t, profOut)

	maskOut := filepath.Join(dir, "masked.csv")
	run(t, bin, "mask", "-i", csvOut, "-o", maskOut, "--key", "test-key")
	mustExist(t, maskOut)
	if bytes.Equal(readFile(t, csvOut), readFile(t, maskOut)) {
		t.Fatal("mask produced an identical file")
	}

	cdcOut := filepath.Join(dir, "events.jsonl")
	run(t, bin, "cdc", "-s", spec, "-o", cdcOut, "-n", "20")
	mustExist(t, cdcOut)
}

// An unknown subcommand must exit non-zero with usage on stderr.
func TestUnknownSubcommand(t *testing.T) {
	// ... asserts exit code != 0 and stderr mentions the valid subcommands
}
```

- [x] **Step 2: Run it and confirm it fails**

Run: `go test ./cmd/synth/ -v`
Expected: FAIL — `profile`, `mask` and `cdc` are not recognized.

- [x] **Step 3: Restructure main into subcommands**

In `cmd/synth/main.go`, dispatch on `os.Args[1]`:

```go
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "gen":
		err = runGen(os.Args[2:])
	case "profile":
		err = runProfile(os.Args[2:])
	case "mask":
		err = runMask(os.Args[2:])
	case "cdc":
		err = runCDC(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "synth:", err)
		os.Exit(1)
	}
}
```

Move the existing flag parsing into `runGen`, keeping every current flag
(`-s -o -f -l -n --seed --chaos`) working unchanged. Accept `-s` as either a
YAML spec, a `.sql` DDL file, an OpenAPI document, or a `.proto` file, chosen
by extension — the frontends already exist.

`runProfile` reads an exported CSV/JSONL file and writes an inferred YAML spec.
`runMask` reads a file and writes a masked copy, requiring `--key`. `runCDC`
writes a Debezium-shaped JSONL event stream. None of them opens a connection.

- [x] **Step 4: Add Parquet to the format list — as an error with a pointer**

Parquet lives in a submodule, so the core CLI cannot link it. When
`-f parquet` is passed, fail with a clear message rather than silently
producing something else:

```go
	case "parquet":
		return fmt.Errorf("parquet output lives in the optional submodule: " +
			"go get github.com/bakhod1r/synth/sink/parquet")
```

- [x] **Step 5: Run the tests**

Run: `go test ./cmd/synth/ -v`
Expected: PASS.

Run: `go test -race ./... && go build ./cmd/synth`
Expected: PASS.

- [x] **Step 6: Document the CLI**

Add a CLI section to `README.md` showing one worked example per subcommand,
with real flags copied from the implementation.

- [x] **Step 7: Commit**

```bash
git add cmd/synth/main.go cmd/synth/main_test.go README.md
git commit -m "feat: expose profile, mask and cdc as CLI subcommands"
```

---

### Task 6: Constraint mining — learn invariants from a sample

Profiling already infers per-column types and distributions. It does not learn
relationships *between* columns, so a generated table can hold `total = 100`
while its line items sum to 340. This task mines invariants from a real sample
and enforces them at generation time. It is the strongest evidence for the
"not a faker, a dataset" claim.

**Files:**
- Create: `constraint/constraint.go` — the `Constraint` interface and the IR
- Create: `constraint/mine.go` — detection over a sample
- Create: `constraint/enforce.go` — repair pass applied after generation
- Create: `constraint/constraint_test.go`
- Modify: `profileapi.go` (emit mined constraints into the inferred spec)
- Modify: `yamlfe/yamlfe.go` (parse a `constraints:` block)
- Modify: `gen/gen.go` (run the enforce pass before a record is returned)

**Interfaces:**
- Consumes: `[]map[string]any` (the same shape `profile` already reads from a
  CSV/JSONL file — no database)
- Produces:
  ```go
  type Kind string
  const (
      Ordering    Kind = "ordering"    // A <= B
      SumEquals   Kind = "sum"         // sum(parts) == whole
      Implication Kind = "implication" // A == v  =>  B is non-null
      Range       Kind = "range"       // lo <= A <= hi
  )
  type Constraint struct {
      Kind    Kind
      Left    string   // column
      Right   string   // column, for Ordering
      Parts   []string // columns, for SumEquals
      Whole   string
      When    string   // column, for Implication
      Equals  string   // value that triggers the implication
      Then    string   // column that must then be non-null
      Lo, Hi  float64
      Support int      // rows the invariant held over
    }
  func Mine(rows []map[string]any, minSupport float64) []Constraint
  func Enforce(cs []Constraint, rec map[string]any)
  func (c Constraint) Holds(rec map[string]any) bool
  ```

- [x] **Step 1: Write the failing test**

```go
package constraint_test

// Mining must find an ordering invariant that holds across the whole sample
// and must NOT report one that holds only by chance.
func TestMineOrdering(t *testing.T) {
    var rows []map[string]any
    for i := 0; i < 200; i++ {
        rows = append(rows, map[string]any{
            "created_at": float64(i),
            "updated_at": float64(i + 5), // always later
            "noise":      float64(200 - i),
        })
    }
    cs := constraint.Mine(rows, 0.99)

    var found bool
    for _, c := range cs {
        if c.Kind == constraint.Ordering && c.Left == "created_at" && c.Right == "updated_at" {
            found = true
        }
        if c.Kind == constraint.Ordering && c.Left == "noise" {
            t.Fatalf("mined a spurious ordering on noise: %+v", c)
        }
    }
    if !found {
        t.Fatalf("did not mine created_at <= updated_at; got %+v", cs)
    }
}

// A sum invariant must be found even with floating point representation error.
func TestMineSum(t *testing.T) {
    var rows []map[string]any
    for i := 0; i < 100; i++ {
        sub, tax := float64(i)*1.1, float64(i)*0.07
        rows = append(rows, map[string]any{
            "subtotal": sub, "tax": tax, "total": sub + tax,
        })
    }
    cs := constraint.Mine(rows, 0.99)
    for _, c := range cs {
        if c.Kind == constraint.SumEquals && c.Whole == "total" && len(c.Parts) == 2 {
            return
        }
    }
    t.Fatalf("did not mine subtotal + tax = total; got %+v", cs)
}

// An implication must be mined only when the trigger value actually occurs
// often enough to be evidence.
func TestMineImplication(t *testing.T) {
    var rows []map[string]any
    for i := 0; i < 300; i++ {
        r := map[string]any{"status": "paid", "refund_at": nil}
        if i%3 == 0 {
            r = map[string]any{"status": "refunded", "refund_at": "2026-01-01T00:00:00Z"}
        }
        rows = append(rows, r)
    }
    cs := constraint.Mine(rows, 0.99)
    for _, c := range cs {
        if c.Kind == constraint.Implication && c.Equals == "refunded" && c.Then == "refund_at" {
            return
        }
    }
    t.Fatalf("did not mine status=refunded => refund_at non-null; got %+v", cs)
}

// Generated records must satisfy every mined constraint after enforcement.
func TestEnforceRepairsRecords(t *testing.T) {
    cs := []constraint.Constraint{
        {Kind: constraint.Ordering, Left: "created_at", Right: "updated_at"},
        {Kind: constraint.SumEquals, Parts: []string{"subtotal", "tax"}, Whole: "total"},
    }
    rec := map[string]any{
        "created_at": 100.0, "updated_at": 20.0, // violates ordering
        "subtotal": 10.0, "tax": 1.0, "total": 999.0, // violates sum
    }
    constraint.Enforce(cs, rec)
    for _, c := range cs {
        if !c.Holds(rec) {
            t.Fatalf("constraint %+v still violated after Enforce: %+v", c, rec)
        }
    }
}
```

- [x] **Step 2: Run and confirm it fails**

Run: `go test ./constraint/ -v`
Expected: FAIL — the package does not exist.

- [x] **Step 3: Implement mining**

Mining is candidate generation plus falsification. For each ordered pair of
numeric columns, assume `A <= B` and scan; drop the candidate on the first
counterexample beyond the `minSupport` budget. For sums, test every pair and
triple of numeric columns against every other numeric column using a relative
epsilon (`1e-9 * max(|whole|, 1)`) so float noise does not hide a real
invariant. For implications, group by each low-cardinality string column's
values and check whether another column is non-null in every row of the group.

Require a minimum group size (at least 20 rows, and at least 5% of the sample)
before reporting an implication — otherwise a single row invents a rule. Record
`Support` on every constraint so a human can judge it.

- [x] **Step 4: Implement enforcement**

`Enforce` repairs rather than rejects, so generation stays O(1) per record:

- `Ordering`: if `Left > Right`, swap the two values.
- `SumEquals`: overwrite `Whole` with the sum of `Parts` — the parts are the
  generated facts, the total is derived.
- `Implication`: if the trigger matches and `Then` is null, fill it from the
  field's own provider; if it does not match, leave the record alone.
- `Range`: clamp.

Apply constraints in declaration order and document that order matters, since
a later repair can undo an earlier one.

- [x] **Step 5: Wire into profiling, YAML and the engine**

`synth profile` appends a `constraints:` block to the inferred spec listing
what it mined, with the support count as a comment on each line. `yamlfe`
parses that block back. `gen.Engine` runs `Enforce` on each record just
before returning it.

- [x] **Step 6: Round-trip test**

```go
// Mining a real sample, generating from the inferred spec, and re-mining the
// generated data must yield the same invariants.
func TestConstraintRoundTrip(t *testing.T) {
    // profile sample.csv -> spec with constraints -> generate -> re-mine
    // assert every originally mined constraint still holds on generated rows
}
```

Run: `go test -race ./...`
Expected: PASS.

- [x] **Step 7: Commit**

```bash
git add constraint/ profileapi.go yamlfe/yamlfe.go gen/gen.go
git commit -m "feat: mine cross-column invariants and enforce them during generation"
```

---

### Task 7: `synth verify` — the inverse product

The same knowledge that generates a coherent dataset can audit one. This is a
second product from one codebase, and it reads files only: no database.

**Files:**
- Create: `verify/verify.go` — the `Report` type and the check runner
- Create: `verify/checks.go` — individual checks
- Create: `verify/verify_test.go`
- Modify: `cmd/synth/main.go` (add the `verify` subcommand)

**Interfaces:**
- Consumes: one or more CSV/JSONL files plus an optional spec for FK relations
- Produces:
  ```go
  type Severity string
  const (SevError Severity = "error"; SevWarn Severity = "warn")
  type Finding struct {
      Check    string   // "luhn", "fk", "temporal", "distribution", "constraint"
      Severity Severity
      Column   string
      Row      int      // -1 for dataset-wide findings
      Detail   string
      Sample   []string // up to 3 offending values
  }
  type Report struct {
      Rows     int
      Findings []Finding
  }
  func Run(rows []map[string]any, opts Options) Report
  func (r Report) Text(w io.Writer) error
  func (r Report) JSON(w io.Writer) error
  ```

- [x] **Step 1: Write the failing test**

```go
// A card column with a broken check digit must be reported, and a valid one
// must not be.
func TestVerifyDetectsLuhnFailure(t *testing.T) {
    rows := []map[string]any{
        {"card": "4111111111111111"}, // valid
        {"card": "4111111111111112"}, // invalid check digit
    }
    rep := verify.Run(rows, verify.Options{})
    var got *verify.Finding
    for i := range rep.Findings {
        if rep.Findings[i].Check == "luhn" {
            got = &rep.Findings[i]
        }
    }
    if got == nil {
        t.Fatal("did not flag the invalid card")
    }
    if got.Row != 1 {
        t.Fatalf("flagged row %d, want row 1", got.Row)
    }
}

// A child row pointing at a missing parent key must be reported.
func TestVerifyDetectsBrokenForeignKey(t *testing.T) {
    parents := []map[string]any{{"id": "a"}, {"id": "b"}}
    children := []map[string]any{{"user_id": "a"}, {"user_id": "ghost"}}
    rep := verify.Run(children, verify.Options{
        Refs: []verify.Ref{{Column: "user_id", Parent: parents, ParentKey: "id"}},
    })
    if len(rep.Findings) == 0 {
        t.Fatal("did not flag the dangling user_id")
    }
    if !strings.Contains(rep.Findings[0].Detail, "ghost") {
        t.Fatalf("finding does not name the bad value: %+v", rep.Findings[0])
    }
}

// An updated_at before created_at is a temporal anomaly.
func TestVerifyDetectsTemporalAnomaly(t *testing.T) {
    rows := []map[string]any{
        {"created_at": "2026-01-02T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"},
    }
    rep := verify.Run(rows, verify.Options{})
    for _, f := range rep.Findings {
        if f.Check == "temporal" {
            return
        }
    }
    t.Fatalf("did not flag updated_at before created_at; got %+v", rep.Findings)
}

// A column that is 99% one value is a distribution warning, not an error.
func TestVerifyFlagsDegenerateDistribution(t *testing.T) {
    var rows []map[string]any
    for i := 0; i < 1000; i++ {
        v := "same"
        if i == 0 {
            v = "different"
        }
        rows = append(rows, map[string]any{"status": v})
    }
    rep := verify.Run(rows, verify.Options{})
    for _, f := range rep.Findings {
        if f.Check == "distribution" && f.Severity == verify.SevWarn {
            return
        }
    }
    t.Fatal("did not warn about a degenerate status column")
}

// A clean dataset must produce an empty report — no false positives.
func TestVerifyCleanDatasetIsSilent(t *testing.T) {
    type Row struct {
        ID   string `synth:"uuid"`
        Card string `synth:"card"`
        Mail string `synth:"email"`
    }
    recs := synth.Make[Row](500, synth.WithSeed(1))
    // convert to []map[string]any, then:
    rep := verify.Run(rows, verify.Options{})
    if len(rep.Findings) != 0 {
        t.Fatalf("clean Synth output produced findings: %+v", rep.Findings)
    }
}
```

- [x] **Step 2: Run and confirm it fails**

Run: `go test ./verify/ -v`
Expected: FAIL — the package does not exist.

- [x] **Step 3: Implement the checks**

Detect each column's semantic kind by reusing `infer` on the column name, then
run the matching validator: `luhn` for cards and IMEI, `mod97` for IBAN, EAN-13
and UPC check digits, RFC-shaped email and URL parsing, `netip` for addresses.
Temporal checks compare any pair of parseable timestamp columns whose names
suggest ordering (`created`/`updated`, `start`/`end`, `opened`/`closed`).
Distribution checks flag a column where one value exceeds 95% of non-null rows,
or where a numeric column has zero variance.

The clean-dataset test is the important one: a check that fires on Synth's own
correct output is a false positive and must be fixed, not tolerated.

- [x] **Step 4: Add the CLI subcommand**

```bash
synth verify -i orders.csv --ref user_id=users.csv:id --format text
```

Exit code 0 when there are no errors, 1 when any finding has `SevError`, so it
drops into CI without a wrapper. Warnings alone do not fail the run.

- [x] **Step 5: Run the tests**

Run: `go test -race ./... && go build ./cmd/synth`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add verify/ cmd/synth/main.go
git commit -m "feat: add synth verify, an auditor for existing datasets"
```

---

### Task 8: Time-travel snapshots

One seed already determines an entire dataset. Make time an explicit axis of
that determinism: ask for the world as of any instant, and the CDC event log
between two instants must reconstruct the difference exactly. That equivalence
is what makes the feature trustworthy for migration and incremental-ETL tests.

**Files:**
- Create: `snapshot/snapshot.go`
- Create: `snapshot/snapshot_test.go`
- Modify: `cdc/cdc.go` (give each event a wall-clock time drawn from the same
  timeline the snapshot uses)
- Modify: `cmd/synth/main.go` (`synth snapshot --at`, `--from`, `--to`)

**Interfaces:**
- Produces:
  ```go
  type Config struct {
      Table    string
      Rows     int       // steady-state row count
      Start    time.Time // when the table came into existence
      Churn    float64   // updates+deletes per row per year
  }
  type Timeline[T any] struct{ ... }
  func New[T any](cfg Config, opts ...synth.Option) *Timeline[T]
  func (t *Timeline[T]) At(when time.Time) []map[string]any
  func (t *Timeline[T]) Between(from, to time.Time) []cdc.Event
  ```

- [x] **Step 1: Write the failing test — the equivalence that defines the feature**

```go
// Applying the CDC events between two instants to the earlier snapshot must
// produce the later snapshot, exactly. This is the whole contract.
func TestEventsReconstructLaterSnapshot(t *testing.T) {
    t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
    t1 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

    tl := snapshot.New[Order](snapshot.Config{
        Table: "orders", Rows: 500, Start: t0.AddDate(-1, 0, 0), Churn: 2.0,
    }, synth.WithSeed(42))

    before := tl.At(t0)
    after := tl.At(t1)
    events := tl.Between(t0, t1)

    got := apply(before, events) // insert/update/delete by primary key
    if !equalByKey(got, after, "id") {
        t.Fatalf("replaying %d events gave %d rows, want %d — the log and the "+
            "snapshots disagree", len(events), len(got), len(after))
    }
}

// The same instant and seed must always give byte-identical output.
func TestSnapshotIsDeterministic(t *testing.T) {
    at := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
    a := snapshot.New[Order](cfg, synth.WithSeed(7)).At(at)
    b := snapshot.New[Order](cfg, synth.WithSeed(7)).At(at)
    if !reflect.DeepEqual(a, b) {
        t.Fatal("two snapshots at the same instant differ")
    }
}

// A snapshot before the table existed must be empty, not an error.
func TestSnapshotBeforeStartIsEmpty(t *testing.T) {
    tl := snapshot.New[Order](snapshot.Config{
        Table: "orders", Rows: 100,
        Start: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
    }, synth.WithSeed(1))
    if got := tl.At(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); len(got) != 0 {
        t.Fatalf("got %d rows before the table existed", len(got))
    }
}

// Rows created after the requested instant must not appear in it.
func TestSnapshotExcludesFutureRows(t *testing.T) {
    // assert every row's created_at <= the requested instant
}
```

- [x] **Step 2: Run and confirm it fails**

Run: `go test ./snapshot/ -v`
Expected: FAIL — the package does not exist.

- [x] **Step 3: Implement the timeline**

Do not simulate forward — that would make `At` cost O(history). Instead give
every row a deterministic life derived from its index: a birth instant from
`Fork(i)`, then a sequence of mutation instants from the same forked stream at
the configured churn rate, and possibly a death instant. `At(when)` walks each
row's own event list — an O(events per row) operation independent of how far
`when` is from `Start` — and returns the row's state as of that moment, or
nothing if it was not yet born or already dead. `Between` emits the same events
in timestamp order.

Because both `At` and `Between` read the identical per-row event list, the
equivalence in Step 1 holds by construction rather than by luck.

- [x] **Step 4: Add the CLI subcommand**

```bash
synth snapshot -s spec.yaml --at 2026-01-01 -o jan.csv
synth snapshot -s spec.yaml --from 2026-01-01 --to 2026-07-01 -o changes.jsonl
```

- [x] **Step 5: Run the tests**

Run: `go test -race ./...`
Expected: PASS — especially the reconstruction test.

- [x] **Step 6: Commit**

```bash
git add snapshot/ cdc/cdc.go cmd/synth/main.go
git commit -m "feat: add time-travel snapshots with a matching CDC event log"
```

---

### Task 9: Browser workbench (`synth ui`)

Mockaroo's advantage is that you can see the data while you shape the schema.
Match that: a local page listing field types, a schema builder, and a live
preview that regenerates as you edit.

**Security boundary (non-negotiable):** the server binds `127.0.0.1` only,
never `0.0.0.0`. No telemetry, no CDN, no outbound request of any kind — the
page is served from `embed.FS` with inline CSS and JS. It generates data and
writes files on the user's own machine; nothing leaves it. This respects the
project's no-network rule: the browser connects in, Synth never connects out.

**Files:**
- Create: `ui/ui.go` — handlers
- Create: `ui/static/index.html`, `ui/static/app.js`, `ui/static/app.css`
- Create: `ui/ui_test.go`
- Modify: `cmd/synth/main.go` (`synth ui --port 8080`)

**Interfaces:**
- Produces:
  ```go
  func Handler() http.Handler // all routes, for testing without a listener
  func Serve(addr string) error
  ```
  Routes: `GET /` (page), `GET /api/types` (catalog with locale coverage),
  `GET /api/locales`, `POST /api/preview` (spec → 10 sample rows),
  `POST /api/generate` (spec → file download).

- [x] **Step 1: Write the failing test**

```go
// The type catalog must be served from the real registry, not a copy that
// can drift.
func TestTypesEndpointListsEveryKind(t *testing.T) {
    rec := httptest.NewRecorder()
    ui.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/types", nil))
    if rec.Code != 200 {
        t.Fatalf("status %d", rec.Code)
    }
    var got []struct {
        Kind string `json:"kind"`
    }
    json.NewDecoder(rec.Body).Decode(&got)
    if len(got) < 150 {
        t.Fatalf("only %d types exposed; the registry has more", len(got))
    }
}

// Preview must return real generated rows for a posted spec.
func TestPreviewReturnsRows(t *testing.T) {
    body := `{"fields":{"name":{"kind":"name"},"email":{"kind":"email"}},"locale":"uz_UZ"}`
    rec := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/api/preview", strings.NewReader(body))
    ui.Handler().ServeHTTP(rec, req)
    if rec.Code != 200 {
        t.Fatalf("status %d: %s", rec.Code, rec.Body)
    }
    var rows []map[string]any
    json.NewDecoder(rec.Body).Decode(&rows)
    if len(rows) != 10 {
        t.Fatalf("got %d preview rows, want 10", len(rows))
    }
    if !strings.Contains(rows[0]["email"].(string), "@") {
        t.Fatalf("preview email is not real: %v", rows[0])
    }
}

// A malformed spec must return 400 with a readable message, not a panic.
func TestPreviewRejectsBadSpec(t *testing.T) {
    rec := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/api/preview", strings.NewReader(`{"fields":{"x":{"kind":"nope"}}}`))
    ui.Handler().ServeHTTP(rec, req)
    if rec.Code != 400 {
        t.Fatalf("status %d, want 400", rec.Code)
    }
    if !strings.Contains(rec.Body.String(), "nope") {
        t.Fatalf("error does not name the bad kind: %s", rec.Body)
    }
}

// Preview must be capped so a huge count cannot hang the browser.
func TestPreviewCountIsCapped(t *testing.T) {
    // post count: 1000000, assert at most 100 rows come back
}

// The page must not reference any external origin.
func TestPageHasNoExternalResources(t *testing.T) {
    rec := httptest.NewRecorder()
    ui.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
    for _, bad := range []string{"http://", "https://", "//cdn", "fonts.googleapis"} {
        if strings.Contains(rec.Body.String(), bad) {
            t.Fatalf("page references an external origin: %q", bad)
        }
    }
}
```

- [x] **Step 2: Run and confirm it fails**

Run: `go test ./ui/ -v`
Expected: FAIL — the package does not exist.

- [x] **Step 3: Implement the handlers**

`/api/types` iterates the provider registry so it can never drift from what
the engine actually supports, and reports each type's locale coverage using
the `localeCatalog` table from Task 3 — the UI shows honestly which types are
English-only. `/api/preview` unmarshals the posted spec into the same YAML IR
`yamlfe` produces, caps the count at 100, and returns generated rows.
`/api/generate` streams a file with a `Content-Disposition` header, reusing the
existing encoders, and offers csv/jsonl/sql.

Return 400 with the offending value named for any unknown kind — the whole
point of the UI is to make mistakes visible.

- [x] **Step 4: Build the page**

Three panes: type palette on the left (searchable, grouped by category), the
schema builder in the middle (add/remove/reorder fields, set params, locale,
row count, seed), and a live preview table on the right that re-requests on
change with a short debounce. Plain JavaScript, no framework, no build step —
`go:embed` ships it inside the binary so `synth ui` is one command with no
install.

Show the seed prominently and make it editable: reproducibility is the feature
Mockaroo does not have, so the UI should teach it rather than hide it.

- [x] **Step 5: Verify the binding is localhost-only**

`Serve` must reject a non-loopback address:

```go
func Serve(addr string) error {
    host, _, err := net.SplitHostPort(addr)
    if err != nil {
        return err
    }
    if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
        return fmt.Errorf("ui: refusing to bind %q — the workbench is "+
            "loopback-only by design", addr)
    }
    ...
}
```

Add a test asserting `Serve("0.0.0.0:8080")` returns an error.

- [x] **Step 6: Run the tests**

Run: `go test -race ./... && go build ./cmd/synth`
Expected: PASS.

- [x] **Step 7: Document it**

Add a README section with a screenshot-free description, the exact command,
and an explicit statement that the server is loopback-only and sends nothing
anywhere.

- [x] **Step 8: Commit**

```bash
git add ui/ cmd/synth/main.go README.md
git commit -m "feat: add a local browser workbench for schema design and preview"
```
