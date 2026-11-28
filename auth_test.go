package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/komapotter/gh-aipr/internal/auth"
)

func newTestAuthCLI(t *testing.T, store auth.Store) (*authCLI, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return &authCLI{
		store:  store,
		getenv: func(string) (string, bool) { return "", false },
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: stderr,
	}, stdout, stderr
}

func TestAuthRegisterAndStatusHideSecret(t *testing.T) {
	store := auth.NewMemoryStore()
	cli, stdout, _ := newTestAuthCLI(t, store)
	secret := "sk-test-secret-do-not-print"
	cli.readSecret = func(string) (string, error) { return secret, nil }

	if err := cli.run([]string{"register", "-p", "openai"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), secret) {
		t.Fatalf("register output leaked secret:\n%s", stdout.String())
	}
	got, err := store.Get(auth.ProviderOpenAI)
	if err != nil || got != secret {
		t.Fatalf("store Get = %q, %v", got, err)
	}

	stdout.Reset()
	if err := cli.run([]string{"status"}); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if strings.Contains(out, secret) {
		t.Fatalf("status leaked secret:\n%s", out)
	}
	if !strings.Contains(out, "openai") || !strings.Contains(out, "registered") {
		t.Fatalf("status missing openai registration:\n%s", out)
	}
	if strings.Contains(strings.ToLower(out), "anthropic") {
		t.Fatalf("status should not mention anthropic:\n%s", out)
	}
}

