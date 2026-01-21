package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/launcher"
	"github.com/mattabdou/gantry/internal/powerline"
	"github.com/mattabdou/gantry/internal/updater"
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

// parseModeFlag extracts --mode or -m flag from args and returns the mode value and filtered args
func parseModeFlag(args []string) (mode string, filteredArgs []string) {
	filteredArgs = make([]string, 0, len(args))
	skipNext := false

	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		// Handle --mode=value or -m=value
		if strings.HasPrefix(arg, "--mode=") {
			mode = strings.TrimPrefix(arg, "--mode=")
			continue
		}
		if strings.HasPrefix(arg, "-m=") {
			mode = strings.TrimPrefix(arg, "-m=")
			continue
		}

		// Handle --mode value or -m value
		if arg == "--mode" || arg == "-m" {
			if i+1 < len(args) {
				mode = args[i+1]
				skipNext = true
			}
			continue
		}

		filteredArgs = append(filteredArgs, arg)
	}

	return mode, filteredArgs
}

func runGantry(cmd *cobra.Command, args []string) {
	// Parse --mode flag from args before other processing
	modeFlag, filteredArgs := parseModeFlag(args)

	// Handle --help and --version flags manually since we disabled flag parsing
	for _, arg := range filteredArgs {
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

	// Determine effective mode: command-line flag overrides config
	mode := modeFlag
	modeSource := "flag"
	if mode == "" {
		if globalConfig.Gantry != nil && globalConfig.Gantry.Mode != "" {
			mode = globalConfig.Gantry.Mode
			modeSource = "config"
		}
	}

	// Validate mode is specified
	if mode == "" {
		fmt.Fprintln(os.Stderr, "Error: mode must be specified.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Set \"gantry.mode\" in ~/.gantryrc.json to \"bedrock\" or \"litellm\",")
		fmt.Fprintln(os.Stderr, "or use the --mode flag:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  gantry --mode bedrock")
		fmt.Fprintln(os.Stderr, "  gantry --mode litellm")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// Validate mode value
	if mode != "bedrock" && mode != "litellm" {
		fmt.Fprintf(os.Stderr, "Error: invalid mode %q.\n", mode)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Mode must be \"bedrock\" or \"litellm\".")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// Validate mode-specific configuration exists
	if mode == "bedrock" {
		if globalConfig.Bedrock == nil {
			fmt.Fprintln(os.Stderr, "Error: Bedrock mode selected but bedrock section not configured.")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Add a \"bedrock\" section to ~/.gantryrc.json with awsProfile, awsRegion, and model.")
			fmt.Fprintln(os.Stderr, "")
			os.Exit(1)
		}
		if globalConfig.Bedrock.AWSProfile == "" {
			fmt.Fprintln(os.Stderr, "Error: bedrock.awsProfile is required when using Bedrock mode.")
			fmt.Fprintln(os.Stderr, "")
			os.Exit(1)
		}
		if globalConfig.Bedrock.AWSRegion == "" {
			fmt.Fprintln(os.Stderr, "Error: bedrock.awsRegion is required when using Bedrock mode.")
			fmt.Fprintln(os.Stderr, "")
			os.Exit(1)
		}
	} else if mode == "litellm" {
		if globalConfig.LiteLLM == nil {
			fmt.Fprintln(os.Stderr, "Error: LiteLLM mode selected but litellm section not configured.")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Add a \"litellm\" section to ~/.gantryrc.json with baseUrl, authToken, and model.")
			fmt.Fprintln(os.Stderr, "")
			os.Exit(1)
		}
		if globalConfig.LiteLLM.BaseURL == "" {
			fmt.Fprintln(os.Stderr, "Error: litellm.baseUrl is required when using LiteLLM mode.")
			fmt.Fprintln(os.Stderr, "")
			os.Exit(1)
		}
		if globalConfig.LiteLLM.AuthToken == "" {
			fmt.Fprintln(os.Stderr, "Error: litellm.authToken is required when using LiteLLM mode.")
			fmt.Fprintln(os.Stderr, "")
			os.Exit(1)
		}
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

	// Check for updates if enabled (defaults to true) and 6+ hours since last check
	var updateAvailable bool
	var latestVersion string
	if shouldCheckForUpdate(globalConfig.Gantry) {
		result := updater.CheckForUpdate(Version)
		if result.Error == nil {
			updateAvailable = result.UpdateAvailable
			latestVersion = result.LatestVersion
			// Save the check timestamp and result to config
			saveUpdateCheckResult(globalConfig, latestVersion)
		}
	} else {
		// Use cached result if available
		if globalConfig.Gantry != nil && globalConfig.Gantry.LastUpdateResult != "" {
			latestVersion = globalConfig.Gantry.LastUpdateResult
			if updater.CompareVersions(Version, latestVersion) < 0 {
				updateAvailable = true
			}
		}
	}

	// ============================================
	// PHASE 2: Show confirmation screen (if enabled)
	// ============================================

	if !bypassLoadingScreen {
		fmt.Println()
		fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════╗")
		fmt.Println("║                    GANTRY - Claude Code Launcher                                       ║")
		fmt.Printf("║                    Version: %-59s ║\n", Version)
		// Show update notification if available
		if updateAvailable {
			fmt.Println("║                                                                                        ║")
			fmt.Println("║                    *** UPDATE AVAILABLE ***                                            ║")
			fmt.Printf("║                    Version %-61s ║\n", truncateString(latestVersion+" is available", 61))
			fmt.Println("║                    Run 'gantry update' to install                                      ║")
		}
		fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════╣")
		fmt.Println("║  The following configuration will be applied:                                          ║")
		fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════╣")

		// User info
		fmt.Println("║  USER IDENTITY                                                                         ║")
		if usernameSource == "env" {
			fmt.Printf("║    Username:        %-64s ║\n", truncateString(username+" (env override)", 64))
		} else {
			fmt.Printf("║    Username:        %-64s ║\n", truncateString(username, 64))
		}
		fmt.Println("║                                                                                        ║")

		// Project info
		fmt.Println("║  PROJECT                                                                               ║")
		fmt.Printf("║    Project Name:    %-64s ║\n", truncateString(projectConfig.Config.ProjectName, 64))
		if projectConfig.Path != "" {
			fmt.Printf("║    Config File:     %-64s ║\n", truncateString(projectConfig.Path, 64))
		} else {
			fmt.Printf("║    Config File:     %-64s ║\n", "(none - using defaults)")
		}
		if gitBranch != "" {
			fmt.Printf("║    Git Branch:      %-64s ║\n", truncateString(gitBranch, 64))
		}
		fmt.Println("║                                                                                        ║")

		// OTEL info
		fmt.Println("║  TELEMETRY (OTEL)                                                                      ║")
		fmt.Printf("║    Endpoint:        %-64s ║\n", truncateString(globalConfig.OTEL.Endpoint, 64))
		fmt.Println("║                                                                                        ║")

		// Provider info (Bedrock or LiteLLM based on mode)
		if mode == "bedrock" {
			fmt.Println("║  AWS BEDROCK                                                                           ║")
			if modeSource == "flag" {
				fmt.Printf("║    Mode:            %-64s ║\n", "bedrock (from --mode flag)")
			} else {
				fmt.Printf("║    Mode:            %-64s ║\n", "bedrock (from config)")
			}
			fmt.Printf("║    AWS Profile:     %-64s ║\n", truncateString(globalConfig.Bedrock.AWSProfile, 64))
			fmt.Printf("║    Region:          %-64s ║\n", globalConfig.Bedrock.AWSRegion)
			fmt.Printf("║    Model:           %-64s ║\n", truncateString(globalConfig.Bedrock.Model, 64))
		} else {
			fmt.Println("║  LITELLM                                                                               ║")
			if modeSource == "flag" {
				fmt.Printf("║    Mode:            %-64s ║\n", "litellm (from --mode flag)")
			} else {
				fmt.Printf("║    Mode:            %-64s ║\n", "litellm (from config)")
			}
			fmt.Printf("║    Base URL:        %-64s ║\n", truncateString(globalConfig.LiteLLM.BaseURL, 64))
			fmt.Printf("║    Model:           %-64s ║\n", truncateString(globalConfig.LiteLLM.Model, 64))
		}
		fmt.Println("║                                                                                        ║")

		// Powerline info
		fmt.Println("║  POWERLINE STATUS BAR                                                                  ║")
		fmt.Printf("║    Action:          %-64s ║\n", truncateString(powerlineAction, 64))
		fmt.Println("║                                                                                        ║")

		fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════╣")
		fmt.Println("║  Press ENTER to continue, or 'q' to cancel...                                          ║")
		fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════╝")

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

	// Build environment based on mode
	env := launcher.BuildEnvironment(globalConfig, resourceAttributes, mode)

	fmt.Println()
	fmt.Println("Starting Claude Code with GANTRY configuration...")
	fmt.Println()

	// Launch Claude Code with filtered arguments (mode flag removed)
	if err := launcher.LaunchClaude(filteredArgs, env); err != nil {
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
  gantry [options] [claude-code-args...]    Run Claude Code with GANTRY configuration
  gantry init [--force]                     Create the global configuration file
  gantry config                             Interactive configuration editor
  gantry config show                        Display current configuration
  gantry config get <key>                   Get a configuration value
  gantry config set <key> <val>             Set a configuration value
  gantry update                             Update gantry to the latest version
  gantry update --check                     Check if an update is available
  gantry version                            Show version information
  gantry version --check                    Show version and check for updates
  gantry --help                             Show this help message
  gantry --version                          Show version number

Options:
  --mode, -m <mode>               Set the provider mode (bedrock or litellm)
                                  Overrides gantry.mode in config file

Modes:
  bedrock                         Use AWS Bedrock as the Claude API provider
                                  Requires: bedrock.awsProfile, bedrock.awsRegion
  litellm                         Use LiteLLM proxy as the Claude API provider
                                  Requires: litellm.baseUrl, litellm.authToken

Environment Variables:
  GANTRY_USERNAME                 Optional. Overrides gantry.username from config.

Configuration Files:
  ~/.gantryrc.json                Global configuration (created by 'gantry init')
  .gantry.json                    Per-project configuration (optional)

Examples:
  gantry                          Start Claude Code (uses mode from config)
  gantry --mode bedrock           Start Claude Code with AWS Bedrock
  gantry --mode litellm           Start Claude Code with LiteLLM proxy
  gantry -m bedrock               Short form of --mode
  gantry --help                   Show GANTRY help
  gantry init                     Initialize global configuration
  gantry init --force             Reinitialize global configuration
  gantry config                   Edit configuration interactively
  gantry config show              View current configuration
  gantry config set gantry.mode bedrock
  gantry config set gantry.username john.doe
  gantry config set bedrock.awsProfile my-profile
  gantry config set litellm.baseUrl https://litellm.example.com
  gantry update                   Update to latest version

For more information, see: https://github.com/mattabdou/gantry
`, Version)
}

// shouldCheckForUpdate determines if we should perform an update check
// Returns true if checkForUpdateOnLaunch is enabled (or not set, defaults to true)
// AND more than 6 hours have passed since the last check
func shouldCheckForUpdate(gantryConfig *config.GantryConfig) bool {
	if gantryConfig == nil {
		return true // Default to checking
	}

	// Check if feature is disabled
	if gantryConfig.CheckForUpdateOnLaunch != nil && !*gantryConfig.CheckForUpdateOnLaunch {
		return false
	}

	// Check if 6 hours have passed since last check
	if gantryConfig.LastUpdateCheck != "" {
		lastCheck, err := time.Parse(time.RFC3339, gantryConfig.LastUpdateCheck)
		if err == nil && time.Since(lastCheck) < 6*time.Hour {
			return false // Use cached result
		}
	}

	return true
}

// saveUpdateCheckResult saves the update check timestamp and result to the config file
func saveUpdateCheckResult(globalConfig *config.GlobalConfig, latestVersion string) {
	if globalConfig.Gantry == nil {
		globalConfig.Gantry = &config.GantryConfig{}
	}

	globalConfig.Gantry.LastUpdateCheck = time.Now().Format(time.RFC3339)
	globalConfig.Gantry.LastUpdateResult = latestVersion

	// Save to config file (ignore errors - this is non-critical)
	_ = config.SaveGlobalConfig(globalConfig)
}
