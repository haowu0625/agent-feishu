---
name: agent-feishu
description: Use the agent-feishu executable to push Codex/Claude approval prompts and task-completion notices to Feishu/Lark. Use when the user wants Feishu notifications for approval requests, done/failed status, long-running task completion, or agent attention alerts.
---

# Agent Feishu

Use this skill when Codex or Claude should notify the user in Feishu.

The executable is a notifier, not a permission bypass. It cannot click Codex/Claude host-level approval buttons. For approval requests, send the exact approval text to Feishu, then stop and wait for the user to approve or reject in the current chat.

On Windows, double-clicking the EXE opens a native app window where users can connect a Feishu self-built app by QR scan, add project folders, and keep the app resident in the tray. On macOS, the current build uses a Terminal setup flow and prints the QR code there.

Default installed Windows path after double-click setup:

```text
%LOCALAPPDATA%\agent-feishu\agent-feishu.exe
```

Default installed macOS path:

```text
~/Library/Application Support/agent-feishu/agent-feishu
```

Prefer using the exact absolute path from the generated `AGENTS-snippet.md`.

The EXE can add project rules automatically:

```powershell
& "$env:LOCALAPPDATA\agent-feishu\agent-feishu.exe" projects add "E:\path\to\project"
```

macOS:

```bash
"$HOME/Library/Application Support/agent-feishu/agent-feishu" projects add "/path/to/project"
```

This appends or updates the marked `agent-feishu` block in `AGENTS.md` and `CLAUDE.md`. Codex rules use `--agent Codex`; Claude Code rules use `--agent Claude`.

## Approval Prompt

Copy the original approval prompt exactly:

```powershell
@"
<exact approval request text>
"@ | & "$env:LOCALAPPDATA\agent-feishu\agent-feishu.exe" approval --stdin --agent Codex --title "Codex approval request" --risk high
```

## Task Done

```powershell
& "$env:LOCALAPPDATA\agent-feishu\agent-feishu.exe" done --agent Codex --status success --title "Task complete" --summary "Finished the requested work."
```

Use `--dry-run` to inspect payloads without sending.

## Host Instruction

Prefer letting the EXE add this with `projects add`. If adding manually, use `--agent Codex` in `AGENTS.md` and `--agent Claude` in `CLAUDE.md`.

```markdown
When an approval request appears, call `%LOCALAPPDATA%\agent-feishu\agent-feishu.exe approval --agent Codex --stdin` with the exact original approval text, then stop and wait for the user to approve or reject in this chat. When a substantial task completes, fails, or needs attention, call `%LOCALAPPDATA%\agent-feishu\agent-feishu.exe done --agent Codex` with a concise status summary. Do not expose Feishu app credentials or tokens.
```
