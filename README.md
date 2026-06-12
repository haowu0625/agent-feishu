# Agent Feishu

Agent Feishu sends Codex and Claude Code approval prompts and task status notices to Feishu/Lark.

It is push-only. It cannot approve inside Codex or Claude Code for you. When your phone receives an approval notice, return to the current agent chat and answer approved or rejected.

## Install

Windows: download `agent-feishu.exe` from GitHub Releases, then double-click it.

macOS: download `agent-feishu-macos.zip` from GitHub Releases. On Apple Silicon Macs, use `agent-feishu-macos-arm64`; on Intel Macs, use `agent-feishu-macos-amd64`. You can also open `Agent Feishu.app`, which runs the setup flow in Terminal.

Normal users do not need PowerShell, a local web page, localhost, a public IP, or a tunnel.

The Windows app will:

- copy itself to `%LOCALAPPDATA%\agent-feishu\agent-feishu.exe`
- open the Feishu QR flow for creating or connecting a self-built app
- save the app credentials and receiver ID locally
- let you add project folders and append rules to `AGENTS.md` / `CLAUDE.md`
- let you send one optional test notice
- optionally enable Windows startup so it stays resident after reboot

Closing the window hides it to the system tray. Use the window's `退出` button to quit.

The macOS build currently uses a terminal setup flow. It prints the Feishu QR code in Terminal, saves credentials locally, and writes Codex/Claude rules to project folders. It does not yet include the Windows native resident window.

## Configure

Click `生成二维码` in the native app, then scan the QR code shown inside the app with the Feishu mobile app and confirm. The self-built app name is fixed as `飞书提醒agent`. The app sends messages through the Feishu IM API.

Codex and Claude support is enabled by default when you add a project folder. The app writes Codex rules to `AGENTS.md` and Claude Code rules to `CLAUDE.md`.

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

The default config path is:

```text
%USERPROFILE%\.agent-feishu.json
```

On macOS the same config is stored at:

```text
~/.agent-feishu.json
```

Do not commit real app credentials or tokens.

## Add Project Folders

Windows: double-click `agent-feishu.exe`, then use the `项目文件夹` section in the native app.

macOS: run setup from `Agent Feishu.app` or from Terminal, then paste project folders when prompted.

The default target is `Codex + Claude`, which updates both:

```text
AGENTS.md
CLAUDE.md
```

Advanced users can also add folders from a terminal:

```powershell
agent-feishu.exe projects add "E:\path\to\project"
```

macOS:

```bash
agent-feishu projects add "/path/to/project"
```

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

The inserted block is marked with:

```text
<!-- BEGIN:agent-feishu -->
<!-- END:agent-feishu -->
```

Adding the same project again updates the existing block instead of duplicating it.

## Test Push

Daily approval and completion notices are triggered by the Codex/Claude project rules.

## Approval Notice

Send the exact original approval request:

```powershell
@"
Run command with escalated permissions: npm run deploy
Justification: deploy current build after checks passed.
"@ | agent-feishu.exe approval --stdin --agent Codex --title "Codex approval request" --risk high
```

The Feishu message includes:

- agent name
- risk level
- current working directory
- host name
- SHA256 of the original text
- the original approval request
- instruction to reply in the Codex/Claude chat

Dry run:

```powershell
agent-feishu.exe approval --text "exact approval text" --dry-run
```

macOS uses the same commands without `.exe`:

```bash
agent-feishu done --agent Codex --status success --title "Task complete" --summary "Finished."
```

## Task Done Notice

```powershell
agent-feishu.exe done --agent Codex --status success --title "Task complete" --summary "Finished the requested work."
```

Useful statuses:

```text
success
failed
attention
info
```

Add detail lines:

```powershell
agent-feishu.exe done --status failed --title "Build failed" --summary "npm run build failed." --detail "Check terminal output." --detail "No files were deployed."
```

## Codex / Claude Instruction

The UI can append this automatically when you add a project folder. If you want to add it manually, use the block generated in:

```text
%LOCALAPPDATA%\agent-feishu\AGENTS-snippet.md
```

macOS:

```text
~/Library/Application Support/agent-feishu/AGENTS-snippet.md
```

## Build From Source

Requirements:

- Go 1.22+

Build:

```powershell
go build -trimpath -ldflags="-H windowsgui -s -w" -o dist\agent-feishu.exe .\cmd\agent-feishu
```

macOS cross-build from Windows/PowerShell:

```powershell
.\scripts\build-macos.ps1
```

GitHub Actions builds Windows and macOS artifacts on every push.

## Release

For public releases, create a tag such as:

```powershell
git tag v0.1.0
git push origin v0.1.0
```

Then attach the generated `agent-feishu.exe` and `agent-feishu-macos.zip` artifacts to the GitHub Release.

## Security Notes

- This tool sends text through a Feishu self-built app.
- Store app credentials only in local config.
- The EXE does not control Codex/Claude approval UI.
- Avoid sending secrets inside approval text.
- Use dry-run mode when checking payload shape.
