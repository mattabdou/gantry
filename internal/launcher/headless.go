package launcher

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mattabdou/gantry/internal/config"
)

// HeadlessExportIntervalMs caps the OTEL export intervals for headless runs.
//
// The default metricExportInterval is 60000ms, which is longer than a typical
// headless invocation. Capping it makes cost metrics far more likely to be
// exported before the tool exits.
const HeadlessExportIntervalMs = 5000

// MaxHeadlessPromptBytes bounds the prompt size. Prompts are delivered as a
// command-line argument, so an oversized prompt would fail with a bare E2BIG
// from the OS; we return a clear error instead. The limit is well under the
// smallest ARG_MAX we target (~256KiB on macOS).
const MaxHeadlessPromptBytes = 100000

// HeadlessTools lists the tools that support a non-interactive invocation.
// OpenCode Desktop and the Cline variants have no headless entrypoint.
var HeadlessTools = []string{"cc", "co", "oc"}

// IsHeadlessTool reports whether a tool short name can run headless.
func IsHeadlessTool(tool string) bool {
	for _, t := range HeadlessTools {
		if tool == t {
			return true
		}
	}
	return false
}

// HeadlessRequest is a fully-resolved, tool-agnostic description of one
// headless run. It carries no I/O and no environment.
type HeadlessRequest struct {
	Tool   string // "cc", "co" or "oc"
	Prompt string
	// OutputFormat is "", "text", "json" or "stream-json". "" and "text" mean
	// "leave the tool at its default" and add no flags.
	OutputFormat string
	Model        string
	// SkipPermissions adds the tool's permission-bypass flag.
	SkipPermissions bool
	// ExtraArgs are passed to the tool verbatim (everything after "--").
	ExtraArgs []string
}

// HeadlessCommand returns the executable name to spawn for a tool.
func HeadlessCommand(tool string) (string, error) {
	switch tool {
	case "cc":
		// Handles the .cmd/.exe resolution needed on Windows.
		return GetClaudeCommand(), nil
	case "co":
		return "codex", nil
	case "oc":
		return "opencode", nil
	default:
		return "", fmt.Errorf("tool %q does not support headless execution (supported: %s)", tool, strings.Join(HeadlessTools, ", "))
	}
}

// isStructuredFormat reports whether the format requests machine-readable output.
func isStructuredFormat(format string) bool {
	return format == "json" || format == "stream-json"
}

// BuildHeadlessArgs builds the complete argument list (everything after argv[0])
// for a headless run. It is pure so that the exact command line for every tool
// is directly testable.
func BuildHeadlessArgs(req HeadlessRequest) ([]string, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("prompt must not be empty")
	}
	if len(req.Prompt) > MaxHeadlessPromptBytes {
		return nil, fmt.Errorf("prompt is %d bytes, which exceeds the %d byte limit; write it to a file and reference that file from a shorter prompt", len(req.Prompt), MaxHeadlessPromptBytes)
	}
	if req.OutputFormat != "" && req.OutputFormat != "text" && !isStructuredFormat(req.OutputFormat) {
		return nil, fmt.Errorf("invalid output format %q (must be text, json or stream-json)", req.OutputFormat)
	}

	var args []string

	switch req.Tool {
	case "cc":
		// The prompt goes immediately after -p, ahead of any passthrough args.
		// Claude Code has variadic flags (--add-dir, --allowedTools, --tools),
		// and a trailing prompt would be swallowed by one of them.
		args = append(args, "-p", req.Prompt)
		if isStructuredFormat(req.OutputFormat) {
			args = append(args, "--output-format", req.OutputFormat)
			if req.OutputFormat == "stream-json" {
				// Claude Code hard-errors without this:
				// "When using --print, --output-format=stream-json requires --verbose"
				args = append(args, "--verbose")
			}
		}
		if req.Model != "" {
			args = append(args, "--model", req.Model)
		}
		if req.SkipPermissions {
			args = append(args, "--dangerously-skip-permissions")
		}
		args = append(args, req.ExtraArgs...)

	case "co":
		// --profile is a global flag and must precede the exec subcommand.
		// It layers $CODEX_HOME/gantry.config.toml, which codex.ConfigureProvider writes.
		args = append(args, "--profile", "gantry", "exec", req.Prompt)
		if isStructuredFormat(req.OutputFormat) {
			// Codex emits JSONL for both json and stream-json.
			args = append(args, "--json")
		}
		if req.Model != "" {
			args = append(args, "--model", req.Model)
		}
		if req.SkipPermissions {
			args = append(args, "--dangerously-bypass-approvals-and-sandbox")
		}
		// IDE workspaces are not always git repositories; codex exec refuses to
		// start in one unless told otherwise.
		args = append(args, "--skip-git-repo-check")
		args = append(args, req.ExtraArgs...)

	case "oc":
		// "message" is a variadic positional, so the prompt goes last to keep
		// the preceding flags unambiguously attached to their own values.
		args = append(args, "run")
		if isStructuredFormat(req.OutputFormat) {
			args = append(args, "--format", "json")
		}
		if req.Model != "" {
			args = append(args, "--model", req.Model)
		}
		if req.SkipPermissions {
			args = append(args, "--auto")
		}
		args = append(args, req.ExtraArgs...)
		args = append(args, req.Prompt)

	default:
		return nil, fmt.Errorf("tool %q does not support headless execution (supported: %s)", req.Tool, strings.Join(HeadlessTools, ", "))
	}

	return args, nil
}

