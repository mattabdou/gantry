package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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

	// ============================================
	// PHASE 1: Gather all information (no actions)
	// ============================================

	// Get username from config, with env var override
	username := globalConfig.Gantry.Username
	usernameSource := "config"
	if envUsername := os.Getenv("GANTRY_USERNAME"); envUsername != "" {
		username = envUsername
		usernameSource = "env"
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

	// Get git branch
	gitBranch := launcher.GetGitBranch()

	// Determine powerline action
	ignorePowerline := true
	if globalConfig.Gantry != nil && globalConfig.Gantry.IgnorePowerline != nil {
		ignorePowerline = *globalConfig.Gantry.IgnorePowerline
	}

	enablePowerline := true
	if globalConfig.Gantry != nil && globalConfig.Gantry.EnablePowerline != nil {
		enablePowerline = *globalConfig.Gantry.EnablePowerline
	}

	// Determine powerline status message
	var powerlineAction string
	if ignorePowerline {
		powerlineAction = "Ignored (no changes will be made)"
	} else if enablePowerline {
		if globalConfig.Powerline != nil {
			theme := globalConfig.Powerline.Theme
			if theme == "" {
				theme = "dark"
			}
			style := globalConfig.Powerline.Style
			if style == "" {
				style = "powerline"
			}
			powerlineAction = fmt.Sprintf("Configure (theme=%s, style=%s)", theme, style)
		} else {
			powerlineAction = "Enabled (no theme configured)"
		}
	} else {
		powerlineAction = "Disabled (will remove if exists)"
	}

	// Check if we should bypass the loading screen
	bypassLoadingScreen := false
	if globalConfig.Gantry != nil && globalConfig.Gantry.BypassLoadingScreen != nil {
		bypassLoadingScreen = *globalConfig.Gantry.BypassLoadingScreen
	}

	// ============================================
	// PHASE 2: Show confirmation screen (if enabled)
	// ============================================

	if !bypassLoadingScreen {
		fmt.Println()
		fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
		fmt.Println("║              GANTRY - Claude Code Launcher                       ║")
		fmt.Printf("║              Version: %-43s ║\n", Version)
		fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
		fmt.Println("║  The following configuration will be applied:                    ║")
		fmt.Println("╠══════════════════════════════════════════════════════════════════╣")

		// User info
		fmt.Println("║  USER IDENTITY                                                   ║")
		if usernameSource == "env" {
			fmt.Printf("║    Username:        %-44s ║\n", truncateString(username, 44)+" (env override)")
		} else {
			fmt.Printf("║    Username:        %-44s ║\n", truncateString(username, 44))
		}
		fmt.Println("║                                                                  ║")

		// Project info
		fmt.Println("║  PROJECT                                                         ║")
		fmt.Printf("║    Project Name:    %-44s ║\n", truncateString(projectConfig.Config.ProjectName, 44))
		if projectConfig.Path != "" {
			fmt.Printf("║    Config File:     %-44s ║\n", truncateString(projectConfig.Path, 44))
		} else {
			fmt.Printf("║    Config File:     %-44s ║\n", "(none - using defaults)")
		}
		if gitBranch != "" {
			fmt.Printf("║    Git Branch:      %-44s ║\n", truncateString(gitBranch, 44))
		}
		fmt.Println("║                                                                  ║")

		// OTEL info
		fmt.Println("║  TELEMETRY (OTEL)                                                ║")
		fmt.Printf("║    Endpoint:        %-44s ║\n", truncateString(globalConfig.OTEL.Endpoint, 44))
		fmt.Println("║                                                                  ║")

		// Bedrock info
		fmt.Println("║  AWS BEDROCK                                                     ║")
		if globalConfig.Bedrock != nil && globalConfig.Bedrock.Enabled {
			fmt.Printf("║    Status:          %-44s ║\n", "Enabled")
			fmt.Printf("║    AWS Profile:     %-44s ║\n", truncateString(globalConfig.Bedrock.AWSProfile, 44))
			fmt.Printf("║    Region:          %-44s ║\n", globalConfig.Bedrock.AWSRegion)
			fmt.Printf("║    Model:           %-44s ║\n", truncateString(globalConfig.Bedrock.Model, 44))
		} else {
			fmt.Printf("║    Status:          %-44s ║\n", "Disabled")
		}
		fmt.Println("║                                                                  ║")

		// Powerline info
		fmt.Println("║  POWERLINE STATUS BAR                                            ║")
		fmt.Printf("║    Action:          %-44s ║\n", truncateString(powerlineAction, 44))
		fmt.Println("║                                                                  ║")

		fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
		fmt.Println("║  Press ENTER to continue, or 'q' to cancel...                    ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════╝")

		// Wait for user input
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "q" || input == "quit" || input == "n" || input == "no" || input == "cancel" {
			fmt.Println("Cancelled.")
			os.Exit(0)
		}
	}

	// ============================================
	// PHASE 3: Execute actions
	// ============================================

	if bypassLoadingScreen {
		// Show minimal output when bypassing
		if projectConfig.Path != "" {
			fmt.Printf("Using project config: %s\n", projectConfig.Path)
		}
		if gitBranch != "" {
			fmt.Printf("Git branch: %s\n", gitBranch)
		}
	}

	// Handle powerline configuration (skip if ignorePowerline is true)
	if !ignorePowerline {
		if enablePowerline {
			if globalConfig.Powerline != nil {
				powerlineResult := powerline.UpdatePowerlineSettings(globalConfig.Powerline)
				if powerlineResult.Updated {
					fmt.Printf("Powerline: %s\n", powerlineResult.Message)
				}
			} else if bypassLoadingScreen {
				// Only show warning if bypassing loading screen (otherwise it was shown in confirmation)
				powerlineCheck := powerline.CheckClaudePowerline()
				if !powerlineCheck.Installed {
					fmt.Println()
					fmt.Println("Warning: claude-powerline is not configured.")
					fmt.Println("To enable it, add powerline settings to your ~/.gantryrc.json")
					fmt.Println()
				}
			}
		} else {
			removeResult := powerline.RemovePowerlineSettings()
			if removeResult.Updated {
				fmt.Printf("Powerline: %s\n", removeResult.Message)
			}
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

// truncateString truncates a string to maxLen and adds "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
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
  gantry config set gantry.ignorePowerline false
  gantry config set gantry.enablePowerline false
  gantry config set gantry.bypassLoadingScreen true
  gantry config set otel.endpoint https://collector.example.com/otlp
  gantry update                   Update to latest version

For more information, see: https://github.com/mattabdou/gantry
`, Version)
}
