# CLAUDE.md - Project Guide for GANTRY

## Project Overview

GANTRY (Gateway for AI Navigation, Telemetry, and Runtime Yield) is a Go CLI application that acts as a launcher/wrapper for AI coding tools. It supports:

- **Multiple Tools** - Launch Claude Code, OpenCode Terminal, or OpenCode Desktop via `defaultTool` setting or `--tool` flag
- **Multiple Providers** - Supports AWS Bedrock or LiteLLM proxy via `mode` setting
- **Shell Mode** - Configure environment without launching the tool (`--shell` / `-s`), spawning a shell with all env vars set and a `(gantry)` prompt prefix
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
│   ├── exec.go                 # 'gantry exec' - headless/non-interactive run for IDEs
│   ├── tools.go                # 'gantry tools' - list supported AI tools
│   ├── version.go              # 'gantry version' - displays version info
│   └── update.go               # 'gantry update' - self-update functionality
├── internal/
│   ├── config/                 # Configuration handling
│   │   ├── types.go            # Config struct definitions (GlobalConfig, ProjectConfig, etc.)
│   │   └── config.go           # Load/save/validate config, find project config
│   ├── launcher/               # Tool launcher
│   │   ├── launcher.go         # Environment building, resource attributes, launch process
│   │   ├── detection.go        # Tool installation detection (Claude Code, OpenCode)
│   │   ├── headless.go         # Headless arg builders, OTEL clamping, RunHeadless spawner
│   │   └── shell.go            # Shell mode: spawn user's shell with configured environment
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
- `allowDangerousHeadless` - Allow `gantry exec` to bypass the tool's permission checks. **Undefined means `true`** so that headless mode works for configs that predate the setting; set it to `false` to disable bypass fleet-wide

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
- `--shell, -s` - Launch a configured shell instead of the AI tool. All normal setup is performed; the user runs commands manually
- `--resetconfig, -r` - Reset the selected tool's configuration file to defaults. Creates a timestamped `.gantrybackup` before resetting

## Environment Variables

GANTRY reads username from `gantry.username` in config. The `GANTRY_USERNAME` environment variable can optionally override this for CI/CD scenarios.

When shell mode is active (`--shell` / `-s`), `GANTRY_SHELL=1` is set in the spawned shell so users and scripts can detect they are inside a gantry shell session.

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

## Headless Mode (`gantry exec`)

`gantry exec [flags] "<prompt>" [-- tool-args...]` runs an AI tool non-interactively and exits.
It exists so an IDE (or any automation) can spawn a fully configured AI session per request with
a single command.

**Supported tools:** `cc` (Claude Code), `co` (Codex), `oc` (OpenCode Terminal). `ocd` and the
Cline variants have no non-interactive entrypoint and are rejected with a clear error.

**Per-tool command mapping** (built by `launcher.BuildHeadlessArgs`):

| Tool | Command | Permission bypass | `-o json` | `-o stream-json` |
|------|---------|-------------------|-----------|------------------|
| `cc` | `claude -p "<prompt>"` | `--dangerously-skip-permissions` | `--output-format json` | `--output-format stream-json --verbose` |
| `co` | `codex --profile gantry exec "<prompt>" --skip-git-repo-check` | `--dangerously-bypass-approvals-and-sandbox` | `--json` | `--json` |
| `oc` | `opencode run "<prompt>"` | `--auto` | `--format json` | `--format json` |

Argument order is deliberate, not cosmetic:
- **cc**: the prompt goes immediately after `-p`, *before* passthrough args. Claude Code has
  variadic flags (`--add-dir`, `--allowedTools`, `--tools`) that would otherwise swallow a
  trailing prompt.
- **cc + stream-json**: `--verbose` is added automatically because Claude Code hard-errors with
  `When using --print, --output-format=stream-json requires --verbose`.
- **oc**: `run` takes a variadic `message` positional, so the prompt goes last.
- **co**: `--profile` is a global flag and must precede the `exec` subcommand.

**What `exec` deliberately skips, and why:**
- **Confirmation screen** - it reads stdin (`root.go` `runGantry`) and would consume a piped prompt.
- **Update check and auto-update** - auto-update can download a binary and `syscall.Exec` itself
  mid-invocation. An IDE must never have gantry swap its own binary under a live request.
- **Powerline** - a statusline for interactive TUIs; skipping it also removes a read-modify-write
  race on `~/.claude/settings.json` between concurrent invocations.
