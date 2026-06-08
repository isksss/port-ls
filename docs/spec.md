# port-ls specification

## Scope

`port-ls` is a Go 1.24+ CLI for listing local ports in use on Windows, WSL, macOS, and Linux.

The initial implementation uses OS command providers and parser adapters. Provider code must be isolated so it can later be replaced by native API or library implementations.

`port-ls` is search-only. It must not kill or terminate processes.

## Commands

### `port-ls [port]`

Lists ports in use.

Default scope:

- TCP and UDP
- listening connections only
- local namespace only

Options:

- `--all`: include non-listening connections
- `--tcp`: include TCP
- `--udp`: include UDP
- `--name <value>`: case-insensitive process name substring
- `--address <value>`: address substring
- `--state <value>`: state filter, repeatable
- `--host`: on WSL only, include Windows host namespace
- `--json`: output JSON Lines
- `--verbose`: write diagnostics to stderr
- `--check <port>`: no stdout; exit code only
- `--version`: print `version commit date`
- `--config <path>`: explicit TOML config path

`--tcp --udp` is allowed and means both protocols.

Protocol filters and state filters are OR conditions within their kind. Port, name, address, namespace, and other filters are AND conditions.

`--check` requires a port argument, forbids `--json`, and ignores config/environment-derived filters, `host`, and `verbose`. CLI-provided `--host`, `--tcp`, `--udp`, `--name`, `--address`, `--state`, `--all`, and `--verbose` still apply.

### `port-ls free [start|start-end]`

Prints one free port to stdout.

Rules:

- `3000-3010` searches that inclusive range.
- `3000` searches `3000-4000`.
- If `start+1000 > 65535`, end is clamped to `65535`.
- Without an argument, it is allowed only when `free.default_start` or `free.default_range` is configured.
- TCP is checked by default.
- `--udp` checks UDP.
- `--tcp --udp` requires both TCP and UDP to be free.
- The default bind address is `127.0.0.1`.
- `--address` changes the bind address and may be IPv4 or IPv6.
- `--json` writes JSON Lines like `{"port":3000,"protocol":["tcp"]}`.
- No free port exits `1` and writes no stdout.

`free` must reject `--host`, `--all`, `--state`, and `--name` with usage error `2`.

`free` applies config/environment-derived `free.default_*`, `address`, `tcp`, and `udp`. It applies `json` and `verbose` only when explicitly provided as CLI flags.

### `port-ls completion [bash|zsh|fish|powershell]`

Generates shell completion. This command must not load config files or environment variables.

## Output

Table columns:

```text
PORT PROTO ADDRESS STATE PID NAME NAMESPACE
```

Table output uses `text/tabwriter` and does not truncate values. If no rows match, table output writes only the header.

JSON Lines fields:

```json
{"port":3000,"protocol":"tcp","address":"127.0.0.1","state":"listen","pid":1234,"name":"node","namespace":"local"}
```

Field rules:

- `port`: number
- `protocol`: `tcp` or `udp`
- `address`: host/interface only, no port
- `state`: lower snake_case
- `pid`: number or `null`
- `name`: process executable name or `"unknown"`
- `namespace`: `local`, `wsl`, or `windows`

`--json` writes data only to stdout. `--verbose` diagnostics go only to stderr.

## Normalization

- Protocol values are lowercase `tcp` or `udp`.
- State values are lower snake_case. Examples: `listen`, `established`, `time_wait`, `unknown`.
- User-provided state values are normalized the same way. Invalid empty states are usage error `2`.
- Unknown OS state values are lower snake_case and preserved.
- Address values are host/interface only.
- IPv4 wildcard is `0.0.0.0`.
- IPv6 wildcard is `::`.
- IPv6 addresses are supported for listing, filtering, and `free --address`.
- Process name is executable name only. Command line arguments must not be shown or matched.
- `--name` is case-insensitive substring matching.
- Windows `.exe` is preserved in output, but `.exe` presence does not affect `--name` matching.

Ports must be `1-65535`. Port `0` and out-of-range values are usage error `2`.

## Sorting

Rows are sorted by:

1. `port`
2. `protocol`
3. `address`
4. `state`
5. `pid`

Ascending order is used.

## Providers

Provider fallback order:

- Linux: `ss`, `netstat`, `lsof`
- macOS: `lsof`, `netstat`
- Windows: PowerShell, `netstat`
- WSL host: PowerShell via Windows, `netstat.exe`

If all provider candidates fail, exit `3`. If PID/name is unavailable, use `unknown` instead of failing.

Provider execution must not pass user input through a shell. PowerShell providers may use a fixed script, but user filters must be applied in Go after data collection.

No automatic `sudo`, UAC, or privilege escalation is allowed.

`--verbose` may print provider name, command name, exit status, and stderr summary. It must not print raw connection output, full command lines, or process command lines.

## WSL

WSL is detected by reading `/proc/sys/kernel/osrelease` or `/proc/version` and checking for `microsoft` or `wsl` case-insensitively.

Default WSL namespace is `wsl`.

`--host` includes Windows entries with namespace `windows`. If Windows-side collection fails, the whole command exits `3`; partial WSL-only success is not allowed.

`--check <port> --host` succeeds when either WSL or Windows namespace has a matching row.

Non-WSL `--host` is usage error `2`.

## Config

TOML config paths:

- Linux, WSL, macOS: `$XDG_CONFIG_HOME/port-ls/config.toml` or `~/.config/port-ls/config.toml`
- Windows: `%AppData%\port-ls\config.toml`

`--config` may explicitly specify a path. An explicitly specified missing file is usage error `2`. Missing auto-discovered config is ignored.

Unknown config keys are usage error `2`.

Supported TOML keys:

```toml
json = false
all = false
tcp = false
udp = false
address = "127.0.0.1"
state = ["listen"]
host = false
verbose = false
name = "node"

[free]
default_start = 3000
default_range = "3000-3010"
```

`free.default_start` and `free.default_range` are mutually exclusive.

Environment variables:

- `PORT_LS_JSON`
- `PORT_LS_ALL`
- `PORT_LS_TCP`
- `PORT_LS_UDP`
- `PORT_LS_ADDRESS`
- `PORT_LS_STATE`
- `PORT_LS_HOST`
- `PORT_LS_VERBOSE`
- `PORT_LS_NAME`
- `PORT_LS_FREE_DEFAULT_START`
- `PORT_LS_FREE_DEFAULT_RANGE`

Boolean environment variables use Go `strconv.ParseBool`.

`PORT_LS_STATE` is comma-separated. Empty values are usage error `2`.

Precedence:

1. CLI arguments
2. Environment variables
3. Config file
4. Defaults

## Exit Codes

- `0`: success
- `1`: not found for `--check`, or no free port
- `2`: usage error
- `3`: provider or lookup failure

`--help` and `help` exit `0`.

## Testing

- Parser behavior must be covered by fixture unit tests.
- Linux CI must run a minimal provider smoke test.
- Windows and macOS CI require fixture unit tests, lint, and build.
- GoReleaser snapshot must run on push and pull request.
- Release workflow must run `go test ./...` and `goreleaser check` before publishing.

## Release

GoReleaser publishes on `v*` tags.

Initial release targets:

- `windows/amd64`
- `linux/amd64`
- `darwin/amd64`
- `darwin/arm64`
- `linux/arm64`

Archive rules:

- Windows: `.zip`
- Linux/macOS: `.tar.gz`
- name: `port-ls_<version>_<os>_<arch>.<ext>`
- checksum file: `port-ls_checksums.txt`

Initial release verification uses checksums only. SBOM, cosign, Homebrew, and Scoop are future work.
