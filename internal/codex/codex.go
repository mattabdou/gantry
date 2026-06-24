package codex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattabdou/gantry/internal/config"
)

// ConfigureResult contains the result of configuring Codex
type ConfigureResult struct {
	Updated    bool
	BackupPath string
	ConfigPath string
	Message    string
}

// GetConfigPath returns the path to the Codex config file
func GetConfigPath() (string, error) {
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		codexHome = filepath.Join(home, ".codex")
	}
	return filepath.Join(codexHome, "config.toml"), nil
}

// BackupConfig creates a timestamped backup of the Codex config
func BackupConfig(configPath string) (string, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config for backup: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupPath := fmt.Sprintf("%s.%s.gantrybackup", configPath, timestamp)

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// ConfigureProvider writes the Codex config.toml with LiteLLM gateway and OTel settings
func ConfigureProvider(model string, litellmConfig *config.LiteLLMConfig, otelConfig config.OTELConfig) (*ConfigureResult, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Generate desired config content
	desiredContent := generateConfigContent(model, litellmConfig, otelConfig)

	// Check if existing config already matches
	if existingData, err := os.ReadFile(configPath); err == nil {
		if string(existingData) == desiredContent {
			return &ConfigureResult{
				Updated:    false,
				ConfigPath: configPath,
				Message:    "Codex configuration is already up to date",
			}, nil
		}
	}

	// Create backup before modifying
	backupPath, err := BackupConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Ensure the config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, []byte(desiredContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write Codex config: %w", err)
	}

	result := &ConfigureResult{
		Updated:    true,
		BackupPath: backupPath,
		ConfigPath: configPath,
		Message:    "Codex configuration updated",
	}

	if backupPath != "" {
		result.Message = fmt.Sprintf("Codex configuration updated (backup: %s)", filepath.Base(backupPath))
	}

	return result, nil
}

func generateConfigContent(model string, litellmConfig *config.LiteLLMConfig, otelConfig config.OTELConfig) string {
	var sb strings.Builder

	sb.WriteString("# Managed by gantry - LiteLLM gateway configuration\n")
	sb.WriteString(fmt.Sprintf("model = %q\n", model))
	sb.WriteString("model_provider = \"gantry-litellm\"\n")
	sb.WriteString("\n")
	sb.WriteString("[model_providers.gantry-litellm]\n")
	sb.WriteString("name = \"Gantry LiteLLM Gateway\"\n")
	sb.WriteString(fmt.Sprintf("base_url = %q\n", litellmConfig.BaseURL))
	sb.WriteString("env_key = \"GANTRY_LITELLM_API_KEY\"\n")

	// OTel configuration
	if otelConfig.Endpoint != "" {
		sb.WriteString("\n")
		sb.WriteString("[otel]\n")

		headers := buildOtelHeaders(otelConfig.Headers)
		if headers != "" {
			sb.WriteString(fmt.Sprintf("exporter = { otlp-http = { endpoint = %q, protocol = \"binary\", headers = { %s } } }\n", otelConfig.Endpoint, headers))
		} else {
			sb.WriteString(fmt.Sprintf("exporter = { otlp-http = { endpoint = %q, protocol = \"binary\" } }\n", otelConfig.Endpoint))
		}
	}

	return sb.String()
}

func buildOtelHeaders(headersStr string) string {
	if headersStr == "" {
		return ""
	}

	var pairs []string
	for _, part := range strings.Split(headersStr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		eqIdx := strings.Index(part, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:eqIdx])
		value := strings.TrimSpace(part[eqIdx+1:])
		pairs = append(pairs, fmt.Sprintf("%q = %q", key, value))
	}

	return strings.Join(pairs, ", ")
}
