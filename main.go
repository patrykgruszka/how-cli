package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	apiURL       = "https://openrouter.ai/api/v1/chat/completions"
	defaultModel = "mistralai/mistral-7b-instruct:free"
	systemPrompt = `You are an expert Linux and shell command assistant. Your ONLY task is to provide a single, executable shell command that directly solves the user's request.

Rules:
1. Provide ONLY the raw shell command.
2. Do NOT provide any explanations or introductory text (e.g., do not start with "Sure, here is the command:").
3. Do NOT use any markdown formatting like ` + "```bash" + ` or ` + "``" + `.
4. If the request is ambiguous, provide the most common and safest command.
5. If a command is not applicable, provide a very short, direct answer.

Example User Request: how to clean pacman cache
Your Response: sudo pacman -Scc`
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
	model string
	rootCmd = &cobra.Command{
		Use:   "how [query...]",
		Short: "A simple AI assistant for your CLI",
		Long:  "Ask 'how' to do something and get the shell command you need.\n\nSpecial commands:\n  how setup - Configure your OpenRouter API key",
		Run:   runQuery,
		// Allow any arguments without treating them as subcommands
		Args: cobra.ArbitraryArgs,
	}
)

func init() {
	// Initialize viper for config management
	initConfig()
	
	// Add flags
	rootCmd.PersistentFlags().StringVar(&model, "model", defaultModel, "Specify an OpenRouter model to use")
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
	
	// Query OpenRouter API
	command, err := queryOpenRouter(apiKey, query, model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println(command)
}

func queryOpenRouter(apiKey, query, model string) (string, error) {
	// Prepare request
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}