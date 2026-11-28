# AGENTS.md

Guidance for coding agents working in `github.com/urzeye/lazyssh`.

## Project Overview

`lazyssh` is a keyboard-driven terminal UI (TUI) for managing OpenSSH configuration files and connections. It is an independently maintained fork of `Adembc/lazyssh` supporting recursive `Include` trees, dotfiles-friendly managed writes, read-only guards, and SSH/SFTP/SCP workflows.

- **Language & Go version**: Go `1.24.6` (defined in `go.mod`).
- **Core dependencies**: `github.com/rivo/tview` + `github.com/gdamore/tcell/v2` (TUI), `github.com/spf13/cobra` (CLI), `go.uber.org/zap` (logging), `github.com/atotto/clipboard` (clipboard).
- **Important `go.mod` replace**: `github.com/kevinburke/ssh_config` is replaced with `github.com/adembc/ssh_config v1.4.2`. Keep imports as `github.com/kevinburke/ssh_config`.
- **Makefile filename**: lowercase `makefile` at the repository root.

## Build, Lint & Test Commands

CI (`.github/workflows/go.yml`) runs `make build` and `make test` on `ubuntu-latest` using the Go version in `go.mod`.

```bash
# Run full test suite with race detector and coverage output (coverage.out)
make test

# Run tests without caching or in short mode
go test -race -count=1 ./...
make test-short

# Code quality checks (all run automatically before `make build`)
make fmt        # runs ./bin/gofumpt -l -w . and go fmt ./...
make vet        # runs go vet ./...
make lint       # runs fmt + ./bin/golangci-lint run
make lint-fix   # runs ./bin/golangci-lint run --fix
make quality    # fmt + vet + lint

# Build binary to ./bin/lazyssh (depends on `quality`)
make build

# Cross-platform compile verification (fast check without writing artifacts)
GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd
GOOS=darwin GOARCH=arm64 go build -o /dev/null ./cmd
GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd
```

**Local toolchain note**: `makefile` downloads pinned tools (`golangci-lint v1.64.2`, `gofumpt v0.7.0`, `staticcheck 2024.1.1`) into `./bin/`. If your local Go version does not match the toolchain that built the cached `./bin/golangci-lint-v1.64.2` (or is newer than what `golangci-lint v1.64.2` supports), `make lint` may fail with `cannot decode ... export data version`. Run with the `go.mod` toolchain (e.g. `GOTOOLCHAIN=go1.24.6`) or remove `./bin/golangci-lint*` and rebuild with a compatible Go version.

## Architecture & Layout

The codebase follows Hexagonal Architecture (Ports & Adapters). Dependencies must point inward toward `internal/core`.

```text
cmd/
  main.go                                   # Cobra CLI wiring, flags/env parsing, direct `-c` connect, exit-code preservation
internal/
  core/
    domain/server.go                        # Pure domain models: Server, ActiveSession
    ports/                                  # Interfaces: ServerRepository, ServerService
    services/
      server_service.go                     # Fuzzy ranking, core validation, ssh/sftp execution, port-forward lifecycle, Ping
      ssh_session.go                        # Interactive command runner with bounded stderr tail & WaitDelay
      active_sessions.go                    # SSH client process parsing & formatting
      active_sessions_{linux,unix,windows}.go # OS-specific process listing (/proc, ps, PowerShell CIM)
      sysprocattr_{unix,windows}.go         # OS-specific background process detachment (Setsid on Unix)
  adapters/
    data/ssh_config_file/
      ssh_config_file_repo.go               # Repository CRUD, read-only & managed-mode guards
      include.go                            # Recursive Include flattening, OpenSSH evaluation order, cycle/depth limits
      config_io.go                          # Deferred Include decode, negated pattern restoration, atomic symlink-safe save
      crud.go                               # Host insertion before wildcards, Match-safe node updates, forward format conversion
      mapper.go                             # Effective single/multi-value resolution & KV-to-domain mapping
      metadata_manager.go                   # ~/.lazyssh/metadata.json (tags, pins, ssh_count, last_seen)
      backup.go                             # One-time .original.backup + rolling 10 timestamped backups
      capabilities.go                       # Offline OpenSSH SessionType capability probe
      file_system.go                        # FileSystem abstraction for deterministic testing
    ui/
      tui.go                                # App lifecycle, layout, theme/sort init, 10s active-sessions ticker, UI state save/restore
      handlers.go                           # Global keybindings, modals, async ping/refresh/forwarding, interactive session suspend
      server_form.go                        # 5-tab host editor (Basic, Connection, Forwarding, Authentication, Advanced)
      defaults.go / field_help.go / validation.go # Centralized SSH defaults, placeholders, help panel docs, field validators
      ssh_command_parser.go                 # Safe, non-executing parser for pasted `ssh` commands & ssh:// URIs
      scp.go / native_command.go            # SCP protocol detection (-s vs legacy), remote path escaping, POSIX/Windows command formatting
      server_list.go / server_details.go / active_sessions.go / search_bar.go / header.go / hint_bar.go / status_bar.go
      clone.go / sort.go / theme.go / settings.go / utils.go / network_interfaces.go / const.go
  logger/logger.go                          # Zap SugaredLogger writing to ~/.lazyssh/lazyssh.log
```

