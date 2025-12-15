package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func resetForTest(t *testing.T) string {
	t.Helper()

	// Isolate filesystem writes (config + history) per test
	tmpDir := t.TempDir()
	t.Setenv("HOW_CONFIG_DIR", tmpDir)

	// Reset global flags/state that can leak between tests
	modelFlag = ""
	debug = false
	runFlag = false
	yesFlag = false

	// Reset viper to avoid cross-test contamination, then re-init config
	viper.Reset()
	initConfig()

	return tmpDir
}

func TestRoot_NoArgs_ShowsHelp(t *testing.T) {
	_ = resetForTest(t)

	bOut, bErr := &bytes.Buffer{}, &bytes.Buffer{}
	rootCmd.SetOut(bOut)
	rootCmd.SetErr(bErr)
	rootCmd.SetArgs([]string{})

	_, _ = rootCmd.ExecuteC()

	all := bOut.String() + bErr.String()
	if !strings.Contains(all, "how [query...]") {
		t.Fatalf("expected help text, got: %s", all)
	}
}

func TestRoot_MissingAPIKey_Errors(t *testing.T) {
	_ = resetForTest(t)

	viper.Set("api_key", "")

	rootCmd.SetArgs([]string{"echo", "hi"})
	_, err := rootCmd.ExecuteC()
	if err == nil || !strings.Contains(err.Error(), "API key not found") {
		t.Fatalf("expected missing key error, got: %v", err)
	}
}

func TestSetModel_WritesConfigToConfigDir(t *testing.T) {
	cfgDir := resetForTest(t)

	// Ensure API key presence doesn't matter for set-model
	viper.Set("api_key", "")

	rootCmd.SetArgs([]string{"set-model", "new-model"})
	_, err := rootCmd.ExecuteC()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// With HOW_CONFIG_DIR, config should be written here:
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("expected config file at %s, read error: %v", cfgPath, err)
	}
	if !strings.Contains(string(b), "model: new-model") {
		t.Fatalf("expected model in config, got:\n%s", string(b))
	}
}

func TestLast_NoHistory_Errors(t *testing.T) {
	_ = resetForTest(t)

	rootCmd.SetArgs([]string{"last"})
	_, err := rootCmd.ExecuteC()
	if err == nil || !strings.Contains(err.Error(), "no history yet") {
		t.Fatalf("expected no history error, got: %v", err)
	}
}

func TestRoot_Query_PrintsCommand_AndAppendsHistory(t *testing.T) {
	cfgDir := resetForTest(t)

	// Stub out the LLM call so we don't hit network.
	orig := llmQuery
	t.Cleanup(func() { llmQuery = orig })
	llmQuery = func(endpoint, apiKey, query, model string, refererNeeded bool) (string, error) {
		return "echo hi", nil
	}

	viper.Set("api_key", "dummy-test-key")

	bOut, bErr := &bytes.Buffer{}, &bytes.Buffer{}
	rootCmd.SetOut(bOut)
	rootCmd.SetErr(bErr)

	rootCmd.SetArgs([]string{"say", "hi"})
	_, err := rootCmd.ExecuteC()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if strings.TrimSpace(bOut.String()) != "echo hi" {
		t.Fatalf("expected printed command %q, got %q", "echo hi", bOut.String())
	}

	histPath := filepath.Join(cfgDir, "history.jsonl")
	hb, err := os.ReadFile(histPath)
	if err != nil {
		t.Fatalf("expected history file at %s, read error: %v\nstderr:%s", histPath, err, bErr.String())
	}
	if !strings.Contains(string(hb), `"command":"echo hi"`) {
		t.Fatalf("expected history to contain command, got:\n%s", string(hb))
	}
}
