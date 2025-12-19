package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage GANTRY configuration",
	Long:  `View and edit GANTRY configuration interactively or via subcommands.`,
	Run: func(cmd *cobra.Command, args []string) {
		editConfig()
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		showConfig()
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long:  `Get a configuration value using dot notation (e.g., otel.endpoint)`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		getConfigValue(args[0])
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  `Set a configuration value using dot notation (e.g., otel.endpoint https://...)`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		value := strings.Join(args[1:], " ")
		setConfigValue(args[0], value)
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Interactive configuration editor",
	Run: func(cmd *cobra.Command, args []string) {
		editConfig()
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configEditCmd)
	rootCmd.AddCommand(configCmd)
}

func maskToken(value string) string {
	if value == "" {
		return "(not set)"
	}
	if strings.Contains(value, "YOUR_TOKEN") || strings.Contains(value, "your-") {
		return value + " (placeholder - needs configuration)"
	}
	if len(value) > 30 {
		return value[:20] + "..." + strings.Repeat("*", 10)
	}
	return value
}

func displayConfig(cfg *config.GlobalConfig) {
	fmt.Println()
	fmt.Println("Current GANTRY Configuration")
	fmt.Println("============================")
	fmt.Println()
	fmt.Println("General Settings:")
	fmt.Printf("  Auto-Update Claude:    %v\n", cfg.AutoUpdate)
	fmt.Println()
	fmt.Println("OTEL Settings:")
	fmt.Printf("  Endpoint:              %s\n", cfg.OTEL.Endpoint)
	fmt.Printf("  Headers:               %s\n", maskToken(cfg.OTEL.Headers))
	fmt.Printf("  Protocol:              %s\n", cfg.OTEL.Protocol)
	fmt.Printf("  Metrics Exporter:      %s\n", cfg.OTEL.MetricsExporter)
	fmt.Printf("  Logs Exporter:         %s\n", cfg.OTEL.LogsExporter)
	fmt.Printf("  Metric Export Interval: %dms\n", cfg.OTEL.MetricExportInterval)
	fmt.Printf("  Logs Export Interval:  %dms\n", cfg.OTEL.LogsExportInterval)
	fmt.Printf("  Log User Prompts:      %v\n", cfg.OTEL.LogUserPrompts)
	if cfg.OTEL.IncludeSessionID != nil {
		fmt.Printf("  Include Session ID:    %v\n", *cfg.OTEL.IncludeSessionID)
	}
	if cfg.OTEL.IncludeVersion != nil {
		fmt.Printf("  Include Version:       %v\n", *cfg.OTEL.IncludeVersion)
	}
	if cfg.OTEL.IncludeAccountUUID != nil {
		fmt.Printf("  Include Account UUID:  %v\n", *cfg.OTEL.IncludeAccountUUID)
	}
	fmt.Println()

	if cfg.Bedrock != nil {
		fmt.Println("AWS Bedrock Settings:")
		fmt.Printf("  Enabled:               %v\n", cfg.Bedrock.Enabled)
		fmt.Printf("  AWS Profile:           %s\n", maskToken(cfg.Bedrock.AWSProfile))
		fmt.Printf("  AWS Region:            %s\n", cfg.Bedrock.AWSRegion)
		fmt.Printf("  Model:                 %s\n", cfg.Bedrock.Model)
		fmt.Printf("  Max Output Tokens:     %d\n", cfg.Bedrock.MaxOutputTokens)
		fmt.Printf("  Max Thinking Tokens:   %d\n", cfg.Bedrock.MaxThinkingTokens)
		fmt.Println()
	}

	if cfg.Powerline != nil {
		fmt.Println("Powerline Settings:")
		fmt.Printf("  Theme:                 %s\n", cfg.Powerline.Theme)
		fmt.Printf("  Style:                 %s\n", cfg.Powerline.Style)
		fmt.Println()
	}
}

func showConfig() {
	configPath, _ := config.GetGlobalConfigPath()

	if !config.GlobalConfigExists() {
		fmt.Fprintf(os.Stderr, "Configuration file not found at %s\n", configPath)
		fmt.Fprintln(os.Stderr, "Run \"gantry init\" to create the configuration file.")
		os.Exit(1)
	}

	cfg, err := config.LoadGlobalConfigRaw()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading configuration: %v\n", err)
		os.Exit(1)
	}

	displayConfig(cfg)
	fmt.Printf("Configuration file: %s\n", configPath)

	// Try to validate and warn if invalid
	if err := config.ValidateGlobalConfig(cfg); err != nil {
		fmt.Println()
		fmt.Printf("Warning: %v\n", err)
	}
}

