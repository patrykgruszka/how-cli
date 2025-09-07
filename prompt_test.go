package main

import (
	"os"
	"strings"
	"testing"
)

func TestBuildSystemPrompt_Golden(t *testing.T) {
	info := "- OS: Linux\n- Architecture: amd64\n- Shell: bash"
	got := buildSystemPromptFrom(info)
	want, err := os.ReadFile("testdata/prompt.golden")
	if err != nil {
		t.Fatal(err)
	}
	// Normalize line endings and trailing newlines for cross-platform consistency
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		return strings.TrimRight(s, "\n")
	}
	if normalize(string(want)) != normalize(got) {
		t.Fatalf("prompt mismatch\nWANT:\n%s\nGOT:\n%s", string(want), got)
	}
}
