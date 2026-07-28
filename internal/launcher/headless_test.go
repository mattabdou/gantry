package launcher

import (
	"bytes"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/mattabdou/gantry/internal/config"
)

func TestIsHeadlessTool(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		expected bool
	}{
		{"claude code", "cc", true},
		{"codex", "co", true},
		{"opencode terminal", "oc", true},
		{"opencode desktop has no headless mode", "ocd", false},
		{"cline", "cl", false},
		{"cline kanban", "clk", false},
		{"cline plugin", "clp", false},
		{"unknown", "zz", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHeadlessTool(tt.tool); got != tt.expected {
				t.Errorf("IsHeadlessTool(%q) = %v, want %v", tt.tool, got, tt.expected)
			}
		})
	}
}

func TestHeadlessCommand(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		expected string
		wantErr  bool
	}{
		{"codex", "co", "codex", false},
		{"opencode", "oc", "opencode", false},
		{"unsupported tool", "ocd", "", true},
		{"unknown tool", "nope", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HeadlessCommand(tt.tool)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HeadlessCommand(%q) error = %v, wantErr %v", tt.tool, err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("HeadlessCommand(%q) = %q, want %q", tt.tool, got, tt.expected)
			}
		})
	}

	// Claude Code delegates to GetClaudeCommand for the Windows .cmd/.exe dance.
	t.Run("claude code matches GetClaudeCommand", func(t *testing.T) {
		got, err := HeadlessCommand("cc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != GetClaudeCommand() {
			t.Errorf("HeadlessCommand(\"cc\") = %q, want %q", got, GetClaudeCommand())
		}
	})
}

func TestBuildHeadlessArgs(t *testing.T) {
	tests := []struct {
		name     string
		req      HeadlessRequest
		expected []string
	}{
		{
			name:     "claude code text with bypass",
			req:      HeadlessRequest{Tool: "cc", Prompt: "fix it", OutputFormat: "text", SkipPermissions: true},
			expected: []string{"-p", "fix it", "--dangerously-skip-permissions"},
		},
		{
			name:     "claude code empty format behaves like text",
			req:      HeadlessRequest{Tool: "cc", Prompt: "fix it", SkipPermissions: true},
			expected: []string{"-p", "fix it", "--dangerously-skip-permissions"},
		},
		{
			name:     "claude code json",
			req:      HeadlessRequest{Tool: "cc", Prompt: "fix it", OutputFormat: "json", SkipPermissions: true},
			expected: []string{"-p", "fix it", "--output-format", "json", "--dangerously-skip-permissions"},
		},
		{
			// Claude Code hard-errors on stream-json without --verbose.
			name:     "claude code stream-json adds verbose",
			req:      HeadlessRequest{Tool: "cc", Prompt: "fix it", OutputFormat: "stream-json", SkipPermissions: true},
			expected: []string{"-p", "fix it", "--output-format", "stream-json", "--verbose", "--dangerously-skip-permissions"},
		},
		{
			name:     "claude code without bypass omits the flag",
			req:      HeadlessRequest{Tool: "cc", Prompt: "fix it", OutputFormat: "text", SkipPermissions: false},
			expected: []string{"-p", "fix it"},
		},
		{
			// The prompt must precede passthrough args so a variadic flag
			// such as --add-dir cannot swallow it.
			name:     "claude code prompt precedes extra args",
			req:      HeadlessRequest{Tool: "cc", Prompt: "fix it", OutputFormat: "text", Model: "opus", ExtraArgs: []string{"--add-dir", "/tmp"}},
			expected: []string{"-p", "fix it", "--model", "opus", "--add-dir", "/tmp"},
		},
		{
			name:     "codex text with bypass",
			req:      HeadlessRequest{Tool: "co", Prompt: "fix it", OutputFormat: "text", SkipPermissions: true},
			expected: []string{"--profile", "gantry", "exec", "fix it", "--dangerously-bypass-approvals-and-sandbox", "--skip-git-repo-check"},
		},
		{
			name:     "codex json without bypass",
			req:      HeadlessRequest{Tool: "co", Prompt: "fix it", OutputFormat: "json"},
			expected: []string{"--profile", "gantry", "exec", "fix it", "--json", "--skip-git-repo-check"},
		},
		{
			name:     "codex maps stream-json to jsonl",
			req:      HeadlessRequest{Tool: "co", Prompt: "fix it", OutputFormat: "stream-json"},
			expected: []string{"--profile", "gantry", "exec", "fix it", "--json", "--skip-git-repo-check"},
		},
		{
			name:     "codex with model and extras",
			req:      HeadlessRequest{Tool: "co", Prompt: "fix it", Model: "gpt-5.6-terra", SkipPermissions: true, ExtraArgs: []string{"-c", "foo=1"}},
			expected: []string{"--profile", "gantry", "exec", "fix it", "--model", "gpt-5.6-terra", "--dangerously-bypass-approvals-and-sandbox", "--skip-git-repo-check", "-c", "foo=1"},
		},
		{
			// message is a variadic positional, so the prompt goes last.
			name:     "opencode text with bypass",
			req:      HeadlessRequest{Tool: "oc", Prompt: "fix it", OutputFormat: "text", SkipPermissions: true},
			expected: []string{"run", "--auto", "fix it"},
		},
		{
			name:     "opencode json with model and extras",
			req:      HeadlessRequest{Tool: "oc", Prompt: "fix it", OutputFormat: "json", Model: "gantry-litellm/claude-opus-4-6", SkipPermissions: true, ExtraArgs: []string{"--agent", "build"}},
			expected: []string{"run", "--format", "json", "--model", "gantry-litellm/claude-opus-4-6", "--auto", "--agent", "build", "fix it"},
		},
		{
			name:     "opencode without bypass omits auto",
			req:      HeadlessRequest{Tool: "oc", Prompt: "fix it", OutputFormat: "text"},
			expected: []string{"run", "fix it"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildHeadlessArgs(tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("BuildHeadlessArgs() =\n  %#v\nwant\n  %#v", got, tt.expected)
			}
		})
	}
}

