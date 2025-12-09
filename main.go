package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	providerOpenRouter = "openrouter"
	providerOpenAI     = "openai"

	openRouterURL          = "https://openrouter.ai/api/v1/chat/completions"
	openRouterDefaultModel = "anthropic/claude-haiku-4.5"

	openAiURL          = "https://api.openai.com/v1/chat/completions"
	openAiDefaultModel = "gpt-4o"
)

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Sys interface {
	Env(key string) string
	LookPath(cmd string) (string, error)
	CurrentUser() (*user.User, error)
	GOOS() string
	GOARCH() string
	Stat(name string) (os.FileInfo, error)
}

type realSys struct{}

func (realSys) Env(key string) string                 { return os.Getenv(key) }
func (realSys) LookPath(cmd string) (string, error)   { return exec.LookPath(cmd) }
func (realSys) CurrentUser() (*user.User, error)      { return user.Current() }
func (realSys) GOOS() string                          { return runtime.GOOS }
func (realSys) GOARCH() string                        { return runtime.GOARCH }
func (realSys) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }

var defaultSys Sys = realSys{}

var (
	modelFlag string
	debug     bool
	rootCmd   = &cobra.Command{
		Use:   "how [query...]",
		Short: "A simple AI assistant for your CLI",
		Long:  "Ask 'how' to do something and get the shell command you need.\n\nCommands:\n  how setup - Configure provider and API key\n  how set-model <model> - Set default model",
		RunE:  runQuery,
		Args:  cobra.ArbitraryArgs,
	}
	setModelCmd = &cobra.Command{
		Use:   "set-model <model>",
		Short: "Set and persist the default model",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			m := strings.TrimSpace(args[0])
			if m == "" {
				fmt.Fprintln(os.Stderr, "Model cannot be empty.")
				os.Exit(1)
			}
			viper.Set("model", m)
			if err := saveConfig(); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ Default model saved successfully!")
		},
	}
)

func init() {
	initConfig()
	rootCmd.PersistentFlags().StringVar(&modelFlag, "model", "", "Override configured model")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Print debug information")
	rootCmd.AddCommand(setModelCmd)
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	howConfigDir := filepath.Join(configDir, "how")
	viper.AddConfigPath(howConfigDir)
	if err := os.MkdirAll(howConfigDir, 0755); err != nil {
		return
	}
	_ = viper.ReadInConfig()
}

func saveConfig() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(configDir, "how", "config.yaml")
	return viper.WriteConfigAs(configPath)
}

func runSetup(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Select AI Provider:")
	fmt.Println("1. OpenRouter (default)")
	fmt.Println("2. OpenAI")
	fmt.Print("Choice [1]: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	provider := providerOpenRouter
	if choice == "2" {
		provider = providerOpenAI
	}

	fmt.Printf("\nEnter your %s API key: ", provider)
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "API key cannot be empty.")
		os.Exit(1)
	}

	viper.Set("provider", provider)
	viper.Set("api_key", apiKey)

	// Clear specific model config to avoid using an invalid model ID for the new provider
	viper.Set("model", "")

	if err := saveConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Configuration saved successfully!")
}

func runQuery(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && args[0] == "setup" {
		runSetup(cmd, args[1:])
		return nil
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	apiKey := viper.GetString("api_key")
	if apiKey == "" {
		return fmt.Errorf("API key not found. Please run 'how setup'")
	}

	provider := viper.GetString("provider")
	if provider == "" {
		provider = providerOpenRouter
	}

	var endpoint, defaultModel string
	isRefererNeeded := false

	switch provider {
	case providerOpenAI:
		endpoint = openAiURL
		defaultModel = openAiDefaultModel
	default:
		endpoint = openRouterURL
		defaultModel = openRouterDefaultModel
		isRefererNeeded = true
	}

	effectiveModel := defaultModel
	if m := viper.GetString("model"); m != "" {
		effectiveModel = m
	}
	if modelFlag != "" {
		effectiveModel = modelFlag
	}

	query := strings.Join(args, " ")

	if debug {
		fmt.Fprintln(os.Stderr, "=== DEBUG INFO ===")
		fmt.Fprintf(os.Stderr, "Provider: %s\n", provider)
		fmt.Fprintf(os.Stderr, "Endpoint: %s\n", endpoint)
		fmt.Fprintf(os.Stderr, "Model: %s\n", effectiveModel)
		fmt.Fprintln(os.Stderr, "System Prompt:\n", buildSystemPrompt())
		fmt.Fprintln(os.Stderr, "=== END DEBUG INFO ===")
	}

	command, err := queryLLM(endpoint, apiKey, query, effectiveModel, isRefererNeeded)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), command)
	return err
}

