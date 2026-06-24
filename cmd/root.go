package cmd

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mattabdou/gantry/internal/cline"
	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/launcher"
	"github.com/mattabdou/gantry/internal/opencode"
	"github.com/mattabdou/gantry/internal/powerline"
	"github.com/mattabdou/gantry/internal/updater"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gantry [tool-args...]",
	Short: "GANTRY - Gateway for AI Navigation, Telemetry, and Runtime Yield",
	Long: `GANTRY is a launcher for AI coding tools (Claude Code, OpenCode) that configures
environment, telemetry, and provider settings.

It supports AWS Bedrock and LiteLLM as providers, enriches OpenTelemetry telemetry
with user, project, and organizational attributes for AI cost tracking, and
configures tool-specific settings.`,
	Version: Version,
	// Allow unknown flags to be passed through to the AI tool
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

// parseToolFlag extracts --tool or -t flag from args and returns the tool value and filtered args
func parseToolFlag(args []string) (tool string, filteredArgs []string) {
	filteredArgs = make([]string, 0, len(args))
	skipNext := false

	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		// Handle --tool=value or -t=value
		if strings.HasPrefix(arg, "--tool=") {
			tool = strings.TrimPrefix(arg, "--tool=")
			continue
		}
		if strings.HasPrefix(arg, "-t=") {
			tool = strings.TrimPrefix(arg, "-t=")
			continue
		}

		// Handle --tool value or -t value
		if arg == "--tool" || arg == "-t" {
			if i+1 < len(args) {
				tool = args[i+1]
				skipNext = true
			}
			continue
		}

		filteredArgs = append(filteredArgs, arg)
	}

	return tool, filteredArgs
}

// parseShellFlag extracts --shell or -s flag from args and returns whether shell mode is enabled
func parseShellFlag(args []string) (shellMode bool, filteredArgs []string) {
	filteredArgs = make([]string, 0, len(args))

	for _, arg := range args {
		if arg == "--shell" || arg == "-s" {
			shellMode = true
			continue
		}
		filteredArgs = append(filteredArgs, arg)
	}

	return shellMode, filteredArgs
}

// parseResetConfigFlag extracts --resetconfig or -r flag from args
func parseResetConfigFlag(args []string) (resetConfig bool, filteredArgs []string) {
	filteredArgs = make([]string, 0, len(args))

	for _, arg := range args {
		if arg == "--resetconfig" || arg == "-r" {
			resetConfig = true
			continue
		}
		filteredArgs = append(filteredArgs, arg)
	}

	return resetConfig, filteredArgs
}

