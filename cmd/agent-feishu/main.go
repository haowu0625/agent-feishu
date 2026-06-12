package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/larksuite/oapi-sdk-go/v3/scene/registration"
)

var version = "0.1.0"

type Config struct {
	FeishuAppID       string   `json:"feishu_app_id,omitempty"`
	FeishuAppSecret   string   `json:"feishu_app_secret,omitempty"`
	FeishuReceiveID   string   `json:"feishu_receive_id,omitempty"`
	FeishuReceiveType string   `json:"feishu_receive_type,omitempty"`
	FeishuTenantBrand string   `json:"feishu_tenant_brand,omitempty"`
	DefaultAgent      string   `json:"default_agent,omitempty"`
	ProjectFolders    []string `json:"project_folders,omitempty"`
}

type FeishuPayload map[string]any

const (
	ruleBegin = "<!-- BEGIN:agent-feishu -->"
	ruleEnd   = "<!-- END:agent-feishu -->"
)

func main() {
	if len(os.Args) == 1 {
		if err := runUI(nil, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return runUI(nil, stdout)
	}

	switch args[0] {
	case "ui", "gui":
		return runUI(args[1:], stdout)
	case "setup":
		return runSetup(args[1:], stdin, stdout)
	case "approval":
		return runApproval(args[1:], stdin, stdout)
	case "done":
		return runDone(args[1:], stdout)
	case "test":
		return runTest(args[1:], stdout)
	case "projects", "project":
		return runProjects(args[1:], stdout)
	case "app":
		return runApp(args[1:], stdout)
	case "config":
		return runConfig(args[1:], stdout)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, version)
		return nil
	case "help", "--help", "-h":
		printUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runUI(args []string, stdout io.Writer) error {
	return runNativeGUI(args, stdout)
}

func runSetup(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	defaultAgent := fs.String("default-agent", "Codex", "notification source name")
	installDirFlag := fs.String("install-dir", "", "install directory")
	noTest := fs.Bool("no-test", false, "skip test message")
	dryRun := fs.Bool("dry-run", false, "show actions without writing files or sending")
	if err := fs.Parse(args); err != nil {
		return err
	}

	reader := bufio.NewReader(stdin)
	interactive := len(args) == 0
	if interactive {
		fmt.Fprintln(stdout, "Agent Feishu setup")
		fmt.Fprintln(stdout, "This will install Agent Feishu, register a Feishu app by QR scan, add project rules, and optionally send a test message.")
		fmt.Fprintln(stdout)
	}

	existingCfg, _ := loadConfig("")
	installDir := firstNonEmpty(*installDirFlag, defaultInstallDir())
	installPath := filepath.Join(installDir, executableFileName())
	if existingCfg.DefaultAgent != "" && *defaultAgent == "Codex" {
		*defaultAgent = existingCfg.DefaultAgent
	}
	if *defaultAgent == "" {
		*defaultAgent = "Codex"
	}
	if interactive {
		value, err := promptLine(reader, stdout, "Notification source name", *defaultAgent, false)
		if err != nil {
			return err
		}
		*defaultAgent = firstNonEmpty(value, *defaultAgent)
	}

	cfg := Config{
		FeishuAppID:       existingCfg.FeishuAppID,
		FeishuAppSecret:   existingCfg.FeishuAppSecret,
		FeishuReceiveID:   existingCfg.FeishuReceiveID,
		FeishuReceiveType: existingCfg.FeishuReceiveType,
		FeishuTenantBrand: existingCfg.FeishuTenantBrand,
		DefaultAgent:      *defaultAgent,
		ProjectFolders:    existingCfg.ProjectFolders,
	}
	configFile := configPath("")
	snippetPath := filepath.Join(installDir, "AGENTS-snippet.md")

	if *dryRun {
		return printJSON(stdout, map[string]any{
			"install_path": installPath,
			"config_path":  configFile,
			"snippet_path": snippetPath,
			"config": map[string]any{
				"feishu_app_id":       cfg.FeishuAppID,
				"feishu_app_secret":   redactSecret(cfg.FeishuAppSecret),
				"feishu_receive_id":   cfg.FeishuReceiveID,
				"feishu_receive_type": cfg.FeishuReceiveType,
				"default_agent":       cfg.DefaultAgent,
			},
		})
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return err
	}
	if err := copySelfTo(installPath); err != nil {
		return err
	}
	if err := writeConfig(configFile, cfg); err != nil {
		return err
	}
	if err := writeSnippet(snippetPath, installPath); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "Installed:", installPath)
	fmt.Fprintln(stdout, "Config:", configFile)
	fmt.Fprintln(stdout, "Agent instruction snippet:", snippetPath)

	if interactive {
		if cfg.FeishuAppID == "" || cfg.FeishuAppSecret == "" || cfg.FeishuReceiveID == "" {
			answer, err := promptLine(reader, stdout, "Register Feishu self-built app by QR scan now? (Y/n)", "Y", false)
			if err != nil {
				return err
			}
			if !strings.EqualFold(strings.TrimSpace(answer), "n") {
				if err := runAppRegister(nil, stdout); err != nil {
					return err
				}
				cfg, _ = loadConfig(configFile)
			}
		}
		if err := interactiveAddProjects(reader, stdout, installPath, &cfg, configFile); err != nil {
			return err
		}
	}

	if interactive && !*noTest {
		answer, err := promptLine(reader, stdout, "Send a Feishu test message now? (Y/n)", "Y", false)
		if err != nil {
			return err
		}
		if !strings.EqualFold(strings.TrimSpace(answer), "n") {
			if err := sendTestNotice(stdout, cfg, installDir, false); err != nil {
				return err
			}
		}
	}
	if interactive {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Done. You can close this window.")
		pause(stdin, stdout)
	}
	return nil
}

func runProjects(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("projects command requires add or list")
	}
	switch args[0] {
	case "add":
		return runProjectsAdd(args[1:], stdout)
	case "list":
		cfg, err := loadConfig("")
		if err != nil {
			return err
		}
		if len(cfg.ProjectFolders) == 0 {
			fmt.Fprintln(stdout, "No project folders configured.")
			return nil
		}
		for _, path := range cfg.ProjectFolders {
			fmt.Fprintln(stdout, path)
		}
		return nil
	default:
		return fmt.Errorf("unknown projects command %q", args[0])
	}
}