**Runtime local state files** (all under `~/.lazyssh/`):
- `metadata.json`: tags, pin timestamps, SSH counts, last-seen timestamps (mode `0600`).
- `settings.json`: persisted UI sort mode and dark/light theme (mode `0600`).
- `lazyssh.log`: structured debug logs.

## Core Invariants & Gotchas

### 1. OpenSSH Config Parsing, Resolution & Safe Writes (`internal/adapters/data/ssh_config_file`)
- **Always decode through `decodeConfig` (`config_io.go`)**: It temporarily rewrites `Include` keys (`__lazyssh_include_` prefix) before calling `ssh_config.Decode` so the upstream library does not eagerly glob/read files (which breaks on directories, deep cycles, or conditional `Match` includes), then restores keys and lets `flattenConfig` (`include.go`) expand includes in OpenSSH evaluation order (up to `maxIncludeDepth = 16`, with cycle detection).
- **Preserve negated host patterns**: `ssh_config.Decode` strips `!` from `Pattern.Str`; `decodeConfig` restores leading `!` so round-trip saves do not corrupt `Host !foo` blocks.
- **Respect `Match` boundaries on every write**: `ssh_config` attaches `Match` directives to the preceding `Host`'s `Nodes` slice. All host node edits, insertions, removals, and host deletions (`crud.go`) must respect `matchBoundaryIndex` and preserve trailing `Match` sections. On reads (`include.go`), unconditional `Match all` resumes global scope while conditional `Match` blocks are safely skipped.
- **Preserve OpenSSH first-match precedence**:
  - `insertHost` places newly added hosts before matching wildcard/default blocks (e.g., `Host *`).
  - `updateHostNodes` updates single-value directives from an ordered slice (never an unordered map) and normalizes deprecated aliases (`PubkeyAcceptedKeyTypes`, `HostbasedKeyTypes`, `HostbasedAcceptedKeyTypes`, `ChallengeResponseAuthentication`) in place.
  - Maintain explicit `ProxyJump` vs `ProxyCommand` ordering, including explicit `ProxyJump none` alongside `ProxyCommand`.
- **Atomic, symlink-safe writes and backups**: `saveConfig` resolves symlinks via `resolvedWriteConfigPath` (so dotfiles symlinks are never replaced with regular files), writes to a same-directory temp file with `0600` permissions, calls `Sync()`, maintains `<name>.original.backup` plus up to 10 rolling `<name>-<unix-ms>-lazyssh.backup` files, and replaces via `os.Rename`.
- **Read/write separation in Managed & Read-Only modes**:
  - `GetConfigFile()` always returns `readConfigPath` (the root config passed to `ssh`/`sftp`/`scp` via `-F`).
  - In managed mode, only hosts defined in `writeConfigPath` are writable (`Readonly == false`); earlier read-only included definitions must not shadow writable ownership, while effective values still resolve in full OpenSSH order.
  - `UpdateTags` and `SetPinned` write **only** to `metadata.json` and must never rewrite SSH config files. `UpdateTags` enforces read-only restrictions; `SetPinned` remains allowed in read-only mode.
