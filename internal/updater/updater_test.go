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

	result := CheckForUpdate("0.0.1")

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
}
