package codex

import (
	"bytes"
	"encoding/json"
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

	if litellmConfig == nil {
		return nil, fmt.Errorf("no litellm configuration available")
	}
	bundled, err := readBundledCatalog()
	if err != nil {
		return nil, err
	}
	catalog, err := buildModelCatalog(bundled)
	if err != nil {
		return nil, err
	}
	catalogData, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode Codex model catalog: %w", err)
	}
	catalogData = append(catalogData, '\n')
	catalogPath := filepath.Join(filepath.Dir(profilePath), catalogFilename)
	desiredContent := generateConfigContent(model, catalogPath, litellmConfig, otelConfig)

	// Write the catalog first so the profile never points at an absent file.
	// Repair a missing/stale catalog even when the profile itself is unchanged.
	catalogUpdated, err := writeIfChanged(catalogPath, catalogData)
	if err != nil {
		return nil, fmt.Errorf("failed to write Codex model catalog: %w", err)
	}
	profileUpdated, err := writeIfChanged(profilePath, []byte(desiredContent))
	if err != nil {
		return nil, fmt.Errorf("failed to write Codex profile: %w", err)
	}
	updated := catalogUpdated || profileUpdated
	message := "Codex gantry profile and model catalog are already up to date"
	if updated {
		message = "Codex gantry profile and model catalog updated"
	}
	return &ConfigureResult{Updated: updated, ConfigPath: profilePath, Message: message}, nil
}

// Atomic replacement keeps concurrent interactive/headless readers from seeing
// a partial JSON catalog or TOML profile. Unchanged files keep their timestamps.
func writeIfChanged(path string, data []byte) (bool, error) {
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, data) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".gantry-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return false, err
	}
	return true, nil
}

func generateConfigContent(model, catalogPath string, litellmConfig *config.LiteLLMConfig, otelConfig config.OTELConfig) string {
	var sb strings.Builder

	sb.WriteString("# Gantry profile for Codex - LiteLLM gateway configuration\n")
	sb.WriteString("# Launch with: codex --profile gantry\n")
	sb.WriteString(fmt.Sprintf("model = %q\n", model))
	sb.WriteString(fmt.Sprintf("model_catalog_json = %q\n", catalogPath))
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
