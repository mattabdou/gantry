package powerline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/jsonconf"
)

// ~/.claude/settings.json belongs to Claude Code and to the user. It holds
// permissions, env, hooks, model, mcpServers and more, none of which GANTRY
// knows about. Everything here therefore works on a generic map: an earlier
// implementation unmarshalled the file into a one-field struct and marshalled it
// back, which silently deleted every other key in it.
//
// statusLine is the one key GANTRY manages, so it is set rather than merged -
// but the fields are set inside any existing statusLine object so that sibling
// keys survive.

// statusLineKey is the settings.json key GANTRY manages.
const statusLineKey = "statusLine"

// powerlineMarker identifies a statusLine command as one GANTRY installed.
const powerlineMarker = "claude-powerline"

// userHomeDir is a seam for tests, which need a writable home directory to
// exercise the backup-and-write behaviour.
var userHomeDir = os.UserHomeDir

// CheckResult contains the result of checking for claude-powerline configuration
type CheckResult struct {
	Installed bool
	// StatusLine is the statusLine block found in settings.json, if any.
	// Reading into a narrow struct is safe; only writing one back is
	// destructive, which is why the write paths use maps.
	StatusLine   *config.StatusLineConfig
	SettingsPath string
}

// UpdateResult contains the result of updating powerline settings
type UpdateResult struct {
	Updated bool
	Message string
}

// GetClaudeSettingsPath returns the path to Claude Code's settings.json
func GetClaudeSettingsPath() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// CheckClaudePowerline checks if claude-powerline is configured in Claude Code settings
func CheckClaudePowerline() *CheckResult {
	settingsPath, err := GetClaudeSettingsPath()
	if err != nil {
		return &CheckResult{Installed: false, SettingsPath: ""}
	}

	result := &CheckResult{
		Installed:    false,
		SettingsPath: settingsPath,
	}

	settings, err := jsonconf.ReadObject(settingsPath)
	if err != nil {
		return result
	}

	statusType, _ := jsonconf.Lookup(settings, statusLineKey, "type").(string)
	command, _ := jsonconf.Lookup(settings, statusLineKey, "command").(string)
	if statusType == "" && command == "" {
		return result
	}

	result.StatusLine = &config.StatusLineConfig{Type: statusType, Command: command}
	if statusType == "command" && strings.Contains(command, powerlineMarker) {
		result.Installed = true
	}

	return result
}

// BuildPowerlineCommand builds the claude-powerline command with theme and style options
func BuildPowerlineCommand(powerlineConfig *config.PowerlineConfig) string {
	theme, style := themeAndStyle(powerlineConfig)
	return fmt.Sprintf("npx -y @owloops/claude-powerline@latest --theme=%s --style=%s", theme, style)
}

func themeAndStyle(powerlineConfig *config.PowerlineConfig) (theme, style string) {
	theme, style = "dark", "powerline"
	if powerlineConfig != nil {
		if powerlineConfig.Theme != "" {
			theme = powerlineConfig.Theme
		}
		if powerlineConfig.Style != "" {
			style = powerlineConfig.Style
		}
	}
	return theme, style
}

// BuildSettings returns the Claude Code settings that should be on disk with
// powerline enabled. cur is not mutated.
//
// Every key other than statusLine is carried through untouched, including keys
// GANTRY knows nothing about. Within statusLine, only type and command are set,
// so any sibling key the user or a future Claude Code version added survives.
//
// It returns nil if statusLine exists but is not a JSON object, so the caller
// can decline to overwrite it.
func BuildSettings(cur map[string]interface{}, powerlineConfig *config.PowerlineConfig) map[string]interface{} {
	out := jsonconf.Clone(cur)

	statusLine := jsonconf.Object(out, statusLineKey)
	if statusLine == nil {
		return nil
	}
	statusLine["type"] = "command"
	statusLine["command"] = BuildPowerlineCommand(powerlineConfig)

	return out
}

