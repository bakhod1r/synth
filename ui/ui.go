// Package ui serves a local browser workbench for designing a schema and
// seeing the data it produces.
//
// # Security boundary
//
// The server binds loopback only and refuses anything else. The page is served
// from an embedded filesystem with inline CSS and JavaScript: no CDN, no fonts,
// no telemetry, no outbound request of any kind. Data is generated and files
// are written on your own machine and nothing leaves it.
//
// This is consistent with Synth's no-network rule rather than an exception to
// it: the browser connects in, Synth never connects out.
package ui

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"time"

	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/internal/webspec"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/schema"
)

//go:embed static
var static embed.FS

// maxPreview caps preview size. The browser has to render this on every
// keystroke, so a request for a million rows must not be honored.
const maxPreview = 100

// Handler returns every route, so tests can exercise the API without opening
// a socket.
func Handler() http.Handler {
	mux := http.NewServeMux()

	sub, err := fs.Sub(static, "static")
	if err != nil {
		panic("ui: embedded assets missing: " + err.Error())
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/types", handleTypes)
	mux.HandleFunc("/api/locales", handleLocales)
	mux.HandleFunc("/api/preview", handlePreview)
	mux.HandleFunc("/api/generate", handleGenerate)
	mux.HandleFunc("/api/presets", handlePresets)
	return mux
}

// Serve runs the workbench. It refuses to bind a non-loopback address: a data
// generator that answers to the network is a very different, much riskier
// thing than one that answers to you.
func Serve(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("ui: %q is not host:port: %w", addr, err)
	}
	if host == "" {
		host = "127.0.0.1"
		addr = net.JoinHostPort(host, portOf(addr))
	}
	ip := net.ParseIP(host)
	if ip == nil {
		if host != "localhost" {
			return fmt.Errorf("ui: refusing to bind %q — the workbench is "+
				"loopback-only by design", addr)
		}
	} else if !ip.IsLoopback() {
		return fmt.Errorf("ui: refusing to bind %q — the workbench is "+
			"loopback-only by design", addr)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Printf("synth ui on http://%s (loopback only; nothing leaves this machine)\n", addr)
	return srv.ListenAndServe()
}

func portOf(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "8080"
	}
	return port
}

// typeInfo describes one generatable kind.
type typeInfo struct {
	Kind     string `json:"kind"`
	Category string `json:"category"`
	// Localized reports whether this type's values change with the locale.
	// The UI shows this honestly rather than implying every type is localized.
	Localized bool `json:"localized"`
	// Locales lists the locales that have their own dataset for this type,
	// empty when the type is the same everywhere or is localized structurally
	// (a name or an address) rather than through a per-locale word list.
	Locales []string `json:"locales,omitempty"`
}

func handleTypes(w http.ResponseWriter, r *http.Request) {
	kinds := providers.Kinds()
	out := make([]typeInfo, 0, len(kinds))
	for _, k := range kinds {
		if k == schema.KindObject || k == schema.KindArray || k == schema.KindUnknown {
			continue // structural, not a value type
		}
		covered := providers.LocalesFor(k)
		out = append(out, typeInfo{
			Kind:      string(k),
			Category:  webspec.CategoryOf(k),
			Localized: webspec.IsLocalized(k, covered),
			Locales:   covered,
		})
	}
	writeJSON(w, out)
}

func handleLocales(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, locale.Names())
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	_, spec, opts, err := resolveSpec(r, maxPreview)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rows, err := spec.Generate(opts...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, rows)
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	req, spec, opts, err := resolveSpec(r, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rows, err := spec.Generate(opts...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := req.Name
	if name == "" {
		name = "data"
	}
	cols := spec.Columns()

	switch req.Format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		json.NewEncoder(w).Encode(rows)
	case "jsonl":
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.jsonl"`)
		enc := json.NewEncoder(w)
		for _, row := range rows {
			if err := enc.Encode(row); err != nil {
				return
			}
		}
	case "sql":
		w.Header().Set("Content-Type", "application/sql")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.sql"`)
		webspec.WriteSQL(w, name, cols, rows)
	default:
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.csv"`)
		webspec.WriteCSV(w, cols, rows)
	}
}

// parseSpec turns the posted JSON into a spec, rejecting an unknown kind by
// name. Making a mistake visible is the whole reason the workbench exists.
func parseSpec(r *http.Request, cap int) (*webspec.Request, *synth.YAMLSpec, error) {
	var req webspec.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, nil, fmt.Errorf("cannot read the schema: %w", err)
	}
	if len(req.Fields) == 0 {
		return nil, nil, fmt.Errorf("the schema has no fields")
	}
	for name, def := range req.Fields {
		kind, _ := def["kind"].(string)
		if kind == "" {
			return nil, nil, fmt.Errorf("field %q has no kind", name)
		}
		if !providers.Has(schema.Kind(kind)) && kind != "enum" {
			return nil, nil, fmt.Errorf("unknown kind %q on field %q", kind, name)
		}
	}
	if req.Count <= 0 {
		req.Count = 10
	}
	if cap > 0 && req.Count > cap {
		req.Count = cap
	}
	doc, err := webspec.BuildYAML(&req)
	if err != nil {
		return nil, nil, err
	}
	spec, err := synth.YAMLBytes(doc)
	if err != nil {
		return nil, nil, err
	}
	return &req, spec, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	// The page is same-origin and served from this binary; nothing here should
	// ever be readable cross-origin.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	json.NewEncoder(w).Encode(v)
}