func runProjectsAdd(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("projects add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	target := fs.String("target", "both", "both, codex, or claude")
	exePath := fs.String("exe", installedExePath(), "agent-feishu executable path to write into rules")
	dryRun := fs.Bool("dry-run", false, "show files that would be updated")
	args = reorderFlags(args, map[string]bool{"target": true, "exe": true})
	if err := fs.Parse(args); err != nil {
		return err
	}
	paths := fs.Args()
	if len(paths) == 0 {
		return errors.New("projects add requires at least one project folder")
	}

	cfg, _ := loadConfig("")
	for _, projectPath := range paths {
		results, err := addProjectRules(projectPath, *target, *exePath, *dryRun)
		if err != nil {
			return err
		}
		for _, result := range results {
			fmt.Fprintln(stdout, result)
		}
		if !*dryRun {
			abs, _ := filepath.Abs(projectPath)
			cfg.ProjectFolders = appendUniquePath(cfg.ProjectFolders, abs)
		}
	}
	if !*dryRun {
		if err := writeConfig(configPath(""), cfg); err != nil {
			return err
		}
	}
	return nil
}

func runApp(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("app command requires register or use")
	}
	switch args[0] {
	case "register":
		return runAppRegister(args[1:], stdout)
	case "use":
		return runAppUse(args[1:], stdout)
	default:
		return fmt.Errorf("unknown app command %q", args[0])
	}
}

func runAppRegister(args []string, stdout io.Writer) error {
	return runAppRegisterWithQRCode(args, stdout, nil)
}

