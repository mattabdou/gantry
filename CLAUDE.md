# CLAUDE.md - Project Guide for GANTRY

## Project Overview

GANTRY (Gateway for AI Navigation, Telemetry, and Runtime Yield) is a Go CLI application that acts as a launcher/wrapper for AI coding tools. It supports:

- **Multiple Tools** - Launch Claude Code, OpenCode Terminal, or OpenCode Desktop via `defaultTool` setting or `--tool` flag
- **Multiple Providers** - Supports AWS Bedrock or LiteLLM proxy via `mode` setting
- **OpenTelemetry Telemetry** - Adds custom resource attributes for AI cost tracking (username, project, team, cost center, git branch) - Claude Code only
- **claude-powerline** - Configures the status bar theme and style - Claude Code only

## Project Structure

```
gantry/
├── main.go                     # Entry point
├── cmd/                        # Cobra command definitions
│   ├── root.go                 # Main command, runs AI tool with env configuration
│   ├── init.go                 # 'gantry init' - creates ~/.gantryrc.json
│   ├── config.go               # 'gantry config' - interactive/CLI config management
│   ├── models.go               # 'gantry models' - list LiteLLM models
│   ├── cost.go                 # 'gantry cost' - show API spend from telemetry data
│   ├── version.go              # 'gantry version' - displays version info
│   └── update.go               # 'gantry update' - self-update functionality
├── internal/
│   ├── config/                 # Configuration handling
│   │   ├── types.go            # Config struct definitions (GlobalConfig, ProjectConfig, etc.)
│   │   └── config.go           # Load/save/validate config, find project config
│   ├── launcher/               # Tool launcher
│   │   ├── launcher.go         # Environment building, resource attributes, launch process
│   │   └── detection.go        # Tool installation detection (Claude Code, OpenCode)
│   ├── opencode/               # OpenCode integration
│   │   └── opencode.go         # OpenCode config file management (~/.config/opencode/opencode.json)
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
- `~/.config/opencode/opencode.json` - OpenCode settings (modified by GANTRY to configure provider)

## Configuration Sections in ~/.gantryrc.json

**gantry section:**
- `mode` - Provider mode: `bedrock` or `litellm` (required, or use `--mode` flag)
- `defaultTool` - Default AI tool: `cc` (Claude Code), `oc` (OpenCode Terminal), `ocd` (OpenCode Desktop). Defaults to `cc`
- `username` - Username for telemetry attribution (required)
- `release` - Release channel: `stable` or `beta`. Defaults to `stable`
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
- Theme and style settings for claude-powerline (optional, Claude Code only)

## Command Line Flags

- `--tool, -t <tool>` - Override defaultTool. Values: `cc`, `oc`, `ocd`
- `--mode, -m <mode>` - Override mode. Values: `bedrock`, `litellm`

## Environment Variables

GANTRY reads username from `gantry.username` in config. The `GANTRY_USERNAME` environment variable can optionally override this for CI/CD scenarios.

GANTRY sets the following environment variables for Claude Code:

**OTEL:**
- `CLAUDE_CODE_ENABLE_TELEMETRY=1`
- `OTEL_METRICS_EXPORTER`, `OTEL_LOGS_EXPORTER`
- `OTEL_EXPORTER_OTLP_PROTOCOL`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_HEADERS`
- `OTEL_RESOURCE_ATTRIBUTES` (contains gantry.username, gantry.version, gantry.project_name, gantry.git_branch, etc.)

**Bedrock mode:**
- `CLAUDE_CODE_USE_BEDROCK=1`
- `AWS_PROFILE`, `AWS_REGION`, `ANTHROPIC_MODEL`
- `CLAUDE_CODE_MAX_OUTPUT_TOKENS`, `MAX_THINKING_TOKENS`

**LiteLLM mode:**
- `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_MODEL`
- `CLAUDE_CODE_MAX_OUTPUT_TOKENS`, `MAX_THINKING_TOKENS`

## OpenCode Integration

When using OpenCode (oc or ocd), GANTRY:
1. Detects if OpenCode is installed
2. Creates a timestamped backup of existing `~/.config/opencode/opencode.json`
3. Configures the provider in the OpenCode config:
   - LiteLLM mode: Adds `gantry-litellm` provider with baseURL and apiKey
   - Bedrock mode: Adds `gantry-bedrock` provider with region and profile
