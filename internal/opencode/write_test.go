package opencode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattabdou/gantry/internal/jsonconf"
)

// fakeHome points the package at a temporary home directory for the duration of
// a test, so the write paths can be exercised without touching the real
// ~/.config/opencode.
func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	original := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDir = original })
	return home
}

func configPathIn(home string) string {
	return filepath.Join(home, ".config", ConfigDirName, ConfigFileName)
}

// writeConfig seeds the OpenCode config file with raw JSON.
func writeConfig(t *testing.T, home, contents string) string {
	t.Helper()
	path := configPathIn(home)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func countBackups(t *testing.T, home string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(home, ".config", ConfigDirName))
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "."+jsonconf.BackupSuffix) {
			count++
		}
	}
	return count
}

// TestConfigureLiteLLMBacksUpOnlyWhenItWrites is the end-to-end regression test
// for accumulating backups. The old staleness check counted models, so a user
// who added one made every launch rewrite the config and drop another
// .gantrybackup file into ~/.config/opencode.
func TestConfigureLiteLLMBacksUpOnlyWhenItWrites(t *testing.T) {
	home := fakeHome(t)
	// Seven catalog models plus one of the user's own: the exact shape that used
	// to defeat the len(models) != 6 check.
	writeConfig(t, home, `{"provider":{"gantry-litellm":{"models":{"my-own":{"name":"Mine"}}}}}`)

	first, err := ConfigureLiteLLM(testLiteLLM())
	if err != nil {
		t.Fatalf("first ConfigureLiteLLM() error = %v", err)
	}
	if !first.Updated {
		t.Fatal("first run did not update the config")
	}
	if got := countBackups(t, home); got != 1 {
		t.Errorf("after the first run: %d backups, want 1", got)
	}

	second, err := ConfigureLiteLLM(testLiteLLM())
	if err != nil {
		t.Fatalf("second ConfigureLiteLLM() error = %v", err)
	}
	if second.Updated {
		t.Errorf("second run reported an update; message = %q", second.Message)
	}
	if got := countBackups(t, home); got != 1 {
		t.Errorf("after the second run: %d backups, want still 1", got)
	}

	// A third run must also be quiet.
	if _, err := ConfigureLiteLLM(testLiteLLM()); err != nil {
		t.Fatalf("third ConfigureLiteLLM() error = %v", err)
	}
	if got := countBackups(t, home); got != 1 {
		t.Errorf("after the third run: %d backups, want still 1", got)
	}
}

func TestConfigureLiteLLMPreservesUnknownKeysOnDisk(t *testing.T) {
	home := fakeHome(t)
	path := writeConfig(t, home, `{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {"brave": {"command": ["npx", "-y", "server"], "enabled": true}},
  "mode": {"build": {"model": "gantry-litellm/claude-sonnet-4-6"}},
  "theme": "tokyonight",
  "keybinds": {"leader": "ctrl+x"},
  "model": "gantry-litellm/gpt-5.6-luna"
}`)

	if _, err := ConfigureLiteLLM(testLiteLLM()); err != nil {
		t.Fatalf("ConfigureLiteLLM() error = %v", err)
	}

	after, err := jsonconf.ReadObject(path)
	if err != nil {
		t.Fatalf("reading config back: %v", err)
	}

	if got := jsonconf.Lookup(after, "mcp", "brave", "enabled"); got != true {
		t.Errorf("mcp configuration lost: %v", got)
	}
	if got := jsonconf.Lookup(after, "mode", "build", "model"); got != "gantry-litellm/claude-sonnet-4-6" {
		t.Errorf("mode configuration lost: %v", got)
	}
	if got := after["theme"]; got != "tokyonight" {
		t.Errorf("theme lost: %v", got)
	}
	if got := jsonconf.Lookup(after, "keybinds", "leader"); got != "ctrl+x" {
		t.Errorf("keybinds lost: %v", got)
	}
	if got := after["$schema"]; got != "https://opencode.ai/config.json" {
		t.Errorf("$schema lost: %v", got)
	}
	// The user's chosen default model must not be reset.
	if got := after["model"]; got != "gantry-litellm/gpt-5.6-luna" {
		t.Errorf("model = %v, want the user's choice preserved", got)
	}
	// And gantry's own settings are in place.
	if got := jsonconf.Lookup(after, "provider", ProviderLiteLLM, "options", "baseURL"); got != "https://llm.example.com" {
		t.Errorf("baseURL = %v", got)
	}
}

