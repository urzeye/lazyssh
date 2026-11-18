<div align="center">
  <img src="./docs/logo.png" alt="lazyssh logo" width="180" />
  <h1>⚡ lazyssh</h1>
  <p><b>A modern, keyboard-first terminal SSH manager designed for real-world SSH setups</b></p>
  <p><i>Recursive OpenSSH <code>Include</code> • Dotfiles-friendly Managed Mode • Non-destructive Safe Writes • Instant Terminal Productivity</i></p>

  <p>
    English | <a href="README.zh-CN.md">简体中文</a>
  </p>

  <div>
    <a href="https://github.com/urzeye/lazyssh/releases"><img src="https://img.shields.io/github/v/release/urzeye/lazyssh?style=flat-square&color=blue" alt="GitHub release" /></a>
    <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/urzeye/lazyssh?style=flat-square&color=00ADD8" alt="Go Version" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/urzeye/lazyssh?style=flat-square&color=green" alt="License" /></a>
    <a href="https://github.com/urzeye/lazyssh/actions/workflows/go.yml"><img src="https://img.shields.io/github/actions/workflow/status/urzeye/lazyssh/go.yml?branch=main&style=flat-square" alt="Build Status" /></a>
    <a href="https://github.com/urzeye/lazyssh/releases"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey?style=flat-square" alt="Platform" /></a>
  </div>
</div>

---

