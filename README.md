<div align="center">
  <img src="./docs/logo.png" alt="lazyssh logo" width="600" height="600"/>
</div>

这是基于原版 `lazyssh` 的增强版本，目前主要补充了这些能力：

- 支持递归读取 `Include` / `conf.d` 形式的 SSH 配置，不再局限于单一 `~/.ssh/config`
- 支持 managed 模式：从 root config 实时读取完整配置树，同时只把新增 / 修改写回指定的 managed 文件
- 对来自 include 文件或其他非托管文件的主机做了只读保护，并在列表中标记来源，避免误编辑外部配置
- 新增 `--sshconfig`、`--managed-mode`、`--managed-sshconfig` 参数，适合和 `config.local` / `conf.d` 工作流配合
- 搜索改为更实用的模糊相关性排序，按别名、主机、用户、标签综合匹配
- 增强了日常使用体验：可折叠搜索栏、`0/1/2` 面板切换、复制 Host、记住排序方式
- 合并了一批高价值修复：输入框退格恢复正常、鼠标点击后 `j/k` 仍可导航、用户名支持 `@` 和 `:`、列表别名对齐更稳定

---

lazyssh 是一个运行在终端中的交互式 SSH 管理器，灵感来自 `lazydocker` 和 `k9s`，但专门服务于 SSH 主机管理。
借助 lazyssh，你可以直接在终端里浏览、搜索、连接、编辑和整理 `~/.ssh/config` 中定义的服务器，无需再记忆 IP 地址，也不用频繁手敲长串 `ssh` / `scp` 命令，一切都围绕高效的键盘工作流展开。

---

## ✨ 功能特性

### 主机管理
- 从 `~/.ssh/config` 中读取并展示主机列表，支持滚动浏览
- 在 UI 中直接新增主机，覆盖常见 SSH 配置项
- 通过分标签页表单编辑已有主机配置
- 安全删除主机条目
- 支持置顶 / 取消置顶，便于固定常用主机
- 支持对主机执行 `ping`，快速判断在线状态

### 快速导航
- 支持按别名、IP、用户或标签进行模糊搜索
- 选中主机后按回车即可直接 SSH 登录
- 支持用标签组织主机，例如 `prod`、`dev`、`test`
- 支持按别名或最近连接时间排序，并可反转排序方向

### 高级 SSH 配置
- 支持端口转发：`LocalForward`、`RemoteForward`、`DynamicForward`
- 支持连接复用，加快后续连接速度
- 支持高级认证方式，例如公钥、密码、Agent Forwarding
- 支持安全相关配置，例如加密算法、MAC、密钥交换算法
- 支持代理配置，例如 `ProxyJump`、`ProxyCommand`
- 通过标签页组织大量 SSH 选项，编辑更清晰

### 密钥管理
- 自动发现本地 SSH 密钥，并提供自动补全
- 支持在多把密钥之间快速选择

### 计划中的能力
- 在本地与远端主机之间复制文件，并提供更友好的选择界面
- SSH 公钥部署能力：
  - 使用默认本地公钥，例如 `~/.ssh/id_ed25519.pub` 或 `~/.ssh/id_rsa.pub`
  - 手动粘贴自定义公钥
  - 生成新的密钥对并部署到目标主机
  - 自动追加到 `~/.ssh/authorized_keys`，并处理正确权限

---

## 🔐 安全说明

lazyssh 不会引入额外的安全风险。
它本质上只是对你现有 SSH 配置的一层 TUI / UI 封装。

- 所有 SSH 连接都通过系统自带的 `ssh` 二进制程序执行，也就是 OpenSSH
- lazyssh 不会存储、传输或修改你的私钥、密码或其他凭据
- 你已有的 `IdentityFile` 路径和 `ssh-agent` 工作流都可以照常使用
- lazyssh 只会读取并更新你的 SSH 配置文件；在首次修改前会自动创建备份
- 它会尽量保留原有文件权限，避免因为写回配置带来额外风险

## 🧭 Managed Mode

如果你的 SSH 配置结构类似这样：

```sshconfig
Include ~/.orbstack/ssh/config
Include ~/.ssh/config.d/*.conf
Include ~/.ssh/config.local
```

可以让 lazyssh 进入 managed 模式：

```bash
lazyssh \
  --sshconfig ~/.ssh/config \
  --managed-mode \
  --managed-sshconfig ~/.ssh/config.local
```

这个模式下：

