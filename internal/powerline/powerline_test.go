package powerline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattabdou/gantry/internal/config"
)

func TestGetClaudeSettingsPath(t *testing.T) {
	path, err := GetClaudeSettingsPath()
	if err != nil {
		t.Fatalf("GetClaudeSettingsPath() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".claude", "settings.json")

	if path != expected {
		t.Errorf("GetClaudeSettingsPath() = %v, want %v", path, expected)
	}
}

func TestBuildPowerlineCommand(t *testing.T) {
	tests := []struct {
		name   string
		config *config.PowerlineConfig
		want   string
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
			want:   "npx -y @owloops/claude-powerline@latest --theme=dark --style=powerline",
		},
		{
			name:   "empty config uses defaults",
			config: &config.PowerlineConfig{},
			want:   "npx -y @owloops/claude-powerline@latest --theme=dark --style=powerline",
		},
		{
			name: "custom theme",
			config: &config.PowerlineConfig{
				Theme: "nord",
			},
			want: "npx -y @owloops/claude-powerline@latest --theme=nord --style=powerline",
		},
		{
			name: "custom style",
			config: &config.PowerlineConfig{
				Style: "minimal",
			},
			want: "npx -y @owloops/claude-powerline@latest --theme=dark --style=minimal",
		},
		{
			name: "custom theme and style",
			config: &config.PowerlineConfig{
				Theme: "tokyo-night",
				Style: "capsule",
			},
			want: "npx -y @owloops/claude-powerline@latest --theme=tokyo-night --style=capsule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPowerlineCommand(tt.config)
			if got != tt.want {
				t.Errorf("BuildPowerlineCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckClaudePowerline(t *testing.T) {
	result := CheckClaudePowerline()

	// Just verify it returns a valid result without panic
	if result == nil {
		t.Fatal("CheckClaudePowerline() returned nil")
	}

	// SettingsPath should be set even if file doesn't exist
	if result.SettingsPath == "" {
		t.Error("SettingsPath should not be empty")
	}
}
