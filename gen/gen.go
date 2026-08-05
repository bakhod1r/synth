// Package gen is the engine: schema.Schema + rng → records. It knows nothing
// about Go structs — it produces map[string]any keyed by field name, which the
// public API scatters back into T.
package gen

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"
	"strings"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/schema"
)

// Engine holds a compiled, validated schema ready to generate records.
type Engine struct {
	schema *schema.Schema
	loc    *locale.Locale
	// base is the locale a field falls back to when it sets localize=false.
	base *locale.Locale
	// fieldLoc caches the locales named by locale= overrides, resolved once at
	// compile time so generating a row is still a map lookup per field.
	fieldLoc map[string]*locale.Locale
	order    []int // field indices in dependency order
	// Chaos is the probability [0,1] that a string/numeric field carries an
	// edge-case value instead of a normal one (see WithChaos).
	Chaos float64
	// seen tracks generated values per unique field, to enforce distinctness.
	seen map[string]map[any]bool
	// counter lists the unique fields enforced by record-index suffix rather
	// than by tracking, keyed by field name (see schema.Field.UniqueMode).
	counter map[string]bool
	// err records the first generation failure. Record cannot return an error
	// without changing every caller, so the failure is held here and reported
	// by Err once the run is over.
	err error
	// sub holds compiled engines for nested object schemas, keyed by the
	// nested *schema.Schema pointer.
	sub map[*schema.Schema]*Engine
}

// Compile validates the schema (unknown kinds, from= cycles) and computes the
// field generation order once. Returns an error for configuration problems.
func Compile(s *schema.Schema, localeName string) (*Engine, error) {
	order, err := topoOrder(s)
	if err != nil {
		return nil, err
	}
	for _, f := range s.Fields {
		if f.Kind == schema.KindUnknown {
			continue // handled as zero value + warning upstream
		}
		if f.FromRef != nil {
			continue // filled from a parent slice, no provider needed
		}
		if f.Kind == schema.KindArray && f.Elem == nil {
			// An array declared without an element type — `kind: array` with no
			// items, or a `synth:"array"` tag on a field whose element could
			// not be inferred — has nothing to generate. Saying so at compile
			// time beats a nil dereference on the first record.
			return nil, fmt.Errorf("synth: field %q is an array with no element type", f.Name)
		}
		if f.Kind == schema.KindObject || f.Kind == schema.KindArray {
			continue // handled by recursive sub-engines, not a provider
		}
		if providers.Get(f.Kind) == nil {
			return nil, fmt.Errorf("synth: field %q has unknown kind %q", f.Name, f.Kind)
		}
		// A correlated field is a linear function of its target, and a
		// time-series field reads its axis, so the target must produce a
		// number — an axis a time. topoOrder has already checked the target
		// exists; here we check its type, at compile time, so a mistake is a
		// clear error rather than a column of zeros.
		if d := f.Params["derive"]; d != "" {
			if t := s.FieldByName(d); t != nil && !providers.IsNumericKind(t.Kind) {
				return nil, fmt.Errorf("synth: field %q derives from %q, which is not numeric", f.Name, d)
			}
		}
		if a := f.Params["axis"]; a != "" {
			if t := s.FieldByName(a); t != nil && t.Kind != schema.KindTime && t.Kind != schema.KindUnixTime {
				return nil, fmt.Errorf("synth: field %q uses axis %q, which is not a time field", f.Name, a)
			}
		}
	}
	e := &Engine{schema: s, loc: locale.Get(localeName), base: locale.Get(baseLocale), order: order}
	// Resolve every locale= override up front: an unknown name is a mistake in
	// the spec, and reporting it here beats a column that silently generates in
	// English because locale.Get fell back.
	for i := range s.Fields {
		name := localeParam(&s.Fields[i])
		if name == "" {
			continue
		}
		if !locale.Has(name) {
			return nil, fmt.Errorf("synth: field %q names unknown locale %q", s.Fields[i].Name, name)
		}
		if e.fieldLoc == nil {
			e.fieldLoc = map[string]*locale.Locale{}
		}
		e.fieldLoc[name] = locale.Get(name)
	}
	// Compile nested object schemas recursively (sharing the same locale).
	for i := range s.Fields {
		f := &s.Fields[i]
		var nested *schema.Schema
		if f.Kind == schema.KindObject {
			nested = f.Nested
		} else if f.Kind == schema.KindArray && f.Elem != nil && f.Elem.Kind == schema.KindObject {
			nested = f.Elem.Nested
		}
		if nested != nil {
			sub, err := Compile(nested, localeName)
			if err != nil {
				return nil, err
			}
			if e.sub == nil {
				e.sub = map[*schema.Schema]*Engine{}
			}
			e.sub[nested] = sub
		}
	}
	for _, f := range s.Fields {
		// UUIDs are unique by construction — no stateful tracking needed, which
		// also keeps them safe for parallel generation.
		if !f.Unique || f.Kind == schema.KindUUID {
			continue
		}
		switch f.UniqueMode {
		case "":
			if e.seen == nil {
				e.seen = map[string]map[any]bool{}
			}
			e.seen[f.Name] = map[any]bool{}
		case "counter":
			// The suffix is applied to the rendered value, so the kind only has
			// to produce something printable. Composite kinds are not.
			if f.Kind == schema.KindObject || f.Kind == schema.KindArray {
				return nil, fmt.Errorf("synth: field %q: unique=counter needs a scalar field, got %s", f.Name, f.Kind)
			}
			if e.counter == nil {
				e.counter = map[string]bool{}
			}
			e.counter[f.Name] = true
		default:
			return nil, fmt.Errorf("synth: field %q: unknown unique mode %q (want %q or %q)", f.Name, f.UniqueMode, "", "counter")
		}
	}
	return e, nil
}

