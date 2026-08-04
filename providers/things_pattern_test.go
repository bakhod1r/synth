package providers

import (
	"strings"
	"testing"

	"github.com/bakhod1r/synth/internal/rng"
)

// articleTitle substitutes {token} placeholders until none are left. A pattern
// whose brace is never closed would otherwise loop forever, so it stops and
// emits what it has. No shipped pattern is malformed — this substitutes one to
// prove the guard holds, because the failure it prevents is a hang, not a bad
// value.
func TestArticleTitleStopsOnAnUnclosedPlaceholder(t *testing.T) {
	saved := titlePatterns
	t.Cleanup(func() { titlePatterns = saved })
	titlePatterns = []string{"Breaking: {adj news"}

	got := articleTitle(Ctx{Rand: rng.New(1)})
	s, ok := got.(string)
	if !ok {
		t.Fatalf("articleTitle returned %T, want a string", got)
	}
	if !strings.Contains(s, "{adj news") {
		t.Errorf("articleTitle = %q, want the unclosed placeholder left as written", s)
	}
}
