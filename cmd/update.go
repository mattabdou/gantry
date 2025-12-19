package cmd

import (
	"fmt"
	"os"

	"github.com/mattabdou/gantry/internal/updater"
	"github.com/spf13/cobra"
)

var checkOnly bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update gantry to the latest version",
	Long: `Check for and install the latest version of gantry.

Examples:
  gantry update          Download and install the latest version
  gantry update --check  Check if an update is available without installing`,
	Run: func(cmd *cobra.Command, args []string) {
		if checkOnly {
			runCheckUpdate()
		} else {
			runUpdate()
		}
	},
}

func init() {
	updateCmd.Flags().BoolVarP(&checkOnly, "check", "c", false, "Check for updates without installing")
	rootCmd.AddCommand(updateCmd)
}

func runCheckUpdate() {
	result := updater.CheckForUpdate(Version)

	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", result.Error)
		os.Exit(1)
	}

	fmt.Printf("Current version: v%s\n", result.CurrentVersion)
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
	if err := updater.Update(Version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
