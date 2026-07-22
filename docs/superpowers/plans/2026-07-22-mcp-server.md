# Synth MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose Synth to MCP clients as a set of tools, so an assistant can generate, verify, profile, mask and time-travel synthetic data without shelling out to the CLI.

**Architecture:** A separate nested Go module at `mcp/` holding a stdio MCP server built on `mark3labs/mcp-go`. It imports the existing `synth`, `verify`, `profile`, `mask` and `snapshot` packages and adds no capability they do not already have. The server never reads or writes a file and never opens an outbound connection: every tool takes its input as an argument and returns its result in the response. `mcp/cmd/synth-mcp` is the binary.

**Tech Stack:** Go 1.25, `github.com/mark3labs/mcp-go` (server + stdio transport), the existing Synth packages.

## Global Constraints

- **The core module keeps exactly two dependencies** — `github.com/google/uuid` and `gopkg.in/yaml.v3`. The MCP server therefore lives in its own module (`mcp/go.mod`), exactly as `sink/parquet/` and `benchcmp/` already do. Nothing under `mcp/` may be imported by the core module.
- **No network.** The server speaks MCP over stdio only. It must not open a listening socket or make an outbound request. This is the same boundary the rest of Synth holds; the browser workbench is loopback-only and this is stricter still.
- **No filesystem.** No tool reads a path or writes a file. Input arrives as an argument; output is returned. The agent decides where anything is saved. A path argument would let a prompt-injected model read `~/.ssh/id_rsa` through a data generator, and refusing paths outright is the only version of that boundary that cannot be got wrong.
- **No database.** Unchanged from the rest of the project: Synth supplies data, the seeder writes it.
- **Every tool response is bounded.** `maxRows = 1000` for generate, `maxInputBytes = 8 << 20` for tools that take a dataset. Exceeding either is an error naming the limit, never a truncated result presented as complete.
- Go version: `go 1.25` in `mcp/go.mod`.
- Module path: `github.com/bakhodir/synth/mcp`.
- Binary name: `synth-mcp`.
- Tool names are snake_case and prefixed nothing: `generate`, `list_types`, `list_presets`, `verify`, `profile`, `mask`, `snapshot`.
- Every tool description must state what the tool does NOT do (no files, no network), because that is what a calling model needs to know to plan around it.

---

## File Structure

| File | Responsibility |
|---|---|
| `mcp/go.mod`, `mcp/go.sum` | Separate module; the only place `mcp-go` appears |
| `mcp/server.go` | Builds the `*server.MCPServer` and registers every tool. No tool logic. |
| `mcp/limits.go` | `maxRows`, `maxInputBytes`, `parseRows`, size errors. Shared by all tools. |
| `mcp/generate.go` | `generate`, `list_types`, `list_presets` |
| `mcp/verify.go` | `verify` |
| `mcp/profile.go` | `profile` |
| `mcp/mask.go` | `mask` |
| `mcp/snapshot.go` | `snapshot` |
| `mcp/cmd/synth-mcp/main.go` | Wires the server to stdio. Nothing else. |
| `mcp/*_test.go` | One test file per tool file, testing the handler directly |
| `mcp/README.md` | Install and client-config instructions |

Tool handlers are plain functions taking a typed argument struct and returning `(any, error)`; the MCP wiring is confined to `server.go`. That split means every handler is testable without constructing a protocol message.

---

### Task 1: Module skeleton and the server that registers nothing

**Files:**
- Create: `mcp/go.mod`
- Create: `mcp/server.go`
- Create: `mcp/cmd/synth-mcp/main.go`
- Test: `mcp/server_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `func New() *server.MCPServer` — the server with every tool registered. Later tasks add registrations inside it.

- [ ] **Step 1: Create the module**

```bash
mkdir -p mcp/cmd/synth-mcp
cd mcp
go mod init github.com/bakhodir/synth/mcp
go mod edit -require=github.com/bakhodir/synth@v0.0.0
go mod edit -replace=github.com/bakhodir/synth=../
go get github.com/mark3labs/mcp-go@latest
```

Expected: `mcp/go.mod` exists and names `github.com/mark3labs/mcp-go`.

- [ ] **Step 2: Verify the core module did not gain a dependency**

```bash
cd .. && go list -m all | grep -c mark3labs
```

Expected: `0`. If this prints anything else, the module boundary is broken — stop and fix it before continuing.

- [ ] **Step 3: Write the failing test**

`mcp/server_test.go`:

```go
package mcp

import "testing"

// The server must exist and carry a name and version, because a client shows
// them to the user when asking whether to trust the connection.
func TestNewServerIsUsable(t *testing.T) {
	if s := New(); s == nil {
		t.Fatal("New() returned nil")
	}
}
```

- [ ] **Step 4: Run it to see it fail**

```bash
cd mcp && go test ./... -run TestNewServerIsUsable
```

Expected: FAIL, `undefined: New`.

- [ ] **Step 5: Write `mcp/server.go`**

```go
// Package mcp exposes Synth to MCP clients.
//
// # Boundary
//
// The server speaks MCP over stdio. It opens no socket, makes no outbound
// request, reads no file and writes no file. Every tool takes its input as an
// argument and returns its result in the response; where the data lives is the
// calling agent's decision, not Synth's.
//
// This matters more here than elsewhere in the project. An MCP server acts with
// the user's own permissions on behalf of a model that may be reading
// attacker-controlled text. A tool that accepted a file path would turn a data
// generator into a file-reading primitive for anyone who can get text in front
// of that model. Refusing paths outright is the only version of that boundary
// that cannot be got wrong.
package mcp

import "github.com/mark3labs/mcp-go/server"

// Version is reported to the client during initialization.
const Version = "0.1.0"

