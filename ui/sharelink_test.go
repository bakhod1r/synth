package ui_test

import (
	"strings"
	"testing"
)

// The share-link feature is JS, but the wiring must be served: the button in
// the page, the encode/decode functions and the fragment format in the script.
// If any goes missing the feature is silently dead, and a Go test is the only
// guard the build has over the static assets.
func TestShareLinkWiringIsServed(t *testing.T) {
	html := get(t, "/").Body.String()
	if !strings.Contains(html, `id="share"`) {
		t.Error("index.html is missing the Share button")
	}

	js := get(t, "/app.js").Body.String()
	for _, want := range []string{"function encodeSpec", "function decodeSpec",
		"function loadSharedSpec", "readSharedSpec", "#s="} {
		if !strings.Contains(js, want) {
			t.Errorf("app.js is missing %q", want)
		}
	}
}

// The fragment must never leave the browser for the server, or the "touches
// nothing" promise breaks. Assert the link is built from location.pathname with
// a hash, not posted anywhere.
func TestShareLinkUsesFragmentNotRequest(t *testing.T) {
	js := get(t, "/app.js").Body.String()
	if !strings.Contains(js, "location.pathname + '#s='") {
		t.Error("the share link should be a fragment on the current page")
	}
}