func getConfigValue(key string) {
	if !config.GlobalConfigExists() {
		fmt.Fprintln(os.Stderr, "Configuration file not found. Run \"gantry init\" first.")
		os.Exit(1)
	}

	cfg, err := config.LoadGlobalConfigRaw()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading configuration: %v\n", err)
		os.Exit(1)
	}

	value, err := config.GetConfigValue(cfg, key)
	if err != nil {
		fmt.Printf("Key %q not found\n", key)
		os.Exit(1)
	}

	fmt.Println(value)
}

func setConfigValue(key, value string) {
	configPath, _ := config.GetGlobalConfigPath()

	var cfg *config.GlobalConfig
	var err error

	if config.GlobalConfigExists() {
		cfg, err = config.LoadGlobalConfigRaw()
		if err != nil {
			cfg = config.GetDefaultGlobalConfigTemplate()
		}
	} else {
		cfg = config.GetDefaultGlobalConfigTemplate()
	}

	if err := config.SetConfigValue(cfg, key, value); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting value: %v\n", err)
		os.Exit(1)
	}

	if err := config.SaveGlobalConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	// Get the value back to show what was actually set
	newValue, _ := config.GetConfigValue(cfg, key)
	fmt.Printf("Set %s = %v\n", key, newValue)
	fmt.Printf("Configuration saved to %s\n", configPath)
}

func prompt(scanner *bufio.Scanner, question string, defaultValue string) string {
	defaultDisplay := ""
	if defaultValue != "" {
		defaultDisplay = fmt.Sprintf(" (%s)", defaultValue)
	}

	fmt.Printf("%s%s: ", question, defaultDisplay)
	if scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			return defaultValue
		}
		return text
	}
	return defaultValue
}

func promptBoolean(scanner *bufio.Scanner, question string, defaultValue bool) bool {
	defaultDisplay := "y/N"
	if defaultValue {
		defaultDisplay = "Y/n"
	}

	fmt.Printf("%s (%s): ", question, defaultDisplay)
	if scanner.Scan() {
		text := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if text == "" {
			return defaultValue
		}
		if text == "y" || text == "yes" || text == "true" {
			return true
		}
		if text == "n" || text == "no" || text == "false" {
			return false
		}
		return defaultValue
	}
	return defaultValue
}

func promptNumber(scanner *bufio.Scanner, question string, defaultValue int) int {
	fmt.Printf("%s (%d): ", question, defaultValue)
	if scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			return defaultValue
		}
		if num, err := strconv.Atoi(text); err == nil {
			return num
		}
		return defaultValue
	}
	return defaultValue
}

