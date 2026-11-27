<div align="center">
  <img src="./docs/logo.png" alt="lazyssh logo" width="160" />
  <h1>lazyssh</h1>
  <p>面向 OpenSSH 配置的终端键盘流 SSH 管理器。</p>
  <p><a href="README.md">English</a> | 简体中文</p>
  <p>
    <a href="https://github.com/urzeye/lazyssh/releases"><img src="https://img.shields.io/github/v/release/urzeye/lazyssh?style=flat-square&color=blue" alt="GitHub release" /></a>
    <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/urzeye/lazyssh?style=flat-square&color=00ADD8" alt="Go Version" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/urzeye/lazyssh?style=flat-square&color=green" alt="License" /></a>
    <a href="https://github.com/urzeye/lazyssh/actions/workflows/go.yml"><img src="https://img.shields.io/github/actions/workflow/status/urzeye/lazyssh/go.yml?branch=main&style=flat-square" alt="Build Status" /></a>
    <a href="https://github.com/urzeye/lazyssh/releases"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey?style=flat-square" alt="Platform" /></a>
  </p>
</div>

> [!NOTE]
> 本项目是 [Adembc/lazyssh](https://github.com/Adembc/lazyssh) 的独立维护分支。原仓库已长期未更新，本分支持续维护并补齐了多文件 `Include`、dotfiles 托管写入模式、安全保存机制以及日常 SSH/SFTP/SCP 工作流。感谢原作者 [Adembc](https://github.com/Adembc) 的原始设计与基础代码。

---

## 核心功能

### 多文件 SSH 配置与 Dotfiles 适配
- **递归解析 `Include`**：按 OpenSSH 生效顺序递归解析 `Include`（支持 `~`、相对路径、通配符如 `~/.ssh/config.d/*.conf` 以及 OrbStack/Colima 等自动生成的配置），遇到 `Match` 指令块安全跳过不报错。
- **生效参数计算**：按 OpenSSH 首次匹配规则，实时展示主机从 `Host *` 及通配规则（如 `Host lab-*`）继承后的实际连接参数。
- **Managed 托管模式（`--managed-mode`）**：从根配置读取整棵配置树，但所有新增、编辑和删除只写到指定的受管文件（默认 `~/.ssh/config.local`），不改动纳入 Git 管理的根 dotfiles。
- **来源标记与只读保护**：可写主机在列表右侧显示 `🏠`，来自引入/非受管文件或 `--ssh-config-readonly` 模式下的只读主机显示 `🔗`，阻止误修改和误删除。
- **安全写入机制**：
  - 先写同目录临时文件，再通过 `os.Rename` 原子替换目标文件，写入权限固定为 `0600`。
  - 如果配置文件是软链接（Symlink），直接写入软链接指向的实际文件，保留软链接不变。
  - 首次修改前自动创建一次性原始备份（`<文件名>.original.backup`）；之后每次保存生成带 Unix 毫秒时间戳的滚动备份（`<文件名>-<unix-ms>-lazyssh.backup`），自动保留最近 10 份。
  - 新增主机自动插在匹配的通配符默认块之前，避免被 `Host *` 等规则覆盖。
  - 位于用户家目录下的 `IdentityFile` 路径自动归一化为 `~/.ssh/...`，并保留带引号的 Host 别名（`Host "name"`）与否定模式（`Host !foo`）。
- **元数据与配置分离**：标签、置顶、连接次数和最近连接时间存在 `~/.lazyssh/metadata.json`；排序与主题偏好存在 `~/.lazyssh/settings.json`；运行日志存在 `~/.lazyssh/lazyssh.log`。修改标签不会改写 SSH 配置文件。

### 日常终端操作
- **模糊搜索与快捷聚焦**：按别名、主机名/IP、用户名、标签综合匹配并排序；按 `0`/`/` 打开搜索栏，`1`/`2` 切换列表与详情面板，支持 `j`/`k` 移动列表或在详情面板上下滚动。
- **连接与文件传输辅助**：
  - `Enter` 直接调用系统 `ssh` 登录，连接失败时保留完整错误输出与退出码。
  - `F` 使用当前 `-F` 根配置直接拉起系统 `sftp`。
  - `P` 打开 SCP 上传/下载命令生成器，支持实时预览和一键复制到剪贴板。
  - `c` 复制完整的独立 `ssh` 命令；`h` 复制主机地址（`HostName`/IP，未配置时回退为别名）。
- **建机辅助**：
  - `a` 打开五标签页编辑器（`Basic` / `Connection` / `Forwarding` / `Authentication` / `Advanced`），宽屏下右侧附带字段说明，支持本地私钥与 `known_hosts` 自动补全。
  - `C` 克隆选中主机到预填新增表单（自动生成不冲突的新别名并清空历史统计）。
  - `v` 解析剪贴板中的 `ssh` 命令填入新增表单，不执行任何命令。
- **端口转发、Ping 与活跃会话**：
  - `f` / `x` 交互式创建本地（`-L`）、远程（`-R`）或动态 SOCKS5（`-D`）转发（支持仅转发或转发+登录），并可一键停止由 lazyssh 启动的后台转发。
  - `g` 通过 `ssh -G -F <config>` 解析真实目标地址和端口，发起 TCP 连通性测试并显示耗时。
  - 左下角只读**活跃会话（Active Sessions）**面板每 10 秒自动刷新，展示本机正在运行的 `ssh` 客户端进程。

---

## 界面预览

<details>
<summary>展开查看截图</summary>
<br>

**主机列表、详情与活跃会话**

<img src="./docs/list server.png" alt="主机列表与活跃会话" width="880" />

**模糊搜索**

<img src="./docs/search.png" alt="模糊搜索" width="880" />

**五标签页主机编辑器**

<img src="./docs/add server.png" alt="主机配置编辑界面" width="880" />

**SSH 连接**

<img src="./docs/ssh.png" alt="SSH 连接" width="880" />

</details>

---

## 安装

### 方式 1：使用 mise（推荐）

```bash
mise use -g github:urzeye/lazyssh@latest
lazyssh --version
```

安装指定版本：

```bash
mise use -g github:urzeye/lazyssh@v2026.11.18
```

### 方式 2：下载预编译二进制

从 [GitHub Releases](https://github.com/urzeye/lazyssh/releases) 下载对应平台压缩包，解压后将 `lazyssh`（Windows 为 `lazyssh.exe`）放入 `PATH` 目录：

| 操作系统 | 架构 | 压缩包格式 |
|---|---|---|
| Linux | `x86_64` (amd64)、`arm64`、`i386`、`armv6` | `.tar.gz` |
| macOS | `arm64`、`x86_64` (amd64) | `.tar.gz` |
| Windows | `x86_64` (amd64)、`arm64`、`i386` | `.zip` |

### 方式 3：从源码编译

需要 Go 1.24.6+。

```bash
git clone https://github.com/urzeye/lazyssh.git
cd lazyssh

# 编译到 ./bin/lazyssh
make build

# 或直接安装到 GOBIN
make install
```

---

## 命令行用法

```bash
# 启动交互式 TUI（默认读取 ~/.ssh/config）
lazyssh

# 启动时直接带入模糊搜索关键词
lazyssh prod
lazyssh --filter dev

# 不打开 TUI，直接连接精确匹配的别名（保留 ssh 退出码）
lazyssh -c production-server

# 会话结束后自动退出 lazyssh
lazyssh --exit-after-session

# 指定自定义 SSH 配置文件并直连
lazyssh --sshconfig ~/.ssh/work_config -c bastion

# 只读模式（禁止修改配置文件和标签）
lazyssh --ssh-config-readonly
LAZYSSH_SSH_CONFIG_READONLY=1 lazyssh
```

### 参数一览

| 参数 | 环境变量 / 默认值 | 说明 |
|---|---|---|
| `[filter]` / `-f, --filter` | `""` | 启动时预填模糊过滤词（不可与 `--connect` 同时使用） |
| `-c, --connect` | `""` | 不启动 TUI，直接连接精确匹配的别名（支持多别名 Host 行里的次要别名） |
| `--exit-after-session` | `false` | 所选 SSH 或 SFTP 会话结束后自动退出 |
| `--sshconfig` | `~/.ssh/config` | 根 SSH 配置文件路径，同时作为 `ssh`/`sftp`/`scp` 的 `-F` 参数 |
| `--managed-mode` | `false` | 从根配置读取全部主机，只向受管配置文件写入变更 |
| `--managed-sshconfig` | `~/.ssh/config.local` | 受管写入文件路径（指定该参数会自动开启 managed mode） |
| `--ssh-config-readonly` | `LAZYSSH_SSH_CONFIG_READONLY` | 禁止修改配置与标签；浏览、连接、SCP、SFTP、端口转发、Ping、置顶仍可用 |
| `-v, --version` | — | 打印版本号与 commit 哈希 |

---

## Managed 模式（Dotfiles 场景）

如果 `~/.ssh/config` 由 Git dotfiles 仓库管理，建议在其中引入一个本地受管文件，并放在全局默认块之前：

```sshconfig
# ~/.ssh/config（由 dotfiles 仓库管理）
Include ~/.orbstack/ssh/config
Include ~/.ssh/config.d/*.conf
Include ~/.ssh/config.local

Host *
    ServerAliveInterval 60
    TCPKeepAlive yes
```

> [!TIP]
> 请把 `Include ~/.ssh/config.local` 写在需要被覆盖的 `Host *` 默认块**之前**，且**不要**写在某个具体 `Host <名称>` 块的下方（按 OpenSSH 规则，写在具体 `Host` 块下面的 `Include` 会变成只对该主机生效的条件引入）。

启动命令：

```bash
lazyssh \
  --sshconfig ~/.ssh/config \
  --managed-mode \
  --managed-sshconfig ~/.ssh/config.local
```

在该模式下：
- 根配置与所有 `Include` 文件中的主机都会完整列出，并展示继承后的生效参数。
- 只有定义在 `~/.ssh/config.local` 中的主机可编辑（`🏠`），根配置与其他被引入文件的主机为只读（`🔗`）。
- 新增、编辑、删除只会改写 `~/.ssh/config.local`。
- 实际连接时依然使用 `-F ~/.ssh/config`，与直接运行 `ssh` 的行为一致。

---

## 命令导入、SCP 与 SFTP 说明

- **剪贴板导入 SSH 命令（`v`）**：仅本地解析文本，绝不执行命令或 shell 展开。支持常用 OpenSSH 参数（`-p`、`-i`、`-J`、`-L/-R/-D`、`-l`、`-b`、`-B`、`-e`、`-c`、`-m`、`-A/-a`、`-X/-x/-Y`、`-C`、`-4/-6`、`-T/-t`、`-N`、`-s`、`-q/-v`、`-o Key=Value`）及 `ssh://` URI；明确拒绝不支持的参数（包括独立 `-F`）、shell 运算符（`;`、`|`、`&`、`<`、`>`、`` ` ``）和 URI 密码。位置参数形式的远端命令会自动转义字面量 `%`，而 `-o RemoteCommand=...` 会保留 OpenSSH 占位符模板。保存 `SessionType`（含导入的 `-N`/`-s`）前会离线探测本机 `ssh` 是否支持（需 OpenSSH 8.7+）。
- **SCP 命令生成器（`P`）**：按当前主机别名和根 `-F` 配置生成可复制的上传/下载命令，不会自动开始传输。打开时探测本机 `scp` 能力：支持 `-s` 时自动启用 SFTP 模式，旧版客户端则对远端路径做安全转义。本地 `~` 展开为本机路径，远端 `~` 留给远端主机展开。只读模式下同样可用。
- **系统 SFTP（`F`）**：执行 `sftp -F <config> -- <alias>`，退出后返回 lazyssh。只读模式下同样可用。

---

## 快捷键速查

### 导航与焦点
| 按键 | 说明 |
|---|---|
| `0` / `/` | 打开并聚焦搜索栏 |
| `Esc` / `Enter` *(搜索栏内)* | 清空并收起搜索栏 |
| `↑` / `↓` *(搜索栏内)* | 回到主机列表并移动选择 |
| `1` | 聚焦主机列表 |
| `2` | 聚焦主机详情面板 |
| `↑` `↓` / `j` `k` | 在列表中移动，或聚焦详情面板时上下滚动 |
| `Enter` | 通过 SSH 连接当前主机 |
| `q` | 退出 |

### 主机管理
| 按键 | 说明 |
|---|---|
| `a` | 新增主机 |
| `C` | 克隆选中主机到预填新增表单 |
| `v` | 从剪贴板导入 SSH 命令到新增表单 |
| `e` | 编辑选中主机（只读主机不可编辑） |
| `d` | 删除选中主机（带二次确认） |
| `t` | 编辑标签（仅改写元数据，不改 SSH 配置；只读主机不可编辑） |
| `p` | 置顶 / 取消置顶当前主机 |

### 传输、转发与工具
| 按键 | 说明 |
|---|---|
| `c` | 复制完整 `ssh` 命令到剪贴板 |
| `h` | 复制主机地址（`HostName`/IP，未配置时复制别名）到剪贴板 |
| `P` | 打开 SCP 上传/下载命令生成器 |
| `F` | 为选中主机启动系统 `sftp` |
| `f` | 打开端口转发向导（本地 / 远程 / 动态 SOCKS5） |
| `x` | 停止当前主机通过 lazyssh 启动的后台端口转发 |
| `g` | 对选中主机执行 TCP Ping 检测 |
| `r` | 刷新主机列表与活跃会话 |

### 视图与偏好
| 按键 | 说明 |
|---|---|
| `s` | 切换排序字段（`Alias` ↔ `Last SSH`） |
| `S` | 反转当前排序方向 |
| `T` | 切换深色 / 浅色主题 |

### 主机编辑表单内
| 按键 | 说明 |
|---|---|
| `Ctrl+←` / `Ctrl+→` | 切换上一页 / 下一页（表单内任意位置均可用） |
| `Ctrl+H` / `Ctrl+L` | 切换上一页 / 下一页（在文本输入框中 `Ctrl+H` 保留为退格删除） |
| `Ctrl+S` | 保存主机配置 |
| `Esc` | 取消并返回主界面 |

---

## 安全说明

- **不存储凭证**：lazyssh 不读取、不保存、也不传输私钥、密码或通行短语。
- **直接调用系统 OpenSSH**：所有 SSH、SFTP 和端口转发都直接调用系统 `ssh`/`sftp` 命令，原生兼容 `ssh-agent`、1Password SSH Agent 与硬件密钥。
- **安全权限**：写回配置文件时使用原子替换并固定权限为 `0600`。

---

## 参与贡献

```bash
git clone https://github.com/urzeye/lazyssh.git
cd lazyssh

make test
make lint
```

PR 标题需符合 Conventional Commits 规范（类型：`fix`、`feat`、`improve`、`refactor`、`revert`、`test`、`ci`、`docs`、`chore`；可选 scope：`ui`、`cli`、`config`、`parser`）。

---

## 协议与致谢

本项目基于 [Apache License 2.0](LICENSE) 开源。

- [Adembc](https://github.com/Adembc) — 原项目 [`lazyssh`](https://github.com/Adembc/lazyssh) 作者
- [rivo/tview](https://github.com/rivo/tview) & [gdamore/tcell](https://github.com/gdamore/tcell) — 终端 UI 框架
- [derailed/k9s](https://github.com/derailed/k9s) & [jesseduffield/lazydocker](https://github.com/jesseduffield/lazydocker) — 交互灵感来源
