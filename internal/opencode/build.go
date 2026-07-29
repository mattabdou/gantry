package opencode

import (
	"fmt"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/jsonconf"
)

// opencode.json belongs to the user, not to GANTRY. It holds MCP servers, agent
// definitions, themes, keybinds and permissions that GANTRY knows nothing
// about, so the builders below fill in keys and update credentials rather than
// assigning whole objects.
//
// The merge policy, by key:
//
//	provider.gantry-*.npm             GANTRY wins - the wrong adapter changes
//	                                  which request shape the models get
//	provider.gantry-*.options.*       GANTRY wins - credentials rotate, and
//	                                  they come from ~/.gantryrc.json
//	provider.gantry-*.name            fill only - it is a display label
//	provider.gantry-*.models.*        fill only, recursively, never deleting
//	provider.amazon-bedrock.options   fill only - GANTRY does not own this
//	                                  provider ID
//	model (top level)                 written only when absent or blank
//	everything else                   untouched
//
// A user who wants GANTRY's defaults back runs `gantry --resetconfig`.

// BuildLiteLLMConfig returns the OpenCode config that should be on disk, given
// what is there now and GANTRY's LiteLLM settings. cur is not mutated.
//
// It returns an error when the user has a non-object where GANTRY needs to
// descend, rather than replacing that value.
func BuildLiteLLMConfig(cur OpenCodeConfig, litellmConfig *config.LiteLLMConfig) (OpenCodeConfig, error) {
	if litellmConfig == nil {
		return nil, fmt.Errorf("no litellm configuration available")
	}

	out := OpenCodeConfig(jsonconf.Clone(cur))

	provider := jsonconf.Object(out, "provider", ProviderLiteLLM)
	if provider == nil {
		return nil, conflictError("provider." + ProviderLiteLLM)
	}
	provider["npm"] = npmOpenAICompatible
	jsonconf.SetIfBlank(provider, "name", "Gantry LiteLLM")

	options := jsonconf.Object(provider, "options")
	if options == nil {
		return nil, conflictError("provider." + ProviderLiteLLM + ".options")
	}
	options["baseURL"] = litellmConfig.BaseURL
	options["apiKey"] = litellmConfig.AuthToken

	models := jsonconf.Object(provider, "models")
	if models == nil {
		return nil, conflictError("provider." + ProviderLiteLLM + ".models")
	}
	jsonconf.MergeMissing(models, litellmModels())

	jsonconf.SetIfBlank(out, "model", DefaultLiteLLMModel)

	return out, nil
}

// BuildBedrockConfig returns the OpenCode config that should be on disk, given
// what is there now and GANTRY's Bedrock settings. cur is not mutated.
func BuildBedrockConfig(cur OpenCodeConfig, bedrockConfig *config.BedrockConfig) (OpenCodeConfig, error) {
	if bedrockConfig == nil {
		return nil, fmt.Errorf("no bedrock configuration available")
	}

	out := OpenCodeConfig(jsonconf.Clone(cur))

	provider := jsonconf.Object(out, "provider", ProviderBedrock)
	if provider == nil {
		return nil, conflictError("provider." + ProviderBedrock)
	}
	jsonconf.SetIfBlank(provider, "name", "Gantry Bedrock")

	options := jsonconf.Object(provider, "options")
	if options == nil {
		return nil, conflictError("provider." + ProviderBedrock + ".options")
	}
	options["region"] = bedrockConfig.AWSRegion
	options["profile"] = bedrockConfig.AWSProfile

	models := jsonconf.Object(provider, "models")
	if models == nil {
		return nil, conflictError("provider." + ProviderBedrock + ".models")
	}
	jsonconf.MergeMissing(models, bedrockModels())

	// amazon-bedrock is OpenCode's own provider, not GANTRY's. Earlier versions
	// replaced this block outright; now GANTRY only fills in blanks, and only
	// when the user already has the block - Lookup rather than Object, so this
	// never brings the provider into existence.
	if jsonconf.Lookup(out, "provider", ProviderAmazonBedrock) != nil {
		if builtinOptions := jsonconf.Object(out, "provider", ProviderAmazonBedrock, "options"); builtinOptions != nil {
			jsonconf.SetIfBlank(builtinOptions, "region", bedrockConfig.AWSRegion)
			jsonconf.SetIfBlank(builtinOptions, "profile", bedrockConfig.AWSProfile)
		}
	}

	jsonconf.SetIfBlank(out, "model", DefaultBedrockModel)

	return out, nil
}

// ResetGantryKeys returns cur with only the keys GANTRY owns removed: its two
// provider entries and the top-level default model. cur is not mutated.
//
// This is what `--resetconfig` uses. Deleting the whole file would also discard
// the user's MCP servers, agents, themes and keybinds, which is not what
// "reset gantry's configuration" means.
func ResetGantryKeys(cur OpenCodeConfig) OpenCodeConfig {
	out := OpenCodeConfig(jsonconf.Clone(cur))

	if providers, ok := out["provider"].(map[string]interface{}); ok {
		delete(providers, ProviderLiteLLM)
		delete(providers, ProviderBedrock)
		// Drop an empty provider object rather than leaving "provider": {}.
		if len(providers) == 0 {
			delete(out, "provider")
		}
	}
	delete(out, "model")

	return out
}

func conflictError(path string) error {
	return fmt.Errorf("cannot configure OpenCode: %q in your OpenCode config is not a JSON object; "+
		"fix or remove it, or run with --resetconfig", path)
}
