# Transport Regression Fixes Implementation Plan

**Goal:** Fix all four review findings and commit the verified changes locally.

**Architecture:** Preserve OpenSSH's own parsing and filename checks. Validate unsupported SessionType settings before repository writes. Keep RemoteCommand as a config template when copying. Test native argument transport in both installed PowerShell versions.

**Tech Stack:** Go, OpenSSH, POSIX sh, Windows PowerShell 5.1, PowerShell 7.

1. Add a regression using the real SCP receiver with a controlled SSH transport. Replace legacy path single quotes with backslash escapes; retain filename validation and tilde expansion. Run the targeted SCP tests and commit.
2. Add repository add/update checks for SessionType using an isolated, offline SSH config probe. Test rejection leaves both configuration and metadata unchanged, supported values persist, and ordinary saves need no probe. Commit.
3. Exercise import → form → config → copied command → native arguments → reimport. Copy RemoteCommand through `-o`, retaining literal percent signs and template tokens. Commit.
4. Test double quotes, backslashes, whitespace and shell metacharacters through native executables in PowerShell 5.1 and 7. Make Windows argument transport independent of native argument passing mode. Commit.

Run the full tests, vet, configured lint, and cross-platform builds after the targeted regressions pass. Do not push or run race tests. Record any validation limits explicitly.

## Completed validation

- `go test ./... -count=1`: passed. The real OpenSSH 8.5 SCP receiver accepts escaped paths and rejects an unrequested filename, with checks enabled.
- Loopback SSH exec requests match before copying, after copying, and after reimporting for literal percent signs and explicit `%h` templates. This optional test requires Python with Paramiko; both are available and the fixture ran here. It records commands without executing them.
- Native argv round trips passed in Windows PowerShell 5.1 and PowerShell 7 Legacy/Standard/Windows modes, including embedded quotes, backslashes, spaces, percent signs and shell metacharacters. A separate test verifies exit-code propagation and preference isolation.
- SessionType add/update rejection preserves config and metadata. Actual OpenSSH 8.5 rejects the setting before writing; Windows OpenSSH 9.5 saves it and continues to parse the existing host.
- `go vet ./...`, configured `golangci-lint run ./...`, and Linux/macOS/Windows amd64 builds: passed.

The SCP path and Windows argv fixes share the same transport boundary and are committed together with the remaining two fixes. No remote host was contacted. SCP receiver testing uses a controlled SSH transport; the Windows 9.5 client selects SFTP in the product, so its unused legacy mode is not forced in this test. Linux/macOS were build-checked, not run. Race tests and pushing are excluded as requested.
