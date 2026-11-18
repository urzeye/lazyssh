<div align="center">
  <img src="./docs/logo.png" alt="lazyssh logo" width="180" />
  <h1>⚡ lazyssh</h1>
  <p><b>面向真实生产与开发环境的终端键盘流 SSH 管理器</b></p>
  <p><i>递归解析 OpenSSH <code>Include</code> • 适配 Dotfiles 的 Managed 托管写入模式 • 零破坏原子级安全备份 • 极致高效的全键盘终端操作</i></p>

  <p>
    <a href="README.md">English</a> | 简体中文
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
> **lazyssh 是基于 [Adembc/lazyssh](https://github.com/Adembc/lazyssh) 的独立维护增强分支。**
> 原仓库目前已停止维护。本项目专注于解决多层级 `Include`、dotfiles 跨设备同步、受管配置隔离、通配规则最终生效值实时计算以及全键盘高效运维等真实场景痛点。
> 在此向原作者 **[Adembc](https://github.com/Adembc)** 的卓越创意与底层架构致以崇高的敬意与感谢。

---

## 📑 目录

- [💡 为什么选择 lazyssh？](#-为什么选择-lazyssh)
- [🌟 核心亮点](#-核心亮点)
- [📷 界面预览](#-界面预览)
- [📦 安装指南](#-安装指南)
  - [方式 1：使用 mise 安装（推荐）](#方式-1使用-mise-安装推荐)
  - [方式 2：使用 Go 安装](#方式-2使用-go-安装)
  - [方式 3：直接下载预编译二进制文件](#方式-3直接下载预编译二进制文件)
  - [方式 4：从源码编译](#方式-4从源码编译)
- [⚡ 快速上手与 CLI 工作流](#-快速上手与-cli-工作流)
- [🧭 深度解析：Dotfiles 与 Managed 模式](#-深度解析dotfiles-与-managed-模式)
- [🛡️ 安全写入与自动备份机制](#️-安全写入与自动备份机制)
- [⌨️ 键盘快捷键速查表](#️-键盘快捷键速查表)
- [🔐 安全与权限架构](#-安全与权限架构)
- [🤝 参与贡献](#-参与贡献)
- [📄 开源协议与鸣谢](#-开源协议与鸣谢)

---

## 💡 为什么选择 lazyssh？

**lazyssh** 是一款运行在终端中的全键盘交互式 SSH 客户端与服务器管理器，设计灵感来自 `lazydocker` 与 `k9s`，但针对日常真实 OpenSSH 配置生态进行了深度重构与专项强化。

随着管理的基础设施增多，开发者往往要在公有云、自建机房、跳板机、本地容器及测试环境中频繁切换，面临以下常见痛点：
- 依靠记忆或笔记寻找服务器 IP 与端口，频繁手敲繁琐冗长的 `ssh` / `scp` 命令。
- 现有许多 TUI/GUI 工具只能解析单个平铺的 `~/.ssh/config`，遇到 `Include`、OrbStack、Colima 或 `conf.d/` 分离配置就无法正确工作。
- 传统工具保存时会直接格式化重写文件，导致私人注释丢失、符号链接（Symlink）断裂，甚至污染纳入 Git 版本控制的 dotfiles 仓库。

**lazyssh 彻底化解了这些难题：**
- **原生支持完整 `Include` 配置树**：透明解析所有被引用的分层配置文件。
- **独创 Managed 托管模式**：全局只读读取全部配置，但所有新增与修改精准写回指定的局部文件（如 `~/.ssh/config.local`），让 dotfiles 根配置始终保持纯净。
- **配置最终生效值引擎**：自动继承并计算 `Host *` 与模式匹配规则，所见即 OpenSSH 实际连接参数。
- **零破坏原子写入与滚动备份**：临时文件原子替换、保留符号链接、初次基线备份及自动轮转的 10 份历史备份。
- **极速全键盘操作流**：涵盖模糊智能搜索、面板一键切换、SCP 命令生成、SFTP 直启、端口转发向导以及剪贴板命令导入。

---

## 🌟 核心亮点

### 📁 深度配置与 Dotfiles 协同
- **递归 `Include` 深度解析**：支持多层 `Include` 语法（包括通配符 `~/.ssh/config.d/*.conf`、OrbStack、Colima 等），无需将配置强行压平在单一文件。
- **Managed 托管模式（`--managed-mode`）**：以根配置为入口读取整棵配置树，但将所有新增、修改和删除操作完全隔离在受管配置文件中（默认为 `~/.ssh/config.local`）。
- **来源标记与只读保护**：非当前受管文件或被引入的主机在列表中自动标记 `[inc]` 来源标签，阻止误触编辑。同时支持 `--ssh-config-readonly` 纯浏览模式。
- **最终生效值（Effective Values）计算**：实时合并根配置全局 `Host *`、`Host lab-*` 等通配模式规则，列表与详情页中展示的即为实际连接所采用的生效值。
- **智能插入顺序**：新增主机自动放置在匹配的通配符默认规则之前，严格遵循 OpenSSH 从上到下首次匹配原则。
- **语法细节全兼容**：完美支持带引号主机名（`Host "my-server"`）、单行多别名定义以及否定模式（`Host !foo bar-*`）。

### 🛡️ 零破坏与高可靠文件写入
- **原子替换写入**：修改先写入临时兄弟文件，验证无误后调用系统级原子重命名（`os.Rename`），杜绝因断电或异常终止产生半截文件损坏。
- **符号链接安全**：如果受管配置本身是 Symlink，修改将精确写入目标物理文件，绝不会打破软链接关系。
- **双重备份安全网**：初次修改前自动生成永久基线备份（如 `config.original.backup`）；后续每次保存自动生成带时间戳备份，并保持 **10 份滚动备份** 自动轮转淘汰。
- **可移植密钥路径归一化**：写入时自动将位于用户家目录下的私钥绝对路径标准化为 `~/.ssh/...` 相对路径，便于 dotfiles 跨不同系统与机器无缝同步。
- **配置与元数据分离**：标签、置顶顺序、连接统计保存在 `~/.lazyssh/metadata.json` 中，编辑标签绝不重写或污染 SSH 配置文件。

### ⌨️ 极致流畅的终端全键盘体验
- **多维度模糊搜索**：基于别名、IP 地址、用户名及自定义标签进行智能权重排序，快速定位目标主机。
- **数字面板跳转**：通过快捷键 <kbd>0</kbd> 或 <kbd>/</kbd>（搜索栏）、<kbd>1</kbd>（主机列表）、<kbd>2</kbd>（详情面板）实现光标毫秒级切换。
- **可折叠搜索栏**：按 <kbd>/</kbd> 随时展开或收起搜索栏，最大化利用垂直终端空间。
- **排序与主题偏好记忆**：支持按别名或最近连接时间排序（<kbd>s</kbd>），支持逆序切换（<kbd>S</kbd>），并支持随时切换暗色/亮色主题（<kbd>T</kbd>），配置自动持久化保存。
- **平滑导航与防点击错乱**：针对终端鼠标点击、光标重获、退格删除进行了底层鲁棒性加固，保证 <kbd>j</kbd>/<kbd>k</kbd> 导航与输入状态丝滑稳定。

### 🚀 连接、转发与运维效率工具
- **一键直连与错误保留**：按下 <kbd>Enter</kbd> 立即调用系统 OpenSSH 建立连接；若发生连接失败，保留详细诊断日志与退出码。
- **快速克隆主机（<kbd>C</kbd>）**：复制已有主机的所有网络与认证选项至新增表单，自动生成去重别名并重置连接统计。
- **剪贴板命令导入（<kbd>v</kbd>）**：直接解析剪贴板中的完整 `ssh user@host -p 2222 -i ...` 命令行并填入表单，自动支持常见标志与引号转义。
- **SCP 命令交互式生成器（<kbd>P</kbd>）**：提供直观的上传/下载路径输入向导，根据本地环境智能探测并选用 OpenSSH `-s` SFTP 模式或安全引号转义，一键复制生成完整 SCP 命令。
- **系统 SFTP 直启（<kbd>F</kbd>）**：基于当前主机配置一键拉起系统原生 `sftp` 客户端，传输完毕后优雅返回 lazyssh。
- **端口转发向导（<kbd>f</kbd> / <kbd>x</kbd>）**：可视化配置本地转发（`-L`）、远程转发（`-R`）或动态 SOCKS5 代理（`-D`），支持纯转发与转发+登录双模式，一键随时停止转发。
- **活跃会话看板**：自动刷新展示本机当前正在运行的全部 SSH 连接会话，连接状态一目了然。
- **在线状态 Ping 探测（<kbd>g</kbd>）**：利用主机实际配置的目标参数发起在线探测，即刻反馈网络延迟与可用性。

---

## 📷 界面预览

<details open>
<summary><b>点击展开 / 折叠查看应用截图</b></summary>
<br>

<div align="center">

### 📋 主机管理面板与活跃会话
<img src="./docs/list server.png" alt="主机列表与活跃会话" width="900" />
<p><i>浏览主机配置、查看实时计算的生效参数、监控本地活跃会话、识别 [inc] 来源标记。</i></p>

---

### 🔎 智能模糊搜索
<img src="./docs/search.png" alt="模糊搜索界面" width="900" />
<p><i>支持按别名、IP、用户、标签综合匹配，精准命中所需主机。</i></p>

---

### ➕ 五标签页主机编辑器
<img src="./docs/add server.png" alt="主机配置编辑界面" width="900" />
<p><i>按 <b>基础 (Basic)</b>、<b>连接 (Connection)</b>、<b>转发 (Forwarding)</b>、<b>认证 (Authentication)</b>、<b>高级 (Advanced)</b> 模块化管理数十项 OpenSSH 指令。</i></p>

---

### 🔐 SSH 连接与命令辅助工具
<img src="./docs/ssh.png" alt="SSH 连接与命令生成" width="900" />
<p><i>终端一键登录、快速复制命令、生成生产级安全 SCP 传输命令。</i></p>

---

### 🚀 异步启动加载
<img src="./docs/loader.png" alt="启动界面" width="800" />
<p><i>快速平滑加载多层级 Include 配置文件与本地元数据。</i></p>

</div>

</details>

---

## 📦 安装指南

### 方式 1：使用 mise 安装（推荐）

如果你使用现代环境管理工具 [mise-en-place](https://mise.jdx.dev/)，可以直接从 GitHub Releases 安装并全局生效：

```bash
# 全局安装最新发布版本
mise use -g github:urzeye/lazyssh@latest

# 验证安装
lazyssh --version
```

若需锁定特定日期发布版本：
```bash
mise use -g github:urzeye/lazyssh@v2026.6.4
```

### 方式 2：使用 Go 安装

如果你的系统已配置 Go 环境（Go 1.22+）：

```bash
go install github.com/urzeye/lazyssh/cmd@latest
```

请确保 `$GOPATH/bin` 或 `$HOME/go/bin` 已加入系统的 `$PATH` 环境变量中。

### 方式 3：直接下载预编译二进制文件

你可以直接前往 [GitHub Releases](https://github.com/urzeye/lazyssh/releases) 页面下载适合你系统的压缩包：

| 操作系统 | 架构 | 压缩包格式 |
|---|---|---|
| **Linux** | `x86_64` (amd64), `aarch64` (arm64), `i386`, `armv6` | `.tar.gz` |
| **macOS** | `Apple Silicon` (arm64), `Intel` (amd64) | `.tar.gz` |
| **Windows** | `x86_64` (amd64), `arm64`, `i386` | `.zip` |

解压后将 `lazyssh`（或 Windows 下的 `lazyssh.exe`）移动到你的 `PATH` 目录（例如 Linux/macOS 下的 `/usr/local/bin`）。

### 方式 4：从源码编译

```bash
# 克隆代码仓库
git clone https://github.com/urzeye/lazyssh.git
cd lazyssh

# 编译二进制至 ./bin/lazyssh
make build

# 或直接运行
make run
```

---

## ⚡ 快速上手与 CLI 工作流

```bash
# 1. 直接打开交互式 TUI（默认读取 ~/.ssh/config）
lazyssh

# 2. 携带初始模糊过滤条件打开界面
lazyssh prod
lazyssh --filter dev

# 3. 命令行静默直连（无需打开 TUI 界面，保留 SSH 原生退出状态码）
lazyssh -c production-server

# 4. 会话结束后自动退出 lazyssh
lazyssh --exit-after-session

# 5. 指定自定义根配置文件并直连目标主机
lazyssh --sshconfig ~/.ssh/work_config -c bastion

# 6. 安全只读模式启动（禁用一切配置写入操作）
lazyssh --ssh-config-readonly
# 或通过环境变量启用：
export LAZYSSH_SSH_CONFIG_READONLY=1
lazyssh
```

---

## 🧭 深度解析：Dotfiles 与 Managed 模式

### 经典痛点
许多资深工程师会使用 Git 管理家目录的 dotfiles 仓库（例如 `~/.ssh/config` 常常是指向 `dotfiles/ssh/config` 的符号链接，包含团队通用设置或多层 `Include`）。常规 SSH 工具直接修改此文件时，极易损坏符号链接、格式化掉说明注释，或将临时的测试机器直接提交进版本库中。

### 优雅方案：Managed 模式
通过 lazyssh 的 **Managed 托管模式**，你可以以 dotfiles 根配置为统一入口，但所有写操作均被限制在专用的本地文件中：

```sshconfig
# ~/.ssh/config（由 Git 版本库管理）
Include ~/.orbstack/ssh/config
Include ~/.ssh/config.d/*.conf
Include ~/.ssh/config.local      # <-- lazyssh 专用的增删改写回目标！

Host *
    ServerAliveInterval 60
    TCPKeepAlive yes
```

使用托管模式运行 lazyssh：

```bash
lazyssh \
  --sshconfig ~/.ssh/config \
  --managed-mode \
  --managed-sshconfig ~/.ssh/config.local
```

```
~/.ssh/config（根配置文件 / Dotfiles 版本库）
 ├── Include ~/.orbstack/ssh/config  [只读状态：UI 标记为 [inc]]
 ├── Include ~/.ssh/config.d/*.conf  [只读状态：UI 标记为 [inc]]
 └── Include ~/.ssh/config.local     [受管目标：新增/修改的主机均写在此处]
```

#### Managed 模式的核心优势：
1. **全局视图**：完整解析并展示所有层级 `Include` 中的主机。
2. **写隔离保护**：所有新增、修改、删除操作仅作用于 `~/.ssh/config.local`。
3. **误修改拦截**：若试图修改来自其他被引入文件的条目，系统将弹出警告并明确提示该主机的原始定义来源。
4. **层叠规则继承**：根配置中的 `Host *` 与模式配置依然完美生效，UI 会动态计算并展示继承后的参数。
5. **行为原生一致**：执行连接时始终遵循 `--sshconfig ~/.ssh/config`，与系统原生 `ssh` 命令的解析逻辑 100% 契合。

---

## 🛡️ 安全写入与自动备份机制

lazyssh 对待你的 SSH 配置文件如同对待核心数据一般慎重：

- **原子保存（Atomic Save）**：写入时先在同目录下生成临时文件，完成内容校验后再执行系统级重命名替换（`os.Rename`），避免半途崩溃导致文件损坏。
- **符号链接防护**：自动探测目标文件是否为软链接。若是软链接，则穿透更新其链接的真实物理文件，绝对不破坏软链接结构。
- **首次修改基准备份**：在对任何配置文件进行第一次写操作前，自动生成基准副本（如 `config.original.backup`）。
- **滚动时间戳备份**：后续每次保存均会自动生成带精确时间戳的备份文件（如 `config-20261118-190516-lazyssh.backup`），并自动保留 **最近 10 份历史备份**，旧备份自动清理。
- **可移植 IdentityFile 处理**：自动将家目录路径规整为 `~/.ssh/...` 相对写法，避免因绝对路径硬编码导致多系统同步时密钥失效。
- **最小化变更差异**：仅针对目标主机段落进行局部增删改，原文件的注释行、空白行及排版格式完整保留。

---

## ⌨️ 键盘快捷键速查表

### 导航与焦点
| 按键 | 说明 |
|:---:|---|
| <kbd>0</kbd> / <kbd>/</kbd> | 激活 / 展开 / 收起模糊搜索栏 |
| <kbd>1</kbd> | 聚焦至主机列表面板 |
| <kbd>2</kbd> | 聚焦至主机详情面板 |
| <kbd>↑</kbd> <kbd>↓</kbd> / <kbd>j</kbd> <kbd>k</kbd> | 在主机列表中上下移动选择 |
| <kbd>Enter</kbd> | 通过 SSH 登录连接当前选中的主机 |
| <kbd>q</kbd> | 退出 lazyssh |

### 主机管理操作
| 按键 | 说明 |
|:---:|---|
| <kbd>a</kbd> | 新增主机配置 |
| <kbd>C</kbd> | 克隆当前选中的主机（自动去重别名并预填表单） |
| <kbd>v</kbd> | 从剪贴板粘贴并解析 SSH 命令快速建机 |
| <kbd>e</kbd> | 编辑当前主机（只读受管/引入主机受保护） |
| <kbd>d</kbd> | 删除当前选中的主机（带二次确认弹窗） |
| <kbd>t</kbd> | 编辑主机标签（仅更新元数据，不重写 SSH 配置文件） |
| <kbd>p</kbd> | 置顶 / 取消置顶当前主机 |

### 命令行辅助、SFTP 与隧道
| 按键 | 说明 |
|:---:|---|
| <kbd>c</kbd> | 复制当前主机的 SSH 连接命令至系统剪贴板 |
| <kbd>h</kbd> | 复制当前主机的 Host 别名至系统剪贴板 |
| <kbd>P</kbd> | 打开 SCP 命令生成向导（支持上传/下载，一键复制） |
| <kbd>F</kbd> | 为当前主机直接调用系统 `sftp` 客户端 |
| <kbd>f</kbd> | 打开交互式端口转发向导（本地、远程、动态 SOCKS5） |
| <kbd>x</kbd> | 停止当前主机正在运行的端口转发进程 |
| <kbd>g</kbd> | 对选中主机执行在线 Ping 状态探测 |
| <kbd>r</kbd> | 手动刷新后台主机数据及活跃会话 |

### 界面与偏好设置
| 按键 | 说明 |
|:---:|---|
| <kbd>s</kbd> | 切换排序字段（按别名 ↔ 按最近连接时间） |
| <kbd>S</kbd> | 反转当前排序顺序（升序 ↔ 降序） |
| <kbd>T</kbd> | 切换颜色主题（暗色 ↔ 亮色），偏好自动记忆 |

### 在主机编辑表单内
| 按键 | 说明 |
|:---:|---|
| <kbd>Ctrl+H</kbd> / <kbd>Ctrl+←</kbd> | 切换至上一个标签页 |
| <kbd>Ctrl+L</kbd> / <kbd>Ctrl+→</kbd> | 切换至下一个标签页 |
| <kbd>Ctrl+S</kbd> | 保存主机配置 |
| <kbd>Esc</kbd> | 取消编辑 / 返回主界面 |

---

## 🔐 安全与权限架构

lazyssh 遵循严格的安全工程原则：

- **绝不存储凭证**：lazyssh 不会提取、保存或传输你的私钥、通行短语（Passphrase）或密码。
- **系统级 OpenSSH 委托**：所有 SSH 连接、端口转发及 SFTP 交互均通过拉起系统原生 OpenSSH 二进制进程完成，完美兼容已有的 `ssh-agent`、1Password SSH Agent 以及硬件安全密钥（YubiKey / FIDO2）。
- **文件权限安全保障**：所有写操作均严格遵循 SSH 配置权限规范（`0600` / `0644`），防止因权限过宽被 OpenSSH 拒绝加载。
- **命令语法安全校验**：对剪贴板导入的命令和参数进行语法有效性隔离验证，防止潜在的恶意注入。

---

## 🤝 参与贡献

欢迎提交 Issue 反馈问题、提出建议或提交 Pull Request！

```bash
# 1. Fork 并克隆本仓库
git clone https://github.com/urzeye/lazyssh.git
cd lazyssh

# 2. 运行代码质量检查与测试
make test
make lint

# 3. 创建特性分支并提交 PR
git checkout -b feat/my-feature
```

---

## 📄 开源协议与鸣谢

本项目基于 [Apache License 2.0](LICENSE) 开源。

### 致谢与原项目溯源
- **[Adembc](https://github.com/Adembc)** — 原项目 [`lazyssh`](https://github.com/Adembc/lazyssh) 的创建者。我们对原作者杰出的架构奠基表示由衷的感谢。
- **[WhiteRoseLK/neossh](https://github.com/WhiteRoseLK/neossh)** — 为现代化 README 设计风格与社区 SSH 终端生态提供了宝贵参考。
- **[rivo/tview](https://github.com/rivo/tview)** 与 **[gdamore/tcell](https://github.com/gdamore/tcell)** — 提供了功能强大且底蕴扎实的终端 UI 框架。
- **[derailed/k9s](https://github.com/derailed/k9s)** 与 **[jesseduffield/lazydocker](https://github.com/jesseduffield/lazydocker)** — 启发了出色的键盘流终端交互设计范式。

<div align="center">
  <sub>Maintained with ❤️ by <a href="https://github.com/urzeye">urzeye</a> and community contributors.</sub>
</div>