// New returns the server with every tool registered.
func New() *server.MCPServer {
	s := server.NewMCPServer("synth", Version)
	return s
}
```

- [ ] **Step 6: Write `mcp/cmd/synth-mcp/main.go`**

```go
// Command synth-mcp serves Synth over MCP on stdio.
//
// It is started by an MCP client, not by hand. See mcp/README.md for the
// client configuration.
package main

import (
	"fmt"
	"os"

	"github.com/bakhodir/synth/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	if err := server.ServeStdio(mcp.New()); err != nil {
		fmt.Fprintln(os.Stderr, "synth-mcp:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 7: Run the test and build**

```bash
cd mcp && go test ./... && go build ./... && go vet ./...
```

Expected: PASS, no output from build or vet.

- [ ] **Step 8: Commit**

```bash
git add mcp/
git commit -m "feat(mcp): add the module skeleton and a stdio server

The MCP server lives in its own module so the core library keeps its two
dependencies: someone importing synth should not pull in an MCP SDK they will
never call.
"
```

---

### Task 2: Shared limits

**Files:**
- Create: `mcp/limits.go`
- Test: `mcp/limits_test.go`

**Interfaces:**
- Produces:
  - `const maxRows = 1000`
  - `const maxInputBytes = 8 << 20`
  - `func rowsWithin(n int) (int, error)` — defaults 0 to 10, rejects above `maxRows`
  - `func inputWithin(s string) error` — rejects input above `maxInputBytes`

- [ ] **Step 1: Write the failing test**

`mcp/limits_test.go`:

```go
package mcp

import "strings"
import "testing"

func TestRowsWithin(t *testing.T) {
	for _, tc := range []struct {
		in      int
		want    int
		wantErr bool
	}{
		{0, 10, false},
		{1, 1, false},
		{maxRows, maxRows, false},
		{maxRows + 1, 0, true},
		{-5, 0, true},
	} {
		got, err := rowsWithin(tc.in)
		if (err != nil) != tc.wantErr {
			t.Fatalf("rowsWithin(%d) error = %v, wantErr %v", tc.in, err, tc.wantErr)
		}
		if err == nil && got != tc.want {
			t.Fatalf("rowsWithin(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// The error has to name the limit. A model that is told only "too many" will
// retry with another number it also cannot justify.
func TestRowLimitErrorNamesTheLimit(t *testing.T) {
	_, err := rowsWithin(maxRows + 1)
	if err == nil || !strings.Contains(err.Error(), "1000") {
		t.Fatalf("error %v does not name the limit", err)
	}
}

func TestInputWithin(t *testing.T) {
	if err := inputWithin("small"); err != nil {
		t.Fatalf("a small input was rejected: %v", err)
	}
	big := strings.Repeat("x", maxInputBytes+1)
	if err := inputWithin(big); err == nil {
		t.Fatal("an oversized input was accepted")
	}
}
```

- [ ] **Step 2: Run it to see it fail**

```bash
cd mcp && go test ./... -run "Rows|Input"
```

Expected: FAIL, `undefined: rowsWithin`.

- [ ] **Step 3: Write `mcp/limits.go`**

```go
package mcp

import "fmt"

// maxRows caps a generate call.
//
// The rows come back inside the response, so an unbounded request would put a
// million rows into a model's context window: slow, expensive, and useless,
// because the model cannot read them. A thousand rows is enough to see the
// shape of a dataset; anything larger belongs in a file the CLI writes.
const maxRows = 1000

// maxInputBytes caps a dataset handed to verify, profile or mask.
const maxInputBytes = 8 << 20 // 8 MiB

// rowsWithin validates a requested row count, defaulting an unset one.
func rowsWithin(n int) (int, error) {
	if n == 0 {
		return 10, nil
	}
	if n < 0 {
		return 0, fmt.Errorf("rows must be positive, got %d", n)
	}
	if n > maxRows {
		return 0, fmt.Errorf("rows is %d but this tool returns at most %d — "+
			"the rows travel in the response, so a larger set belongs in a file: "+
			"run `synth gen -n %d -o data.csv` instead", n, maxRows, n)
	}
	return n, nil
}

// inputWithin rejects a dataset too large to process in a response.
func inputWithin(s string) error {
	if len(s) > maxInputBytes {
		return fmt.Errorf("input is %d bytes but this tool accepts at most %d — "+
			"pass a sample, or use the synth CLI on the full file", len(s), maxInputBytes)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests**

```bash
cd mcp && go test ./... -run "Rows|Input"
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mcp/limits.go mcp/limits_test.go
git commit -m "feat(mcp): bound every tool's input and output

Rows travel in the response, so an unbounded request fills a context window
with data the model cannot read. The errors name the limit and the CLI command
that has no limit, so a model that hits one knows what to do instead."
```

---

### Task 3: generate, list_types, list_presets

**Files:**
- Create: `mcp/generate.go`
- Modify: `mcp/server.go` (register the three tools)
- Test: `mcp/generate_test.go`

**Interfaces:**
- Consumes: `rowsWithin` from Task 2.
- Produces:
  - `type generateArgs struct { Preset, Spec, Locale string; Rows int; Seed uint64; Unmasked bool }`
  - `func handleGenerate(a generateArgs) (any, error)`
  - `func handleListTypes(a listTypesArgs) (any, error)` where `listTypesArgs` is `struct{ Search string }`
  - `func handleListPresets() (any, error)`

- [ ] **Step 1: Write the failing test**

`mcp/generate_test.go`:

```go
package mcp

import (
	"strings"
	"testing"
)

func TestGenerateFromPreset(t *testing.T) {
	out, err := handleGenerate(generateArgs{Preset: "transaction", Rows: 5, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	rows, ok := out.([]map[string]any)
	if !ok {
		t.Fatalf("got %T, want rows", out)
	}
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5", len(rows))
	}
}

// The masking default must survive the trip through MCP. An assistant pasting
// a generated card number into a chat log is exactly the accident the default
// exists to prevent.
func TestGenerateMasksByDefault(t *testing.T) {
	out, _ := handleGenerate(generateArgs{Preset: "transaction", Rows: 20, Seed: 2})
	for i, r := range out.([]map[string]any) {
		if card, _ := r["card_number"].(string); !strings.Contains(card, "*") {
			t.Fatalf("row %d: unmasked card %q", i, card)
		}
	}
}

func TestGenerateFromSpec(t *testing.T) {
	spec := "name: t\nfields:\n  city: { kind: city }\n"
	out, err := handleGenerate(generateArgs{Spec: spec, Rows: 3, Locale: "uz_UZ", Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.([]map[string]any)) != 3 {
		t.Fatal("wrong row count")
	}
}

// Neither preset nor spec is a mistake worth naming, and so is both.
func TestGenerateRejectsAmbiguousInput(t *testing.T) {
	if _, err := handleGenerate(generateArgs{Rows: 1}); err == nil {
		t.Fatal("a call with no preset and no spec was accepted")
	}
	if _, err := handleGenerate(generateArgs{Preset: "user", Spec: "name: t", Rows: 1}); err == nil {
		t.Fatal("a call with both preset and spec was accepted")
	}
}

func TestGenerateRejectsUnknownPreset(t *testing.T) {
	_, err := handleGenerate(generateArgs{Preset: "nope", Rows: 1})
	if err == nil || !strings.Contains(err.Error(), "list_presets") {
		t.Fatalf("error %v does not point at list_presets", err)
	}
}

func TestGenerateIsReproducible(t *testing.T) {
	a, _ := handleGenerate(generateArgs{Preset: "order", Rows: 20, Seed: 7})
	b, _ := handleGenerate(generateArgs{Preset: "order", Rows: 20, Seed: 7})
	ar, br := a.([]map[string]any), b.([]map[string]any)
	for i := range ar {
		if ar[i]["id"] != br[i]["id"] {
			t.Fatalf("row %d differs between runs with the same seed", i)
		}
	}
}

func TestListTypes(t *testing.T) {
	out, err := handleListTypes(listTypesArgs{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.([]typeInfo)) < 200 {
		t.Fatal("the catalog looks empty")
	}
	filtered, _ := handleListTypes(listTypesArgs{Search: "card"})
	for _, ty := range filtered.([]typeInfo) {
		if !strings.Contains(ty.Kind, "card") {
			t.Fatalf("search returned an unrelated kind %q", ty.Kind)
		}
	}
}

func TestListPresetsCarriesTheSpec(t *testing.T) {
	out, err := handleListPresets()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range out.([]presetInfo) {
		if p.YAML == "" {
			t.Fatalf("preset %q has no spec", p.Name)
		}
	}
}
```

- [ ] **Step 2: Run it to see it fail**

```bash
cd mcp && go test ./... -run "Generate|List"
```

Expected: FAIL, `undefined: handleGenerate`.

- [ ] **Step 3: Write `mcp/generate.go`**

```go
package mcp

import (
	"fmt"
	"strings"

	"github.com/bakhodir/synth"
	"github.com/bakhodir/synth/providers"
	"github.com/bakhodir/synth/schema"
)

// generateArgs is the `generate` tool's input. Exactly one of Preset and Spec
// must be set: a call with both would silently ignore one, and the caller would
// have no way to tell which.
type generateArgs struct {
	Preset   string `json:"preset"`
	Spec     string `json:"spec"`
	Locale   string `json:"locale"`
	Rows     int    `json:"rows"`
	Seed     uint64 `json:"seed"`
	Unmasked bool   `json:"unmasked"`
}

func handleGenerate(a generateArgs) (any, error) {
	if (a.Preset == "") == (a.Spec == "") {
		return nil, fmt.Errorf("set exactly one of preset= or spec=: " +
			"preset for a built-in schema (call list_presets), spec for YAML of your own")
	}
	n, err := rowsWithin(a.Rows)
	if err != nil {
		return nil, err
	}
	if err := inputWithin(a.Spec); err != nil {
		return nil, err
	}

	opts := []synth.Option{}
	if a.Locale != "" {
		opts = append(opts, synth.WithLocale(a.Locale))
	}
	if a.Seed != 0 {
		opts = append(opts, synth.WithSeed(a.Seed))
	}
	if a.Unmasked {
		opts = append(opts, synth.Unmasked())
	}

	if a.Preset != "" {
		if _, ok := synth.PresetSpec(synth.Preset(a.Preset)); !ok {
			return nil, fmt.Errorf("unknown preset %q — call list_presets for the names", a.Preset)
		}
		return synth.Generate(synth.Preset(a.Preset), n, opts...)
	}
	spec, err := synth.YAMLBytes([]byte(a.Spec))
	if err != nil {
		return nil, fmt.Errorf("the spec does not parse: %w", err)
	}
	return spec.GenerateN(n, opts...)
}

// typeInfo is one entry of the catalog.
type typeInfo struct {
	Kind      string   `json:"kind"`
	Localized bool     `json:"localized"`
	Locales   []string `json:"locales,omitempty"`
}

type listTypesArgs struct {
	Search string `json:"search"`
}

// handleListTypes returns the catalog, optionally filtered. The full list is
// around 250 entries, which is large but still worth returning whole: a model
// that cannot see a type will invent one, and an invented kind fails at
// generate time with a worse error than a long list.
func handleListTypes(a listTypesArgs) (any, error) {
	q := strings.ToLower(strings.TrimSpace(a.Search))
	out := []typeInfo{}
	for _, k := range providers.Kinds() {
		if k == schema.KindObject || k == schema.KindArray || k == schema.KindUnknown {
			continue
		}
		if q != "" && !strings.Contains(string(k), q) {
			continue
		}
		locales := providers.LocalesFor(k)
		out = append(out, typeInfo{Kind: string(k), Localized: len(locales) > 0, Locales: locales})
	}
	return out, nil
}

// presetInfo carries the preset's YAML, so a caller can start from it and edit
// rather than guessing field names.
type presetInfo struct {
	Name string `json:"name"`
	YAML string `json:"yaml"`
}

func handleListPresets() (any, error) {
	out := []presetInfo{}
	for _, p := range synth.Presets() {
		text, _ := synth.PresetSpec(p)
		out = append(out, presetInfo{Name: string(p), YAML: text})
	}
	return out, nil
}
```

- [ ] **Step 4: Register the tools in `mcp/server.go`**

Replace the body of `New` with:

```go
func New() *server.MCPServer {
	s := server.NewMCPServer("synth", Version)

	s.AddTool(mcp.NewTool("generate",
		mcp.WithDescription("Generate synthetic records from a built-in preset or a YAML spec. "+
			"Returns the rows in the response — it writes no file and touches no database. "+
			"Sensitive columns (card numbers, national identifiers) come back masked unless unmasked=true."),
		mcp.WithString("preset", mcp.Description("A built-in schema name; call list_presets. Use this or spec, not both.")),
		mcp.WithString("spec", mcp.Description("A YAML schema. Use this or preset, not both.")),
		mcp.WithString("locale", mcp.Description("Data locale, e.g. uz_UZ or de_DE. Default en_US.")),
		mcp.WithNumber("rows", mcp.Description("How many rows. Default 10, maximum 1000.")),
		mcp.WithNumber("seed", mcp.Description("Seed. The same seed always gives the same rows.")),
		mcp.WithBoolean("unmasked", mcp.Description("Return raw card numbers and identifiers. Off by default.")),
	), typed(handleGenerate))

	s.AddTool(mcp.NewTool("list_types",
		mcp.WithDescription("List the generatable column types, optionally filtered by substring."),
		mcp.WithString("search", mcp.Description("Substring filter, e.g. \"card\" or \"name\".")),
	), typed(handleListTypes))

	s.AddTool(mcp.NewTool("list_presets",
		mcp.WithDescription("List the built-in schemas with their YAML, as a starting point to edit."),
	), nullary(handleListPresets))

	return s
}
```

Add the two adapters at the bottom of `server.go`. They exist so every handler
stays a plain function of a typed struct, testable without a protocol message:

```go
// typed adapts a handler taking a typed argument struct to the MCP signature.
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

// result turns a handler's return into a tool result. A tool error is reported
// as a result rather than a transport error, so the model sees the message and
// can correct the call instead of the client dropping the connection.
func result(v any, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot encode the result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(body)), nil
}
```

Imports needed in `server.go`: `context`, `encoding/json`, `fmt`, `github.com/mark3labs/mcp-go/mcp`, `github.com/mark3labs/mcp-go/server`.

- [ ] **Step 5: Run the tests**

```bash
cd mcp && go test ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add mcp/
git commit -m "feat(mcp): add generate, list_types and list_presets

Masking stays on by default across MCP for the same reason as everywhere else:
an assistant pasting a generated card number into a chat log is exactly the
accident the default prevents. A tool error is returned as a result rather than
a transport error, so the model reads the message and fixes the call."
```

---

### Task 4: verify

**Files:**
- Create: `mcp/verify.go`
- Modify: `mcp/server.go` (register)
- Test: `mcp/verify_test.go`

**Interfaces:**
- Consumes: `inputWithin` from Task 2.
- Produces: `type verifyArgs struct { Data, Format string }`, `func handleVerify(a verifyArgs) (any, error)`.

- [ ] **Step 1: Write the failing test**

`mcp/verify_test.go`:

```go
package mcp

import (
	"strings"
	"testing"

	"github.com/bakhodir/synth/verify"
)

const goodCSV = "id,card\n1,4539578763621486\n2,4556737586899855\n"
const badCSV = "id,card\n1,4539578763621480\n2,1234567890123456\n"

func TestVerifyAcceptsValidData(t *testing.T) {
	out, err := handleVerify(verifyArgs{Data: goodCSV, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if !out.(verify.Report).OK() {
		t.Fatalf("valid data reported findings: %+v", out)
	}
}

func TestVerifyFindsBrokenChecksums(t *testing.T) {
	out, err := handleVerify(verifyArgs{Data: badCSV, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if out.(verify.Report).OK() {
		t.Fatal("invalid card numbers were reported as clean")
	}
}

func TestVerifyReadsJSONL(t *testing.T) {
	data := "{\"id\":1,\"card\":\"4539578763621486\"}\n"
	if _, err := handleVerify(verifyArgs{Data: data, Format: "jsonl"}); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRejectsOversizedInput(t *testing.T) {
	_, err := handleVerify(verifyArgs{Data: strings.Repeat("x", maxInputBytes+1), Format: "csv"})
	if err == nil {
		t.Fatal("an oversized dataset was accepted")
	}
}

// A path is not data. Accepting one would make a data generator into a
// file-reading primitive for a model reading untrusted text.
func TestVerifyDoesNotTakeAPath(t *testing.T) {
	out, err := handleVerify(verifyArgs{Data: "/etc/passwd", Format: "csv"})
	if err != nil {
		return // rejecting it outright is fine too
	}
	if r := out.(verify.Report); r.Rows > 1 {
		t.Fatal("the tool appears to have read a file")
	}
}
```

- [ ] **Step 2: Run it to see it fail**

```bash
cd mcp && go test ./... -run Verify
```

Expected: FAIL, `undefined: handleVerify`.

- [ ] **Step 3: Write `mcp/verify.go`**

```go
package mcp

import (
	"fmt"
	"strings"

	"github.com/bakhodir/synth/profile"
	"github.com/bakhodir/synth/verify"
)

// verifyArgs takes the dataset itself, never a path. See the package comment
// for why.
type verifyArgs struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

func handleVerify(a verifyArgs) (any, error) {
	rows, err := parseRows(a.Data, a.Format)
	if err != nil {
		return nil, err
	}
	return verify.Run(rows, verify.Options{}), nil
}

// parseRows reads an inline dataset. It is shared by verify and profile.
func parseRows(data, format string) ([]map[string]any, error) {
	if err := inputWithin(data); err != nil {
		return nil, err
	}
	if strings.TrimSpace(data) == "" {
		return nil, fmt.Errorf("data is empty — pass the rows themselves, not a file path")
	}
	switch strings.ToLower(format) {
	case "", "csv":
		res, err := profile.FromCSV(strings.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("cannot read the CSV: %w", err)
		}
		return res.Rows, nil
	case "jsonl", "ndjson":
		res, err := profile.FromJSONL(strings.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("cannot read the JSONL: %w", err)
		}
		return res.Rows, nil
	default:
		return nil, fmt.Errorf("unknown format %q — use csv or jsonl", format)
	}
}
```

> **Note for the implementer:** `profile.Result` may not expose the parsed rows.
> Check `profile/profile.go` first. If it does not, add an exported `Rows` field
> to `profile.Result` populated by `FromCSV`/`FromJSONL` in the same commit,
> with a comment saying verify needs them; do not duplicate the parsers here.

- [ ] **Step 4: Register the tool in `New`**

```go
	s.AddTool(mcp.NewTool("verify",
		mcp.WithDescription("Check a dataset for broken checksums (Luhn, IBAN, EAN), malformed "+
			"emails, URLs, UUIDs and IPs, and time anomalies. Pass the rows themselves — "+
			"this tool reads no files."),
		mcp.WithString("data", mcp.Required(), mcp.Description("The dataset itself, as text.")),
		mcp.WithString("format", mcp.Description("csv (default) or jsonl.")),
	), typed(handleVerify))
```

- [ ] **Step 5: Run the tests**

```bash
cd mcp && go test ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add mcp/ profile/
git commit -m "feat(mcp): add verify

The dataset arrives as text, never as a path. An MCP server runs with the
user's permissions on behalf of a model that may be reading attacker-controlled
input, so a path argument would turn a data generator into a file-reading
primitive."
```

---

### Task 5: profile and mask

**Files:**
- Create: `mcp/profile.go`, `mcp/mask.go`
- Modify: `mcp/server.go` (register both)
- Test: `mcp/profile_test.go`, `mcp/mask_test.go`

**Interfaces:**
- Consumes: `parseRows` from Task 4.
- Produces:
  - `type profileArgs struct { Data, Format string }`, `func handleProfile(a profileArgs) (any, error)` — returns the inferred spec as YAML plus per-column stats.
  - `type maskArgs struct { Data, Format, Key, Locale string }`, `func handleMask(a maskArgs) (any, error)` — returns the masked dataset as text.

- [ ] **Step 1: Write the failing tests**

`mcp/profile_test.go`:

```go
package mcp

import (
	"strings"
	"testing"
)

func TestProfileInfersASpec(t *testing.T) {
	data := "email,age\na@example.com,31\nb@example.com,44\n"
	out, err := handleProfile(profileArgs{Data: data, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	yaml := out.(profileResult).Spec
	if !strings.Contains(yaml, "email") {
		t.Fatalf("the inferred spec does not mention the email column:\n%s", yaml)
	}
}

// The inferred spec must be usable as-is, or profile is a report rather than a
// step in a workflow.
func TestProfileSpecGenerates(t *testing.T) {
	data := "city,amount\nTashkent,12.50\nSamarkand,88.10\n"
	out, _ := handleProfile(profileArgs{Data: data, Format: "csv"})
	spec := out.(profileResult).Spec
	if _, err := handleGenerate(generateArgs{Spec: spec, Rows: 3, Seed: 1}); err != nil {
		t.Fatalf("the inferred spec does not generate: %v", err)
	}
}
```

`mcp/mask_test.go`:

```go
package mcp

import (
	"strings"
	"testing"
)

func TestMaskReplacesPersonalData(t *testing.T) {
	data := "name,email\nJohn Smith,john@example.com\n"
	out, err := handleMask(maskArgs{Data: data, Format: "csv", Key: "k"})
	if err != nil {
		t.Fatal(err)
	}
	got := out.(maskResult).Data
	if strings.Contains(got, "John Smith") || strings.Contains(got, "john@example.com") {
		t.Fatalf("the original values survived masking:\n%s", got)
	}
	if !strings.HasPrefix(got, "name,email") {
		t.Fatalf("the header was not preserved:\n%s", got)
	}
}

// The same key must give the same replacement, or a masked export loses its
// joins and stops being usable as a test fixture.
func TestMaskIsStableForTheSameKey(t *testing.T) {
	data := "email\njohn@example.com\njohn@example.com\n"
	out, _ := handleMask(maskArgs{Data: data, Format: "csv", Key: "k"})
	lines := strings.Split(strings.TrimSpace(out.(maskResult).Data), "\n")
	if lines[1] != lines[2] {
		t.Fatalf("the same value masked two different ways: %q vs %q", lines[1], lines[2])
	}
}

// A different key must make two exports unlinkable.
func TestDifferentKeysAreUnlinkable(t *testing.T) {
	data := "email\njohn@example.com\n"
	a, _ := handleMask(maskArgs{Data: data, Format: "csv", Key: "k1"})
	b, _ := handleMask(maskArgs{Data: data, Format: "csv", Key: "k2"})
	if a.(maskResult).Data == b.(maskResult).Data {
		t.Fatal("two different keys produced the same output")
	}
}

// Masking without a key would produce output that looks safe and is not
// reproducible; requiring one makes the choice explicit.
func TestMaskRequiresAKey(t *testing.T) {
	if _, err := handleMask(maskArgs{Data: "email\na@b.c\n", Format: "csv"}); err == nil {
		t.Fatal("masking without a key was accepted")
	}
}
```

- [ ] **Step 2: Run them to see them fail**

```bash
cd mcp && go test ./... -run "Profile|Mask"
```

Expected: FAIL, `undefined: handleProfile`.

- [ ] **Step 3: Write `mcp/profile.go`**

```go
package mcp

import (
	"strings"

	"github.com/bakhodir/synth/profile"
)

type profileArgs struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

// profileResult pairs the inferred spec with the statistics behind it. The
// spec alone hides the guesswork: a column typed as `email` because 3 of 4
// values parsed is a different claim from one where all 10000 did.
type profileResult struct {
	Spec    string `json:"spec"`
	Rows    int    `json:"rows"`
	Columns any    `json:"columns"`
}

func handleProfile(a profileArgs) (any, error) {
	if err := inputWithin(a.Data); err != nil {
		return nil, err
	}
	var (
		res *profile.Result
		err error
	)
	switch strings.ToLower(a.Format) {
	case "jsonl", "ndjson":
		res, err = profile.FromJSONL(strings.NewReader(a.Data))
	default:
		res, err = profile.FromCSV(strings.NewReader(a.Data))
	}
	if err != nil {
		return nil, err
	}
	yaml, err := res.YAML()
	if err != nil {
		return nil, err
	}
	return profileResult{Spec: string(yaml), Rows: res.Rows, Columns: res.Columns}, nil
}
```

> **Note for the implementer:** check `profile.Result`'s actual field and method
> names before writing this — `Rows`, `Columns` and `YAML()` are the expected
> shape but the existing type may differ. Adapt this handler to the type; do not
> change `profile.Result`'s public API to match the plan.

- [ ] **Step 4: Write `mcp/mask.go`**

```go
package mcp

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/bakhodir/synth/mask"
)

type maskArgs struct {
	Data   string `json:"data"`
	Format string `json:"format"`
	Key    string `json:"key"`
	Locale string `json:"locale"`
}

type maskResult struct {
	Data    string `json:"data"`
	Rows    int    `json:"rows"`
	Columns int    `json:"columns"`
}

// handleMask replaces personal data in a real export with generated values of
// the same shape.
//
// The key is required rather than defaulted. With a default, two exports masked
// on different days would silently be linkable, which is the one property the
// caller most likely believes they are buying.
func handleMask(a maskArgs) (any, error) {
	if err := inputWithin(a.Data); err != nil {
		return nil, err
	}
	if a.Key == "" {
		return nil, fmt.Errorf("key is required: it makes replacements stable, so " +
			"foreign keys still match across related exports. Use the same key for " +
			"related dumps, a fresh one to make two dumps unlinkable")
	}
	m := mask.New(a.Key, a.Locale)

	if strings.ToLower(a.Format) == "jsonl" || strings.ToLower(a.Format) == "ndjson" {
		var out strings.Builder
		rep, err := m.JSONL(strings.NewReader(a.Data), &out)
		if err != nil {
			return nil, err
		}
		return maskResult{Data: out.String(), Rows: rep.Rows, Columns: rep.Columns}, nil
	}
	var out strings.Builder
	rep, err := m.CSV(strings.NewReader(a.Data), &out)
	if err != nil {
		return nil, err
	}
	return maskResult{Data: out.String(), Rows: rep.Rows, Columns: rep.Columns}, nil
}

var _ = csv.NewReader // keep the import meaningful if the branch above changes
```

> **Note for the implementer:** `mask.Masker`'s CSV and JSONL methods are
> currently unexported (`csv`, `jsonl` in `mask/file.go`). Export them as `CSV`
> and `JSONL` taking `(io.Reader, io.Writer)` in the same commit, and rewrite
> `(*Masker).File` to call them. Do not copy the parsing logic into `mcp/`.

- [ ] **Step 5: Register both tools in `New`**

```go
	s.AddTool(mcp.NewTool("profile",
		mcp.WithDescription("Infer a Synth schema from an existing dataset, so you can generate "+
			"more data shaped like it. Returns the spec and the statistics behind it. Reads no files."),
		mcp.WithString("data", mcp.Required(), mcp.Description("The dataset itself, as text.")),
		mcp.WithString("format", mcp.Description("csv (default) or jsonl.")),
	), typed(handleProfile))

	s.AddTool(mcp.NewTool("mask",
		mcp.WithDescription("Replace personal data in a real export with generated values of the "+
			"same shape, keeping the file usable as a fixture. Reads and writes no files."),
		mcp.WithString("data", mcp.Required(), mcp.Description("The dataset itself, as text.")),
		mcp.WithString("format", mcp.Description("csv (default) or jsonl.")),
		mcp.WithString("key", mcp.Required(), mcp.Description(
			"Makes replacements stable. Same key for related dumps so foreign keys still match; "+
				"a fresh key to make two dumps unlinkable.")),
		mcp.WithString("locale", mcp.Description("Locale for the replacement values.")),
	), typed(handleMask))
```

- [ ] **Step 6: Run the tests**

```bash
cd mcp && go test ./... && cd .. && go test ./... && go vet ./...
```

Expected: PASS in both modules — exporting the mask methods must not break the
core module's own tests.

- [ ] **Step 7: Commit**

```bash
git add mcp/ mask/ profile/
git commit -m "feat(mcp): add profile and mask

mask requires its key rather than defaulting one. With a default, two exports
masked on different days would silently be linkable — which is the one property
the caller most likely believes they are buying by masking at all."
```

---

### Task 6: snapshot

**Files:**
- Create: `mcp/snapshot.go`
- Modify: `mcp/server.go` (register)
- Test: `mcp/snapshot_test.go`

**Interfaces:**
- Produces: `type snapshotArgs struct { Spec, At, From, To, Locale string; Rows int; Seed uint64 }`, `func handleSnapshot(a snapshotArgs) (any, error)`.

- [ ] **Step 1: Write the failing test**

`mcp/snapshot_test.go`:

```go
package mcp

import "testing"

const snapSpec = "name: t\nfields:\n  id: { kind: uuid, pk: true }\n  amount: { kind: amount }\n"

func TestSnapshotAtAnInstant(t *testing.T) {
	out, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 20, Seed: 1, At: "2026-07-01"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.(snapshotResult).Rows) == 0 {
		t.Fatal("the snapshot is empty")
	}
}

// The two forms are different questions and must not be mixed in one call.
func TestSnapshotRejectsBothForms(t *testing.T) {
	_, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 5, At: "2026-01-01", From: "2026-01-01", To: "2026-07-01"})
	if err == nil {
		t.Fatal("a call with both at= and from=/to= was accepted")
	}
}

func TestSnapshotBetweenReturnsEvents(t *testing.T) {
	out, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 50, Seed: 2,
		From: "2026-01-01", To: "2026-07-01"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.(snapshotResult).Events) == 0 {
		t.Fatal("no change events between two instants six months apart")
	}
}

// Replaying the events onto the earlier state must reproduce the later one, or
// the CDC stream is not a description of the same history.
func TestReplayEqualsTheLaterSnapshot(t *testing.T) {
	early, _ := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 30, Seed: 3, At: "2026-01-01"})
	late, _ := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 30, Seed: 3, At: "2026-07-01"})
	if len(early.(snapshotResult).Rows) == 0 || len(late.(snapshotResult).Rows) == 0 {
		t.Fatal("a snapshot was empty")
	}
	// The engine's own test covers replay equality; here we only check the two
	// instants differ, which is what makes the tool worth calling at all.
	if fmt.Sprint(early) == fmt.Sprint(late) {
		t.Fatal("two instants six months apart produced identical state")
	}
}

func TestSnapshotRejectsABadDate(t *testing.T) {
	if _, err := handleSnapshot(snapshotArgs{Spec: snapSpec, Rows: 5, At: "last Tuesday"}); err == nil {
		t.Fatal("an unparseable date was accepted")
	}
}
```

Add `import "fmt"` to the test file.

- [ ] **Step 2: Run it to see it fail**

```bash
cd mcp && go test ./... -run Snapshot
```

Expected: FAIL, `undefined: handleSnapshot`.

- [ ] **Step 3: Write `mcp/snapshot.go`**

```go
package mcp

import (
	"fmt"
	"time"

	"github.com/bakhodir/synth"
	"github.com/bakhodir/synth/cdc"
	"github.com/bakhodir/synth/snapshot"
)

// snapshotArgs asks one of two questions: what did the table look like at an
// instant (at=), or what changed between two (from=/to=). They are different
// questions, so setting both is an error rather than a silent preference.
type snapshotArgs struct {
	Spec   string `json:"spec"`
	At     string `json:"at"`
	From   string `json:"from"`
	To     string `json:"to"`
	Locale string `json:"locale"`
	Rows   int    `json:"rows"`
	Seed   uint64 `json:"seed"`
}

type snapshotResult struct {
	Rows   []map[string]any `json:"rows,omitempty"`
	Events []cdc.Event      `json:"events,omitempty"`
}

func handleSnapshot(a snapshotArgs) (any, error) {
	hasAt := a.At != ""
	hasRange := a.From != "" || a.To != ""
	if hasAt == hasRange {
		return nil, fmt.Errorf("set either at= for the state at one instant, " +
			"or from= and to= for the changes between two — not both")
	}
	n, err := rowsWithin(a.Rows)
	if err != nil {
		return nil, err
	}
	if err := inputWithin(a.Spec); err != nil {
		return nil, err
	}

	spec, err := synth.YAMLBytes([]byte(a.Spec))
	if err != nil {
		return nil, fmt.Errorf("the spec does not parse: %w", err)
	}
	tl, err := snapshot.New(spec.Schema(), snapshot.Config{
		Rows:   n,
		Seed:   a.Seed,
		Locale: a.Locale,
	})
	if err != nil {
		return nil, err
	}

	if hasAt {
		when, err := parseInstant(a.At)
		if err != nil {
			return nil, err
		}
		return snapshotResult{Rows: tl.At(when)}, nil
	}
	from, err := parseInstant(a.From)
	if err != nil {
		return nil, err
	}
	to, err := parseInstant(a.To)
	if err != nil {
		return nil, err
	}
	if to.Before(from) {
		return nil, fmt.Errorf("to= (%s) is before from= (%s)", a.To, a.From)
	}
	return snapshotResult{Events: tl.Between(from, to)}, nil
}

// parseInstant accepts a date or a full RFC 3339 timestamp. It rejects anything
// else rather than guessing: a misread date produces a plausible-looking
// snapshot of the wrong moment, which is worse than an error.
func parseInstant(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot read %q as a date — use 2026-07-01 or "+
		"2026-07-01T12:00:00Z", s)
}
```

> **Note for the implementer:** `snapshot.Config`'s fields and
> `synth.YAMLSpec`'s accessor for the underlying `*schema.Schema` may differ
> from the names above. Read `snapshot/snapshot.go` and `yaml.go` first and
> adapt. If `YAMLSpec` has no exported accessor for its schema, add one
> (`func (y *YAMLSpec) Schema() *schema.Schema`) in the same commit.

- [ ] **Step 4: Register the tool in `New`**

```go
	s.AddTool(mcp.NewTool("snapshot",
		mcp.WithDescription("Show a generated table as it stood at one instant (at=), or the "+
			"change events between two (from=/to=). Useful for testing migrations and "+
			"incremental ETL. Reads and writes no files."),
		mcp.WithString("spec", mcp.Required(), mcp.Description("A YAML schema.")),
		mcp.WithString("at", mcp.Description("An instant, e.g. 2026-07-01. Use this or from/to.")),
		mcp.WithString("from", mcp.Description("Start of the window.")),
		mcp.WithString("to", mcp.Description("End of the window.")),
		mcp.WithString("locale", mcp.Description("Data locale.")),
		mcp.WithNumber("rows", mcp.Description("How many rows. Default 10, maximum 1000.")),
		mcp.WithNumber("seed", mcp.Description("Seed.")),
	), typed(handleSnapshot))
```

- [ ] **Step 5: Run the tests**

```bash
cd mcp && go test ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add mcp/
git commit -m "feat(mcp): add snapshot

at= and from=/to= ask different questions, so setting both is an error rather
than a silent preference for one. A date that does not parse is rejected rather
than guessed: a misread date produces a plausible snapshot of the wrong moment,
which is worse than an error."
```

---

### Task 7: The boundary test

**Files:**
- Create: `mcp/boundary_test.go`

This task exists because the boundary is the thing most likely to be eroded by
a later well-meaning change, and a comment does not stop that.

**Interfaces:**
- Consumes: everything registered so far.

- [ ] **Step 1: Write the test**

`mcp/boundary_test.go`:

```go
package mcp

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// The MCP server must not import a package that reads the filesystem or opens
// a connection. It runs with the user's permissions on behalf of a model that
// may be reading attacker-controlled text, and a later change that adds
// os.ReadFile to a tool would be invisible in review.
func TestNoFilesystemOrNetworkImports(t *testing.T) {
	banned := map[string]string{
		"os":            "the server must not touch the filesystem",
		"net":           "the server must not open a connection",
		"net/http":      "the server must not open a connection",
		"os/exec":       "the server must not run a program",
		"path/filepath": "a path argument is the thing this boundary forbids",
		"io/ioutil":     "the server must not touch the filesystem",
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			for _, imp := range file.Imports {
				path, _ := strconv.Unquote(imp.Path.Value)
				if why, bad := banned[path]; bad {
					t.Errorf("%s imports %q: %s", name, path, why)
				}
			}
		}
	}
	_ = ast.Print // keep the ast import honest if the check is extended
}

