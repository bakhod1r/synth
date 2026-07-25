package providers

import (
	"sort"

	"github.com/bakhod1r/synth/schema"
)

// Which kinds a locale actually changes.
//
// A field can opt out of localization with localize=false, but that switch is
// only meaningful where the locale had any say to begin with. A UUID or an
// HTTP status code is the same string in every locale, so answering "does this
// kind follow the locale?" honestly is worth more than letting a user set
// localize= on a field and wonder why nothing moved.
//
// Two sources feed the answer: kinds whose provider reads the locale or the
// record's place directly (names, addresses, phone, currency, national IDs),
// and catalog kinds that have per-locale datasets (see LocalesFor).

// localeDriven are the kinds whose providers read Ctx.Locale or Ctx.Place.
var localeDriven = map[schema.Kind]bool{
	schema.KindName:       true,
	schema.KindFirstName:  true,
	schema.KindLastName:   true,
	schema.KindEmail:      true,
	schema.KindUsername:   true,
	schema.KindPhone:      true,
	schema.KindCity:       true,
	schema.KindRegion:     true,
	schema.KindPostcode:   true,
	schema.KindCountry:    true,
	schema.KindStreet:     true,
	schema.KindCompany:    true,
	schema.KindDomain:     true,
	schema.KindURL:        true,
	schema.KindIPv4:       true,
	schema.KindCurrency:   true,
	schema.KindJob:        true,
	schema.KindProduct:    true,
	schema.KindIBAN:       true,
	schema.KindSwift:      true,
	schema.KindCard:       true,
	schema.KindNationalID: true,
	schema.KindPINFL:      true,
	schema.KindTaxID:      true,
	schema.KindSSN:        true,
}

// Localizable reports whether the locale changes what this kind produces —
// either because its provider reads the locale directly, or because at least
// one locale carries its own dataset for it. Fields of any other kind ignore
// localize= entirely, and that is not a bug: there is nothing to translate.
func Localizable(k schema.Kind) bool {
	if localeDriven[k] {
		return true
	}
	return len(LocalesFor(k)) > 0
}

// LocalizableKinds returns every kind Localizable reports true for, sorted.
func LocalizableKinds() []schema.Kind {
	seen := map[schema.Kind]bool{}
	for k := range localeDriven {
		seen[k] = true
	}
	for _, k := range LocalizedKinds() {
		seen[k] = true
	}
	out := make([]schema.Kind, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
