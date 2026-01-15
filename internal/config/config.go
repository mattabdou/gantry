package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// GlobalConfigFilename is the name of the global config file
	GlobalConfigFilename = ".gantryrc.json"
	// ProjectConfigFilename is the name of the project config file
	ProjectConfigFilename = ".gantry.json"
)

// GetGlobalConfigPath returns the path to the global config file (~/.gantryrc.json)
func GetGlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, GlobalConfigFilename), nil
}

// GlobalConfigExists checks if the global config file exists
func GlobalConfigExists() bool {
	path, err := GetGlobalConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// LoadGlobalConfig loads and parses the global config file
func LoadGlobalConfig() (*GlobalConfig, error) {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("global configuration file not found at %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config GlobalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", configPath, err)
	}

	if err := ValidateGlobalConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ValidateGlobalConfig validates the global config has required fields
func ValidateGlobalConfig(config *GlobalConfig) error {
	// Validate gantry section
	if config.Gantry == nil || config.Gantry.Username == "" {
		return errors.New("global config missing \"gantry.username\" - please configure your username")
	}

	if strings.Contains(config.Gantry.Username, "YOUR_") || strings.Contains(config.Gantry.Username, "your-") {
		return errors.New("global config \"gantry.username\" contains placeholder value - please configure your actual username")
	}

	// Validate OTEL section
	if config.OTEL.Endpoint == "" {
		return errors.New("global config missing \"otel.endpoint\" - please configure your OTEL collector endpoint")
	}

	if strings.Contains(config.OTEL.Endpoint, "YOUR_") || strings.Contains(config.OTEL.Endpoint, "your-") {
		return errors.New("global config \"otel.endpoint\" contains placeholder value - please configure your actual OTEL collector endpoint")
	}

	return nil
}

// FindProjectConfig walks up directories looking for .gantry.json
func FindProjectConfig(startDir string) *ProjectConfigResult {
	currentDir, err := filepath.Abs(startDir)
	if err != nil {
		return nil
	}

	root := filepath.VolumeName(currentDir) + string(filepath.Separator)
	if root == string(filepath.Separator) {
		root = "/"
	}

	for currentDir != root {
		configPath := filepath.Join(currentDir, ProjectConfigFilename)

		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err != nil {
				// Continue searching up if can't read
				currentDir = filepath.Dir(currentDir)
				continue
			}

			var config ProjectConfig
			if err := json.Unmarshal(data, &config); err != nil {
				// Warn about invalid JSON but continue searching
				fmt.Fprintf(os.Stderr, "Warning: Invalid JSON in %s, skipping\n", configPath)
				currentDir = filepath.Dir(currentDir)
				continue
			}

			return &ProjectConfigResult{
				Config:    config,
				Path:      configPath,
				Directory: currentDir,
			}
		}

		currentDir = filepath.Dir(currentDir)
	}

	return nil
}

// GetDefaultProjectConfig returns default project config when .gantry.json is not found
func GetDefaultProjectConfig() *ProjectConfigResult {
	return &ProjectConfigResult{
		Config: ProjectConfig{
			ProjectName: "Unknown",
		},
	}
}

// GetDefaultGlobalConfigTemplate returns the default global config template
func GetDefaultGlobalConfigTemplate() *GlobalConfig {
	trueVal := true
	falseVal := false

	return &GlobalConfig{
		Gantry: &GantryConfig{
			Mode:                "YOUR_MODE_HERE",
			Username:            "YOUR_USERNAME_HERE",
			IgnorePowerline:     &trueVal,
			EnablePowerline:     &trueVal,
			BypassLoadingScreen: &falseVal,
		},
		OTEL: OTELConfig{
			Endpoint:             "https://your-otel-collector.example.com/otlp",
			Headers:              "Authorization=Bearer YOUR_TOKEN_HERE",
			Protocol:             "http/protobuf",
			MetricsExporter:      "otlp",
			LogsExporter:         "otlp",
			MetricExportInterval: 60000,
			LogsExportInterval:   5000,
			LogUserPrompts:       false,
			IncludeSessionID:     &trueVal,
			IncludeVersion:       &falseVal,
			IncludeAccountUUID:   &trueVal,
		},
		Bedrock: &BedrockConfig{
			AWSProfile:        "YOUR_AWS_PROFILE",
			AWSRegion:         "us-east-2",
			Model:             "us.anthropic.claude-opus-4-5-20251101-v1:0",
			MaxOutputTokens:   8192,
			MaxThinkingTokens: 1024,
		},
		LiteLLM: &LiteLLMConfig{
			BaseURL:           "https://your-litellm-proxy.example.com",
			AuthToken:         "YOUR_AUTH_TOKEN_HERE",
			Model:             "us.anthropic.claude-opus-4-5-20251101-v1:0",
			MaxOutputTokens:   8192,
			MaxThinkingTokens: 1024,
		},
		Powerline: &PowerlineConfig{
			Theme: "dark",
			Style: "powerline",
		},
	}
}

// SaveGlobalConfig saves the global config to file
func SaveGlobalConfig(config *GlobalConfig) error {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadGlobalConfigRaw loads the global config without validation (for display purposes)
func LoadGlobalConfigRaw() (*GlobalConfig, error) {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config GlobalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", configPath, err)
	}

	return &config, nil
}

// GetConfigValue gets a value from the config using dot notation
func GetConfigValue(config *GlobalConfig, key string) (interface{}, error) {
	keys := strings.Split(key, ".")

	// Convert config to map for easier traversal
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	var current interface{} = m
	for _, k := range keys {
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[k]
			if !ok {
				return nil, fmt.Errorf("key %q not found", key)
			}
		default:
			return nil, fmt.Errorf("key %q not found", key)
		}
	}

	return current, nil
}

// SetConfigValue sets a value in the config using dot notation
func SetConfigValue(config *GlobalConfig, key string, value string) error {
	// Convert config to map for modification
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	keys := strings.Split(key, ".")
	current := m

	// Navigate to the parent of the target key
	for i := 0; i < len(keys)-1; i++ {
		k := keys[i]
		if _, ok := current[k]; !ok {
			current[k] = make(map[string]interface{})
		}
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return fmt.Errorf("cannot set nested key on non-object value at %q", k)
		}
	}

	finalKey := keys[len(keys)-1]

	// Type coercion based on known types
	booleanKeys := map[string]bool{
		"logUserPrompts": true, "includeSessionId": true,
		"includeVersion": true, "includeAccountUuid": true,
		"ignorePowerline": true, "enablePowerline": true, "bypassLoadingScreen": true,
	}
	numberKeys := map[string]bool{
		"metricExportInterval": true, "logsExportInterval": true,
		"maxOutputTokens": true, "maxThinkingTokens": true,
	}

	if booleanKeys[finalKey] {
		current[finalKey] = value == "true" || value == "1" || value == "yes"
	} else if numberKeys[finalKey] {
		var num int
		fmt.Sscanf(value, "%d", &num)
		current[finalKey] = num
	} else {
		current[finalKey] = value
	}

	// Convert back to config struct
	data, err = json.Marshal(m)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, config); err != nil {
		return err
	}

	return nil
}