func runAppRegisterWithQRCode(args []string, stdout io.Writer, onQRCode func(url string, expires int)) error {
	fs := flag.NewFlagSet("app register", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	timeout := fs.Duration("timeout", 10*time.Minute, "registration timeout")
	_ = fs.Bool("no-open", true, "kept for compatibility; browser is not opened by default")
	openBrowser := fs.Bool("open-browser", false, "open QR URL in browser")
	terminalQR := fs.Bool("terminal-qr", runtime.GOOS != "windows", "print QR code in terminal")
	receiveType := fs.String("receive-type", "open_id", "open_id or chat_id")
	receiveID := fs.String("receive-id", "", "receiver id; defaults to scanning user open_id")
	dryRun := fs.Bool("dry-run", false, "show QR URL and result without writing config")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	var qrURL string
	var expires int
	fmt.Fprintln(stdout, "正在创建飞书自建应用。请用飞书手机端扫码/确认授权。")
	result, err := registration.RegisterApp(ctx, &registration.Options{
		Source: "agent-feishu",
		AppPreset: &registration.AppPreset{
			Name: "飞书提醒agent",
		},
		OnQRCode: func(info *registration.QRCodeInfo) {
			qrURL = info.URL
			expires = info.ExpireIn
			fmt.Fprintln(stdout, "扫码链接:", info.URL)
			fmt.Fprintf(stdout, "有效期: %d 秒\n", info.ExpireIn)
			if *terminalQR {
				if matrix, err := makeQRCodeMatrix(info.URL); err == nil {
					printTerminalQR(stdout, matrix)
				} else {
					fmt.Fprintln(stdout, "二维码生成失败:", err)
				}
			}
			if onQRCode != nil {
				onQRCode(info.URL, info.ExpireIn)
			}
			if *openBrowser {
				_ = openURL(info.URL)
			}
		},
		OnStatusChange: func(info *registration.StatusChangeInfo) {
			if info.Interval > 0 {
				fmt.Fprintf(stdout, "状态: %s，下次轮询 %d 秒后\n", info.Status, info.Interval)
			} else {
				fmt.Fprintln(stdout, "状态:", info.Status)
			}
		},
	})
	if err != nil {
		var regErr *registration.RegisterAppError
		if errors.As(err, &regErr) {
			return fmt.Errorf("飞书应用创建失败: %s %s", regErr.Code, regErr.Description)
		}
		return err
	}

	scannerOpenID := ""
	tenantBrand := ""
	if result.UserInfo != nil {
		scannerOpenID = result.UserInfo.OpenID
		tenantBrand = result.UserInfo.TenantBrand
	}
	finalReceiveID := strings.TrimSpace(*receiveID)
	if finalReceiveID == "" {
		finalReceiveID = scannerOpenID
	}
	finalReceiveType := normalizeReceiveType(*receiveType)
	if finalReceiveID == "" {
		return errors.New("飞书已创建应用，但没有拿到接收人 open_id；请用 --receive-id 手动指定")
	}

	if *dryRun {
		return printJSON(stdout, map[string]any{
			"qr_url":         qrURL,
			"expires":        expires,
			"app_id":         result.ClientID,
			"app_secret":     redactSecret(result.ClientSecret),
			"receive_id":     finalReceiveID,
			"receive_type":   finalReceiveType,
			"tenant_brand":   tenantBrand,
			"would_write_to": configPath(""),
		})
	}

	cfg, _ := loadConfig("")
	cfg.FeishuAppID = result.ClientID
	cfg.FeishuAppSecret = result.ClientSecret
	cfg.FeishuReceiveID = finalReceiveID
	cfg.FeishuReceiveType = finalReceiveType
	cfg.FeishuTenantBrand = tenantBrand
	if cfg.DefaultAgent == "" {
		cfg.DefaultAgent = "Codex"
	}
	if err := writeConfig(configPath(""), cfg); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "自建应用已保存。")
	fmt.Fprintln(stdout, "App ID:", result.ClientID)
	fmt.Fprintln(stdout, "接收对象:", finalReceiveType, finalReceiveID)
	return nil
}

