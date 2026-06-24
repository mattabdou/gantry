package cmd

import (
	"fmt"
	"runtime"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/updater"
	"github.com/spf13/cobra"
)

// Version is the current version of GANTRY
const Version = "1.2.1-beta.7"

var versionCheckUpdate bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the current version of gantry and optionally check for updates.`,
	Run: func(cmd *cobra.Command, args []string) {
		showVersion()
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&versionCheckUpdate, "check", "c", false, "Check if a newer version is available")
	rootCmd.AddCommand(versionCmd)
}

func showVersion() {
	// Get current channel from config
	channel := "stable"
	globalConfig, err := config.LoadGlobalConfigRaw()
	if err == nil {
		channel = config.GetReleaseChannel(globalConfig)
	}

	fmt.Printf("gantry v%s (%s channel)\n", Version, channel)
	fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	if versionCheckUpdate {
		fmt.Println()
		fmt.Printf("Checking for updates (%s channel)...\n", channel)

		result := updater.CheckForUpdate(Version, channel)
		if result.Error != nil {
			fmt.Printf("  Could not check for updates: %v\n", result.Error)
			return
		}

		if result.UpdateAvailable {
			fmt.Printf("  New version available: v%s\n", result.LatestVersion)
			fmt.Println("  Run 'gantry update' to install it.")
		} else {
			fmt.Println("  You are running the latest version.")
		}
	}
}
