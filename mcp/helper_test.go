package mcp

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

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
