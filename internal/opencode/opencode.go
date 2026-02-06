package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/mattabdou/gantry/internal/config"
)

const (
	// ConfigDirName is the OpenCode config directory name
	ConfigDirName = "opencode"
	// ConfigFileName is the OpenCode config file name
	ConfigFileName = "opencode.json"
	// ConfigFileNameC is the alternative OpenCode config file name with comments
	ConfigFileNameC = "opencode.jsonc"
)

// ProviderOptions contains the options for a provider
type ProviderOptions struct {
	BaseURL  string            `json:"baseURL,omitempty"`
	APIKey   string            `json:"apiKey,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Region   string            `json:"region,omitempty"`
	Profile  string            `json:"profile,omitempty"`
	Endpoint string            `json:"endpoint,omitempty"`
}

// ProviderConfig contains configuration for a single provider
type ProviderConfig struct {
	NPM     string          `json:"npm,omitempty"`
	Name    string          `json:"name,omitempty"`
	Options ProviderOptions `json:"options,omitempty"`
}

// OpenCodeConfig represents the opencode.json configuration structure
// We use a generic map to preserve unknown fields
type OpenCodeConfig map[string]interface{}

// GetConfigDir returns the path to the OpenCode config directory
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", ConfigDirName), nil
}

// GetConfigPath returns the path to the OpenCode config file
// It prefers opencode.json over opencode.jsonc if both exist
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	// Check for opencode.json first
	jsonPath := filepath.Join(configDir, ConfigFileName)
	if _, err := os.Stat(jsonPath); err == nil {
		return jsonPath, nil
	}

	// Check for opencode.jsonc
	jsoncPath := filepath.Join(configDir, ConfigFileNameC)
	if _, err := os.Stat(jsoncPath); err == nil {
		return jsoncPath, nil
	}

	// Return the default path (opencode.json) if neither exists
	return jsonPath, nil
}

// ConfigExists checks if an OpenCode config file exists
func ConfigExists() bool {
	configDir, err := GetConfigDir()
	if err != nil {
		return false
	}

	jsonPath := filepath.Join(configDir, ConfigFileName)
	if _, err := os.Stat(jsonPath); err == nil {
		return true
	}

	jsoncPath := filepath.Join(configDir, ConfigFileNameC)
	if _, err := os.Stat(jsoncPath); err == nil {
		return true
	}

	return false
}

// LoadConfig loads the OpenCode configuration
func LoadConfig() (OpenCodeConfig, string, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return make(OpenCodeConfig), configPath, nil
		}
		return nil, "", fmt.Errorf("failed to read OpenCode config: %w", err)
	}

	// Strip comments if it's a JSONC file
	if filepath.Ext(configPath) == ".jsonc" {
		data = stripJSONComments(data)
	}

	var cfg OpenCodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, "", fmt.Errorf("invalid JSON in OpenCode config: %w", err)
	}

	return cfg, configPath, nil
}

// stripJSONComments removes single-line and multi-line comments from JSON
func stripJSONComments(data []byte) []byte {
	// Remove single-line comments (// ...)
	singleLine := regexp.MustCompile(`(?m)//.*$`)
	data = singleLine.ReplaceAll(data, []byte{})

	// Remove multi-line comments (/* ... */)
	multiLine := regexp.MustCompile(`(?s)/\*.*?\*/`)
	data = multiLine.ReplaceAll(data, []byte{})

	return data
}

// SaveConfig saves the OpenCode configuration
func SaveConfig(cfg OpenCodeConfig, configPath string) error {
	// Ensure the config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OpenCode config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write OpenCode config: %w", err)
	}

	return nil
}

// BackupConfig creates a timestamped backup of the OpenCode config
func BackupConfig(configPath string) (string, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Nothing to backup
		return "", nil
	}

	// Read the original file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config for backup: %w", err)
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	ext := filepath.Ext(configPath)
	base := configPath[:len(configPath)-len(ext)]
	backupPath := fmt.Sprintf("%s.backup.%s%s", base, timestamp, ext)

	// Write the backup
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// ConfigureResult contains the result of configuring OpenCode
type ConfigureResult struct {
	Updated    bool
	BackupPath string
	ConfigPath string
	Message    string
}

