# Maintenance implementation and review

Date: 2026-11-18. Baseline: `cee1c46`. Code reviewed through `9e2b880`.

## Completed scope

| Item | Implementation commit | Verification |
| --- | --- | --- |
| Preserve negated Host patterns in root and included files | `8bf3462` | Save round trip, excluded aliases, included patterns |
| Insert new hosts before matching defaults | `a090ae8` | Wildcard, patterned, negated, global, and managed-file cases |
| Keep mouse-focused search input out of global commands | `688c3ee` | Simulated mouse focus, printable keys, state capture |
| Save tags without rewriting SSH config; retain selection | `85ea02a` | Fresh tags, last-tag clearing, metadata errors, unchanged config, reorder/removal/empty list |
| Accept digit-leading usernames | `c4af093` | Valid accounts and rejected invalid input |
| Retain SSH diagnostics after terminal restoration | `174ea78` | Real helper subprocess success/failure, exit-code preservation, bounded diagnostics |
| Resolve Ping with the selected config | `d300119` | Assert `ssh -G -F` arguments and connect to a local test listener |
| Clone hosts into a prefilled Add form | `dc29c00` | Unique aliases, independent slices, reset statistics/source, Add callback semantics |
| Import pasted SSH commands | `69c9651` | Options, quoting, IPv6 URI, continuation, proxy/remote commands, explicit rejection |
| CLI initial filter, exact connection, session exit | `09ced3e` | No initial unfiltered query, alternate aliases, conflicts, successful/failed session results |
| SCP upload/download helper | `0b31487` | Alias/config preservation, direction, paths, recursive flag, actual PowerShell quoting round trip |
| Launch system SFTP | `c854cd4` | Correct alias/config arguments and retained failure diagnostics |

Both READMEs document the new flags, keys, read-only behavior, and supported import/transfer workflows. Each implementation item was committed locally; nothing was pushed.

## Review findings fixed

- The existing form and new-host writer omitted some advanced fields. Clones now retain those fields through the actual save callback and repository round trip (`89c8eaf` and subsequent coverage).
- Preserving refresh selection must not change fuzzy search behavior. A changed query still selects the best match; mouse focus also survives UI state capture (`8cf9255`).
- TUI panics now return an error instead of reporting success, and failed sessions retain their result when exiting (`8cf9255`).
- Add/Edit save errors now return to the same form instead of discarding user input (`1586633`); tag failures are exercised through simulated form/button interaction.
- `-Y` also enables X11 forwarding; repeated boolean flags follow OpenSSH precedence while repeated ports retain the first value. Checked against the installed OpenSSH using an isolated empty config (`064b95f`, `75a8d45`).
- Local SCP `~` paths are expanded locally while remote `~` remains remote (`064b95f`).
- Capturing stderr introduces a pipe that a surviving proxy child could keep open. A bounded wait prevents hanging after SSH exits; a subprocess regression covers inherited stderr handles (`9e2b880`).
- Existing HOME-dependent tests now isolate both HOME and USERPROFILE on Windows and Unix (`7976684`).

## Final validation

- `go test ./... -count=1`: passed.
- `go vet ./...`: passed.
- Repository-configured golangci-lint v1.64.2, built with Go 1.24.6: passed.
- `git diff --check`: passed.
- `go build -trimpath ./cmd` for linux/amd64, darwin/amd64, and windows/amd64: passed; outputs are in a temporary directory.
- Built CLI smoke with an isolated home and a helper SSH executable: exact secondary alias, custom config path containing spaces, read-only connection, unchanged SSH config, unknown-alias rejection, help flags, and SSH diagnostic/exit code 255 all passed.

## Validation limits and retained boundaries

- `go test -race ./...` could not execute: this Windows environment has CGO disabled and no C compiler. Ordinary tests, subprocess tests, static analysis, and builds passed; no race-detector result is claimed.
- No production server was contacted and no real remote file transfer was performed. Interactive behavior uses simulated terminals and controlled subprocesses.
- Managed mode changes only the writable file. Earlier root/global rules intentionally retain OpenSSH precedence; the READMEs explain Include ordering.
- Import supports the existing editable SSH settings. Unsupported flags, including an independent `-F` configuration, are rejected explicitly.
- SCP generation copies a command for review; system OpenSSH/SFTP remain responsible for actual authentication and transport.
