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
		// On Windows, should return one of the valid candidates
		validCandidates := map[string]bool{"claude.cmd": true, "claude.exe": true, "claude": true}
		if !validCandidates[got] {
			t.Errorf("GetClaudeCommand() = %v, want one of claude.cmd, claude.exe, or claude", got)
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

func TestBuildEnvironmentBedrock(t *testing.T) {
	// Temporarily unset LiteLLM-related env vars if they exist
	litellmEnvVars := []string{"ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN"}
	originalValues := make(map[string]string)
	for _, v := range litellmEnvVars {
		if val, ok := os.LookupEnv(v); ok {
			originalValues[v] = val
			os.Unsetenv(v)
		}
	}
	defer func() {
		for k, v := range originalValues {
			os.Setenv(k, v)
		}
	}()

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
			AWSProfile:        "test-profile",
			AWSRegion:         "us-east-2",
			Model:             "claude-3-opus",
			MaxOutputTokens:   8192,
			MaxThinkingTokens: 1024,
		},
	}

	resourceAttributes := "gantry.username=test"

	env := BuildEnvironment(globalConfig, resourceAttributes, "bedrock")

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

	// LiteLLM vars should NOT be set in bedrock mode
	if _, ok := envMap["ANTHROPIC_BASE_URL"]; ok {
		t.Error("ANTHROPIC_BASE_URL should not be set in bedrock mode")
	}
	if _, ok := envMap["ANTHROPIC_AUTH_TOKEN"]; ok {
		t.Error("ANTHROPIC_AUTH_TOKEN should not be set in bedrock mode")
	}
}

func TestBuildEnvironmentLiteLLM(t *testing.T) {
	// Temporarily unset Bedrock-related env vars if they exist
	bedrockEnvVars := []string{"CLAUDE_CODE_USE_BEDROCK", "AWS_PROFILE", "AWS_REGION"}
	originalValues := make(map[string]string)
	for _, v := range bedrockEnvVars {
		if val, ok := os.LookupEnv(v); ok {
			originalValues[v] = val
			os.Unsetenv(v)
		}
	}
	defer func() {
		for k, v := range originalValues {
			os.Setenv(k, v)
		}
	}()

	globalConfig := &config.GlobalConfig{
		OTEL: config.OTELConfig{
			Endpoint: "https://collector.example.com/otlp",
		},
		LiteLLM: &config.LiteLLMConfig{
			BaseURL:           "https://litellm.example.com",
			AuthToken:         "test-token-123",
			Model:             "claude-sonnet",
			MaxOutputTokens:   4096,
			MaxThinkingTokens: 512,
		},
	}

	resourceAttributes := "gantry.username=test"

	env := BuildEnvironment(globalConfig, resourceAttributes, "litellm")

	// Convert to map for easier testing
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Check LiteLLM vars
	if envMap["ANTHROPIC_BASE_URL"] != "https://litellm.example.com" {
		t.Error("ANTHROPIC_BASE_URL not set correctly")
	}
	if envMap["ANTHROPIC_AUTH_TOKEN"] != "test-token-123" {
		t.Error("ANTHROPIC_AUTH_TOKEN not set correctly")
	}
	if envMap["ANTHROPIC_MODEL"] != "claude-sonnet" {
		t.Error("ANTHROPIC_MODEL not set correctly")
	}
	if envMap["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] != "4096" {
		t.Error("CLAUDE_CODE_MAX_OUTPUT_TOKENS not set correctly")
	}
	if envMap["MAX_THINKING_TOKENS"] != "512" {
		t.Error("MAX_THINKING_TOKENS not set correctly")
	}

	// Bedrock vars should NOT be set in litellm mode
	if _, ok := envMap["CLAUDE_CODE_USE_BEDROCK"]; ok {
		t.Error("CLAUDE_CODE_USE_BEDROCK should not be set in litellm mode")
	}
	if _, ok := envMap["AWS_PROFILE"]; ok {
		t.Error("AWS_PROFILE should not be set in litellm mode")
	}
	if _, ok := envMap["AWS_REGION"]; ok {
		t.Error("AWS_REGION should not be set in litellm mode")
	}
}

func TestBuildEnvironmentLiteLLMModeSetsNoBedrockVars(t *testing.T) {
	// Temporarily unset Bedrock-related env vars if they exist
	bedrockEnvVars := []string{"CLAUDE_CODE_USE_BEDROCK", "AWS_PROFILE", "AWS_REGION"}
	originalValues := make(map[string]string)
	for _, v := range bedrockEnvVars {
		if val, ok := os.LookupEnv(v); ok {
			originalValues[v] = val
			os.Unsetenv(v)
		}
	}
	defer func() {
		for k, v := range originalValues {
			os.Setenv(k, v)
		}
	}()

	globalConfig := &config.GlobalConfig{
		OTEL: config.OTELConfig{
			Endpoint: "https://collector.example.com/otlp",
		},
		Bedrock: &config.BedrockConfig{
			AWSProfile: "test-profile",
			AWSRegion:  "us-east-2",
		},
		LiteLLM: &config.LiteLLMConfig{
			BaseURL:   "https://litellm.example.com",
			AuthToken: "test-token",
		},
	}

	// Even with Bedrock config present, litellm mode should not set Bedrock vars
	env := BuildEnvironment(globalConfig, "", "litellm")

	// Convert to map for easier testing
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Bedrock vars should NOT be set in litellm mode
	if _, ok := envMap["CLAUDE_CODE_USE_BEDROCK"]; ok {
		t.Error("CLAUDE_CODE_USE_BEDROCK should not be set in litellm mode")
	}
	if _, ok := envMap["AWS_PROFILE"]; ok {
		t.Error("AWS_PROFILE should not be set in litellm mode")
	}

	// LiteLLM vars should be set
	if envMap["ANTHROPIC_BASE_URL"] != "https://litellm.example.com" {
		t.Error("ANTHROPIC_BASE_URL should be set in litellm mode")
	}
}

