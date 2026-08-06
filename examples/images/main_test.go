package main

import "testing"

// TestMainSmoke runs the example end to end, so the demo cannot rot silently
// and the package reports coverage.
func TestMainSmoke(t *testing.T) {
	main()
}
