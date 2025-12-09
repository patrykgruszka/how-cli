package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryLLM_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test" {
			t.Fatalf("missing or wrong auth header: %q", r.Header.Get("Authorization"))
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("unexpected content-type: %s", ct)
		}
		b, _ := io.ReadAll(r.Body)
		if err := r.Body.Close(); err != nil {
			t.Fatalf("close body: %v", err)
		}
		var body ChatRequest
		if err := json.Unmarshal(b, &body); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if len(body.Messages) != 2 || body.Messages[0].Role != "system" || body.Messages[1].Role != "user" {
			t.Fatalf("unexpected messages: %#v", body.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"echo hello"}}]}`); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer srv.Close()

	cmd, err := queryLLM(srv.URL, "test", "say hi", "mistral", true)
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "echo hello" {
		t.Fatalf("got %q", cmd)
	}
}

func TestQueryLLM_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("bad")); err != nil {
			t.Fatalf("write error body: %v", err)
		}
	}))
	defer srv.Close()

	_, err := queryLLM(srv.URL, "k", "q", "m", false)
	if err == nil {
		t.Fatal("expected error")
	}
}
