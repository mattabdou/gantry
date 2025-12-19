package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/mattabdou/gantry/internal/config"
)

// UpdateResult contains the result of updating Claude Code
type UpdateResult struct {
	Success bool
	Message string
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// GetClaudeCommand returns the appropriate Claude command for the current OS
func GetClaudeCommand() string {
	if IsWindows() {
		return "claude.cmd"
	}
	return "claude"
}

// GetNpmCommand returns the appropriate npm command for the current OS
func GetNpmCommand() string {
	if IsWindows() {
		return "npm.cmd"
	}
	return "npm"
}

// UpdateClaudeCode updates Claude Code to the latest version
func UpdateClaudeCode() *UpdateResult {
	fmt.Println("Checking for Claude Code updates...")

	npmCmd := GetNpmCommand()
	cmd := exec.Command(npmCmd, "update", "-g", "@anthropic-ai/claude-code")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return &UpdateResult{
			Success: false,
			Message: fmt.Sprintf("Update check failed: %v", err),
		}
	}

	return &UpdateResult{
		Success: true,
		Message: "Claude Code is up to date",
	}
}

// GetGitBranch returns the current git branch name or empty string if not in a git repo
func GetGitBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Stderr = nil

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// BuildResourceAttributes builds the OTEL_RESOURCE_ATTRIBUTES string from all sources
func BuildResourceAttributes(username, workingPath string, projectConfig *config.ProjectConfigResult, gitBranch string) string {
	var attributes []string

	// Always include username and working path
	attributes = append(attributes, fmt.Sprintf("gantry.username=%s", username))
	attributes = append(attributes, fmt.Sprintf("gantry.working_path=%s", workingPath))

	// Project config attributes
	projectName := "Unknown"
	if projectConfig != nil && projectConfig.Config.ProjectName != "" {
		projectName = projectConfig.Config.ProjectName
	}
	attributes = append(attributes, fmt.Sprintf("gantry.project_name=%s", projectName))

	if projectConfig != nil {
		if projectConfig.Config.Repository != "" {
			attributes = append(attributes, fmt.Sprintf("gantry.repository=%s", projectConfig.Config.Repository))
		}
		if projectConfig.Config.Team != "" {
			attributes = append(attributes, fmt.Sprintf("gantry.team=%s", projectConfig.Config.Team))
		}
		if projectConfig.Config.CostCenter != "" {
			attributes = append(attributes, fmt.Sprintf("gantry.cost_center=%s", projectConfig.Config.CostCenter))
		}
	}

	// Git branch if available
	if gitBranch != "" {
		attributes = append(attributes, fmt.Sprintf("gantry.git_branch=%s", gitBranch))
	}

	return strings.Join(attributes, ",")
}

// BuildEnvironment builds environment variables for Claude Code from global config
func BuildEnvironment(globalConfig *config.GlobalConfig, resourceAttributes string) []string {
	env := os.Environ()

	// Helper to add environment variable
	addEnv := func(key, value string) {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	otel := globalConfig.OTEL

	// Enable telemetry
	addEnv("CLAUDE_CODE_ENABLE_TELEMETRY", "1")

	// Exporter configuration
	metricsExporter := otel.MetricsExporter
	if metricsExporter == "" {
		metricsExporter = "otlp"
	}
	addEnv("OTEL_METRICS_EXPORTER", metricsExporter)

	logsExporter := otel.LogsExporter
	if logsExporter == "" {
		logsExporter = "otlp"
	}
	addEnv("OTEL_LOGS_EXPORTER", logsExporter)

	// Protocol and endpoint
	protocol := otel.Protocol
	if protocol == "" {
		protocol = "http/protobuf"
	}
	addEnv("OTEL_EXPORTER_OTLP_PROTOCOL", protocol)
	addEnv("OTEL_EXPORTER_OTLP_ENDPOINT", otel.Endpoint)

	// Authentication headers
	if otel.Headers != "" {
		addEnv("OTEL_EXPORTER_OTLP_HEADERS", otel.Headers)
	}

	// Export intervals
	if otel.MetricExportInterval > 0 {
		addEnv("OTEL_METRIC_EXPORT_INTERVAL", strconv.Itoa(otel.MetricExportInterval))
	}
	if otel.LogsExportInterval > 0 {
		addEnv("OTEL_LOGS_EXPORT_INTERVAL", strconv.Itoa(otel.LogsExportInterval))
	}

	// Optional flags
	if otel.LogUserPrompts {
		addEnv("OTEL_LOG_USER_PROMPTS", "1")
	}

	if otel.IncludeSessionID != nil {
		addEnv("OTEL_METRICS_INCLUDE_SESSION_ID", strconv.FormatBool(*otel.IncludeSessionID))
	}
	if otel.IncludeVersion != nil {
		addEnv("OTEL_METRICS_INCLUDE_VERSION", strconv.FormatBool(*otel.IncludeVersion))
	}
	if otel.IncludeAccountUUID != nil {
		addEnv("OTEL_METRICS_INCLUDE_ACCOUNT_UUID", strconv.FormatBool(*otel.IncludeAccountUUID))
	}

	// Resource attributes (our custom attributes)
	addEnv("OTEL_RESOURCE_ATTRIBUTES", resourceAttributes)

	// AWS Bedrock configuration
	if globalConfig.Bedrock != nil && globalConfig.Bedrock.Enabled {
		bedrock := globalConfig.Bedrock

		addEnv("CLAUDE_CODE_USE_BEDROCK", "1")

		if bedrock.AWSProfile != "" {
			addEnv("AWS_PROFILE", bedrock.AWSProfile)
		}
		if bedrock.AWSRegion != "" {
			addEnv("AWS_REGION", bedrock.AWSRegion)
		}
		if bedrock.Model != "" {
			addEnv("ANTHROPIC_MODEL", bedrock.Model)
		}
		if bedrock.MaxOutputTokens > 0 {
			addEnv("CLAUDE_CODE_MAX_OUTPUT_TOKENS", strconv.Itoa(bedrock.MaxOutputTokens))
		}
		if bedrock.MaxThinkingTokens > 0 {
			addEnv("MAX_THINKING_TOKENS", strconv.Itoa(bedrock.MaxThinkingTokens))
		}
	}

	return env
}

// LaunchClaude spawns Claude Code with the configured environment
func LaunchClaude(args []string, env []string) error {
	claudeCmd := GetClaudeCommand()

	cmd := exec.Command(claudeCmd, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}
