# Review fixes Implementation Plan

**Goal:** Fix all six regressions identified in the static review of `cee1c46..7692b63`, with local commits and no race tests.

**Architecture:** Preserve system OpenSSH execution and the existing form/repository boundaries. Select SCP protocol from client capabilities, keep proxy order explicit, and preserve the distinction between command-line text and SSH configuration values.

**Tech Stack:** Go 1.24, OpenSSH, tview/tcell.

1. SCP: detect supported protocol flags once when opening the helper. Force SFTP when supported; otherwise quote the remote shell operand separately from the local shell. Test both protocols, metacharacters, spaces, apostrophes, and remote home paths.
2. Proxy order: initialize new Host proxy directives in deterministic order before the shared updater. Verify serialized order and effective OpenSSH selection.
3. Remote command: escape literal percent signs only when converting positional command text to RemoteCommand; preserve explicit `-o RemoteCommand` templates. Test form/config round trips and reject conflicting commands.
4. Windows paths: use platform-appropriate backslash handling while preserving POSIX parsing. Exercise both modes in tests.
5. Destination precedence: apply destination user/port when the destination is encountered, while allowing later HostName options. Cover aliases, URIs, options before/after the destination, and `--`.
6. Comments: recognize unquoted word-initial `#`, preserve quoted/literal hashes, and continue rejecting multiple commands.

For each item: add regression tests, implement, run targeted checks, and commit locally. Finish with full ordinary tests, vet, lint, cross-platform builds, and a review of the final diff. Do not push or run race tests.
