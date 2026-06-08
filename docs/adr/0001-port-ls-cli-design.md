# ADR 0001: port-ls CLI design

## Status

Accepted

## Context

`port-ls` needs to list local ports in use on Windows, WSL, macOS, and Linux. The useful output includes PID and process name, but those fields are OS- and permission-dependent.

The repository started with no implementation, README, docs, ADR, CI, or release settings. The initial decision therefore defines the public CLI contract and the project operating model.

## Decision

Build `port-ls` as a Go 1.24+ CLI using Cobra.

The initial provider layer uses OS standard commands:

- Linux: `ss`, `netstat`, `lsof`
- macOS: `lsof`, `netstat`
- Windows: PowerShell, `netstat`
- WSL host: Windows PowerShell, `netstat.exe`

Providers are hidden behind an internal interface so they can later be replaced by native OS APIs or maintained Go libraries.

Default behavior lists listening TCP/UDP ports. `--all` expands to all connections. Output is a table by default and JSON Lines with `--json`.

WSL lists WSL ports by default. `--host` includes Windows host ports and marks namespaces explicitly.

`port-ls free` is included from the initial release for developer workflows that need a free port. It validates candidates by checking current usage and then binding.

Configuration is supported through TOML and `PORT_LS_` environment variables. Precedence is CLI, environment, config file, defaults.

Releases use GoReleaser on `v*` tags. CI runs unit tests, lint, native builds on Windows/macOS/Linux, Linux smoke tests, and GoReleaser snapshot builds.

## Consequences

The CLI can ship quickly and remain useful without elevated privileges.

OS command output differences become a maintenance concern. Fixture tests are required for parsers, and provider diagnostics must stay available through `--verbose`.

PID and process name are best-effort fields. Missing values are represented as `unknown` instead of failing the command.

The setting system increases initial scope, but it gives stable defaults for normal listing and `free`.

## Alternatives Considered

- Native OS APIs from the start: more robust long term, but significantly higher initial complexity.
- Existing Go library: lower implementation effort, but dependency health, OS coverage, and license compatibility would need separate evaluation.
- No config file or environment variables: simpler, but does not meet the accepted requirements.
- Delay `port-ls free`: smaller initial CLI, but less useful for development workflows.
- Add kill/terminate support: rejected for initial scope due to safety, permission, and OS behavior risks.