func runAppUse(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("app use", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	appID := fs.String("app-id", "", "Feishu/Lark app id")
	appSecret := fs.String("app-secret", "", "Feishu/Lark app secret")
	receiveType := fs.String("receive-type", "open_id", "open_id or chat_id")
	receiveID := fs.String("receive-id", "", "open_id or chat_id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *appID == "" || *appSecret == "" || *receiveID == "" {
		return errors.New("app use requires --app-id, --app-secret, and --receive-id")
	}
	cfg, _ := loadConfig("")
	cfg.FeishuAppID = strings.TrimSpace(*appID)
	cfg.FeishuAppSecret = strings.TrimSpace(*appSecret)
	cfg.FeishuReceiveID = strings.TrimSpace(*receiveID)
	cfg.FeishuReceiveType = normalizeReceiveType(*receiveType)
	if cfg.DefaultAgent == "" {
		cfg.DefaultAgent = "Codex"
	}
	if err := writeConfig(configPath(""), cfg); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "自建应用配置已保存。")
	return nil
}

func reorderFlags(args []string, valueFlags map[string]bool) []string {
	var flags []string
	var values []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			values = append(values, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if idx := strings.Index(name, "="); idx >= 0 {
			name = name[:idx]
		}
		if !strings.Contains(arg, "=") && valueFlags[name] && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, values...)
}

func runApproval(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("approval", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	agent := fs.String("agent", "", "agent name")
	title := fs.String("title", "审批请求", "notice title")
	risk := fs.String("risk", "high", "risk level")
	summary := fs.String("summary", "", "optional summary")
	rawText := fs.String("text", "", "approval text")
	rawFile := fs.String("file", "", "file containing approval text")
	rawStdin := fs.Bool("stdin", false, "read approval text from stdin")
	cwd := fs.String("cwd", "", "working directory")
	configPath := fs.String("config", "", "config path")
	dryRun := fs.Bool("dry-run", false, "print payload without sending")
	if err := fs.Parse(args); err != nil {
		return err
	}

	text, err := readRawText(stdin, *rawText, *rawFile, *rawStdin)
	if err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		return errors.New("approval text is empty")
	}

	cfg, _ := loadConfig(*configPath)
	if *agent == "" {
		*agent = firstNonEmpty(cfg.DefaultAgent, "Codex")
	}
	if *cwd == "" {
		*cwd, _ = os.Getwd()
	}

	payload := buildApprovalPayload(*agent, *title, *risk, *summary, *cwd, text)
	return sendOrPrint(stdout, cfg, payload, *dryRun)
}

func runDone(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("done", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	agent := fs.String("agent", "", "agent name")
	status := fs.String("status", "success", "success, failed, attention, or info")
	title := fs.String("title", "任务完成", "notice title")
	summary := fs.String("summary", "", "task summary")
	detail := multiFlag{}
	cwd := fs.String("cwd", "", "working directory")
	configPath := fs.String("config", "", "config path")
	dryRun := fs.Bool("dry-run", false, "print payload without sending")
	fs.Var(&detail, "detail", "extra detail line, repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, _ := loadConfig(*configPath)
	if *agent == "" {
		*agent = firstNonEmpty(cfg.DefaultAgent, "Codex")
	}
	if *cwd == "" {
		*cwd, _ = os.Getwd()
	}

	payload := buildDonePayload(*agent, *status, *title, *summary, *cwd, detail)
	return sendOrPrint(stdout, cfg, payload, *dryRun)
}

func runTest(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	agent := fs.String("agent", "", "agent name")
	cwd := fs.String("cwd", "", "working directory")
	configPath := fs.String("config", "", "config path")
	dryRun := fs.Bool("dry-run", false, "print payload without sending")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, _ := loadConfig(*configPath)
	if *agent != "" {
		cfg.DefaultAgent = *agent
	}
	if *cwd == "" {
		*cwd, _ = os.Getwd()
	}
	return sendTestNotice(stdout, cfg, *cwd, *dryRun)
}

func runConfig(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("config command requires path, init, or show")
	}
	path := configPath("")
	switch args[0] {
	case "path":
		fmt.Fprintln(stdout, path)
		return nil
	case "init":
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config already exists: %s", path)
		}
		cfg := Config{
			FeishuAppID:       "",
			FeishuAppSecret:   "",
			FeishuReceiveID:   "",
			FeishuReceiveType: "open_id",
			DefaultAgent:      "Codex",
		}
		body, _ := json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(path, append(body, '\n'), 0600); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "created", path)
		return nil
	case "show":
		cfg, err := loadConfig("")
		if err != nil {
			return err
		}
		cfg.FeishuAppSecret = redactSecret(cfg.FeishuAppSecret)
		body, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Fprintln(stdout, string(body))
		return nil
	default:
		return fmt.Errorf("unknown config command %q", args[0])
	}
}

