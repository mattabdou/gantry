package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// GitHubRepo is the repository path
	GitHubRepo = "mattabdou/gantry"
	// GitHubAPIURL is the base URL for GitHub API
	GitHubAPIURL = "https://api.github.com"
	// GitHubReleasesURL is the URL for releases
	GitHubReleasesURL = "https://github.com/" + GitHubRepo + "/releases/download"
)

// Release represents a GitHub release
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
	HTMLURL     string    `json:"html_url"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateResult contains the result of an update check or update
type UpdateResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseURL      string
	Channel         string
	Error           error
}

// GetCurrentOS returns the current operating system name for binary naming
func GetCurrentOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "darwin"
	case "linux":
		return "linux"
	case "windows":
		return "windows"
	default:
		return runtime.GOOS
	}
}

// GetCurrentArch returns the current architecture for binary naming
func GetCurrentArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return runtime.GOARCH
	}
}

// GetBinaryName returns the expected binary name for the current platform
func GetBinaryName() string {
	name := fmt.Sprintf("gantry-%s-%s", GetCurrentOS(), GetCurrentArch())
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// GetLatestRelease fetches the latest stable release information from GitHub
// This uses the /releases/latest endpoint which automatically skips pre-releases
func GetLatestRelease() (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", GitHubAPIURL, GitHubRepo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "gantry-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("no releases found")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// GetAllReleases fetches all releases from GitHub (including pre-releases)
func GetAllReleases() ([]Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases", GitHubAPIURL, GitHubRepo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "gantry-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("no releases found")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}

	return releases, nil
}

// GetLatestPreRelease fetches the latest pre-release from GitHub
func GetLatestPreRelease() (*Release, error) {
	releases, err := GetAllReleases()
	if err != nil {
		return nil, err
	}

	// Find the latest pre-release (they come sorted by date, newest first)
	for i := range releases {
		if releases[i].Prerelease && !releases[i].Draft {
			return &releases[i], nil
		}
	}

	return nil, fmt.Errorf("no pre-releases found")
}

// GetLatestReleaseForChannel fetches the latest release for the specified channel
func GetLatestReleaseForChannel(channel string) (*Release, error) {
	if channel == "beta" {
		return GetLatestPreRelease()
	}
	return GetLatestRelease()
}

// ParseVersion extracts version number from tag (e.g., "v1.0.0" -> "1.0.0")
func ParseVersion(tag string) string {
	return strings.TrimPrefix(tag, "v")
}

// splitVersionAndPrerelease splits a version into base version and prerelease suffix
// e.g., "1.2.0-beta.1" -> ("1.2.0", "beta.1")
// e.g., "1.2.0" -> ("1.2.0", "")
func splitVersionAndPrerelease(version string) (string, string) {
	version = strings.TrimPrefix(version, "v")
	parts := strings.SplitN(version, "-", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

// IsPrerelease checks if a version string contains a prerelease suffix
func IsPrerelease(version string) bool {
	_, prerelease := splitVersionAndPrerelease(version)
	return prerelease != ""
}

// comparePrerelease compares two prerelease strings
// Returns: -1 if p1 < p2, 0 if p1 == p2, 1 if p1 > p2
// Empty string (stable) is considered greater than any prerelease
func comparePrerelease(p1, p2 string) int {
	// Both empty means equal
	if p1 == "" && p2 == "" {
		return 0
	}
	// Empty (stable) > any prerelease
	if p1 == "" {
		return 1
	}
	if p2 == "" {
		return -1
	}

	// Compare prerelease identifiers
	// Split by dots and compare each part
	parts1 := strings.Split(p1, ".")
	parts2 := strings.Split(p2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var s1, s2 string
		if i < len(parts1) {
			s1 = parts1[i]
		}
		if i < len(parts2) {
			s2 = parts2[i]
		}

		// Try numeric comparison first
		var n1, n2 int
		cnt1, _ := fmt.Sscanf(s1, "%d", &n1)
		cnt2, _ := fmt.Sscanf(s2, "%d", &n2)
		isNum1 := cnt1 == 1
		isNum2 := cnt2 == 1

		if isNum1 && isNum2 {
			if n1 < n2 {
				return -1
			}
			if n1 > n2 {
				return 1
			}
		} else if isNum1 {
			// Numeric < string in semver
			return -1
		} else if isNum2 {
			return 1
		} else {
			// String comparison
			if s1 < s2 {
				return -1
			}
			if s1 > s2 {
				return 1
			}
		}
	}

	return 0
}

// CompareVersions compares two version strings following semver rules
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
// Handles prerelease versions: 1.0.0-beta.1 < 1.0.0-beta.2 < 1.0.0
func CompareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	if v1 == v2 {
		return 0
	}

	// Split into base version and prerelease
	base1, pre1 := splitVersionAndPrerelease(v1)
	base2, pre2 := splitVersionAndPrerelease(v2)

	// Compare base versions first
	parts1 := strings.Split(base1, ".")
	parts2 := strings.Split(base2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &n1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &n2)
		}

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	// Base versions are equal, compare prerelease
	return comparePrerelease(pre1, pre2)
}

// CheckForUpdate checks if an update is available for the specified channel
func CheckForUpdate(currentVersion string, channel string) *UpdateResult {
	result := &UpdateResult{
		CurrentVersion: currentVersion,
		Channel:        channel,
	}

	release, err := GetLatestReleaseForChannel(channel)
	if err != nil {
		result.Error = err
		return result
	}

	result.LatestVersion = ParseVersion(release.TagName)
	result.ReleaseURL = release.HTMLURL

	if CompareVersions(currentVersion, result.LatestVersion) < 0 {
		result.UpdateAvailable = true
	}

	return result
}

// GetExecutablePath returns the path to the current executable
func GetExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	return exe, nil
}

// DownloadRelease downloads the release binary for the current platform
func DownloadRelease(release *Release) (string, error) {
	binaryName := GetBinaryName()

	// Find the asset for current platform
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		// Try constructing URL directly
		downloadURL = fmt.Sprintf("%s/%s/%s", GitHubReleasesURL, release.TagName, binaryName)
	}

	// Download to temp file
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "gantry-update-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	// Copy download to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to save download: %w", err)
	}

	tmpFile.Close()

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
			os.Remove(tmpFile.Name())
			return "", fmt.Errorf("failed to make executable: %w", err)
		}
	}

	return tmpFile.Name(), nil
}

// ReplaceExecutable replaces the current executable with the new one
func ReplaceExecutable(newPath string) error {
	currentPath, err := GetExecutablePath()
	if err != nil {
		return err
	}

	// On Unix (including macOS), we can delete/rename a running executable.
	// The kernel keeps the file open until the process exits.
	// On Windows, we rename to .old since you can't delete a running executable.
	//
	// We use atomic rename for all platforms to avoid issues with:
	// - macOS code signing (kills process if signed binary is modified in-place)
	// - Partial writes if interrupted

	oldPath := currentPath + ".old"

	// Remove old backup if exists
	os.Remove(oldPath)

	// Rename current executable to .old
	if err := os.Rename(currentPath, oldPath); err != nil {
		// On Unix, if rename fails, try to remove the current file instead
		if runtime.GOOS != "windows" {
			if removeErr := os.Remove(currentPath); removeErr != nil {
				return fmt.Errorf("failed to remove current executable: %w", err)
			}
		} else {
			return fmt.Errorf("failed to backup current executable: %w", err)
		}
	}

	// Move new file to current location
	if err := os.Rename(newPath, currentPath); err != nil {
		// Rename failed, try copy as fallback
		newFile, err := os.Open(newPath)
		if err != nil {
			return fmt.Errorf("failed to open new file: %w", err)
		}
		defer newFile.Close()

		destFile, err := os.OpenFile(currentPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("failed to create destination file: %w", err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, newFile); err != nil {
			return fmt.Errorf("failed to copy new executable: %w", err)
		}

		os.Remove(newPath)
	}

	// Clean up old file
	// On Windows this may fail if still in use, which is fine
	os.Remove(oldPath)

	return nil
}

// Update performs the full update process for the specified channel
func Update(currentVersion string, channel string) error {
	fmt.Printf("Checking for updates (%s channel)...\n", channel)

	release, err := GetLatestReleaseForChannel(channel)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latestVersion := ParseVersion(release.TagName)

	if CompareVersions(currentVersion, latestVersion) >= 0 {
		fmt.Printf("Already up to date (v%s on %s channel)\n", currentVersion, channel)
		return nil
	}

	fmt.Printf("New version available: v%s -> v%s\n", currentVersion, latestVersion)
	fmt.Printf("Downloading %s...\n", GetBinaryName())

	tmpPath, err := DownloadRelease(release)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	fmt.Println("Installing update...")

	if err := ReplaceExecutable(tmpPath); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	fmt.Printf("Successfully updated to v%s (%s channel)!\n", latestVersion, channel)
	return nil
}
