package launcher

import (
	"runtime"
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