> [!NOTE]
> **lazyssh is an independent, actively maintained continuation and enhanced fork of [Adembc/lazyssh](https://github.com/Adembc/lazyssh).**
> The upstream repository is no longer actively maintained. This fork addresses real-world SSH workflows: multi-file `Include` configs, dotfiles synchronisation, managed write isolation, cascading pattern resolution, and rock-solid terminal ergonomics.
> All foundational credit and appreciation go to the original creator, **[Adembc](https://github.com/Adembc)**.

---

## 📑 Table of Contents

- [💡 About lazyssh](#-about-lazyssh)
- [🌟 Key Highlights](#-key-highlights)
- [📷 Screenshots](#-screenshots)
- [📦 Installation](#-installation)
  - [Option 1: mise (Recommended)](#option-1-mise-recommended)
  - [Option 2: Go Install](#option-2-go-install)
  - [Option 3: Pre-compiled Binaries](#option-3-pre-compiled-binaries)
  - [Option 4: Build from Source](#option-4-build-from-source)
- [⚡ Quick Start & CLI Workflows](#-quick-start--cli-workflows)
- [🧭 Deep Dive: Dotfiles & Managed Mode](#-deep-dive-dotfiles--managed-mode)
- [🛡️ Safe Writes & Automatic Backups](#-safe-writes--automatic-backups)
- [⌨️ Keyboard Shortcuts Reference](#️-keyboard-shortcuts-reference)
- [🔐 Security Architecture](#-security-architecture)
- [🤝 Contributing](#-contributing)
- [📄 License & Attribution](#-license--attribution)

---

## 💡 About lazyssh

**lazyssh** is an interactive, keyboard-driven SSH manager for your terminal, inspired by tools like `lazydocker` and `k9s` but purpose-built for OpenSSH workflows.

Managing dozens of servers across cloud providers, home labs, jump hosts, and client environments often leaves developers drowning in forgotten IP addresses, complicated proxy flags, and fragile `~/.ssh/config` files. Most terminal pickers assume a simple flat file and overwrite custom comments or break symlinked dotfiles.

**lazyssh solves this comprehensively:**
- Recursively parses your entire OpenSSH `Include` hierarchy (e.g. `conf.d/`, OrbStack, Colima).
- Isolates user edits into a dedicated managed file so your version-controlled root dotfiles stay untouched.
- Computes cascading wildcards (`Host *`, `Host lab-*`) to display the true, effective connection configuration.
- Operates strictly with non-destructive atomic writes and rolling backups.
- Provides a lighting-fast keyboard navigation experience with fuzzy search, panel jumps, SCP command generation, SFTP launching, and port forwarding.

---

## 🌟 Key Highlights

### 📁 Advanced Config & Dotfiles Architecture
- **Recursive `Include` Support**: Traverses full OpenSSH `Include` trees (including wildcards like `~/.ssh/conf.d/*.conf`, OrbStack, etc.) instead of assuming a single flat config file.
- **Managed Mode (`--managed-mode`)**: Reads the complete config tree from your root config, but isolates all additions, updates, and deletions to a dedicated managed file (such as `~/.ssh/config.local`).
- **Read-Only Protection**: Hosts originating from included or external files are visually tagged with `[inc]` and protected against accidental edits. A dedicated `--ssh-config-readonly` flag provides safe browsing mode.
- **Effective Values Resolution**: Automatically computes and displays cascading OpenSSH directives (e.g., global `Host *` defaults, pattern-based `Host dev-*` rules, root inheritance) so you see the exact parameters OpenSSH will use.
- **Safe Host Ordering**: Automatically places newly created hosts before matching wildcard rules to guarantee OpenSSH first-match precedence.
- **Full Syntax Fidelity**: Robust handling of quoted host entries (`Host "example-name"`), multi-alias lines, and negated patterns (`Host !foo bar-*`).

### 🛡️ Non-Destructive & Resilient File Writes
- **Atomic File Swapping**: Configuration updates are written to temporary files and atomically renamed to prevent file corruption during sudden interruptions.
- **Symlink Preservation**: Modifies symlink targets in-place instead of replacing symlinks with regular files, keeping dotfile repositories intact.
- **Dual-Layer Backups**: Creates a baseline `.original.backup` before the very first modification, plus rolling timestamped backups (`config.local-YYYYMMDD-HHMMSS-lazyssh.backup`) with automatic 10-version rotation.
- **Portable Identity Paths**: Normalizes private key paths under `$HOME` to portable `~/.ssh/...` formats for seamless multi-machine dotfile sharing.
- **Metadata Separation**: Tags, pinned states, and connection statistics are saved to `~/.lazyssh/metadata.json` without dirtying or rewriting your SSH config files.

### ⌨️ Terminal UX & Keyboard Ergonomics
- **Fast Fuzzy Search**: Multi-field fuzzy search ranks hosts by relevance across alias, hostname, user, and tags.
- **Numbered Panel Switching**: Instantly jump between Search (<kbd>0</kbd> or <kbd>/</kbd>), Host List (<kbd>1</kbd>), and Details (<kbd>2</kbd>).
- **Collapsible Search Bar**: Press <kbd>/</kbd> to show/hide the search bar and maximize vertical terminal space.
- **Persisted Sorting & Themes**: Toggle sorting by Alias or Last Connection (<kbd>s</kbd>), reverse order (<kbd>S</kbd>), and toggle Dark/Light themes (<kbd>T</kbd>) with persisted preferences.
- **Mouse & Terminal Resiliency**: Safe mouse-click handling ensures <kbd>j</kbd>/<kbd>k</kbd> navigation, focus state, and input backspaces remain completely reliable.

### 🚀 Connection, Forwarding & Transfer Helpers
- **Instant Connection**: Hit <kbd>Enter</kbd> to launch OpenSSH. Diagnostic error output is retained if a connection fails.
- **Clone Host (<kbd>C</kbd>)**: Duplicate any existing host configuration into an Add form with an auto-incremented alias and clean stats.
- **Import from Clipboard (<kbd>v</kbd>)**: Paste a raw `ssh user@host -p 2222 -i ...` command line directly into a structured form with automatic OpenSSH flag parsing.
- **SCP Command Helper (<kbd>P</kbd>)**: Interactive upload/download SCP command builder with automatic client capability detection (OpenSSH `-s` SFTP protocol vs legacy quoting, POSIX vs Windows PowerShell quoting) and 1-click clipboard copy.
- **System SFTP Integration (<kbd>F</kbd>)**: Launch the system `sftp` client directly for the selected host and cleanly return to lazyssh upon exit.
- **Port Forwarding Wizard (<kbd>f</kbd> / <kbd>x</kbd>)**: Setup Local (`-L`), Remote (`-R`), or Dynamic SOCKS5 (`-D`) tunnels with forward-only or forward+SSH modes, and stop active tunnels with one keypress.
- **Active Session Monitor**: Auto-refreshing read-only panel displaying currently active local SSH processes.
- **Ping Health Check (<kbd>g</kbd>)**: Check host reachability on demand using actual configured target parameters.

---

## 📷 Screenshots

<details open>
<summary><b>Click to expand / collapse screenshots</b></summary>
<br>

<div align="center">

### 📋 Main Host Management & Active Sessions
<img src="./docs/list server.png" alt="Host list view and active sessions" width="900" />
<p><i>Browse configured hosts, inspect real-time effective SSH options, monitor active local sessions, and view source config markers.</i></p>

---

### 🔎 Intelligent Fuzzy Search
<img src="./docs/search.png" alt="Fuzzy host search" width="900" />
<p><i>Search across aliases, hostnames, usernames, and tags with smart relevance ranking.</i></p>

---

### ➕ Tabbed Host Editor (5 Categories)
<img src="./docs/add server.png" alt="Tabbed host editor" width="900" />
<p><i>Full control over OpenSSH options organized into <b>Basic</b>, <b>Connection</b>, <b>Forwarding</b>, <b>Authentication</b>, and <b>Advanced</b> tabs.</i></p>

---

### 🔐 Instant SSH Connection & SCP Helper
<img src="./docs/ssh.png" alt="SSH connection and command helpers" width="900" />
<p><i>Launch direct sessions, build robust SCP commands, or open SFTP file management with zero friction.</i></p>

---

### 🚀 Startup Loader
<img src="./docs/loader.png" alt="Startup screen" width="800" />
<p><i>Clean, asynchronous startup loading your complete Include hierarchy.</i></p>

</div>

</details>

---

## 📦 Installation

### Option 1: mise (Recommended)

If you use [mise-en-place](https://mise.jdx.dev/), install lazyssh directly from GitHub Releases:

```bash
# Install and activate globally
mise use -g github:urzeye/lazyssh@latest

# Verify installation
lazyssh --version
```

To pin a specific version:
```bash
mise use -g github:urzeye/lazyssh@v2026.6.4
```

### Option 2: Go Install

If you have Go installed (Go 1.22+):

```bash
go install github.com/urzeye/lazyssh/cmd@latest
```

Ensure your `$GOPATH/bin` or `$HOME/go/bin` is in your `$PATH`.

### Option 3: Pre-compiled Binaries

Download pre-compiled binaries for your operating system and architecture from the [GitHub Releases](https://github.com/urzeye/lazyssh/releases) page:

| OS | Architecture | Archive Format |
|---|---|---|
| **Linux** | `x86_64` (amd64), `aarch64` (arm64), `i386`, `armv6` | `.tar.gz` |
| **macOS** | `Apple Silicon` (arm64), `Intel` (amd64) | `.tar.gz` |
| **Windows** | `x86_64` (amd64), `arm64`, `i386` | `.zip` |

Unpack the archive and move the `lazyssh` binary into your system `PATH` (e.g. `/usr/local/bin` on Linux/macOS).

### Option 4: Build from Source

```bash
# Clone repository
git clone https://github.com/urzeye/lazyssh.git
cd lazyssh

# Build binary into ./bin/lazyssh
make build

# Or run directly
make run
```

---

## ⚡ Quick Start & CLI Workflows

```bash
# 1. Open the interactive TUI (defaults to ~/.ssh/config)
lazyssh

# 2. Launch directly with a fuzzy search filter
lazyssh prod
lazyssh --filter dev

# 3. Headless direct connect (bypasses the TUI, preserves SSH exit codes)
lazyssh -c production-server

# 4. Connect and automatically exit the TUI when the session ends
lazyssh --exit-after-session

# 5. Combine custom config file with headless connection
lazyssh --sshconfig ~/.ssh/work_config -c bastion

# 6. Safe read-only mode (prevents any config modifications)
lazyssh --ssh-config-readonly
# or via environment variable:
export LAZYSSH_SSH_CONFIG_READONLY=1
lazyssh
```

---

## 🧭 Deep Dive: Dotfiles & Managed Mode

### The Problem
Power users often maintain their primary SSH configurations inside Git-tracked dotfile repositories (e.g. `~/.ssh/config` is a symlink or contains shared company defaults). Using traditional SSH GUIs or TUIs can overwrite comments, break symlinks, or pollute version control with ad-hoc test hosts.

### The Solution: Managed Mode
With lazyssh's **Managed Mode**, you can read from your root config hierarchy while directing all writes to a dedicated local file:

```sshconfig
# ~/.ssh/config (Tracked in dotfiles git repo)
Include ~/.orbstack/ssh/config
Include ~/.ssh/config.d/*.conf
Include ~/.ssh/config.local      # <-- lazyssh writes exclusively here!

Host *
    ServerAliveInterval 60
    TCPKeepAlive yes
```

Run lazyssh with managed mode enabled:

```bash
lazyssh \
  --sshconfig ~/.ssh/config \
  --managed-mode \
  --managed-sshconfig ~/.ssh/config.local
```

```
~/.ssh/config (Root Config / Dotfiles Repo)
 ├── Include ~/.orbstack/ssh/config  [Read-Only: marked with [inc]]
 ├── Include ~/.ssh/config.d/*.conf  [Read-Only: marked with [inc]]
 └── Include ~/.ssh/config.local     [Writable Target: adds/edits land here]
```

#### What Managed Mode Guarantees:
1. **Full Visibility**: All hosts across all included files are parsed and listed.
2. **Safe Isolation**: Adding a host, editing, or deleting only modifies `~/.ssh/config.local`.
3. **Accidental Edit Guard**: Attempting to edit or delete a host defined in an external or root file is blocked with a clear warning indicating its source file.
4. **Cascading Inheritance**: Global settings from `Host *` in `~/.ssh/config` cascade seamlessly into the UI's effective value display.
5. **OpenSSH Consistency**: Connections always use `--sshconfig ~/.ssh/config`, ensuring 100% fidelity with native `ssh` command behavior.

---

## 🛡️ Safe Writes & Automatic Backups

lazyssh treats your SSH configuration with surgical precision:

- **Atomic Saves**: Config changes are written to a temporary sibling file first and then atomically renamed (`os.Rename`), eliminating partial-write corruptions.
- **Symlink Protection**: If your config file is a symlink, lazyssh resolves the target path and updates the destination file directly, preserving the symlink untouched.
- **Baseline Backup**: Before the very first modification to a config file, a sibling `.original.backup` is created (e.g. `config.original.backup`).
- **Rolling Backups**: Subsequent writes automatically generate timestamped backups (e.g. `config-20261118-190516-lazyssh.backup`). Up to **10 rolling backups** are retained automatically with older files pruned.
- **Portable IdentityFile Normalization**: Keys located inside the user's home directory are normalized to `~/.ssh/...` relative syntax, avoiding machine-specific absolute path issues when syncing configs across devices.
- **Minimal Diff Updates**: Only targeted host blocks are touched; your blank lines, indentation, comments, and untouched directives remain intact.

---

## ⌨️ Keyboard Shortcuts Reference

### Navigation & Focus
| Key | Action |
|:---:|---|
| <kbd>0</kbd> / <kbd>/</kbd> | Focus / Toggle collapsible search bar |
| <kbd>1</kbd> | Focus server list |
| <kbd>2</kbd> | Focus server details panel |
| <kbd>↑</kbd> <kbd>↓</kbd> / <kbd>j</kbd> <kbd>k</kbd> | Navigate through server list |
| <kbd>Enter</kbd> | Connect via SSH to the selected server |
| <kbd>q</kbd> | Quit lazyssh |

### Server Management
| Key | Action |
|:---:|---|
| <kbd>a</kbd> | Add a new server |
| <kbd>C</kbd> | Clone selected server into a prefilled Add form |
| <kbd>v</kbd> | Import SSH command from clipboard into Add form |
| <kbd>e</kbd> | Edit selected server (disabled for read-only / included hosts) |
| <kbd>d</kbd> | Delete selected server with confirmation dialog |
| <kbd>t</kbd> | Edit server tags (saved to metadata without rewriting config) |
| <kbd>p</kbd> | Pin / unpin server to the top of the list |

### Command Generation, SFTP & Tunnels
| Key | Action |
|:---:|---|
| <kbd>c</kbd> | Copy SSH connection command to system clipboard |
| <kbd>h</kbd> | Copy server host alias to system clipboard |
| <kbd>P</kbd> | Open interactive SCP command generator (upload / download) |
| <kbd>F</kbd> | Launch system `sftp` client for selected server |
| <kbd>f</kbd> | Open interactive Port Forwarding wizard (Local, Remote, Dynamic) |
| <kbd>x</kbd> | Stop active port forwarding tunnel for selected server |
| <kbd>g</kbd> | Ping selected server to test reachability |
| <kbd>r</kbd> | Refresh background data (servers and active sessions) |

### View & Preferences
| Key | Action |
|:---:|---|
| <kbd>s</kbd> | Switch sort field (Alias ↔ Last Connection Time) |
| <kbd>S</kbd> | Reverse current sort direction (Ascending ↔ Descending) |
| <kbd>T</kbd> | Toggle Color Theme (Dark ↔ Light) with persisted preference |

### Inside Host Editor Forms
| Key | Action |
|:---:|---|
| <kbd>Ctrl+H</kbd> / <kbd>Ctrl+←</kbd> | Switch to Previous tab |
| <kbd>Ctrl+L</kbd> / <kbd>Ctrl+→</kbd> | Switch to Next tab |
| <kbd>Ctrl+S</kbd> | Save host configuration |
| <kbd>Esc</kbd> | Cancel / Return to main view |

---

## 🔐 Security Architecture

lazyssh is designed with strict security boundaries:

- **Zero Credential Storage**: lazyssh never stores, inspects, or transmits private keys, passphrases, or passwords.
- **OpenSSH Passthrough**: All SSH connections, port tunnels, and SFTP sessions are executed by delegating directly to the system OpenSSH binary (`ssh`, `sftp`). Existing `ssh-agent` keys and hardware tokens (YubiKey/FIDO2) work out-of-the-box.
- **Permission Safety**: Writes respect original file permissions (`0600` / `0644`) to guarantee OpenSSH permission checks never fail.
- **Syntax Validation**: Pasted SSH commands and imported flags are sanitized and validated against OpenSSH specifications before being accepted into forms.

---

## 🤝 Contributing

Contributions, bug reports, and suggestions are warmly welcome!

```bash
# 1. Fork and clone the repository
git clone https://github.com/urzeye/lazyssh.git
cd lazyssh

# 2. Run quality checks and tests
make test
make lint

# 3. Create a feature branch and submit a PR
git checkout -b feat/my-new-feature
```

---

## 📄 License & Attribution

This project is licensed under the [Apache License 2.0](LICENSE).

### Acknowledgments & Upstream Credits
- **[Adembc](https://github.com/Adembc)** — Creator of the original [`lazyssh`](https://github.com/Adembc/lazyssh). We express our deepest gratitude for his vision and foundational codebase.
- **[WhiteRoseLK/neossh](https://github.com/WhiteRoseLK/neossh)** — For inspiring modern README aesthetics and demonstrating community dedication to SSH terminal tools.
- **[rivo/tview](https://github.com/rivo/tview)** & **[gdamore/tcell](https://github.com/gdamore/tcell)** — For providing the rich terminal UI toolkit.
- **[derailed/k9s](https://github.com/derailed/k9s)** & **[jesseduffield/lazydocker](https://github.com/jesseduffield/lazydocker)** — For inspiring the terminal user experience.

<div align="center">
  <sub>Maintained with ❤️ by <a href="https://github.com/urzeye">urzeye</a> and community contributors.</sub>
</div>