func editConfig() {
	configPath, _ := config.GetGlobalConfigPath()

	var cfg *config.GlobalConfig
	var err error

	if config.GlobalConfigExists() {
		cfg, err = config.LoadGlobalConfigRaw()
		if err != nil {
			cfg = config.GetDefaultGlobalConfigTemplate()
		}
	} else {
		cfg = config.GetDefaultGlobalConfigTemplate()
	}

	// Ensure all sections exist
	if cfg.Bedrock == nil {
		cfg.Bedrock = config.GetDefaultGlobalConfigTemplate().Bedrock
	}
	if cfg.Powerline == nil {
		cfg.Powerline = config.GetDefaultGlobalConfigTemplate().Powerline
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Println("GANTRY Configuration Editor")
	fmt.Println("===========================")
	fmt.Println("Press Enter to keep the current value.")
	fmt.Println()

	// General settings
	fmt.Println("--- General Settings ---")
	fmt.Println()

	cfg.AutoUpdate = promptBoolean(scanner, "Auto-update Claude Code on launch?", cfg.AutoUpdate)

	fmt.Println()

	// Required settings
	fmt.Println("--- Required Settings ---")
	fmt.Println()

	cfg.OTEL.Endpoint = prompt(scanner, "OTEL Endpoint URL", cfg.OTEL.Endpoint)
	cfg.OTEL.Headers = prompt(scanner, "OTEL Headers (e.g., Authorization=Bearer TOKEN)", cfg.OTEL.Headers)

	fmt.Println()
	fmt.Println("--- Protocol & Exporters ---")
	fmt.Println()

	protocol := cfg.OTEL.Protocol
	if protocol == "" {
		protocol = "http/protobuf"
	}
	cfg.OTEL.Protocol = prompt(scanner, "OTEL Protocol (http/protobuf, http/json, grpc)", protocol)

	metricsExporter := cfg.OTEL.MetricsExporter
	if metricsExporter == "" {
		metricsExporter = "otlp"
	}
	cfg.OTEL.MetricsExporter = prompt(scanner, "Metrics Exporter (otlp, console, prometheus)", metricsExporter)

	logsExporter := cfg.OTEL.LogsExporter
	if logsExporter == "" {
		logsExporter = "otlp"
	}
	cfg.OTEL.LogsExporter = prompt(scanner, "Logs Exporter (otlp, console)", logsExporter)

	fmt.Println()
	fmt.Println("--- Export Intervals ---")
	fmt.Println()

	metricInterval := cfg.OTEL.MetricExportInterval
	if metricInterval == 0 {
		metricInterval = 60000
	}
	cfg.OTEL.MetricExportInterval = promptNumber(scanner, "Metric Export Interval (ms)", metricInterval)

	logsInterval := cfg.OTEL.LogsExportInterval
	if logsInterval == 0 {
		logsInterval = 5000
	}
	cfg.OTEL.LogsExportInterval = promptNumber(scanner, "Logs Export Interval (ms)", logsInterval)

	fmt.Println()
	fmt.Println("--- Optional Flags ---")
	fmt.Println()

	cfg.OTEL.LogUserPrompts = promptBoolean(scanner, "Log user prompts?", cfg.OTEL.LogUserPrompts)

	includeSessionID := true
	if cfg.OTEL.IncludeSessionID != nil {
		includeSessionID = *cfg.OTEL.IncludeSessionID
	}
	sessionID := promptBoolean(scanner, "Include session ID in metrics?", includeSessionID)
	cfg.OTEL.IncludeSessionID = &sessionID

	includeVersion := false
	if cfg.OTEL.IncludeVersion != nil {
		includeVersion = *cfg.OTEL.IncludeVersion
	}
	version := promptBoolean(scanner, "Include app version in metrics?", includeVersion)
	cfg.OTEL.IncludeVersion = &version

	includeAccountUUID := true
	if cfg.OTEL.IncludeAccountUUID != nil {
		includeAccountUUID = *cfg.OTEL.IncludeAccountUUID
	}
	accountUUID := promptBoolean(scanner, "Include account UUID in metrics?", includeAccountUUID)
	cfg.OTEL.IncludeAccountUUID = &accountUUID

	fmt.Println()
	fmt.Println("--- AWS Bedrock Settings ---")
	fmt.Println()

	cfg.Bedrock.Enabled = promptBoolean(scanner, "Enable AWS Bedrock?", cfg.Bedrock.Enabled)

	if cfg.Bedrock.Enabled {
		cfg.Bedrock.AWSProfile = prompt(scanner, "AWS Profile", cfg.Bedrock.AWSProfile)

		awsRegion := cfg.Bedrock.AWSRegion
		if awsRegion == "" {
			awsRegion = "us-east-2"
		}
		cfg.Bedrock.AWSRegion = prompt(scanner, "AWS Region", awsRegion)

		model := cfg.Bedrock.Model
		if model == "" {
			model = "us.anthropic.claude-opus-4-5-20251101-v1:0"
		}
		cfg.Bedrock.Model = prompt(scanner, "Anthropic Model", model)

		maxOutput := cfg.Bedrock.MaxOutputTokens
		if maxOutput == 0 {
			maxOutput = 8192
		}
		cfg.Bedrock.MaxOutputTokens = promptNumber(scanner, "Max Output Tokens", maxOutput)

		maxThinking := cfg.Bedrock.MaxThinkingTokens
		if maxThinking == 0 {
			maxThinking = 1024
		}
		cfg.Bedrock.MaxThinkingTokens = promptNumber(scanner, "Max Thinking Tokens", maxThinking)
	}

	fmt.Println()
	fmt.Println("--- Powerline Settings ---")
	fmt.Println("(Themes: dark, light, nord, tokyo-night, rose-pine, gruvbox)")
	fmt.Println("(Styles: minimal, powerline, capsule)")
	fmt.Println()

	theme := cfg.Powerline.Theme
	if theme == "" {
		theme = "dark"
	}
	cfg.Powerline.Theme = prompt(scanner, "Powerline Theme", theme)

	style := cfg.Powerline.Style
	if style == "" {
		style = "powerline"
	}
	cfg.Powerline.Style = prompt(scanner, "Powerline Style", style)

	// Save configuration
	if err := config.SaveGlobalConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Configuration saved successfully.")
	fmt.Printf("File: %s\n", configPath)
	fmt.Println()
}
