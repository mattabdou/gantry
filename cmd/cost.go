package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/spf13/cobra"
)

var costCmd = &cobra.Command{
	Use:   "cost [numberOfDays]",
	Short: "Show AI API spend from telemetry data",
	Long:  `Query the telemetry database to show total API spend for your user. Requires OTEL configuration in ~/.gantryrc.json.`,
	Run: func(cmd *cobra.Command, args []string) {
		showCost(args)
	},
}

func init() {
	rootCmd.AddCommand(costCmd)
}

// lokiResponse represents the Loki instant query response
type lokiResponse struct {
	Status string   `json:"status"`
	Data   lokiData `json:"data"`
}

type lokiData struct {
	ResultType string       `json:"resultType"`
	Result     []lokiResult `json:"result"`
}

type lokiResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

func showCost(args []string) {
	// Parse numberOfDays argument (default: 1)
	numberOfDays := 1
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n <= 0 {
			fmt.Fprintln(os.Stderr, "Error: numberOfDays must be a positive integer.")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Usage: gantry cost [numberOfDays]")
			fmt.Fprintln(os.Stderr, "Example: gantry cost 7")
			fmt.Fprintln(os.Stderr, "")
			os.Exit(1)
		}
		numberOfDays = n
	}

	// Load config
	globalConfig, err := config.LoadGlobalConfigRaw()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Require OTEL endpoint
	if globalConfig.OTEL.Endpoint == "" {
		fmt.Fprintln(os.Stderr, "Error: otel.endpoint is required for cost tracking.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Ensure your ~/.gantryrc.json has an \"otel\" section with endpoint configured.")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// Require OTEL headers (for bearer token)
	if globalConfig.OTEL.Headers == "" {
		fmt.Fprintln(os.Stderr, "Error: otel.headers is required for cost tracking.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Ensure your ~/.gantryrc.json has otel.headers configured with an Authorization header.")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// Require username
	username := ""
	if globalConfig.Gantry != nil {
		username = globalConfig.Gantry.Username
	}
	if envUsername := os.Getenv("GANTRY_USERNAME"); envUsername != "" {
		username = envUsername
	}
	if username == "" {
		fmt.Fprintln(os.Stderr, "Error: gantry.username is required for cost tracking.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Set your username in ~/.gantryrc.json under gantry.username.")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// Derive Loki base URL from otel.endpoint
	// e.g., "https://host.example.com/otlp" -> "https://host.example.com/loki"
	lokiBaseURL := strings.TrimSuffix(globalConfig.OTEL.Endpoint, "/otlp")
	lokiBaseURL = strings.TrimSuffix(lokiBaseURL, "/")
	lokiBaseURL += "/loki"

	// Extract bearer token from otel.headers
	// Format: "Authorization=Bearer <token>"
	authHeader := ""
	parts := strings.SplitN(globalConfig.OTEL.Headers, "=", 2)
	if len(parts) == 2 {
		authHeader = parts[1]
	}
	if authHeader == "" {
		fmt.Fprintln(os.Stderr, "Error: could not extract authorization from otel.headers.")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// Build the LogQL query
	durationHours := numberOfDays * 24
	logqlQuery := fmt.Sprintf(
		`sum(sum_over_time({service_name="claude-code"} | event_name="api_request" | gantry_username="%s" | unwrap cost_usd [%dh]))`,
		username, durationHours,
	)

	// Build the Loki API URL
	nowNanos := time.Now().UnixNano()
	queryURL := fmt.Sprintf("%s/loki/api/v1/query?query=%s&time=%d",
		lokiBaseURL,
		url.QueryEscape(logqlQuery),
		nowNanos,
	)

	// Create the request
	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", authHeader)

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying telemetry data: %v\n", err)
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
		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Fprintln(os.Stderr, "Error: unauthorized. Check that your otel.headers token is correct.")
		} else {
			fmt.Fprintf(os.Stderr, "Error: telemetry server returned status %d: %s\n", resp.StatusCode, string(body))
		}
		os.Exit(1)
	}

	// Parse the response
	var lokiResp lokiResponse
	if err := json.Unmarshal(body, &lokiResp); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	// Extract spend total
	spend := 0.0
	if len(lokiResp.Data.Result) > 0 && len(lokiResp.Data.Result[0].Value) >= 2 {
		if spendStr, ok := lokiResp.Data.Result[0].Value[1].(string); ok {
			spend, _ = strconv.ParseFloat(spendStr, 64)
		}
	}

	fmt.Printf("Total spent over the past %d day(s): $%.2f\n", numberOfDays, spend)
}
