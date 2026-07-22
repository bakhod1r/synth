// Package webspec holds the pieces the workbench needs on both sides of its
// two backends: the local HTTP server and the WebAssembly build that runs the
// same page with no server at all.
//
// It exists so those two cannot drift. The page's JavaScript is byte-identical
// between them, so the schema it posts and the CSV it gets back have to be too
// — and the only way to guarantee that is one implementation rather than two
// that look alike today.
//
// Nothing here knows about HTTP. That is what makes it importable from a wasm
// binary without dragging net/http along.
package webspec

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bakhod1r/synth/schema"
)

// Request is what the page posts: the same shape as a YAML spec.
type Request struct {
	Name   string                    `json:"name"`
	Count  int                       `json:"count"`
	Locale string                    `json:"locale"`
	Seed   uint64                    `json:"seed"`
	Format string                    `json:"format"`
	Fields map[string]map[string]any `json:"fields"`
	Order  []string                  `json:"order"`
}

// buildYAML renders the posted schema as a YAML spec, so the UI and the CLI
// go through exactly the same parser. A preview that took a different path
// than `synth gen` would be a preview of the wrong thing.
func BuildYAML(req *Request) ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", NonEmpty(req.Name, "data"))
	fmt.Fprintf(&b, "count: %d\n", req.Count)
	if req.Locale != "" {
		fmt.Fprintf(&b, "locale: %s\n", req.Locale)
	}
	fmt.Fprintf(&b, "seed: %d\n", req.Seed)
	b.WriteString("fields:\n")

	order := req.Order
	if len(order) == 0 {
		for name := range req.Fields {
			order = append(order, name)
		}
		sort.Strings(order)
	}
	for _, name := range order {
		def, ok := req.Fields[name]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "  %q: {%s}\n", name, renderDef(def))
	}
	return []byte(b.String()), nil
}

// renderDef emits one field's inline mapping. Keys are sorted so the same
// schema always produces the same document.
func renderDef(def map[string]any) string {
	keys := make([]string, 0, len(def))
	for k := range def {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		switch v := def[k].(type) {
		case string:
			parts = append(parts, fmt.Sprintf("%s: %q", k, v))
		case bool:
			parts = append(parts, fmt.Sprintf("%s: %t", k, v))
		case float64:
			parts = append(parts, fmt.Sprintf("%s: %g", k, v))
		case []any:
			items := make([]string, len(v))
			for i, item := range v {
				items[i] = fmt.Sprintf("%q", fmt.Sprint(item))
			}
			parts = append(parts, fmt.Sprintf("%s: [%s]", k, strings.Join(items, ", ")))
		default:
			parts = append(parts, fmt.Sprintf("%s: %q", k, fmt.Sprint(v)))
		}
	}
	return strings.Join(parts, ", ")
}

func NonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func WriteCSV(w io.Writer, cols []string, rows []map[string]any) {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	cw.Write(cols)
	rec := make([]string, len(cols))
	for _, r := range rows {
		for i, c := range cols {
			rec[i] = fmt.Sprint(r[c])
		}
		cw.Write(rec)
	}
}

func WriteSQL(w io.Writer, table string, cols []string, rows []map[string]any) {
	colList := strings.Join(cols, ", ")
	for _, r := range rows {
		vals := make([]string, len(cols))
		for i, c := range cols {
			vals[i] = sqlValue(r[c])
		}
		fmt.Fprintf(w, "INSERT INTO %s (%s) VALUES (%s);\n", table, colList, strings.Join(vals, ", "))
	}
}

func sqlValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(x)
	case bool:
		if x {
			return "TRUE"
		}
		return "FALSE"
	default:
		return "'" + strings.ReplaceAll(fmt.Sprint(x), "'", "''") + "'"
	}
}

// categoryOf groups a kind for the palette. The grouping is presentational
// only — an unrecognized kind still appears, under "other", so a newly added
// type is never invisible.
func CategoryOf(k schema.Kind) string {
	name := string(k)
	for _, g := range categories {
		for _, member := range g.kinds {
			if member == name {
				return g.name
			}
		}
	}
	for _, g := range categories {
		for _, frag := range g.fragments {
			if strings.Contains(name, frag) {
				return g.name
			}
		}
	}
	return "other"
}

var categories = []struct {
	name      string
	kinds     []string
	fragments []string
}{
	{name: "person", kinds: []string{"name", "firstname", "lastname", "middlename",
		"gender", "username", "email", "phone", "title", "namesuffix",
		"maritalstatus", "education", "bloodtype", "nickname", "petname",
		"birthdate", "age", "pinfl", "nationalid", "taxid", "ssn"}},
	{name: "location", kinds: []string{"city", "region", "country", "postcode",
		"street", "countryname", "countrycode", "continent", "timezone",
		"latitude", "longitude", "airport", "airline"}, fragments: []string{"geo"}},
	{name: "finance", kinds: []string{"iban", "card", "currency", "amount",
		"bankname", "accounttype", "paymentmethod", "swift", "salary",
		"stockticker", "crypto", "isin", "lei", "cusip", "routingnumber",
		"accountnumber", "ein", "cardbrand", "cardexpiry", "cardtoken",
		"cvv", "cvc", "cvv2", "cvc2", "csc", "cid", "balance"}},
	{name: "health", kinds: []string{"icd10", "ndc", "drugname"}},
	{name: "internet", kinds: []string{"ipv4", "ipv6", "url", "domain", "mac",
		"macvendor", "useragent", "port", "protocol", "cidr", "asn",
		"httpmethod", "httpstatus", "httpheader", "mimetype", "browser", "os"}},
	{name: "tech", kinds: []string{"uuid", "md5", "sha256", "jwt", "gitcommit",
		"gitbranch", "gittag", "semver", "filename", "filepath", "fileext",
		"filesize", "loglevel", "environment", "awsregion", "cloudprovider",
		"containerimage", "framework", "cron", "errorcode", "base64", "slug"}},
	{name: "commerce", kinds: []string{"product", "productcategory",
		"productmaterial", "brand", "sku", "ean13", "upc", "isbn", "orderstatus",
		"couponcode", "company", "job", "jobarea", "joblevel", "department"}},
	{name: "text", kinds: []string{"lorem", "word", "sentence", "paragraph",
		"catchphrase", "appname", "password"}},
	{name: "basic", kinds: []string{"int", "float", "bool", "time", "unixtime",
		"enum", "year", "month", "weekday", "percentage", "rating",
		"temperature", "duration", "unit"}},
	{name: "culture", kinds: []string{"book", "movie", "celebrity", "band",
		"musicgenre", "instrument", "musicnote", "sportsteam", "sport",
		"superhero", "food", "drink", "cocktail", "coffee", "fruit",
		"vegetable", "animal", "dogbreed", "catbreed", "flower", "language",
		"languagename", "university", "emoji"}},
}
