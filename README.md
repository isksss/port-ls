# port-ls

`port-ls` lists local ports in use.

It works on Windows, WSL, macOS, and Linux.

## Install

```sh
go install github.com/isksss/port-ls/cmd/port-ls@latest
```

Release archives are also published from `v*` tags on GitHub Releases.

## Usage

```sh
port-ls
port-ls 3000
port-ls --all
port-ls --json
port-ls --name node
port-ls --address 127.0.0.1
port-ls --state established --all
port-ls --check 3000
port-ls free 3000
port-ls free 3000-3010
```

Default output is a table:

```text
PORT  PROTO  ADDRESS    STATE   PID      NAME     NAMESPACE
3000  tcp    127.0.0.1  listen  12345    node     local
```

`--json` writes JSON Lines:

```json
{"port":3000,"protocol":"tcp","address":"127.0.0.1","state":"listen","pid":12345,"name":"node","namespace":"local"}
```

`--check <port>` writes no stdout. It exits `0` when the port is in use and `1` when it is not found.

`port-ls` only searches ports. It does not kill or terminate processes.

## WSL

On WSL, `port-ls` lists WSL ports by default.

Use `--host` to include Windows host ports:

```sh
port-ls --host
port-ls --check 3000 --host
```

The `NAMESPACE` field distinguishes `wsl` and `windows`.

`--host` is valid only on WSL. `port-ls free --host` is not supported; run `port-ls free` on Windows if you need a Windows-side free port.

## Free Port

`port-ls free` prints a single free port to stdout:

```sh
port-ls free 3000
```

`port-ls free 3000` searches `3000-4000`. If `start+1000` exceeds `65535`, the end is clamped to `65535`.

`port-ls free` checks candidate ports by reading the current port list and then trying to bind. TCP is checked by default. Use `--udp` for UDP, or `--tcp --udp` to require both protocols to be free.

The default bind address is `127.0.0.1`. Use `--address` to change it:

```sh
port-ls free 3000 --address 0.0.0.0
```

## Configuration

Configuration precedence is:

1. CLI flags
2. Environment variables
3. TOML config file
4. Defaults

Default config paths:

- Linux, WSL, macOS: `$XDG_CONFIG_HOME/port-ls/config.toml` or `~/.config/port-ls/config.toml`
- Windows: `%AppData%\port-ls\config.toml`

Use `--config` to specify a file explicitly:

```sh
port-ls --config ./config.toml
port-ls free --config ./config.toml
```

Example:

```toml
json = false
all = false
tcp = true
udp = true
address = "127.0.0.1"
state = ["listen"]
host = false
verbose = false
name = "node"

[free]
default_start = 3000
# or:
# default_range = "3000-3010"
```

`free.default_start` and `free.default_range` are mutually exclusive.

Environment variables use the `PORT_LS_` prefix:

```sh
PORT_LS_JSON=true
PORT_LS_STATE=listen,established
PORT_LS_FREE_DEFAULT_START=3000
PORT_LS_FREE_DEFAULT_RANGE=3000-3010
```

Boolean values follow Go's `strconv.ParseBool`.

## Providers

Initial providers use OS commands and fall back when possible:

- Linux: `ss` -> `netstat` -> `lsof`
- macOS: `lsof` -> `netstat`
- Windows: PowerShell -> `netstat`
- WSL `--host`: PowerShell on Windows -> `netstat.exe`

If no provider succeeds, `port-ls` exits `3`. If PID or process name cannot be read due to permissions or provider limits, the value is shown as `unknown`.

`port-ls` never performs automatic privilege escalation.

## Exit Codes

- `0`: success
- `1`: no matching result for `--check` or `free`
- `2`: usage error
- `3`: provider or port lookup failure

## Completion

```sh
port-ls completion bash
port-ls completion zsh
port-ls completion fish
port-ls completion powershell
```

## Development

```sh
go test ./...
go build ./cmd/port-ls
```

Release builds are handled by GoReleaser on `v*` tags.
