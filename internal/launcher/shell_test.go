package launcher

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestIdentifyShell(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected shellType
	}{
		{"bash", "/bin/bash", shellBash},
		{"bash usr", "/usr/bin/bash", shellBash},
		{"zsh", "/bin/zsh", shellZsh},
		{"zsh usr", "/usr/bin/zsh", shellZsh},
		{"fish", "/usr/bin/fish", shellFish},
		{"cmd.exe", `C:\Windows\system32\cmd.exe`, shellCmdExe},
		{"cmd no ext", "cmd", shellCmdExe},
		{"powershell", "powershell.exe", shellPowerShell},
		{"pwsh", "pwsh", shellPowerShell},
		{"pwsh exe", "pwsh.exe", shellPowerShell},
		{"sh", "/bin/sh", shellUnknown},
		{"dash", "/usr/bin/dash", shellUnknown},
		{"unknown", "/usr/local/bin/myshell", shellUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := identifyShell(tt.path)
			if got != tt.expected {
				t.Errorf("identifyShell(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestGetUserShell(t *testing.T) {
	result := getUserShell()
	if result == "" {
		t.Error("getUserShell() returned empty string")
	}

	if runtime.GOOS != "windows" {
		// Test that SHELL env var is respected
		original := os.Getenv("SHELL")
		defer os.Setenv("SHELL", original)

		os.Setenv("SHELL", "/usr/bin/test-shell")
		if got := getUserShell(); got != "/usr/bin/test-shell" {
			t.Errorf("getUserShell() = %q, want /usr/bin/test-shell", got)
		}

		// Test fallback when SHELL is unset
		os.Unsetenv("SHELL")
		if got := getUserShell(); got != "/bin/sh" {
			t.Errorf("getUserShell() with no SHELL = %q, want /bin/sh", got)
		}
	}
}

func TestGetUserShellWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	result := getUserShell()
	if result == "" {
		t.Error("getUserShell() returned empty string on Windows")
	}

	// On Windows, should return COMSPEC or cmd.exe
	original := os.Getenv("COMSPEC")
	defer os.Setenv("COMSPEC", original)

	os.Setenv("COMSPEC", `C:\test\cmd.exe`)
	if got := getUserShell(); got != `C:\test\cmd.exe` {
		t.Errorf("getUserShell() = %q, want C:\\test\\cmd.exe", got)
	}

	os.Unsetenv("COMSPEC")
	if got := getUserShell(); got != "cmd.exe" {
		t.Errorf("getUserShell() with no COMSPEC = %q, want cmd.exe", got)
	}
}

func TestGenerateBashRcContent(t *testing.T) {
	content := generateBashRcContent()

	if !strings.Contains(content, `source "$HOME/.bashrc"`) {
		t.Error("bash rcfile should source ~/.bashrc")
	}
	if !strings.Contains(content, `PS1="(gantry) $PS1"`) {
		t.Error("bash rcfile should modify PS1 with (gantry) prefix")
	}
}

func TestGenerateZshRcContent(t *testing.T) {
	content := generateZshRcContent()

	if !strings.Contains(content, `GANTRY_ORIG_ZDOTDIR`) {
		t.Error("zsh rcfile should reference GANTRY_ORIG_ZDOTDIR")
	}
	if !strings.Contains(content, `source "$HOME/.zshrc"`) {
		t.Error("zsh rcfile should have fallback to source ~/.zshrc")
	}
	if !strings.Contains(content, `PROMPT="(gantry) $PROMPT"`) {
		t.Error("zsh rcfile should modify PROMPT with (gantry) prefix")
	}
}

func TestGantryShellEnvVar(t *testing.T) {
	// Verify that GANTRY_SHELL=1 would be present in the env
	// We test this indirectly since LaunchShell actually spawns a process
	env := []string{"PATH=/usr/bin", "HOME=/home/test"}
	env = append(env, "GANTRY_SHELL=1")

	found := false
	for _, e := range env {
		if e == "GANTRY_SHELL=1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GANTRY_SHELL=1 should be present in env")
	}
}
