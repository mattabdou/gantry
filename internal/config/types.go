package config

// GlobalConfig represents the ~/.gantryrc.json configuration file
type GlobalConfig struct {
	Gantry             *GantryConfig             `json:"gantry,omitempty"`
	OTEL               OTELConfig                `json:"otel"`
	Bedrock            *BedrockConfig            `json:"bedrock,omitempty"`
	LiteLLM            *LiteLLMConfig            `json:"litellm,omitempty"`
	Powerline          *PowerlineConfig          `json:"powerline,omitempty"`
	ClaudeCodeTerminal *ClaudeCodeTerminalConfig `json:"claudeCodeTerminal,omitempty"`
}

// GantryConfig contains GANTRY-specific configuration
type GantryConfig struct {
	Mode                   string `json:"mode,omitempty"`
	DefaultTool            string `json:"defaultTool,omitempty"` // "cc" for Claude Code, "oc" for OpenCode Terminal, "ocd" for OpenCode Desktop
	Username               string `json:"username"`
	Release                string `json:"release,omitempty"` // "stable" or "beta" - which release channel to use for updates
	IgnorePowerline        *bool  `json:"ignorePowerline,omitempty"`
	EnablePowerline        *bool  `json:"enablePowerline,omitempty"`
	BypassLoadingScreen    *bool  `json:"bypassLoadingScreen,omitempty"`
	AutoUpdate             *bool  `json:"autoUpdate,omitempty"`             // Auto-update gantry on launch when update available
	CheckForUpdateOnLaunch *bool  `json:"checkForUpdateOnLaunch,omitempty"`
	LastUpdateCheck        string `json:"lastUpdateCheck,omitempty"`  // RFC3339 timestamp of last update check
	LastUpdateResult       string `json:"lastUpdateResult,omitempty"` // Cached latest version from last check
}

// OTELConfig contains OpenTelemetry configuration
type OTELConfig struct {
	Endpoint             string `json:"endpoint"`
	Headers              string `json:"headers,omitempty"`
	Protocol             string `json:"protocol,omitempty"`
	MetricsExporter      string `json:"metricsExporter,omitempty"`
	LogsExporter         string `json:"logsExporter,omitempty"`
	MetricExportInterval int    `json:"metricExportInterval,omitempty"`
	LogsExportInterval   int    `json:"logsExportInterval,omitempty"`
	LogUserPrompts       bool   `json:"logUserPrompts,omitempty"`
	IncludeSessionID     *bool  `json:"includeSessionId,omitempty"`
	IncludeVersion       *bool  `json:"includeVersion,omitempty"`
	IncludeAccountUUID   *bool  `json:"includeAccountUuid,omitempty"`
}

// BedrockConfig contains AWS Bedrock configuration
type BedrockConfig struct {
	AWSProfile        string `json:"awsProfile,omitempty"`
	AWSRegion         string `json:"awsRegion,omitempty"`
	Model             string `json:"model,omitempty"`
	MaxOutputTokens   int    `json:"maxOutputTokens,omitempty"`
	MaxThinkingTokens int    `json:"maxThinkingTokens,omitempty"`
}

// LiteLLMConfig contains LiteLLM proxy configuration
type LiteLLMConfig struct {
	BaseURL           string `json:"baseUrl,omitempty"`
	AuthToken         string `json:"authToken,omitempty"`
	Model             string `json:"model,omitempty"`
	MaxOutputTokens   int    `json:"maxOutputTokens,omitempty"`
	MaxThinkingTokens int    `json:"maxThinkingTokens,omitempty"`
}

// PowerlineConfig contains claude-powerline configuration
type PowerlineConfig struct {
	Theme string `json:"theme,omitempty"`
	Style string `json:"style,omitempty"`
}

// ClaudeCodeTerminalConfig contains Claude Code terminal-specific settings
type ClaudeCodeTerminalConfig struct {
	DisableExperimentalBetas *int `json:"disableExperimentalBetas,omitempty"` // 1 to disable (default), 0 to enable
}

// ProjectConfig represents the .gantry.json project configuration file
type ProjectConfig struct {
	ProjectName string `json:"projectName,omitempty"`
	Repository  string `json:"repository,omitempty"`
	Team        string `json:"team,omitempty"`
	CostCenter  string `json:"costCenter,omitempty"`
}

// ProjectConfigResult contains the loaded project config and metadata
type ProjectConfigResult struct {
	Config    ProjectConfig
	Path      string
	Directory string
}

// ClaudeSettings represents the ~/.claude/settings.json file structure
type ClaudeSettings struct {
	StatusLine *StatusLineConfig `json:"statusLine,omitempty"`
}

// StatusLineConfig represents the statusLine configuration in Claude settings
type StatusLineConfig struct {
	Type    string `json:"type,omitempty"`
	Command string `json:"command,omitempty"`
}
