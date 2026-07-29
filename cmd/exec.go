package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattabdou/gantry/internal/codex"
	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/launcher"
	"github.com/mattabdou/gantry/internal/opencode"
	"github.com/spf13/cobra"
)

// defaultCodexModel mirrors the migration default in config.LoadGlobalConfig.
// 'gantry exec' loads the config without migrating it, so the codex section may
// legitimately be absent.
const defaultCodexModel = "gpt-5.6-terra"

var (
	execTool              string
	execMode              string
	execOutputFormat      string
	execModel             string
	execPromptFile        string
	execStdin             bool
	execNoSkipPermissions bool
	execNoConfigure       bool
	execVerbose           bool
	execPrintCommand      bool
)

var execCmd = &cobra.Command{
	Use:   "exec [flags] <prompt> [-- tool-args...]",
	Short: "Run an AI tool non-interactively with a prompt (headless / IDE integration)",
	Long: `Run an AI tool non-interactively and exit when it finishes.

Intended for IDEs and automation that spawn an AI coding session per request.
All GANTRY setup (telemetry attributes, provider configuration) is applied, then
the tool runs with permission checks bypassed and the given prompt.

Only gantry's own diagnostics are written to stderr; stdout carries the tool's
output verbatim so it can be parsed. GANTRY never reads stdin unless --stdin is
passed, and the confirmation screen, update check, auto-update and powerline
configuration are all skipped.

Permission bypass is enabled by default. Disable it for a single run with
--no-skip-permissions, or for all runs by setting gantry.allowDangerousHeadless
to false in ~/.gantryrc.json.

GANTRY's own flags must come before the prompt. Anything after the prompt (or
after --) is passed to the tool verbatim.`,
	Example: `  gantry exec "fix the failing auth test"
  gantry exec -t cc -o stream-json "refactor the parser"
  gantry exec --prompt-file ./task.md
  gantry exec --print-command "dry run"
  gantry exec "add tests" -- --add-dir /tmp/scratch`,
	Run: func(cmd *cobra.Command, args []string) {
		runExec(cmd, args)
	},
}

func init() {
	f := execCmd.Flags()
	// Flags must precede the prompt; everything after it belongs to the tool.
	f.SetInterspersed(false)
	f.StringVarP(&execTool, "tool", "t", "", "AI tool to run: cc, co, oc")
	f.StringVarP(&execMode, "mode", "m", "", "Provider mode: bedrock or litellm")
	f.StringVarP(&execOutputFormat, "output-format", "o", "text", "Output format: text, json or stream-json")
	f.StringVar(&execModel, "model", "", "Override the model for this run")
	f.StringVar(&execPromptFile, "prompt-file", "", "Read the prompt from a file")
	f.BoolVar(&execStdin, "stdin", false, "Read the prompt from stdin")
	f.BoolVar(&execNoSkipPermissions, "no-skip-permissions", false, "Do not bypass the tool's permission checks")
	f.BoolVar(&execNoConfigure, "no-configure", false, "Skip provider configuration file writes")
	f.BoolVar(&execVerbose, "verbose", false, "Log the resolved configuration to stderr")
	f.BoolVar(&execPrintCommand, "print-command", false, "Print the resolved command to stderr and exit without running it")
	rootCmd.AddCommand(execCmd)
}

// execErrorf writes a gantry-level error to stderr and exits 1.
// The prefix lets callers distinguish gantry failures from tool failures.
func execErrorf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "gantry exec: "+format+"\n", a...)
	os.Exit(1)
}

// execLogf writes a diagnostic to stderr when --verbose is set.
func execLogf(format string, a ...interface{}) {
	if execVerbose {
		fmt.Fprintf(os.Stderr, "gantry exec: "+format+"\n", a...)
	}
}

