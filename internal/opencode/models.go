package opencode

// Provider IDs GANTRY manages inside opencode.json, and the adapter it pins.
const (
	// ProviderLiteLLM is the provider entry GANTRY owns in LiteLLM mode.
	ProviderLiteLLM = "gantry-litellm"
	// ProviderBedrock is the provider entry GANTRY owns in Bedrock mode.
	ProviderBedrock = "gantry-bedrock"
	// ProviderAmazonBedrock is OpenCode's own built-in Bedrock provider.
	// GANTRY does not own it and only fills in blanks there.
	ProviderAmazonBedrock = "amazon-bedrock"

	// npmOpenAICompatible is the AI SDK adapter the LiteLLM gateway speaks.
	// This choice is what makes reasoningEffort the right knob for every model
	// on that provider, including the Anthropic ones - OpenCode routes them all
	// through its OpenAI-compatible transform.
	npmOpenAICompatible = "@ai-sdk/openai-compatible"

	// DefaultLiteLLMModel is used for a config that has no top-level model yet.
	DefaultLiteLLMModel = ProviderLiteLLM + "/claude-opus-5"

	// DefaultBedrockModel deliberately lags DefaultLiteLLMModel. The Opus 5
	// cross-region inference profile ID has not been confirmed against Bedrock
	// (verify with `aws bedrock list-inference-profiles`), and a wrong default
	// is sticky: GANTRY writes the top-level model only when it is absent, so
	// it never self-corrects on a later run.
	DefaultBedrockModel = ProviderBedrock + "/us.anthropic.claude-opus-4-6-v1"

	// defaultReasoningEffort is the always-applied effort for LiteLLM models.
	// OpenCode has no provider/model/variant syntax for the top-level model
	// key - variants are switched interactively via /models or the
	// variant_cycle keybind - so a default effort has to live here.
	defaultReasoningEffort = "medium"
)

// effortModel builds a model entry for a model reached through the
// OpenAI-compatible adapter, where the reasoning control is reasoningEffort.
//
// options applies on every request; variants are the named overlays that
// /models and the variant_cycle keybind switch between. The variant set matches
// what the gateway is known to accept for openai-compatible models. Note that
// Anthropic's "max" effort is deliberately absent: it has no representation in
// the OpenAI-compatible request shape these models travel through.
func effortModel(displayName string) map[string]interface{} {
	return map[string]interface{}{
		"name":      displayName,
		"reasoning": true,
		"options": map[string]interface{}{
			"reasoningEffort": defaultReasoningEffort,
		},
		"variants": map[string]interface{}{
			"none":   map[string]interface{}{"reasoningEffort": "none"},
			"low":    map[string]interface{}{"reasoningEffort": "low"},
			"medium": map[string]interface{}{"reasoningEffort": "medium"},
			"high":   map[string]interface{}{"reasoningEffort": "high"},
			"xhigh":  map[string]interface{}{"reasoningEffort": "xhigh"},
		},
	}
}

// litellmModels returns the model catalog GANTRY publishes on the LiteLLM
// provider. A fresh map is returned on every call so that callers merging it
// into a user's config cannot alias shared state.
//
// The keys are the gateway's own aliases, not Claude API model IDs - which is
// why a bare "claude-opus-4-6" sits next to a fully-qualified
// "claude-haiku-4-5-20251001-v1:0". Confirm new keys with `gantry models`
// before adding them.
func litellmModels() map[string]interface{} {
	return map[string]interface{}{
		"claude-opus-5":                  effortModel("Claude Opus 5"),
		"claude-opus-4-6":                effortModel("Claude Opus 4.6"),
		"claude-sonnet-4-6":              effortModel("Claude Sonnet 4.6"),
		"claude-haiku-4-5-20251001-v1:0": effortModel("Claude Haiku 4.5"),
		"gpt-5.6-sol":                    effortModel("GPT 5.6 Sol"),
		"gpt-5.6-terra":                  effortModel("GPT 5.6 Terra"),
		"gpt-5.6-luna":                   effortModel("GPT 5.6 Luna"),
	}
}

// bedrockModels returns the model catalog GANTRY publishes on the Bedrock
// provider.
//
// These entries carry no reasoningEffort. This provider does not use the
// OpenAI-compatible adapter, so effort is not the knob here - Anthropic on
// Bedrock takes reasoningConfig.budgetTokens, and its built-in variants are
// high and max. "reasoning": true is still set because gantry-bedrock is a
// custom provider ID that inherits no metadata from the models.dev catalog,
// so without it OpenCode has no reason to expose thinking at all.
func bedrockModels() map[string]interface{} {
	return map[string]interface{}{
		"us.anthropic.claude-opus-5": map[string]interface{}{
			"name":      "Claude Opus 5",
			"reasoning": true,
		},
		"us.anthropic.claude-opus-4-6-v1": map[string]interface{}{
			"name":      "Claude Opus 4.6",
			"reasoning": true,
		},
		"us.anthropic.claude-opus-4-5-20251101-v1:0": map[string]interface{}{
			"name":      "Claude Opus 4.5",
			"reasoning": true,
		},
		"us.anthropic.claude-sonnet-4-5-20250929-v1:0": map[string]interface{}{
			"name":      "Claude Sonnet 4.5",
			"reasoning": true,
		},
		"us.anthropic.claude-haiku-4-5-20251001-v1:0": map[string]interface{}{
			"name":      "Claude Haiku 4.5",
			"reasoning": true,
		},
	}
}
