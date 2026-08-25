# Versioning and stability

Synth follows [semantic versioning](https://semver.org). The public API is
frozen at `v1`: a breaking change means `v2`, and in Go `v2` is a different
import path — so nothing that would break your build can arrive by accident.

See [CHANGELOG.md](https://github.com/bakhod1r/synth/blob/main/CHANGELOG.md).

## What is already enforced rather than promised

| | |
| --- | --- |
| Tests | 380+, race-clean, on every push |
| Fuzzing | six parser targets, nightly |
| Core dependencies | exactly two, and CI fails if that slips |
| Boundaries | no database, no network, no files from MCP — each with a test |

That last row is the load-bearing one. "Synth never connects to your database"
is not a design intention here; it is a test that fails if an import appears.

## Security

Report vulnerabilities per
[SECURITY.md](https://github.com/bakhod1r/synth/blob/main/SECURITY.md).

## Contributing

See
[CONTRIBUTING.md](https://github.com/bakhod1r/synth/blob/main/CONTRIBUTING.md)
and the
[code of conduct](https://github.com/bakhod1r/synth/blob/main/CODE_OF_CONDUCT.md).
