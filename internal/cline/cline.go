package cline

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattabdou/gantry/internal/config"
)

// ConfigureResult contains the result of configuring the Cline plugin
type ConfigureResult struct {
	Updated    bool
	BackupPath string
	ConfigPath string
	Message    string
}

// sanitizeModel strips context window suffixes like "[1m]" that are specific
// to Claude Code and not recognized by other tools.
func sanitizeModel(model string) string {
	if idx := strings.Index(model, "["); idx >= 0 {
		return model[:idx]
	}
	return model
}

// ConfigureProvider runs cline auth to configure the OpenAI-native provider with the LiteLLM gateway
func ConfigureProvider(litellmConfig *config.LiteLLMConfig) error {
	model := sanitizeModel(litellmConfig.Model)
	cmd := exec.Command("cline", "auth",
		"--provider", "openai-native",
		"--apikey", litellmConfig.AuthToken,
		"--modelid", model,
		"--baseurl", litellmConfig.BaseURL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cline auth failed: %w\n%s", err, string(output))
	}

	return nil
}

// ConfigurePlugin writes the Cline provider config directly to ~/.cline/data/settings/providers.json
func ConfigurePlugin(litellmConfig *config.LiteLLMConfig) (*ConfigureResult, error) {
	configPath, err := getProvidersPath()
	if err != nil {
		return nil, err
	}

	model := sanitizeModel(litellmConfig.Model)

	// Build desired provider config
	desired := map[string]interface{}{
		"openai-native": map[string]interface{}{
			"apiKey":  litellmConfig.AuthToken,
			"baseUrl": litellmConfig.BaseURL,
			"modelId": model,
		},
	}

	// Check if existing config already matches
	if existingData, err := os.ReadFile(configPath); err == nil {
		var existing map[string]interface{}
		if json.Unmarshal(existingData, &existing) == nil {
			if provider, ok := existing["openai-native"].(map[string]interface{}); ok {
				if provider["apiKey"] == litellmConfig.AuthToken &&
					provider["baseUrl"] == litellmConfig.BaseURL &&
					provider["modelId"] == model {
					return &ConfigureResult{
						Updated:    false,
						ConfigPath: configPath,
						Message:    "Cline VS Code plugin configuration is already up to date",
					}, nil
				}
			}
		}
	}

	// Back up existing file
	backupPath, err := backupProviders(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// If existing file has other providers, merge
	finalConfig := desired
	if existingData, err := os.ReadFile(configPath); err == nil {
		var existing map[string]interface{}
		if json.Unmarshal(existingData, &existing) == nil {
			for k, v := range desired {
				existing[k] = v
			}
			finalConfig = existing
		}
	}

	data, err := json.MarshalIndent(finalConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write providers.json: %w", err)
	}

	result := &ConfigureResult{
		Updated:    true,
		BackupPath: backupPath,
		ConfigPath: configPath,
		Message:    "Cline VS Code plugin configured for LiteLLM gateway",
	}

	if backupPath != "" {
		result.Message = fmt.Sprintf("Cline VS Code plugin configured for LiteLLM gateway (backup: %s)", filepath.Base(backupPath))
	}

	return result, nil
}

func getProvidersPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".cline", "data", "settings", "providers.json"), nil
}

func backupProviders(configPath string) (string, error) {
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
