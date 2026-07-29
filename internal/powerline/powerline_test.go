package powerline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/jsonconf"
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

// realisticSettings mirrors a settings.json with the keys users actually have.
// None of them belong to GANTRY, and all of them must survive a powerline edit.
func realisticSettings(t *testing.T) map[string]interface{} {
	t.Helper()
	settings, err := jsonconf.UnmarshalObject([]byte(`{
		"permissions": {
			"allow": ["Bash(git diff:*)", "Bash(npm test:*)"],
			"deny": ["Bash(rm -rf:*)"]
		},
		"env": {"FOO": "bar", "DEBUG": "1"},
		"hooks": {
			"PostToolUse": [
				{"matcher": "Edit", "hooks": [{"type": "command", "command": "gofmt -w ."}]}
			]
		},
		"model": "claude-opus-4-8",
		"mcpServers": {"context7": {"command": "npx", "args": ["-y", "@upstash/context7-mcp"]}},
		"includeCoAuthoredBy": false,
		"cleanupPeriodDays": 30
	}`))
	if err != nil {
		t.Fatalf("test fixture is not valid JSON: %v", err)
	}
	return settings
}

// TestBuildSettingsPreservesUnknownKeys is the regression test for the reported
// bug. UpdatePowerlineSettings used to unmarshal settings.json into a one-field
// struct and marshal it back, which silently deleted permissions, env, hooks,
// model and mcpServers.
func TestBuildSettingsPreservesUnknownKeys(t *testing.T) {
	current := realisticSettings(t)
	snapshot := jsonconf.Clone(current)

	out := BuildSettings(current, nil)
	if out == nil {
		t.Fatal("BuildSettings() = nil")
	}

	for key := range snapshot {
		got, present := out[key]
		if !present {
			t.Errorf("key %q was deleted from settings.json", key)
			continue
		}
		if !jsonconf.Equal(got, snapshot[key]) {
			t.Errorf("key %q was modified:\n got %#v\nwant %#v", key, got, snapshot[key])
		}
	}

	// And the statusLine we came to write is there.
	if got := jsonconf.Lookup(out, "statusLine", "command"); got != BuildPowerlineCommand(nil) {
		t.Errorf("statusLine.command = %v, want the powerline command", got)
	}
	if got := jsonconf.Lookup(out, "statusLine", "type"); got != "command" {
		t.Errorf("statusLine.type = %v, want \"command\"", got)
	}

	// The input must be untouched.
	if !jsonconf.Equal(current, snapshot) {
		t.Error("BuildSettings() mutated its input")
	}
}

func TestBuildSettings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, out map[string]interface{})
	}{
		{
			name:  "adds statusLine when absent",
			input: `{}`,
			check: func(t *testing.T, out map[string]interface{}) {
				if got := jsonconf.Lookup(out, "statusLine", "type"); got != "command" {
					t.Errorf("type = %v", got)
				}
			},
		},
		{
			name:  "corrects a stale command",
			input: `{"statusLine":{"type":"command","command":"npx -y @owloops/claude-powerline@latest --theme=nord --style=minimal"}}`,
			check: func(t *testing.T, out map[string]interface{}) {
				if got := jsonconf.Lookup(out, "statusLine", "command"); got != BuildPowerlineCommand(nil) {
					t.Errorf("command = %v, want it refreshed", got)
				}
			},
		},
		{
			name:  "preserves sibling keys inside statusLine",
			input: `{"statusLine":{"type":"command","command":"old","padding":0,"custom":"keep"}}`,
			check: func(t *testing.T, out map[string]interface{}) {
				if got := jsonconf.Lookup(out, "statusLine", "custom"); got != "keep" {
					t.Errorf("statusLine.custom = %v, want it preserved", got)
				}
				if got := jsonconf.Lookup(out, "statusLine", "padding"); got != float64(0) {
					t.Errorf("statusLine.padding = %v, want it preserved", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current, err := jsonconf.UnmarshalObject([]byte(tt.input))
			if err != nil {
				t.Fatalf("bad test input: %v", err)
			}
			out := BuildSettings(current, nil)
			if out == nil {
				t.Fatal("BuildSettings() = nil")
			}
			tt.check(t, out)
		})
	}
}

func TestBuildSettingsRefusesNonObjectStatusLine(t *testing.T) {
	current, err := jsonconf.UnmarshalObject([]byte(`{"statusLine":"a string"}`))
	if err != nil {
		t.Fatal(err)
	}
	if out := BuildSettings(current, nil); out != nil {
		t.Errorf("BuildSettings() = %#v, want nil so the caller declines to write", out)
	}
}

// Building twice must reach the same result, or UpdatePowerlineSettings would
// rewrite settings.json and take a backup on every launch.
func TestBuildSettingsIsFixedPointAcrossJSONRoundTrip(t *testing.T) {
	first := BuildSettings(realisticSettings(t), nil)
	if first == nil {
		t.Fatal("BuildSettings() = nil")
	}

	data, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	onDisk, err := jsonconf.UnmarshalObject(data)
	if err != nil {
		t.Fatal(err)
	}

	second := BuildSettings(onDisk, nil)
	if !jsonconf.Equal(onDisk, second) {
		t.Errorf("not a fixed point; gantry would rewrite settings.json every run\ndisk:   %#v\nsecond: %#v", onDisk, second)
	}
}

func TestClearStatusLine(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantRemoved bool
	}{
		{
			name:        "removes a gantry-managed statusLine",
			input:       `{"statusLine":{"type":"command","command":"npx -y @owloops/claude-powerline@latest --theme=dark"}}`,
			wantRemoved: true,
		},
		{
			name:        "leaves the user's own statusLine alone",
			input:       `{"statusLine":{"type":"command","command":"my-custom-statusline"}}`,
			wantRemoved: false,
		},
		{
			name:        "no statusLine to remove",
			input:       `{"model":"claude-opus-4-8"}`,
			wantRemoved: false,
		},
		{
			name:        "statusLine is a scalar",
			input:       `{"statusLine":"nope"}`,
			wantRemoved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current, err := jsonconf.UnmarshalObject([]byte(tt.input))
			if err != nil {
				t.Fatalf("bad test input: %v", err)
			}
			snapshot := jsonconf.Clone(current)

			out, removed := ClearStatusLine(current)
			if removed != tt.wantRemoved {
				t.Errorf("removed = %v, want %v", removed, tt.wantRemoved)
			}
			if removed {
				if _, present := out["statusLine"]; present {
					t.Error("statusLine survived removal")
				}
			} else if !jsonconf.Equal(out, snapshot) {
				t.Errorf("config changed even though nothing was removed:\n got %#v\nwant %#v", out, snapshot)
			}
			if !jsonconf.Equal(current, snapshot) {
				t.Error("ClearStatusLine() mutated its input")
			}
		})
	}
}

// Disabling powerline must clear only statusLine, not the rest of the file.
func TestClearStatusLinePreservesUnknownKeys(t *testing.T) {
	current := realisticSettings(t)
	current["statusLine"] = map[string]interface{}{
		"type":    "command",
		"command": BuildPowerlineCommand(nil),
	}
	snapshot := jsonconf.Clone(current)

	out, removed := ClearStatusLine(current)
	if !removed {
		t.Fatal("ClearStatusLine() did not remove a gantry-managed statusLine")
	}

	for key := range snapshot {
		if key == "statusLine" {
			continue
		}
		if !jsonconf.Equal(out[key], snapshot[key]) {
			t.Errorf("key %q was altered while disabling powerline", key)
		}
	}
}