- **Offline `SessionType` probe (`capabilities.go`)**: Saving a non-empty `SessionType` (`none`, `subsystem`, `default`) must run `ssh -G -F <empty-temp-file> -o SessionType=<val> lazyssh-capability.invalid` before mutating config or metadata. Never read the user's real SSH config or initiate network I/O during this probe.

### 2. System OpenSSH Execution, Import Parser & SCP
- **No credential storage & no shell execution**: Invoke `ssh`/`sftp` directly via `exec.Command` with `--` before the host alias. Never execute clipboard text; `ParseSSHCommand` must remain a pure lexical parser that rejects shell operators (`;`, `|`, `&`, `<`, `>`, `` ` ``), unsupported flags (including `-F`), and URI passwords.
- **Interactive session diagnostics & exit codes**: `runInteractiveSSH` (`ssh_session.go`) tees stderr to a bounded 16 KiB `diagnosticTail` and sets `cmd.WaitDelay = 2s` so lingering proxy children holding stderr pipes cannot hang exit. Direct CLI `-c` preserves the underlying `*exec.ExitError` exit code and suppresses duplicate stderr printing when `StderrAlreadyPrinted()` is true.
- **`RemoteCommand` percent-escaping boundary**:
  - Positional remote commands parsed by `ParseSSHCommand` escape literal `%` to `%%` (because CLI commands bypass OpenSSH config token expansion).
  - Explicit `-o RemoteCommand=...` retains OpenSSH `%` templates as-is.
  - `BuildSSHCommand` (`utils.go`) always emits `RemoteCommand` via `-o 'RemoteCommand=...'` so copy -> reimport round trips remain lossless.
- **SCP protocol & cross-platform quoting (`scp.go`, `native_command.go`)**:
  - Inspect local `scp -h` once when opening the SCP helper: use `-s` (SFTP mode) when supported, or `quoteLegacyRemotePath` (backslash escaping, not shell quotes, to stay compatible with legacy `scp`'s local `fnmatch` check).
  - Expand local `~` locally; keep remote `~` unexpanded for the remote server.
  - On Windows, `formatNativeCommand` wraps invocations via PowerShell `ProcessStartInfo` + `quoteWindowsArgument` to stay independent of PowerShell 5.1/7 native argument passing modes.

### 3. TUI Threading, Escaping & UX Contracts (`internal/adapters/ui`)
- **UI thread safety**: Any background goroutine (ping, async refresh, background port forward, 10s active sessions refresh) must update widgets inside `t.app.QueueUpdateDraw(...)`.
- **Transient status messages**: Use `showStatusTemp` / `showStatusTempColor`, which guard reset timers with atomic `statusSeq` so older timers never clear newer messages.
- **Always escape dynamic text for `tview`**: Widgets with `SetDynamicColors(true)` (`ServerDetails`, `ActiveSessions`, `ServerList`, status bar, `tview.Modal`) interpret `[...]` as color tags. Wrap dynamic values (aliases, hosts, errors, tags, forwards like `[::1]:1080`) with `tview.Escape(...)`.
- **Preserve user state on errors & focus**:
  - Save failures in Add/Edit/Tags/Port-Forward/SCP forms must return to the same form instance when the error modal closes, never discarding user input.
  - Global single-key shortcuts must not fire when `t.searchBar.HasFocus()` is true (including after mouse click focus).
  - Changing the search query selects index `0` (top fuzzy match); background refresh preserves current selection and ignores stale results if the query changed mid-flight.

### 4. Checklist: Adding or Modifying an SSH Config Field
SSH options are wired across multiple layers. When adding or renaming a supported directive, update **all** applicable locations and add round-trip tests:
1. `internal/core/domain/server.go` (`Server` struct field)
2. `internal/adapters/data/ssh_config_file/mapper.go` (`effectiveSingleValueGroups` or `effectiveMultiValueKeys`, plus `map*Config`)
3. `internal/adapters/data/ssh_config_file/crud.go` (`updateHostNodes` ordered slice or multi-value block, plus `getProperKeyCase`)
4. `internal/adapters/ui/defaults.go` (`SSHFieldDefaults`, `GetFieldPlaceholder`)
5. `internal/adapters/ui/field_help.go` (`fieldHelpData`, `formatDefaultValue` if special)
6. `internal/adapters/ui/validation.go` (validator & `fieldOrder` if validated)
7. `internal/adapters/ui/server_form.go` (form item creation, prefill/load, value extraction on save)
8. `internal/adapters/ui/server_details.go` (details panel display group)
9. `internal/adapters/ui/utils.go` (`BuildSSHCommand`)
10. `internal/adapters/ui/ssh_command_parser.go` (flag / `-o` option parsing)
11. `internal/adapters/ui/clone.go` (`slices.Clone` if the field is a slice)

## Code Style, Licensing & Cross-Platform Rules

- **Mandatory license header**: `.golangci.yml` enables `goheader`. Every new `.go` file must begin with the exact Apache 2.0 `Copyright 2025.` header (before any `//go:build` tag):
  ```go
  // Copyright 2025.
  //
  // Licensed under the Apache License, Version 2.0 (the "License");
  // you may not use this file except in compliance with the License.
  // You may obtain a copy of the License at
  //
  //     http://www.apache.org/licenses/LICENSE-2.0
  //
  // Unless required by applicable law or agreed to in writing, software
  // distributed under the License is distributed on an "AS IS" BASIS,
  // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  // See the License for the specific language governing permissions and
  // limitations under the License.
  ```
- **Linters (`.golangci.yml`)**: Enforces `gofmt`, `goimports`, `govet`, `staticcheck`, `gocritic` (diagnostic/experimental/opinionated/performance/style), `gosec`, `errorlint`, `exhaustive`, `prealloc`, `revive` (`comment-spacings`), `misspell` (US English), `nolintlint`, `unparam`, `unused`, etc. (`internal/*` excludes `dupl`/`lll`; `*_test.go` excludes `dupl`/`errcheck`/`goconst`/`gocyclo`/`gosec`.)
- **Linter suppression**: When suppressing `gosec` (e.g., `G204` for OpenSSH exec or `G304` for controlled paths) or `nolint`, include a brief justification comment (e.g., `// #nosec G204 -- ...` or `//nolint:exhaustive // ...`).
- **Cross-platform builds**: Keep OS-specific implementations isolated with build tags (`linux`, `!windows && !linux`, `windows`) and verify `linux`, `darwin`, and `windows` builds when touching process/path/command handling.

## Testing Conventions

- **Never depend on real user configs or remote hosts**: Use `t.TempDir()`, injectable hooks (`FileSystem`, `sshCommand`, `sftpCommand`), simulated terminals (`tcell.NewSimulationScreen`), or test-binary subprocess helpers (see `TestSSHSessionHelper` in `internal/core/services/ssh_session_test.go`).
- **Isolate home directory on both Unix and Windows**: Whenever code reads `os.UserHomeDir()`, set **both**:
  ```go
  t.Setenv("HOME", home)
  t.Setenv("USERPROFILE", home)
  ```
- **Optional external fixtures**: `TestImportedCommandExecRequestSurvivesCopyAndReimport` (`internal/adapters/ui/ssh_command_execution_test.go`) uses `testdata/ssh_command_server.py` and automatically skips if `ssh` or Python `paramiko` is unavailable. `TestLegacySCPReceiverChecksRequestedFilename` (`internal/adapters/ui/scp_transfer_test.go`) builds `./testdata/scp_transport` and exercises the local `scp` client offline.

## PR, Docs & Release Conventions

- **Semantic PR titles** (enforced by `.github/workflows/semantic-prs.yml`):
  - Allowed types: `fix`, `feat`, `improve`, `refactor`, `revert`, `test`, `ci`, `docs`, `chore`.
  - Allowed optional scopes: `ui`, `cli`, `config`, `parser`.
  - Example: `fix(config): preserve negated host patterns on save`.
- **Bilingual documentation**: Keep both `README.md` (English) and `README.zh-CN.md` (Simplified Chinese) in sync when changing CLI flags, keybindings, or user-visible behavior.
- **Release tags** (`.github/workflows/release.yml` + `.goreleaser.yaml`): Strict CalVer without leading zeroes: `vYYYY.M.D` (e.g., `v2026.11.28`).
