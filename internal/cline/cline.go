package cline

import (
	"fmt"
	"os/exec"

	"github.com/mattabdou/gantry/internal/config"
)

// ConfigureProvider runs cline auth to configure the OpenAI-native provider with the LiteLLM gateway
func ConfigureProvider(litellmConfig *config.LiteLLMConfig) error {
	cmd := exec.Command("cline", "auth",
		"--provider", "openai-native",
		"--apikey", litellmConfig.AuthToken,
		"--modelid", litellmConfig.Model,
		"--baseurl", litellmConfig.BaseURL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cline auth failed: %w\n%s", err, string(output))
	}

	return nil
}