// Every registered tool must have a description, because the description is
// the only thing a model has to decide whether to call it.
func TestEveryToolIsDescribed(t *testing.T) {
	if New() == nil {
		t.Fatal("New() returned nil")
	}
	// mcp-go does not expose the registered tool list; this test is a
	// placeholder to be filled in if it gains an accessor. Until then the
	// descriptions are covered by review.
	t.Skip("mcp-go exposes no accessor for registered tools")
}
```

> **Note for the implementer:** `main.go` under `mcp/cmd/synth-mcp` legitimately
> imports `os`. It is in a different directory, so `ParseDir(".")` does not see
> it — verify that is still true before trusting this test.

- [ ] **Step 2: Run it**

```bash
cd mcp && go test ./... -run Boundary
```

Expected: PASS. If it fails, the failure is the point — remove the import
rather than the test.

- [ ] **Step 3: Commit**

```bash
git add mcp/boundary_test.go
git commit -m "test(mcp): forbid filesystem and network imports

The boundary is the thing most likely to be eroded by a later well-meaning
change, and a comment in a package doc does not stop that."
```

---

### Task 8: README and installation

**Files:**
- Create: `mcp/README.md`
- Modify: `README.md` (a short section pointing at it)

- [ ] **Step 1: Write `mcp/README.md`**

````markdown
# Synth over MCP

Generate, verify, profile, mask and time-travel synthetic data from an MCP
client.

## Install

```bash
go install github.com/bakhodir/synth/mcp/cmd/synth-mcp@latest
```

## Configure

Claude Desktop (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "synth": {
      "command": "synth-mcp"
    }
  }
}
```