- `--sshconfig` 指向 root config，lazyssh 会按它递归读取完整 `Include` 树
- `--managed-sshconfig` 指向真正可写的文件，例如 `~/.ssh/config.local`
- 来自 root config、`config.d`、OrbStack 或其他 include 文件的条目会显示为只读
- 只有来自 managed 文件的主机可以被新增、编辑、删除
- SSH 连接和端口转发仍然继续使用 root config，因此和系统里的 `ssh` 行为保持一致

## 🛡️ 配置安全：非破坏性写入与自动备份

- 非破坏性编辑：lazyssh 只写入必要的最小变更，并尽可能保留原有注释、空行、顺序和未触碰的配置项
- 原子写入：更新先写入临时文件，再原子替换原文件，尽量降低部分写入导致配置损坏的风险
- 首次备份：第一次修改某个受管 SSH 配置文件前，会在同目录生成一份 `<文件名>.original.backup`，例如 `config.original.backup` 或 `config.local.original.backup`
- 滚动备份：每次后续保存时，还会生成一份带时间戳的备份，例如 `~/.ssh/config.local-<timestamp>-lazyssh.backup`
- 备份轮换：应用最多保留 10 份滚动备份，超出的旧备份会自动清理

## 📷 截图预览

<div align="center">

### 🚀 启动画面
<img src="./docs/loader.png" alt="应用启动界面" width="800" />

应用启动时的加载页面

---

### 📋 主机管理面板
<img src="./docs/list server.png" alt="主机列表视图" width="900" />

主界面会展示所有已配置主机，包括状态信息、置顶主机和便捷导航能力

---

### 🔎 搜索
<img src="./docs/search.png" alt="模糊搜索主机" width="900" />

可按主机名、IP 地址、用户或标签快速检索目标主机

---

### ➕ 新增 / 编辑主机
<img src="./docs/add server.png" alt="新增主机界面" width="900" />

通过分标签页界面管理 SSH 连接，主要包括：

- **基础信息**：主机、用户、端口、密钥、标签
- **连接设置**：代理、超时、连接复用、规范化
- **转发设置**：端口转发、X11、Agent
- **认证设置**：密钥、密码、认证方式、算法选项
- **高级设置**：安全、加密、环境变量、调试选项

---

### 🔐 连接主机
<img src="./docs/ssh.png" alt="SSH 连接详情" width="900" />

直接从界面发起 SSH 连接

</div>

---

## 📦 安装

### 方式一：使用 mise 安装（推荐）

如果你已经安装了 `mise`，在本仓库发布 Release 后，可以直接通过 GitHub Release 资产安装：

```bash
mise use -g github:urzeye/lazyssh@latest
lazyssh
```

如果你希望固定某个版本，也可以显式指定版本号：

```bash
mise use -g github:urzeye/lazyssh@0.x.y
```

说明：

- 这里使用的是 `mise` 的 `github:` backend，会直接从 `urzeye/lazyssh` 的 Release 下载对应平台的二进制包
- `mise` 会把版本写入全局配置，后续在任意终端里都可以直接使用 `lazyssh`

### 方式二：从源码构建

```bash
git clone https://github.com/urzeye/lazyssh.git
cd lazyssh

# 构建
make build
./bin/lazyssh

# 或直接运行
make run
```

---

## ⌨️ 快捷键

| 按键 | 说明 |
| ---- | ---- |
| `/` | 展开 / 收起搜索栏 |
| `↑↓` / `j` `k` | 在主机列表中移动 |
| `Enter` | SSH 连接到当前选中主机 |
| `c` | 复制 SSH 命令到剪贴板 |
| `h` | 复制 Host 到剪贴板 |
| `g` | Ping 当前主机 |
| `r` | 刷新后台数据 |
| `a` | 新增主机 |
| `e` | 编辑主机 |
| `t` | 编辑标签 |
| `d` | 删除主机 |
| `p` | 置顶 / 取消置顶 |
| `s` | 切换排序字段 |
| `S` | 反转排序顺序 |
| `0` / `1` / `2` | 聚焦搜索栏 / 主机列表 / 详情面板 |
| `q` | 退出 |

**在主机表单中：**

| 按键 | 说明 |
| ---- | ---- |
| `Ctrl+H` | 切换到上一个标签页 |
| `Ctrl+L` | 切换到下一个标签页 |
| `Ctrl+S` | 保存 |
| `Esc` | 取消 |

提示：列表顶部的提示栏会显示最常用的快捷键，方便随时查看。

---

## 🙏 致谢

- 基于 [tview](https://github.com/rivo/tview) 与 [tcell](https://github.com/gdamore/tcell) 构建
- 设计灵感来自 [k9s](https://github.com/derailed/k9s) 和 [lazydocker](https://github.com/jesseduffield/lazydocker)
