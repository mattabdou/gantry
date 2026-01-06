package cmd

import (
	"fmt"
	"os"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/launcher"
	"github.com/mattabdou/gantry/internal/powerline"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gantry [claude-code-args...]",
	Short: "GANTRY - Gateway for AI Navigation, Telemetry, and Runtime Yield",
	Long: `GANTRY is a launcher for Claude Code that configures environment and telemetry.

It configures AWS Bedrock API, enriches OpenTelemetry telemetry with user, project,
and organizational attributes for AI cost tracking, and configures the claude-powerline
status bar.`,
	Version: Version,
	// Allow unknown flags to be passed through to Claude Code
	DisableFlagParsing: true,
}

func init() {
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		runGantry(cmd, args)
	}
	rootCmd.SetVersionTemplate(fmt.Sprintf("GANTRY v%s\n", Version))
}

func Execute() error {
	return rootCmd.Execute()
}

func runGantry(cmd *cobra.Command, args []string) {
	// Handle --help and --version flags manually since we disabled flag parsing
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			showHelp()
			return
		}
		if arg == "--version" || arg == "-v" {
			fmt.Printf("GANTRY v%s\n", Version)
			return
		}
		// Handle subcommands
		if arg == "init" || arg == "config" || arg == "update" || arg == "version" {
			// Re-enable cobra to handle subcommand
			cmd.DisableFlagParsing = false
			cmd.SetArgs(args)
			cmd.Execute()
			return
		}
	}

	// Check for global config
	if !config.GlobalConfigExists() {
		fmt.Fprintln(os.Stderr, "Error: GANTRY is not configured.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Please run the following command to create the configuration file:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  gantry init")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// Load global config
	globalConfig, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get username from config, with env var override
	username := globalConfig.Gantry.Username
	if envUsername := os.Getenv("GANTRY_USERNAME"); envUsername != "" {
		username = envUsername
	}

	// Get current working directory
	workingPath, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	// Find project config (walk up directories like git)
	projectConfig := config.FindProjectConfig(workingPath)
	if projectConfig == nil {
		projectConfig = config.GetDefaultProjectConfig()
	}

	if projectConfig.Path != "" {
		fmt.Printf("Using project config: %s\n", projectConfig.Path)
	} else {
		fmt.Println("No .gantry.json found, using default project name: Unknown")
	}

	// Get git branch
	gitBranch := launcher.GetGitBranch()
	if gitBranch != "" {
		fmt.Printf("Git branch: %s\n", gitBranch)
	}

	// Handle powerline configuration based on enablePowerline setting
	enablePowerline := true // default to enabled
	if globalConfig.Gantry != nil && globalConfig.Gantry.EnablePowerline != nil {
		enablePowerline = *globalConfig.Gantry.EnablePowerline
	}

	if enablePowerline {
		// Configure claude-powerline theme if powerline settings exist
		if globalConfig.Powerline != nil {
			powerlineResult := powerline.UpdatePowerlineSettings(globalConfig.Powerline)
			if powerlineResult.Updated {
				fmt.Printf("Powerline: %s\n", powerlineResult.Message)
			}
		} else {
			// Check for claude-powerline and warn if not configured
			powerlineCheck := powerline.CheckClaudePowerline()
			if !powerlineCheck.Installed {
				fmt.Println()
				fmt.Println("Warning: claude-powerline is not configured.")
				fmt.Println()
				fmt.Println("claude-powerline provides a helpful status line at the bottom of Claude Code.")
				fmt.Println("To enable it, add powerline settings to your ~/.gantryrc.json:")
				fmt.Println()
				fmt.Println("  \"powerline\": {")
				fmt.Println("    \"theme\": \"dark\",")
				fmt.Println("    \"style\": \"powerline\"")
				fmt.Println("  }")
				fmt.Println()
				fmt.Println("Available themes: dark, light, nord, tokyo-night, rose-pine, gruvbox")
				fmt.Println("Available styles: minimal, powerline, capsule")
				fmt.Println()
				fmt.Println("For more information: https://github.com/Owloops/claude-powerline")
				fmt.Println()
			}
		}
	} else {
		// Powerline is disabled - remove any existing configuration
		removeResult := powerline.RemovePowerlineSettings()
		if removeResult.Updated {
			fmt.Printf("Powerline: %s\n", removeResult.Message)
		}
	}

	// Build resource attributes
	resourceAttributes := launcher.BuildResourceAttributes(username, workingPath, projectConfig, gitBranch)

	// Build environment
	env := launcher.BuildEnvironment(globalConfig, resourceAttributes)

	fmt.Println()
	fmt.Println("Starting Claude Code with GANTRY configuration...")
	fmt.Println()

	// Launch Claude Code with all arguments passed through
	if err := launcher.LaunchClaude(args, env); err != nil {
		if err.Error() == "executable file not found in $PATH" || err.Error() == "executable file not found in %PATH%" {
			fmt.Fprintln(os.Stderr, "Error: Claude Code is not installed or not in PATH.")
			fmt.Fprintln(os.Stderr, "Please install Claude Code: npm install -g @anthropic-ai/claude-code")
		} else {
			fmt.Fprintf(os.Stderr, "Error starting Claude Code: %v\n", err)
		}
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf(`
GANTRY - Gateway for AI Navigation, Telemetry, and Runtime Yield v%s

A launcher for Claude Code that configures environment and telemetry.

Usage:
  gantry [claude-code-args...]    Run Claude Code with GANTRY configuration
  gantry init [--force]           Create the global configuration file
  gantry config                   Interactive configuration editor
  gantry config show              Display current configuration
  gantry config get <key>         Get a configuration value
  gantry config set <key> <val>   Set a configuration value
  gantry update                   Update gantry to the latest version
  gantry update --check           Check if an update is available
  gantry version                  Show version information
  gantry version --check          Show version and check for updates
  gantry --help                   Show this help message
  gantry --version                Show version number

Environment Variables:
  GANTRY_USERNAME                 Optional. Overrides gantry.username from config.

Configuration Files:
  ~/.gantryrc.json                Global configuration (created by 'gantry init')
  .gantry.json                    Per-project configuration (optional)

Examples:
  gantry                          Start Claude Code in current directory
  gantry --help                   Show GANTRY help
  gantry init                     Initialize global configuration
  gantry init --force             Reinitialize global configuration
  gantry config                   Edit configuration interactively
  gantry config show              View current configuration
  gantry config set gantry.username john.doe
  gantry config set gantry.enablePowerline false
  gantry config set otel.endpoint https://collector.example.com/otlp
  gantry update                   Update to latest version

For more information, see: https://github.com/mattabdou/gantry
`, Version)
}
