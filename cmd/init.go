package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/spf13/cobra"
)

var forceInit bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create the global configuration file",
	Long:  `Initialize GANTRY by creating the global configuration file at ~/.gantryrc.json`,
	Run: func(cmd *cobra.Command, args []string) {
		runInit()
	},
}

func init() {
	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "Overwrite existing configuration")
	rootCmd.AddCommand(initCmd)
}

func runInit() {
	configPath, err := config.GetGlobalConfigPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if config.GlobalConfigExists() && !forceInit {
		fmt.Printf("Configuration file already exists at: %s\n", configPath)
		fmt.Println("Use --force to overwrite the existing configuration.")
		return
	}

	template := config.GetDefaultGlobalConfigTemplate()
	content, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create configuration: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(configPath, content, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create configuration file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("GANTRY configuration initialized successfully.")
	fmt.Println()
	fmt.Printf("Configuration file created at: %s\n", configPath)
	fmt.Println()
	fmt.Println("Before running gantry, please edit the configuration file to set:")
	fmt.Println("  - otel.endpoint: Your OTEL collector endpoint URL")
	fmt.Println("  - otel.headers: Your authentication headers (e.g., Bearer token)")
	fmt.Println()
	fmt.Println("You must also set the GANTRY_USERNAME environment variable on your system.")
	fmt.Println()
}