func defaultInstallDir() string {
	home, err := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "agent-feishu")
		}
		if err != nil {
			return filepath.Join(".", "agent-feishu")
		}
		return filepath.Join(home, "AppData", "Local", "agent-feishu")
	}
	if err != nil {
		return filepath.Join(".", "agent-feishu")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "agent-feishu")
	}
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "agent-feishu")
	}
	return filepath.Join(home, ".local", "share", "agent-feishu")
}

func promptLine(reader *bufio.Reader, stdout io.Writer, label string, defaultValue string, required bool) (string, error) {
	for {
		if defaultValue != "" {
			fmt.Fprintf(stdout, "%s [%s]: ", label, defaultValue)
		} else {
			fmt.Fprintf(stdout, "%s: ", label)
		}
		text, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		eof := errors.Is(err, io.EOF)
		value := strings.TrimSpace(text)
		if value == "" {
			value = defaultValue
		}
		if value != "" || !required {
			return value, nil
		}
		if eof {
			return "", fmt.Errorf("%s is required", label)
		}
		fmt.Fprintln(stdout, "This value is required.")
	}
}

func pause(stdin io.Reader, stdout io.Writer) {
	fmt.Fprint(stdout, "Press Enter to exit...")
	_, _ = bufio.NewReader(stdin).ReadString('\n')
}

func copySelfTo(dest string) error {
	src, err := os.Executable()
	if err != nil {
		return err
	}
	srcAbs, _ := filepath.Abs(src)
	destAbs, _ := filepath.Abs(dest)
	if strings.EqualFold(srcAbs, destAbs) {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(dest, 0755)
	}
	return nil
}

func writeConfig(path string, cfg Config) error {
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0600)
}

func writeSnippet(path string, exePath string) error {
	content := ruleBlock(exePath, "Codex")
	return os.WriteFile(path, []byte(content), 0644)
}

func interactiveAddProjects(reader *bufio.Reader, stdout io.Writer, exePath string, cfg *Config, configFile string) error {
	answer, err := promptLine(reader, stdout, "Add Codex/Claude rules to project folders now? (Y/n)", "Y", false)
	if err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(answer), "n") {
		return nil
	}
	fmt.Fprintln(stdout, "Paste one project folder path at a time. Leave blank when done.")
	for {
		projectPath, err := promptLine(reader, stdout, "Project folder path", "", false)
		if err != nil {
			return err
		}
		projectPath = strings.Trim(strings.TrimSpace(projectPath), `"`)
		if projectPath == "" {
			return nil
		}
		target, err := promptLine(reader, stdout, "Write rules to AGENTS.md, CLAUDE.md, or both? (agents/claude/both)", "both", false)
		if err != nil {
			return err
		}
		results, err := addProjectRules(projectPath, normalizeTarget(target), exePath, false)
		if err != nil {
			fmt.Fprintln(stdout, "Could not add project:", err)
			continue
		}
		for _, result := range results {
			fmt.Fprintln(stdout, result)
		}
		abs, _ := filepath.Abs(projectPath)
		cfg.ProjectFolders = appendUniquePath(cfg.ProjectFolders, abs)
		if err := writeConfig(configFile, *cfg); err != nil {
			return err
		}
	}
}

func addProjectRules(projectPath string, target string, exePath string, dryRun bool) ([]string, error) {
	projectPath = strings.Trim(strings.TrimSpace(projectPath), `"`)
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", abs)
	}
	target = normalizeTarget(target)
	files, err := targetFiles(abs, target)
	if err != nil {
		return nil, err
	}
	var results []string
	for _, file := range files {
		action, err := upsertRuleBlock(file.Path, ruleBlock(exePath, file.Agent), dryRun)
		if err != nil {
			return nil, err
		}
		results = append(results, fmt.Sprintf("%s: %s", action, file.Path))
	}
	return results, nil
}

