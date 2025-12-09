package main

import (
	"errors"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"
)

type fakeSys struct {
	env   map[string]string
	look  map[string]bool
	usr   *user.User
	goos  string
	arch  string
	files map[string]bool
}

func (f fakeSys) Env(k string) string { return f.env[k] }
func (f fakeSys) LookPath(cmd string) (string, error) {
	if f.look[cmd] {
		return "/bin/" + cmd, nil
	}
	return "", errors.New("not found")
}
func (f fakeSys) CurrentUser() (*user.User, error) {
	if f.usr != nil {
		return f.usr, nil
	}
	return &user.User{Username: "me", Uid: "1000"}, nil
}
func (f fakeSys) GOOS() string   { return f.goos }
func (f fakeSys) GOARCH() string { return f.arch }
func (f fakeSys) Stat(name string) (os.FileInfo, error) {
	if f.files[name] {
		return fakeFileInfo{name: name}, nil
	}
	return nil, errors.New("no")
}

type fakeFileInfo struct{ name string }

func (f fakeFileInfo) Name() string           { return f.name }
func (f fakeFileInfo) Size() int64            { return 0 }
func (f fakeFileInfo) Mode() os.FileMode      { return 0 }
func (f fakeFileInfo) ModTime() (t time.Time) { return }
func (f fakeFileInfo) IsDir() bool            { return false }
func (f fakeFileInfo) Sys() any               { return nil }

func TestGetSystemInfo_UnixWithBrewAndSudo(t *testing.T) {
	fs := fakeSys{
		env:  map[string]string{"SHELL": "/bin/bash"},
		look: map[string]bool{"brew": true, "sudo": true},
		usr:  &user.User{Username: "me", Uid: "1000"},
		goos: "linux",
		arch: "amd64",
	}
	info := getSystemInfoWith(fs)
	if !containsAll(info, []string{"- OS: Linux", "- Architecture: amd64", "- Shell: bash", "- Package Managers: Homebrew", "- User Privileges: sudo available"}) {
		t.Fatalf("unexpected info: %s", info)
	}
}

func TestDetectUserPrivileges_WindowsAdmin(t *testing.T) {
	fs := fakeSys{
		goos:  "windows",
		files: map[string]bool{"C:\\Windows\\System32\\config\\SAM": true},
	}
	priv := detectUserPrivilegesWith(fs)
	if priv == "" || !strings.Contains(priv, "administrator") {
		t.Fatalf("expected admin, got %q", priv)
	}
}

func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
