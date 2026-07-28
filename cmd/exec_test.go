package cmd

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mattabdou/gantry/internal/config"
)

func TestSplitExecArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		dashPos    int
		wantPrompt []string
		wantExtras []string
	}{
		{
			name:       "no dash at all",
			args:       []string{"fix", "the", "test"},
			dashPos:    -1,
			wantPrompt: []string{"fix", "the", "test"},
			wantExtras: nil,
		},
		{
			// With interspersed parsing disabled pflag leaves the "--" in args.
			name:       "literal dash survives in args",
			args:       []string{"fix it", "--", "--add-dir", "/tmp"},
			dashPos:    -1,
			wantPrompt: []string{"fix it"},
			wantExtras: []string{"--add-dir", "/tmp"},
		},
		{
			// When cobra consumes the "--" it reports its position instead.
			name:       "cobra reported dash position",
			args:       []string{"fix it", "--add-dir", "/tmp"},
			dashPos:    1,
			wantPrompt: []string{"fix it"},
			wantExtras: []string{"--add-dir", "/tmp"},
		},
		{
			name:       "dash before any prompt yields no prompt",
			args:       []string{"-p", "x"},
			dashPos:    0,
			wantPrompt: nil,
			wantExtras: []string{"-p", "x"},
		},
		{
			name:       "trailing dash with nothing after",
			args:       []string{"fix it", "--"},
			dashPos:    -1,
			wantPrompt: []string{"fix it"},
			wantExtras: nil,
		},
		{
			name:       "no args",
			args:       nil,
			dashPos:    -1,
			wantPrompt: nil,
			wantExtras: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPrompt, gotExtras := splitExecArgs(tt.args, tt.dashPos)
			if !reflect.DeepEqual(gotPrompt, tt.wantPrompt) {
				t.Errorf("promptWords = %#v, want %#v", gotPrompt, tt.wantPrompt)
			}
			if !reflect.DeepEqual(gotExtras, tt.wantExtras) {
				t.Errorf("extras = %#v, want %#v", gotExtras, tt.wantExtras)
			}
		})
	}

	t.Run("returned slices do not alias the input", func(t *testing.T) {
		args := []string{"prompt", "--", "--flag"}
		prompt, extras := splitExecArgs(args, -1)

		if len(prompt) > 0 {
			prompt[0] = "mutated"
		}
		if len(extras) > 0 {
			extras[0] = "mutated"
		}

		if args[0] != "prompt" || args[2] != "--flag" {
			t.Errorf("input was mutated: %#v", args)
		}
	})
}

