package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetGlobalConfigPath(t *testing.T) {
	path, err := GetGlobalConfigPath()
	if err != nil {
		t.Fatalf("GetGlobalConfigPath() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, GlobalConfigFilename)

	if path != expected {
		t.Errorf("GetGlobalConfigPath() = %v, want %v", path, expected)
	}
}

func TestValidateGlobalConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *GlobalConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Username: "john.doe",
				},
				OTEL: OTELConfig{
					Endpoint: "https://collector.example.com/otlp",
				},
			},
			wantErr: false,
		},
		{
			name: "missing gantry section",
			config: &GlobalConfig{
				OTEL: OTELConfig{
					Endpoint: "https://collector.example.com/otlp",
				},
			},
			wantErr: true,
		},
		{
			name: "empty username",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Username: "",
				},
				OTEL: OTELConfig{
					Endpoint: "https://collector.example.com/otlp",
				},
			},
			wantErr: true,
		},
		{
			name: "placeholder username YOUR_",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Username: "YOUR_USERNAME_HERE",
				},
				OTEL: OTELConfig{
					Endpoint: "https://collector.example.com/otlp",
				},
			},
			wantErr: true,
		},
		{
			name: "missing endpoint",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Username: "john.doe",
				},
				OTEL: OTELConfig{},
			},
			wantErr: true,
		},
		{
			name: "placeholder endpoint YOUR_",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Username: "john.doe",
				},
				OTEL: OTELConfig{
					Endpoint: "https://YOUR_ENDPOINT.example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "placeholder endpoint your-",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Username: "john.doe",
				},
				OTEL: OTELConfig{
					Endpoint: "https://your-endpoint.example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "valid defaultTool cc",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Username:    "john.doe",
					DefaultTool: "cc",
				},
				OTEL: OTELConfig{
					Endpoint: "https://collector.example.com/otlp",
				},
			},
			wantErr: false,
		},
		{
			name: "valid defaultTool oc",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Username:    "john.doe",
					DefaultTool: "oc",
				},
				OTEL: OTELConfig{
					Endpoint: "https://collector.example.com/otlp",
				},
			},
			wantErr: false,
		},
		{
			name: "valid defaultTool ocd",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Username:    "john.doe",
					DefaultTool: "ocd",
				},
				OTEL: OTELConfig{
					Endpoint: "https://collector.example.com/otlp",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid defaultTool",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Username:    "john.doe",
					DefaultTool: "invalid",
				},
				OTEL: OTELConfig{
					Endpoint: "https://collector.example.com/otlp",
				},
			},
			wantErr: true,
		},
		{
			name: "valid disableExperimentalBetas 1",
			config: func() *GlobalConfig {
				val := 1
				return &GlobalConfig{
					Gantry: &GantryConfig{Username: "john.doe"},
					OTEL:   OTELConfig{Endpoint: "https://collector.example.com/otlp"},
					ClaudeCodeTerminal: &ClaudeCodeTerminalConfig{
						DisableExperimentalBetas: &val,
					},
				}
			}(),
			wantErr: false,
		},
		{
			name: "valid disableExperimentalBetas 0",
			config: func() *GlobalConfig {
				val := 0
				return &GlobalConfig{
					Gantry: &GantryConfig{Username: "john.doe"},
					OTEL:   OTELConfig{Endpoint: "https://collector.example.com/otlp"},
					ClaudeCodeTerminal: &ClaudeCodeTerminalConfig{
						DisableExperimentalBetas: &val,
					},
				}
			}(),
			wantErr: false,
		},
		{
			name: "invalid disableExperimentalBetas",
			config: func() *GlobalConfig {
				val := 2
				return &GlobalConfig{
					Gantry: &GantryConfig{Username: "john.doe"},
					OTEL:   OTELConfig{Endpoint: "https://collector.example.com/otlp"},
					ClaudeCodeTerminal: &ClaudeCodeTerminalConfig{
						DisableExperimentalBetas: &val,
					},
				}
			}(),
			wantErr: true,
		},
		{
			name: "nil claudeCodeTerminal is valid",
			config: &GlobalConfig{
				Gantry: &GantryConfig{Username: "john.doe"},
				OTEL:   OTELConfig{Endpoint: "https://collector.example.com/otlp"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGlobalConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGlobalConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDefaultGlobalConfigTemplate(t *testing.T) {
	template := GetDefaultGlobalConfigTemplate()

	if template == nil {
		t.Fatal("GetDefaultGlobalConfigTemplate() returned nil")
	}

	if template.Gantry == nil {
		t.Error("Gantry should not be nil")
	}

	if template.Gantry.Username == "" {
		t.Error("Gantry.Username should not be empty")
	}

	if template.Gantry.IgnorePowerline == nil {
		t.Error("Gantry.IgnorePowerline should not be nil")
	}

	if *template.Gantry.IgnorePowerline != true {
		t.Error("Gantry.IgnorePowerline should default to true")
	}

	if template.Gantry.EnablePowerline == nil {
		t.Error("Gantry.EnablePowerline should not be nil")
	}

	if template.Gantry.BypassLoadingScreen == nil {
		t.Error("Gantry.BypassLoadingScreen should not be nil")
	}

	if *template.Gantry.BypassLoadingScreen != false {
		t.Error("Gantry.BypassLoadingScreen should default to false")
	}

	if template.OTEL.Endpoint == "" {
		t.Error("OTEL.Endpoint should not be empty")
	}

	if template.Bedrock == nil {
		t.Error("Bedrock should not be nil")
	}

	if template.LiteLLM == nil {
		t.Error("LiteLLM should not be nil")
	}

	if template.Powerline == nil {
		t.Error("Powerline should not be nil")
	}
}

func TestGetDefaultProjectConfig(t *testing.T) {
	// Test without .git folder - should use directory name
	t.Run("no git folder uses directory name", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-no-git")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		result := GetDefaultProjectConfig(tmpDir)

		if result == nil {
			t.Fatal("GetDefaultProjectConfig() returned nil")
		}

		expectedName := filepath.Base(tmpDir)
		if result.Config.ProjectName != expectedName {
			t.Errorf("ProjectName = %v, want %v", result.Config.ProjectName, expectedName)
		}

		if result.Path != "" {
			t.Errorf("Path should be empty, got %v", result.Path)
		}
	})

	// Test with .git folder - should use git root directory name
	t.Run("with git folder uses git root name", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-with-git")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create .git folder at root
		gitDir := filepath.Join(tmpDir, ".git")
		if err := os.Mkdir(gitDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create a subdirectory to test from
		subDir := filepath.Join(tmpDir, "src", "components")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		result := GetDefaultProjectConfig(subDir)

		if result == nil {
			t.Fatal("GetDefaultProjectConfig() returned nil")
		}

		expectedName := filepath.Base(tmpDir)
		if result.Config.ProjectName != expectedName {
			t.Errorf("ProjectName = %v, want %v", result.Config.ProjectName, expectedName)
		}
	})
}

func TestFindProjectConfig(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "gantry-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directory
	nestedDir := filepath.Join(tmpDir, "subdir1", "subdir2")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .gantry.json in tmpDir
	projectConfig := ProjectConfig{
		ProjectName: "test-project",
		Team:        "test-team",
	}
	configData, _ := json.Marshal(projectConfig)
	configPath := filepath.Join(tmpDir, ProjectConfigFilename)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatal(err)
	}

	// Test finding config from nested directory
	result := FindProjectConfig(nestedDir)
	if result == nil {
		t.Fatal("FindProjectConfig() returned nil, expected to find config")
	}

	if result.Config.ProjectName != "test-project" {
		t.Errorf("ProjectName = %v, want %v", result.Config.ProjectName, "test-project")
	}

	if result.Config.Team != "test-team" {
		t.Errorf("Team = %v, want %v", result.Config.Team, "test-team")
	}

	if result.Path != configPath {
		t.Errorf("Path = %v, want %v", result.Path, configPath)
	}
}

func TestGetConfigValue(t *testing.T) {
	trueVal := true
	config := &GlobalConfig{
		OTEL: OTELConfig{
			Endpoint:         "https://test.example.com",
			IncludeSessionID: &trueVal,
		},
	}

	tests := []struct {
		key     string
		want    interface{}
		wantErr bool
	}{
		{"otel.endpoint", "https://test.example.com", false},
		{"nonexistent", nil, true},
		{"otel.nonexistent", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := GetConfigValue(config, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("GetConfigValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetConfigValue(t *testing.T) {
	config := &GlobalConfig{
		OTEL: OTELConfig{
			Endpoint: "https://old.example.com",
		},
	}

	// Test setting string value
	err := SetConfigValue(config, "otel.endpoint", "https://new.example.com")
	if err != nil {
		t.Fatalf("SetConfigValue() error = %v", err)
	}

	if config.OTEL.Endpoint != "https://new.example.com" {
		t.Errorf("OTEL.Endpoint = %v, want %v", config.OTEL.Endpoint, "https://new.example.com")
	}

	// Test setting boolean value
	err = SetConfigValue(config, "otel.logUserPrompts", "true")
	if err != nil {
		t.Fatalf("SetConfigValue() error = %v", err)
	}

	if config.OTEL.LogUserPrompts != true {
		t.Errorf("LogUserPrompts = %v, want %v", config.OTEL.LogUserPrompts, true)
	}

	// Test setting number value
	err = SetConfigValue(config, "otel.metricExportInterval", "30000")
	if err != nil {
		t.Fatalf("SetConfigValue() error = %v", err)
	}

	if config.OTEL.MetricExportInterval != 30000 {
		t.Errorf("MetricExportInterval = %v, want %v", config.OTEL.MetricExportInterval, 30000)
	}
}

func TestIsValidTool(t *testing.T) {
	tests := []struct {
		tool  string
		valid bool
	}{
		{"cc", true},
		{"oc", true},
		{"ocd", true},
		{"cl", true},
		{"clk", true},
		{"CC", false},   // case sensitive
		{"claude", false},
		{"opencode", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			result := IsValidTool(tt.tool)
			if result != tt.valid {
				t.Errorf("IsValidTool(%q) = %v, want %v", tt.tool, result, tt.valid)
			}
		})
	}
}

func TestValidToolValues(t *testing.T) {
	// Ensure the ValidToolValues slice has the expected values
	expected := []string{"cc", "oc", "ocd", "cl", "clk"}

	if len(ValidToolValues) != len(expected) {
		t.Errorf("ValidToolValues has %d elements, expected %d", len(ValidToolValues), len(expected))
	}

	for _, v := range expected {
		found := false
		for _, vt := range ValidToolValues {
			if v == vt {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidToolValues missing expected value %q", v)
		}
	}
}

func TestIsValidReleaseChannel(t *testing.T) {
	tests := []struct {
		channel string
		valid   bool
	}{
		{"stable", true},
		{"beta", true},
		{"STABLE", false}, // case sensitive
		{"BETA", false},
		{"alpha", false},
		{"", false},
		{"release", false},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			result := IsValidReleaseChannel(tt.channel)
			if result != tt.valid {
				t.Errorf("IsValidReleaseChannel(%q) = %v, want %v", tt.channel, result, tt.valid)
			}
		})
	}
}

func TestGetReleaseChannel(t *testing.T) {
	tests := []struct {
		name    string
		config  *GlobalConfig
		want    string
	}{
		{
			name:   "nil config",
			config: nil,
			want:   "stable",
		},
		{
			name:   "nil gantry section",
			config: &GlobalConfig{},
			want:   "stable",
		},
		{
			name: "empty release",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Release: "",
				},
			},
			want: "stable",
		},
		{
			name: "stable release",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Release: "stable",
				},
			},
			want: "stable",
		},
		{
			name: "beta release",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Release: "beta",
				},
			},
			want: "beta",
		},
		{
			name: "invalid release defaults to stable",
			config: &GlobalConfig{
				Gantry: &GantryConfig{
					Release: "invalid",
				},
			},
			want: "stable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetReleaseChannel(tt.config)
			if got != tt.want {
				t.Errorf("GetReleaseChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidReleaseChannels(t *testing.T) {
	// Ensure the ValidReleaseChannels slice has the expected values
	expected := []string{"stable", "beta"}

	if len(ValidReleaseChannels) != len(expected) {
		t.Errorf("ValidReleaseChannels has %d elements, expected %d", len(ValidReleaseChannels), len(expected))
	}

	for _, v := range expected {
		found := false
		for _, vc := range ValidReleaseChannels {
			if v == vc {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidReleaseChannels missing expected value %q", v)
		}
	}
}
