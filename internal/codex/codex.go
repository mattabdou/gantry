package codex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattabdou/gantry/internal/config"
)

// ConfigureResult contains the result of configuring Codex
type ConfigureResult struct {
	Updated    bool
	ConfigPath string
	Message    string
}

// GetProfilePath returns the path to the gantry profile config file
func GetProfilePath() (string, error) {
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		codexHome = filepath.Join(home, ".codex")
	}
	return filepath.Join(codexHome, "gantry.config.toml"), nil
}

// ConfigureProvider writes the gantry profile config for Codex
func ConfigureProvider(model string, litellmConfig *config.LiteLLMConfig, otelConfig config.OTELConfig) (*ConfigureResult, error) {
	profilePath, err := GetProfilePath()
	if err != nil {
		return nil, err
	}

	desiredContent := generateConfigContent(model, litellmConfig, otelConfig)

	// Check if existing profile already matches
	if existingData, err := os.ReadFile(profilePath); err == nil {
		if string(existingData) == desiredContent {
			return &ConfigureResult{
				Updated:    false,
				ConfigPath: profilePath,
				Message:    "Codex gantry profile is already up to date",
			}, nil
		}
	}

	// Ensure the config directory exists
	configDir := filepath.Dir(profilePath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(profilePath, []byte(desiredContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write Codex profile: %w", err)
	}

	return &ConfigureResult{
		Updated:    true,
		ConfigPath: profilePath,
		Message:    "Codex gantry profile updated",
	}, nil
}

func generateConfigContent(model string, litellmConfig *config.LiteLLMConfig, otelConfig config.OTELConfig) string {
	var sb strings.Builder

	sb.WriteString("# Gantry profile for Codex - LiteLLM gateway configuration\n")
	sb.WriteString("# Launch with: codex --profile gantry\n")
	sb.WriteString(fmt.Sprintf("model = %q\n", model))
	sb.WriteString("model_provider = \"gantry-litellm\"\n")
	sb.WriteString("model_reasoning_effort = \"medium\"\n")
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
