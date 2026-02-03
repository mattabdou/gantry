package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/updater"
	"github.com/spf13/cobra"
)

var (
	checkOnly    bool
	switchToBeta bool
	switchToStable bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update gantry to the latest version",
	Long: `Check for and install the latest version of gantry.

By default, gantry uses the stable release channel. You can switch to the beta
channel to get early access to new features.

Examples:
  gantry update            Download and install the latest version (uses saved channel preference)
  gantry update --check    Check if an update is available without installing
  gantry update --beta     Switch to beta channel and update to latest beta release
  gantry update --stable   Switch to stable channel and update to latest stable release`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate flag combinations
		if switchToBeta && switchToStable {
			fmt.Fprintln(os.Stderr, "Error: cannot specify both --beta and --stable")
			os.Exit(1)
		}

		if checkOnly && (switchToBeta || switchToStable) {
			fmt.Fprintln(os.Stderr, "Error: cannot use --beta or --stable with --check")
			fmt.Fprintln(os.Stderr, "The --check flag uses your saved channel preference.")
			fmt.Fprintln(os.Stderr, "To change channels, run 'gantry update --beta' or 'gantry update --stable' without --check.")
			os.Exit(1)
		}

		if checkOnly {
			runCheckUpdate()
		} else {
			runUpdate()
		}
	},
}

func init() {
	updateCmd.Flags().BoolVarP(&checkOnly, "check", "c", false, "Check for updates without installing")
	updateCmd.Flags().BoolVar(&switchToBeta, "beta", false, "Switch to beta release channel")
	updateCmd.Flags().BoolVar(&switchToStable, "stable", false, "Switch to stable release channel")
	rootCmd.AddCommand(updateCmd)
}

func runCheckUpdate() {
	// Get current channel from config
	globalConfig, err := config.LoadGlobalConfigRaw()
	if err != nil {
		// Config might not exist or be invalid, default to stable
		globalConfig = &config.GlobalConfig{}
	}

	channel := config.GetReleaseChannel(globalConfig)
	result := updater.CheckForUpdate(Version, channel)

	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", result.Error)
		os.Exit(1)
	}

	fmt.Printf("Current version: v%s (%s channel)\n", result.CurrentVersion, channel)
	fmt.Printf("Latest version:  v%s\n", result.LatestVersion)

	if result.UpdateAvailable {
		fmt.Println()
		fmt.Println("A new version is available!")
		fmt.Println("Run 'gantry update' to install it.")
		fmt.Printf("Release notes: %s\n", result.ReleaseURL)
	} else {
		fmt.Println()
		fmt.Println("You are running the latest version.")
	}
}

func runUpdate() {
	// Load current config
	globalConfig, err := config.LoadGlobalConfigRaw()
	if err != nil {
		// Config might not exist or be invalid, create a minimal one
		globalConfig = &config.GlobalConfig{
			Gantry: &config.GantryConfig{},
		}
	}

	currentChannel := config.GetReleaseChannel(globalConfig)
	targetChannel := currentChannel

	// Determine target channel based on flags
	if switchToBeta {
		targetChannel = "beta"
	} else if switchToStable {
		targetChannel = "stable"
	}

	// Handle channel switching
	if targetChannel != currentChannel {
		if err := handleChannelSwitch(globalConfig, currentChannel, targetChannel); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Perform the update
	if err := updater.Update(Version, targetChannel); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Save channel preference after successful update
	if globalConfig.Gantry == nil {
		globalConfig.Gantry = &config.GantryConfig{}
	}
	if globalConfig.Gantry.Release != targetChannel {
		if err := config.SetReleaseChannel(globalConfig, targetChannel); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save channel preference: %v\n", err)
		}
	}
}

func handleChannelSwitch(globalConfig *config.GlobalConfig, fromChannel, toChannel string) error {
	// Switching from stable to beta
	if fromChannel == "stable" && toChannel == "beta" {
		fmt.Println("Switching to beta release channel...")
		fmt.Println()
		fmt.Println("Note: Beta releases may include experimental features and could")
		fmt.Println("potentially introduce changes to your configuration format.")
		fmt.Println()

		// Create backup of stable config
		backupPath, err := config.BackupConfigForChannelSwitch(fromChannel, toChannel)
		if err != nil {
			return fmt.Errorf("failed to backup config: %w", err)
		}
		if backupPath != "" {
			fmt.Printf("Created backup: %s\n", backupPath)
			fmt.Println("You can restore this manually if needed.")
			fmt.Println()
		}

		return nil
	}

	// Switching from beta to stable
	if fromChannel == "beta" && toChannel == "stable" {
		fmt.Println("Switching to stable release channel...")
		fmt.Println()
		fmt.Println("WARNING: Downgrading from beta to stable may cause issues with")
		fmt.Println("your .gantryrc.json configuration file if the beta version")
		fmt.Println("introduced new configuration options.")
		fmt.Println()

		// Create backup of beta config
		backupPath, err := config.BackupConfigForChannelSwitch(fromChannel, toChannel)
		if err != nil {
			return fmt.Errorf("failed to backup config: %w", err)
		}
		if backupPath != "" {
			fmt.Printf("Created backup: %s\n", backupPath)
		}

		// Ask for confirmation
		fmt.Print("Do you want to proceed? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("update cancelled")
		}

		fmt.Println()
		return nil
	}

	return nil
}
