package powerline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattabdou/gantry/internal/jsonconf"
)

// fakeHome points the package at a temporary home directory for the duration of
// a test, so the write paths can be exercised without touching the real
// ~/.claude/settings.json.
func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	original := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDir = original })
	return home
}

func settingsPathIn(home string) string {
	return filepath.Join(home, ".claude", "settings.json")
}

func writeSettings(t *testing.T, home, contents string) string {
	t.Helper()
	path := settingsPathIn(home)
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
	entries, err := os.ReadDir(filepath.Join(home, ".claude"))
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

const richSettings = `{
  "permissions": {"allow": ["Bash(git diff:*)"], "deny": ["Bash(rm -rf:*)"]},
  "env": {"DEBUG": "1"},
  "hooks": {"PostToolUse": [{"matcher": "Edit", "hooks": [{"type": "command", "command": "gofmt -w ."}]}]},
  "model": "claude-opus-4-8",
  "mcpServers": {"context7": {"command": "npx"}},
  "cleanupPeriodDays": 30
}`

// TestUpdatePowerlineSettingsPreservesUnknownKeysOnDisk is the end-to-end
// regression test for the reported bug: enabling powerline used to wipe
// permissions, env, hooks, model and mcpServers out of settings.json.
func TestUpdatePowerlineSettingsPreservesUnknownKeysOnDisk(t *testing.T) {
	home := fakeHome(t)
	path := writeSettings(t, home, richSettings)

	result := UpdatePowerlineSettings(nil)
	if !result.Updated {
		t.Fatalf("UpdatePowerlineSettings() did not update: %s", result.Message)
	}

	after, err := jsonconf.ReadObject(path)
	if err != nil {
		t.Fatalf("reading settings back: %v", err)
	}

	if jsonconf.Lookup(after, "permissions", "allow") == nil {
		t.Error("permissions were deleted")
	}
	if jsonconf.Lookup(after, "env", "DEBUG") != "1" {
		t.Error("env was deleted")
	}
	if jsonconf.Lookup(after, "hooks", "PostToolUse") == nil {
		t.Error("hooks were deleted")
	}
	if after["model"] != "claude-opus-4-8" {
		t.Errorf("model was deleted or changed: %v", after["model"])
	}
	if jsonconf.Lookup(after, "mcpServers", "context7", "command") != "npx" {
		t.Error("mcpServers were deleted")
	}
	if after["cleanupPeriodDays"] != float64(30) {
		t.Errorf("cleanupPeriodDays was deleted or changed: %v", after["cleanupPeriodDays"])
	}

	// And the statusLine we came to write is there.
	if got := jsonconf.Lookup(after, "statusLine", "command"); got != BuildPowerlineCommand(nil) {
		t.Errorf("statusLine.command = %v", got)
	}
}

func TestUpdatePowerlineSettingsBacksUpOnlyWhenItWrites(t *testing.T) {
	home := fakeHome(t)
	writeSettings(t, home, richSettings)

	if result := UpdatePowerlineSettings(nil); !result.Updated {
		t.Fatalf("first run did not update: %s", result.Message)
	}
	if got := countBackups(t, home); got != 1 {
		t.Errorf("after the first run: %d backups, want 1", got)
	}

	second := UpdatePowerlineSettings(nil)
	if second.Updated {
		t.Errorf("second run reported an update: %s", second.Message)
	}
	if got := countBackups(t, home); got != 1 {
		t.Errorf("after the second run: %d backups, want still 1", got)
	}
}

// The previous implementation warned and then overwrote an unparseable
// settings.json with an empty object, destroying whatever was in it.
func TestUpdatePowerlineSettingsRefusesCorruptFile(t *testing.T) {
	home := fakeHome(t)
	corrupt := `{"permissions": {"allow": [`
	path := writeSettings(t, home, corrupt)

	result := UpdatePowerlineSettings(nil)
	if result.Updated {
		t.Error("UpdatePowerlineSettings() reported an update for an unparseable file")
	}
	if !strings.Contains(result.Message, path) {
		t.Errorf("message should name the offending file; got %q", result.Message)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != corrupt {
		t.Errorf("the unparseable file was modified:\n got %q\nwant %q", after, corrupt)
	}
	if got := countBackups(t, home); got != 0 {
		t.Errorf("%d backups created on the refuse path, want 0", got)
	}
}

func TestUpdatePowerlineSettingsCreatesFileWhenAbsent(t *testing.T) {
	home := fakeHome(t)

	if result := UpdatePowerlineSettings(nil); !result.Updated {
		t.Fatalf("UpdatePowerlineSettings() did not update: %s", result.Message)
	}

	after, err := jsonconf.ReadObject(settingsPathIn(home))
	if err != nil {
		t.Fatalf("reading settings back: %v", err)
	}
	if got := jsonconf.Lookup(after, "statusLine", "type"); got != "command" {
		t.Errorf("statusLine.type = %v", got)
	}
	if got := countBackups(t, home); got != 0 {
		t.Errorf("%d backups created for a fresh file, want 0", got)
	}
}

func TestRemovePowerlineSettingsPreservesUnknownKeysOnDisk(t *testing.T) {
	home := fakeHome(t)
	path := writeSettings(t, home, richSettings)

	// Enable, then disable.
	if result := UpdatePowerlineSettings(nil); !result.Updated {
		t.Fatalf("enabling powerline failed: %s", result.Message)
	}
	if result := RemovePowerlineSettings(); !result.Updated {
		t.Fatalf("RemovePowerlineSettings() did not update: %s", result.Message)
	}

	after, err := jsonconf.ReadObject(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, present := after["statusLine"]; present {
		t.Error("statusLine survived removal")
	}
	for _, key := range []string{"permissions", "env", "hooks", "model", "mcpServers", "cleanupPeriodDays"} {
		if _, present := after[key]; !present {
			t.Errorf("key %q was deleted while disabling powerline", key)
		}
	}
}

func TestRemovePowerlineSettingsLeavesForeignStatusLine(t *testing.T) {
	home := fakeHome(t)
	path := writeSettings(t, home, `{"statusLine":{"type":"command","command":"my-own-statusline"},"model":"claude-opus-4-8"}`)

	result := RemovePowerlineSettings()
	if result.Updated {
		t.Errorf("RemovePowerlineSettings() removed a statusLine GANTRY does not manage: %s", result.Message)
	}

	after, err := jsonconf.ReadObject(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := jsonconf.Lookup(after, "statusLine", "command"); got != "my-own-statusline" {
		t.Errorf("the user's statusLine was altered: %v", got)
	}
}

func TestRemovePowerlineSettingsNoFile(t *testing.T) {
	fakeHome(t)
	result := RemovePowerlineSettings()
	if result.Updated {
		t.Error("RemovePowerlineSettings() reported an update with no settings file")
	}
}

func TestRemovePowerlineSettingsRefusesCorruptFile(t *testing.T) {
	home := fakeHome(t)
	corrupt := `{"permissions": [`
	path := writeSettings(t, home, corrupt)

	if result := RemovePowerlineSettings(); result.Updated {
		t.Error("RemovePowerlineSettings() reported an update for an unparseable file")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != corrupt {
		t.Errorf("the unparseable file was modified: %q", after)
	}
}

func TestCheckClaudePowerlineDetectsInstalled(t *testing.T) {
	home := fakeHome(t)
	writeSettings(t, home, `{"statusLine":{"type":"command","command":"npx -y @owloops/claude-powerline@latest --theme=dark --style=powerline"}}`)

	result := CheckClaudePowerline()
	if !result.Installed {
		t.Error("CheckClaudePowerline() did not detect an installed powerline")
	}
	if result.StatusLine == nil {
		t.Fatal("StatusLine is nil")
	}
	if result.StatusLine.Type != "command" {
		t.Errorf("StatusLine.Type = %q", result.StatusLine.Type)
	}
}

func TestCheckClaudePowerlineForeignStatusLine(t *testing.T) {
	home := fakeHome(t)
	writeSettings(t, home, `{"statusLine":{"type":"command","command":"my-own-statusline"}}`)

	result := CheckClaudePowerline()
	if result.Installed {
		t.Error("CheckClaudePowerline() reported a foreign statusLine as installed")
	}
	if result.StatusLine == nil || result.StatusLine.Command != "my-own-statusline" {
		t.Errorf("StatusLine = %+v, want the foreign command reported", result.StatusLine)
	}
}

func TestResetClaudeSettingsBacksUpAndRemoves(t *testing.T) {
	home := fakeHome(t)
	path := writeSettings(t, home, richSettings)

	result, err := ResetClaudeSettings()
	if err != nil {
		t.Fatalf("ResetClaudeSettings() error = %v", err)
	}
	if !result.Updated {
		t.Error("ResetClaudeSettings() did not report an update")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("settings.json still exists after the reset")
	}
	if got := countBackups(t, home); got != 1 {
		t.Errorf("%d backups, want exactly 1", got)
	}
}

func TestResetClaudeSettingsNoFile(t *testing.T) {
	fakeHome(t)
	result, err := ResetClaudeSettings()
	if err != nil {
		t.Fatalf("ResetClaudeSettings() error = %v", err)
	}
	if result.Updated {
		t.Error("ResetClaudeSettings() reported an update with no settings file")
	}
}
