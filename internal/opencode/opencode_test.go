package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "opencode")
	if dir != expected {
		t.Errorf("GetConfigDir() = %v, want %v", dir, expected)
	}
}

func TestStripJSONComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line comment",
			input:    `{"key": "value"} // comment`,
			expected: `{"key": "value"} `,
		},
		{
			name:     "multi-line comment",
			input:    `{"key": /* comment */ "value"}`,
			expected: `{"key":  "value"}`,
		},
		{
			name:     "no comments",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(stripJSONComments([]byte(tt.input)))
			if result != tt.expected {
				t.Errorf("stripJSONComments() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigExists(t *testing.T) {
	// This test just verifies the function runs without panic
	// The actual result depends on the system state
	_ = ConfigExists()
}

func TestGetProviderConfig(t *testing.T) {
	cfg := OpenCodeConfig{
		"provider": map[string]interface{}{
			"test-provider": map[string]interface{}{
				"name": "Test Provider",
				"options": map[string]interface{}{
					"baseURL": "https://example.com",
				},
			},
		},
	}

	result := getProviderConfig(cfg, "test-provider")
	if result == nil {
		t.Fatal("Expected to find test-provider")
	}

	if result["name"] != "Test Provider" {
		t.Errorf("Expected name to be 'Test Provider', got %v", result["name"])
	}

	// Test non-existent provider
	result = getProviderConfig(cfg, "non-existent")
	if result != nil {
		t.Error("Expected nil for non-existent provider")
	}

	// Test with no provider section
	emptyConfig := OpenCodeConfig{}
	result = getProviderConfig(emptyConfig, "test")
	if result != nil {
		t.Error("Expected nil for empty config")
	}
}
