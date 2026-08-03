# Synth over MCP

Generate, verify, profile, mask and time-travel synthetic data from an MCP
client.

## Install

```bash
go install github.com/bakhod1r/synth/mcp/cmd/synth-mcp@latest
```

## Configure

Claude Code:

```bash
claude mcp add synth -- synth-mcp
```

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

## Tools

| Tool | What it does |
|---|---|
| `generate` | Rows from a preset or a YAML spec |
| `list_types` | The ~250 column types, with whether each follows the locale |
| `list_presets` | The built-in schemas, with their YAML |
| `verify` | Checksums, formats and time anomalies in a dataset |
| `profile` | Infer a schema from an existing dataset |
| `mask` | Replace personal data in a real export |
| `snapshot` | State at an instant, or the changes between two |
| `diff` | Compare two datasets' shape — columns, types, ranges, categories |

## What it does not do

- **No files.** Every tool takes its input as an argument and returns its result
  in the response. Where the data is saved is your agent's decision.
- **No network.** stdio only. The server opens no socket and makes no outbound
  request.
- **No database.** Synth supplies data; a seeder writes it.

The file rule is not a simplification, and it is enforced by a test rather than
by a comment. An MCP server runs with your own permissions on behalf of a model
that may be reading text someone else wrote. A tool that accepted a file path
would turn a data generator into a file-reading primitive for whoever can get
text in front of that model.

For a million rows, use the CLI, which has no such limit:

```bash
synth gen --preset user -n 1000000 -o users.csv
```

The MCP tools cap at 1000 rows, because the rows travel back through the model's
context window.

## Masking

`generate` masks card numbers and national identifiers by default. Pass
`unmasked: true` when a test genuinely needs the raw value — for example to
check that a validator accepts it.

`mask` requires a `key`. Use the same key across related dumps so foreign keys
still match, and a fresh key when two dumps must not be linkable. There is no
default, because a default would silently make every export linkable to every
other — the one property you are most likely masking to avoid.

Read the `untouched` list in the result. It names the columns nothing was
recognized in, which is where unmasked personal data would be hiding.

## Examples

Ask for a fixture:

> Give me 100 fake transactions in Uzbek locale.

Check an export someone sent you:

> Here's a CSV — verify it. *(paste the rows)*

Learn a schema from real data, then generate more of it:

> Profile this sample and generate 50 rows from the spec you infer.

## Development

```bash
cd mcp
go test ./...
go vet ./...
```

This is a separate Go module. `mcp-go` brings 20 transitive dependencies, so it
stays out of the core module's graph — anyone importing `synth` should not pay
for an SDK they never call. The core module must never import anything under
`mcp/`:

```bash
cd .. && go list -m all | grep -c mark3labs   # must print 0
```
