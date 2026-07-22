// Command synth-mcp serves Synth over MCP on stdio.
//
// It is started by an MCP client, not by hand. See mcp/README.md for the client
// configuration.
package main

import (
	"fmt"
	"os"

	"github.com/bakhod1r/synth/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	if err := server.ServeStdio(mcp.New()); err != nil {
		fmt.Fprintln(os.Stderr, "synth-mcp:", err)
		os.Exit(1)
	}
}
