package config

// GlobalConfig represents the ~/.gantryrc.json configuration file
type GlobalConfig struct {
	Gantry    *GantryConfig    `json:"gantry,omitempty"`
	OTEL      OTELConfig       `json:"otel"`
	Bedrock   *BedrockConfig   `json:"bedrock,omitempty"`
	Powerline *PowerlineConfig `json:"powerline,omitempty"`
}

// GantryConfig contains GANTRY-specific configuration
type GantryConfig struct {
	Username        string `json:"username"`
	EnablePowerline *bool  `json:"enablePowerline,omitempty"`
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
	Enabled          bool   `json:"enabled"`
	AWSProfile       string `json:"awsProfile,omitempty"`
	AWSRegion        string `json:"awsRegion,omitempty"`
	Model            string `json:"model,omitempty"`
	MaxOutputTokens  int    `json:"maxOutputTokens,omitempty"`
	MaxThinkingTokens int   `json:"maxThinkingTokens,omitempty"`
}

// PowerlineConfig contains claude-powerline configuration
type PowerlineConfig struct {
	Theme string `json:"theme,omitempty"`
	Style string `json:"style,omitempty"`
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
