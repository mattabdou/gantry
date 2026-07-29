package opencode

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/jsonconf"
)

const (
	// ConfigDirName is the OpenCode config directory name
	ConfigDirName = "opencode"
	// ConfigFileName is the OpenCode config file name
	ConfigFileName = "opencode.json"
	// ConfigFileNameC is the alternative OpenCode config file name with comments
	ConfigFileNameC = "opencode.jsonc"
)

// OpenCodeConfig represents the opencode.json configuration structure.
//
// It is deliberately a generic map rather than a struct. The file belongs to the
// user and to OpenCode, and holds keys GANTRY knows nothing about - MCP servers,
// agents, themes, keybinds, permissions. Unmarshalling into a struct and
// marshalling back would silently delete all of them.
type OpenCodeConfig map[string]interface{}

// userHomeDir is a seam for tests, which need a writable home directory to
// exercise the backup-and-write behaviour.
var userHomeDir = os.UserHomeDir

// GetConfigDir returns the path to the OpenCode config directory
func GetConfigDir() (string, error) {
	home, err := userHomeDir()
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

// LoadConfig loads the OpenCode configuration, returning it alongside the path
// it came from. A missing file yields an empty config and no error.
func LoadConfig() (OpenCodeConfig, string, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config yet: an empty one is the correct starting point.
			return make(OpenCodeConfig), configPath, nil
		}
		return nil, "", fmt.Errorf("failed to read OpenCode config: %w", err)
	}

	// Strip comments if it's a JSONC file
	if filepath.Ext(configPath) == ".jsonc" {
		data = stripJSONComments(data)
	}

	cfg, err := jsonconf.UnmarshalObject(data)
	if err != nil {
		return nil, "", fmt.Errorf("invalid JSON in OpenCode config %s: %w", configPath, err)
	}

	return OpenCodeConfig(cfg), configPath, nil
}

// stripJSONComments removes // and /* */ comments from JSONC.
//
// It scans rather than pattern-matching because comment markers occur inside
// legitimate string values. A regex approach truncates every URL - and GANTRY
// itself writes baseURL into this file - which turned any .jsonc config into
// invalid JSON:
//
//	"baseURL": "https://llm.example.com"  ->  "baseURL": "https:
//
// Comments are replaced with a space rather than deleted so that a comment
// between two tokens cannot join them together.
func stripJSONComments(data []byte) []byte {
	out := make([]byte, 0, len(data))

	inString := false
	for i := 0; i < len(data); i++ {
		c := data[i]

		if inString {
			out = append(out, c)
			switch c {
			case '\\':
				// Copy the escaped byte verbatim so \" does not end the string.
				if i+1 < len(data) {
					i++
					out = append(out, data[i])
				}
			case '"':
				inString = false
			}
			continue
		}

		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}

		if c == '/' && i+1 < len(data) {
			switch data[i+1] {
			case '/':
				// Line comment: skip to the newline, keeping it so that line
				// numbers in any subsequent parse error stay meaningful.
				i += 2
				for i < len(data) && data[i] != '\n' {
					i++
				}
				if i < len(data) {
					out = append(out, '\n')
				}
				continue
			case '*':
				// Block comment: skip to the terminator. An unterminated block
				// comment consumes the remainder, which json.Unmarshal will
				// then report as a syntax error.
				i += 2
				for i+1 < len(data) && !(data[i] == '*' && data[i+1] == '/') {
					i++
				}
				if i+1 < len(data) {
					i++ // land on '/'; the loop's i++ steps past it
				} else {
					i = len(data)
				}
				out = append(out, ' ')
				continue
			}
		}

		out = append(out, c)
	}

	return out
}

// SaveConfig saves the OpenCode configuration.
//
// Writing a .jsonc file drops the user's comments, since the config is
// round-tripped as plain JSON. Callers only write when the configuration has
// actually changed, so this is rare, but it is warned about.
func SaveConfig(cfg OpenCodeConfig, configPath string) error {
	if filepath.Ext(configPath) == ".jsonc" {
		fmt.Fprintf(os.Stderr, "Warning: rewriting %s as plain JSON; comments in that file will be lost.\n",
			filepath.Base(configPath))
	}
	if err := jsonconf.WriteObject(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write OpenCode config: %w", err)
	}
	return nil
}