func queryLLM(endpoint, apiKey, query, model string, refererNeeded bool) (string, error) {
	reqBody := ChatRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: buildSystemPrompt()},
			{Role: "user", Content: query},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if refererNeeded {
		req.Header.Set("HTTP-Referer", "https://github.com/patrykgruszka/how-cli")
		req.Header.Set("X-Title", "how-cli")
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var apiResp ChatResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", err
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return strings.TrimSpace(apiResp.Choices[0].Message.Content), nil
}

func getSystemInfo() string { return getSystemInfoWith(defaultSys) }

func getSystemInfoWith(sys Sys) string {
	var info []string

	osName := sys.GOOS()
	switch osName {
	case "linux":
		osName = "Linux"
	case "darwin":
		osName = "macOS"
	case "windows":
		osName = "Windows"
	}
	info = append(info, fmt.Sprintf("- OS: %s", osName))
	info = append(info, fmt.Sprintf("- Architecture: %s", sys.GOARCH()))

	if shell := sys.Env("SHELL"); shell != "" {
		info = append(info, fmt.Sprintf("- Shell: %s", filepath.Base(shell)))
	} else if sys.GOOS() == "windows" {
		if _, err := sys.LookPath("pwsh"); err == nil {
			info = append(info, "- Shell: PowerShell Core")
		} else if _, err := sys.LookPath("powershell"); err == nil {
			info = append(info, "- Shell: Windows PowerShell")
		} else {
			info = append(info, "- Shell: Command Prompt")
		}
	}

	if managers := detectPackageManagersWith(sys); len(managers) > 0 {
		info = append(info, fmt.Sprintf("- Package Managers: %s", strings.Join(managers, ", ")))
	}

	if priv := detectUserPrivilegesWith(sys); priv != "" {
		info = append(info, fmt.Sprintf("- User Privileges: %s", priv))
	}

	return strings.Join(info, "\n")
}

func detectPackageManagersWith(sys Sys) []string {
	var managers []string
	pmMap := map[string]string{
		"pacman": "pacman (Arch)", "apt": "apt (Debian)", "dnf": "dnf (Fedora)",
		"yum": "yum (RHEL)", "zypper": "zypper (openSUSE)", "brew": "Homebrew",
		"port": "MacPorts", "winget": "winget", "choco": "Chocolatey",
		"scoop": "Scoop", "snap": "Snap", "flatpak": "Flatpak", "nix": "Nix",
	}
	for cmd, name := range pmMap {
		if _, err := sys.LookPath(cmd); err == nil {
			managers = append(managers, name)
		}
	}
	return managers
}

func detectUserPrivilegesWith(sys Sys) string {
	var privs []string
	if u, err := sys.CurrentUser(); err == nil && (u.Uid == "0" || u.Username == "root") {
		privs = append(privs, "root")
	}
	if sys.GOOS() != "windows" {
		if _, err := sys.LookPath("sudo"); err == nil {
			privs = append(privs, "sudo available")
		}
	}
	if sys.GOOS() == "windows" {
		if _, err := sys.Stat("C:\\Windows\\System32\\config\\SAM"); err == nil {
			privs = append(privs, "administrator")
		} else {
			privs = append(privs, "standard user")
		}
	}
	if len(privs) == 0 {
		return "standard user"
	}
	return strings.Join(privs, ", ")
}

func buildSystemPrompt() string {
	return buildSystemPromptFrom(getSystemInfo())
}

func buildSystemPromptFrom(systemInfo string) string {
	basePrompt := `You are an expert shell command assistant. Output exactly one single-line command that can be pasted into the user's shell and run as-is to complete the task.

System Info:
%s

Strict output policy:
1. Output ONLY the raw command on a single line. No commentary, no code fences, no leading/trailing spaces.
2. Do NOT prefix with explanations (e.g., "Sure", "Run:") and do NOT use markdown.
3. Prefer non-interactive, idempotent, and safe defaults; use flags that avoid prompts (-y, --noconfirm) when appropriate.
4. Respect the detected OS, shell, and available package managers above. Prefer the most standard/common manager for that OS if multiple are present.
5. If elevated privileges are required and sudo is available (Unix-like), prefix with sudo
6. If the request is ambiguous, choose the most common and safest interpretation and produce a single best command.
7. If no single applicable command exists, output a very short direct answer (still a single line).
8. Avoid destructive operations unless explicitly requested; when editing files, prefer in-place options that create backups when available.
9. Quote paths and arguments safely for the detected shell
10. Favor cross-distro commands when possible; otherwise select the correct package manager from the detected list.

Examples:
- Request: install ripgrep
  Response (apt): sudo apt update -y && sudo apt install -y ripgrep
- Request: find and remove node_modules directories
  Response (POSIX): find . -type d -name node_modules -prune -exec rm -rf {} +`

	return fmt.Sprintf(basePrompt+"\n", systemInfo)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
