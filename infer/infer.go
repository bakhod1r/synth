// Package infer turns an untagged field (its name + Go type) into a
// schema.Kind. This is the "just works without tags" path: a synonym table
// first, then a Go-type fallback, then a warning.
package infer

import (
	"strings"

	"github.com/bakhodir/synth/schema"
)

// synonyms maps normalized field-name aliases to a Kind. Extendable via Alias.
var synonyms = map[string]schema.Kind{
	"id": schema.KindUUID, "uuid": schema.KindUUID, "guid": schema.KindUUID,
	"name": schema.KindName, "fullname": schema.KindName, "username": schema.KindName,
	"firstname": schema.KindFirstName, "givenname": schema.KindFirstName,
	"lastname": schema.KindLastName, "surname": schema.KindLastName, "familyname": schema.KindLastName,
	"email": schema.KindEmail, "mail": schema.KindEmail, "emailaddress": schema.KindEmail,
	"phone": schema.KindPhone, "tel": schema.KindPhone, "mobile": schema.KindPhone, "phonenumber": schema.KindPhone,
	"city": schema.KindCity, "town": schema.KindCity,
	"region": schema.KindRegion, "state": schema.KindRegion, "province": schema.KindRegion, "viloyat": schema.KindRegion,
	"postcode": schema.KindPostcode, "zip": schema.KindPostcode, "zipcode": schema.KindPostcode, "postal": schema.KindPostcode,
	"iban": schema.KindIBAN,
	"card": schema.KindCard, "cardnumber": schema.KindCard, "pan": schema.KindCard,
	"passport": schema.KindPassport,
	"age":      schema.KindInt,
	"bio":      schema.KindLorem, "description": schema.KindLorem, "notes": schema.KindLorem,
}

// Alias registers an extra field-name synonym (e.g. Uzbek "ismi" → name).
func Alias(fieldName string, kind schema.Kind) {
	synonyms[normalize(fieldName)] = kind
}

// normalize lowercases and strips separators: "Full_Name" → "fullname".
func normalize(s string) string {
	s = strings.ToLower(s)
	r := strings.NewReplacer("_", "", "-", "", " ", "")
	return r.Replace(s)
}

// Kind infers a field's kind from its name, then its Go type.
// Returns (kind, matchedByName). KindUnknown means the caller should warn.
func Kind(fieldName, goType string) (schema.Kind, bool) {
	if k, ok := synonyms[normalize(fieldName)]; ok {
		return k, true
	}
	switch goType {
	case "time.Time":
		return schema.KindTime, false
	case "uuid.UUID":
		return schema.KindUUID, false
	case "bool":
		return schema.KindBool, false
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return schema.KindInt, false
	case "float32", "float64":
		return schema.KindFloat, false
	case "string":
		return schema.KindLorem, false
	}
	return schema.KindUnknown, false
}

// LinkDependencies wires up automatic coherence between inferred fields:
// name→email (from), city↔postcode↔region share a Place, and multiple time
// fields get an ordering hint by name semantics.
func LinkDependencies(s *schema.Schema) {
	var nameField, timeCreated string
	for i := range s.Fields {
		switch s.Fields[i].Kind {
		case schema.KindName, schema.KindFirstName:
			if nameField == "" {
				nameField = s.Fields[i].Name
			}
		}
	}
	// email derives from the name field, when the email has no explicit from.
	for i := range s.Fields {
		f := &s.Fields[i]
		if f.Kind == schema.KindEmail && f.From == "" && nameField != "" {
			f.From = nameField
		}
	}
	// time ordering: a "created" field becomes the anchor; "updated"/
	// "deleted" come after it.
	for i := range s.Fields {
		f := &s.Fields[i]
		if f.Kind != schema.KindTime {
			continue
		}
		n := normalize(f.Name)
		if strings.Contains(n, "creat") && timeCreated == "" {
			timeCreated = f.Name
		}
	}
	if timeCreated != "" {
		for i := range s.Fields {
			f := &s.Fields[i]
			if f.Kind != schema.KindTime || f.Name == timeCreated {
				continue
			}
			n := normalize(f.Name)
			if strings.Contains(n, "updat") || strings.Contains(n, "delet") || strings.Contains(n, "modif") {
				f.From = timeCreated // engine reads From on time as "after"
			}
		}
	}
}