type targetFile struct {
	Path  string
	Agent string
}

func targetFiles(projectPath string, target string) ([]targetFile, error) {
	switch normalizeTarget(target) {
	case "both":
		return []targetFile{
			{Path: filepath.Join(projectPath, "AGENTS.md"), Agent: "Codex"},
			{Path: filepath.Join(projectPath, "CLAUDE.md"), Agent: "Claude"},
		}, nil
	case "agents", "codex":
		return []targetFile{{Path: filepath.Join(projectPath, "AGENTS.md"), Agent: "Codex"}}, nil
	case "claude":
		return []targetFile{{Path: filepath.Join(projectPath, "CLAUDE.md"), Agent: "Claude"}}, nil
	default:
		return nil, fmt.Errorf("target must be both, agents, codex, or claude")
	}
}

func normalizeTarget(target string) string {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return "both"
	}
	if target == "agent" || target == "agents.md" || target == "ag" {
		return "agents"
	}
	if target == "claude.md" || target == "cl" {
		return "claude"
	}
	return target
}

func upsertRuleBlock(path string, block string, dryRun bool) (string, error) {
	existingBytes, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	existing := string(existingBytes)
	action := "created"
	var next string
	if strings.Contains(existing, ruleBegin) && strings.Contains(existing, ruleEnd) {
		start := strings.Index(existing, ruleBegin)
		end := strings.Index(existing[start:], ruleEnd)
		end += start + len(ruleEnd)
		next = strings.TrimRight(existing[:start], "\r\n") + "\n\n" + strings.TrimRight(block, "\r\n") + "\n" + strings.TrimLeft(existing[end:], "\r\n")
		action = "updated"
	} else if strings.TrimSpace(existing) == "" {
		next = strings.TrimRight(block, "\r\n") + "\n"
	} else {
		next = strings.TrimRight(existing, "\r\n") + "\n\n" + strings.TrimRight(block, "\r\n") + "\n"
		action = "appended"
	}
	if dryRun {
		return "would " + action, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(next), 0644); err != nil {
		return "", err
	}
	return action, nil
}

func ruleBlock(exePath string, agent string) string {
	agentArg := ""
	if strings.TrimSpace(agent) != "" {
		agentArg = " --agent " + agent
	}
	return fmt.Sprintf(`%s
## Agent Feishu Notifications

When an approval request appears, MUST call "%s" approval%s --stdin with the exact original approval text, then stop and wait for the user to approve or reject in this chat.

Before sending the final answer for any non-trivial task, MUST call "%s" done%s with --status success and a concise status summary.

If the task fails, is blocked, or needs user attention, MUST call "%s" done%s with --status failed or --status attention before replying.

Do not expose Feishu app credentials or tokens. Do not summarize, translate, or rewrite approval text before sending it.
%s
`, ruleBegin, exePath, agentArg, exePath, agentArg, exePath, agentArg, ruleEnd)
}

func installedExePath() string {
	return filepath.Join(defaultInstallDir(), executableFileName())
}

func executableFileName() string {
	if runtime.GOOS == "windows" {
		return "agent-feishu.exe"
	}
	return "agent-feishu"
}

func printTerminalQR(stdout io.Writer, matrix [][]bool) {
	if len(matrix) == 0 {
		return
	}
	quiet := 2
	size := len(matrix) + quiet*2
	module := func(y, x int) bool {
		my := y - quiet
		mx := x - quiet
		if my >= 0 && my < len(matrix) && mx >= 0 && mx < len(matrix[my]) {
			return matrix[my][mx]
		}
		return false
	}
	color := func(dark bool, foreground bool) int {
		if foreground {
			if dark {
				return 30
			}
			return 97
		}
		if dark {
			return 40
		}
		return 47
	}

	fmt.Fprintln(stdout)
	for y := 0; y < size; y += 2 {
		for x := 0; x < size; x++ {
			topDark := module(y, x)
			bottomDark := module(y+1, x)
			fmt.Fprintf(stdout, "\x1b[%d;%dm▀", color(topDark, true), color(bottomDark, false))
		}
		fmt.Fprint(stdout, "\x1b[0m\n")
	}
	fmt.Fprintln(stdout)
}

