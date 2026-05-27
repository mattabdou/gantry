package claudedesktop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mattabdou/gantry/internal/config"
)

const ConfigFileName = "claude_desktop_config.json"

// ConfigureResult contains the result of configuring Claude Desktop
type ConfigureResult struct {
	Updated    bool
	BackupPath string
	ConfigPath string
	Message    string
}

// GetConfigPath returns the path to the Claude Desktop config file
func GetConfigPath() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "Claude", ConfigFileName), nil
	case "windows":
		return getWindowsConfigPath()
	default:
		return "", fmt.Errorf("Claude Desktop is not supported on %s", runtime.GOOS)
	}
}

// getWindowsConfigPath returns the config path, checking for MSIX installations first
func getWindowsConfigPath() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		// Check for MSIX package installation - uses virtualized filesystem
		packagesDir := filepath.Join(localAppData, "Packages")
		matches, _ := filepath.Glob(filepath.Join(packagesDir, "Claude_*"))
		if len(matches) > 0 {
			// MSIX apps read from LocalCache\Roaming instead of real %APPDATA%
			return filepath.Join(matches[0], "LocalCache", "Roaming", "Claude", ConfigFileName), nil
		}
	}

	// Fall back to standard %APPDATA% path for traditional installations
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("APPDATA environment variable not set")
	}
	return filepath.Join(appData, "Claude", ConfigFileName), nil
}

// LoadConfig loads the Claude Desktop configuration as a generic map to preserve unknown fields
func LoadConfig(configPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to read Claude Desktop config: %w", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON in Claude Desktop config: %w", err)
	}

	return cfg, nil
}

// BackupConfig creates a timestamped backup of the Claude Desktop config
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

// ConfigureGateway configures Claude Desktop to use the LiteLLM gateway
func ConfigureGateway(litellmConfig *config.LiteLLMConfig) (*ConfigureResult, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Check if config already matches
	if cfg["inferenceProvider"] == "gateway" &&
		cfg["inferenceGatewayBaseUrl"] == litellmConfig.BaseURL &&
		cfg["inferenceGatewayApiKey"] == litellmConfig.AuthToken &&
		cfg["inferenceGatewayAuthScheme"] == "bearer" {
		return &ConfigureResult{
			Updated:    false,
			ConfigPath: configPath,
			Message:    "Claude Desktop gateway configuration is already up to date",
		}, nil
	}

	// Create backup before modifying
	backupPath, err := BackupConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Set gateway configuration
	cfg["inferenceProvider"] = "gateway"
	cfg["inferenceGatewayBaseUrl"] = litellmConfig.BaseURL
	cfg["inferenceGatewayApiKey"] = litellmConfig.AuthToken
	cfg["inferenceGatewayAuthScheme"] = "bearer"

	// Ensure the config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write Claude Desktop config: %w", err)
	}

	result := &ConfigureResult{
		Updated:    true,
		BackupPath: backupPath,
		ConfigPath: configPath,
		Message:    "Claude Desktop gateway configuration updated",
	}

	if backupPath != "" {
		result.Message = fmt.Sprintf("Claude Desktop gateway configuration updated (backup: %s)", filepath.Base(backupPath))
	}

	return result, nil
}