func runGantry(cmd *cobra.Command, args []string) {
	// Parse --mode, --tool, --shell, and --resetconfig flags from args before other processing
	modeFlag, filteredArgs := parseModeFlag(args)
	toolFlag, filteredArgs := parseToolFlag(filteredArgs)
	shellMode, filteredArgs := parseShellFlag(filteredArgs)
	resetConfig, filteredArgs := parseResetConfigFlag(filteredArgs)

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
		if arg == "init" || arg == "config" || arg == "update" || arg == "version" || arg == "models" || arg == "cost" {
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

	// Cline requires LiteLLM mode
	if (toolFlag == "cl" || toolFlag == "clk") && mode != "litellm" {
		fmt.Fprintln(os.Stderr, "Error: Cline requires LiteLLM mode.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Use --mode litellm or set gantry.mode to \"litellm\" in ~/.gantryrc.json.")
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

	// Determine effective tool: command-line flag overrides config
	tool := toolFlag
	toolSource := "flag"
	if tool == "" {
		if globalConfig.Gantry != nil && globalConfig.Gantry.DefaultTool != "" {
			tool = globalConfig.Gantry.DefaultTool
			toolSource = "config"
		} else {
			tool = "cc" // Default to Claude Code
			toolSource = "default"
		}
	}

	// Validate tool value
	if !config.IsValidTool(tool) {
		fmt.Fprintf(os.Stderr, "Error: invalid tool %q.\n", tool)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Tool must be one of: cc (Claude Code), oc (OpenCode Terminal), ocd (OpenCode Desktop), cl (Cline), clk (Cline Kanban).")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// Check tool installation
	switch tool {
	case "cc":
		result := launcher.DetectClaudeCode()
		if !result.Installed {
			if shellMode {
				fmt.Fprintln(os.Stderr, "Warning: Claude Code installation not detected. Shell mode will continue.")
			} else {
				fmt.Fprintln(os.Stderr, "Error: Claude Code installation not detected.")
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, "Please install Claude Code: npm install -g @anthropic-ai/claude-code")
				fmt.Fprintln(os.Stderr, "")
				os.Exit(1)
			}
		}
	case "oc":
		result := launcher.DetectOpenCodeTerminal()
		if !result.Installed {
			if shellMode {
				fmt.Fprintln(os.Stderr, "Warning: OpenCode Terminal installation not detected. Shell mode will continue.")
			} else {
				fmt.Fprintln(os.Stderr, "OpenCode Terminal installation not detected.")
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, "Please visit https://opencode.ai/download to get started with installation.")
				fmt.Fprintln(os.Stderr, "")
				os.Exit(1)
			}
		}
	case "ocd":
		result := launcher.DetectOpenCodeDesktop()
		if !result.Installed {
			if shellMode {
				fmt.Fprintln(os.Stderr, "Warning: OpenCode Desktop installation not detected. Shell mode will continue.")
			} else {
				fmt.Fprintln(os.Stderr, "OpenCode Desktop installation not detected.")
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, "Please visit https://opencode.ai/download to get started with installation.")
				fmt.Fprintln(os.Stderr, "")
				os.Exit(1)
			}
		}
	case "cl", "clk":
		result := launcher.DetectCline()
		if !result.Installed {
			if shellMode {
				fmt.Fprintln(os.Stderr, "Warning: Cline installation not detected. Shell mode will continue.")
			} else {
				fmt.Fprintln(os.Stderr, "Error: Cline installation not detected.")
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, "Please install Cline: npm install -g cline")
				fmt.Fprintln(os.Stderr, "")
				os.Exit(1)
			}
		}
	}

	// Handle --resetconfig flag
	if resetConfig {
		// Determine tool display name for confirmation message
		var resetToolName string
		var resetFilePath string
		switch tool {
		case "cc":
			resetToolName = "Claude Code"
			settingsPath, _ := powerline.GetClaudeSettingsPath()
			resetFilePath = settingsPath
		case "oc", "ocd":
			resetToolName = "OpenCode"
			configPath, _ := opencode.GetConfigPath()
			resetFilePath = configPath
		}

		fmt.Printf("This will revert your %s configuration (%s) to defaults. Proceed? (y/n): ", resetToolName, resetFilePath)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input != "y" {
			fmt.Println("Cancelled.")
			return
		}

		switch tool {
		case "cc":
			result, err := powerline.ResetClaudeSettings()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(result.Message)
		case "oc", "ocd":
			result, err := opencode.ResetConfig(globalConfig.LiteLLM, globalConfig.Bedrock, mode)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(result.Message)
		}
		return
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
		projectConfig = config.GetDefaultProjectConfig(workingPath)
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
	releaseChannel := config.GetReleaseChannel(globalConfig)
	if shouldCheckForUpdate(globalConfig.Gantry) {
		result := updater.CheckForUpdate(Version, releaseChannel)
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

	// Auto-update if enabled and update is available
	if updateAvailable && os.Getenv("GANTRY_AUTO_UPDATED") == "" {
		autoUpdate := true // default
		if globalConfig.Gantry != nil && globalConfig.Gantry.AutoUpdate != nil {
			autoUpdate = *globalConfig.Gantry.AutoUpdate
		}
		if autoUpdate {
			fmt.Printf("Updating gantry to v%s...\n", latestVersion)
			release, err := updater.GetLatestReleaseForChannel(releaseChannel)
			if err == nil {
				tmpPath, err := updater.DownloadRelease(release)
				if err == nil {
					if err := updater.ReplaceExecutable(tmpPath); err == nil {
						fmt.Printf("Update complete, restarting...\n\n")
						exe, err := updater.GetExecutablePath()
						if err == nil {
							os.Setenv("GANTRY_AUTO_UPDATED", "1")
							if runtime.GOOS == "windows" {
								// Windows: spawn new process and exit
								proc, err := os.StartProcess(exe, append([]string{exe}, os.Args[1:]...), &os.ProcAttr{
									Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
									Env:   os.Environ(),
								})
								if err == nil {
									proc.Release()
									os.Exit(0)
								}
								fmt.Fprintf(os.Stderr, "Warning: failed to restart after update: %v\n", err)
							} else {
								// Unix: replace current process
								err := syscall.Exec(exe, append([]string{exe}, os.Args[1:]...), os.Environ())
								// If we get here, exec failed
								fmt.Fprintf(os.Stderr, "Warning: failed to restart after update: %v\n", err)
							}
						} else {
							fmt.Fprintf(os.Stderr, "Warning: updated but failed to restart: %v\n", err)
						}
					} else {
						fmt.Fprintf(os.Stderr, "Warning: auto-update failed to install: %v\n", err)
					}
				} else {
					fmt.Fprintf(os.Stderr, "Warning: auto-update failed to download: %v\n", err)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Warning: auto-update failed to fetch release: %v\n", err)
			}
			// Continue normally if auto-update failed
		}
	}

	// ============================================
	// PHASE 2: Show confirmation screen (if enabled)
	// ============================================

	// Determine tool display name
	var toolDisplayName string
	switch tool {
	case "cc":
		toolDisplayName = "Claude Code"
	case "oc":
		toolDisplayName = "OpenCode Terminal"
	case "ocd":
		toolDisplayName = "OpenCode Desktop"
	case "cl":
		toolDisplayName = "Cline"
	case "clk":
		toolDisplayName = "Cline Kanban"
	}

	if !bypassLoadingScreen {
		fmt.Println()
		fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════╗")
		fmt.Println("║                    gantry - AI Tool Launcher                                           ║")
		fmt.Println(boxLine("║                    Version: ", Version))
		// Show update notification if available
		if updateAvailable {
			fmt.Println("║                                                                                        ║")
			fmt.Println("║                    *** UPDATE AVAILABLE ***                                            ║")
			fmt.Println(boxLine("║                    Version ", latestVersion+" is available"))
			fmt.Println("║                    Run 'gantry update' to install                                      ║")
		}
		fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════╣")
		fmt.Println("║  The following configuration will be applied:                                          ║")
		fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════╣")

		// Tool info
		fmt.Println("║  TOOL                                                                                  ║")
		if shellMode {
			fmt.Println(boxLine("║    Launching:       ", "Shell Mode (configured for "+toolDisplayName+")"))
		} else if toolSource == "flag" {
			fmt.Println(boxLine("║    Launching:       ", toolDisplayName+" (from --tool flag)"))
		} else if toolSource == "config" {
			fmt.Println(boxLine("║    Launching:       ", toolDisplayName+" (from config)"))
		} else {
			fmt.Println(boxLine("║    Launching:       ", toolDisplayName+" (default)"))
		}
		fmt.Println("║                                                                                        ║")

		// User info
		fmt.Println("║  USER IDENTITY                                                                         ║")
		if usernameSource == "env" {
			fmt.Println(boxLine("║    Username:        ", username+" (env override)"))
		} else {
			fmt.Println(boxLine("║    Username:        ", username))
		}
		fmt.Println("║                                                                                        ║")

		// Project info
		fmt.Println("║  PROJECT                                                                               ║")
		fmt.Println(boxLine("║    Project Name:    ", projectConfig.Config.ProjectName))
		if projectConfig.Path != "" {
			fmt.Println(boxLine("║    Config File:     ", projectConfig.Path))
		} else {
			fmt.Println(boxLine("║    Config File:     ", "(none - using defaults)"))
		}
		if gitBranch != "" {
			fmt.Println(boxLine("║    Git Branch:      ", gitBranch))
		}
		fmt.Println("║                                                                                        ║")

		// OTEL info
		fmt.Println("║  TELEMETRY (OTEL)                                                                      ║")
		fmt.Println(boxLine("║    Endpoint:        ", globalConfig.OTEL.Endpoint))
		fmt.Println("║                                                                                        ║")

		// Provider info (Bedrock or LiteLLM based on mode)
		if mode == "bedrock" {
			fmt.Println("║  AWS BEDROCK                                                                           ║")
			if modeSource == "flag" {
				fmt.Println(boxLine("║    Mode:            ", "bedrock (from --mode flag)"))
			} else {
				fmt.Println(boxLine("║    Mode:            ", "bedrock (from config)"))
			}
			fmt.Println(boxLine("║    AWS Profile:     ", globalConfig.Bedrock.AWSProfile))
			fmt.Println(boxLine("║    Region:          ", globalConfig.Bedrock.AWSRegion))
			fmt.Println(boxLine("║    Model:           ", globalConfig.Bedrock.Model))
		} else {
			fmt.Println("║  LITELLM                                                                               ║")
			if modeSource == "flag" {
				fmt.Println(boxLine("║    Mode:            ", "litellm (from --mode flag)"))
			} else {
				fmt.Println(boxLine("║    Mode:            ", "litellm (from config)"))
			}
			fmt.Println(boxLine("║    Base URL:        ", globalConfig.LiteLLM.BaseURL))
			fmt.Println(boxLine("║    Model:           ", globalConfig.LiteLLM.Model))
		}
		fmt.Println("║                                                                                        ║")

		// Powerline info (only for Claude Code)
		if tool == "cc" {
			fmt.Println("║  POWERLINE STATUS BAR                                                                  ║")
			fmt.Println(boxLine("║    Action:          ", powerlineAction))
			fmt.Println("║                                                                                        ║")
		}

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
		fmt.Printf("Launching: %s\n", toolDisplayName)
		if projectConfig.Path != "" {
			fmt.Printf("Using project config: %s\n", projectConfig.Path)
		}
		if gitBranch != "" {
			fmt.Printf("Git branch: %s\n", gitBranch)
		}
	}

	// Handle powerline configuration (only for Claude Code, skip if ignorePowerline is true)
	if tool == "cc" && !ignorePowerline {
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

	// Configure OpenCode if using OpenCode tools
	if tool == "oc" || tool == "ocd" {
		var configResult *opencode.ConfigureResult
		var err error

		if mode == "litellm" {
			configResult, err = opencode.ConfigureLiteLLM(globalConfig.LiteLLM)
		} else if mode == "bedrock" {
			configResult, err = opencode.ConfigureBedrock(globalConfig.Bedrock)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error configuring OpenCode: %v\n", err)
			os.Exit(1)
		}

		if configResult != nil && configResult.Updated {
			fmt.Printf("OpenCode: %s\n", configResult.Message)
		}
	}

	// Configure Cline if using Cline tools
	if tool == "cl" || tool == "clk" {
		if err := cline.ConfigureProvider(globalConfig.LiteLLM); err != nil {
			fmt.Fprintf(os.Stderr, "Error configuring Cline: %v\n", err)
			os.Exit(1)
		}
	}

	// Build resource attributes (for Claude Code telemetry)
	resourceAttributes := launcher.BuildResourceAttributes(username, workingPath, projectConfig, gitBranch, Version)

	// Build environment based on mode (for Claude Code)
	env := launcher.BuildEnvironment(globalConfig, resourceAttributes, mode)

	if shellMode {
		fmt.Println()
		fmt.Printf("Starting shell with GANTRY configuration for %s...\n", toolDisplayName)
		fmt.Println()

		// Determine the tool command for the welcome message
		var toolCommand string
		switch tool {
		case "cc":
			toolCommand = launcher.GetClaudeCommand()
		case "oc":
			toolCommand = "opencode"
		case "ocd":
			command, _ := launcher.GetOpenCodeDesktopLaunchCommand()
			toolCommand = command
		case "cl":
			toolCommand = "cline"
		case "clk":
			toolCommand = "cline kanban"
		}

		if err := launcher.LaunchShell(env, toolDisplayName, toolCommand); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting shell: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Println()
	fmt.Printf("Starting %s with GANTRY configuration...\n", toolDisplayName)
	fmt.Println()

	// Launch the selected tool
	switch tool {
	case "cc":
		// Launch Claude Code with filtered arguments (mode and tool flags removed)
		if err := launcher.LaunchClaude(filteredArgs, env); err != nil {
			if err.Error() == "executable file not found in $PATH" || err.Error() == "executable file not found in %PATH%" {
				fmt.Fprintln(os.Stderr, "Error: Claude Code is not installed or not in PATH.")
				fmt.Fprintln(os.Stderr, "Please install Claude Code: npm install -g @anthropic-ai/claude-code")
			} else {
				fmt.Fprintf(os.Stderr, "Error starting Claude Code: %v\n", err)
			}
			os.Exit(1)
		}
	case "oc":
		// Launch OpenCode Terminal
		if err := launcher.LaunchOpenCodeTerminal(filteredArgs, env); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting OpenCode Terminal: %v\n", err)
			os.Exit(1)
		}
	case "ocd":
		// Launch OpenCode Desktop
		if err := launcher.LaunchOpenCodeDesktop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting OpenCode Desktop: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("OpenCode Desktop launched.")
	case "cl":
		// Launch Cline CLI
		if err := launcher.LaunchCline(filteredArgs, env); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting Cline: %v\n", err)
			os.Exit(1)
		}
	case "clk":
		// Launch Cline Kanban
		if err := launcher.LaunchClineKanban(env); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting Cline Kanban: %v\n", err)
			os.Exit(1)
		}
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

// boxLine formats a line for the confirmation screen box with consistent width (90 chars total)
// prefix is the text before the value (e.g., "║    Username:        ")
// value is the dynamic content to display
// The function pads or truncates the value to ensure the line is exactly 90 characters
func boxLine(prefix, value string) string {
	const totalWidth = 90
	const suffix = " ║"

	// Calculate available space for value
	prefixLen := len([]rune(prefix))
	suffixLen := len([]rune(suffix))
	valueWidth := totalWidth - prefixLen - suffixLen

	// Truncate value if too long
	valueRunes := []rune(value)
	if len(valueRunes) > valueWidth {
		if valueWidth > 3 {
			value = string(valueRunes[:valueWidth-3]) + "..."
		} else {
			value = string(valueRunes[:valueWidth])
		}
	}

	// Pad value if too short
	return fmt.Sprintf("%s%-*s%s", prefix, valueWidth, value, suffix)
}

func showHelp() {
	fmt.Printf(`
GANTRY - Gateway for AI Navigation, Telemetry, and Runtime Yield v%s

A launcher for AI coding tools (Claude Code, OpenCode) that configures
environment, telemetry, and provider settings.

Usage:
  gantry [options] [tool-args...]           Run AI tool with GANTRY configuration
  gantry --shell [options]                  Start a configured shell session
  gantry --resetconfig [--tool <tool>]      Reset tool configuration to defaults
  gantry init [--force]                     Create the global configuration file
  gantry config                             Interactive configuration editor
  gantry config show                        Display current configuration
  gantry config get <key>                   Get a configuration value
  gantry config set <key> <val>             Set a configuration value
  gantry models                             List available models from LiteLLM
  gantry cost [numberOfDays]                Show AI API spend from telemetry data
  gantry update                             Update gantry to the latest version
  gantry update --check                     Check if an update is available
  gantry version                            Show version information
  gantry version --check                    Show version and check for updates
  gantry --help                             Show this help message
  gantry --version                          Show version number

Options:
  --tool, -t <tool>               Set the AI tool to launch (or configure)
                                  Overrides gantry.defaultTool in config file
                                  Values: cc (Claude Code), oc (OpenCode Terminal),
                                          ocd (OpenCode Desktop), cl (Cline),
                                          clk (Cline Kanban)

  --mode, -m <mode>               Set the provider mode (bedrock or litellm)
                                  Overrides gantry.mode in config file

  --shell, -s                     Launch a configured shell instead of the AI tool
                                  All normal setup is performed; you run commands manually

  --resetconfig, -r               Reset the selected tool's configuration to defaults
                                  Creates a backup before resetting

Tools:
  cc                              Claude Code - Anthropic's CLI for Claude
  oc                              OpenCode Terminal - Terminal-based AI coding agent
  ocd                             OpenCode Desktop - Desktop AI coding application
  cl                              Cline - AI coding agent CLI (requires LiteLLM mode)
  clk                             Cline Kanban - Local web board for parallel agents

Modes:
  bedrock                         Use AWS Bedrock as the AI provider
                                  Requires: bedrock.awsProfile, bedrock.awsRegion
  litellm                         Use LiteLLM proxy as the AI provider
                                  Requires: litellm.baseUrl, litellm.authToken

Environment Variables:
  GANTRY_USERNAME                 Optional. Overrides gantry.username from config.
  GANTRY_SHELL                    Set to "1" inside a gantry shell session.

Configuration Files:
  ~/.gantryrc.json                Global configuration (created by 'gantry init')
  .gantry.json                    Per-project configuration (optional)

Examples:
  gantry                          Start default tool (uses config settings)
  gantry --tool cc                Start Claude Code
  gantry --tool oc                Start OpenCode Terminal
  gantry --tool ocd               Start OpenCode Desktop
  gantry --tool cl                Start Cline CLI with LiteLLM gateway
  gantry --tool clk               Start Cline Kanban board
  gantry -t oc --mode litellm     Start OpenCode Terminal with LiteLLM
  gantry --mode bedrock           Start with AWS Bedrock provider
  gantry --mode litellm           Start with LiteLLM proxy provider
  gantry --shell                  Start a configured shell (manual tool execution)
  gantry -s -t cc --mode bedrock  Shell with Bedrock Claude Code config
  gantry --resetconfig            Reset default tool's config to defaults
  gantry -r -t oc                 Reset OpenCode configuration to defaults
  gantry init                     Initialize global configuration
  gantry init --force             Reinitialize global configuration
  gantry config                   Edit configuration interactively
  gantry config show              View current configuration
  gantry config set gantry.defaultTool oc
  gantry config set gantry.mode bedrock
  gantry config set gantry.username john.doe
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