// BackupConfig creates a timestamped backup of the OpenCode config. A missing
// file is not an error and yields an empty path.
func BackupConfig(configPath string) (string, error) {
	return jsonconf.Backup(configPath)
}

// ConfigureResult contains the result of configuring OpenCode
type ConfigureResult struct {
	Updated    bool
	BackupPath string
	ConfigPath string
	Message    string
}

// buildFunc produces the configuration that should be on disk from the
// configuration that currently is.
type buildFunc func(OpenCodeConfig) (OpenCodeConfig, error)

// apply loads the OpenCode config, runs build over it, and writes the result
// only when it differs from what is already on disk.
//
// Writing conditionally is what keeps GANTRY out of the user's way. The earlier
// implementation decided whether to update by counting models, so a user who
// added one of their own made that check fire on every launch - rewriting the
// config and leaving another .gantrybackup file behind each time. Comparing the
// built config to the current one has no such coupling: no semantic change means
// no write, and no write means no backup.
func apply(build buildFunc, label string) (*ConfigureResult, error) {
	current, configPath, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	desired, err := build(current)
	if err != nil {
		return nil, err
	}

	if jsonconf.Equal(desired, current) {
		return &ConfigureResult{
			Updated:    false,
			ConfigPath: configPath,
			Message:    label + " is up to date",
		}, nil
	}

	backupPath, err := BackupConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	if err := SaveConfig(desired, configPath); err != nil {
		return nil, err
	}

	result := &ConfigureResult{
		Updated:    true,
		BackupPath: backupPath,
		ConfigPath: configPath,
		Message:    label + " updated",
	}
	if backupPath != "" {
		result.Message = fmt.Sprintf("%s updated (backup: %s)", label, filepath.Base(backupPath))
	}
	return result, nil
}

// ConfigureLiteLLM points OpenCode at the LiteLLM gateway, merging into the
// user's existing configuration rather than replacing it.
func ConfigureLiteLLM(litellmConfig *config.LiteLLMConfig) (*ConfigureResult, error) {
	return apply(func(current OpenCodeConfig) (OpenCodeConfig, error) {
		return BuildLiteLLMConfig(current, litellmConfig)
	}, "OpenCode LiteLLM configuration")
}

// ConfigureBedrock points OpenCode at AWS Bedrock, merging into the user's
// existing configuration rather than replacing it.
func ConfigureBedrock(bedrockConfig *config.BedrockConfig) (*ConfigureResult, error) {
	return apply(func(current OpenCodeConfig) (OpenCodeConfig, error) {
		return BuildBedrockConfig(current, bedrockConfig)
	}, "OpenCode Bedrock configuration")
}

// ResetConfig restores GANTRY's own OpenCode settings to their defaults.
//
// Only the keys GANTRY owns are reset - its provider entries and the top-level
// default model. The user's MCP servers, agents, themes, keybinds and
// permissions are left alone, because that is not what "reset gantry's
// configuration" means. A backup is taken unconditionally here, unlike the
// Configure functions: the user asked for a reset and should get the safety net
// whether or not anything ended up changing.
//
// The reset and the rebuild are composed in memory so that the whole operation
// is a single backup and a single write.
func ResetConfig(litellmConfig *config.LiteLLMConfig, bedrockConfig *config.BedrockConfig, mode string) (*ConfigureResult, error) {
	current, configPath, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	base := ResetGantryKeys(current)

	var desired OpenCodeConfig
	switch {
	case mode == "litellm" && litellmConfig != nil:
		desired, err = BuildLiteLLMConfig(base, litellmConfig)
	case mode == "bedrock" && bedrockConfig != nil:
		desired, err = BuildBedrockConfig(base, bedrockConfig)
	default:
		return nil, fmt.Errorf("no provider config available for mode %q", mode)
	}
	if err != nil {
		return nil, err
	}

	backupPath, err := BackupConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	if err := SaveConfig(desired, configPath); err != nil {
		return nil, err
	}

	result := &ConfigureResult{
		Updated:    true,
		BackupPath: backupPath,
		ConfigPath: configPath,
		Message:    "OpenCode configuration created with defaults",
	}
	if backupPath != "" {
		result.Message = fmt.Sprintf("GANTRY's OpenCode settings reset to defaults, other settings preserved (backup: %s)",
			filepath.Base(backupPath))
	}
	return result, nil
}
