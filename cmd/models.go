package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models from LiteLLM",
	Long:  `Query the LiteLLM proxy to list all available models. Requires LiteLLM configuration in ~/.gantryrc.json.`,
	Run: func(cmd *cobra.Command, args []string) {
		listModels()
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}

// modelsResponse represents the OpenAI-compatible /v1/models response
type modelsResponse struct {
	Data []modelInfo `json:"data"`
}

type modelInfo struct {
	ID string `json:"id"`
}

// modelsBoxLine formats a line for the models output box
func modelsBoxLine(prefix, value string) string {
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

func listModels() {
	// Load config (use raw to skip validation since we only need LiteLLM section)
	globalConfig, err := config.LoadGlobalConfigRaw()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Check if LiteLLM is configured
	if globalConfig.LiteLLM == nil || globalConfig.LiteLLM.BaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: feature only available with LiteLLM")
		os.Exit(1)
	}

	// Build the request URL
	baseURL := strings.TrimSuffix(globalConfig.LiteLLM.BaseURL, "/")
	url := baseURL + "/v1/models"

	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}

	// Add authorization header if auth token is configured
	if globalConfig.LiteLLM.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+globalConfig.LiteLLM.AuthToken)
	}

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying LiteLLM: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Check for non-200 status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: LiteLLM returned status %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	// Parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	var models modelsResponse
	if err := json.Unmarshal(body, &models); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	// Print header
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    gantry - Claude Code Launcher                                       ║")
	fmt.Println(modelsBoxLine("║                    Version: ", Version))
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  The following models are available to use through LiteLLM:                            ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════════════════════════════╣")

	// Print each model ID
	for _, model := range models.Data {
		fmt.Println(modelsBoxLine("║    ", model.ID))
	}

	// Print footer
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════════════╝")
}
