package launcher

import (
	"runtime"
	"strings"
	"testing"
)

func TestDetectClaudeCode(t *testing.T) {
	// This test just verifies the function runs without panic
	// The actual result depends on the system state
	result := DetectClaudeCode()
	// Should return a valid result (either installed or not)
	if result.Installed && result.Path == "" {
		t.Error("If installed, path should be set")
	}
}

// The install hint must name Anthropic's native installer, not npm. npm still
// works but is documented only under advanced options and needs Node.js 22+, so
// it is the wrong thing to tell a user who just failed to launch.
func TestClaudeCodeInstallCommandFor(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{"darwin", "curl -fsSL https://claude.ai/install.sh | bash"},
		{"linux", "curl -fsSL https://claude.ai/install.sh | bash"},
		{"windows", "irm https://claude.ai/install.ps1 | iex"},
		{"freebsd", "curl -fsSL https://claude.ai/install.sh | bash"},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			got := claudeCodeInstallCommandFor(tt.goos)
			if got != tt.want {
				t.Errorf("claudeCodeInstallCommandFor(%q) = %q, want %q", tt.goos, got, tt.want)
			}
			if strings.Contains(got, "npm") {
				t.Errorf("install command still points at npm: %q", got)
			}
		})
	}
}

func TestClaudeCodeInstallCommandUsesHostPlatform(t *testing.T) {
	if got, want := ClaudeCodeInstallCommand(), claudeCodeInstallCommandFor(runtime.GOOS); got != want {
		t.Errorf("ClaudeCodeInstallCommand() = %q, want %q", got, want)
	}
}

func TestClaudeCodeInstallDocsURL(t *testing.T) {
	if !strings.HasPrefix(ClaudeCodeInstallDocsURL, "https://") {
		t.Errorf("ClaudeCodeInstallDocsURL = %q, want an https URL", ClaudeCodeInstallDocsURL)
	}
	if strings.Contains(ClaudeCodeInstallDocsURL, "npm") {
		t.Errorf("docs URL points at npm: %q", ClaudeCodeInstallDocsURL)
	}
}

func TestDetectOpenCodeTerminal(t *testing.T) {
	// This test just verifies the function runs without panic
	result := DetectOpenCodeTerminal()
	if result.Installed && result.Path == "" {
		t.Error("If installed, path should be set")
	}
}

func TestDetectOpenCodeDesktop(t *testing.T) {
	// This test just verifies the function runs without panic
	result := DetectOpenCodeDesktop()
	// Just ensure it doesn't panic and returns a valid struct
	_ = result.Installed
}

func TestGetOpenCodeDesktopLaunchCommand(t *testing.T) {
	cmd, args := GetOpenCodeDesktopLaunchCommand()

	switch runtime.GOOS {
	case "darwin":
		if cmd != "open" {
			t.Errorf("Expected 'open' on macOS, got %s", cmd)
		}
		if len(args) != 2 || args[0] != "-a" || args[1] != "OpenCode" {
			t.Errorf("Expected args [-a, OpenCode], got %v", args)
		}
	case "windows":
		// On Windows, the command should contain OpenCode
		if cmd == "" {
			t.Error("Expected non-empty command on Windows")
		}
	case "linux":
		// On Linux, command should be set
		if cmd == "" {
			t.Error("Expected non-empty command on Linux")
		}
	}
}
