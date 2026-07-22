package providers

import (
	"sort"

	"github.com/bakhod1r/synth/schema"
)

// Locale-aware catalog types.
//
// Most catalog entries — a superhero, a cloud provider, an HTTP header — are
// the same word everywhere, so a locale cannot change them. But a weekday, a
// colour, a dish or a fruit is different in every language, and returning
// "Monday" or "Apple" inside an otherwise Uzbek record is exactly the kind of
// seam that makes generated data look generated.
//
// Coverage here is partial and always will be. What matters is that the gap is
// *visible*: LocalesFor reports which locales a type actually has data for, so
// the UI and the docs can say so instead of letting a user assume that picking
// a locale translated everything.

// localeCatalog maps a locale code to that locale's datasets, keyed by kind.
var localeCatalog = map[string]map[schema.Kind][]string{}

// localized returns a value from the locale's dataset for this kind, falling
// back to the English list when the locale has no data for it.
func localized(c Ctx, kind schema.Kind, fallback []string) string {
	if c.Locale != nil {
		if byKind, ok := localeCatalog[c.Locale.Name]; ok {
			if vals, ok := byKind[kind]; ok && len(vals) > 0 {
				return pick(c.Rand, vals)
			}
		}
	}
	return pick(c.Rand, fallback)
}

// setLocalized registers a kind whose values follow the locale where data
// exists, and fall back to the English list where it does not.
func setLocalized(k schema.Kind, fallback []string) {
	registry[k] = func(c Ctx) any { return localized(c, k, fallback) }
}

// LocalesFor returns the locales that have their own dataset for a kind,
// sorted. An empty result means the kind returns the same values in every
// locale — which is the honest answer for most of the catalog.
func LocalesFor(k schema.Kind) []string {
	var out []string
	for code, byKind := range localeCatalog {
		if vals, ok := byKind[k]; ok && len(vals) > 0 {
			out = append(out, code)
		}
	}
	sort.Strings(out)
	return out
}

// LocalizedKinds returns every kind that has locale-specific data for at least
// one locale, sorted.
func LocalizedKinds() []schema.Kind {
	seen := map[schema.Kind]bool{}
	for _, byKind := range localeCatalog {
		for k, vals := range byKind {
			if len(vals) > 0 {
				seen[k] = true
			}
		}
	}
	out := make([]schema.Kind, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// LocaleValues returns a locale's dataset for a kind, or nil when it has none.
// The slice is a copy: callers inspecting coverage must not be able to mutate
// the catalog.
func LocaleValues(localeCode string, k schema.Kind) []string {
	byKind, ok := localeCatalog[localeCode]
	if !ok {
		return nil
	}
	vals, ok := byKind[k]
	if !ok {
		return nil
	}
	return append([]string(nil), vals...)
}
