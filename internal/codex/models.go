package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/mattabdou/gantry/internal/jsonconf"
)

const catalogFilename = "gantry.models.json"

// Use the installed client's schema, prompts and tool definitions rather than
// freezing a copy of them in Gantry. --bundled is local and never refreshes over
// the network. Keep this seam separate so tests do not require a Codex install.
var readBundledCatalog = func() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	data, err := exec.CommandContext(ctx, "codex", "debug", "models", "--bundled").Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read Codex's bundled model catalog; install Codex CLI 0.156.0 or later: %w", err)
	}
	return data, nil
}

// buildModelCatalog retains the native catalog and adds the gateway aliases
// absent in older clients. Matching 5.6 models provide the Sol/Luna coding
// instructions until those models are bundled by Codex itself.
func buildModelCatalog(data []byte) (map[string]interface{}, error) {
	var catalog map[string]interface{}
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("invalid Codex model catalog: %w", err)
	}
	models, ok := catalog["models"].([]interface{})
	if !ok || len(models) == 0 {
		return nil, fmt.Errorf("Codex model catalog has no models; update Codex CLI")
	}
	byID := make(map[string]map[string]interface{})
	for _, value := range models {
		model, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid model entry in Codex catalog")
		}
		id, _ := model["slug"].(string)
		if id == "" || byID[id] != nil {
			return nil, fmt.Errorf("missing or duplicate model slug in Codex catalog")
		}
		byID[id] = model
	}
	for _, spec := range []struct{ id, fallback, name string }{
		{"gpt-6-astra", "", "GPT-6 Astra"},
		{"gpt-6-sol", "gpt-5.6-sol", "GPT-6 Sol"},
		{"gpt-6-luna", "gpt-5.6-luna", "GPT-6 Luna"},
		{"gpt-5.6-terra", "", "GPT-5.6 Terra"},
	} {
		model := byID[spec.id]
		if model == nil {
			template := byID[spec.fallback]
			if template == nil {
				return nil, fmt.Errorf("Codex catalog lacks metadata for %s; update Codex CLI to 0.156.0 or later", spec.id)
			}
			model = jsonconf.Clone(template)
			model["slug"] = spec.id
			model["display_name"] = spec.name
			model["description"] = spec.name + " through the Gantry LiteLLM gateway."
			model["upgrade"] = nil
			model["default_reasoning_level"] = "medium"
			// These aliases support API reasoning levels, not Codex's Ultra
			// orchestration preset inherited from the 5.6 template.
			levels := make([]interface{}, 0, 6)
			for _, effort := range []string{"none", "low", "medium", "high", "xhigh", "max"} {
				levels = append(levels, map[string]interface{}{"effort": effort, "description": effort + " reasoning effort"})
			}
			model["supported_reasoning_levels"] = levels
			models = append(models, model)
		}
		model["visibility"] = "list"
		model["supported_in_api"] = true
	}
	catalog["models"] = models
	return catalog, nil
}