func TestBuildHeadlessArgsErrors(t *testing.T) {
	tests := []struct {
		name string
		req  HeadlessRequest
	}{
		{"empty prompt", HeadlessRequest{Tool: "cc", Prompt: ""}},
		{"whitespace only prompt", HeadlessRequest{Tool: "cc", Prompt: "   \n\t "}},
		{"oversized prompt", HeadlessRequest{Tool: "cc", Prompt: strings.Repeat("x", MaxHeadlessPromptBytes+1)}},
		{"invalid output format", HeadlessRequest{Tool: "cc", Prompt: "hi", OutputFormat: "yaml"}},
		{"unsupported tool ocd", HeadlessRequest{Tool: "ocd", Prompt: "hi"}},
		{"unsupported tool cl", HeadlessRequest{Tool: "cl", Prompt: "hi"}},
		{"unknown tool", HeadlessRequest{Tool: "zz", Prompt: "hi"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := BuildHeadlessArgs(tt.req); err == nil {
				t.Errorf("BuildHeadlessArgs(%+v) expected an error, got nil", tt.req)
			}
		})
	}
}

func TestBuildHeadlessResourceAttributes(t *testing.T) {
	tests := []struct {
		name         string
		base         string
		tool         string
		outputFormat string
		expected     string
	}{
		{
			name:         "appends to an existing base",
			base:         "gantry.username=alice,gantry.project_name=demo",
			tool:         "cc",
			outputFormat: "json",
			expected:     "gantry.username=alice,gantry.project_name=demo,gantry.headless=true,gantry.invocation=exec,gantry.tool=cc,gantry.output_format=json",
		},
		{
			name:         "empty base has no leading comma",
			base:         "",
			tool:         "co",
			outputFormat: "text",
			expected:     "gantry.headless=true,gantry.invocation=exec,gantry.tool=co,gantry.output_format=text",
		},
		{
			name:         "empty output format is omitted",
			base:         "gantry.username=bob",
			tool:         "oc",
			outputFormat: "",
			expected:     "gantry.username=bob,gantry.headless=true,gantry.invocation=exec,gantry.tool=oc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildHeadlessResourceAttributes(tt.base, tt.tool, tt.outputFormat)
			if got != tt.expected {
				t.Errorf("BuildHeadlessResourceAttributes() =\n  %q\nwant\n  %q", got, tt.expected)
			}
		})
	}
}

func TestClampHeadlessOTEL(t *testing.T) {
	tests := []struct {
		name           string
		metricInterval int
		logsInterval   int
		wantMetric     int
		wantLogs       int
	}{
		{"default 60s metric interval is clamped", 60000, 5000, HeadlessExportIntervalMs, HeadlessExportIntervalMs},
		{"zero is clamped because BuildEnvironment would omit it", 0, 0, HeadlessExportIntervalMs, HeadlessExportIntervalMs},
		{"negative is clamped", -1, -1, HeadlessExportIntervalMs, HeadlessExportIntervalMs},
		{"already short intervals are preserved", 1000, 250, 1000, 250},
		{"exactly at the cap is preserved", HeadlessExportIntervalMs, HeadlessExportIntervalMs, HeadlessExportIntervalMs, HeadlessExportIntervalMs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClampHeadlessOTEL(config.OTELConfig{
				MetricExportInterval: tt.metricInterval,
				LogsExportInterval:   tt.logsInterval,
			})
			if got.MetricExportInterval != tt.wantMetric {
				t.Errorf("MetricExportInterval = %d, want %d", got.MetricExportInterval, tt.wantMetric)
			}
			if got.LogsExportInterval != tt.wantLogs {
				t.Errorf("LogsExportInterval = %d, want %d", got.LogsExportInterval, tt.wantLogs)
			}
		})
	}

	t.Run("other fields are untouched and the input is not mutated", func(t *testing.T) {
		original := config.OTELConfig{
			Endpoint:             "https://collector.example.com",
			Headers:              "Authorization=Bearer token",
			Protocol:             "http/protobuf",
			MetricsExporter:      "otlp",
			MetricExportInterval: 60000,
		}
		got := ClampHeadlessOTEL(original)

		if original.MetricExportInterval != 60000 {
			t.Errorf("input was mutated: MetricExportInterval = %d, want 60000", original.MetricExportInterval)
		}
		if got.Endpoint != original.Endpoint {
			t.Errorf("Endpoint = %q, want %q", got.Endpoint, original.Endpoint)
		}
		if got.Headers != original.Headers {
			t.Errorf("Headers = %q, want %q", got.Headers, original.Headers)
		}
		if got.Protocol != original.Protocol {
			t.Errorf("Protocol = %q, want %q", got.Protocol, original.Protocol)
		}
		if got.MetricsExporter != original.MetricsExporter {
			t.Errorf("MetricsExporter = %q, want %q", got.MetricsExporter, original.MetricsExporter)
		}
	})
}

