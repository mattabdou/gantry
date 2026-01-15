# GANTRY - Gateway for AI Navigation, Telemetry, and Runtime Yield

A launcher for Claude Code that configures environment and telemetry.

## Overview

GANTRY automatically configures Claude Code with:

- **Multiple Providers** - Support for AWS Bedrock or LiteLLM proxy
- **Telemetry** - OpenTelemetry attributes for AI cost tracking
  - **Username** - Who is using the AI
  - **Project** - Which project the work is for
  - **Team & Cost Center** - Organizational attribution
  - **Git Branch** - Current working branch (for issue/PR tracking)
  - **Working Path** - Where the developer is working
- **claude-powerline** - Configures status bar theme and style

This data flows to your OTEL collector and can be visualized in Grafana to answer questions like:

- How much is each developer spending on AI?
- How much AI spend is attributed to each project?
- What's the AI cost for a specific feature branch or issue?

## Installation

### Linux / macOS

**Option 1: Install script (recommended)**

```bash
curl -fsSL https://raw.githubusercontent.com/mattabdou/gantry/main/install.sh | bash
```

Or download and run locally:

```bash
wget https://raw.githubusercontent.com/mattabdou/gantry/main/install.sh
chmod +x install.sh
./install.sh
```

**Option 2: Download binary manually**

