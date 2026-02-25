package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/spf13/cobra"
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Show AI API spend from LiteLLM",
	Long:  `Query the LiteLLM proxy to show total API spend for your key. Requires LiteLLM configuration in ~/.gantryrc.json.`,
	Run: func(cmd *cobra.Command, args []string) {
		showCost()
	},
}

func init() {
	rootCmd.AddCommand(costCmd)
}

// keyInfoResponse represents the LiteLLM /key/info response
type keyInfoResponse struct {
	Key  string  `json:"key"`
	Info keyInfo `json:"info"`
}

type keyInfo struct {
	Spend float64 `json:"spend"`
}

func showCost() {
	// Load config (use raw to skip validation since we only need LiteLLM section)
	globalConfig, err := config.LoadGlobalConfigRaw()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Check if LiteLLM is configured
	if globalConfig.LiteLLM == nil || globalConfig.LiteLLM.BaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: gantry cost requires LiteLLM mode.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "This feature queries the LiteLLM proxy for spend data.")
		fmt.Fprintln(os.Stderr, "Ensure your ~/.gantryrc.json has a \"litellm\" section with baseUrl and authToken.")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	if globalConfig.LiteLLM.AuthToken == "" {
		fmt.Fprintln(os.Stderr, "Error: litellm.authToken is required for spend tracking.")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// Build the request URL
	baseURL := strings.TrimSuffix(globalConfig.LiteLLM.BaseURL, "/")
	url := baseURL + "/key/info?key=" + globalConfig.LiteLLM.AuthToken

	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Authorization", "Bearer "+globalConfig.LiteLLM.AuthToken)

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying LiteLLM: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	// Check for non-200 status
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: LiteLLM returned status %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	// Parse the response
	var keyResp keyInfoResponse
	if err := json.Unmarshal(body, &keyResp); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Total spend: $%.2f\n", keyResp.Info.Spend)
}
