package synth

import (
	"time"

	"github.com/bakhodir/synth/gen"
	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/openapi"
)

// APISpec wraps a parsed OpenAPI spec for payload generation.
type APISpec struct {
	spec *openapi.Spec
}

// OpenAPI loads an OpenAPI 3 spec (YAML or JSON) from a file.
func OpenAPI(path string) (*APISpec, error) {
	s, err := openapi.Load(path)
	if err != nil {
		return nil, err
	}
	return &APISpec{spec: s}, nil
}

// OpenAPIBytes parses an OpenAPI 3 spec from bytes.
func OpenAPIBytes(data []byte) (*APISpec, error) {
	s, err := openapi.Parse(data)
	if err != nil {
		return nil, err
	}
	return &APISpec{spec: s}, nil
}

// Payload generates one valid request-body payload (as a field→value map) for
// the given method and path.
func (a *APISpec) Payload(method, path string, opts ...Option) (map[string]any, error) {
	recs, err := a.Payloads(method, path, 1, opts...)
	if err != nil {
		return nil, err
	}
	return recs[0], nil
}

// Payloads generates n valid request-body payloads for method+path.
func (a *APISpec) Payloads(method, path string, n int, opts ...Option) ([]map[string]any, error) {
	sc, err := a.spec.Schema(method, path)
	if err != nil {
		return nil, err
	}
	cfg := config{seed: uint64(time.Now().UnixNano()), locale: "en_US"}
	for _, o := range opts {
		o(&cfg)
	}
	applyWeighted(sc, cfg.weighted)
	eng, err := gen.Compile(sc, cfg.locale)
	if err != nil {
		return nil, err
	}
	base := rng.New(cfg.seed)
	out := make([]map[string]any, n)
	for i := 0; i < n; i++ {
		out[i] = eng.Record(base, i)
	}
	return out, nil
}

// PayloadJSON generates one payload and marshals it to indented JSON.
func (a *APISpec) PayloadJSON(method, path string, opts ...Option) ([]byte, error) {
	rec, err := a.Payload(method, path, opts...)
	if err != nil {
		return nil, err
	}
	return openapi.PayloadJSON(rec)
}
