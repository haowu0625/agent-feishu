# 飞书 Agent 通知助手

**语言:** [English](README.md) | 简体中文

Agent Feishu 会把 Codex 和 Claude Code 的审批请求、任务完成、失败或需要关注的状态推送到飞书 / Lark。

它只负责推送通知，不会替你在 Codex 或 Claude Code 里点击批准。手机收到审批通知后，请回到当前 agent 对话里回复 `approved` 或 `rejected`。

不同于飞书中控台或桥接类工具，Agent Feishu 不远程运行、不替你审批、不接管 Codex / Claude。它只负责把本地 agent 工作流里的通知推送到飞书 / Lark。

## 安装

Windows：从 GitHub Releases 下载 `agent-feishu.exe`，然后双击运行。

macOS：从 GitHub Releases 下载 `agent-feishu-macos.zip`。Apple Silicon Mac 使用 `agent-feishu-macos-arm64`，Intel Mac 使用 `agent-feishu-macos-amd64`。也可以打开 `Agent Feishu.app`，它会在 Terminal 中运行配置流程。

普通用户不需要 PowerShell、本地网页、localhost、公网 IP 或内网穿透。

Windows 应用会自动完成这些事：

- 打开飞书扫码流程，用于创建或连接一个飞书自建应用
- 把 App 凭证和接收人 ID 保存在本机
- 添加项目文件夹，并向 `AGENTS.md` / `CLAUDE.md` 写入通知规则
- 发送一次可选的测试通知
- 可选开启 Windows 开机启动；只有开启后才会复制自身到 `%LOCALAPPDATA%\agent-feishu\agent-feishu.exe`

关闭窗口会隐藏到系统托盘。需要彻底退出时，请点击窗口里的 `退出`。

macOS 版本目前使用终端配置流程：它会在 Terminal 中显示飞书二维码，把凭证保存在本机，并把 Codex / Claude 规则写入项目文件夹。它暂时还没有 Windows 那种原生驻留窗口。

## 配置

在 Windows 原生应用里点击 `生成二维码`，然后用飞书手机端扫码并确认。自建应用名称固定为 `飞书提醒agent`。消息通过飞书 IM API 发送。

添加项目文件夹时，默认会同时启用 Codex 和 Claude 支持：Codex 规则写入 `AGENTS.md`，Claude Code 规则写入 `CLAUDE.md`。

生成的配置大致如下：

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

```text
%USERPROFILE%\.agent-feishu.json
```

macOS 默认配置路径：

```text
~/.agent-feishu.json
```

不要把真实 App 凭证或 token 提交到代码仓库。

## 添加项目文件夹

Windows 用户可以双击 `agent-feishu.exe`，然后在原生应用的 `项目文件夹` 区域添加项目。

macOS 用户可以从 `Agent Feishu.app` 或 Terminal 启动配置流程，然后按提示粘贴项目文件夹路径。

默认目标是 `Codex + Claude`，也就是同时更新：

```text
AGENTS.md
CLAUDE.md
```

高级用户也可以在终端里添加项目：

```powershell
agent-feishu.exe projects add "E:\path\to\project"
```

macOS：

```bash
agent-feishu projects add "/path/to/project"
```

只写入某一个目标：

```powershell
agent-feishu.exe projects add "E:\path\to\project" --target codex
agent-feishu.exe projects add "E:\path\to\project" --target claude
```

macOS：

```bash
agent-feishu projects add "/path/to/project" --target codex
agent-feishu projects add "/path/to/project" --target claude
```

插入的规则块会带有这些标记：

```text
<!-- BEGIN:agent-feishu -->
<!-- END:agent-feishu -->
```

重复添加同一个项目时，会更新已有规则块，不会重复插入。

添加项目规则后，请在该项目中新建 Codex 或 Claude Code 会话，规则才能稳定生效。已有对话不一定会自动重新读取 `AGENTS.md` 或 `CLAUDE.md`。

## 测试推送

日常审批和完成通知由 Codex / Claude 项目规则自动触发。配置完成后，可以在应用里发送一次测试通知，确认飞书消息能正常送达。

## 审批通知

审批通知必须发送原始审批文本，不要改写、翻译或总结。

```powershell
@"
Run command with escalated permissions: npm run deploy
Justification: deploy current build after checks passed.
"@ | agent-feishu.exe approval --stdin --agent Codex --title "Codex approval request" --risk high
```

飞书消息会包含：

- agent 名称
- 风险等级
- 当前工作目录
- 主机名
- 原始文本的 SHA256
- 原始审批请求
- 提醒用户回到 Codex / Claude 对话里回复

Dry run：

```powershell
agent-feishu.exe approval --text "exact approval text" --dry-run
```

macOS 命令基本一致，只是不带 `.exe`：

```bash
agent-feishu approval --text "exact approval text" --dry-run
```

## 任务状态通知

```powershell
agent-feishu.exe done --agent Codex --status success --title "Task complete" --summary "Finished the requested work."
```

常用状态：

```text
success
failed
attention
info
```

添加更多说明行：

```powershell
agent-feishu.exe done --status failed --title "Build failed" --summary "npm run build failed." --detail "Check terminal output." --detail "No files were deployed."
```

macOS：

```bash
agent-feishu done --agent Codex --status success --title "Task complete" --summary "Finished."
```

## Codex / Claude 规则

添加项目文件夹时，应用会自动写入规则。需要手动添加时，可以复制生成的规则片段。

Windows：

```text
%LOCALAPPDATA%\agent-feishu\AGENTS-snippet.md
```

macOS：

```text
~/Library/Application Support/agent-feishu/AGENTS-snippet.md
```

Codex 使用 `--agent Codex`，Claude Code 使用 `--agent Claude`。规则会要求 agent 在审批出现时先推送飞书通知，然后等待用户回到当前对话确认。

## 从源码构建

要求：

- Go 1.22+

Windows 构建：

```powershell
go build -trimpath -ldflags="-H windowsgui -s -w" -o dist\agent-feishu.exe .\cmd\agent-feishu
```

在 Windows / PowerShell 中交叉构建 macOS 版本：

```powershell
.\scripts\build-macos.ps1
```

GitHub Actions 会在每次 push 后构建 Windows 和 macOS 产物。

## 发布

公开发布时，创建一个 tag，例如：

```powershell
git tag v0.1.0
git push origin v0.1.0
```

然后把生成的 `agent-feishu.exe` 和 `agent-feishu-macos.zip` 附加到 GitHub Release。

## 杀软误报说明

部分杀软可能会拦截未签名的 Go 单文件程序，尤其是它包含联网发送消息、本地保存配置、可选开机启动这些行为时。Agent Feishu 只负责推送通知，不会远程执行命令，也不会替你审批。

在找回被隔离文件前，请先确认文件来自本仓库的 GitHub Release，并用同一个 Release 里的 `SHA256SUMS.txt` 校验文件哈希。正式分发时，最好使用可信代码签名证书给 Windows EXE 签名，这能明显降低误报概率。

## 安全说明

- 本工具会通过飞书自建应用发送文本消息。
- App 凭证只保存在本地配置文件里。
- EXE 不会控制 Codex / Claude 的审批 UI。
- 不要在审批文本里发送密码、密钥或其他敏感信息。
- 检查消息结构时，优先使用 `--dry-run`。
- Windows 开机启动是可选项，只有用户开启时才会写入当前用户的 Run 注册表项。
