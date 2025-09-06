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
	apiURL       = "https://openrouter.ai/api/v1/chat/completions"
	defaultModel = "mistralai/mistral-7b-instruct:free"
)

type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

var (
	model   string
	debug   bool
	rootCmd = &cobra.Command{
		Use:   "how [query...]",
		Short: "A simple AI assistant for your CLI",
		Long:  "Ask 'how' to do something and get the shell command you need.\n\nSpecial commands:\n  how setup - Configure your OpenRouter API key\n  how set-model <model> - Set and persist the default model",
		Run:   runQuery,
		// Allow any arguments without treating them as subcommands
		Args: cobra.ArbitraryArgs,
	}
	setModelCmd = &cobra.Command{
		Use:   "set-model <model>",
		Short: "Set and persist the default OpenRouter model",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			m := strings.TrimSpace(args[0])
			if m == "" {
				fmt.Fprintf(os.Stderr, "Model cannot be empty.\n")
				os.Exit(1)
			}
			viper.Set("model", m)
			if err := viper.WriteConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok {
					configDir, _ := os.UserConfigDir()
					configPath := filepath.Join(configDir, "how", "config.yaml")
					if err := viper.WriteConfigAs(configPath); err != nil {
						fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
						os.Exit(1)
					}
				} else {
					fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
					os.Exit(1)
				}
			}
			fmt.Println("✅ Default model saved successfully!")
		},
	}
)

func init() {
	// Initialize viper for config management
	initConfig()

	// Add flags
	rootCmd.PersistentFlags().StringVar(&model, "model", defaultModel, "Specify an OpenRouter model to use (overrides saved default)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Print debug information (system prompt, user prompt, model)")

	// Add subcommands
	rootCmd.AddCommand(setModelCmd)
}

func initConfig() {
	// Set config name and paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting config directory: %v\n", err)
		os.Exit(1)
	}

	howConfigDir := filepath.Join(configDir, "how")
	viper.AddConfigPath(howConfigDir)

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(howConfigDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	// Read config
	viper.ReadInConfig()
}

func runSetup(cmd *cobra.Command, args []string) {
	fmt.Println("Welcome to 'how' setup.")
	fmt.Println("Please get your free API key from https://openrouter.ai/keys")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your OpenRouter.ai API key: ")
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "API key cannot be empty. Aborting.\n")
		os.Exit(1)
	}

	// Save API key to config
	viper.Set("api_key", apiKey)
	if err := viper.WriteConfig(); err != nil {
		// If config file doesn't exist, create it
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configDir, _ := os.UserConfigDir()
			configPath := filepath.Join(configDir, "how", "config.yaml")
			if err := viper.WriteConfigAs(configPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("✅ API key saved successfully!")
}

func runQuery(cmd *cobra.Command, args []string) {
	// Check if first argument is "setup" - handle it specially
	if len(args) > 0 && args[0] == "setup" {
		runSetup(cmd, args[1:])
		return
	}

	if len(args) == 0 {
		cmd.Help()
		return
	}

	// Get API key from config
	apiKey := viper.GetString("api_key")
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "Error: API key not found. Please run 'how setup' first.\n")
		os.Exit(1)
	}

	// Join query arguments
	query := strings.Join(args, " ")

	// Resolve model: flag > config > default
	effectiveModel := model
	if !cmd.Flags().Changed("model") {
		if m := viper.GetString("model"); strings.TrimSpace(m) != "" {
			effectiveModel = m
		}
	}

	// Optional debug info
	systemPrompt := buildSystemPrompt()
	if debug {
		fmt.Fprintln(os.Stderr, "=== DEBUG INFO ===")
		fmt.Fprintf(os.Stderr, "Model: %s\n", effectiveModel)
		fmt.Fprintln(os.Stderr, "User Prompt:")
		fmt.Fprintln(os.Stderr, query)
		fmt.Fprintln(os.Stderr, "System Prompt:")
		fmt.Fprintln(os.Stderr, systemPrompt)
		fmt.Fprintln(os.Stderr, "=== END DEBUG INFO ===")
	}

	// Query OpenRouter API
	command, err := queryOpenRouter(apiKey, query, effectiveModel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(command)
}

