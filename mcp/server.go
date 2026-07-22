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

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Version is reported to the client during initialization. A client shows it to
// the user when asking whether to trust the connection.
const Version = "0.1.0"

// New returns the server with every tool registered.
func New() *server.MCPServer {
	s := server.NewMCPServer("synth", Version)

	s.AddTool(mcp.NewTool("generate",
		mcp.WithDescription("Generate synthetic records from a built-in preset or a YAML spec. "+
			"Returns the rows in the response — it writes no file and touches no database. "+
			"Card numbers and national identifiers come back masked unless unmasked=true."),
		mcp.WithString("preset", mcp.Description(
			"A built-in schema name; call list_presets. Use this or spec, not both.")),
		mcp.WithString("spec", mcp.Description(
			"A YAML schema. Use this or preset, not both.")),
		mcp.WithString("locale", mcp.Description(
			"Data locale, e.g. uz_UZ or de_DE. Default en_US.")),
		mcp.WithNumber("rows", mcp.Description(
			"How many rows. Default 10, maximum 1000.")),
		mcp.WithNumber("seed", mcp.Description(
			"Seed. The same seed always gives the same rows.")),
		mcp.WithBoolean("unmasked", mcp.Description(
			"Return raw card numbers and identifiers. Off by default.")),
	), typed(handleGenerate))

	s.AddTool(mcp.NewTool("list_types",
		mcp.WithDescription("List the generatable column types, optionally filtered by substring. "+
			"Each entry says whether its values follow the locale."),
		mcp.WithString("search", mcp.Description(
			`Substring filter, e.g. "card" or "name".`)),
	), typed(handleListTypes))

	s.AddTool(mcp.NewTool("list_presets",
		mcp.WithDescription("List the built-in schemas with their YAML, as a starting point to edit."),
	), nullary(handleListPresets))

	s.AddTool(mcp.NewTool("verify",
		mcp.WithDescription("Check a dataset for broken checksums (Luhn, IBAN, EAN, UPC), "+
			"malformed emails, URLs, UUIDs and IPs, and time anomalies. Pass the rows "+
			"themselves — this tool reads no files."),
		mcp.WithString("data", mcp.Required(), mcp.Description(
			"The dataset itself, as text. Not a path.")),
		mcp.WithString("format", mcp.Description("csv (default) or jsonl.")),
	), typed(handleVerify))

	return s
}

// typed adapts a handler taking a typed argument struct to the MCP signature.
// Keeping the handlers plain functions means each is testable without building
// a protocol message.
func typed[T any](h func(T) (any, error)) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args T
		if err := req.BindArguments(&args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("bad arguments: %v", err)), nil
		}
		return result(h(args))
	}
}

// nullary adapts a handler that takes no arguments.
func nullary(h func() (any, error)) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return result(h())
	}
}

// result turns a handler's return into a tool result.
//
// A tool error is reported as a result rather than as a transport error, so the
// model reads the message and corrects the call. Returning it as a transport
// error instead makes the client drop the connection over a mistyped argument.
func result(v any, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out, err := mcp.NewToolResultJSON(v)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot encode the result: %v", err)), nil
	}
	return out, nil
}