func TestResolvePrompt(t *testing.T) {
	tmpDir := t.TempDir()

	promptFile := filepath.Join(tmpDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("  fix the parser  \n"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	emptyFile := filepath.Join(tmpDir, "empty.md")
	if err := os.WriteFile(emptyFile, []byte("   \n"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	tests := []struct {
		name        string
		promptWords []string
		promptFile  string
		useStdin    bool
		stdin       string
		expected    string
		wantErr     bool
	}{
		{
			name:        "single word",
			promptWords: []string{"fix"},
			expected:    "fix",
		},
		{
			name:        "multiple words are joined",
			promptWords: []string{"fix", "the", "failing", "test"},
			expected:    "fix the failing test",
		},
		{
			name:        "quoted prompt arrives as one word",
			promptWords: []string{"fix the failing test"},
			expected:    "fix the failing test",
		},
		{
			name:       "prompt file is trimmed",
			promptFile: promptFile,
			expected:   "fix the parser",
		},
		{
			name:     "stdin is trimmed",
			useStdin: true,
			stdin:    "\nrefactor everything\n",
			expected: "refactor everything",
		},
		{
			name:    "no source at all",
			wantErr: true,
		},
		{
			name:        "argument and prompt file conflict",
			promptWords: []string{"fix"},
			promptFile:  promptFile,
			wantErr:     true,
		},
		{
			name:        "argument and stdin conflict",
			promptWords: []string{"fix"},
			useStdin:    true,
			stdin:       "other",
			wantErr:     true,
		},
		{
			name:       "prompt file and stdin conflict",
			promptFile: promptFile,
			useStdin:   true,
			stdin:      "other",
			wantErr:    true,
		},
		{
			name:        "whitespace only argument",
			promptWords: []string{"   "},
			wantErr:     true,
		},
		{
			name:       "empty prompt file",
			promptFile: emptyFile,
			wantErr:    true,
		},
		{
			name:     "empty stdin",
			useStdin: true,
			stdin:    "",
			wantErr:  true,
		},
		{
			name:       "missing prompt file",
			promptFile: filepath.Join(tmpDir, "nope.md"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdin io.Reader
			if tt.useStdin {
				stdin = strings.NewReader(tt.stdin)
			}

			got, err := resolvePrompt(tt.promptWords, tt.promptFile, tt.useStdin, stdin)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolvePrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("resolvePrompt() = %q, want %q", got, tt.expected)
			}
		})
	}

	t.Run("stdin requested but unavailable", func(t *testing.T) {
		if _, err := resolvePrompt(nil, "", true, nil); err == nil {
			t.Error("expected an error when --stdin is set with no reader")
		}
	})
}

func TestNormalizeOutputFormat(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected string
		wantErr  bool
	}{
		{"empty defaults to text", "", "text", false},
		{"text", "text", "text", false},
		{"json", "json", "json", false},
		{"stream-json", "stream-json", "stream-json", false},
		{"invalid", "yaml", "", true},
		{"wrong case is rejected", "JSON", "", true},
		{"underscore variant is rejected", "stream_json", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeOutputFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeOutputFormat(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("normalizeOutputFormat(%q) = %q, want %q", tt.format, got, tt.expected)
			}
		})
	}
}

func TestValidateExecTool(t *testing.T) {
	tests := []struct {
		name    string
		tool    string
		wantErr bool
	}{
		{"claude code", "cc", false},
		{"codex", "co", false},
		{"opencode terminal", "oc", false},
		{"opencode desktop", "ocd", true},
		{"cline", "cl", true},
		{"cline kanban", "clk", true},
		{"cline plugin", "clp", true},
		{"garbage", "nope", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExecTool(tt.tool)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateExecTool(%q) error = %v, wantErr %v", tt.tool, err, tt.wantErr)
			}
			// A valid-but-interactive-only tool should get the more specific message.
			if tt.wantErr && config.IsValidTool(tt.tool) {
				if !strings.Contains(err.Error(), "no non-interactive mode") {
					t.Errorf("expected a headless-specific message for %q, got: %v", tt.tool, err)
				}
			}
		})
	}
}

func TestResolveExecTool(t *testing.T) {
	tests := []struct {
		name         string
		globalConfig *config.GlobalConfig
		toolFlag     string
		wantTool     string
		wantSource   string
	}{
		{
			name:         "flag wins over config",
			globalConfig: &config.GlobalConfig{Gantry: &config.GantryConfig{DefaultTool: "oc"}},
			toolFlag:     "cc",
			wantTool:     "cc",
			wantSource:   "flag",
		},
		{
			name:         "config is used when no flag",
			globalConfig: &config.GlobalConfig{Gantry: &config.GantryConfig{DefaultTool: "co"}},
			wantTool:     "co",
			wantSource:   "config",
		},
		{
			name:         "defaults to claude code",
			globalConfig: &config.GlobalConfig{Gantry: &config.GantryConfig{}},
			wantTool:     "cc",
			wantSource:   "default",
		},
		{
			name:         "nil gantry section defaults to claude code",
			globalConfig: &config.GlobalConfig{},
			wantTool:     "cc",
			wantSource:   "default",
		},
		{
			name:       "nil config defaults to claude code",
			wantTool:   "cc",
			wantSource: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTool, gotSource := resolveExecTool(tt.globalConfig, tt.toolFlag)
			if gotTool != tt.wantTool {
				t.Errorf("tool = %q, want %q", gotTool, tt.wantTool)
			}
			if gotSource != tt.wantSource {
				t.Errorf("source = %q, want %q", gotSource, tt.wantSource)
			}
		})
	}
}

