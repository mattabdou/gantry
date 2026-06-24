package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ToolDetectionResult contains the result of checking for a tool installation
type ToolDetectionResult struct {
	Installed bool
	Path      string
	Error     error
}

// DetectClaudeCode checks if Claude Code is installed
func DetectClaudeCode() ToolDetectionResult {
	claudeCmd := GetClaudeCommand()
	path, err := exec.LookPath(claudeCmd)
	if err != nil {
		return ToolDetectionResult{
			Installed: false,
			Error:     err,
		}
	}
	return ToolDetectionResult{
		Installed: true,
		Path:      path,
	}
}

// DetectOpenCodeTerminal checks if OpenCode Terminal (CLI) is installed
func DetectOpenCodeTerminal() ToolDetectionResult {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return ToolDetectionResult{
			Installed: false,
			Error:     err,
		}
	}
	return ToolDetectionResult{
		Installed: true,
		Path:      path,
	}
}

// DetectOpenCodeDesktop checks if OpenCode Desktop is installed
func DetectOpenCodeDesktop() ToolDetectionResult {
	switch runtime.GOOS {
	case "darwin":
		return detectOpenCodeDesktopMacOS()
	case "windows":
		return detectOpenCodeDesktopWindows()
	case "linux":
		return detectOpenCodeDesktopLinux()
	default:
		return ToolDetectionResult{
			Installed: false,
		}
	}
}

// detectOpenCodeDesktopMacOS checks for OpenCode Desktop on macOS
func detectOpenCodeDesktopMacOS() ToolDetectionResult {
	// Check standard application locations
	paths := []string{
		"/Applications/OpenCode.app",
	}

	// Also check user's Applications folder
	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(home, "Applications", "OpenCode.app"))
	}

	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return ToolDetectionResult{
				Installed: true,
				Path:      path,
			}
		}
	}

	return ToolDetectionResult{
		Installed: false,
	}
}

// detectOpenCodeDesktopWindows checks for OpenCode Desktop on Windows
func detectOpenCodeDesktopWindows() ToolDetectionResult {
	var paths []string

	// Check common installation locations
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		paths = append(paths,
			filepath.Join(localAppData, "Programs", "OpenCode", "OpenCode.exe"),
			filepath.Join(localAppData, "opencode-desktop", "OpenCode.exe"),
		)
	}

	programFiles := os.Getenv("PROGRAMFILES")
	if programFiles != "" {
		paths = append(paths, filepath.Join(programFiles, "OpenCode", "OpenCode.exe"))
	}

	programFilesX86 := os.Getenv("PROGRAMFILES(X86)")
	if programFilesX86 != "" {
		paths = append(paths, filepath.Join(programFilesX86, "OpenCode", "OpenCode.exe"))
	}

	// Check Scoop installation
	userProfile := os.Getenv("USERPROFILE")
	if userProfile != "" {
		paths = append(paths, filepath.Join(userProfile, "scoop", "apps", "opencode-desktop", "current", "OpenCode.exe"))
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return ToolDetectionResult{
				Installed: true,
				Path:      path,
			}
		}
	}

	return ToolDetectionResult{
		Installed: false,
	}
}

// detectOpenCodeDesktopLinux checks for OpenCode Desktop on Linux
func detectOpenCodeDesktopLinux() ToolDetectionResult {
	// Check for the desktop binary in common locations
	paths := []string{
		"/usr/bin/opencode-desktop",
		"/usr/local/bin/opencode-desktop",
		"/opt/opencode-desktop/opencode-desktop",
		"/opt/OpenCode/opencode-desktop",
	}

	// Check if installed via snap
	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(home, "snap", "opencode-desktop", "current", "opencode-desktop"))
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return ToolDetectionResult{
				Installed: true,
				Path:      path,
			}
		}
	}

	// Also check for .desktop file as an indicator
	desktopFiles := []string{
		"/usr/share/applications/opencode-desktop.desktop",
		"/usr/share/applications/OpenCode.desktop",
	}
	if home != "" {
		desktopFiles = append(desktopFiles, filepath.Join(home, ".local", "share", "applications", "opencode-desktop.desktop"))
	}

	for _, path := range desktopFiles {
		if _, err := os.Stat(path); err == nil {
			return ToolDetectionResult{
				Installed: true,
				Path:      path,
			}
		}
	}

	return ToolDetectionResult{
		Installed: false,
	}
}

// DetectCline checks if Cline CLI is installed
func DetectCline() ToolDetectionResult {
	path, err := exec.LookPath("cline")
	if err != nil {
		return ToolDetectionResult{
			Installed: false,
			Error:     err,
		}
	}
	return ToolDetectionResult{
		Installed: true,
		Path:      path,
	}
}

// GetOpenCodeDesktopLaunchCommand returns the command to launch OpenCode Desktop
func GetOpenCodeDesktopLaunchCommand() (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{"-a", "OpenCode"}
	case "windows":
		result := DetectOpenCodeDesktop()
		if result.Installed && result.Path != "" {
			return result.Path, nil
		}
		return "OpenCode.exe", nil
	case "linux":
		result := DetectOpenCodeDesktop()
		if result.Installed && result.Path != "" {
			// If we found a .desktop file, use xdg-open
			if filepath.Ext(result.Path) == ".desktop" {
				return "xdg-open", []string{result.Path}
			}
			return result.Path, nil
		}
		return "opencode-desktop", nil
	default:
		return "opencode-desktop", nil
	}
}
