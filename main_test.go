package main

import (
	"bytes"
	"strings"
	"testing"
	"github.com/spf13/viper"
	"net/http/httptest"
	"net/http"
	"io"
	"os"
)

// TestMain sets a default api_key to prevent tests from failing due to
// missing configuration, unless a test explicitly overrides it.
func TestMain(m *testing.M) {
	// Provide a dummy key so CLI paths do not exit early in tests.
	viper.Set("api_key", "dummy-test-key")
	code := m.Run()
	// cleanup
	viper.Set("api_key", "")
	os.Exit(code)
}

func TestRoot_NoArgs_ShowsHelp(t *testing.T) {
	bOut, bErr := &bytes.Buffer{}, &bytes.Buffer{}
	rootCmd.SetOut(bOut)
	rootCmd.SetErr(bErr)
	rootCmd.SetArgs([]string{})
	_, _ = rootCmd.ExecuteC()
	all := bOut.String() + bErr.String()
	if !strings.Contains(all, "AI assistant") && !strings.Contains(all, "how [query...]") {
		t.Fatalf("expected help text, got: %s", all)
	}
}

func TestRoot_MissingAPIKey_Errors(t *testing.T) {
	bOut, bErr := &bytes.Buffer{}, &bytes.Buffer{}
	rootCmd.SetOut(bOut)
	rootCmd.SetErr(bErr)
	rootCmd.SetArgs([]string{"echo", "hi"})
	// Ensure no api_key in viper in tests
	viper.Set("api_key", "")
	defer viper.Set("api_key", "")
	// Call runQuery directly and assert error instead of process exit
	err := runQuery(rootCmd, []string{"echo", "hi"})
	if err == nil || !strings.Contains(err.Error(), "API key not found") {
		t.Fatalf("expected missing key error, got: %v", err)
	}
}

func TestRoot_WithMockedAPI_PrintsCommand(t *testing.T) {
	// Mock server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer testkey" {
			t.Fatalf("bad auth: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"echo ok"}}]}`); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer srv.Close()

	// Point CLI to mocked endpoint
	oldURL := apiURL
	apiURL = srv.URL
	defer func(){ apiURL = oldURL }()

	viper.Set("api_key", "testkey")
	defer viper.Set("api_key", "")

	bOut, bErr := &bytes.Buffer{}, &bytes.Buffer{}
	rootCmd.SetOut(bOut)
	rootCmd.SetErr(bErr)
	rootCmd.SetArgs([]string{"do", "it"})
	_, err := rootCmd.ExecuteC()
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if strings.TrimSpace(bOut.String()) != "echo ok" {
		t.Fatalf("unexpected stdout: %q stderr=%q", bOut.String(), bErr.String())
	}
}
