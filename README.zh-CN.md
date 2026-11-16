# Pi Agent Portable

自带运行环境的原版 Pi 命令行代理。Windows 和 Linux x86_64 分别提供一个可执行文件，不需要额外安装 Node.js 或 Pi。

下载：[Windows](https://github.com/JiugeLi/Pi-Agent-Portable/releases/latest/download/pi-windows-x64.exe) · [Linux](https://github.com/JiugeLi/Pi-Agent-Portable/releases/latest/download/pi-linux-x64) · [版本与校验值](https://github.com/JiugeLi/Pi-Agent-Portable/releases)

## 快速开始

Windows：双击 `pi-windows-x64.exe`，通过 Windows PowerShell 启动，使用较大的窗口和 Consolas 字体。从已有终端启动时继续使用原终端，不修改 PowerShell 配置文件或持久化控制台设置。

Linux：

```bash
chmod +x ./pi-linux-x64
./pi-linux-x64
```

首次启动会在程序旁自动创建并复用 `pi-project` 项目文件夹，配置 `config.json`、中英文 README、会话和运行环境均放在项目文件夹内。已有配置不会被覆盖。

### 登录 SoloMux

SoloMux 已内置默认接口地址，无须自己添加提供商：

1. 在 Pi 中输入 `/login solomux`，也可在 `/login` 的 API Key 列表中选择 SoloMux。
2. 粘贴自己的 API Key。
3. 程序自动加载账号可用的对话模型。输入 `/model` 选择模型，默认使用 `solo-auto`。

登录凭据会保存在当前工作区的 `.pi-portable/agent/auth.json` 中，重启后继续使用。`/solomux-models` 可手动刷新模型；重新登录可更换密钥。

### 终端操作

默认使用 Pi 深色主题、带留白的输入框、隐藏思考正文，以及适应窗口宽度的底部信息栏。通过 `/settings` 保存的显示设置会保留。

- **Ctrl+O**：展开或折叠工具输出；也可输入 `/tools`、`/tools show`、`/tools hide`。
- **Ctrl+T**：显示或隐藏思考正文。
- **Esc**：停止当前操作。
- `/model`、`/resume`、`/settings`：选择模型、恢复会话、修改显示设置。

折叠后仍保留简短预览。折叠由 Pi 提供，在 PowerShell 中同样可用；PowerShell 本身不负责折叠终端输出。非交互命令、JSON/RPC 输出和 Linux 的启动方式保持正常行为。

### 修改配置

也可打开 `pi-project/config.json`，填写 `apiKey`，重启生效：

```json
{
  "baseUrl": "https://solomux.com/v1",
  "api": "openai-responses",
  "apiKey": "",
  "defaultModel": "solo-auto",
  "thinkingLevel": "medium"
}
```

通常只需填写 `apiKey`。其他字段可保留默认值；`headers`、`compat` 是可选的网关兼容设置。也支持环境变量 `SOLOMUX_API_KEY`，或使用 `PI_PORTABLE_CONFIG` 指定其他配置文件。配置文件中的非空密钥优先；使用 `/login` 管理密钥时保持 `apiKey` 为空。

## 项目文件夹与历史会话

Pi 以 `pi-project` 为工作目录，从不同目录启动同一个程序，也会复用同一份配置、登录状态和会话。项目文件夹不是沙箱，原有 HOME、SSH、Git 和系统命令仍可使用。

```text
pi-windows-x64.exe（或 pi-linux-x64）
pi-project/
  config.json
  README.en.md
  README.zh-CN.md
  .gitignore
  .pi-portable/
    agent/          登录状态、设置和 sessions/ 历史会话
    runtime/        解压后的内置运行环境
```

会话自动保存。输入 `/resume` 选择历史会话，`/new` 开启新会话；也可启动时加 `--resume` 直接打开选择器。显式使用 `--no-session` 时不保存会话。原版 Pi 的显式会话目录覆盖设置仍优先生效。

通过 `--project <文件夹>` 或 `PI_PORTABLE_PROJECT` 选择其他项目。相对路径按启动目录解析，每个项目分别保存配置和会话。`PI_PORTABLE_CONFIG` 仍可指定其他配置文件。运行 `--portable-diagnostics` 可查看实际路径，输出不含密钥。

默认项目启动时，会检查程序目录与启动目录中的旧配置和旧 `.pi-portable/agent`，复制到项目内，保留原文件，不覆盖项目已有配置与登录状态。旧会话保留原工作目录信息，并列入新项目的原生恢复目录；恢复旧会话、切换工作目录后，如需查看全部会话，可在选择器中按 **Tab** 切换范围。仅检查这两个已知位置；其他位置的旧会话可通过 `--session <文件>` 打开，不修改原来的 `~/.pi/agent`。

自动生成的项目 `.gitignore` 会忽略 `config.json` 和 `.pi-portable/`。两者可能包含密钥和私人对话，请勿公开分享。首次启动显示运行环境解压进度，后续直接复用。

模型调用及刷新列表需要联网。程序不含个人密钥，填写过密钥的配置和凭据文件请勿公开分享。

## 系统要求

- Windows 10 / Windows Server 2016 及以上，x86_64。
- Linux x86_64，glibc >= 2.28，内核 >= 4.18；已有 Bash 和基本命令，不支持 Alpine/musl。
- 项目目录可写，创建默认项目时程序目录也需要可写，并有足够磁盘空间用于展开运行环境。