// quoteCommand renders a command and its arguments so the result can be pasted
// into a shell. Only used for stderr diagnostics.
func quoteCommand(command string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	for _, a := range append([]string{command}, args...) {
		if a == "" || strings.ContainsAny(a, " \t\n\"'\\$`") {
			parts = append(parts, fmt.Sprintf("%q", a))
		} else {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, " ")
}

// splitExecArgs separates the prompt words from the tool passthrough args.
//
// dashPos is cobra's ArgsLenAtDash. With interspersed parsing disabled pflag
// stops at the first positional and leaves a later "--" in the args, so both
// forms have to be handled.
func splitExecArgs(args []string, dashPos int) (promptWords, extras []string) {
	clone := func(s []string) []string {
		if len(s) == 0 {
			return nil
		}
		return append([]string(nil), s...)
	}

	if dashPos >= 0 && dashPos <= len(args) {
		return clone(args[:dashPos]), clone(args[dashPos:])
	}

	for i, a := range args {
		if a == "--" {
			return clone(args[:i]), clone(args[i+1:])
		}
	}

	return clone(args), nil
}

// resolvePrompt determines the prompt from exactly one of the supported sources.
func resolvePrompt(promptWords []string, promptFile string, useStdin bool, stdin io.Reader) (string, error) {
	sources := 0
	if len(promptWords) > 0 {
		sources++
	}
	if promptFile != "" {
		sources++
	}
	if useStdin {
		sources++
	}

	if sources == 0 {
		return "", errors.New("no prompt provided; pass a prompt argument, --prompt-file or --stdin")
	}
	if sources > 1 {
		return "", errors.New("provide the prompt exactly one way: a prompt argument, --prompt-file or --stdin")
	}

	var prompt string
	switch {
	case promptFile != "":
		data, err := os.ReadFile(promptFile)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file: %w", err)
		}
		prompt = string(data)
	case useStdin:
		if stdin == nil {
			return "", errors.New("--stdin was given but no input is available")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt from stdin: %w", err)
		}
		prompt = string(data)
	default:
		prompt = strings.Join(promptWords, " ")
	}

	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", errors.New("prompt is empty")
	}

	return prompt, nil
}

// normalizeOutputFormat validates the output format, defaulting to text.
func normalizeOutputFormat(format string) (string, error) {
	switch format {
	case "":
		return "text", nil
	case "text", "json", "stream-json":
		return format, nil
	default:
		return "", fmt.Errorf("invalid output format %q (must be text, json or stream-json)", format)
	}
}

// validateExecTool checks that a tool can run headless.
func validateExecTool(tool string) error {
	if launcher.IsHeadlessTool(tool) {
		return nil
	}
	if config.IsValidTool(tool) {
		return fmt.Errorf("tool %q has no non-interactive mode; headless execution supports cc (Claude Code), co (Codex) and oc (OpenCode Terminal)", tool)
	}
	return fmt.Errorf("invalid tool %q; headless execution supports cc (Claude Code), co (Codex) and oc (OpenCode Terminal)", tool)
}

// resolveExecTool determines the effective tool, preferring the flag over config.
func resolveExecTool(globalConfig *config.GlobalConfig, toolFlag string) (tool, source string) {
	if toolFlag != "" {
		return toolFlag, "flag"
	}
	if globalConfig != nil && globalConfig.Gantry != nil && globalConfig.Gantry.DefaultTool != "" {
		return globalConfig.Gantry.DefaultTool, "config"
	}
	return "cc", "default"
}

// resolveExecMode determines the effective provider mode.
func resolveExecMode(globalConfig *config.GlobalConfig, modeFlag string) (mode, source string, err error) {
	mode, source = modeFlag, "flag"
	if mode == "" {
		if globalConfig != nil && globalConfig.Gantry != nil && globalConfig.Gantry.Mode != "" {
			mode, source = globalConfig.Gantry.Mode, "config"
		}
	}

	if mode == "" {
		return "", "", errors.New("mode must be specified; set \"gantry.mode\" in ~/.gantryrc.json or pass --mode bedrock|litellm")
	}
	if mode != "bedrock" && mode != "litellm" {
		return "", "", fmt.Errorf("invalid mode %q (must be bedrock or litellm)", mode)
	}

	return mode, source, nil
}

// execToolRequiresLiteLLM reports whether a headless tool only works via LiteLLM.
func execToolRequiresLiteLLM(tool string) bool {
	return tool == "co"
}

// validateExecProviderConfig checks that the config for the selected mode is usable.
func validateExecProviderConfig(globalConfig *config.GlobalConfig, mode string) error {
	switch mode {
	case "bedrock":
		if globalConfig.Bedrock == nil {
			return errors.New("bedrock mode selected but the \"bedrock\" section is not configured in ~/.gantryrc.json")
		}
		if globalConfig.Bedrock.AWSProfile == "" {
			return errors.New("bedrock.awsProfile is required when using bedrock mode")
		}
		if globalConfig.Bedrock.AWSRegion == "" {
			return errors.New("bedrock.awsRegion is required when using bedrock mode")
		}
	case "litellm":
		if globalConfig.LiteLLM == nil {
			return errors.New("litellm mode selected but the \"litellm\" section is not configured in ~/.gantryrc.json")
		}
		if globalConfig.LiteLLM.BaseURL == "" {
			return errors.New("litellm.baseUrl is required when using litellm mode")
		}
		if globalConfig.LiteLLM.AuthToken == "" {
			return errors.New("litellm.authToken is required when using litellm mode")
		}
	}
	return nil
}

