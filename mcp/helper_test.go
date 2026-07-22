package mcp

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// listTools returns the registered tools by name, so a test can assert on what
// a client actually sees rather than on what New() was written to do.
func listTools(t *testing.T) map[string]mcp.Tool {
	t.Helper()
	out := map[string]mcp.Tool{}
	for name, st := range New().ListTools() {
		out[name] = st.Tool
	}
	if len(out) == 0 {
		t.Fatal("the server registered no tools")
	}
	return out
}

// resultText pulls the text out of a tool result, so a test can check what the
// caller actually receives rather than what the handler returned.
func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("the result carries no content")
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("the result content is %T, want text", res.Content[0])
	}
	return text.Text
}
