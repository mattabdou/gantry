package cmd

import (
	"fmt"
	"runtime"

	"github.com/mattabdou/gantry/internal/updater"
	"github.com/spf13/cobra"
)

// Version is the current version of GANTRY
const Version = "1.0.0"

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
	fmt.Printf("GANTRY v%s\n", Version)
	fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	if versionCheckUpdate {
		fmt.Println()
		fmt.Println("Checking for updates...")

		result := updater.CheckForUpdate(Version)
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
