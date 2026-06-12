# Agent Feishu / 飞书 Agent 通知助手

中文 | Agent Feishu 会把 Codex 和 Claude Code 的审批请求、任务完成、失败或需要关注的状态推送到飞书 / Lark。

English | Agent Feishu sends Codex and Claude Code approval prompts, task-completion notices, failure notices, and attention alerts to Feishu/Lark.

中文 | 它只负责推送通知，不会替你在 Codex 或 Claude Code 里点击批准。手机收到审批通知后，请回到当前 agent 对话里回复 `approved` 或 `rejected`。

English | It is push-only. It cannot approve anything inside Codex or Claude Code for you. When your phone receives an approval notice, return to the current agent chat and reply `approved` or `rejected`.

## Install / 安装

中文 | 普通用户不需要 PowerShell、本地网页、localhost、公网 IP 或内网穿透。下载后双击即可开始配置。

English | Normal users do not need PowerShell, a local web page, localhost, a public IP, or a tunnel. Download the release artifact and double-click it to start setup.

### Windows

中文 | 从 GitHub Releases 下载 `agent-feishu.exe`，然后双击运行。

English | Download `agent-feishu.exe` from GitHub Releases, then double-click it.

Windows 应用会自动完成这些事：

The Windows app will:

- 中文 | 复制自身到 `%LOCALAPPDATA%\agent-feishu\agent-feishu.exe`
- English | Copy itself to `%LOCALAPPDATA%\agent-feishu\agent-feishu.exe`
- 中文 | 打开飞书扫码流程，用于创建或连接一个飞书自建应用
- English | Open the Feishu QR flow for creating or connecting a self-built app
- 中文 | 把 App 凭证和接收人 ID 保存在本机
- English | Save the app credentials and receiver ID locally
- 中文 | 添加项目文件夹，并向 `AGENTS.md` / `CLAUDE.md` 写入通知规则
- English | Add project folders and append notification rules to `AGENTS.md` / `CLAUDE.md`
- 中文 | 发送一次可选的测试通知
- English | Send one optional test notice
- 中文 | 可选开启 Windows 开机启动，让它重启后继续驻留
- English | Optionally enable Windows startup so it stays resident after reboot

中文 | 关闭窗口会隐藏到系统托盘。需要彻底退出时，请点击窗口里的 `退出`。

English | Closing the window hides it to the system tray. Use the window's `退出` button to quit.

### macOS

中文 | 从 GitHub Releases 下载 `agent-feishu-macos.zip`。Apple Silicon Mac 使用 `agent-feishu-macos-arm64`，Intel Mac 使用 `agent-feishu-macos-amd64`。也可以打开 `Agent Feishu.app`，它会在 Terminal 中运行配置流程。

English | Download `agent-feishu-macos.zip` from GitHub Releases. On Apple Silicon Macs, use `agent-feishu-macos-arm64`; on Intel Macs, use `agent-feishu-macos-amd64`. You can also open `Agent Feishu.app`, which runs the setup flow in Terminal.

中文 | macOS 版本目前使用终端配置流程：它会在 Terminal 中显示飞书二维码，把凭证保存在本机，并把 Codex / Claude 规则写入项目文件夹。它暂时还没有 Windows 那种原生驻留窗口。

English | The macOS build currently uses a terminal setup flow. It prints the Feishu QR code in Terminal, saves credentials locally, and writes Codex/Claude rules to project folders. It does not yet include the Windows native resident window.

## Configure / 配置

中文 | 在 Windows 原生应用里点击 `生成二维码`，然后用飞书手机端扫码并确认。自建应用名称固定为 `飞书提醒agent`。消息通过飞书 IM API 发送。

English | In the Windows native app, click `生成二维码`, then scan the QR code with the Feishu mobile app and confirm. The self-built app name is fixed as `飞书提醒agent`. Messages are sent through the Feishu IM API.

中文 | 添加项目文件夹时，默认会同时启用 Codex 和 Claude 支持：Codex 规则写入 `AGENTS.md`，Claude Code 规则写入 `CLAUDE.md`。

English | Codex and Claude support is enabled by default when you add a project folder. The app writes Codex rules to `AGENTS.md` and Claude Code rules to `CLAUDE.md`.

生成的配置大致如下：

The generated config looks like this:

```json
{
  "feishu_app_id": "cli_xxx",
  "feishu_app_secret": "local-only-secret",
  "feishu_receive_id": "ou_xxx",
  "feishu_receive_type": "open_id",
  "feishu_tenant_brand": "feishu",
  "default_agent": "Codex",
  "project_folders": []
}
```

Windows 默认配置路径：

Default config path on Windows:

```text
%USERPROFILE%\.agent-feishu.json
```

macOS 默认配置路径：

Default config path on macOS:

```text
~/.agent-feishu.json
```

中文 | 不要把真实 App 凭证或 token 提交到代码仓库。

English | Do not commit real app credentials or tokens.

## Add Project Folders / 添加项目文件夹

中文 | Windows 用户可以双击 `agent-feishu.exe`，然后在原生应用的 `项目文件夹` 区域添加项目。

English | On Windows, double-click `agent-feishu.exe`, then use the `项目文件夹` section in the native app.

中文 | macOS 用户可以从 `Agent Feishu.app` 或 Terminal 启动配置流程，然后按提示粘贴项目文件夹路径。

English | On macOS, run setup from `Agent Feishu.app` or from Terminal, then paste project folders when prompted.

中文 | 默认目标是 `Codex + Claude`，也就是同时更新：

English | The default target is `Codex + Claude`, which updates both:

```text
AGENTS.md
CLAUDE.md
```

