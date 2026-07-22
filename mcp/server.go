// Package mcp exposes Synth to MCP clients.
//
// # Boundary
//
// The server speaks MCP over stdio. It opens no socket, makes no outbound
// request, reads no file and writes no file. Every tool takes its input as an
// argument and returns its result in the response; where the data is saved is
// the calling agent's decision, not Synth's.
//
// This matters more here than elsewhere in the project. An MCP server acts with
// the user's own permissions on behalf of a model that may be reading
// attacker-controlled text. A tool that accepted a file path would turn a data
// generator into a file-reading primitive for anyone who can get text in front
// of that model. Refusing paths outright is the only version of that boundary
// that cannot be got wrong — and boundary_test.go enforces it, because a
// package comment does not stop a later well-meaning change.
package mcp

import "github.com/mark3labs/mcp-go/server"

// Version is reported to the client during initialization. A client shows it to
// the user when asking whether to trust the connection.
const Version = "0.1.0"

// New returns the server with every tool registered.
func New() *server.MCPServer {
	s := server.NewMCPServer("synth", Version)
	return s
}