Download the appropriate binary from the [releases page](https://github.com/mattabdou/gantry/releases):

| Platform | Architecture | Binary |
|----------|--------------|--------|
| Linux | AMD64 (x86_64) | `gantry-linux-amd64` |
| Linux | ARM64 | `gantry-linux-arm64` |
| macOS | Intel | `gantry-darwin-amd64` |
| macOS | Apple Silicon | `gantry-darwin-arm64` |

```bash
# Example for Linux AMD64
curl -LO https://github.com/mattabdou/gantry/releases/download/v1.0.0/gantry-linux-amd64
chmod +x gantry-linux-amd64
sudo mv gantry-linux-amd64 /usr/local/bin/gantry
```

### Windows

**Option 1: Install script (recommended)**

PowerShell:
```powershell
irm https://raw.githubusercontent.com/mattabdou/gantry/main/install.ps1 | iex
```

**Option 2: Download binary manually**

Download the appropriate `.exe` from the [releases page](https://github.com/mattabdou/gantry/releases):

| Architecture | Binary |
|--------------|--------|
| AMD64 (x86_64) | `gantry-windows-amd64.exe` |
| ARM64 | `gantry-windows-arm64.exe` |

Place it in a directory that's in your `PATH` (e.g., `%LOCALAPPDATA%\Programs\gantry`), or add its location to your `PATH`.

### Build from Source

Requires Go 1.21 or later:

```bash
git clone https://github.com/mattabdou/gantry.git
cd gantry
make build
sudo mv gantry /usr/local/bin/
```

Or build for all platforms:

```bash
make build-all
# Binaries are in build/
```

## Setup

### 1. Initialize GANTRY

Run the init command to create the global configuration file:

```bash
gantry init
```

This creates `~/.gantryrc.json` with default settings.

### 2. Configure Your Settings

Edit `~/.gantryrc.json` or run `gantry config` for an interactive editor.

At minimum, you need to configure:
- Your mode (bedrock or litellm)
- Your username (for telemetry attribution)
- Your OTEL collector endpoint and authentication
- Provider-specific settings (Bedrock or LiteLLM)

```json
{
  "gantry": {
    "mode": "bedrock",
    "username": "your.username",
    "ignorePowerline": true,
    "enablePowerline": true,
    "bypassLoadingScreen": false,
    "checkForUpdateOnLaunch": true
  },
  "otel": {
    "endpoint": "https://your-otel-collector.example.com/otlp",
    "headers": "Authorization=Bearer YOUR_TOKEN_HERE",
    "protocol": "http/protobuf",
    "metricsExporter": "otlp",
    "logsExporter": "otlp",
    "metricExportInterval": 60000,
    "logsExportInterval": 5000,
    "logUserPrompts": false,
    "includeSessionId": true,
    "includeVersion": false,
    "includeAccountUuid": true
  },
  "bedrock": {
    "awsProfile": "YOUR_AWS_PROFILE",
    "awsRegion": "us-east-2",
    "model": "us.anthropic.claude-opus-4-5-20251101-v1:0",
    "maxOutputTokens": 8192,
    "maxThinkingTokens": 1024
  },
  "litellm": {
    "baseUrl": "https://your-litellm-proxy.example.com",
    "authToken": "YOUR_AUTH_TOKEN",
    "model": "us.anthropic.claude-opus-4-5-20251101-v1:0",
    "maxOutputTokens": 8192,
    "maxThinkingTokens": 1024
  }
}
```

#### Provider Mode

Set `gantry.mode` to select which provider to use:

| Mode | Description |
|------|-------------|
| `bedrock` | Use AWS Bedrock as the Claude API provider |
| `litellm` | Use LiteLLM proxy as the Claude API provider |

You can also override the mode from the command line:

```bash
gantry --mode bedrock    # Use AWS Bedrock
gantry --mode litellm    # Use LiteLLM proxy
gantry -m bedrock        # Short form
```

#### AWS Bedrock Settings

Used when `mode` is set to `bedrock`.

| Field | Description | Default |
|-------|-------------|---------|
| `awsProfile` | AWS profile name for authentication | *Required* |
| `awsRegion` | AWS region for Bedrock | `us-east-2` |
| `model` | Anthropic model ID | `us.anthropic.claude-opus-4-5-20251101-v1:0` |
| `maxOutputTokens` | Maximum output tokens | `8192` |
| `maxThinkingTokens` | Maximum thinking tokens | `1024` |

When in Bedrock mode, GANTRY sets the following environment variables:
- `CLAUDE_CODE_USE_BEDROCK=1`
- `AWS_PROFILE`
- `AWS_REGION`
- `ANTHROPIC_MODEL`
- `CLAUDE_CODE_MAX_OUTPUT_TOKENS`
- `MAX_THINKING_TOKENS`

#### LiteLLM Settings

Used when `mode` is set to `litellm`.

| Field | Description | Default |
|-------|-------------|---------|
| `baseUrl` | LiteLLM proxy base URL | *Required* |
| `authToken` | Authentication token for LiteLLM | *Required* |
| `model` | Model name | `us.anthropic.claude-opus-4-5-20251101-v1:0` |
| `maxOutputTokens` | Maximum output tokens | `8192` |
| `maxThinkingTokens` | Maximum thinking tokens | `1024` |

When in LiteLLM mode, GANTRY sets the following environment variables:
- `ANTHROPIC_BASE_URL`
- `ANTHROPIC_AUTH_TOKEN`
- `ANTHROPIC_MODEL`
- `CLAUDE_CODE_MAX_OUTPUT_TOKENS`
- `MAX_THINKING_TOKENS`

#### OTEL Settings

| Field | Description | Default |
|-------|-------------|---------|
| `endpoint` | OTEL collector URL | *Required* |
| `headers` | Authentication headers | *Required* |
| `protocol` | OTLP protocol (`http/protobuf`, `http/json`, `grpc`) | `http/protobuf` |
| `metricsExporter` | Metrics exporter type | `otlp` |
| `logsExporter` | Logs exporter type | `otlp` |
| `metricExportInterval` | Metrics export interval (ms) | `60000` |
| `logsExportInterval` | Logs export interval (ms) | `5000` |
| `logUserPrompts` | Include user prompts in logs | `false` |
| `includeSessionId` | Include session ID in metrics | `true` |
| `includeVersion` | Include app version in metrics | `false` |
| `includeAccountUuid` | Include account UUID in metrics | `true` |

### 3. Configure Your Projects (Optional)

Create a `.gantry.json` file in your project root:

```json
{
  "projectName": "billing-api",
  "repository": "github.com/your-org/billing-api",
  "team": "platform",
  "costCenter": "ENG-001"
}
```

| Field | Description | Required |
|-------|-------------|----------|
| `projectName` | Name of the project | Yes |
| `repository` | Repository URL or identifier | No |
| `team` | Team name | No |
| `costCenter` | Cost center code | No |

GANTRY searches for `.gantry.json` starting from the current directory and walking up parent directories (like git does with `.git`). If no config is found, `projectName` defaults to `"Unknown"`.

## Usage

Instead of running `claude`, run `gantry`:

```bash
# Start Claude Code with GANTRY configuration (uses mode from config)
gantry

# Override the provider mode from command line
gantry --mode bedrock    # Use AWS Bedrock
gantry --mode litellm    # Use LiteLLM proxy
gantry -m bedrock        # Short form

# Pass arguments through to Claude Code
gantry /path/to/file

# GANTRY commands
gantry init           # Create global config
gantry init --force   # Recreate global config
gantry config         # Interactive configuration editor
gantry config show    # Display current configuration
gantry update         # Update gantry to latest version
gantry update --check # Check for updates
gantry version        # Show version info
gantry --help         # Show GANTRY help
gantry --version      # Show GANTRY version
```

## Confirmation Screen

When you run `gantry`, it displays a confirmation screen showing all the configuration that will be applied before launching Claude Code:

```
╔══════════════════════════════════════════════════════════════════╗
║              GANTRY - Claude Code Launcher                       ║
║              Version: 1.0.0                                      ║
╠══════════════════════════════════════════════════════════════════╣
║  The following configuration will be applied:                    ║
╠══════════════════════════════════════════════════════════════════╣
║  USER IDENTITY                                                   ║
║    Username:        john.doe                                     ║
║                                                                  ║
║  PROJECT                                                         ║
║    Project Name:    billing-api                                  ║
║    Config File:     /home/user/projects/api/.gantry.json         ║
║    Git Branch:      feature/JIRA-123-add-auth                    ║
║                                                                  ║
║  TELEMETRY (OTEL)                                                ║
║    Endpoint:        https://collector.example.com/otlp           ║
║                                                                  ║
║  AWS BEDROCK                                                     ║
║    Mode:            bedrock (from config)                        ║
║    AWS Profile:     my-profile                                   ║
║    Region:          us-east-2                                    ║
║    Model:           us.anthropic.claude-opus-4-5-20251101-v1:0   ║
║                                                                  ║
║  POWERLINE STATUS BAR                                            ║
║    Action:          Ignored (no changes will be made)            ║
║                                                                  ║
╠══════════════════════════════════════════════════════════════════╣
║  Press ENTER to continue, or 'q' to cancel...                    ║
╚══════════════════════════════════════════════════════════════════╝
```

This allows you to review the configuration before Claude Code starts. Press **Enter** to continue or **q** to cancel.

### Disabling the Confirmation Screen

If you prefer to skip the confirmation screen and launch Claude Code immediately, set `bypassLoadingScreen` to `true` in your `~/.gantryrc.json`:

```json
{
  "gantry": {
    "mode": "bedrock",
    "username": "your.username",
    "ignorePowerline": true,
    "enablePowerline": true,
    "bypassLoadingScreen": true
  }
}
```

Or use the command line:

```bash
gantry config set gantry.bypassLoadingScreen true
```

When bypassed, GANTRY will show minimal output (project config path and git branch) before launching Claude Code.

## Configuration Management

GANTRY provides an interactive configuration editor and command-line tools for managing your settings.

### Interactive Editor

Run `gantry config` to launch the interactive configuration editor:

```bash
gantry config
```

This walks you through each setting, showing the current value and allowing you to update it.

### View Current Configuration

```bash
gantry config show
```

Displays all current settings (with sensitive values partially masked).

### Get/Set Individual Values

```bash
# Get a specific value
gantry config get otel.endpoint

# Set a specific value
gantry config set otel.endpoint https://collector.example.com/otlp
gantry config set otel.headers "Authorization=Bearer mytoken123"
gantry config set otel.metricExportInterval 30000
gantry config set otel.logUserPrompts true
```

### Configuration Keys

Use dot notation for nested values:

| Key | Type | Description |
|-----|------|-------------|
| `gantry.mode` | string | Provider mode: `bedrock` or `litellm` |
| `gantry.username` | string | Your username for telemetry |
| `gantry.ignorePowerline` | boolean | Skip all powerline configuration (default: true) |
| `gantry.enablePowerline` | boolean | Enable powerline status bar (requires ignorePowerline=false) |
| `gantry.bypassLoadingScreen` | boolean | Skip confirmation screen on startup |
| `gantry.checkForUpdateOnLaunch` | boolean | Check for updates on startup (default: true) |
| `bedrock.awsProfile` | string | AWS profile name |
| `bedrock.awsRegion` | string | AWS region |
| `bedrock.model` | string | Anthropic model ID |
| `bedrock.maxOutputTokens` | number | Max output tokens |
| `bedrock.maxThinkingTokens` | number | Max thinking tokens |
| `litellm.baseUrl` | string | LiteLLM proxy base URL |
| `litellm.authToken` | string | LiteLLM authentication token |
| `litellm.model` | string | Model name |
| `litellm.maxOutputTokens` | number | Max output tokens |
| `litellm.maxThinkingTokens` | number | Max thinking tokens |
| `powerline.theme` | string | Powerline color theme |
| `powerline.style` | string | Powerline separator style |
| `otel.endpoint` | string | OTEL collector URL |
| `otel.headers` | string | Authentication headers |
| `otel.protocol` | string | OTLP protocol |
| `otel.metricsExporter` | string | Metrics exporter type |
| `otel.logsExporter` | string | Logs exporter type |
| `otel.metricExportInterval` | number | Metrics interval (ms) |
| `otel.logsExportInterval` | number | Logs interval (ms) |
| `otel.logUserPrompts` | boolean | Log prompt content |
| `otel.includeSessionId` | boolean | Include session ID |
| `otel.includeVersion` | boolean | Include app version |
| `otel.includeAccountUuid` | boolean | Include account UUID |

## Telemetry Attributes

GANTRY adds the following resource attributes to OTEL telemetry:

| Attribute | Source | Example |
|-----------|--------|---------|
| `gantry.username` | `gantry.username` config (or `GANTRY_USERNAME` env override) | `john.doe` |
| `gantry.working_path` | Current working directory | `/home/user/projects/api` |
| `gantry.project_name` | `.gantry.json` or `"Unknown"` | `billing-api` |
| `gantry.repository` | `.gantry.json` | `github.com/your-org/billing-api` |
| `gantry.team` | `.gantry.json` | `platform` |
| `gantry.cost_center` | `.gantry.json` | `ENG-001` |
| `gantry.git_branch` | Git current branch | `feature/JIRA-123-add-auth` |

## Grafana Queries

### Prometheus (Metrics)

```promql
# Total tokens by user
sum by (gantry_username) (claude_code_tokens_total)

# Tokens by project
sum by (gantry_project_name) (claude_code_tokens_total)

# Tokens by user and project
sum by (gantry_username, gantry_project_name) (claude_code_tokens_total)

# Tokens by team
sum by (gantry_team) (claude_code_tokens_total)

# Tokens by git branch (for issue/PR tracking)
sum by (gantry_git_branch) (claude_code_tokens_total)

# Cost center breakdown
sum by (gantry_cost_center) (claude_code_tokens_total)
```

### Loki (Logs)

```logql
# All Claude Code logs for a user
{service_name="claude-code"} | gantry_username="john.doe"

# Logs for a specific project
{service_name="claude-code"} | gantry_project_name="billing-api"

# Logs for a specific branch
{service_name="claude-code"} | gantry_git_branch=~".*JIRA-123.*"
```

## Updating GANTRY

### Self-Update

GANTRY can update itself to the latest version:

```bash
# Update to latest version
gantry update

# Check if an update is available without installing
gantry update --check
```

### Version Information

```bash
# Show current version
gantry version

# Show version and check for updates
gantry version --check
```

## claude-powerline Integration

GANTRY can optionally configure [claude-powerline](https://github.com/Owloops/claude-powerline), which provides a helpful status line at the bottom of Claude Code showing context about your session.

By default, GANTRY does not modify powerline settings (`ignorePowerline: true`). To enable powerline configuration, set `ignorePowerline` to `false`.

### Configuration

Add the following to your `~/.gantryrc.json`:

```json
{
  "gantry": {
    "ignorePowerline": false,
    "enablePowerline": true
  },
  "powerline": {
    "theme": "dark",
    "style": "powerline"
  }
}
```

### Available Options

**Themes:**
- `dark` (default)
- `light`
- `nord`
- `tokyo-night`
- `rose-pine`
- `gruvbox`

**Styles:**
- `minimal`
- `powerline` (default)
- `capsule`

### How It Works

When you run `gantry` with `ignorePowerline: false`, it:
1. Checks `enablePowerline` to determine whether to configure or remove powerline
2. Reads your powerline theme/style settings from `~/.gantryrc.json`
3. Updates `~/.claude/settings.json` with the correct statusLine configuration
4. Claude Code then uses these settings to display the powerline

If `ignorePowerline` is `true` (the default), GANTRY will not modify `~/.claude/settings.json` at all.

## Troubleshooting

### "Global config missing gantry.username"

Edit `~/.gantryrc.json` and set your username in the `gantry` section, or run `gantry config` to configure interactively.

### "GANTRY is not configured"

Run `gantry init` to create the configuration file, then edit `~/.gantryrc.json` with your OTEL collector details.

### "Global config contains placeholder value"

Edit `~/.gantryrc.json` and replace the placeholder endpoint and headers with your actual OTEL collector configuration.

### "Claude Code is not installed or not in PATH"

Install Claude Code using one of these methods:

**macOS / Linux / WSL:**
```bash
curl -fsSL https://claude.ai/install.sh | bash
```

**macOS (Homebrew):**
```bash
brew install --cask claude-code
```

**Windows (PowerShell):**
```powershell
irm https://claude.ai/install.ps1 | iex
```

**Windows (CMD):**
```cmd
curl -fsSL https://claude.ai/install.cmd -o install.cmd && install.cmd && del install.cmd
```

For more details, see the [Claude Code setup guide](https://code.claude.com/docs/en/setup).

### No project config found

This is normal - GANTRY will use `"Unknown"` as the project name. To enable project tracking, create a `.gantry.json` file in your project root.

## Platform Support

GANTRY provides pre-built binaries for:

| Platform | Architectures |
|----------|---------------|
| Linux | AMD64, ARM64 |
| macOS | AMD64 (Intel), ARM64 (Apple Silicon) |
| Windows | AMD64, ARM64 |

## Requirements

- [Claude Code](https://code.claude.com/docs/en/setup) installed
- For building from source: Go 1.21 or later

## Uninstallation

### Linux / macOS

```bash
sudo rm /usr/local/bin/gantry
# or if installed to ~/.local/bin:
rm ~/.local/bin/gantry

# Optionally remove configuration:
rm ~/.gantryrc.json
```

### Windows

Delete the `gantry.exe` file from your PATH directory and optionally remove `%USERPROFILE%\.gantryrc.json`.

## License

MIT
