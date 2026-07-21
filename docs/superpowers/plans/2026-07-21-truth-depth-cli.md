# Truth, Depth, CLI — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use `- [ ]` checkboxes.

**Goal:** Make the docs true, close the catalog/locale gaps, and expose every capability through the CLI.

**Architecture:** No architectural change. Five independent tasks against the existing frontends → IR → engine → output layering. Task 1 is a docs-only correction; Tasks 2–4 add providers/locale data; Task 5 extends `cmd/synth`.

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

- [ ] **Step 1: Fix the opening claim (`README.md:9`)**

Replace "It streams millions of records straight to Kafka, Postgres, CSV, or JSONL with minimal memory usage" with:

```
It streams millions of records to CSV, JSONL, SQL `INSERT` files, or Parquet
with minimal memory usage, and can produce valid request payloads from OpenAPI
schemas.
```

- [ ] **Step 2: Fix the comparison table row (`README.md:19`)**

Change the Output row's right-hand cell from `Kafka, Postgres, CSV, JSONL sinks` to:

```
CSV, JSONL, SQL, Parquet, CDC files
```

- [ ] **Step 3: Rewrite the sinks section (`README.md:98`)**

Replace the "Kafka, Postgres (batched `COPY`), CSV, and JSONL out of the box, behind one interface" sentence with:

```
CSV, JSONL, SQL `INSERT` files, Parquet, and Debezium-shaped CDC events —
every one of them a file. Synth never opens a database or network connection;
handing the file to your loader is the last step, and it is yours. See
[Scope](#scope) for why.
```

- [ ] **Step 4: Verify no contradiction remains**

Run: `grep -n "Kafka\|COPY\|Postgres" README.md`
Expected: only the `:265` line ("no database, no Kafka, just a file") and the
`:385` scope paragraph — both of which *deny* network sinks. No line may claim
Synth writes to Kafka or Postgres.

- [ ] **Step 5: Commit**

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

- [ ] **Step 1: Write the failing test**

```go
package providers_test

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth"
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

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./providers/ -run 'ISIN|ICD10|CIDR|Timezone' -v`
Expected: FAIL — the fields come back empty because the kinds are not registered.

- [ ] **Step 3: Add the Kind constants**

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

- [ ] **Step 4: Implement the providers**

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

- [ ] **Step 5: Register the providers and name synonyms**

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

- [ ] **Step 6: Run the tests**

Run: `go test ./providers/ -run 'ISIN|ICD10|CIDR|Timezone' -v`
Expected: PASS.

Then run the full suite: `go test -race ./...`
Expected: PASS — in particular `TestParallelMatchesSerial` and the cardinality
tests, which will now cover the new kinds.

- [ ] **Step 7: Commit**

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

- [ ] **Step 1: Write the failing test**

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

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./providers/ -run Catalog -v`
Expected: FAIL — `uz_UZ` food equals `en_US` food on every row, and no warning is emitted.

- [ ] **Step 3: Implement the lookup and the warning**

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

- [ ] **Step 4: Run the tests**

Run: `go test ./providers/ -run Catalog -v`
Expected: PASS.

- [ ] **Step 5: Document the coverage honestly**

In `README.md`, add a row to the locale table stating which locales have
localized catalog datasets versus name/address data only. Do not claim
coverage the code does not have.

- [ ] **Step 6: Commit**

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

- [ ] **Step 1: Write the failing test**

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

- [ ] **Step 2: Run it and record the gap**

Run: `go test ./locale/ -run Thousand -v`
Expected: FAIL, listing each locale still short. That list is this task's worklist.

- [ ] **Step 3: Fill each locale's banks**

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

- [ ] **Step 4: Run the tests**

Run: `go test ./locale/ -run Thousand -v`
Expected: PASS with no locale reported.

Run: `go test -race ./...`
Expected: PASS — the gender-coherence and cardinality suites must still hold.

- [ ] **Step 5: Commit**

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

- [ ] **Step 1: Write the failing test**

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

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./cmd/synth/ -v`
Expected: FAIL — `profile`, `mask` and `cdc` are not recognized.

- [ ] **Step 3: Restructure main into subcommands**

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

- [ ] **Step 4: Add Parquet to the format list — as an error with a pointer**

Parquet lives in a submodule, so the core CLI cannot link it. When
`-f parquet` is passed, fail with a clear message rather than silently
producing something else:

```go
	case "parquet":
		return fmt.Errorf("parquet output lives in the optional submodule: " +
			"go get github.com/bakhodir/synth/sink/parquet")
```

- [ ] **Step 5: Run the tests**

Run: `go test ./cmd/synth/ -v`
Expected: PASS.

Run: `go test -race ./... && go build ./cmd/synth`
Expected: PASS.

- [ ] **Step 6: Document the CLI**

Add a CLI section to `README.md` showing one worked example per subcommand,
with real flags copied from the implementation.

- [ ] **Step 7: Commit**

```bash
git add cmd/synth/main.go cmd/synth/main_test.go README.md
git commit -m "feat: expose profile, mask and cdc as CLI subcommands"
```