4. Launches the OpenCode tool

Note: OTEL telemetry and powerline are Claude Code features and are not configured for OpenCode.

## Code Patterns

### Config Loading
Project config is found by walking up directories from CWD (like git). Global config is always at `~/.gantryrc.json`.

### Config Auto-Migration
When loading config, if `defaultTool` is missing, GANTRY automatically adds it with value `cc` and saves the config.

### Config Value Access
Dot notation for get/set operations (e.g., `gantry config set otel.endpoint https://...`). Type coercion handled in `config.go:SetConfigValue()`.

### Tool Detection
- Claude Code: Checks PATH for `claude` executable
- OpenCode Terminal: Checks PATH for `opencode` executable
- OpenCode Desktop: Checks platform-specific locations:
  - macOS: `/Applications/OpenCode.app`, `~/Applications/OpenCode.app`
  - Windows: `%LOCALAPPDATA%\Programs\OpenCode\OpenCode.exe`, Scoop installation
  - Linux: `/usr/bin/opencode-desktop`, desktop entries

### Command Structure
Uses Cobra with `DisableFlagParsing: true` on root command to pass unknown flags through to the AI tool. Subcommands (init, config, update, version, models, cost) are detected and handled specially in `root.go:runGantry()`.

### Self-Update
Downloads platform-specific binary from GitHub releases (`mattabdou/gantry`), replaces current executable. Supports two release channels:
- **stable** (default): Uses `/releases/latest` endpoint which skips pre-releases
- **beta**: Uses `/releases` endpoint and finds latest pre-release

Channel preference is stored in `gantry.release` config. Users switch channels via `gantry update --beta` or `gantry update --stable`. When switching channels, a backup of the config is created (`.gantryrc.json.stable` or `.gantryrc.json.beta`). Version comparison follows semver rules including pre-release suffixes (e.g., `1.0.0-beta.1 < 1.0.0`).

## Testing

Each internal package has corresponding `*_test.go` files. Run with:

```bash
make test
make test-coverage  # Generates coverage.html
```

## Version

Current version is defined in two places (both must be updated for releases):
- `cmd/version.go:Version` constant
- `Makefile:VERSION` variable

## Creating Releases

### Stable Releases

When creating a new stable release:

1. **Update version** in both `cmd/version.go` and `Makefile`
2. **Build binaries**: Run `make clean && make release`
3. **Create GitHub release** with `gh release create vX.Y.Z` including:

**Versioned archives** (for manual installation):
- `gantry-X.Y.Z-darwin-amd64.tar.gz`
- `gantry-X.Y.Z-darwin-arm64.tar.gz`
- `gantry-X.Y.Z-linux-amd64.tar.gz`
- `gantry-X.Y.Z-linux-arm64.tar.gz`
- `gantry-X.Y.Z-windows-amd64.zip`
- `gantry-X.Y.Z-windows-arm64.zip`

**Raw binaries** (required for `gantry update` self-updater):
- `gantry-darwin-amd64`
- `gantry-darwin-arm64`
- `gantry-linux-amd64`
- `gantry-linux-arm64`
- `gantry-windows-amd64.exe`
- `gantry-windows-arm64.exe`

**Important**: The raw binaries (without version in filename) are required because the self-updater in `internal/updater/updater.go` looks for assets matching the pattern `gantry-{os}-{arch}[.exe]`.

### Beta/Pre-releases

For beta releases that won't notify stable users:

1. **Update version** with beta suffix (e.g., `1.1.6-beta.1`) in `cmd/version.go` and `Makefile`
2. **Build binaries**: Run `make clean && make release`
3. **Create GitHub pre-release** with `gh release create vX.Y.Z-beta.N --prerelease` including the same assets as stable

**Key differences for pre-releases:**
- Use the `--prerelease` flag when creating the release
- Version tag should include pre-release suffix (e.g., `v1.1.6-beta.1`)
- Pre-releases are hidden from `/releases/latest` API endpoint
- Only users on beta channel (`gantry update --beta`) will see these updates
- Stable users won't be notified of beta releases

### Promoting Beta to Stable

To promote a beta release to stable:
1. Create a new stable release with the same version (without beta suffix)
2. Users on stable channel will then see the update