// BuildHeadlessResourceAttributes appends headless markers to an
// OTEL_RESOURCE_ATTRIBUTES string produced by BuildResourceAttributes, so that
// automated spend can be separated from interactive developer spend.
func BuildHeadlessResourceAttributes(base, tool, outputFormat string) string {
	attrs := []string{
		"gantry.headless=true",
		"gantry.invocation=" + encodeAttrValue("exec"),
		"gantry.tool=" + encodeAttrValue(tool),
	}
	if outputFormat != "" {
		attrs = append(attrs, "gantry.output_format="+encodeAttrValue(outputFormat))
	}

	joined := strings.Join(attrs, ",")
	if base == "" {
		return joined
	}
	return base + "," + joined
}

// ClampHeadlessOTEL returns a copy of otel with the export intervals capped for
// a short-lived process. A zero interval is also clamped, because
// BuildEnvironment omits the variable entirely when the value is not positive,
// which would leave the tool on its own (longer) default.
func ClampHeadlessOTEL(otel config.OTELConfig) config.OTELConfig {
	clamped := otel

	if clamped.MetricExportInterval <= 0 || clamped.MetricExportInterval > HeadlessExportIntervalMs {
		clamped.MetricExportInterval = HeadlessExportIntervalMs
	}
	if clamped.LogsExportInterval <= 0 || clamped.LogsExportInterval > HeadlessExportIntervalMs {
		clamped.LogsExportInterval = HeadlessExportIntervalMs
	}

	return clamped
}

// headlessStripEnv are markers that indicate the parent process is itself an AI
// tool session. Inheriting them into a headless child misreports the session.
var headlessStripEnv = []string{
	"CLAUDECODE",
	"CLAUDE_CODE_ENTRYPOINT",
	"CLAUDE_CODE_SSE_PORT",
	"GANTRY_SHELL",
}

// SanitizeHeadlessEnv removes inherited parent-session markers.
func SanitizeHeadlessEnv(env []string) []string {
	return filterEnvVars(env, headlessStripEnv)
}

// IsRunningAsRoot reports whether the current process is root. Geteuid returns
// -1 on Windows, so this is false there.
func IsRunningAsRoot() bool {
	return os.Geteuid() == 0
}

// RunHeadless spawns the tool and returns its exit code.
//
// Unlike the interactive Launch* helpers it never calls os.Exit, so callers stay
// testable and own the exit contract. A nil stdin gives the child /dev/null,
// which guarantees gantry cannot consume a piped prompt and normalizes behavior
// across tools that would otherwise treat inherited stdin as extra input.
func RunHeadless(command string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return 1, err
	}

	// Forward cancellation to the child. Without this an IDE's cancel button
	// kills gantry and orphans the tool, which keeps consuming tokens.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	done := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-sigCh:
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()

	err := cmd.Wait()
	close(done)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}

	return 0, nil
}
