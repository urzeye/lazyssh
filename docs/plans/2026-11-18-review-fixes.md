# Review fixes Implementation Plan

**Goal:** Fix all six regressions identified in the static review of `cee1c46..7692b63`, with local commits and no race tests.

**Architecture:** Preserve system OpenSSH execution and the existing form/repository boundaries. Select SCP protocol from client capabilities, keep proxy order explicit, and preserve the distinction between command-line text and SSH configuration values.

**Tech Stack:** Go 1.24, OpenSSH, tview/tcell.

1. SCP: detect supported protocol flags once when opening the helper. Force SFTP when supported; otherwise quote the remote shell operand separately from the local shell. Test both protocols, metacharacters, spaces, apostrophes, and remote home paths.
2. Proxy order: keep mutually exclusive proxy updates outside the unordered scalar map. Preserve existing directive order, insert new pairs deterministically, and retain effective command-line proxy selection on import. Verify serialized order and OpenSSH selection.
3. Remote command: escape literal percent signs only when converting positional command text to RemoteCommand; preserve explicit `-o RemoteCommand` templates. Test form/config round trips and reject conflicting commands.
4. Windows paths: use platform-appropriate backslash handling while preserving POSIX parsing. Exercise both modes in tests.
5. Destination precedence: apply destination user/port when the destination is encountered, while allowing later HostName options. Cover aliases, URIs, options before/after the destination, and `--`.
6. Comments: recognize unquoted word-initial `#`, preserve quoted/literal hashes, and continue rejecting multiple commands.

For each item: add regression tests, implement, run targeted checks, and commit locally. Finish with full ordinary tests, vet, lint, cross-platform builds, and a review of the final diff. Do not push or run race tests.

## Completed fixes

| Finding | Commits | Regression evidence |
| --- | --- | --- |
| Legacy SCP remote-path interpretation | `d36d60c` | Explicit SFTP flag for supporting clients; legacy remote-shell quoting; spaces, apostrophes, shell metacharacters and home paths; native PowerShell argument round trip; actual installed-client form opening |
| Nondeterministic proxy selection | `ee615b6`, `f484919`, `1c0f4ea` | Shared writer keeps existing order and inserts new pairs deterministically; OpenSSH effective proxy checked before/after import, including explicit `none` |
| Positional remote-command percent expansion | `125d22f` | Literal `%Y`/`%h`/`%` escaped through form and config storage; explicit RemoteCommand templates retained; conflicting commands rejected |
| Windows backslash paths | `ba9f56e` | Drive, quoted and UNC paths; doubled PowerShell single quotes; POSIX escape behavior tested separately |
| Destination user/port precedence | `b432615` | User, URI port and HostName ordering compared with actual OpenSSH; option termination before/after destination |
| Comments imported as commands | `cab1f54` | Unquoted comments ignored; quoted, embedded and escaped hashes retained; comments cannot hide a second command |

## Final validation

- `go test ./... -count=1`: passed.
- `go vet ./...`: passed.
- Repository golangci-lint v1.64.2 configuration: passed.
- `git diff --check`: passed.
- Final cross-builds for linux/amd64, darwin/amd64 and windows/amd64: passed.
- Final production diff reviewed against `7692b63`; both READMEs describe the corrected import and protocol behavior.
- No race tests were run, as requested. Tests use isolated configs, local shells and OpenSSH configuration dumps; no production SSH connection or remote transfer was performed.