func TestAuthRegisterDefaultsToOpenAI(t *testing.T) {
	store := auth.NewMemoryStore()
	cli, stdout, _ := newTestAuthCLI(t, store)
	cli.readSecret = func(provider string) (string, error) {
		if provider != auth.ProviderOpenAI {
			t.Fatalf("provider = %s", provider)
		}
		return "sk-openai-default", nil
	}
	if err := cli.run([]string{"register"}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(auth.ProviderOpenAI)
	if err != nil || got != "sk-openai-default" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if !strings.Contains(stdout.String(), "openai") {
		t.Fatalf("output:\n%s", stdout.String())
	}
}

func TestAuthRemove(t *testing.T) {
	store := auth.NewMemoryStore()
	if err := store.Set(auth.ProviderOpenAI, "sk-openai"); err != nil {
		t.Fatal(err)
	}
	cli, stdout, _ := newTestAuthCLI(t, store)
	if err := cli.run([]string{"remove", "--provider", "openai"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(auth.ProviderOpenAI); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("expected key removed, got %v", err)
	}
	if !strings.Contains(stdout.String(), "Removed openai API key") {
		t.Fatalf("remove output:\n%s", stdout.String())
	}
}

func TestAuthRemoveDefaultsToOpenAI(t *testing.T) {
	store := auth.NewMemoryStore()
	if err := store.Set(auth.ProviderOpenAI, "sk-openai"); err != nil {
		t.Fatal(err)
	}
	cli, _, _ := newTestAuthCLI(t, store)
	if err := cli.run([]string{"remove"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(auth.ProviderOpenAI); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("expected key removed, got %v", err)
	}
}

func TestAuthRegisterRejectsAnthropic(t *testing.T) {
	cli, _, _ := newTestAuthCLI(t, auth.NewMemoryStore())
	err := cli.run([]string{"register", "-p", "anthropic"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "anthropic is no longer supported") {
		t.Fatalf("got %v", err)
	}
}

func TestAuthRegisterRejectsUnknownProvider(t *testing.T) {
	cli, _, _ := newTestAuthCLI(t, auth.NewMemoryStore())
	err := cli.run([]string{"register", "gemini"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("got %v", err)
	}
}

func TestAuthSwitchRemoved(t *testing.T) {
	cli, _, stderr := newTestAuthCLI(t, auth.NewMemoryStore())
	err := cli.run([]string{"switch", "openai"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr.String(), "unknown auth command") {
		t.Fatalf("stderr:\n%s", stderr.String())
	}
}

func TestAuthRemoveMissingKey(t *testing.T) {
	cli, _, _ := newTestAuthCLI(t, auth.NewMemoryStore())
	err := cli.run([]string{"remove", "-p", "openai"})
	if err == nil || !strings.Contains(err.Error(), "no API key registered") {
		t.Fatalf("got %v", err)
	}
}

func TestAuthUnknownCommand(t *testing.T) {
	cli, _, stderr := newTestAuthCLI(t, auth.NewMemoryStore())
	err := cli.run([]string{"login"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr.String(), "unknown auth command") {
		t.Fatalf("stderr:\n%s", stderr.String())
	}
}

func TestAuthHelp(t *testing.T) {
	cli, stdout, _ := newTestAuthCLI(t, auth.NewMemoryStore())
	if err := cli.run([]string{"-h"}); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	for _, cmd := range []string{"register", "remove", "status"} {
		if !strings.Contains(out, cmd) {
			t.Fatalf("auth help missing %s:\n%s", cmd, out)
		}
	}
	if !strings.Contains(out, "gh aipr auth") {
		t.Fatalf("auth help should show gh aipr auth:\n%s", out)
	}
	if strings.Contains(out, "login") || strings.Contains(out, "logout") {
		t.Fatalf("auth help should not mention login/logout:\n%s", out)
	}
	if strings.Contains(out, "switch") {
		t.Fatalf("auth help should not mention switch:\n%s", out)
	}
	if strings.Contains(strings.ToLower(out), "anthropic") {
		t.Fatalf("auth help should not mention anthropic:\n%s", out)
	}
}

func TestIsAuthCommand(t *testing.T) {
	if !isAuthCommand([]string{"auth", "status"}) {
		t.Fatal("expected auth command")
	}
	if isAuthCommand([]string{"--version"}) || isAuthCommand([]string{"-v"}) || isAuthCommand(nil) {
		t.Fatal("flags only is not auth")
	}
}

func TestLoadAppConfigEnvOverridesKeyring(t *testing.T) {
	orig := credentialStore
	t.Cleanup(func() { credentialStore = orig })

	store := auth.NewMemoryStore()
	if err := store.Set(auth.ProviderOpenAI, "keyring-openai"); err != nil {
		t.Fatal(err)
	}
	credentialStore = store

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "env-openai")
	t.Setenv("ANTHROPIC_API_KEY", "env-anthropic")
	t.Setenv("AI_PROVIDER", "anthropic")

	cfg, err := loadAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIKey != "env-openai" {
		t.Fatalf("OpenAIKey = %q", cfg.OpenAIKey)
	}
}

func TestLoadAppConfigUsesKeyringWhenEnvUnset(t *testing.T) {
	orig := credentialStore
	t.Cleanup(func() { credentialStore = orig })

	store := auth.NewMemoryStore()
	if err := store.Set(auth.ProviderOpenAI, "keyring-openai"); err != nil {
		t.Fatal(err)
	}
	credentialStore = store

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "env-anthropic")
	t.Setenv("AI_PROVIDER", "anthropic")

	cfg, err := loadAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIKey != "keyring-openai" {
		t.Fatalf("OpenAIKey = %q", cfg.OpenAIKey)
	}
}

func TestLoadAppConfigMissingKeySuggestsRegister(t *testing.T) {
	orig := credentialStore
	t.Cleanup(func() { credentialStore = orig })
	credentialStore = auth.NewMemoryStore()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "env-anthropic")
	t.Setenv("AI_PROVIDER", "anthropic")

	_, err := loadAppConfig()
	if err == nil {
		t.Fatal("expected missing key error")
	}
	if !strings.Contains(err.Error(), "gh aipr auth register") {
		t.Fatalf("got %v", err)
	}
}

func TestAuthStatusShowsEnvironmentWithoutSecret(t *testing.T) {
	store := auth.NewMemoryStore()
	cli, stdout, _ := newTestAuthCLI(t, store)
	secret := "sk-env-must-not-appear"
	cli.getenv = func(key string) (string, bool) {
		if key == "OPENAI_API_KEY" {
			return secret, true
		}
		return "", false
	}
	if err := cli.run([]string{"status"}); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if strings.Contains(out, secret) {
		t.Fatalf("status leaked secret:\n%s", out)
	}
	if !strings.Contains(out, "available (environment)") {
		t.Fatalf("status:\n%s", out)
	}
	if !strings.Contains(out, "openai") {
		t.Fatalf("status:\n%s", out)
	}
}

func TestLoadAppConfigIgnoresLegacyAnthropicConfig(t *testing.T) {
	orig := credentialStore
	t.Cleanup(func() { credentialStore = orig })

	store := auth.NewMemoryStore()
	if err := store.Set(auth.ProviderOpenAI, "keyring-openai"); err != nil {
		t.Fatal(err)
	}
	credentialStore = store

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "gh-aipr", "config.yml")
	if err := auth.SaveFileConfig(path, auth.FileConfig{ModelProvider: "anthropic"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "env-anthropic")
	t.Setenv("AI_PROVIDER", "anthropic")

	cfg, err := loadAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIKey != "keyring-openai" {
		t.Fatalf("OpenAIKey = %q", cfg.OpenAIKey)
	}
}
