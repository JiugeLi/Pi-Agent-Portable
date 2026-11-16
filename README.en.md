# Pi Agent Portable

The original Pi coding agent with a bundled runtime. Separate executables are provided for Windows and Linux x86_64. No additional Node.js or Pi installation is required.

Download: [Windows](https://github.com/JiugeLi/Pi-Agent-Portable/releases/latest/download/pi-windows-x64.exe) · [Linux](https://github.com/JiugeLi/Pi-Agent-Portable/releases/latest/download/pi-linux-x64) · [Releases and checksums](https://github.com/JiugeLi/Pi-Agent-Portable/releases)

## Quick start

Windows: double-click `pi-windows-x64.exe` to start through Windows PowerShell with a larger window and Consolas font. Starting from an existing terminal keeps that terminal. PowerShell profiles and saved console defaults are not changed.

Linux:

```bash
chmod +x ./pi-linux-x64
./pi-linux-x64
```

On first launch, the app creates a reusable `pi-project` folder beside the executable. It places `config.json`, both READMEs, saved sessions and the extracted runtime inside that project. Existing configuration files are preserved.

### Sign in to SoloMux

SoloMux is included with its default API endpoint:

1. Enter `/login solomux`, or select SoloMux from `/login` → API key.
2. Paste your API key.
3. Available chat models are loaded automatically. Use `/model` to select one. The default is `solo-auto`.

Login credentials are saved in the current workspace's `.pi-portable/agent/auth.json` and survive restarts. Use `/solomux-models` to refresh the catalog, or sign in again to replace a key.

### Terminal controls

The default display uses Pi's dark theme, padded input, hidden thinking text, and a footer that fits the window width. Saved display choices in `/settings` are preserved.

- **Ctrl+O**: expand or collapse tool output. You can also enter `/tools`, `/tools show`, or `/tools hide`.
- **Ctrl+T**: show or hide thinking text.
- **Esc**: stop the current operation.
- `/model`, `/resume`, `/settings`: select a model, reopen a session, or change the display.

Collapsed tools still show a short preview. Folding is provided by Pi, including when it runs in PowerShell. PowerShell itself does not fold terminal output. Noninteractive commands, JSON/RPC output, and Linux startup retain their normal launch behavior.

### Edit configuration

Alternatively, fill `apiKey` in `pi-project/config.json` and restart:

```json
{
  "baseUrl": "https://solomux.com/v1",
  "api": "openai-responses",
  "apiKey": "",
  "defaultModel": "solo-auto",
  "thinkingLevel": "medium"
}
```

Usually only `apiKey` needs changing. Optional `headers` and `compat` objects support gateway compatibility settings. `SOLOMUX_API_KEY` is also supported. Set `PI_PORTABLE_CONFIG` to use a different configuration file. An explicit nonempty config key takes precedence; leave it empty when managing credentials through `/login`.

## Project folder and saved sessions

Pi works in `pi-project`, regardless of which directory you launch the executable from. Repeated launches reuse the same config, login and sessions. The project is not a sandbox; existing HOME, SSH, Git and system tools remain available.

```text
pi-windows-x64.exe  (or pi-linux-x64)
pi-project/
  config.json
  README.en.md
  README.zh-CN.md
  .gitignore
  .pi-portable/
    agent/          saved login, settings and sessions/
    runtime/        extracted bundled runtime
```

Sessions are saved automatically. Enter `/resume` to select a past chat, `/new` to start another, or launch with `--resume` to open the picker. `--no-session` deliberately disables saving. Explicit upstream session-directory overrides still take precedence.

Use `--project <folder>` or `PI_PORTABLE_PROJECT` for a different project. Relative paths are resolved from the launch directory. Each project keeps its own config and sessions. `PI_PORTABLE_CONFIG` can still override the config file location. Run `--portable-diagnostics` to inspect the effective paths without displaying keys.

On default-project startup, old adjacent config files and old `.pi-portable/agent` data in the executable or launch directory are imported without changing their originals. Existing project config and logins are not overwritten. Legacy sessions keep their original working-directory metadata and appear in the new project's native resume catalogue; after switching working directories, use the picker's **Tab** scope switch to view all chats if needed. Only these known locations are checked; use `--session <file>` to open an older session stored elsewhere. Your existing `~/.pi/agent` is unchanged.

The generated project `.gitignore` excludes `config.json` and `.pi-portable/`. Do not share either: they can contain credentials and private conversations. Runtime extraction shows progress on first launch; later launches reuse the runtime.

An internet connection is required for model requests and catalog refreshes. No personal API key is embedded. Do not publicly share configuration or credential files after entering a key.

## Requirements

- Windows 10 / Windows Server 2016 or later, x86_64.
- Linux x86_64, glibc >= 2.28, kernel >= 4.18, with Bash and basic system tools. Alpine/musl is not supported.
- A writable project folder (the executable folder must be writable when creating the default project), with enough free space to extract the runtime.
