# CLAUDE.md - Project Guide for GANTRY

## Project Overview

GANTRY (Gateway for AI Navigation, Telemetry, and Runtime Yield) is a Go CLI application that acts as a launcher/wrapper for Claude Code. It enriches Claude Code with:

- **Multiple Providers** - Supports AWS Bedrock or LiteLLM proxy via `mode` setting
- **OpenTelemetry Telemetry** - Adds custom resource attributes for AI cost tracking (username, project, team, cost center, git branch)
- **claude-powerline** - Configures the status bar theme and style

## Project Structure

```
gantry/
├── main.go                     # Entry point
├── cmd/                        # Cobra command definitions
│   ├── root.go                 # Main command, runs Claude Code with env configuration
│   ├── init.go                 # 'gantry init' - creates ~/.gantryrc.json
│   ├── config.go               # 'gantry config' - interactive/CLI config management
│   ├── version.go              # 'gantry version' - displays version info
│   └── update.go               # 'gantry update' - self-update functionality
├── internal/
│   ├── config/                 # Configuration handling
│   │   ├── types.go            # Config struct definitions (GlobalConfig, ProjectConfig, etc.)
│   │   └── config.go           # Load/save/validate config, find project config
│   ├── launcher/               # Claude Code launcher
│   │   └── launcher.go         # Environment building, resource attributes, launch process
│   ├── powerline/              # claude-powerline integration
│   │   └── powerline.go        # Update ~/.claude/settings.json with statusLine config
│   └── updater/                # Self-update functionality
│       └── updater.go          # GitHub release checking, download, replace executable
├── Makefile                    # Build targets for all platforms
├── go.mod                      # Go 1.21+, single dependency: spf13/cobra
└── go.sum
```

## Build Commands

```bash
make build          # Build for current platform
make test           # Run tests
make build-all      # Cross-compile for all platforms (Linux/macOS/Windows, AMD64/ARM64)
make release        # Build all and create release archives
make clean          # Remove build artifacts
```

## Key Configuration Files

- `~/.gantryrc.json` - Global config (gantry settings, OTEL endpoint, headers, Bedrock/LiteLLM settings, powerline)
- `.gantry.json` - Per-project config (projectName, repository, team, costCenter)
- `~/.claude/settings.json` - Claude Code settings (modified by GANTRY for powerline only if ignorePowerline=false)

## Configuration Sections in ~/.gantryrc.json

**gantry section:**
- `mode` - Provider mode: `bedrock` or `litellm` (required, or use `--mode` flag)
- `username` - Username for telemetry attribution (required)
- `ignorePowerline` - Skip all powerline configuration (default: true)
- `enablePowerline` - Whether to configure powerline status bar (default: true, requires ignorePowerline=false)
- `bypassLoadingScreen` - Skip the confirmation screen on startup (default: false)

**otel section:**
- `endpoint` - OTEL collector endpoint URL (required)
- `headers` - Authentication headers
- Other OTEL configuration options

**bedrock section:** (used when mode is `bedrock`)
- `awsProfile` - AWS profile name
- `awsRegion` - AWS region
- `model`, `maxOutputTokens`, `maxThinkingTokens`

**litellm section:** (used when mode is `litellm`)
- `baseUrl` - LiteLLM proxy base URL
- `authToken` - Authentication token
- `model`, `maxOutputTokens`, `maxThinkingTokens`

**powerline section:**
- Theme and style settings for claude-powerline (optional)

## Environment Variables

GANTRY reads username from `gantry.username` in config. The `GANTRY_USERNAME` environment variable can optionally override this for CI/CD scenarios.

GANTRY sets the following environment variables for Claude Code:

**OTEL:**
- `CLAUDE_CODE_ENABLE_TELEMETRY=1`
- `OTEL_METRICS_EXPORTER`, `OTEL_LOGS_EXPORTER`
- `OTEL_EXPORTER_OTLP_PROTOCOL`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_HEADERS`
- `OTEL_RESOURCE_ATTRIBUTES` (contains gantry.username, gantry.project_name, gantry.git_branch, etc.)

**Bedrock mode:**
- `CLAUDE_CODE_USE_BEDROCK=1`
- `AWS_PROFILE`, `AWS_REGION`, `ANTHROPIC_MODEL`
- `CLAUDE_CODE_MAX_OUTPUT_TOKENS`, `MAX_THINKING_TOKENS`

**LiteLLM mode:**
- `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_MODEL`
- `CLAUDE_CODE_MAX_OUTPUT_TOKENS`, `MAX_THINKING_TOKENS`

## Code Patterns

### Config Loading
Project config is found by walking up directories from CWD (like git). Global config is always at `~/.gantryrc.json`.

### Config Value Access
Dot notation for get/set operations (e.g., `gantry config set otel.endpoint https://...`). Type coercion handled in `config.go:SetConfigValue()`.

### Command Structure
Uses Cobra with `DisableFlagParsing: true` on root command to pass unknown flags through to Claude Code. Subcommands (init, config, update, version) are detected and handled specially in `root.go:runGantry()`.

### Self-Update
Downloads platform-specific binary from GitHub releases (`mattabdou/gantry`), replaces current executable. Version comparison is simple semver-like string comparison.

## Testing

Each internal package has corresponding `*_test.go` files. Run with:

```bash
make test
make test-coverage  # Generates coverage.html
```

## Version

Current version is defined in `cmd/version.go:Version` constant (currently "1.0.0").