func TestSanitizeHeadlessEnv(t *testing.T) {
	input := []string{
		"PATH=/usr/bin",
		"CLAUDECODE=1",
		"HOME=/home/alice",
		"CLAUDE_CODE_ENTRYPOINT=cli",
		"GANTRY_SHELL=1",
		"CLAUDE_CODE_SSE_PORT=1234",
		"OTEL_RESOURCE_ATTRIBUTES=gantry.username=alice",
	}
	expected := []string{
		"PATH=/usr/bin",
		"HOME=/home/alice",
		"OTEL_RESOURCE_ATTRIBUTES=gantry.username=alice",
	}

	got := SanitizeHeadlessEnv(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("SanitizeHeadlessEnv() =\n  %#v\nwant\n  %#v", got, expected)
	}
}

func TestIsRunningAsRoot(t *testing.T) {
	// Assert self-consistency rather than a fixed value, since the test may run
	// as any user. Geteuid returns -1 on Windows, so this is false there.
	expected := os.Geteuid() == 0
	if got := IsRunningAsRoot(); got != expected {
		t.Errorf("IsRunningAsRoot() = %v, want %v", got, expected)
	}
}

// TestRunHeadlessEnvPlumbing spawns a real process to confirm that the headless
// environment actually reaches the child, and that stdout is wired to the
// writer the caller supplies.
func TestRunHeadlessEnvPlumbing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /usr/bin/env")
	}

	globalConfig := &config.GlobalConfig{
		Gantry: &config.GantryConfig{Username: "alice"},
		OTEL: config.OTELConfig{
			Endpoint:             "https://collector.example.com",
			MetricExportInterval: 60000,
		},
		LiteLLM: &config.LiteLLMConfig{
			BaseURL:   "https://proxy.example.com",
			AuthToken: "token",
			Model:     "claude-opus-4-6",
		},
	}

	attrs := BuildHeadlessResourceAttributes("gantry.username=alice", "cc", "json")

	clamped := *globalConfig
	clamped.OTEL = ClampHeadlessOTEL(globalConfig.OTEL)

	env := BuildEnvironment(&clamped, attrs, "litellm")
	env = SanitizeHeadlessEnv(append(env, "CLAUDECODE=1", "GANTRY_SHELL=1"))

	var stdout, stderr bytes.Buffer
	code, err := RunHeadless("/usr/bin/env", nil, env, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("RunHeadless() error = %v (stderr: %s)", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("RunHeadless() exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}

	got := stdout.String()

	wantSubstrings := []string{
		"CLAUDE_CODE_ENABLE_TELEMETRY=1",
		"OTEL_EXPORTER_OTLP_ENDPOINT=https://collector.example.com",
		"gantry.headless=true",
		"gantry.invocation=exec",
		"gantry.tool=cc",
		"gantry.output_format=json",
		// 60000 must have been clamped for the short-lived run.
		"OTEL_METRIC_EXPORT_INTERVAL=5000",
		"ANTHROPIC_BASE_URL=https://proxy.example.com",
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(got, want) {
			t.Errorf("child environment missing %q", want)
		}
	}

	// Parent-session markers must not leak into the child.
	for _, unwanted := range []string{"CLAUDECODE=1", "GANTRY_SHELL=1"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("child environment should not contain %q", unwanted)
		}
	}
}

// TestRunHeadlessExitCode confirms the child's exit status is returned rather
// than triggering os.Exit inside the launcher.
func TestRunHeadlessExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh")
	}

	var stdout, stderr bytes.Buffer
	code, err := RunHeadless("/bin/sh", []string{"-c", "exit 42"}, nil, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("RunHeadless() error = %v", err)
	}
	if code != 42 {
		t.Errorf("RunHeadless() exit code = %d, want 42", code)
	}
}

// TestRunHeadlessNilStdin confirms a nil stdin gives the child EOF rather than
// the parent's terminal, so gantry can never compete for a piped prompt.
func TestRunHeadlessNilStdin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh")
	}

	var stdout, stderr bytes.Buffer
	code, err := RunHeadless("/bin/sh", []string{"-c", "cat; echo done"}, nil, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("RunHeadless() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("RunHeadless() exit code = %d, want 0", code)
	}
	if strings.TrimSpace(stdout.String()) != "done" {
		t.Errorf("stdout = %q, want \"done\" (cat should read EOF immediately)", stdout.String())
	}
}