- **Config auto-migration** - `exec` uses `config.LoadGlobalConfigRaw()` plus an explicit
  `config.ValidateGlobalConfig()` instead of `LoadGlobalConfig()`, because the latter performs up
  to four non-atomic `SaveGlobalConfig()` writes that concurrent invocations could interleave.
  Consequence: the `codex` section may legitimately be nil, so `exec` falls back to
  `defaultCodexModel` rather than dereferencing it.

**I/O contract:**
- **stdout** carries only the tool's output, so `-o json` is directly parseable. `cmd/exec.go`
  contains no `fmt.Print*` calls; `TestExecStdoutPurity` enforces this by scanning the source.
- **stderr** carries all gantry diagnostics, prefixed `gantry exec:` so an IDE can tell a gantry
  failure from a tool failure. Silent unless `--verbose`.
- **stdin** is never read unless `--stdin` is passed. The child gets `/dev/null`, which also
  normalizes behavior across tools (`codex exec` would otherwise append inherited stdin to the
  prompt).
- **Exit code** is the tool's own. `launcher.RunHeadless` returns it rather than calling
  `os.Exit`, so it stays testable. Gantry-level failures exit 1. `SIGINT`/`SIGTERM` are relayed to
  the child so an IDE cancel does not orphan a running tool.

**Permission bypass** is on by default, controlled by `gantry.allowDangerousHeadless`
(undefined = true) and overridable per run with `--no-skip-permissions`. When the config disables
it, the run continues without bypass and prints an unmistakable stderr warning.

**Running as root:** Claude Code refuses `--dangerously-skip-permissions` as root. `exec` detects
this and fails with an explanatory error rather than working around the check. Setting
`IS_SANDBOX=1` is left to the operator as a deliberate choice.

**Telemetry:** headless runs add `gantry.headless=true`, `gantry.invocation=exec`, `gantry.tool`
and `gantry.output_format` to `OTEL_RESOURCE_ATTRIBUTES`, so automated spend can be separated from
interactive spend. The OTEL export intervals are clamped to 5s
(`launcher.ClampHeadlessOTEL`) because the default `metricExportInterval` of 60000ms is longer
than a typical headless run. Reachability differs by tool: full for `cc`, via its own TOML `[otel]`
block for `co`, and **none for `oc`** (OpenCode consumes no `OTEL_*` variables).

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
Uses Cobra with `DisableFlagParsing: true` on the root command to pass unknown flags through to the AI tool.

Subcommand routing is done by Cobra's own `Find()`, which runs *before* flag parsing and is
therefore unaffected by `DisableFlagParsing` - a bare first argument matching a subcommand name
(`init`, `config`, `update`, `version`, `models`, `cost`, `tools`, `exec`) is dispatched without
ever entering `runGantry`. The hardcoded subcommand list inside `runGantry` is a vestige of an
earlier design and is unreachable for those invocations.

One consequence worth knowing: because `Find()` matches on the first bare argument, a passthrough
argument that happens to equal a subcommand name is captured by that subcommand
(`gantry --agent version` errors instead of passing `--agent version` to Claude Code). Quote
multi-word prompts, or use `gantry exec` where the prompt is an explicit positional.

### Self-Update
Downloads platform-specific binary from GitHub releases (`mattabdou/gantry`), replaces current executable. Supports two release channels:
- **stable** (default): Uses `/releases/latest` endpoint which skips pre-releases
- **beta**: Uses `/releases` endpoint and finds latest pre-release

Channel preference is stored in `gantry.release` config. Users switch channels via `gantry update --beta` or `gantry update --stable`. When switching channels, a backup of the config is created (`.gantryrc.json.stable` or `.gantryrc.json.beta`). Version comparison follows semver rules including pre-release suffixes (e.g., `1.0.0-beta.1 < 1.0.0`).

## Testing

Each internal package has corresponding `*_test.go` files, plus `cmd/exec_test.go`. Run with:

```bash
make test
make test-coverage  # Generates coverage.html
```

Conventions: table-driven subtests, standard library `testing` only (the sole dependency is
`spf13/cobra`), and tests live in the same package as the code so unexported helpers are testable
directly. Prefer extracting a pure builder that returns a value over asserting on side effects -
`powerline.BuildPowerlineCommand` and `launcher.BuildHeadlessArgs` are the models to follow.

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
