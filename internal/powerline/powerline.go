package powerline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattabdou/gantry/internal/config"
)

// CheckResult contains the result of checking for claude-powerline configuration
type CheckResult struct {
	Installed    bool
	Settings     *config.ClaudeSettings
	SettingsPath string
}

// UpdateResult contains the result of updating powerline settings
type UpdateResult struct {
	Updated bool
	Message string
}

// GetClaudeSettingsPath returns the path to Claude Code's settings.json
func GetClaudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
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

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return result
	}

	var settings config.ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return result
	}

	result.Settings = &settings

	// Check if statusLine is configured with claude-powerline
	if settings.StatusLine != nil &&
		settings.StatusLine.Type == "command" &&
		settings.StatusLine.Command != "" &&
		strings.Contains(settings.StatusLine.Command, "claude-powerline") {
		result.Installed = true
	}

	return result
}

// BuildPowerlineCommand builds the claude-powerline command with theme and style options
func BuildPowerlineCommand(powerlineConfig *config.PowerlineConfig) string {
	theme := "dark"
	style := "powerline"

	if powerlineConfig != nil {
		if powerlineConfig.Theme != "" {
			theme = powerlineConfig.Theme
		}
		if powerlineConfig.Style != "" {
			style = powerlineConfig.Style
		}
	}

	return fmt.Sprintf("npx -y @owloops/claude-powerline@latest --theme=%s --style=%s", theme, style)
}

// UpdatePowerlineSettings updates Claude Code settings.json with powerline configuration
func UpdatePowerlineSettings(powerlineConfig *config.PowerlineConfig) *UpdateResult {
	settingsPath, err := GetClaudeSettingsPath()
	if err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to get settings path: %v", err)}
	}

	settingsDir := filepath.Dir(settingsPath)

	// Ensure .claude directory exists
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to create directory: %v", err)}
	}

	// Load existing settings or create new
	var settings config.ClaudeSettings

	data, err := os.ReadFile(settingsPath)
	if err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			// If settings file is corrupted, start fresh but warn user
			fmt.Fprintf(os.Stderr, "Warning: Could not parse %s, creating new settings\n", settingsPath)
			settings = config.ClaudeSettings{}
		}
	}

	// Build the new command
	newCommand := BuildPowerlineCommand(powerlineConfig)

	// Check if update is needed
	if settings.StatusLine != nil &&
		settings.StatusLine.Type == "command" &&
		settings.StatusLine.Command == newCommand {
		return &UpdateResult{Updated: false, Message: "Powerline settings already up to date"}
	}

	// Update settings
	settings.StatusLine = &config.StatusLineConfig{
		Type:    "command",
		Command: newCommand,
	}

	// Write back to file
	output, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to marshal settings: %v", err)}
	}

	if err := os.WriteFile(settingsPath, output, 0644); err != nil {
		return &UpdateResult{Updated: false, Message: fmt.Sprintf("Failed to write settings: %v", err)}
	}

	theme := "dark"
	style := "powerline"
	if powerlineConfig != nil {
		if powerlineConfig.Theme != "" {
			theme = powerlineConfig.Theme
		}
		if powerlineConfig.Style != "" {
			style = powerlineConfig.Style
		}
	}

	return &UpdateResult{
		Updated: true,
		Message: fmt.Sprintf("Updated powerline: theme=%s, style=%s", theme, style),
	}
}
