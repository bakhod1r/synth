package ui

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/bakhod1r/synth"
)

// The API is the workbench without the browser: the same two verbs the page
// uses, reachable with curl.
//
//	curl 'localhost:7777/api/preview?preset=transaction&n=5'
//	curl 'localhost:7777/api/generate?preset=user&n=1000&format=csv' -o users.csv
//
// It stays behind the loopback bind in Serve. Reaching it means already having
// a shell on the machine, so it grants nothing new; exposing it publicly would,
// which is why Serve refuses to.

// resolveSpec builds a spec from either a preset named in the query string or a
// JSON body. The query form exists so a fixture is one curl, not a document.
func resolveSpec(r *http.Request, cap int) (*specRequest, *synth.YAMLSpec, []synth.Option, error) {
	q := r.URL.Query()
	name := q.Get("preset")
	if name == "" {
		if r.Method == http.MethodGet {
			return nil, nil, nil, fmt.Errorf("GET needs ?preset=<name>; POST a schema for anything else. See /api/presets")
		}
		req, spec, err := parseSpec(r, cap)
		return req, spec, nil, err
	}

	text, ok := synth.PresetSpec(synth.Preset(name))
	if !ok {
		return nil, nil, nil, fmt.Errorf("unknown preset %q — see /api/presets", name)
	}
	spec, err := synth.YAMLBytes([]byte(text))
	if err != nil {
		return nil, nil, nil, err
	}

	n, err := count(q.Get("n"), cap)
	if err != nil {
		return nil, nil, nil, err
	}
	req := &specRequest{Name: name, Count: n, Locale: q.Get("locale"), Format: q.Get("format")}

	opts := []synth.Option{}
	if req.Locale != "" {
		opts = append(opts, synth.WithLocale(req.Locale))
	}
	if s := q.Get("seed"); s != "" {
		seed, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("seed must be a whole number, got %q", s)
		}
		opts = append(opts, synth.WithSeed(seed))
	}
	// Unmasking is a decision, so it has to be asked for by name and it is
	// reported back in the response headers rather than passing silently.
	if q.Get("unmasked") == "true" {
		opts = append(opts, synth.Unmasked())
	}
	spec.SetCount(n)
	return req, spec, opts, nil
}

// count parses ?n=, defaulting to 10 and clamping to the preview cap.
func count(s string, cap int) (int, error) {
	n := 10
	if s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 {
			return 0, fmt.Errorf("n must be a positive whole number, got %q", s)
		}
		n = v
	}
	if cap > 0 && n > cap {
		n = cap
	}
	return n, nil
}

// handlePresets lists the built-in schemas with their YAML, so a caller can
// start from a preset and then edit it instead of guessing field names.
func handlePresets(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		Name string `json:"name"`
		YAML string `json:"yaml"`
	}
	out := []entry{}
	for _, p := range synth.Presets() {
		text, _ := synth.PresetSpec(p)
		out = append(out, entry{Name: string(p), YAML: text})
	}
	writeJSON(w, out)
}