// topoOrder returns field indices such that every from=/match= dependency is
// generated before its dependent. Errors on cycles.
func topoOrder(s *schema.Schema) ([]int, error) {
	idx := map[string]int{}
	for i, f := range s.Fields {
		idx[f.Name] = i
	}
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make([]int, len(s.Fields))
	var order []int
	var visit func(i int) error
	visit = func(i int) error {
		switch color[i] {
		case gray:
			return fmt.Errorf("synth: dependency cycle involving field %q", s.Fields[i].Name)
		case black:
			return nil
		}
		color[i] = gray
		// derive (a correlated numeric reads another field) and axis (a
		// time-series reads its timestamp) are dependency edges like from and
		// match: the referenced field must be generated first.
		deps := []string{s.Fields[i].From, s.Fields[i].Match,
			s.Fields[i].Params["derive"], s.Fields[i].Params["axis"]}
		for _, dep := range deps {
			if dep == "" {
				continue
			}
			j, ok := idx[dep]
			if !ok {
				return fmt.Errorf("synth: field %q references unknown field %q", s.Fields[i].Name, dep)
			}
			if err := visit(j); err != nil {
				return err
			}
		}
		color[i] = black
		order = append(order, i)
		return nil
	}
	for i := range s.Fields {
		if err := visit(i); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// Record generates one record. seq is the record index, used with the base
// rng to fork a deterministic per-record stream.
func (e *Engine) Record(base *rng.Rand, seq int) map[string]any {
	r := base.Fork(uint64(seq))
	place := e.loc.Places[r.Pick(len(e.loc.Places))]
	gender := "male"
	if r.Bool(0.5) {
		gender = "female"
	}
	return e.generate(r, &place, gender, seq)
}

// uniqueAttempts caps the resampling loop for tracked unique fields. Past this
// many collisions in a row the field's value space is, for practical purposes,
// used up, and more draws would only burn time before landing on a duplicate.
const uniqueAttempts = 1000

// generate fills one record using the given rng, locale place and gender
// (shared so nested objects and name/gender fields stay coherent). seq is the
// record index, used by unique=counter fields to derive distinctness.
func (e *Engine) generate(r *rng.Rand, place *locale.Place, gender string, seq int) map[string]any {
	values := make(map[string]any, len(e.schema.Fields))
	for _, i := range e.order {
		f := &e.schema.Fields[i]
		v := e.field(r, f, place, gender, values, seq)
		if e.Chaos > 0 && f.FromRef == nil && r.Bool(e.Chaos) {
			v = chaosValue(r, v)
		}
		// Unique enforcement: resample until a fresh value is found. Running out
		// of values is reported rather than papered over — a silent duplicate in
		// a column declared unique is the kind of defect that surfaces later, in
		// somebody else's database, as a failed import.
		if seen := e.seen[f.Name]; seen != nil {
			attempt := 0
			for seen[v] && attempt < uniqueAttempts {
				v = e.field(r, f, place, gender, values, seq)
				attempt++
			}
			if seen[v] {
				e.fail(fmt.Errorf("synth: field %q ran out of unique values after %d rows; its value space is too small for the row count (use unique=counter, or widen the field)", f.Name, len(seen)))
			}
			seen[v] = true
		}
		if e.counter[f.Name] {
			v = withCounter(v, seq)
		}
		values[f.Name] = v
	}
	return values
}

// withCounter makes a value distinct by folding in the record index. Numbers
// become the index itself, so an integer key stays an integer; everything else
// is rendered and suffixed, with emails keeping their domain so the result is
// still a valid address.
func withCounter(v any, seq int) any {
	switch t := v.(type) {
	case nil:
		return seq
	case int:
		return seq
	case int64:
		return int64(seq)
	case float64:
		return float64(seq)
	case string:
		if local, domain, ok := strings.Cut(t, "@"); ok {
			return local + strconv.Itoa(seq) + "@" + domain
		}
		return t + "_" + strconv.Itoa(seq)
	default:
		return fmt.Sprint(t) + "_" + strconv.Itoa(seq)
	}
}

// elemSeq derives a distinct index for element i of a repeated field, so that
// unique=counter inside an array stays distinct between elements of the same
// record as well as between records. The shift bounds arrays at 65536 elements
// before two elements could share an index, which is far past any array a
// generated record has reason to hold.
func elemSeq(seq, i int) int { return seq<<16 | i&0xffff }

// fail records the first generation error; later ones are dropped, since they
// are almost always the same problem repeating once per row.
func (e *Engine) fail(err error) {
	if e.err == nil {
		e.err = err
	}
}

// Err reports the first error hit while generating records, if any. Record
// cannot return one, so callers that can surface an error check this after
// their generation loop.
func (e *Engine) Err() error { return e.err }

func (e *Engine) field(r *rng.Rand, f *schema.Field, place *locale.Place, gender string, values map[string]any, seq int) any {
	// A blank share applies to every kind, so it is handled here rather than
	// asked of each provider. Real tables have missing values, and code that
	// has only ever seen complete rows breaks the first time it meets a null.
	//
	// The draw happens before anything else so the cost of a blanked field is
	// nothing, and a primary key is never blanked: a row without an identity
	// is not missing data, it is a broken row.
	if blank := blankShare(f); blank > 0 && !f.PK && !f.Unique && r.Bool(blank) {
		return nil
	}
	// Foreign key: draw from the referenced parent's PK values.
	if f.FromRef != nil {
		return f.FromRef[r.Pick(len(f.FromRef))]
	}
	if f.Kind == schema.KindUnknown {
		return nil
	}
	// Nested object: generate a sub-record with the same rng, place and gender.
	if f.Kind == schema.KindObject {
		if sub := e.sub[f.Nested]; sub != nil {
			return sub.generate(r, place, gender, seq)
		}
		return nil
	}
	// Array: generate a slice of elements (scalars or nested objects).
	if f.Kind == schema.KindArray {
		return e.array(r, f, place, gender, values, seq)
	}
	p := providers.Get(f.Kind)
	loc, pl := e.fieldLocale(f, place)
	c := providers.Ctx{
		Rand:   r,
		Locale: loc,
		Params: f.Params,
		Field:  f,
		Place:  pl,
		Gender: gender,
		Sibling: func(name string) any {
			if name == "__from__" {
				if f.From != "" {
					return values[f.From]
				}
				return nil
			}
			return values[name]
		},
	}
	return clampLen(f, maskValue(f, p(c)))
}

// clampLen truncates a string value to the field's maxlen= in runes.
//
// The limit comes from the schema the user gave us — varchar(n) in a CREATE
// TABLE, maxLength in an OpenAPI document, or the tag directly. Enforcing it
// here rather than only in the generated DDL is the point: a limit that lives
// only in the table surfaces as "value too long for type character varying(n)"
// halfway through a load, after the file is already written.
//
// Runes, not bytes — cutting a Cyrillic or CJK name at a byte offset splits a
// character and yields invalid UTF-8, which is a worse bug than the one being
// prevented.
//
// It runs after masking, since a mask changes the length.
func clampLen(f *schema.Field, v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	n, err := strconv.Atoi(f.Params["maxlen"])
	if err != nil || n <= 0 {
		return v
	}
	r := []rune(s)
	if len(r) <= n {
		return v
	}
	return string(r[:n])
}

// maskValue applies the field's mask= setting to a generated value.
//
// This is what makes a fixture safe to paste into a ticket or a screenshot: a
// card number or a national identifier is realistic in shape but reveals
// nothing, even by accident. It runs on the generated value rather than on real
// data — for masking an actual export, see the mask package.
//
//	mask=partial   keep the last 4 characters, star the rest — on a card
//	               number, the first 4 as well (the BIN, which identifies the
//	               issuer and not the cardholder)
//	mask=hash      SHA-256 hex of the value
//	mask=redact    a fixed marker, no shape preserved
//	mask=token     an opaque reference, unlinkable to the value
//
// hash and token take three further params:
//
//	salt=…         mixed in before hashing, so two datasets built from the same
//	               values do not share digests
//	secret=…       switches to HMAC-SHA256 with this key. Without the key the
//	               digest cannot be recomputed, which is what stops an attacker
//	               from confirming a guessed value by hashing it.
//	digest=16      truncate the hex to this many characters
func maskValue(f *schema.Field, v any) any {
	mode := f.Params["mask"]
	if mode == "" || v == nil {
		return v
	}
	s := fmt.Sprint(v)
	switch mode {
	case "partial":
		if f.Kind == schema.KindCard {
			return cardMask(s)
		}
		return partialMask(s)
	case "hash":
		return truncate(f, digest(f, s))
	case "redact":
		return "[REDACTED]"
	case "token":
		// A token is a reference, not a fingerprint: 24 hex characters is
		// already far more than a table will ever need to stay collision-free,
		// and a shorter value is easier to read in a log.
		d := truncate(f, digest(f, s))
		if len(d) > 24 {
			d = d[:24]
		}
		return "tok_" + d
	}
	return v
}

// scoped binds a masked value to its column before hashing.
//
// Without this, the same national identifier masked in two different columns
// produces the same digest, so anyone holding both tables can join on the
// masked value and re-link the rows the mask was meant to separate. It also
// makes a single rainbow table work against every column at once.
//
// Mixing the column name in keeps the property that matters — the same value in
// the same column always masks the same way, so the column is still joinable —
// while making cross-column correlation useless. The separator cannot occur in
// a column name, so "ab"+"c" and "a"+"bc" cannot collide.
func scoped(f *schema.Field, s string) []byte {
	return []byte(f.Name + "\x00" + f.Params["salt"] + "\x00" + s)
}

// hasher picks the digest's hash function from algo=. Both are in the standard
// library, so the choice adds no dependency. SHA-256 stays the default, which
// keeps every spec written before this option byte-identical.
//
//	algo=sha256   (default) 64 hex characters
//	algo=sha512   128 hex characters, for callers who standardise on it
func hasher(f *schema.Field) func() hash.Hash {
	switch strings.ToLower(strings.TrimSpace(f.Params["algo"])) {
	case "sha512":
		return sha512.New
	default:
		return sha256.New
	}
}

// digest hashes a value for the hash and token masks.
//
// With secret= it is an HMAC rather than a bare hash. That difference matters
// whenever the set of possible values is small: a national identifier or a
// phone number can simply be enumerated and hashed until the digest matches, so
// a plain hash of one hides nothing. A key the attacker does not have removes
// that attack entirely.
func digest(f *schema.Field, s string) string {
	newHash := hasher(f)
	if key := f.Params["secret"]; key != "" {
		m := hmac.New(newHash, []byte(key))
		m.Write(scoped(f, s))
		return hex.EncodeToString(m.Sum(nil))
	}
	h := newHash()
	h.Write(scoped(f, s))
	return hex.EncodeToString(h.Sum(nil))
}

// truncate shortens a digest to digest=N characters.
//
// Shortening raises the chance that two different values collide, which in a
// masked column means two different people silently becoming one row. Sixteen
// hex characters is the floor: below that, collisions stop being theoretical
// in a table of any size.
func truncate(f *schema.Field, s string) string {
	n, err := strconv.Atoi(f.Params["digest"])
	if err != nil || n <= 0 || n >= len(s) {
		return s
	}
	if n < 16 {
		n = 16
	}
	return s[:n]
}

// partialMask keeps the last four characters. Fewer than four would reveal
// most of a short value, so a short value is starred completely rather than
// half-shown.
func partialMask(s string) string {
	const keep = 4
	r := []rune(s)
	if len(r) <= keep {
		return strings.Repeat("*", len(r))
	}
	var b strings.Builder
	for i, c := range r {
		switch {
		case i >= len(r)-keep:
			b.WriteRune(c)
		case c == '-' || c == ' ' || c == '/':
			b.WriteRune(c) // keep the shape readable
		default:
			b.WriteRune('*')
		}
	}
	return b.String()
}

// cardMask keeps the leading four digits and the trailing four. The first four
// are the start of the BIN: they identify the issuer, not the cardholder, and
// keeping them lets a fixture stay recognizably a Visa or a HUMO card while the
// account number stays hidden. PCI DSS permits up to the first six and the last
// four; four is the stricter choice.
//
// Below twelve digits the two windows would meet and reveal nearly everything,
// so a short value falls back to the trailing-four rule.
func cardMask(s string) string {
	const lead, tail = 4, 4
	r := []rune(s)
	digits := 0
	for _, c := range r {
		if c >= '0' && c <= '9' {
			digits++
		}
	}
	if digits < 12 {
		return partialMask(s)
	}
	var b strings.Builder
	seen := 0
	for _, c := range r {
		if c < '0' || c > '9' {
			b.WriteRune(c) // keep the shape readable
			continue
		}
		seen++
		if seen <= lead || seen > digits-tail {
			b.WriteRune(c)
		} else {
			b.WriteRune('*')
		}
	}
	return b.String()
}

// array generates a slice value for a KindArray field.
func (e *Engine) array(r *rng.Rand, f *schema.Field, place *locale.Place, gender string, values map[string]any, seq int) []any {
	min, max := f.ArrMin, f.ArrMax
	if max < min {
		max = min
	}
	if max == 0 {
		min, max = 1, 3
	}
	n := r.IntRange(min, max)
	out := make([]any, n)
	for i := 0; i < n; i++ {
		if f.Elem.Kind == schema.KindObject {
			if sub := e.sub[f.Elem.Nested]; sub != nil {
				out[i] = sub.generate(r, place, gender, elemSeq(seq, i))
				continue
			}
			out[i] = nil
			continue
		}
		out[i] = e.field(r, f.Elem, place, gender, values, elemSeq(seq, i))
	}
	return out
}

// Schema exposes the underlying schema (for encoders needing column order).
func (e *Engine) Schema() *schema.Schema { return e.schema }

// HasUnique reports whether any field requires unique values. Unique tracking
// is stateful, so callers must use serial generation when this is true.
func (e *Engine) HasUnique() bool { return e.seen != nil }

// blankShare reads the field's blank probability. It accepts a fraction
// ("0.15") or a percentage ("15%"), because both readings of "blank: 15" are
// natural and guessing wrong by a factor of a hundred is an unpleasant
// surprise: a bare number is read as a percentage, matching the label users
// see in the interface.
func blankShare(f *schema.Field) float64 {
	raw, ok := f.Params["blank"]
	if !ok || raw == "" {
		return 0
	}
	raw = strings.TrimSpace(raw)
	percent := strings.HasSuffix(raw, "%")
	raw = strings.TrimSuffix(raw, "%")

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 0
	}
	// "0.15" means fifteen percent; "15" and "15%" mean the same thing.
	if percent || v > 1 {
		v /= 100
	}
	if v > 1 {
		return 1
	}
	return v
}