高级用户也可以在终端里添加项目：

Advanced users can also add folders from a terminal:

```powershell
agent-feishu.exe projects add "E:\path\to\project"
```

macOS:

```bash
agent-feishu projects add "/path/to/project"
```

只写入某一个目标：

Choose one target:

```powershell
agent-feishu.exe projects add "E:\path\to\project" --target codex
agent-feishu.exe projects add "E:\path\to\project" --target claude
```

macOS:

```bash
agent-feishu projects add "/path/to/project" --target codex
agent-feishu projects add "/path/to/project" --target claude
```

插入的规则块会带有这些标记：

The inserted block is marked with:

```text
<!-- BEGIN:agent-feishu -->
<!-- END:agent-feishu -->
```

中文 | 重复添加同一个项目时，会更新已有规则块，不会重复插入。

English | Adding the same project again updates the existing block instead of duplicating it.

## Test Push / 测试推送

中文 | 日常审批和完成通知由 Codex / Claude 项目规则自动触发。配置完成后，可以在应用里发送一次测试通知，确认飞书消息能正常送达。

English | Daily approval and completion notices are triggered by the Codex/Claude project rules. After setup, you can send one test notice from the app to confirm that Feishu delivery works.

## Approval Notice / 审批通知

中文 | 审批通知必须发送原始审批文本，不要改写、翻译或总结。

English | Approval notices must send the exact original approval text. Do not rewrite, translate, or summarize it.

```powershell
@"
Run command with escalated permissions: npm run deploy
Justification: deploy current build after checks passed.
"@ | agent-feishu.exe approval --stdin --agent Codex --title "Codex approval request" --risk high
```

飞书消息会包含：

The Feishu message includes:

- 中文 | agent 名称
- English | Agent name
- 中文 | 风险等级
- English | Risk level
- 中文 | 当前工作目录
- English | Current working directory
- 中文 | 主机名
- English | Host name
- 中文 | 原始文本的 SHA256
- English | SHA256 of the original text
- 中文 | 原始审批请求
- English | The original approval request
- 中文 | 提醒用户回到 Codex / Claude 对话里回复
- English | Instruction to reply in the Codex/Claude chat

Dry run:

```powershell
agent-feishu.exe approval --text "exact approval text" --dry-run
```

macOS 命令基本一致，只是不带 `.exe`：

macOS uses the same commands without `.exe`:

```bash
agent-feishu approval --text "exact approval text" --dry-run
```

## Task Done Notice / 任务状态通知

中文 | 任务完成、失败或需要用户关注时，可以发送状态通知。

English | Send a status notice when a task completes, fails, or needs user attention.

```powershell
agent-feishu.exe done --agent Codex --status success --title "Task complete" --summary "Finished the requested work."
```

常用状态：

Useful statuses:

```text
success
failed
attention
info
```

添加更多说明行：

Add detail lines:

```powershell
agent-feishu.exe done --status failed --title "Build failed" --summary "npm run build failed." --detail "Check terminal output." --detail "No files were deployed."
```

macOS:

```bash
agent-feishu done --agent Codex --status success --title "Task complete" --summary "Finished."
```

## Codex / Claude Instruction / Codex 与 Claude 规则

中文 | 添加项目文件夹时，应用会自动写入规则。需要手动添加时，可以复制生成的规则片段。

English | The UI can append the rules automatically when you add a project folder. If you want to add them manually, copy the generated snippet.

Windows:

```text
%LOCALAPPDATA%\agent-feishu\AGENTS-snippet.md
```

macOS:

```text
~/Library/Application Support/agent-feishu/AGENTS-snippet.md
```

中文 | Codex 使用 `--agent Codex`，Claude Code 使用 `--agent Claude`。规则会要求 agent 在审批出现时先推送飞书通知，然后等待用户回到当前对话确认。

English | Codex uses `--agent Codex`; Claude Code uses `--agent Claude`. The rules tell the agent to push a Feishu notice when an approval request appears, then wait for the user to confirm in the current chat.

## Build From Source / 从源码构建

Requirements / 要求：

- Go 1.22+

Windows build:

```powershell
go build -trimpath -ldflags="-H windowsgui -s -w" -o dist\agent-feishu.exe .\cmd\agent-feishu
```

macOS cross-build from Windows/PowerShell:

```powershell
.\scripts\build-macos.ps1
```

中文 | GitHub Actions 会在每次 push 后构建 Windows 和 macOS 产物。

English | GitHub Actions builds Windows and macOS artifacts on every push.

## Release / 发布

中文 | 公开发布时，创建一个 tag，例如：

English | For public releases, create a tag such as:

```powershell
git tag v0.1.0
git push origin v0.1.0
```

中文 | 然后把生成的 `agent-feishu.exe` 和 `agent-feishu-macos.zip` 附加到 GitHub Release。

English | Then attach the generated `agent-feishu.exe` and `agent-feishu-macos.zip` artifacts to the GitHub Release.

## Security Notes / 安全说明

- 中文 | 本工具会通过飞书自建应用发送文本消息。
- English | This tool sends text through a Feishu self-built app.
- 中文 | App 凭证只保存在本地配置文件里。
- English | Store app credentials only in local config.
- 中文 | EXE 不会控制 Codex / Claude 的审批 UI。
- English | The EXE does not control the Codex/Claude approval UI.
- 中文 | 不要在审批文本里发送密码、密钥或其他敏感信息。
- English | Avoid sending passwords, keys, or other secrets inside approval text.
- 中文 | 检查消息结构时，优先使用 `--dry-run`。
- English | Use `--dry-run` mode when checking payload shape.