func appendUniquePath(paths []string, path string) []string {
	for _, item := range paths {
		if strings.EqualFold(item, path) {
			return paths
		}
	}
	return append(paths, path)
}

func buildApprovalPayload(agent, title, risk, summary, cwd, text string) FeishuPayload {
	digest := sha256.Sum256([]byte(text))
	lines := []string{
		fmt.Sprintf("[审批请求] %s", title),
		"来源：" + agent,
		"风险等级：" + displayRisk(risk),
		"工作目录：" + cwd,
		"主机：" + hostname(),
		"SHA256: " + hex.EncodeToString(digest[:]),
	}
	if strings.TrimSpace(summary) != "" {
		lines = append(lines, "", strings.TrimSpace(summary))
	}
	lines = append(lines, "", "原始审批请求：", text, "", "请回到 Codex/Claude 对话中回复：approved / rejected")
	return textPayload(strings.Join(lines, "\n"))
}

func buildDonePayload(agent, status, title, summary, cwd string, details []string) FeishuPayload {
	lines := []string{
		fmt.Sprintf("[%s] %s", agent, title),
		"状态：" + displayStatus(status),
		"工作目录：" + cwd,
		"主机：" + hostname(),
		"系统：" + runtime.GOOS + "/" + runtime.GOARCH,
		"时间：" + time.Now().Format(time.RFC3339),
	}
	if strings.TrimSpace(summary) != "" {
		lines = append(lines, "", strings.TrimSpace(summary))
	}
	for _, item := range details {
		if strings.TrimSpace(item) != "" {
			lines = append(lines, "- "+strings.TrimSpace(item))
		}
	}
	return textPayload(strings.Join(lines, "\n"))
}

func buildTestPayload(agent, cwd string) FeishuPayload {
	return buildDonePayload(
		firstNonEmpty(agent, "Codex"),
		"info",
		"测试通知",
		"这是一条 Agent Feishu 测试通知。收到这条消息，说明飞书推送已经配置成功。",
		cwd,
		nil,
	)
}

func sendTestNotice(stdout io.Writer, cfg Config, cwd string, dryRun bool) error {
	agent := firstNonEmpty(cfg.DefaultAgent, "Codex")
	payload := buildTestPayload(agent, cwd)
	return sendOrPrint(stdout, cfg, payload, dryRun)
}

func displayStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success":
		return "成功"
	case "failed":
		return "失败"
	case "attention":
		return "需要关注"
	case "info":
		return "信息"
	default:
		return strings.TrimSpace(status)
	}
}

func displayRisk(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "low":
		return "低"
	case "medium":
		return "中"
	case "high":
		return "高"
	case "critical":
		return "严重"
	default:
		return strings.TrimSpace(risk)
	}
}

func textPayload(text string) FeishuPayload {
	return FeishuPayload{
		"msg_type": "text",
		"content":  map[string]any{"text": text},
	}
}

func normalizeReceiveType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open_id", "user_id", "union_id", "email", "chat_id":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "open_id"
	}
}

