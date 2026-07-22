package gen

import (
	"testing"

	"github.com/bakhod1r/synth/schema"
)

func TestClampLen(t *testing.T) {
	tests := []struct {
		name   string
		maxlen string
		in     any
		want   any
	}{
		{"no limit", "", "Aleksandr", "Aleksandr"},
		{"under limit", "20", "Aleksandr", "Aleksandr"},
		{"exact limit", "9", "Aleksandr", "Aleksandr"},
		{"over limit", "4", "Aleksandr", "Alek"},
		// Counted in runes, not bytes: a Cyrillic name cut at byte 4 would
		// split a character and produce invalid UTF-8.
		{"multi-byte", "4", "Александр", "Алек"},
		{"nil passes through", "4", nil, nil},
		{"non-string passes through", "4", 123456, 123456},
		{"zero means no limit", "0", "Aleksandr", "Aleksandr"},
		{"garbage means no limit", "abc", "Aleksandr", "Aleksandr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &schema.Field{Params: map[string]string{}}
			if tt.maxlen != "" {
				f.Params["maxlen"] = tt.maxlen
			}
			if got := clampLen(f, tt.in); got != tt.want {
				t.Errorf("clampLen(%q, %v) = %v, want %v", tt.maxlen, tt.in, got, tt.want)
			}
		})
	}
}

func TestClampLenValidUTF8(t *testing.T) {
	f := &schema.Field{Params: map[string]string{"maxlen": "3"}}
	got := clampLen(f, "日本語テスト").(string)
	if got != "日本語" {
		t.Fatalf("got %q, want %q", got, "日本語")
	}
}
