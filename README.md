English | [简体中文](README.zh-CN.md)

<div align="center">
  <img src="./docs/logo.png" alt="lazyssh logo" width="600" height="600"/>
</div>

This repository is an enhanced fork of the original [`lazyssh`](https://github.com/Adembc/lazyssh), with a strong focus on real-world SSH setups used with dotfiles, `Include`, and multi-file config workflows. Key improvements include:

- Recursive support for `Include` and `conf.d` style SSH layouts instead of assuming a single `~/.ssh/config`
- Managed mode: read the full config tree from the root config in real time while writing changes only to a dedicated managed file
- Read-only protection for hosts coming from included or unmanaged files, with source markers in the UI to prevent accidental edits
- New flags: `--sshconfig`, `--managed-mode`, `--managed-sshconfig`, and `--ssh-config-readonly`
- Display of effective SSH values after applying `Host *`, pattern rules such as `Host lab-*`, and defaults inherited from the root config
- Proper handling of quoted entries such as `Host "example-name"`
- Better fuzzy search relevance across alias, hostname, user, and tags
- Daily workflow improvements such as a collapsible search bar, `0/1/2` panel switching, copy-host support, and persisted sort mode
- Dark/light theme support with a persisted `T` toggle
- A set of high-value fixes: backspace works correctly in inputs, `j/k` navigation survives mouse clicks, usernames support `@` and `:`, and alias alignment is more stable
- Safer config writes for dotfiles and multi-machine sync: symlinks are preserved and `IdentityFile` is normalized to portable `~/.ssh/...` paths whenever possible
- Preserve negated Host patterns, insert new hosts before matching defaults, and keep search typing and list selection stable after mouse clicks and refreshes
- Clone hosts (`C`), import an SSH command from the clipboard (`v`), generate SCP commands (`P`), and open system SFTP (`F`)
- Start with a CLI filter, connect to an exact alias directly, or exit after a session; connection failures retain their diagnostics

---

lazyssh is a terminal-based SSH manager inspired by `lazydocker` and `k9s`, but built specifically for SSH host management.
It lets you browse, search, connect to, edit, and organize the hosts defined in your SSH config directly from the terminal, so you do not have to remember raw IP addresses or keep typing long `ssh` and `scp` commands by hand. The whole workflow is designed around fast keyboard-first operations.

---

## ✨ Features

### Host Management
- Read and display hosts from `~/.ssh/config`
- Add new hosts directly from the UI
- Edit existing hosts through a tabbed form
- Delete host entries safely
- Pin and unpin frequently used hosts
- Ping hosts to check whether they are reachable

### Fast Navigation
- Fuzzy search by alias, IP, user, or tag
- Press `Enter` to connect to the selected host immediately
- See currently running local SSH sessions in an auto-refreshing read-only panel
- Organize hosts with tags such as `prod`, `dev`, and `test`
- Sort by alias or last connection time, with reversible order

### Advanced SSH Options
- Port forwarding support: `LocalForward`, `RemoteForward`, and `DynamicForward`
- Connection multiplexing for faster follow-up sessions
- Advanced authentication settings, including keys, passwords, and agent forwarding
- Security-related settings such as ciphers, MACs, and key exchange algorithms
- Proxy settings such as `ProxyJump` and `ProxyCommand`
- A tabbed editor that keeps large SSH configurations manageable

### Key Management
- Auto-discover local SSH keys with completion support
- Switch between multiple keys quickly

### Planned Capabilities
- File copy flows between the local machine and remote hosts with a friendlier picker
- SSH public key deployment:
  - Use a default local key such as `~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`
  - Paste a custom public key manually
  - Generate a new key pair and deploy it to the target host
  - Append it to `~/.ssh/authorized_keys` with the correct permissions

### Connection and file workflows

```bash
lazyssh prod                         # Open the picker with an initial fuzzy filter
lazyssh --filter prod                # Equivalent explicit filter
lazyssh --connect production         # Connect to an exact configured alias without the TUI
lazyssh --exit-after-session         # Exit when the chosen SSH/SFTP session ends
lazyssh --sshconfig ~/.ssh/work --connect production
```

`--connect` also accepts secondary aliases from a multi-alias Host line. Unknown aliases and conflicting selection flags return an error; SSH exit codes are preserved. Filters cannot be combined with `--connect`.

`C` opens a prefilled Add form with a unique alias and fresh connection statistics. `v` parses an SSH command without executing it and opens the same form for review. Common flags (`-p`, `-i`, `-J`, `-L/-R/-D`, `-l`, `-A/-X/-Y`, `-t`, `-o`) and quoted arguments are supported. Unsupported flags (including a separate `-F` config) and shell operators are rejected explicitly.

Import preserves Windows backslash paths on Windows and POSIX escapes elsewhere. Unquoted word-initial `#` starts a comment; quoted hashes remain literal. Destination users/ports retain OpenSSH argument precedence. Literal percent signs in a positional remote command are escaped for storage as `RemoteCommand`; explicit `-o RemoteCommand` templates retain their token expansion.

Copied SSH commands keep `RemoteCommand` as a `-o` option, preserving percent expansion when executed or imported again. Saving `SessionType` (including imported `-N`/`-s`) requires a client that supports it, normally OpenSSH 8.7+. An isolated offline probe rejects unsupported settings before changing config or metadata.

`P` opens an upload/download SCP command helper with a preview and clipboard copy; it does not start a transfer. Commands preserve the selected alias and current `-F` root config. On Windows they use PowerShell quoting; elsewhere they use POSIX shell quoting. `F` starts the installed system `sftp` client with the same alias/config and returns to the picker when it ends. Both remain available in read-only mode.

The SCP helper inspects the installed client once when opened. Clients advertising `-s` use explicit SFTP mode; older clients use separate remote-shell quoting so path spaces and metacharacters stay literal. Remote `~` paths expand on the server. Missing clients or unrecognized capability output produce an error instead of guessing a protocol.

Legacy remote paths use backslash escapes compatible with SCP's receive-side filename check; that check remains enabled. Windows commands launch the native process with explicit arguments, supporting PowerShell 5.1 and 7 without changing the caller's argument-passing preference.

Tag-only edits update metadata without rewriting SSH config. Managed-mode insertion changes only the managed file: earlier root/global rules retain their OpenSSH precedence, so place the managed `Include` before defaults intended to be overridable.

---

## 🔐 Security

lazyssh does not introduce extra security risk on its own.
It is fundamentally a TUI layer on top of your existing SSH setup.

- All SSH connections are executed through the system `ssh` binary, i.e. OpenSSH
- lazyssh does not store, transmit, or modify your private keys, passwords, or other credentials
- Existing `IdentityFile` paths and `ssh-agent` workflows continue to work as usual
- lazyssh only reads and updates your SSH config files, and automatically creates backups before the first modification
- It tries to preserve existing file permissions to avoid introducing new risk during config writes

## 🧭 Managed Mode

If your SSH setup looks like this:

```sshconfig
Include ~/.orbstack/ssh/config
Include ~/.ssh/config.d/*.conf
Include ~/.ssh/config.local
```

you can run lazyssh in managed mode:

```bash
lazyssh \
  --sshconfig ~/.ssh/config \
  --managed-mode \
  --managed-sshconfig ~/.ssh/config.local
```

In this mode:

- `--sshconfig` points to the root config, and lazyssh recursively reads the full `Include` tree from it
- `--managed-sshconfig` points to the actual writable file, such as `~/.ssh/config.local`
- Entries coming from the root config, `config.d`, OrbStack, or any other included file are shown as read-only
- Only hosts defined in the managed file can be added, edited, or deleted
- Effective values inherited through `Host *`, `Host xxx-*`, and similar patterns are rendered correctly in the UI
- SSH connections and forwarding still use the root config, so behavior stays aligned with the system `ssh` command

## 🔒 Read-Only Mode

If you want lazyssh to work strictly as a browser/launcher for your SSH config, run:

```bash
lazyssh --ssh-config-readonly
```

or set:

```bash
LAZYSSH_SSH_CONFIG_READONLY=1 lazyssh
```

In read-only mode, lazyssh still reads the full `Include` tree and allows search, details, copy, ping, SSH connection, forwarding, pinning, and last-seen metadata. It disables SSH config mutations such as adding, editing, deleting, or editing tags.

## 🛡️ Safe Writes and Automatic Backups

- Non-destructive edits: lazyssh writes only the smallest required change set and tries to preserve comments, blank lines, ordering, and untouched fields
- Atomic writes: updates are written to a temporary file first and then atomically replaced to reduce the risk of partial writes
- Symlink-safe behavior: if the managed config file itself is a symlink, lazyssh updates the target file instead of overwriting the symlink
- First-write backup: before the first modification of a managed SSH config file, lazyssh creates a sibling backup such as `config.original.backup` or `config.local.original.backup`
- Rolling backups: every later save also creates a timestamped backup such as `~/.ssh/config.local-<timestamp>-lazyssh.backup`
- Backup rotation: up to 10 rolling backups are kept automatically, and older ones are cleaned up
- Portable paths: `IdentityFile` entries inside the current user's home directory are normalized to `~/.ssh/...` when written back, which is friendlier for cross-machine dotfile sync

## 📷 Screenshots

<div align="center">

### 🚀 Startup Screen
<img src="./docs/loader.png" alt="Startup screen" width="800" />

The loading screen shown when the application starts.

---

### 📋 Host Management Panel
<img src="./docs/list server.png" alt="Host list view" width="900" />

The main view shows configured hosts, status information, pinned entries, and quick navigation actions.

---

### 🔎 Search
<img src="./docs/search.png" alt="Fuzzy host search" width="900" />

Search quickly by hostname, IP address, user, or tag.

---

### ➕ Add / Edit Hosts
<img src="./docs/add server.png" alt="Add host screen" width="900" />

The tabbed editor is organized around common SSH workflows:

- **Basics**: host, user, port, key, and tags
- **Connection**: proxies, timeout, multiplexing, and canonicalization
- **Forwarding**: port forwarding, X11, and agent forwarding
- **Authentication**: keys, passwords, auth methods, and algorithm options
- **Advanced**: security, encryption, environment variables, and debug options

---

### 🔐 Connect to Hosts
<img src="./docs/ssh.png" alt="SSH connection view" width="900" />

Start an SSH session directly from the interface.

</div>

---

## 📦 Installation

### Option 1: Install with mise (Recommended)

If you already use `mise`, you can install lazyssh directly from this repository's GitHub Releases:

```bash
mise use -g github:urzeye/lazyssh@latest
lazyssh
```

If you want to pin a specific release, specify the version explicitly:

```bash
mise use -g github:urzeye/lazyssh@v2026.6.4
```

Notes:

- This uses the `github:` backend in `mise`, which downloads the matching release asset from `urzeye/lazyssh`
- Release tags and release names use `vYYYY.M.D`, so upstream PRs and issues before that release date can be skipped during the next sync pass
- `mise` writes the selected version into your global config, so `lazyssh` becomes available in all terminals afterward

### Option 2: Build from Source

```bash
git clone https://github.com/urzeye/lazyssh.git
cd lazyssh

# Build
make build
./bin/lazyssh

# Or run directly
make run
```

---

## ⌨️ Keyboard Shortcuts

| Key | Action |
| ---- | ---- |
| `/` | Expand or collapse the search bar |
| `↑↓` / `j` `k` | Move through the host list |
| `Enter` | SSH into the selected host |
| `c` | Copy the SSH command to the clipboard |
| `h` | Copy the host alias to the clipboard |
| `g` | Ping the current host |
| `r` | Refresh background data |
| `a` | Add a host |
| `C` | Clone the selected host into an Add form |
| `v` | Paste an SSH command into an Add form |
| `P` | Generate and copy an SCP upload/download command |
| `F` | Open system SFTP for the selected host |
| `e` | Edit a host |
| `t` | Edit tags |
| `d` | Delete a host |
| `p` | Pin or unpin a host |
| `s` | Switch the sort field |
| `S` | Reverse the sort order |
| `T` | Toggle dark/light theme |
| `0` / `1` / `2` | Focus the search bar / host list / details panel |
| `q` | Quit |

**Inside host forms:**

| Key | Action |
| ---- | ---- |
| `Ctrl+H` | Switch to the previous tab |
| `Ctrl+L` | Switch to the next tab |
| `Ctrl+S` | Save |
| `Esc` | Cancel |

Tip: the hint area at the top of the list shows the most common shortcuts so they are easy to rediscover.

---

## ⭐ Original Author Support

This repository is independently maintained. The links below are the original author's support links from the upstream [`Adembc/lazyssh`](https://github.com/Adembc/lazyssh) project, not the funding links for this fork.

If you find LazySSH useful and want to support the original author, please consider giving the upstream repository a **star** and joining its [stargazers](https://github.com/Adembc/lazyssh/stargazers).

You can also support the original author, Adembc, by [buying them a coffee](https://www.buymeacoffee.com/adembc).

---

## 🙏 Credits

- Special thanks to the original [`lazyssh`](https://github.com/Adembc/lazyssh) project and its author/maintainers for creating the foundation this fork builds on
- Built with [tview](https://github.com/rivo/tview) and [tcell](https://github.com/gdamore/tcell)
- Design inspiration from [k9s](https://github.com/derailed/k9s) and [lazydocker](https://github.com/jesseduffield/lazydocker)
