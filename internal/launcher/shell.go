package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// shellType represents the type of shell detected
type shellType int

const (
	shellBash shellType = iota
	shellZsh
	shellFish
	shellCmdExe
	shellPowerShell
	shellUnknown
)

// getUserShell returns the path to the user's shell
func getUserShell() string {
	if runtime.GOOS == "windows" {
		if comspec := os.Getenv("COMSPEC"); comspec != "" {
			return comspec
		}
		return "cmd.exe"
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}

// identifyShell classifies a shell binary path into a shellType
func identifyShell(shellPath string) shellType {
	// Use filepath.Base, but also handle Windows backslash paths on non-Windows
	base := filepath.Base(shellPath)
	if i := strings.LastIndex(base, `\`); i >= 0 {
		base = base[i+1:]
	}
	base = strings.ToLower(base)
	// Remove .exe suffix for Windows
	base = strings.TrimSuffix(base, ".exe")

	switch base {
	case "bash":
		return shellBash
	case "zsh":
		return shellZsh
	case "fish":
		return shellFish
	case "cmd":
		return shellCmdExe
	case "powershell", "pwsh":
		return shellPowerShell
	default:
		return shellUnknown
	}
}

// LaunchShell spawns the user's shell with the configured environment and a modified prompt
func LaunchShell(env []string, toolDisplayName string, toolCommand string) error {
	// Check for nested gantry shell
	for _, e := range env {
		if e == "GANTRY_SHELL=1" {
			fmt.Fprintln(os.Stderr, "Warning: you are already inside a gantry shell. Nesting is not recommended.")
		}
	}

	// Add GANTRY_SHELL marker
	env = append(env, "GANTRY_SHELL=1")

	// Print welcome message
	fmt.Printf("Shell mode: your environment is configured for %s.\n", toolDisplayName)
	fmt.Printf("Run '%s' when ready. Type 'exit' to leave.\n", toolCommand)
	fmt.Println()

	// Detect and launch shell
	shellPath := getUserShell()
	shellKind := identifyShell(shellPath)

	switch shellKind {
	case shellBash:
		return launchBash(shellPath, env, toolDisplayName)
	case shellZsh:
		return launchZsh(shellPath, env, toolDisplayName)
	case shellFish:
		return launchFish(shellPath, env, toolDisplayName)
	case shellCmdExe:
		return launchCmdExe(shellPath, env, toolDisplayName)
	case shellPowerShell:
		return launchPowerShell(shellPath, env, toolDisplayName)
	default:
		return launchUnknownShell(shellPath, env, toolDisplayName)
	}
}

// runShell executes a shell command with stdin/stdout/stderr passthrough
func runShell(cmd *exec.Cmd, env []string) error {
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

// launchBash starts bash with a temp rcfile that sources ~/.bashrc then modifies PS1
func launchBash(shellPath string, env []string, toolDisplayName string) error {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "gantry-shell-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	rcContent := generateBashRcContent(toolDisplayName)
	rcFile := filepath.Join(tmpDir, "bashrc")
	if err := os.WriteFile(rcFile, []byte(rcContent), 0600); err != nil {
		return fmt.Errorf("failed to write temp bashrc: %w", err)
	}

	cmd := exec.Command(shellPath, "--rcfile", rcFile)
	return runShell(cmd, env)
}

// generateBashRcContent returns the content for the temporary bash rcfile
func generateBashRcContent(toolDisplayName string) string {
	return fmt.Sprintf(`# Gantry shell mode
if [ -f "$HOME/.bashrc" ]; then
    source "$HOME/.bashrc"
fi
export PS1="(gantry - %s) $PS1"
`, toolDisplayName)
}

// launchZsh starts zsh with a temp ZDOTDIR that sources the original .zshrc then modifies PROMPT
func launchZsh(shellPath string, env []string, toolDisplayName string) error {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "gantry-shell-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	rcContent := generateZshRcContent(toolDisplayName)
	rcFile := filepath.Join(tmpDir, ".zshrc")
	if err := os.WriteFile(rcFile, []byte(rcContent), 0600); err != nil {
		return fmt.Errorf("failed to write temp zshrc: %w", err)
	}

	// Save original ZDOTDIR and set ours
	origZdotdir := ""
	for _, e := range env {
		if strings.HasPrefix(e, "ZDOTDIR=") {
			origZdotdir = strings.TrimPrefix(e, "ZDOTDIR=")
			break
		}
	}
	if origZdotdir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			origZdotdir = home
		}
	}

	env = append(env, "GANTRY_ORIG_ZDOTDIR="+origZdotdir)
	env = append(env, "ZDOTDIR="+tmpDir)

	cmd := exec.Command(shellPath)
	return runShell(cmd, env)
}

// generateZshRcContent returns the content for the temporary zsh rcfile
func generateZshRcContent(toolDisplayName string) string {
	return fmt.Sprintf(`# Gantry shell mode
if [ -n "$GANTRY_ORIG_ZDOTDIR" ] && [ -f "$GANTRY_ORIG_ZDOTDIR/.zshrc" ]; then
    source "$GANTRY_ORIG_ZDOTDIR/.zshrc"
elif [ -f "$HOME/.zshrc" ]; then
    source "$HOME/.zshrc"
fi
export PROMPT="(gantry - %s) $PROMPT"
`, toolDisplayName)
}

// launchFish starts fish with an init command that wraps fish_prompt
func launchFish(shellPath string, env []string, toolDisplayName string) error {
	initCmd := fmt.Sprintf(`functions -c fish_prompt _gantry_orig_prompt 2>/dev/null; function fish_prompt; echo -n "(gantry - %s) "; _gantry_orig_prompt; end`, toolDisplayName)
	cmd := exec.Command(shellPath, "--init-command", initCmd)
	return runShell(cmd, env)
}

// launchCmdExe starts cmd.exe with a modified PROMPT env var
func launchCmdExe(shellPath string, env []string, toolDisplayName string) error {
	env = append(env, fmt.Sprintf("PROMPT=(gantry - %s) $P$G", toolDisplayName))
	cmd := exec.Command(shellPath)
	return runShell(cmd, env)
}

// launchPowerShell starts PowerShell/pwsh with a custom prompt function
func launchPowerShell(shellPath string, env []string, toolDisplayName string) error {
	promptCmd := fmt.Sprintf(`function prompt { '(gantry - %s) ' + $(Get-Location).Path + '> ' }`, toolDisplayName)
	cmd := exec.Command(shellPath, "-NoExit", "-Command", promptCmd)
	return runShell(cmd, env)
}

// launchUnknownShell starts an unknown shell with PS1 set (best-effort)
func launchUnknownShell(shellPath string, env []string, toolDisplayName string) error {
	env = append(env, fmt.Sprintf("PS1=(gantry - %s) $ ", toolDisplayName))
	cmd := exec.Command(shellPath)
	return runShell(cmd, env)
}