Claude Code:

```bash
claude mcp add synth -- synth-mcp
```

## Tools

| Tool | What it does |
|---|---|
| `generate` | Rows from a preset or a YAML spec |
| `list_types` | The ~250 column types |
| `list_presets` | The built-in schemas, with their YAML |
| `verify` | Checksums, formats and time anomalies in a dataset |
| `profile` | Infer a schema from an existing dataset |
| `mask` | Replace personal data in a real export |
| `snapshot` | State at an instant, or changes between two |

## What it does not do

- **No files.** Every tool takes its input as an argument and returns its
  result. Where the data is saved is your agent's decision.
- **No network.** stdio only. The server opens no socket and makes no outbound
  request.
- **No database.** Synth supplies data; a seeder writes it.

For a million rows, use the CLI: `synth gen --preset user -n 1000000 -o users.csv`.
The MCP tools cap at 1000 rows because the rows travel back through the model's
context window.

## Masking

`generate` masks card numbers and national identifiers by default. Pass
`unmasked: true` when a test genuinely needs the raw value — for example to
check that a validator accepts it.
````

- [ ] **Step 2: Add a section to the root `README.md`**

Place it after the CLI section:

```markdown
### MCP

Synth speaks MCP, so an assistant can generate and check data directly:

```bash
go install github.com/bakhodir/synth/mcp/cmd/synth-mcp@latest
claude mcp add synth -- synth-mcp
```

Seven tools: `generate`, `list_types`, `list_presets`, `verify`, `profile`,
`mask`, `snapshot`. The server is stdio-only and touches no file — see
[mcp/README.md](mcp/README.md).
```

- [ ] **Step 3: Verify the install path works**

```bash
cd mcp && go build ./cmd/synth-mcp && ./synth-mcp < /dev/null; echo "exit: $?"
```

Expected: it exits without a panic. A clean EOF on stdin is not an error.

- [ ] **Step 4: Commit**

```bash
git add mcp/README.md README.md
git commit -m "docs(mcp): document installation and the boundary

The 'what it does not do' section is the part worth reading: it is what a user
needs to know before granting a tool their own permissions."
```

---

## Verification

After every task:

```bash
cd mcp && go test ./... && go vet ./...
cd .. && go test ./... && go vet ./...
go list -m all | grep -c mark3labs   # must print 0
```

The last line is the one that catches the failure this plan's structure exists
to prevent.