// A .jsonc config must be readable. Before the comment scanner was made
// string-aware, GANTRY's own baseURL line broke the parse and the launch failed.
func TestConfigureLiteLLMHandlesJSONCWithURLs(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, ".config", ConfigDirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	jsoncPath := filepath.Join(dir, ConfigFileNameC)
	contents := `{
  // gantry manages this provider
  "theme": "tokyonight",
  "provider": {
    "gantry-litellm": {
      "options": {
        "baseURL": "https://old.example.com", // the gateway
        "apiKey": "sk-old"
      }
    }
  }
}`
	if err := os.WriteFile(jsoncPath, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ConfigureLiteLLM(testLiteLLM())
	if err != nil {
		t.Fatalf("ConfigureLiteLLM() error = %v", err)
	}
	if !result.Updated {
		t.Fatalf("expected an update; message = %q", result.Message)
	}

	after, err := jsonconf.ReadObject(jsoncPath)
	if err != nil {
		t.Fatalf("reading config back: %v", err)
	}
	if got := after["theme"]; got != "tokyonight" {
		t.Errorf("theme lost: %v", got)
	}
	if got := jsonconf.Lookup(after, "provider", ProviderLiteLLM, "options", "baseURL"); got != "https://llm.example.com" {
		t.Errorf("baseURL = %v, want it refreshed", got)
	}
}

func TestConfigureLiteLLMCreatesConfigWhenAbsent(t *testing.T) {
	home := fakeHome(t)

	result, err := ConfigureLiteLLM(testLiteLLM())
	if err != nil {
		t.Fatalf("ConfigureLiteLLM() error = %v", err)
	}
	if !result.Updated {
		t.Error("expected an update when no config exists")
	}
	if result.BackupPath != "" {
		t.Errorf("BackupPath = %q, want empty when there was no file to back up", result.BackupPath)
	}
	if got := countBackups(t, home); got != 0 {
		t.Errorf("%d backups created for a fresh config, want 0", got)
	}

	after, err := jsonconf.ReadObject(configPathIn(home))
	if err != nil {
		t.Fatalf("reading config back: %v", err)
	}
	if after["model"] != DefaultLiteLLMModel {
		t.Errorf("model = %v, want %v", after["model"], DefaultLiteLLMModel)
	}
}

func TestConfigureLiteLLMRefusesCorruptConfig(t *testing.T) {
	home := fakeHome(t)
	path := writeConfig(t, home, `{"provider": {`)

	if _, err := ConfigureLiteLLM(testLiteLLM()); err == nil {
		t.Fatal("ConfigureLiteLLM() error = nil, want an error for unparseable config")
	}

	// The unparseable file must be left exactly as it was.
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != `{"provider": {` {
		t.Errorf("corrupt config was modified: %q", after)
	}
	if got := countBackups(t, home); got != 0 {
		t.Errorf("%d backups created on the error path, want 0", got)
	}
}

func TestResetConfigTakesExactlyOneBackupAndPreservesUserKeys(t *testing.T) {
	home := fakeHome(t)
	path := writeConfig(t, home, `{
  "mcp": {"brave": {"enabled": true}},
  "keybinds": {"leader": "ctrl+x"},
  "theme": "tokyonight",
  "agent": {"reviewer": {"model": "gantry-litellm/gpt-5.6-sol"}},
  "model": "gantry-litellm/gpt-5.6-luna",
  "provider": {
    "gantry-litellm": {"models": {"stale": {"name": "Stale"}}},
    "openai": {"models": {"gpt-4": {"name": "GPT-4"}}}
  }
}`)

	result, err := ResetConfig(testLiteLLM(), nil, "litellm")
	if err != nil {
		t.Fatalf("ResetConfig() error = %v", err)
	}
	if !result.Updated {
		t.Error("ResetConfig() did not report an update")
	}
	if got := countBackups(t, home); got != 1 {
		t.Errorf("%d backups, want exactly 1", got)
	}

	after, err := jsonconf.ReadObject(path)
	if err != nil {
		t.Fatal(err)
	}

	// Gantry's settings are back to defaults.
	if after["model"] != DefaultLiteLLMModel {
		t.Errorf("model = %v, want the default restored", after["model"])
	}
	if jsonconf.Lookup(after, "provider", ProviderLiteLLM, "models", "stale") != nil {
		t.Error("a stale model inside gantry's provider survived the reset")
	}
	if jsonconf.Lookup(after, "provider", ProviderLiteLLM, "models", "claude-opus-5") == nil {
		t.Error("the default catalog was not restored")
	}

	// Everything the user owns survives - this is the point of the change.
	if jsonconf.Lookup(after, "mcp", "brave", "enabled") != true {
		t.Error("reset discarded the user's MCP servers")
	}
	if jsonconf.Lookup(after, "keybinds", "leader") != "ctrl+x" {
		t.Error("reset discarded the user's keybinds")
	}
	if after["theme"] != "tokyonight" {
		t.Error("reset discarded the user's theme")
	}
	if jsonconf.Lookup(after, "agent", "reviewer", "model") != "gantry-litellm/gpt-5.6-sol" {
		t.Error("reset discarded the user's agent configuration")
	}
	if jsonconf.Lookup(after, "provider", "openai", "models", "gpt-4", "name") != "GPT-4" {
		t.Error("reset discarded an unrelated provider")
	}
}

func TestResetConfigRejectsUnknownMode(t *testing.T) {
	fakeHome(t)
	if _, err := ResetConfig(testLiteLLM(), testBedrock(), "nonsense"); err == nil {
		t.Error("ResetConfig() error = nil, want an error for an unknown mode")
	}
}

func TestConfigureBedrockBacksUpOnlyWhenItWrites(t *testing.T) {
	home := fakeHome(t)
	writeConfig(t, home, `{"theme":"dark"}`)

	if _, err := ConfigureBedrock(testBedrock()); err != nil {
		t.Fatalf("first ConfigureBedrock() error = %v", err)
	}
	if got := countBackups(t, home); got != 1 {
		t.Errorf("after the first run: %d backups, want 1", got)
	}

	second, err := ConfigureBedrock(testBedrock())
	if err != nil {
		t.Fatalf("second ConfigureBedrock() error = %v", err)
	}
	if second.Updated {
		t.Errorf("second run reported an update; message = %q", second.Message)
	}
	if got := countBackups(t, home); got != 1 {
		t.Errorf("after the second run: %d backups, want still 1", got)
	}
}

// Rotated gateway credentials must reach the config even though everything else
// is user-owned.
func TestConfigureLiteLLMRefreshesRotatedCredentials(t *testing.T) {
	home := fakeHome(t)
	path := writeConfig(t, home, `{"provider":{"gantry-litellm":{"options":{"baseURL":"https://old.example.com","apiKey":"sk-expired"}}}}`)

	if _, err := ConfigureLiteLLM(testLiteLLM()); err != nil {
		t.Fatalf("ConfigureLiteLLM() error = %v", err)
	}

	after, err := jsonconf.ReadObject(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := jsonconf.Lookup(after, "provider", ProviderLiteLLM, "options", "apiKey"); got != "sk-current" {
		t.Errorf("apiKey = %v, want the rotated token", got)
	}
}
