package updater

import (
	"runtime"
	"testing"
)

func TestGetCurrentOS(t *testing.T) {
	os := GetCurrentOS()

	validOS := map[string]bool{
		"darwin":  true,
		"linux":   true,
		"windows": true,
	}

	if !validOS[os] && os != runtime.GOOS {
		t.Errorf("GetCurrentOS() = %v, expected valid OS", os)
	}
}

func TestGetCurrentArch(t *testing.T) {
	arch := GetCurrentArch()

	validArch := map[string]bool{
		"amd64": true,
		"arm64": true,
	}

	if !validArch[arch] && arch != runtime.GOARCH {
		t.Errorf("GetCurrentArch() = %v, expected valid architecture", arch)
	}
}

func TestGetBinaryName(t *testing.T) {
	name := GetBinaryName()

	// Should contain OS and arch
	os := GetCurrentOS()
	arch := GetCurrentArch()

	expectedPrefix := "gantry-" + os + "-" + arch
	if runtime.GOOS == "windows" {
		expectedPrefix += ".exe"
	}

	if name != expectedPrefix {
		t.Errorf("GetBinaryName() = %v, want %v", name, expectedPrefix)
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		tag  string
		want string
	}{
		{"v1.0.0", "1.0.0"},
		{"v2.1.3", "2.1.3"},
		{"1.0.0", "1.0.0"},
		{"v0.0.1", "0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := ParseVersion(tt.tag)
			if got != tt.want {
				t.Errorf("ParseVersion(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1   string
		v2   string
		want int
	}{
		// Basic version comparisons
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "v1.0.0", 0},
		{"1.0", "1.0.0", 0},
		{"1.0.0", "1.0", 0},
		{"1.10.0", "1.9.0", 1},
		{"1.9.0", "1.10.0", -1},

		// Prerelease version comparisons
		{"1.0.0-beta.1", "1.0.0-beta.1", 0},
		{"1.0.0-beta.1", "1.0.0-beta.2", -1},
		{"1.0.0-beta.2", "1.0.0-beta.1", 1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-beta", "1.0.0-alpha", 1},

		// Prerelease vs stable (stable is always greater)
		{"1.0.0-beta.1", "1.0.0", -1},
		{"1.0.0", "1.0.0-beta.1", 1},
		{"1.0.0-beta.99", "1.0.0", -1},

		// Different base versions with prerelease
		{"1.0.0-beta.1", "1.0.1", -1},
		{"1.0.1-beta.1", "1.0.0", 1},
		{"2.0.0-beta.1", "1.9.9", 1},

		// Complex prerelease identifiers
		{"1.0.0-alpha.1", "1.0.0-alpha.2", -1},
		{"1.0.0-rc.1", "1.0.0-beta.1", 1}, // rc > beta alphabetically
	}

	for _, tt := range tests {
		t.Run(tt.v1+" vs "+tt.v2, func(t *testing.T) {
			got := CompareVersions(tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %v, want %v", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestIsPrerelease(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"1.0.0", false},
		{"1.0.0-beta.1", true},
		{"1.0.0-alpha", true},
		{"1.0.0-rc.1", true},
		{"v1.0.0", false},
		{"v1.0.0-beta.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := IsPrerelease(tt.version)
			if got != tt.want {
				t.Errorf("IsPrerelease(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestGetExecutablePath(t *testing.T) {
	path, err := GetExecutablePath()

	if err != nil {
		t.Fatalf("GetExecutablePath() error = %v", err)
	}

	if path == "" {
		t.Error("GetExecutablePath() returned empty path")
	}
}

func TestCheckForUpdate(t *testing.T) {
	// This test requires network access and may fail if GitHub is unavailable
	// or if there are no releases yet
	t.Skip("Skipping network-dependent test")

	result := CheckForUpdate("0.0.1", "stable")

	if result.Error != nil {
		t.Logf("CheckForUpdate() returned error (may be expected): %v", result.Error)
		return
	}

	if result.CurrentVersion != "0.0.1" {
		t.Errorf("CurrentVersion = %v, want 0.0.1", result.CurrentVersion)
	}

	if result.LatestVersion == "" {
		t.Error("LatestVersion is empty")
	}

	if result.Channel != "stable" {
		t.Errorf("Channel = %v, want stable", result.Channel)
	}
}
