# SSH maintenance and workflow improvements Implementation Plan

> Execution: implement each task sequentially in this session, verify it, and commit it locally as requested.

**Goal:** Complete the two approved batches of fixes and workflow features, then review the complete diff for regressions.

**Architecture:** Keep the existing repository/service/TUI boundaries and system OpenSSH transport. Normalize parser output at one boundary, keep metadata-only changes out of SSH configuration writes, and reuse the existing server form for creation workflows. Generated transfer commands must preserve the selected alias and root configuration path.

**Tech Stack:** Go 1.24, tview/tcell, Cobra, system OpenSSH.

---

For every task: add relevant regression coverage, run the targeted tests, implement the smallest complete change, and commit. Do not push.

1. Preserve negated Host patterns in both decode paths (`config_io.go`, `include.go`); verify save round trips and include listing.
2. Insert new hosts before matching wildcard blocks (`ssh_config_file_repo.go`, `crud.go`); verify ordinary and managed configuration ordering without changing root defaults.
3. Preserve search input after mouse focus (`handlers.go`); verify real tcell mouse focus and printable shortcuts.
4. Save tags as metadata with explicit errors and preserve refresh selection (`ports`, repository/service, `handlers.go`, `server_list.go`); verify failures, tag clearing, and unchanged SSH config.
5. Permit digit-leading usernames (`validation.go`); preserve rejection of invalid characters.
6. Capture bounded SSH stderr while preserving interactive IO, and show failures after terminal restoration (`server_service.go`, `handlers.go`); verify failed and successful subprocesses.
7. Use the selected config for Ping resolution (`server_service.go`); verify subprocess arguments without relying on the user's config or network.
8. Clone selected hosts into the Add form (`server_form.go`, `handlers.go`); verify unique aliases, deep copies, reset metadata, and managed/readonly behavior.
9. Parse pasted SSH commands into the Add form (new parser and UI handler); verify supported options/quoting, reject unsupported or ambiguous input explicitly, never execute pasted content.
10. Add CLI initial filter, exact alias connection, and exit-after-session options (`cmd/main.go`, TUI configuration); verify argument validation, filtered initial display, aliases, and exit codes.
11. Add SCP upload/download command builder and copy form; verify alias/config preservation, quoting, paths, and read-only use.
12. Update both READMEs for new flags and keys; fix Windows HOME test isolation as needed for trustworthy full regression checks.
13. Review the complete baseline-to-HEAD diff, fix findings in separate commits, and run full tests, vet, lint/build, cross-platform compilation, and simulated UI/CLI smoke checks.

**Validation commands:** `go test ./...`, `go vet ./...`, repository lint/build checks from `makefile`; cross-compile Linux/macOS/Windows binaries into a temporary directory. Record actual results in the final review note.
