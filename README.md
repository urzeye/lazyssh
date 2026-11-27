<div align="center">
  <img src="./docs/logo.png" alt="lazyssh logo" width="160" />
  <h1>lazyssh</h1>
  <p>A keyboard-driven terminal UI for managing OpenSSH configs and connections.</p>
  <p>English | <a href="README.zh-CN.md">简体中文</a></p>
  <p>
    <a href="https://github.com/urzeye/lazyssh/releases"><img src="https://img.shields.io/github/v/release/urzeye/lazyssh?style=flat-square&color=blue" alt="GitHub release" /></a>
    <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/urzeye/lazyssh?style=flat-square&color=00ADD8" alt="Go Version" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/urzeye/lazyssh?style=flat-square&color=green" alt="License" /></a>
    <a href="https://github.com/urzeye/lazyssh/actions/workflows/go.yml"><img src="https://img.shields.io/github/actions/workflow/status/urzeye/lazyssh/go.yml?branch=main&style=flat-square" alt="Build Status" /></a>
    <a href="https://github.com/urzeye/lazyssh/releases"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey?style=flat-square" alt="Platform" /></a>
  </p>
</div>

> [!NOTE]
> This repository is an independently maintained fork of [Adembc/lazyssh](https://github.com/Adembc/lazyssh). The upstream repository is no longer actively maintained; this fork continues development with support for multi-file `Include` trees, dotfiles-friendly managed writes, safer edits, and daily SSH/SFTP/SCP workflows. Credit for the original design and codebase goes to [Adembc](https://github.com/Adembc).

---

## Features

### Multi-file SSH Config & Dotfiles Support
- **Recursive `Include` parsing**: Reads full OpenSSH `Include` hierarchies (including `~`, relative paths, and globs such as `~/.ssh/config.d/*.conf` or OrbStack/Colima configs) in OpenSSH evaluation order, and safely skips `Match` blocks.
- **Effective config resolution**: Displays the effective parameters a host inherits from `Host *` and wildcard patterns (e.g. `Host lab-*`), matching OpenSSH first-match rules.
- **Managed mode (`--managed-mode`)**: Reads the full config tree from your root config while writing all additions, edits, and deletions only to a dedicated local file (default `~/.ssh/config.local`), leaving version-controlled dotfiles untouched.
- **Origin indicators & read-only protection**: Writable hosts are marked with `🏠`; read-only hosts (from included/unmanaged files or `--ssh-config-readonly`) are marked with `🔗` and protected against edits and deletion.
- **Safe config writes**:
  - Writes to a temporary file in the target directory and atomically replaces the config via `os.Rename` using `0600` permissions.
  - Preserves symlinks by writing directly to the resolved target file.
  - Creates a one-time baseline backup (`<name>.original.backup`) before the first edit and rolling millisecond-timestamped backups (`<name>-<unix-ms>-lazyssh.backup`), retaining the 10 most recent backups.
  - Inserts newly created hosts before matching wildcard blocks so specific host rules take precedence over defaults.
  - Normalizes `IdentityFile` paths inside `$HOME` to portable `~/.ssh/...` notation and preserves quoted host patterns and negated patterns (`Host !foo`).
- **Separate metadata storage**: Tags, pins, SSH counts, and last-seen timestamps live in `~/.lazyssh/metadata.json`; sort/theme preferences live in `~/.lazyssh/settings.json`; logs are written to `~/.lazyssh/lazyssh.log`. Editing tags never rewrites your SSH config.

### Terminal Workflow
- **Fuzzy search & quick focus**: Search across alias, hostname/IP, user, and tags with relevance ranking. Jump directly to Search (`0` / `/`), Server List (`1`), or Details (`2`), and scroll with `j`/`k`.
- **Direct SSH, SFTP & SCP helpers**:
  - `Enter` launches the system `ssh` client and keeps error diagnostics if the connection fails.
  - `F` launches the system `sftp` client for the selected host using the active `-F` root config.
  - `P` opens an interactive SCP upload/download command builder with live preview and clipboard copy.
  - `c` copies a standalone `ssh` command; `h` copies the host address (`HostName`/IP, falling back to alias).
- **Host creation helpers**:
  - `a` opens a 5-tab editor (`Basic`, `Connection`, `Forwarding`, `Authentication`, `Advanced`) with contextual field help and local key / `known_hosts` autocomplete.
  - `C` clones the selected host into a prefilled Add form with a unique alias and cleared stats.
  - `v` parses a pasted `ssh` command from the clipboard into the Add form without executing anything.
- **Port forwarding, ping & active sessions**:
  - `f` / `x` starts Local (`-L`), Remote (`-R`), or Dynamic SOCKS5 (`-D`) tunnels (forward-only or forward+SSH) and stops background tunnels started by lazyssh.
  - `g` resolves the host with `ssh -G -F <config>` and measures TCP reachability.
  - A read-only **Active Sessions** panel auto-refreshes every 10 seconds to show running local `ssh` client processes.

---

## Screenshots

<details>
<summary>Show screenshots</summary>
<br>

**Host list, details & active sessions**

<img src="./docs/list server.png" alt="Host list view and active sessions" width="880" />

**Fuzzy search**

<img src="./docs/search.png" alt="Fuzzy host search" width="880" />

**Tabbed host editor**

<img src="./docs/add server.png" alt="Tabbed host editor" width="880" />

**SSH connection**

<img src="./docs/ssh.png" alt="SSH connection" width="880" />

</details>

---

## Installation

### Option 1: mise (Recommended)

```bash
mise use -g github:urzeye/lazyssh@latest
lazyssh --version
```

Pin a specific release:

```bash
mise use -g github:urzeye/lazyssh@v2026.11.18
```

### Option 2: Pre-compiled Binaries

Download an archive from [GitHub Releases](https://github.com/urzeye/lazyssh/releases) and place the `lazyssh` binary in your `PATH`:

| OS | Architectures | Format |
|---|---|---|
| Linux | `x86_64` (amd64), `arm64`, `i386`, `armv6` | `.tar.gz` |
| macOS | `arm64`, `x86_64` (amd64) | `.tar.gz` |
| Windows | `x86_64` (amd64), `arm64`, `i386` | `.zip` |

### Option 3: Build from Source

Requires Go 1.24.6+.

```bash
git clone https://github.com/urzeye/lazyssh.git
cd lazyssh

# Build ./bin/lazyssh
make build

# Or install to GOBIN
make install
```

---

## CLI Usage

```bash
# Open the TUI (default config: ~/.ssh/config)
lazyssh

# Start with an initial fuzzy filter
lazyssh prod
lazyssh --filter dev

# Connect directly to an exact configured alias (bypasses TUI, preserves SSH exit code)
lazyssh -c production-server

# Exit lazyssh automatically after the SSH/SFTP session finishes
lazyssh --exit-after-session

# Use a custom SSH config file
lazyssh --sshconfig ~/.ssh/work_config -c bastion

# Read-only mode (disables config and tag edits)
lazyssh --ssh-config-readonly
LAZYSSH_SSH_CONFIG_READONLY=1 lazyssh
```

### Flags

| Flag | Env / Default | Description |
|---|---|---|
| `[filter]` / `-f, --filter` | `""` | Start the TUI with an initial fuzzy filter (cannot combine with `--connect`) |
| `-c, --connect` | `""` | Connect directly to an exact host alias (including secondary aliases) without opening the TUI |
| `--exit-after-session` | `false` | Exit after the selected SSH or SFTP session finishes |
| `--sshconfig` | `~/.ssh/config` | Root SSH config file to read and pass to `ssh`/`sftp`/`scp` via `-F` |
| `--managed-mode` | `false` | Read from root config, write mutations only to the managed config file |
| `--managed-sshconfig` | `~/.ssh/config.local` | Path to the writable managed config file (setting this implicitly enables managed mode) |
| `--ssh-config-readonly` | `LAZYSSH_SSH_CONFIG_READONLY` | Disable all SSH config and tag edits (browsing, connect, SCP, SFTP, forwarding, ping, and pinning remain available) |
| `-v, --version` | — | Print version and commit hash |

---

## Managed Mode (Dotfiles Workflow)

If `~/.ssh/config` is tracked in a dotfiles repo, keep it clean by including a local writable file before your shared defaults:

```sshconfig
# ~/.ssh/config (tracked in dotfiles)
Include ~/.orbstack/ssh/config
Include ~/.ssh/config.d/*.conf
Include ~/.ssh/config.local

Host *
    ServerAliveInterval 60
    TCPKeepAlive yes
```

> [!TIP]
> Place `Include ~/.ssh/config.local` **before** any `Host *` default blocks you want local hosts to override, and **outside** specific `Host <name>` blocks (OpenSSH treats an `Include` inside a `Host` block as conditional on that host).

Run lazyssh in managed mode:

```bash
lazyssh \
  --sshconfig ~/.ssh/config \
  --managed-mode \
  --managed-sshconfig ~/.ssh/config.local
```

In this mode:
- All hosts across the root file and all `Include` files are listed with their effective inherited values.
- Only hosts defined in `~/.ssh/config.local` are editable (`🏠`); hosts from the root config or other includes are read-only (`🔗`).
- Adding, editing, or deleting hosts modifies only `~/.ssh/config.local`.
- Connections still invoke OpenSSH with `-F ~/.ssh/config`, preserving identical behavior to running `ssh` directly.

---

## Import, SCP & SFTP Notes

- **Paste SSH command (`v`)**: Parses the clipboard locally without executing shell expansions or commands. Supports common OpenSSH flags (`-p`, `-i`, `-J`, `-L/-R/-D`, `-l`, `-b`, `-B`, `-e`, `-c`, `-m`, `-A/-a`, `-X/-x/-Y`, `-C`, `-4/-6`, `-T/-t`, `-N`, `-s`, `-q/-v`, `-o Key=Value`) and `ssh://` URIs. Explicitly rejects unsupported flags (including `-F`), shell operators (`;`, `|`, `&`, `<`, `>`, `` ` ``), and URI passwords. Positional remote commands have literal `%` signs escaped so OpenSSH does not expand them as config tokens; `-o RemoteCommand=...` keeps template tokens as-is. Saving `SessionType` (including imported `-N`/`-s`) probes the installed `ssh` client offline first and requires OpenSSH 8.7+.
- **SCP command helper (`P`)**: Generates a ready-to-copy command preserving the selected alias and root `-F` config path; it does not start a transfer automatically. Inspects the local `scp` binary once to use `-s` (SFTP mode) when supported, or safely escaped remote paths on legacy clients. Local `~` expands locally while remote `~` is preserved for the server. Available in read-only mode.
- **System SFTP (`F`)**: Runs `sftp -F <config> -- <alias>` and returns to lazyssh when finished. Available in read-only mode.

---

## Keyboard Shortcuts

### Navigation & Focus
| Key | Action |
|---|---|
| `0` / `/` | Open and focus the search bar |
| `Esc` / `Enter` *(in search bar)* | Clear and collapse the search bar |
| `↑` / `↓` *(in search bar)* | Return to the server list and move selection |
| `1` | Focus server list |
| `2` | Focus server details panel |
| `↑` `↓` / `j` `k` | Navigate server list, or scroll details when Details is focused |
| `Enter` | Connect via SSH to the selected host |
| `q` | Quit |

### Host Management
| Key | Action |
|---|---|
| `a` | Add a new host |
| `C` | Clone selected host into a prefilled Add form |
| `v` | Import SSH command from clipboard into Add form |
| `e` | Edit selected host (disabled for read-only hosts) |
| `d` | Delete selected host (with confirmation) |
| `t` | Edit tags (writes metadata only; disabled for read-only hosts) |
| `p` | Pin / unpin selected host |

### Transfer, Forwarding & Utilities
| Key | Action |
|---|---|
| `c` | Copy full `ssh` command to clipboard |
| `h` | Copy host address (`HostName`/IP, or alias if unset) to clipboard |
| `P` | Open SCP upload/download command helper |
| `F` | Launch system `sftp` for selected host |
| `f` | Open port forwarding wizard (Local / Remote / Dynamic) |
| `x` | Stop background port forwards started by lazyssh for selected host |
| `g` | Ping selected host (TCP connect using resolved SSH target) |
| `r` | Refresh host list and active sessions |

### View & Preferences
| Key | Action |
|---|---|
| `s` | Switch sort field (`Alias` ↔ `Last SSH`) |
| `S` | Reverse sort direction |
| `T` | Toggle Dark / Light theme |

### Inside Host Editor Forms
| Key | Action |
|---|---|
| `Ctrl+←` / `Ctrl+→` | Previous / Next tab (works everywhere in the form) |
| `Ctrl+H` / `Ctrl+L` | Previous / Next tab (`Ctrl+H` acts as Backspace inside text inputs) |
| `Ctrl+S` | Save host configuration |
| `Esc` | Cancel / Return to main view |

---

## Security

- **No credential storage**: lazyssh never stores or transmits private keys, passphrases, or passwords.
- **System OpenSSH delegation**: SSH, SFTP, and port forwarding invoke the local `ssh` and `sftp` binaries directly, so `ssh-agent`, 1Password SSH Agent, and hardware keys work normally.
- **Safe file permissions**: Config files are written atomically with `0600` permissions.

---

## Contributing

```bash
git clone https://github.com/urzeye/lazyssh.git
cd lazyssh

make test
make lint
```

PR titles follow Conventional Commits (`fix`, `feat`, `improve`, `refactor`, `revert`, `test`, `ci`, `docs`, `chore`; optional scopes: `ui`, `cli`, `config`, `parser`).

---

## License & Credits

Licensed under the [Apache License 2.0](LICENSE).

- [Adembc](https://github.com/Adembc) — original author of [`lazyssh`](https://github.com/Adembc/lazyssh)
- [rivo/tview](https://github.com/rivo/tview) & [gdamore/tcell](https://github.com/gdamore/tcell) — terminal UI framework
- [derailed/k9s](https://github.com/derailed/k9s) & [jesseduffield/lazydocker](https://github.com/jesseduffield/lazydocker) — workflow inspiration
