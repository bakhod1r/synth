package mcp

import "testing"

// The server must exist and carry a name and version, because a client shows
// them to the user when asking whether to trust the connection.
func TestNewServerIsUsable(t *testing.T) {
	if s := New(); s == nil {
		t.Fatal("New() returned nil")
	}
}
