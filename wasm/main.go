//go:build js && wasm

// Command synth-wasm runs the workbench entirely in the browser.
//
// It answers the same routes the local server does — /api/types, /api/preview,
// /api/generate and the rest — so the page's JavaScript is byte-identical
// between the two. A backend swapped underneath without the frontend noticing
// is the point: one interface to maintain, and the demo cannot drift from the
// tool it demonstrates.
//
// # Why this exists
//
// The hosted alternatives to Synth are services: you paste a schema into
// someone else's server and it hands data back. That is a reasonable product
// and a poor fit for anyone whose schema describes real columns. This page is
// the same convenience with none of that — the generator is compiled to
// WebAssembly and runs in the tab. There is no server to send anything to,
// which is a stronger claim than a promise not to.
package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/internal/webspec"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/providers"
	"github.com/bakhod1r/synth/schema"
)

// maxPreview matches the server's cap. The browser re-renders the preview on
// every keystroke, so a request for a million rows must not be honoured.
const maxPreview = 100

// version is stamped at build time, so the page can say which build it runs
// rather than leaving a bug report unanchored.
var version = "(devel)"

func main() {
	js.Global().Set("synthWasm", js.ValueOf(map[string]any{
		"handle":  js.FuncOf(handle),
		"version": version,
	}))
	// A wasm main that returns exits the program and takes the exported
	// functions with it.
	select {}
}

// handle answers one API call.
//
// It takes (method, path, body) and returns {status, body} rather than
// throwing, because the caller is a fetch shim: a Go panic would surface in
// JavaScript as "Go program has already exited", and every later request would
// fail with the same unhelpful message.
func handle(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = respond(500, fmt.Sprintf("internal error: %v", r))
		}
	}()
	if len(args) < 2 {
		return respond(400, "handle(method, path, body)")
	}
	method, raw := args[0].String(), args[1].String()
	var body string
	if len(args) > 2 && args[2].Type() == js.TypeString {
		body = args[2].String()
	}

	u, err := url.Parse(raw)
	if err != nil {
		return respond(400, "bad path: "+err.Error())
	}
	q := u.Query()

	switch u.Path {
	case "/api/types":
		return json200(types())
	case "/api/locales":
		return json200(locale.Names())
	case "/api/presets":
		return json200(presets())
	case "/api/stats":
		// The page shows totals; in the browser there is nowhere to keep them
		// between reloads, and saying so beats a number that resets silently.
		return json200(map[string]any{
			"files": 0, "rows": 0, "cells": 0, "bytes": 0, "persistent": false,
		})
	case "/api/preview":
		return generate(method, body, q, maxPreview)
	case "/api/generate":
		return generate(method, body, q, 0)
	}
	return respond(404, "no such endpoint: "+u.Path)
}

// generate mirrors the server's preview and generate handlers.
func generate(method, body string, q url.Values, cap int) any {
	spec, opts, name, format, err := resolve(method, body, q, cap)
	if err != nil {
		return respond(400, err.Error())
	}
	rows, err := spec.Generate(opts...)
	if err != nil {
		return respond(400, err.Error())
	}
	if cap > 0 || format == "json" {
		return json200(rows)
	}
	var out strings.Builder
	switch format {
	case "jsonl":
		enc := json.NewEncoder(&out)
		for _, r := range rows {
			if err := enc.Encode(r); err != nil {
				return respond(500, err.Error())
			}
		}
	case "sql":
		webspec.WriteSQL(&out, name, spec.Columns(), rows)
	default:
		webspec.WriteCSV(&out, spec.Columns(), rows)
	}
	return respond(200, out.String())
}

// resolve builds the spec from a preset named in the query string or from a
// posted schema, matching the server's resolveSpec.
func resolve(method, body string, q url.Values, cap int) (*synth.YAMLSpec, []synth.Option, string, string, error) {
	name, format := q.Get("preset"), q.Get("format")

	var (
		spec *synth.YAMLSpec
		err  error
		rows int
	)
	if name != "" {
		text, ok := synth.PresetSpec(synth.Preset(name))
		if !ok {
			return nil, nil, "", "", fmt.Errorf("unknown preset %q", name)
		}
		if spec, err = synth.YAMLBytes([]byte(text)); err != nil {
			return nil, nil, "", "", err
		}
		rows, _ = strconv.Atoi(q.Get("rows"))
	} else {
		var req webspec.Request
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			return nil, nil, "", "", fmt.Errorf("cannot read the schema: %w", err)
		}
		doc, err := webspec.BuildYAML(&req)
		if err != nil {
			return nil, nil, "", "", err
		}
		if spec, err = synth.YAMLBytes(doc); err != nil {
			return nil, nil, "", "", err
		}
		rows, name, format = req.Count, req.Name, req.Format
		if q.Get("locale") == "" && req.Locale != "" {
			q.Set("locale", req.Locale)
		}
		if req.Seed != 0 {
			q.Set("seed", strconv.FormatUint(req.Seed, 10))
		}
	}

	if rows <= 0 {
		rows = 10
	}
	if cap > 0 && rows > cap {
		rows = cap
	}
	spec.SetCount(rows)

	var opts []synth.Option
	if l := q.Get("locale"); l != "" {
		opts = append(opts, synth.WithLocale(l))
	}
	if s := q.Get("seed"); s != "" {
		if seed, err := strconv.ParseUint(s, 10, 64); err == nil {
			opts = append(opts, synth.WithSeed(seed))
		}
	}
	if q.Get("unmasked") == "true" {
		opts = append(opts, synth.Unmasked())
	}
	if name == "" {
		name = "data"
	}
	return spec, opts, name, format, nil
}

type typeInfo struct {
	Kind      string   `json:"kind"`
	Category  string   `json:"category"`
	Localized bool     `json:"localized"`
	Locales   []string `json:"locales,omitempty"`
}

func types() []typeInfo {
	out := []typeInfo{}
	for _, k := range providers.Kinds() {
		if k == schema.KindObject || k == schema.KindArray || k == schema.KindUnknown {
			continue
		}
		locales := providers.LocalesFor(k)
		out = append(out, typeInfo{
			Kind:      string(k),
			Category:  webspec.CategoryOf(k),
			Localized: len(locales) > 0,
			Locales:   locales,
		})
	}
	return out
}

type presetInfo struct {
	Name string `json:"name"`
	YAML string `json:"yaml"`
}

func presets() []presetInfo {
	out := []presetInfo{}
	for _, p := range synth.Presets() {
		text, _ := synth.PresetSpec(p)
		out = append(out, presetInfo{Name: string(p), YAML: text})
	}
	return out
}

func json200(v any) any {
	body, err := json.Marshal(v)
	if err != nil {
		return respond(500, err.Error())
	}
	return respond(200, string(body))
}

func respond(status int, body string) any {
	return map[string]any{"status": status, "body": body}
}
