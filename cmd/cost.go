package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/spf13/cobra"
)

var costCmd = &cobra.Command{
	Use:   "cost [numberOfDays]",
	Short: "Show AI API spend from LiteLLM",
	Long:  `Query the LiteLLM proxy to show total API spend over a given number of days. Defaults to 1 day if not specified. Requires LiteLLM configuration in ~/.gantryrc.json.`,
	Run: func(cmd *cobra.Command, args []string) {
		days := 1
		if len(args) > 0 {
			parsed, err := strconv.Atoi(args[0])
			if err != nil || parsed < 1 {
				fmt.Fprintln(os.Stderr, "Error: numberOfDays must be a positive integer.")
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, "Usage: gantry cost [numberOfDays]")
				fmt.Fprintln(os.Stderr, "")
				os.Exit(1)
			}
			days = parsed
		}
		showCost(days)
	},
}

func init() {
	rootCmd.AddCommand(costCmd)
}

// dailyActivityResponse represents the LiteLLM /user/daily/activity response
type dailyActivityResponse struct {
	Results  []dailyResult `json:"results"`
	Metadata spendMetadata `json:"metadata"`
}

type dailyResult struct {
	Date    string       `json:"date"`
	Metrics spendMetrics `json:"metrics"`
}

type spendMetrics struct {
	Spend float64 `json:"spend"`
}

type spendMetadata struct {
	TotalSpend float64 `json:"total_spend"`
}

func showCost(days int) {
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

	// Calculate date range
	now := time.Now()
	endDate := now.Format("2006-01-02")
	startDate := now.AddDate(0, 0, -days).Format("2006-01-02")

	// Build the request URL
	baseURL := strings.TrimSuffix(globalConfig.LiteLLM.BaseURL, "/")
	url := fmt.Sprintf("%s/user/daily/activity?start_date=%s&end_date=%s", baseURL, startDate, endDate)

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
	var activity dailyActivityResponse
	if err := json.Unmarshal(body, &activity); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	dayWord := "day"
	if days != 1 {
		dayWord = "days"
	}

	fmt.Printf("Total spent over the past %d %s: $%.2f\n", days, dayWord, activity.Metadata.TotalSpend)
}
