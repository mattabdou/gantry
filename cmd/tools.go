package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List supported AI tools",
	Long:  `Display all AI tools that gantry can launch and configure, with usage instructions.`,
	Run: func(cmd *cobra.Command, args []string) {
		listTools()
	},
}

func init() {
	rootCmd.AddCommand(toolsCmd)
}

type toolInfo struct {
	shortName   string
	displayName string
	description string
}

func listTools() {
	tools := []toolInfo{
		{"cc", "Claude Code", "Terminal AI coding agent by Anthropic"},
		{"oc", "OpenCode Terminal", "Terminal-based AI coding agent"},
		{"ocd", "OpenCode Desktop", "Desktop AI coding application"},
		{"cl", "Cline", "AI coding agent CLI by Cline"},
		{"clk", "Cline Kanban", "Local web board for parallel coding agents"},
		{"clp", "Cline Plugin", "Configure the Cline VS Code extension"},
		{"co", "Codex", "Terminal AI coding agent by OpenAI"},
	}

	fmt.Println()
	fmt.Println("  Supported AI Tools")
	fmt.Println("  ──────────────────")
	fmt.Println()
	fmt.Println("  Launch a tool:        gantry --tool <name>")
	fmt.Println("  Set as default:       gantry config set gantry.defaultTool <name>")
	fmt.Println()
	fmt.Println("  ┌──────┬─────────────────────┬──────────────────────────────────────────────────┐")
	fmt.Println("  │ Name │ Tool                │ Description                                      │")
	fmt.Println("  ├──────┼─────────────────────┼──────────────────────────────────────────────────┤")

	for _, t := range tools {
		fmt.Printf("  │ %-4s │ %-19s │ %-48s │\n", t.shortName, t.displayName, t.description)
	}

	fmt.Println("  └──────┴─────────────────────┴──────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("  Notes:")
	fmt.Println("    • cl, clk, clp, and co require LiteLLM mode (--mode litellm)")
	fmt.Println("    • clp is configure-only (does not launch a tool)")
	fmt.Println("    • Run 'gantry models' to see available models on your LiteLLM proxy")
	fmt.Println()
}