// ClearStatusLine returns cur without its statusLine key, but only when that
// statusLine is a claude-powerline command GANTRY manages. A statusLine the user
// set up themselves is left in place. cur is not mutated.
func ClearStatusLine(cur map[string]interface{}) (out map[string]interface{}, removed bool) {
	out = jsonconf.Clone(cur)

	if _, present := out[statusLineKey]; !present {
		return out, false
	}
	command, _ := jsonconf.Lookup(out, statusLineKey, "command").(string)
	if !strings.Contains(command, powerlineMarker) {
		return out, false
	}

	delete(out, statusLineKey)
	return out, true
}

// UpdatePowerlineSettings updates Claude Code settings.json with powerline configuration
func UpdatePowerlineSettings(powerlineConfig *config.PowerlineConfig) *UpdateResult {
	settingsPath, err := GetClaudeSettingsPath()
	if err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to get settings path: %v", err)}
	}

	current, err := jsonconf.ReadObject(settingsPath)
	if err != nil {
		// Refuse to write over a file we could not parse. Overwriting here would
		// discard the user's permissions, hooks and env - the previous behaviour.
		return &UpdateResult{
			Updated: false,
			Message: fmt.Sprintf("Could not parse %s (%v); leaving it unchanged. Fix the file to enable powerline.", settingsPath, err),
		}
	}

	desired := BuildSettings(current, powerlineConfig)
	if desired == nil {
		return &UpdateResult{
			Updated: false,
			Message: fmt.Sprintf("%q in %s is not a JSON object; leaving it unchanged", statusLineKey, settingsPath),
		}
	}

	if jsonconf.Equal(desired, current) {
		return &UpdateResult{Updated: false, Message: "Powerline settings already up to date"}
	}

	if _, err := jsonconf.Backup(settingsPath); err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to back up settings: %v", err)}
	}
	if err := jsonconf.WriteObject(settingsPath, desired); err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to write settings: %v", err)}
	}

	theme, style := themeAndStyle(powerlineConfig)
	return &UpdateResult{
		Updated: true,
		Message: fmt.Sprintf("Updated powerline: theme=%s, style=%s", theme, style),
	}
}

// RemovePowerlineSettings removes the statusLine configuration from Claude Code settings
func RemovePowerlineSettings() *UpdateResult {
	settingsPath, err := GetClaudeSettingsPath()
	if err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to get settings path: %v", err)}
	}

	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return &UpdateResult{Updated: false, Message: "No settings file to update"}
	}

	current, err := jsonconf.ReadObject(settingsPath)
	if err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to parse settings: %v", err)}
	}

	if _, present := current[statusLineKey]; !present {
		return &UpdateResult{Updated: false, Message: "Powerline not configured"}
	}

	desired, removed := ClearStatusLine(current)
	if !removed {
		return &UpdateResult{Updated: false, Message: "Powerline not managed by GANTRY"}
	}

	if _, err := jsonconf.Backup(settingsPath); err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to back up settings: %v", err)}
	}
	if err := jsonconf.WriteObject(settingsPath, desired); err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to write settings: %v", err)}
	}

	return &UpdateResult{
		Updated: true,
		Message: "Removed powerline configuration (disabled)",
	}
}

// ResetClaudeSettings backs up and removes ~/.claude/settings.json so Claude Code recreates it with defaults.
//
// Unlike GANTRY's OpenCode reset, this deletes the whole file: these are Claude
// Code's own settings and it regenerates them, and the command sits behind an
// explicit confirmation prompt. To clear only what GANTRY manages, set
// enablePowerline to false instead, which removes just the statusLine key.
func ResetClaudeSettings() (*UpdateResult, error) {
	settingsPath, err := GetClaudeSettingsPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings path: %w", err)
	}

	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return &UpdateResult{
			Updated: false,
			Message: "No Claude Code settings file found, nothing to reset",
		}, nil
	}

	backupPath, err := jsonconf.Backup(settingsPath)
	if err != nil {
		return nil, err
	}

	if err := os.Remove(settingsPath); err != nil {
		return nil, fmt.Errorf("failed to remove settings file: %w", err)
	}

	return &UpdateResult{
		Updated: true,
		Message: fmt.Sprintf("Claude Code settings reset to defaults (backup: %s)", filepath.Base(backupPath)),
	}, nil
}