func TestResolveExecMode(t *testing.T) {
	tests := []struct {
		name         string
		globalConfig *config.GlobalConfig
		modeFlag     string
		wantMode     string
		wantSource   string
		wantErr      bool
	}{
		{
			name:         "flag wins over config",
			globalConfig: &config.GlobalConfig{Gantry: &config.GantryConfig{Mode: "bedrock"}},
			modeFlag:     "litellm",
			wantMode:     "litellm",
			wantSource:   "flag",
		},
		{
			name:         "config is used when no flag",
			globalConfig: &config.GlobalConfig{Gantry: &config.GantryConfig{Mode: "bedrock"}},
			wantMode:     "bedrock",
			wantSource:   "config",
		},
		{
			name:         "missing mode is an error",
			globalConfig: &config.GlobalConfig{Gantry: &config.GantryConfig{}},
			wantErr:      true,
		},
		{
			name:         "invalid mode is an error",
			globalConfig: &config.GlobalConfig{Gantry: &config.GantryConfig{Mode: "vertex"}},
			wantErr:      true,
		},
		{
			name:     "invalid flag mode is an error",
			modeFlag: "nope",
			wantErr:  true,
		},
		{
			name:    "nil config is an error",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, gotSource, err := resolveExecMode(tt.globalConfig, tt.modeFlag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveExecMode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotMode != tt.wantMode {
				t.Errorf("mode = %q, want %q", gotMode, tt.wantMode)
			}
			if gotSource != tt.wantSource {
				t.Errorf("source = %q, want %q", gotSource, tt.wantSource)
			}
		})
	}
}

func TestExecToolRequiresLiteLLM(t *testing.T) {
	tests := []struct {
		tool     string
		expected bool
	}{
		{"co", true},
		{"cc", false},
		{"oc", false},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			if got := execToolRequiresLiteLLM(tt.tool); got != tt.expected {
				t.Errorf("execToolRequiresLiteLLM(%q) = %v, want %v", tt.tool, got, tt.expected)
			}
		})
	}
}

