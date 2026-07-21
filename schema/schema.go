// Package schema defines the intermediate representation (IR) that Synth's
// engine consumes. Frontends (reflection over structs, or later YAML) produce a
// Schema; the engine and encoders only ever see this — never Go structs.
package schema

// Kind is the semantic type of a field, independent of its Go type.
type Kind string

const (
	KindUUID      Kind = "uuid"
	KindName      Kind = "name"
	KindFirstName Kind = "firstname"
	KindLastName  Kind = "lastname"
	KindEmail     Kind = "email"
	KindPhone     Kind = "phone"
	KindCity      Kind = "city"
	KindRegion    Kind = "region"
	KindCountry   Kind = "country"
	KindPostcode  Kind = "postcode"
	KindInt       Kind = "int"
	KindFloat     Kind = "float"
	KindBool      Kind = "bool"
	KindTime      Kind = "time"
	KindLorem     Kind = "lorem"
	KindIBAN      Kind = "iban"
	KindCard      Kind = "card"
	KindPassport  Kind = "passport"
	KindCompany   Kind = "company"
	KindUsername  Kind = "username"
	KindIPv4      Kind = "ipv4"
	KindURL       Kind = "url"
	KindCurrency  Kind = "currency"
	KindAmount    Kind = "amount"
	KindEnum      Kind = "enum"
	KindStreet    Kind = "street"
	KindColor     Kind = "color"
	KindJob       Kind = "job"
	KindProduct   Kind = "product"
	KindMAC       Kind = "mac"
	KindHexColor  Kind = "hexcolor"
	KindGender    Kind = "gender"
	KindBook      Kind = "book"
	KindMovie     Kind = "movie"
	KindCelebrity Kind = "celebrity"
	KindBand      Kind = "band"
	KindBrand     Kind = "brand"
	// KindCountryName is a random famous country (not tied to the locale, unlike
	// KindCountry which returns the locale's own country).
	KindCountryName Kind = "countryname"
	KindFood        Kind = "food"
	KindAnimal      Kind = "animal"
	KindSport       Kind = "sport"
	KindPlanet      Kind = "planet"
	KindUniversity  Kind = "university"
	KindLanguage    Kind = "language"
	KindEmoji       Kind = "emoji"
	// KindUnknown marks a field the frontend could not infer. The engine
	// leaves it at its zero value and records a Warning.
	KindUnknown Kind = ""
	// KindObject is a nested struct; its shape is in Field.Nested.
	KindObject Kind = "object"
	// KindArray is a slice; its element shape is in Field.Elem.
	KindArray Kind = "array"
)

// Field is one column of a record.
type Field struct {
	// Name is the Go field name (also the CSV/SQL column header).
	Name string
	Kind Kind
	// GoType is the reflect kind hint ("string", "int", "time.Time"...),
	// used by encoders and by min/max coercion.
	GoType string
	// Params holds parsed tag options, e.g. {"min":"18","max":"65"}.
	Params map[string]string
	// From names another field this one derives from (email from=Name).
	From string
	// Match names another field this one must stay coherent with
	// (postcode match=City).
	Match string
	// Unique requests distinct values across the generated set.
	Unique bool
	// PK marks the primary key, used as the join target for Ref.
	PK bool
	// FromRef, when set, means this field is a foreign key filled from a
	// referenced parent slice (see synth.Ref). Holds the parent PK values.
	FromRef []any
	// RefCardinality controls how FromRef values are distributed
	// (0 means uniform-random).
	RefMin, RefMax int
	// Choices holds the value set for KindEnum.
	Choices []string
	// Weights, if set, are relative probabilities for Choices (same length).
	// If empty, Choices are picked uniformly (or by Zipf if Params["dist"]
	// == "zipf").
	Weights []float64
	// Nested is the sub-schema for a KindObject field.
	Nested *Schema
	// Elem describes the element of a KindArray field (its Kind, and for
	// arrays of structs, its own Nested schema).
	Elem *Field
	// ArrMin/ArrMax bound the length of a KindArray field (default 1..3).
	ArrMin, ArrMax int
}

// Schema is an ordered list of fields describing one record type.
type Schema struct {
	Fields []Field
}

// FieldByName returns the field with the given name, or nil.
func (s *Schema) FieldByName(name string) *Field {
	for i := range s.Fields {
		if s.Fields[i].Name == name {
			return &s.Fields[i]
		}
	}
	return nil
}

// Warning records a field the frontend could not fully handle.
type Warning struct {
	Field  string
	Reason string
}