// resolveBypass decides whether to pass the permission-bypass flag.
func resolveBypass(configAllows bool, noSkipFlag bool) bool {
	if noSkipFlag {
		return false
	}
	return configAllows
}

// detectExecTool verifies the selected tool is installed.
func detectExecTool(tool string) error {
	switch tool {
	case "cc":
		if !launcher.DetectClaudeCode().Installed {
			return fmt.Errorf("Claude Code is not installed; install it with: %s (other methods: %s)",
				launcher.ClaudeCodeInstallCommand(), launcher.ClaudeCodeInstallDocsURL)
		}
	case "co":
		if !launcher.DetectCodex().Installed {
			return errors.New("Codex is not installed; install it with: npm install -g @openai/codex")
		}
	case "oc":
		if !launcher.DetectOpenCodeTerminal().Installed {
			return errors.New("OpenCode Terminal is not installed; see https://opencode.ai/download")
		}
	}
	return nil
}

func runExec(cmd *cobra.Command, args []string) {
	promptWords, extras := splitExecArgs(args, cmd.ArgsLenAtDash())

	// stdin is only ever read when explicitly requested, and only here.
	var promptStdin io.Reader
	if execStdin {
		promptStdin = os.Stdin
	}

	prompt, err := resolvePrompt(promptWords, execPromptFile, execStdin, promptStdin)
	if err != nil {
		execErrorf("%v", err)
	}

	outputFormat, err := normalizeOutputFormat(execOutputFormat)
	if err != nil {
		execErrorf("%v", err)
	}

	if !config.GlobalConfigExists() {
		execErrorf("GANTRY is not configured; run 'gantry init' first")
	}

	// Load without the auto-migration writes that LoadGlobalConfig performs.
	// Concurrent headless invocations would otherwise race on a non-atomic
	// rewrite of ~/.gantryrc.json.
	globalConfig, err := config.LoadGlobalConfigRaw()
	if err != nil {
		execErrorf("%v", err)
	}
	if err := config.ValidateGlobalConfig(globalConfig); err != nil {
		execErrorf("%v", err)
	}

	tool, toolSource := resolveExecTool(globalConfig, execTool)
	if err := validateExecTool(tool); err != nil {
		execErrorf("%v", err)
	}

	mode, modeSource, err := resolveExecMode(globalConfig, execMode)
	if err != nil {
		execErrorf("%v", err)
	}

	if execToolRequiresLiteLLM(tool) && mode != "litellm" {
		execErrorf("tool %q requires litellm mode; pass --mode litellm or set gantry.mode to \"litellm\"", tool)
	}

	if err := validateExecProviderConfig(globalConfig, mode); err != nil {
		execErrorf("%v", err)
	}

	if err := detectExecTool(tool); err != nil {
		execErrorf("%v", err)
	}

	skipPermissions := resolveBypass(config.AllowDangerousHeadless(globalConfig), execNoSkipPermissions)
	if !skipPermissions {
		if execNoSkipPermissions {
			execLogf("permission bypass disabled by --no-skip-permissions")
		} else {
			fmt.Fprintln(os.Stderr, "gantry exec: warning: permission bypass is disabled by gantry.allowDangerousHeadless=false in ~/.gantryrc.json.")
			fmt.Fprintln(os.Stderr, "gantry exec: the run will continue, but the tool may be unable to complete actions that require approval.")
		}
	}

	// Claude Code refuses to bypass permissions as root. Report it plainly
	// rather than working around a deliberate safety check.
	if tool == "cc" && skipPermissions && launcher.IsRunningAsRoot() && os.Getenv("IS_SANDBOX") == "" {
		fmt.Fprintln(os.Stderr, "gantry exec: error: Claude Code refuses --dangerously-skip-permissions when running as root.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Choose one of:")
		fmt.Fprintln(os.Stderr, "  - run gantry exec as a non-root user (recommended)")
		fmt.Fprintln(os.Stderr, "  - re-run with --no-skip-permissions to proceed without bypass")
		fmt.Fprintln(os.Stderr, "  - if this container is genuinely isolated, set IS_SANDBOX=1 in the environment yourself")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	username := globalConfig.Gantry.Username
	if envUsername := os.Getenv("GANTRY_USERNAME"); envUsername != "" {
		username = envUsername
	}

	workingPath, err := os.Getwd()
	if err != nil {
		execErrorf("failed to get working directory: %v", err)
	}

	projectConfig := config.FindProjectConfig(workingPath)
	if projectConfig == nil {
		projectConfig = config.GetDefaultProjectConfig(workingPath)
	}

	execLogf("tool=%s (%s) mode=%s (%s) output=%s bypass=%t", tool, toolSource, mode, modeSource, outputFormat, skipPermissions)
	execLogf("username=%s project=%s", username, projectConfig.Config.ProjectName)

	// Provider configuration files are load-bearing, so they are still written,
	// but any message goes to stderr. Powerline, the update check and the
	// confirmation screen are all skipped entirely.
	if !execNoConfigure {
		if err := configureExecProvider(tool, mode, globalConfig); err != nil {
			execErrorf("%v", err)
		}
	} else {
		execLogf("skipping provider configuration (--no-configure)")
	}

	gitBranch := launcher.GetGitBranch()
	resourceAttributes := launcher.BuildResourceAttributes(username, workingPath, projectConfig, gitBranch, Version)
	resourceAttributes = launcher.BuildHeadlessResourceAttributes(resourceAttributes, tool, outputFormat)

	// Clamp the OTEL export intervals on a copy so a short-lived run still
	// exports its cost metrics. OTEL is a value field, so this does not mutate
	// the loaded config.
	clampedConfig := *globalConfig
	clampedConfig.OTEL = launcher.ClampHeadlessOTEL(globalConfig.OTEL)

	env := launcher.BuildEnvironment(&clampedConfig, resourceAttributes, mode)
	env = launcher.SanitizeHeadlessEnv(env)
	if tool == "co" {
		env = append(env, "GANTRY_LITELLM_API_KEY="+globalConfig.LiteLLM.AuthToken)
	}

	command, err := launcher.HeadlessCommand(tool)
	if err != nil {
		execErrorf("%v", err)
	}

	toolArgs, err := launcher.BuildHeadlessArgs(launcher.HeadlessRequest{
		Tool:            tool,
		Prompt:          prompt,
		OutputFormat:    outputFormat,
		Model:           execModel,
		SkipPermissions: skipPermissions,
		ExtraArgs:       extras,
	})
	if err != nil {
		execErrorf("%v", err)
	}

	if execPrintCommand {
		fmt.Fprintln(os.Stderr, quoteCommand(command, toolArgs))
		return
	}

	execLogf("running: %s", quoteCommand(command, toolArgs))

	// stdin is deliberately nil so the child gets /dev/null: gantry must never
	// compete with the tool for stdin, and inherited stdin changes behavior for
	// some tools (codex appends it to the prompt).
	exitCode, err := launcher.RunHeadless(command, toolArgs, env, nil, os.Stdout, os.Stderr)
	if err != nil {
		execErrorf("failed to run %s: %v", command, err)
	}

	os.Exit(exitCode)
}

// configureExecProvider writes the tool's provider configuration file.
func configureExecProvider(tool, mode string, globalConfig *config.GlobalConfig) error {
	switch tool {
	case "co":
		// The config is loaded without migrations, so the codex section may be nil.
		codexModel := defaultCodexModel
		if globalConfig.Codex != nil && globalConfig.Codex.Model != "" {
			codexModel = globalConfig.Codex.Model
		}
		result, err := codex.ConfigureProvider(codexModel, globalConfig.LiteLLM, globalConfig.OTEL)
		if err != nil {
			return fmt.Errorf("failed to configure Codex: %w", err)
		}
		if result != nil && result.Updated {
			execLogf("codex: %s", result.Message)
		}
	case "oc":
		var result *opencode.ConfigureResult
		var err error
		if mode == "litellm" {
			result, err = opencode.ConfigureLiteLLM(globalConfig.LiteLLM)
		} else {
			result, err = opencode.ConfigureBedrock(globalConfig.Bedrock)
		}
		if err != nil {
			return fmt.Errorf("failed to configure OpenCode: %w", err)
		}
		if result != nil && result.Updated {
			execLogf("opencode: %s", result.Message)
		}
	}
	return nil
}