func TestValidateExecProviderConfig(t *testing.T) {
	tests := []struct {
		name         string
		globalConfig *config.GlobalConfig
		mode         string
		wantErr      bool
	}{
		{
			name: "complete bedrock config",
			globalConfig: &config.GlobalConfig{
				Bedrock: &config.BedrockConfig{AWSProfile: "dev", AWSRegion: "us-east-2"},
			},
			mode: "bedrock",
		},
		{
			name:         "missing bedrock section",
			globalConfig: &config.GlobalConfig{},
			mode:         "bedrock",
			wantErr:      true,
		},
		{
			name: "bedrock without profile",
			globalConfig: &config.GlobalConfig{
				Bedrock: &config.BedrockConfig{AWSRegion: "us-east-2"},
			},
			mode:    "bedrock",
			wantErr: true,
		},
		{
			name: "bedrock without region",
			globalConfig: &config.GlobalConfig{
				Bedrock: &config.BedrockConfig{AWSProfile: "dev"},
			},
			mode:    "bedrock",
			wantErr: true,
		},
		{
			name: "complete litellm config",
			globalConfig: &config.GlobalConfig{
				LiteLLM: &config.LiteLLMConfig{BaseURL: "https://proxy.example.com", AuthToken: "token"},
			},
			mode: "litellm",
		},
		{
			name:         "missing litellm section",
			globalConfig: &config.GlobalConfig{},
			mode:         "litellm",
			wantErr:      true,
		},
		{
			name: "litellm without base url",
			globalConfig: &config.GlobalConfig{
				LiteLLM: &config.LiteLLMConfig{AuthToken: "token"},
			},
			mode:    "litellm",
			wantErr: true,
		},
		{
			name: "litellm without auth token",
			globalConfig: &config.GlobalConfig{
				LiteLLM: &config.LiteLLMConfig{BaseURL: "https://proxy.example.com"},
			},
			mode:    "litellm",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExecProviderConfig(tt.globalConfig, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateExecProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveBypass(t *testing.T) {
	tests := []struct {
		name         string
		configAllows bool
		noSkipFlag   bool
		expected     bool
	}{
		{"allowed and not opted out", true, false, true},
		{"allowed but opted out per run", true, true, false},
		{"disallowed by config", false, false, false},
		{"disallowed and opted out", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveBypass(tt.configAllows, tt.noSkipFlag); got != tt.expected {
				t.Errorf("resolveBypass(%v, %v) = %v, want %v", tt.configAllows, tt.noSkipFlag, got, tt.expected)
			}
		})
	}
}

func TestExecFlagsRegistered(t *testing.T) {
	expected := []string{
		"tool", "mode", "output-format", "model", "prompt-file",
		"stdin", "no-skip-permissions", "no-configure", "verbose", "print-command",
	}

	for _, name := range expected {
		if execCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s is not registered on execCmd", name)
		}
	}

	if execCmd.Name() != "exec" {
		t.Errorf("execCmd.Name() = %q, want \"exec\"", execCmd.Name())
	}
}

// TestExecFlagParsingStopsAtPrompt pins the IDE-facing contract: gantry's own
// flags must precede the prompt, and anything after it is left for the tool.
func TestExecFlagParsingStopsAtPrompt(t *testing.T) {
	origTool, origFormat := execTool, execOutputFormat
	t.Cleanup(func() {
		execTool, execOutputFormat = origTool, origFormat
	})

	fs := execCmd.Flags()
	if err := fs.Parse([]string{"-t", "cc", "fix it", "--output-format", "json"}); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if execTool != "cc" {
		t.Errorf("execTool = %q, want \"cc\" (flags before the prompt are gantry's)", execTool)
	}
	// The trailing --output-format belongs to the tool, not to gantry.
	if execOutputFormat != origFormat {
		t.Errorf("execOutputFormat = %q, want %q (flags after the prompt belong to the tool)", execOutputFormat, origFormat)
	}

	want := []string{"fix it", "--output-format", "json"}
	if got := fs.Args(); !reflect.DeepEqual(got, want) {
		t.Errorf("remaining args = %#v, want %#v", got, want)
	}
}

// TestExecStdoutPurity enforces the invariant that makes headless mode usable:
// stdout carries only the tool's output, so an IDE can parse it. Nothing else
// in the test suite can catch a stray fmt.Println here.
func TestExecStdoutPurity(t *testing.T) {
	data, err := os.ReadFile("exec.go")
	if err != nil {
		t.Fatalf("failed to read exec.go: %v", err)
	}
	src := string(data)

	forbidden := []string{
		"fmt.Print(",
		"fmt.Println(",
		"fmt.Printf(",
		"os.Stdout.Write",
		"bufio.NewReader",
	}
	for _, pattern := range forbidden {
		if strings.Contains(src, pattern) {
			t.Errorf("exec.go must not contain %q: headless stdout is reserved for the tool's output", pattern)
		}
	}

	// os.Stdout may appear exactly once, as the RunHeadless writer.
	if n := strings.Count(src, "os.Stdout"); n != 1 {
		t.Errorf("exec.go references os.Stdout %d times, want exactly 1 (the RunHeadless writer)", n)
	}

	// os.Stdin may appear exactly once, behind the --stdin flag.
	if n := strings.Count(src, "os.Stdin"); n != 1 {
		t.Errorf("exec.go references os.Stdin %d times, want exactly 1 (guarded by --stdin)", n)
	}
}
