package gen

import (
	"strings"

	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/schema"
)

// Per-field localization: localize=false.
//
// A dataset usually wants one voice — an Uzbek record with an Uzbek name, city
// and weekday. But not every column belongs to that voice. A product catalogue
// shipped worldwide keeps its English category names; an export destined for a
// US partner keeps English weekdays even though the customers are Uzbek. Before
// this, the only lever was the whole-dataset locale, so those columns forced
// everything else back to English with them.
//
//	localize=false   generate this field as if the locale were en_US
//	localize=true    follow the dataset locale (the default; stating it is
//	                 harmless and documents the intent)
//
// The switch only bites on kinds the locale actually reaches — see
// providers.Localizable. On the rest it is a no-op, because there was never
// anything locale-specific to turn off.
//
// It applies to scalar fields. A localize=false on an object or array field is
// not inherited by its members: the members carry their own setting, so that
// opting one column out never silently de-localizes a whole sub-record.

// baseLocale is what a de-localized field generates as.
const baseLocale = "en_US"

// localizeField reports whether the field follows the dataset locale. Anything
// other than an explicit false keeps the default (localized) behaviour: a typo
// in the value must not quietly switch a column's language.
func localizeField(f *schema.Field) bool {
	v, ok := f.Params["localize"]
	if !ok {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "0", "no", "off":
		return false
	}
	return true
}

// basePlaceFor maps the record's place onto a place in the base locale. It is
// derived from the record's own place rather than drawn from the rng, for two
// reasons: two de-localized address fields in one record must agree with each
// other, and consuming rng here would shift every later value depending on
// whether some field opted out.
func basePlaceFor(base *locale.Locale, place *locale.Place) *locale.Place {
	if base == nil || len(base.Places) == 0 {
		return place
	}
	key := ""
	if place != nil {
		key = place.City + place.Region + place.Postcode
	}
	p := base.Places[int(fnv32(key)%uint32(len(base.Places)))]
	return &p
}

// fnv32 is FNV-1a, inlined to keep this dependency-free and stable across
// releases — the mapping above is part of the generated output, so it must not
// drift with a hash implementation change.
func fnv32(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
