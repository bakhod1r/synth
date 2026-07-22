package synth

import (
	"fmt"
	"reflect"
	"time"

	"github.com/bakhod1r/synth/reflectfe"
	"github.com/bakhod1r/synth/schema"
	"github.com/bakhod1r/synth/snapshot"
)

// SnapshotConfig describes a table's life across time.
type SnapshotConfig = snapshot.Config

// Timeline answers what a table looked like at any instant, and what changed
// between two of them.
type Timeline = snapshot.Timeline

// Snapshot builds a timeline for type T. Ask it for the table as of any
// instant, or for the change events between two — applying the events to the
// earlier snapshot reproduces the later one exactly.
//
//	tl, _ := synth.Snapshot[Order](synth.SnapshotConfig{Rows: 10_000, Churn: 2})
//	jan := tl.At(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
//	events := tl.Between(jan1, jul1)
func Snapshot[T any](cfg SnapshotConfig) (*Timeline, error) {
	var zero T
	rt := reflect.TypeOf(zero)
	if rt == nil || rt.Kind() != reflect.Struct {
		return nil, errNotStruct(zero)
	}
	cached, _ := reflectfe.Build(rt)
	s := &schema.Schema{Fields: append([]schema.Field(nil), cached.Fields...)}
	return snapshot.New(s, cfg)
}

// Snapshot builds a timeline from a YAML spec rather than a Go type, so the
// CLI can travel through time without compiled structs.
func (y *YAMLSpec) Snapshot(cfg SnapshotConfig) (*Timeline, error) {
	if cfg.Rows == 0 {
		cfg.Rows = y.spec.Count
	}
	if cfg.Table == "" {
		cfg.Table = y.spec.Name
	}
	if cfg.Locale == "" {
		cfg.Locale = y.spec.Locale
	}
	if cfg.Seed == 0 {
		cfg.Seed = y.spec.Seed
	}
	s := &schema.Schema{Fields: append([]schema.Field(nil), y.spec.Schema.Fields...)}
	return snapshot.New(s, cfg)
}

// ParseInstant reads a date or timestamp the way the CLI accepts it.
func ParseInstant(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("synth: %q is not a date or timestamp "+
		"(want 2026-01-01 or 2026-01-01T00:00:00Z)", s)
}