func payloadText(payload FeishuPayload) string {
	if content, ok := payload["content"].(map[string]any); ok {
		if text, ok := content["text"].(string); ok {
			return text
		}
	}
	if content, ok := payload["content"].(map[string]string); ok {
		return content["text"]
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(body)
}

func mustJSONString(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(body)
}

type ioDiscardWriter struct{}

func (ioDiscardWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func sendOrPrint(stdout io.Writer, cfg Config, payload FeishuPayload, dryRun bool) error {
	if dryRun {
		out := map[string]any{
			"sent":          false,
			"delivery_mode": "app",
			"receive_type":  normalizeReceiveType(cfg.FeishuReceiveType),
			"receive_id":    cfg.FeishuReceiveID,
			"payload":       payload,
		}
		return printJSON(stdout, out)
	}
	resp, err := postFeishuApp(cfg, payloadText(payload))
	if err != nil {
		return err
	}
	if !responseOK(resp) {
		body, _ := json.Marshal(resp)
		return fmt.Errorf("Feishu app returned non-success response: %s", string(body))
	}
	fmt.Fprintln(stdout, "sent")
	return nil
}

func postFeishuApp(cfg Config, text string) (map[string]any, error) {
	if cfg.FeishuAppID == "" || cfg.FeishuAppSecret == "" {
		return nil, errors.New("missing Feishu app credentials; run agent-feishu app register")
	}
	if cfg.FeishuReceiveID == "" {
		return nil, errors.New("missing Feishu receive id; run agent-feishu app register")
	}
	token, err := tenantAccessToken(cfg.FeishuAppID, cfg.FeishuAppSecret)
	if err != nil {
		return nil, err
	}
	receiveType := normalizeReceiveType(cfg.FeishuReceiveType)
	body := map[string]any{
		"receive_id": cfg.FeishuReceiveID,
		"msg_type":   "text",
		"content":    mustJSONString(map[string]string{"text": text}),
	}
	endpoint := "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=" + receiveType
	return postJSON(endpoint, token, body)
}

func tenantAccessToken(appID, appSecret string) (string, error) {
	body := map[string]any{
		"app_id":     appID,
		"app_secret": appSecret,
	}
	resp, err := postJSON("https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", "", body)
	if err != nil {
		return "", err
	}
	if !responseOK(resp) {
		raw, _ := json.Marshal(resp)
		return "", fmt.Errorf("tenant_access_token failed: %s", raw)
	}
	if token, ok := resp["tenant_access_token"].(string); ok && token != "" {
		return token, nil
	}
	return "", errors.New("tenant_access_token missing in Feishu response")
}

func postJSON(endpoint string, bearerToken string, body any) (map[string]any, error) {
	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 12 * time.Second}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("Feishu HTTP %d: %s", res.StatusCode, string(raw))
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	return decoded, nil
}

func responseOK(resp map[string]any) bool {
	if n, ok := numberValue(resp["code"]); ok && n == 0 {
		return true
	}
	if n, ok := numberValue(resp["StatusCode"]); ok && n == 0 {
		return true
	}
	if msg, ok := resp["msg"].(string); ok && strings.EqualFold(msg, "success") {
		return true
	}
	return false
}

func numberValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}

func readRawText(stdin io.Reader, rawText, rawFile string, rawStdin bool) (string, error) {
	count := 0
	if rawText != "" {
		count++
	}
	if rawFile != "" {
		count++
	}
	if rawStdin {
		count++
	}
	if count != 1 {
		return "", errors.New("use exactly one of --text, --file, or --stdin")
	}
	if rawText != "" {
		return rawText, nil
	}
	if rawFile != "" {
		body, err := os.ReadFile(rawFile)
		return string(body), err
	}
	body, err := io.ReadAll(stdin)
	return string(body), err
}

func loadConfig(path string) (Config, error) {
	path = configPath(path)
	var cfg Config
	body, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(body, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func configPath(path string) string {
	if path != "" {
		return path
	}
	if fromEnv := os.Getenv("AGENT_FEISHU_CONFIG"); fromEnv != "" {
		return fromEnv
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".agent-feishu.json"
	}
	return filepath.Join(home, ".agent-feishu.json")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func openURL(rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

func printJSON(stdout io.Writer, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, string(body))
	return err
}

func redact(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 12 {
		return "***"
	}
	return value[:8] + "..." + value[len(value)-4:]
}

func redactSecret(value string) string {
	if value == "" {
		return ""
	}
	return "***"
}

func printUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, `agent-feishu sends Codex/Claude notices to Feishu.

Usage:
  agent-feishu ui
  agent-feishu setup
  agent-feishu projects add "E:\path\to\project"
  agent-feishu approval --stdin --agent Codex --title "Codex approval request"
  agent-feishu done --status success --title "Task complete" --summary "Finished."
  agent-feishu test
  agent-feishu config init
  agent-feishu config path

Common flags:
  --config path       Config JSON path
  --dry-run           Print payload without sending`)
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ", ")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
