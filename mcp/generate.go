package mcp

import (
	"fmt"
	"strings"

	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/schema"
)

// generateArgs is the `generate` tool's input.
//
// Exactly one of Preset and Spec must be set. Preferring one silently would
// leave the caller with rows from a schema they did not ask for and no way to
// tell which one they got.
type generateArgs struct {
	Preset   string `json:"preset"`
	Spec     string `json:"spec"`
	Locale   string `json:"locale"`
	Rows     int    `json:"rows"`
	Seed     uint64 `json:"seed"`
	Unmasked bool   `json:"unmasked"`
}

func handleGenerate(a generateArgs) (any, error) {
	if (a.Preset == "") == (a.Spec == "") {
		return nil, fmt.Errorf("set exactly one of preset= or spec=: " +
			"preset for a built-in schema (call list_presets for the names), " +
			"spec for YAML of your own")
	}
	n, err := rowsWithin(a.Rows)
	if err != nil {
		return nil, err
	}
	if err := inputWithin(a.Spec); err != nil {
		return nil, err
	}

	opts := []synth.Option{}
	if a.Locale != "" {
		opts = append(opts, synth.WithLocale(a.Locale))
	}
	if a.Seed != 0 {
		opts = append(opts, synth.WithSeed(a.Seed))
	}
	if a.Unmasked {
		opts = append(opts, synth.Unmasked())
	}

	if a.Preset != "" {
		if _, ok := synth.PresetSpec(synth.Preset(a.Preset)); !ok {
			return nil, fmt.Errorf("unknown preset %q — call list_presets for the names", a.Preset)
		}
		return synth.Generate(synth.Preset(a.Preset), n, opts...)
	}
	spec, err := synth.YAMLBytes([]byte(a.Spec))
	if err != nil {
		return nil, fmt.Errorf("the spec does not parse: %w", err)
	}
	return spec.GenerateN(n, opts...)
}

// typeInfo is one entry of the catalog.
type typeInfo struct {
	Kind string `json:"kind"`
	// Localized reports whether the values change with the locale. Saying so
	// honestly matters: a caller who assumes every type follows the locale will
	// be surprised by a German dataset full of English cocktail names.
	Localized bool     `json:"localized"`
	Locales   []string `json:"locales,omitempty"`
}

type listTypesArgs struct {
	Search string `json:"search"`
}

// handleListTypes returns the catalog, optionally filtered.
//
// The full list is around 250 entries, which is large but still worth returning
// whole: a model that cannot see a type will invent one, and an invented kind
// fails at generate time with a worse error than a long list.
func handleListTypes(a listTypesArgs) (any, error) {
	q := strings.ToLower(strings.TrimSpace(a.Search))
	out := []typeInfo{}
	for _, k := range providers.Kinds() {
		if k == schema.KindObject || k == schema.KindArray || k == schema.KindUnknown {
			continue // structural, not a value type
		}
		if q != "" && !strings.Contains(string(k), q) {
			continue
		}
		locales := providers.LocalesFor(k)
		out = append(out, typeInfo{
			Kind:      string(k),
			Localized: len(locales) > 0,
			Locales:   locales,
		})
	}
	return out, nil
}

// presetInfo carries the preset's YAML, so a caller can start from it and edit
// rather than guessing field names.
type presetInfo struct {
	Name string `json:"name"`
	YAML string `json:"yaml"`
}

func handleListPresets() (any, error) {
	out := []presetInfo{}
	for _, p := range synth.Presets() {
		text, _ := synth.PresetSpec(p)
		out = append(out, presetInfo{Name: string(p), YAML: text})
	}
	return out, nil
}