// ConfigureLiteLLM configures OpenCode for LiteLLM provider
func ConfigureLiteLLM(litellmConfig *config.LiteLLMConfig) (*ConfigureResult, error) {
	cfg, configPath, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Check if we need to update provider
	needsUpdate := false
	currentProvider := getProviderConfig(cfg, "gantry-litellm")

	if currentProvider == nil {
		needsUpdate = true
	} else {
		// Check if values have changed
		if opts, ok := currentProvider["options"].(map[string]interface{}); ok {
			if baseURL, _ := opts["baseURL"].(string); baseURL != litellmConfig.BaseURL {
				needsUpdate = true
			}
			if apiKey, _ := opts["apiKey"].(string); apiKey != litellmConfig.AuthToken {
				needsUpdate = true
			}
		} else {
			needsUpdate = true
		}
	}

	if !needsUpdate {
		return &ConfigureResult{
			Updated:    false,
			ConfigPath: configPath,
			Message:    "OpenCode LiteLLM configuration is up to date",
		}, nil
	}

	// Create backup before modifying
	backupPath, err := BackupConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Ensure provider section exists
	if cfg["provider"] == nil {
		cfg["provider"] = make(map[string]interface{})
	}

	providers, ok := cfg["provider"].(map[string]interface{})
	if !ok {
		providers = make(map[string]interface{})
		cfg["provider"] = providers
	}

	// Set up gantry-litellm provider with models defined inside
	// The model keys are what get sent to the API, "name" is for display
	providers["gantry-litellm"] = map[string]interface{}{
		"npm":  "@ai-sdk/openai-compatible",
		"name": "Gantry LiteLLM",
		"options": map[string]interface{}{
			"baseURL": litellmConfig.BaseURL,
			"apiKey":  litellmConfig.AuthToken,
		},
		"models": map[string]interface{}{
			"bedrock/us.anthropic.claude-opus-4-6-v1": map[string]interface{}{
				"name": "Claude Opus 4.6",
			},
			"us.anthropic.claude-opus-4-5-20251101-v1:0": map[string]interface{}{
				"name": "Claude Opus 4.5",
			},
			"us.anthropic.claude-sonnet-4-5-20250929-v1:0": map[string]interface{}{
				"name": "Claude Sonnet 4.5",
			},
			"us.anthropic.claude-haiku-4-5-20251001-v1:0": map[string]interface{}{
				"name": "Claude Haiku 4.5",
			},
		},
	}

	// Set default model to use gantry-litellm provider with Opus 4
	// Format is "provider/model"
	cfg["model"] = "gantry-litellm/bedrock/us.anthropic.claude-opus-4-6-v1"

	// Use opencode.json for new configs (not jsonc)
	if !ConfigExists() {
		configDir, _ := GetConfigDir()
		configPath = filepath.Join(configDir, ConfigFileName)
	}

	if err := SaveConfig(cfg, configPath); err != nil {
		return nil, err
	}

	result := &ConfigureResult{
		Updated:    true,
		BackupPath: backupPath,
		ConfigPath: configPath,
		Message:    "OpenCode LiteLLM configuration updated",
	}

	if backupPath != "" {
		result.Message = fmt.Sprintf("OpenCode LiteLLM configuration updated (backup: %s)", filepath.Base(backupPath))
	}

	return result, nil
}

// ConfigureBedrock configures OpenCode for AWS Bedrock provider
func ConfigureBedrock(bedrockConfig *config.BedrockConfig) (*ConfigureResult, error) {
	cfg, configPath, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Check if we need to update provider
	needsUpdate := false
	currentProvider := getProviderConfig(cfg, "gantry-bedrock")

	if currentProvider == nil {
		needsUpdate = true
	} else {
		// Check if values have changed
		if opts, ok := currentProvider["options"].(map[string]interface{}); ok {
			if region, _ := opts["region"].(string); region != bedrockConfig.AWSRegion {
				needsUpdate = true
			}
			if profile, _ := opts["profile"].(string); profile != bedrockConfig.AWSProfile {
				needsUpdate = true
			}
		} else {
			needsUpdate = true
		}
	}

	if !needsUpdate {
		return &ConfigureResult{
			Updated:    false,
			ConfigPath: configPath,
			Message:    "OpenCode Bedrock configuration is up to date",
		}, nil
	}

	// Create backup before modifying
	backupPath, err := BackupConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Ensure provider section exists
	if cfg["provider"] == nil {
		cfg["provider"] = make(map[string]interface{})
	}

	providers, ok := cfg["provider"].(map[string]interface{})
	if !ok {
		providers = make(map[string]interface{})
		cfg["provider"] = providers
	}

	// Set up gantry-bedrock provider using amazon-bedrock format
	bedrockOpts := map[string]interface{}{
		"region":  bedrockConfig.AWSRegion,
		"profile": bedrockConfig.AWSProfile,
	}

	providers["gantry-bedrock"] = map[string]interface{}{
		"name":    "Gantry Bedrock",
		"options": bedrockOpts,
		"models": map[string]interface{}{
			"us.anthropic.claude-opus-4-6-v1": map[string]interface{}{
				"name": "Claude Opus 4.6",
			},
			"us.anthropic.claude-opus-4-5-20251101-v1:0": map[string]interface{}{
				"name": "Claude Opus 4.5",
			},
			"us.anthropic.claude-sonnet-4-5-20250929-v1:0": map[string]interface{}{
				"name": "Claude Sonnet 4.5",
			},
			"us.anthropic.claude-haiku-4-5-20251001-v1:0": map[string]interface{}{
				"name": "Claude Haiku 4.5",
			},
		},
	}

	// Also configure the amazon-bedrock provider with gantry settings
	providers["amazon-bedrock"] = map[string]interface{}{
		"options": bedrockOpts,
	}

	// Set default model to use gantry-bedrock provider with Opus 4
	// Format is "provider/model"
	cfg["model"] = "gantry-bedrock/us.anthropic.claude-opus-4-6-v1"

	// Use opencode.json for new configs (not jsonc)
	if !ConfigExists() {
		configDir, _ := GetConfigDir()
		configPath = filepath.Join(configDir, ConfigFileName)
	}

	if err := SaveConfig(cfg, configPath); err != nil {
		return nil, err
	}

	result := &ConfigureResult{
		Updated:    true,
		BackupPath: backupPath,
		ConfigPath: configPath,
		Message:    "OpenCode Bedrock configuration updated",
	}

	if backupPath != "" {
		result.Message = fmt.Sprintf("OpenCode Bedrock configuration updated (backup: %s)", filepath.Base(backupPath))
	}

	return result, nil
}

// getProviderConfig gets a provider configuration from the config
func getProviderConfig(cfg OpenCodeConfig, providerID string) map[string]interface{} {
	providers, ok := cfg["provider"].(map[string]interface{})
	if !ok {
		return nil
	}

	provider, ok := providers[providerID].(map[string]interface{})
	if !ok {
		return nil
	}

	return provider
}

