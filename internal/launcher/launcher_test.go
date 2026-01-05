package launcher

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/mattabdou/gantry/internal/config"
)

func TestIsWindows(t *testing.T) {
	expected := runtime.GOOS == "windows"
	got := IsWindows()

	if got != expected {
		t.Errorf("IsWindows() = %v, want %v", got, expected)
	}
}

func TestGetClaudeCommand(t *testing.T) {
	got := GetClaudeCommand()

	if IsWindows() {
		if got != "claude.cmd" {
			t.Errorf("GetClaudeCommand() = %v, want claude.cmd", got)
		}
	} else {
		if got != "claude" {
			t.Errorf("GetClaudeCommand() = %v, want claude", got)
		}
	}
}

func TestBuildResourceAttributes(t *testing.T) {
	username := "testuser"
	workingPath := "/path/to/project"
	gitBranch := "feature/test"

	projectConfig := &config.ProjectConfigResult{
		Config: config.ProjectConfig{
			ProjectName: "test-project",
			Repository:  "github.com/test/repo",
			Team:        "platform",
			CostCenter:  "ENG-001",
		},
	}

	result := BuildResourceAttributes(username, workingPath, projectConfig, gitBranch)

	expectedParts := []string{
		"gantry.username=testuser",
		"gantry.working_path=/path/to/project",
		"gantry.project_name=test-project",
		"gantry.repository=github.com/test/repo",
		"gantry.team=platform",
		"gantry.cost_center=ENG-001",
		"gantry.git_branch=feature/test",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(result, expected) {
			t.Errorf("BuildResourceAttributes() missing %v", expected)
		}
	}
}

func TestBuildResourceAttributesMinimal(t *testing.T) {
	username := "testuser"
	workingPath := "/path/to/project"
	gitBranch := ""

	projectConfig := &config.ProjectConfigResult{
		Config: config.ProjectConfig{
			ProjectName: "test-project",
		},
	}

	result := BuildResourceAttributes(username, workingPath, projectConfig, gitBranch)

	// Should contain required attributes
	if !strings.Contains(result, "gantry.username=testuser") {
		t.Error("Missing username")
	}
	if !strings.Contains(result, "gantry.working_path=/path/to/project") {
		t.Error("Missing working_path")
	}
	if !strings.Contains(result, "gantry.project_name=test-project") {
		t.Error("Missing project_name")
	}

	// Should NOT contain empty optional attributes
	if strings.Contains(result, "gantry.repository=") {
		t.Error("Should not contain empty repository")
	}
	if strings.Contains(result, "gantry.git_branch=") {
		t.Error("Should not contain empty git_branch")
	}
}

func TestBuildResourceAttributesNilProject(t *testing.T) {
	username := "testuser"
	workingPath := "/path/to/project"
	gitBranch := ""

	result := BuildResourceAttributes(username, workingPath, nil, gitBranch)

	// Should use "Unknown" for project name
	if !strings.Contains(result, "gantry.project_name=Unknown") {
		t.Error("Should use 'Unknown' for nil project config")
	}
}

func TestBuildEnvironment(t *testing.T) {
	trueVal := true
	falseVal := false

	globalConfig := &config.GlobalConfig{
		OTEL: config.OTELConfig{
			Endpoint:             "https://collector.example.com/otlp",
			Headers:              "Authorization=Bearer token123",
			Protocol:             "http/protobuf",
			MetricsExporter:      "otlp",
			LogsExporter:         "otlp",
			MetricExportInterval: 60000,
			LogsExportInterval:   5000,
			LogUserPrompts:       true,
			IncludeSessionID:     &trueVal,
			IncludeVersion:       &falseVal,
			IncludeAccountUUID:   &trueVal,
		},
		Bedrock: &config.BedrockConfig{
			Enabled:          true,
			AWSProfile:       "test-profile",
			AWSRegion:        "us-east-2",
			Model:            "claude-3-opus",
			MaxOutputTokens:  8192,
			MaxThinkingTokens: 1024,
		},
	}

	resourceAttributes := "gantry.username=test"

	env := BuildEnvironment(globalConfig, resourceAttributes)

	// Convert to map for easier testing
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Check OTEL vars
	if envMap["CLAUDE_CODE_ENABLE_TELEMETRY"] != "1" {
		t.Error("CLAUDE_CODE_ENABLE_TELEMETRY not set")
	}
	if envMap["OTEL_EXPORTER_OTLP_ENDPOINT"] != "https://collector.example.com/otlp" {
		t.Error("OTEL_EXPORTER_OTLP_ENDPOINT not set correctly")
	}
	if envMap["OTEL_EXPORTER_OTLP_HEADERS"] != "Authorization=Bearer token123" {
		t.Error("OTEL_EXPORTER_OTLP_HEADERS not set correctly")
	}
	if envMap["OTEL_LOG_USER_PROMPTS"] != "1" {
		t.Error("OTEL_LOG_USER_PROMPTS not set")
	}
	if envMap["OTEL_RESOURCE_ATTRIBUTES"] != resourceAttributes {
		t.Error("OTEL_RESOURCE_ATTRIBUTES not set correctly")
	}

	// Check Bedrock vars
	if envMap["CLAUDE_CODE_USE_BEDROCK"] != "1" {
		t.Error("CLAUDE_CODE_USE_BEDROCK not set")
	}
	if envMap["AWS_PROFILE"] != "test-profile" {
		t.Error("AWS_PROFILE not set correctly")
	}
	if envMap["AWS_REGION"] != "us-east-2" {
		t.Error("AWS_REGION not set correctly")
	}
	if envMap["ANTHROPIC_MODEL"] != "claude-3-opus" {
		t.Error("ANTHROPIC_MODEL not set correctly")
	}
}

func TestBuildEnvironmentBedrockDisabled(t *testing.T) {
	// Temporarily unset CLAUDE_CODE_USE_BEDROCK if it exists in system env
	originalValue, wasSet := os.LookupEnv("CLAUDE_CODE_USE_BEDROCK")
	if wasSet {
		os.Unsetenv("CLAUDE_CODE_USE_BEDROCK")
		defer os.Setenv("CLAUDE_CODE_USE_BEDROCK", originalValue)
	}

	globalConfig := &config.GlobalConfig{
		OTEL: config.OTELConfig{
			Endpoint: "https://collector.example.com/otlp",
		},
		Bedrock: &config.BedrockConfig{
			Enabled: false,
		},
	}

	env := BuildEnvironment(globalConfig, "")

	// Convert to map for easier testing
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Bedrock vars should NOT be set when disabled
	if _, ok := envMap["CLAUDE_CODE_USE_BEDROCK"]; ok {
		t.Error("CLAUDE_CODE_USE_BEDROCK should not be set when disabled")
	}
}

func TestBuildEnvironmentPreservesExisting(t *testing.T) {
	// Set an existing env var
	os.Setenv("TEST_EXISTING_VAR", "existing_value")
	defer os.Unsetenv("TEST_EXISTING_VAR")

	globalConfig := &config.GlobalConfig{
		OTEL: config.OTELConfig{
			Endpoint: "https://collector.example.com/otlp",
		},
	}

	env := BuildEnvironment(globalConfig, "")

	// Convert to map for easier testing
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Existing env vars should be preserved
	if envMap["TEST_EXISTING_VAR"] != "existing_value" {
		t.Error("Existing environment variable was not preserved")
	}
}