func TestBuildEnvironmentLiteLLMFiltersInheritedBedrockVars(t *testing.T) {
	// Simulate system-wide Bedrock environment variables
	os.Setenv("CLAUDE_CODE_USE_BEDROCK", "1")
	os.Setenv("AWS_PROFILE", "system-profile")
	os.Setenv("AWS_REGION", "us-west-2")
	defer func() {
		os.Unsetenv("CLAUDE_CODE_USE_BEDROCK")
		os.Unsetenv("AWS_PROFILE")
		os.Unsetenv("AWS_REGION")
	}()

	globalConfig := &config.GlobalConfig{
		OTEL: config.OTELConfig{
			Endpoint: "https://collector.example.com/otlp",
		},
		LiteLLM: &config.LiteLLMConfig{
			BaseURL:   "https://litellm.example.com",
			AuthToken: "test-token",
		},
	}

	// In LiteLLM mode, inherited Bedrock vars should be filtered out
	env := BuildEnvironment(globalConfig, "", "litellm")

	// Convert to map for easier testing
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Inherited Bedrock vars should be filtered out in litellm mode
	if _, ok := envMap["CLAUDE_CODE_USE_BEDROCK"]; ok {
		t.Error("CLAUDE_CODE_USE_BEDROCK should be filtered out in litellm mode")
	}
	if _, ok := envMap["AWS_PROFILE"]; ok {
		t.Error("AWS_PROFILE should be filtered out in litellm mode")
	}
	if _, ok := envMap["AWS_REGION"]; ok {
		t.Error("AWS_REGION should be filtered out in litellm mode")
	}

	// LiteLLM vars should still be set
	if envMap["ANTHROPIC_BASE_URL"] != "https://litellm.example.com" {
		t.Error("ANTHROPIC_BASE_URL should be set in litellm mode")
	}
}

func TestBuildEnvironmentDisableExperimentalBetas(t *testing.T) {
	// Unset the env var if it's already set in the current environment
	if val, ok := os.LookupEnv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"); ok {
		os.Unsetenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS")
		defer os.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", val)
	}

	tests := []struct {
		name      string
		cctConfig *config.ClaudeCodeTerminalConfig
		wantSet   bool
		wantValue string
	}{
		{
			name: "disableExperimentalBetas=1 sets env var",
			cctConfig: func() *config.ClaudeCodeTerminalConfig {
				val := 1
				return &config.ClaudeCodeTerminalConfig{DisableExperimentalBetas: &val}
			}(),
			wantSet:   true,
			wantValue: "1",
		},
		{
			name: "disableExperimentalBetas=0 does not set env var",
			cctConfig: func() *config.ClaudeCodeTerminalConfig {
				val := 0
				return &config.ClaudeCodeTerminalConfig{DisableExperimentalBetas: &val}
			}(),
			wantSet: false,
		},
		{
			name:      "nil claudeCodeTerminal does not set env var",
			cctConfig: nil,
			wantSet:   false,
		},
		{
			name:      "nil disableExperimentalBetas does not set env var",
			cctConfig: &config.ClaudeCodeTerminalConfig{DisableExperimentalBetas: nil},
			wantSet:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalConfig := &config.GlobalConfig{
				OTEL: config.OTELConfig{
					Endpoint: "https://collector.example.com/otlp",
				},
				Bedrock: &config.BedrockConfig{
					AWSProfile: "test-profile",
					AWSRegion:  "us-east-2",
				},
				ClaudeCodeTerminal: tt.cctConfig,
			}

			env := BuildEnvironment(globalConfig, "", "bedrock")

			envMap := make(map[string]string)
			for _, e := range env {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			val, ok := envMap["CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"]
			if tt.wantSet {
				if !ok {
					t.Error("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS should be set")
				} else if val != tt.wantValue {
					t.Errorf("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS = %q, want %q", val, tt.wantValue)
				}
			} else {
				if ok {
					t.Errorf("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS should not be set, got %q", val)
				}
			}
		})
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
		Bedrock: &config.BedrockConfig{
			AWSProfile: "test-profile",
			AWSRegion:  "us-east-2",
		},
	}

	env := BuildEnvironment(globalConfig, "", "bedrock")

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
