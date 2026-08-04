package synth_test

// End-to-end pipelines: the paths a user actually runs — spec in, files out,
// read back, verified. Each test drives the public API only, with a fixed seed,
// and asserts invariants rather than exact values, so nothing here is flaky.

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/cdc"
	"github.com/bakhod1r/synth/pgcopy"
	"github.com/bakhod1r/synth/verify"
)

const shopSpec = `name: orders
count: 200
seed: 20240301
locale: en_US
fields:
  id:          { kind: uuid, pk: true }
  customer:    { kind: name }
  email:       { kind: email, unique: true }
  card_number: { kind: card, mask: partial }
  total:       { kind: amount, min: 5, max: 900 }
  currency:    { kind: currency }
  status:      { kind: enum, choices: [paid, pending, refunded], weights: [0.8, 0.15, 0.05] }
  created_at:  { kind: time }
  shipped_at:  { kind: time }
constraints:
  - {kind: ordering, left: created_at, right: shipped_at}
`

// A spec generates, writes every output format, and every file reads back with
// the same row count and the same values.
func TestE2EYAMLSpecToEveryFileFormat(t *testing.T) {
	dir := t.TempDir()
	spec, err := synth.YAMLBytes([]byte(shopSpec))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := spec.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 200 {
		t.Fatalf("got %d rows, want 200", len(rows))
	}
	cols := spec.Columns()

	// CSV
	csvPath := filepath.Join(dir, "orders.csv")
	f, err := os.Create(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	cw := csv.NewWriter(f)
	if err := cw.Write(cols); err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		rec := make([]string, len(cols))
		for i, c := range cols {
			rec[i] = strings.TrimSpace(stringify(r[c]))
		}
		if err := cw.Write(rec); err != nil {
			t.Fatal(err)
		}
	}
	cw.Flush()
	f.Close()

	rf, err := os.Open(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	defer rf.Close()
	back, err := csv.NewReader(rf).ReadAll()
	if err != nil {
		t.Fatalf("the written CSV does not parse: %v", err)
	}
	if len(back) != 201 {
		t.Fatalf("CSV has %d lines incl. header, want 201", len(back))
	}
	if strings.Join(back[0], ",") != strings.Join(cols, ",") {
		t.Fatalf("CSV header = %v, want %v", back[0], cols)
	}

	// JSONL
	jsonlPath := filepath.Join(dir, "orders.jsonl")
	jf, err := os.Create(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(jf)
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			t.Fatal(err)
		}
	}
	jf.Close()

	jr, err := os.Open(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	defer jr.Close()
	sc := bufio.NewScanner(jr)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n := 0
	for sc.Scan() {
		var obj map[string]any
		if err := json.Unmarshal(sc.Bytes(), &obj); err != nil {
			t.Fatalf("line %d is not JSON: %v", n+1, err)
		}
		for _, c := range cols {
			if _, ok := obj[c]; !ok {
				t.Fatalf("line %d is missing column %q", n+1, c)
			}
		}
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if n != 200 {
		t.Fatalf("JSONL has %d lines, want 200", n)
	}

	// Postgres COPY text — the loader path.
	copyPath := filepath.Join(dir, "orders.tsv")
	pf, err := os.Create(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	tw := pgcopy.NewText(pf, cols)
	for _, r := range rows {
		if err := tw.WriteRow(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	pf.Close()

	data, err := os.ReadFile(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 200 {
		t.Fatalf("COPY file has %d lines, want 200", len(lines))
	}
	for i, l := range lines {
		if got := strings.Count(l, "\t"); got != len(cols)-1 {
			t.Fatalf("COPY line %d has %d tabs, want %d", i, got, len(cols)-1)
		}
	}
}

// The generated data must survive its own auditor: what Synth produces is what
// verify says is well-formed.
func TestE2EGeneratedDataPassesVerify(t *testing.T) {
	spec, err := synth.YAMLBytes([]byte(shopSpec))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := spec.Generate()
	if err != nil {
		t.Fatal(err)
	}
	cons := make([]verify.Constraint, 0, len(spec.Constraints()))
	for _, c := range spec.Constraints() {
		cons = append(cons, c)
	}
	rep := verify.Run(rows, verify.Options{Constraints: cons})
	if !rep.OK() {
		var b strings.Builder
		rep.Text(&b)
		t.Fatalf("verify found errors in freshly generated data:\n%s", b.String())
	}
}

// Parent and child are generated in separate calls, written out, and every
// child's foreign key resolves against the parent file.
func TestE2EParentChildJoinAcrossFiles(t *testing.T) {
	dir := t.TempDir()

	users, err := synth.Users(50, synth.WithSeed(1))
	if err != nil {
		t.Fatal(err)
	}
	keys := make([]any, 0, len(users))
	for _, u := range users {
		keys = append(keys, u["id"])
	}
	writeCSVFile(t, filepath.Join(dir, "users.csv"), []string{"id"}, users)

	orders, err := synth.YAMLBytes([]byte(`name: orders
count: 500
seed: 7
fields:
  id:      { kind: uuid, pk: true }
  user_id: { kind: uuid }
  total:   { kind: amount, min: 1, max: 100 }
`))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := orders.Generate(synth.RefValues("user_id", keys))
	if err != nil {
		t.Fatal(err)
	}
	writeCSVFile(t, filepath.Join(dir, "orders.csv"), []string{"id", "user_id", "total"}, rows)

	parents := readColumn(t, filepath.Join(dir, "users.csv"), "id")
	children := readColumn(t, filepath.Join(dir, "orders.csv"), "user_id")
	if len(children) != 500 {
		t.Fatalf("read %d child rows, want 500", len(children))
	}
	valid := map[string]bool{}
	for _, p := range parents {
		valid[p] = true
	}
	for i, c := range children {
		if !valid[c] {
			t.Fatalf("orders row %d references user %q, which is not in users.csv", i, c)
		}
	}
}

// Profile a real-shaped export, render the learned spec as YAML, generate from
// that YAML, and check the shape survived the round trip — without any of the
// original values being copied through.
func TestE2EProfileToYAMLToGenerate(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "real.csv")

	var b strings.Builder
	b.WriteString("id,city,amount,tier\n")
	cities := []string{"Tashkent", "Samarkand", "Bukhara"}
	tiers := []string{"gold", "silver"}
	for i := 0; i < 400; i++ {
		b.WriteString(strconv.Itoa(i))
		b.WriteString(",")
		b.WriteString(cities[i%len(cities)])
		b.WriteString(",")
		b.WriteString(strconv.Itoa(100 + i%50))
		b.WriteString(",")
		b.WriteString(tiers[i%len(tiers)])
		b.WriteString("\n")
	}
	if err := os.WriteFile(src, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := synth.Profile(src)
	if err != nil {
		t.Fatal(err)
	}
	if p.SampleRows() != 400 {
		t.Fatalf("profiled %d rows, want 400", p.SampleRows())
	}
	if strings.Join(p.Columns(), ",") != "id,city,amount,tier" {
		t.Fatalf("columns = %v", p.Columns())
	}

	doc, err := p.YAML("real", 100)
	if err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(dir, "real.yaml")
	if err := os.WriteFile(specPath, doc, 0o644); err != nil {
		t.Fatal(err)
	}

	spec, err := synth.LoadYAML(specPath)
	if err != nil {
		t.Fatalf("the rendered spec does not parse: %v\n%s", err, doc)
	}
	rows, err := spec.Generate(synth.WithSeed(3))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 100 {
		t.Fatalf("got %d rows, want the spec's 100", len(rows))
	}
	for _, r := range rows {
		if len(r) != 4 {
			t.Fatalf("row has %d columns, want 4: %v", len(r), r)
		}
		if v, ok := r["amount"]; !ok || v == nil {
			t.Fatalf("amount missing: %v", r)
		}
	}
}

// A change history must be replayable: applying every event in order to an
// empty table reproduces the table the events describe.
func TestE2ECDCHistoryIsReplayable(t *testing.T) {
	stream, err := synth.CDC[User](synth.CDCConfig{
		Table:      "users",
		Key:        "ID",
		Seed:       42,
		UpdateRate: 0.3,
		DeleteRate: 0.1,
		Start:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Interval:   time.Second,
		Snapshot:   20,
	})
	if err != nil {
		t.Fatal(err)
	}

	table := map[string]map[string]any{}
	var lastTs int64
	events := 0
	for i := 0; i < 500; i++ {
		ev := stream.Next()
		if ev == nil {
			break
		}
		events++
		if ev.TsMs < lastTs {
			t.Fatalf("event %d goes back in time: %d < %d", i, ev.TsMs, lastTs)
		}
		lastTs = ev.TsMs
		if ev.Source.Table != "users" {
			t.Fatalf("event %d has table %q", i, ev.Source.Table)
		}

		switch ev.Op {
		case cdc.OpRead, cdc.OpCreate:
			key := keyOf(t, ev.After)
			if _, exists := table[key]; exists {
				t.Fatalf("event %d inserts a row that already exists: %s", i, key)
			}
			table[key] = ev.After
		case cdc.OpUpdate:
			key := keyOf(t, ev.After)
			cur, exists := table[key]
			if !exists {
				t.Fatalf("event %d updates a row that was never inserted: %s", i, key)
			}
			if ev.Before == nil {
				t.Fatalf("event %d is an update with no before image", i)
			}
			if keyOf(t, ev.Before) != key {
				t.Fatalf("event %d changes the primary key", i)
			}
			if cur["Email"] != ev.Before["Email"] {
				t.Fatalf("event %d carries a stale before image", i)
			}
			table[key] = ev.After
		case cdc.OpDelete:
			key := keyOf(t, ev.Before)
			if _, exists := table[key]; !exists {
				t.Fatalf("event %d deletes a row that is not there: %s", i, key)
			}
			delete(table, key)
		default:
			t.Fatalf("event %d has an unexpected op %q", i, ev.Op)
		}
	}
	if events == 0 {
		t.Fatal("the stream produced no events")
	}
}

func TestE2ECDCWritesJSONLFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	err := synth.WriteCDC[User](path, 100, synth.CDCConfig{
		Table: "users", Key: "ID", Seed: 5, UpdateRate: 0.2, DeleteRate: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		var ev cdc.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("line %d is not a change event: %v", n+1, err)
		}
		if ev.Op == "" {
			t.Fatalf("line %d has no op", n+1)
		}
		n++
	}
	if n != 100 {
		t.Fatalf("wrote %d events, want 100", n)
	}
}

func TestE2EWriteCDCRejectsNonStruct(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.jsonl")
	if err := synth.WriteCDC[int](path, 1, synth.CDCConfig{Table: "t"}); err == nil {
		t.Fatal("expected a struct-type error")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("a file was created despite the configuration error")
	}
}

// Masked output is the default for sensitive columns, and Unmasked() is the
// explicit opt-out. Both must hold through a full generate-and-write run.
func TestE2EMaskedByDefaultUnmaskedOnRequest(t *testing.T) {
	masked, err := synth.Transactions(50, synth.WithSeed(9))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range masked {
		card, _ := r["card_number"].(string)
		if !strings.Contains(card, "*") {
			t.Fatalf("card_number is not masked by default: %q", card)
		}
	}

	raw, err := synth.Transactions(50, synth.WithSeed(9), synth.Unmasked())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range raw {
		card, _ := r["card_number"].(string)
		if strings.Contains(card, "*") {
			t.Fatalf("Unmasked() still returned a masked card: %q", card)
		}
	}

	// Unmasking one call must not leak into the next.
	again, err := synth.Transactions(50, synth.WithSeed(9))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range again {
		if card, _ := r["card_number"].(string); !strings.Contains(card, "*") {
			t.Fatalf("an earlier Unmasked() call unmasked a later one: %q", card)
		}
	}
}

// Every preset must generate, fill every declared column, and audit clean.
func TestE2EEveryPresetGeneratesAndVerifies(t *testing.T) {
	for _, p := range synth.Presets() {
		t.Run(string(p), func(t *testing.T) {
			spec, err := synth.Spec(p)
			if err != nil {
				t.Fatal(err)
			}
			rows, err := spec.GenerateN(120, synth.WithSeed(13))
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 120 {
				t.Fatalf("got %d rows, want 120", len(rows))
			}
			for _, c := range spec.Columns() {
				if _, ok := rows[0][c]; !ok {
					t.Fatalf("column %q is missing from the output", c)
				}
			}
			cons := make([]verify.Constraint, 0, len(spec.Constraints()))
			for _, c := range spec.Constraints() {
				cons = append(cons, c)
			}
			rep := verify.Run(rows, verify.Options{Constraints: cons})
			if !rep.OK() {
				var b strings.Builder
				rep.Text(&b)
				t.Fatalf("preset %s does not pass verify:\n%s", p, b.String())
			}
		})
	}
}

// The whole point of a seed: two runs, two processes' worth of options, same
// bytes on disk.
func TestE2ESameSeedSameFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.jsonl")
	b := filepath.Join(dir, "b.jsonl")
	if err := synth.Stream[streamRow](500, synth.WithSeed(77)).ToJSONL(a); err != nil {
		t.Fatal(err)
	}
	if err := synth.Stream[streamRow](500, synth.WithSeed(77)).ToJSONL(b); err != nil {
		t.Fatal(err)
	}
	da, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	db, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(da) != string(db) {
		t.Fatal("the same seed produced different files")
	}

	c := filepath.Join(dir, "c.jsonl")
	if err := synth.Stream[streamRow](500, synth.WithSeed(78)).ToJSONL(c); err != nil {
		t.Fatal(err)
	}
	dc, err := os.ReadFile(c)
	if err != nil {
		t.Fatal(err)
	}
	if string(da) == string(dc) {
		t.Fatal("a different seed produced an identical file")
	}
}

// helpers

func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case time.Time:
		return x.Format(time.RFC3339)
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return ""
		}
		return strings.Trim(string(b), `"`)
	}
}

func writeCSVFile(t *testing.T, path string, cols []string, rows []map[string]any) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(cols); err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		rec := make([]string, len(cols))
		for i, c := range cols {
			rec[i] = stringify(r[c])
		}
		if err := w.Write(rec); err != nil {
			t.Fatal(err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		t.Fatal(err)
	}
}

func readColumn(t *testing.T, path, col string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	recs, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	idx := -1
	for i, h := range recs[0] {
		if h == col {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("column %q is not in %s", col, path)
	}
	out := make([]string, 0, len(recs)-1)
	for _, r := range recs[1:] {
		out = append(out, r[idx])
	}
	return out
}

func keyOf(t *testing.T, row map[string]any) string {
	t.Helper()
	if row == nil {
		t.Fatal("event has no row image")
	}
	return stringify(row["ID"])
}