func queryOpenRouter(apiKey, query, model string) (string, error) {
	// Prepare request with dynamic system prompt
	systemPrompt := buildSystemPrompt()
	reqBody := OpenRouterRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: query},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/patrykgruszka/how-cli")
	req.Header.Set("X-Title", "how-cli")

	// Make request
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("network request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp OpenRouterResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return strings.TrimSpace(apiResp.Choices[0].Message.Content), nil
}

func getSystemInfo() string {
	var info []string

	// OS detection
	osName := runtime.GOOS
	switch osName {
	case "linux":
		osName = "Linux"
	case "darwin":
		osName = "macOS"
	case "windows":
		osName = "Windows"
	}
	info = append(info, fmt.Sprintf("- OS: %s", osName))

	// Architecture
	info = append(info, fmt.Sprintf("- Architecture: %s", runtime.GOARCH))

	// Shell detection
	if shell := os.Getenv("SHELL"); shell != "" {
		shellName := filepath.Base(shell)
		info = append(info, fmt.Sprintf("- Shell: %s", shellName))
	} else if runtime.GOOS == "windows" {
		// Check for PowerShell on Windows
		if _, err := exec.LookPath("pwsh"); err == nil {
			info = append(info, "- Shell: PowerShell Core")
		} else if _, err := exec.LookPath("powershell"); err == nil {
			info = append(info, "- Shell: Windows PowerShell")
		} else {
			info = append(info, "- Shell: Command Prompt")
		}
	}

	// Package manager detection
	packageManagers := detectPackageManagers()
	if len(packageManagers) > 0 {
		info = append(info, fmt.Sprintf("- Package Managers: %s", strings.Join(packageManagers, ", ")))
	}

	// User privilege detection
	privInfo := detectUserPrivileges()
	if privInfo != "" {
		info = append(info, fmt.Sprintf("- User Privileges: %s", privInfo))
	}

	return strings.Join(info, "\n")
}

func detectPackageManagers() []string {
	var managers []string

	packageManagerMap := map[string]string{
		"pacman":  "pacman (Arch)",
		"apt":     "apt (Debian/Ubuntu)",
		"dnf":     "dnf (Fedora)",
		"yum":     "yum (RHEL/CentOS)",
		"zypper":  "zypper (openSUSE)",
		"brew":    "Homebrew",
		"port":    "MacPorts",
		"winget":  "winget",
		"choco":   "Chocolatey",
		"scoop":   "Scoop",
		"snap":    "Snap",
		"flatpak": "Flatpak",
		"nix":     "Nix",
	}

	for cmd, name := range packageManagerMap {
		if _, err := exec.LookPath(cmd); err == nil {
			managers = append(managers, name)
		}
	}

	return managers
}

func detectUserPrivileges() string {
	var privileges []string

	// Check if running as root/admin
	currentUser, err := user.Current()
	if err == nil {
		if currentUser.Uid == "0" || currentUser.Username == "root" {
			privileges = append(privileges, "root")
		}
	}

	// Check sudo availability (Unix-like systems)
	if runtime.GOOS != "windows" {
		if _, err := exec.LookPath("sudo"); err == nil {
			privileges = append(privileges, "sudo available")
		}
	}

	// Check admin privileges on Windows
	if runtime.GOOS == "windows" {
		// Simple check - try to access a system directory
		if _, err := os.Stat("C:\\Windows\\System32\\config\\SAM"); err == nil {
			privileges = append(privileges, "administrator")
		} else {
			privileges = append(privileges, "standard user")
		}
	}

	if len(privileges) == 0 {
		return "standard user"
	}

	return strings.Join(privileges, ", ")
}

func buildSystemPrompt() string {
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

	systemInfo := getSystemInfo()
	return fmt.Sprintf(basePrompt, systemInfo)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
