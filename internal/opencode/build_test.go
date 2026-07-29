package opencode

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/jsonconf"
)

func testLiteLLM() *config.LiteLLMConfig {
	return &config.LiteLLMConfig{
		BaseURL:   "https://llm.example.com",
		AuthToken: "sk-current",
	}
}

func testBedrock() *config.BedrockConfig {
	return &config.BedrockConfig{
		AWSRegion:  "us-east-1",
		AWSProfile: "gantry",
	}
}

// mustJSON parses a JSON literal into an OpenCodeConfig, so test inputs look
// like the file on disk and carry disk's types (every number a float64).
func mustJSON(t *testing.T, s string) OpenCodeConfig {
	t.Helper()
	cfg, err := jsonconf.UnmarshalObject([]byte(s))
	if err != nil {
		t.Fatalf("test input is not valid JSON: %v", err)
	}
	return OpenCodeConfig(cfg)
}

func TestBuildLiteLLMConfigPreservesUserContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, out OpenCodeConfig)
	}{
		{
			name:  "empty config gets the full catalog and a default model",
			input: `{}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				models := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models")
				got, ok := models.(map[string]interface{})
				if !ok {
					t.Fatalf("models is not an object: %#v", models)
				}
				for key := range litellmModels() {
					if _, present := got[key]; !present {
						t.Errorf("catalog model %q missing", key)
					}
				}
				if out["model"] != DefaultLiteLLMModel {
					t.Errorf("model = %v, want %v", out["model"], DefaultLiteLLMModel)
				}
			},
		},
		{
			name: "a model the user added survives",
			input: `{"provider":{"gantry-litellm":{"models":{
				"my-local-llama":{"name":"My Llama"}
			}}}}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "my-local-llama", "name"); got != "My Llama" {
					t.Errorf("user's model lost: name = %v", got)
				}
				if jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "claude-opus-5") == nil {
					t.Error("catalog model was not added alongside the user's")
				}
			},
		},
		{
			name: "the user's reasoningEffort on a catalog model wins",
			input: `{"provider":{"gantry-litellm":{"models":{
				"claude-opus-5":{"options":{"reasoningEffort":"high"}}
			}}}}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "claude-opus-5", "options", "reasoningEffort"); got != "high" {
					t.Errorf("reasoningEffort = %v, want the user's \"high\"", got)
				}
				// The rest of the entry should still be filled in.
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "claude-opus-5", "name"); got != "Claude Opus 5" {
					t.Errorf("name = %v, want it filled in", got)
				}
				if jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "claude-opus-5", "variants", "low") == nil {
					t.Error("variants were not filled in around the user's option")
				}
			},
		},
		{
			name: "a model the user renamed keeps its name",
			input: `{"provider":{"gantry-litellm":{"models":{
				"claude-opus-5":{"name":"Big Brain"}
			}}}}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "claude-opus-5", "name"); got != "Big Brain" {
					t.Errorf("name = %v, want the user's \"Big Brain\"", got)
				}
			},
		},
		{
			name: "a variant the user customized wins",
			input: `{"provider":{"gantry-litellm":{"models":{
				"gpt-5.6-sol":{"variants":{"low":{"reasoningEffort":"minimal"}}}
			}}}}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "gpt-5.6-sol", "variants", "low", "reasoningEffort"); got != "minimal" {
					t.Errorf("variants.low = %v, want the user's \"minimal\"", got)
				}
				if jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "gpt-5.6-sol", "variants", "high") == nil {
					t.Error("sibling variants were not filled in")
				}
			},
		},
		{
			name:  "the user's default model is not reset",
			input: `{"model":"gantry-litellm/gpt-5.6-luna"}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if out["model"] != "gantry-litellm/gpt-5.6-luna" {
					t.Errorf("model = %v, want the user's choice", out["model"])
				}
			},
		},
		{
			name:  "a blank default model is filled in",
			input: `{"model":"   "}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if out["model"] != DefaultLiteLLMModel {
					t.Errorf("model = %q, want %q", out["model"], DefaultLiteLLMModel)
				}
			},
		},
		{
			name:  "a provider name the user set is kept",
			input: `{"provider":{"gantry-litellm":{"name":"Work Gateway"}}}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "name"); got != "Work Gateway" {
					t.Errorf("name = %v, want the user's label", got)
				}
			},
		},
		{
			name: "stale credentials are replaced",
			input: `{"provider":{"gantry-litellm":{"options":{
				"baseURL":"https://old.example.com","apiKey":"sk-expired"
			}}}}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "options", "baseURL"); got != "https://llm.example.com" {
					t.Errorf("baseURL = %v, want it refreshed from gantry config", got)
				}
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "options", "apiKey"); got != "sk-current" {
					t.Errorf("apiKey = %v, want it refreshed from gantry config", got)
				}
			},
		},
		{
			name: "extra provider options are kept alongside credentials",
			input: `{"provider":{"gantry-litellm":{"options":{
				"headers":{"X-Team":"platform"},"timeout":30
			}}}}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "options", "headers", "X-Team"); got != "platform" {
					t.Errorf("custom header lost: %v", got)
				}
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "options", "timeout"); got != float64(30) {
					t.Errorf("timeout = %v, want it preserved", got)
				}
			},
		},
		{
			name:  "an unrelated provider is untouched",
			input: `{"provider":{"openai":{"models":{"gpt-4":{"name":"GPT-4"}}}}}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if got := jsonconf.Lookup(out, "provider", "openai", "models", "gpt-4", "name"); got != "GPT-4" {
					t.Errorf("unrelated provider lost: %v", got)
				}
			},
		},
		{
			name:  "the npm adapter is corrected if wrong",
			input: `{"provider":{"gantry-litellm":{"npm":"@ai-sdk/anthropic"}}}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "npm"); got != npmOpenAICompatible {
					t.Errorf("npm = %v, want %v", got, npmOpenAICompatible)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := BuildLiteLLMConfig(mustJSON(t, tt.input), testLiteLLM())
			if err != nil {
				t.Fatalf("BuildLiteLLMConfig() error = %v", err)
			}
			tt.check(t, out)
		})
	}
}

// TestBuildLiteLLMConfigPreservesUnknownTopLevelKeys is the direct regression
// test for the reported bug: GANTRY was replacing whole objects, so a user's
// MCP servers, agents, themes and keybinds vanished.
func TestBuildLiteLLMConfigPreservesUnknownTopLevelKeys(t *testing.T) {
	input := mustJSON(t, `{
		"$schema": "https://opencode.ai/config.json",
		"mcp": {
			"brave-search": {
				"command": ["npx","-y","@modelcontextprotocol/server-brave-search"],
				"enabled": true,
				"environment": {"BRAVE_API_KEY":"secret"},
				"type": "local"
			}
		},
		"mode": {
			"build": {"model":"gantry-litellm/claude-sonnet-4-6"},
			"plan":  {"model":"gantry-litellm/claude-opus-4-6"}
		},
		"agent": {"reviewer": {"model":"gantry-litellm/gpt-5.6-sol"}},
		"theme": "tokyonight",
		"keybinds": {"leader":"ctrl+x"},
		"permission": {"bash":"ask"},
		"autoupdate": true,
		"plugin": ["oh-my-opencode"]
	}`)

	// Snapshot every key GANTRY does not own.
	untouched := []string{"$schema", "mcp", "mode", "agent", "theme", "keybinds", "permission", "autoupdate", "plugin"}
	before := map[string]interface{}{}
	for _, key := range untouched {
		before[key] = input[key]
	}

	out, err := BuildLiteLLMConfig(input, testLiteLLM())
	if err != nil {
		t.Fatalf("BuildLiteLLMConfig() error = %v", err)
	}

	for _, key := range untouched {
		got, present := out[key]
		if !present {
			t.Errorf("key %q was dropped from the config", key)
			continue
		}
		if !jsonconf.Equal(got, before[key]) {
			t.Errorf("key %q was modified:\n got %#v\nwant %#v", key, got, before[key])
		}
	}
}

func TestBuildLiteLLMConfigDoesNotMutateInput(t *testing.T) {
	input := mustJSON(t, `{
		"model":"gantry-litellm/gpt-5.6-luna",
		"mcp":{"x":{"enabled":true}},
		"provider":{"gantry-litellm":{"models":{"mine":{"name":"Mine"}}}}
	}`)
	snapshot := jsonconf.Clone(input)

	if _, err := BuildLiteLLMConfig(input, testLiteLLM()); err != nil {
		t.Fatalf("BuildLiteLLMConfig() error = %v", err)
	}

	if !jsonconf.Equal(map[string]interface{}(input), snapshot) {
		t.Errorf("input was mutated:\n got %#v\nwant %#v", input, snapshot)
	}
}

// TestBuildLiteLLMConfigIsFixedPointAcrossJSONRoundTrip is the regression test
// for the accumulating-backups bug.
//
// The write path only writes when the built config differs from what is on
// disk, so building twice must produce the same result - otherwise GANTRY
// rewrites the file and takes a backup on every single launch.
//
// The JSON round trip between the two builds is essential, not decoration. It
// reproduces what actually happens between two gantry invocations: the config
// goes to disk and comes back with every number as a float64. A same-process
// idempotency check would pass even with that bug present.
func TestBuildLiteLLMConfigIsFixedPointAcrossJSONRoundTrip(t *testing.T) {
	inputs := []string{
		`{}`,
		`{"mcp":{"x":{"enabled":true}},"theme":"dark"}`,
		`{"provider":{"gantry-litellm":{"models":{"extra":{"name":"Extra"}}}}}`,
		`{"model":"gantry-litellm/gpt-5.6-terra","permission":{"bash":"ask"}}`,
		`{"provider":{"gantry-litellm":{"options":{"timeout":30},"models":{"claude-opus-5":{"options":{"reasoningEffort":"high"}}}}}}`,
	}

	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			first, err := BuildLiteLLMConfig(mustJSON(t, in), testLiteLLM())
			if err != nil {
				t.Fatalf("first build: %v", err)
			}

			// Round-trip through disk representation.
			data, err := json.Marshal(first)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			onDisk, err := jsonconf.UnmarshalObject(data)
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			second, err := BuildLiteLLMConfig(OpenCodeConfig(onDisk), testLiteLLM())
			if err != nil {
				t.Fatalf("second build: %v", err)
			}

			if !jsonconf.Equal(first, second) {
				t.Errorf("not a fixed point; gantry would rewrite and back up on every run\nfirst:  %#v\nsecond: %#v", first, second)
			}
			// And the second build must agree with what is already on disk, or
			// the write path would fire.
			if !jsonconf.Equal(onDisk, second) {
				t.Errorf("second build differs from the on-disk config\ndisk:   %#v\nsecond: %#v", onDisk, second)
			}
		})
	}
}

func TestBuildBedrockConfigIsFixedPointAcrossJSONRoundTrip(t *testing.T) {
	first, err := BuildBedrockConfig(mustJSON(t, `{"mcp":{"x":{"enabled":true}}}`), testBedrock())
	if err != nil {
		t.Fatalf("first build: %v", err)
	}

	data, _ := json.Marshal(first)
	onDisk, err := jsonconf.UnmarshalObject(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	second, err := BuildBedrockConfig(OpenCodeConfig(onDisk), testBedrock())
	if err != nil {
		t.Fatalf("second build: %v", err)
	}
	if !jsonconf.Equal(onDisk, second) {
		t.Errorf("not a fixed point\ndisk:   %#v\nsecond: %#v", onDisk, second)
	}
}

func TestBuildBedrockConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, out OpenCodeConfig)
	}{
		{
			name:  "empty config gets catalog, options and default model",
			input: `{}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if got := jsonconf.Lookup(out, "provider", ProviderBedrock, "options", "region"); got != "us-east-1" {
					t.Errorf("region = %v", got)
				}
				if got := jsonconf.Lookup(out, "provider", ProviderBedrock, "options", "profile"); got != "gantry" {
					t.Errorf("profile = %v", got)
				}
				if jsonconf.Lookup(out, "provider", ProviderBedrock, "models", "us.anthropic.claude-opus-5") == nil {
					t.Error("Opus 5 missing from the Bedrock catalog")
				}
				if out["model"] != DefaultBedrockModel {
					t.Errorf("model = %v, want %v", out["model"], DefaultBedrockModel)
				}
			},
		},
		{
			name: "bedrock catalog entries carry no reasoningEffort",
			// gantry-bedrock does not use the OpenAI-compatible adapter, so
			// effort is not the right knob there.
			input: `{}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				entry := jsonconf.Lookup(out, "provider", ProviderBedrock, "models", "us.anthropic.claude-opus-5")
				obj, ok := entry.(map[string]interface{})
				if !ok {
					t.Fatalf("entry is not an object: %#v", entry)
				}
				if _, present := obj["variants"]; present {
					t.Error("bedrock entry has variants; effort is not the Bedrock knob")
				}
				if _, present := obj["options"]; present {
					t.Error("bedrock entry has options; effort is not the Bedrock knob")
				}
				if obj["reasoning"] != true {
					t.Error("bedrock entry should still declare reasoning: true")
				}
			},
		},
		{
			name: "the built-in amazon-bedrock provider is filled, not replaced",
			input: `{"provider":{"amazon-bedrock":{
				"options":{"region":"eu-west-1"},
				"models":{"custom":{"name":"Custom"}}
			}}}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				// The user's region wins: gantry does not own this provider.
				if got := jsonconf.Lookup(out, "provider", ProviderAmazonBedrock, "options", "region"); got != "eu-west-1" {
					t.Errorf("region = %v, want the user's eu-west-1", got)
				}
				// The blank profile is filled in.
				if got := jsonconf.Lookup(out, "provider", ProviderAmazonBedrock, "options", "profile"); got != "gantry" {
					t.Errorf("profile = %v, want it filled in", got)
				}
				// And the user's models survive - this block used to be wiped.
				if got := jsonconf.Lookup(out, "provider", ProviderAmazonBedrock, "models", "custom", "name"); got != "Custom" {
					t.Errorf("amazon-bedrock models lost: %v", got)
				}
			},
		},
		{
			name:  "amazon-bedrock is not created when absent",
			input: `{}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if jsonconf.Lookup(out, "provider", ProviderAmazonBedrock) != nil {
					t.Error("amazon-bedrock was created; gantry should only fill it in when the user has it")
				}
			},
		},
		{
			name:  "the user's default model is not reset",
			input: `{"model":"anthropic/claude-opus-4-6"}`,
			check: func(t *testing.T, out OpenCodeConfig) {
				if out["model"] != "anthropic/claude-opus-4-6" {
					t.Errorf("model = %v, want the user's choice", out["model"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := BuildBedrockConfig(mustJSON(t, tt.input), testBedrock())
			if err != nil {
				t.Fatalf("BuildBedrockConfig() error = %v", err)
			}
			tt.check(t, out)
		})
	}
}

func TestBuildConfigRefusesToClobberScalars(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"provider is a string", `{"provider":"nope"}`},
		{"gantry provider is a number", `{"provider":{"gantry-litellm":42}}`},
		{"options is a string", `{"provider":{"gantry-litellm":{"options":"nope"}}}`},
		{"models is an array", `{"provider":{"gantry-litellm":{"models":[]}}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := mustJSON(t, tt.input)
			snapshot := jsonconf.Clone(input)

			out, err := BuildLiteLLMConfig(input, testLiteLLM())
			if err == nil {
				t.Fatalf("BuildLiteLLMConfig() error = nil, want an error; got %#v", out)
			}
			if !jsonconf.Equal(map[string]interface{}(input), snapshot) {
				t.Error("input was mutated on the error path")
			}
		})
	}
}

func TestBuildConfigRequiresProviderSettings(t *testing.T) {
	if _, err := BuildLiteLLMConfig(OpenCodeConfig{}, nil); err == nil {
		t.Error("BuildLiteLLMConfig(nil config) error = nil, want an error")
	}
	if _, err := BuildBedrockConfig(OpenCodeConfig{}, nil); err == nil {
		t.Error("BuildBedrockConfig(nil config) error = nil, want an error")
	}
}

func TestResetGantryKeys(t *testing.T) {
	input := mustJSON(t, `{
		"model": "gantry-litellm/claude-opus-5",
		"mcp": {"brave":{"enabled":true}},
		"mode": {"build":{"model":"gantry-litellm/claude-sonnet-4-6"}},
		"agent": {"reviewer":{"model":"x"}},
		"theme": "tokyonight",
		"keybinds": {"leader":"ctrl+x"},
		"provider": {
			"gantry-litellm": {"models":{"claude-opus-5":{"name":"Claude Opus 5"}}},
			"gantry-bedrock": {"models":{"a":{"name":"A"}}},
			"amazon-bedrock": {"options":{"region":"eu-west-1"}},
			"openai": {"models":{"gpt-4":{"name":"GPT-4"}}}
		}
	}`)

	out := ResetGantryKeys(input)

	// Gantry-owned keys are gone.
	if _, present := out["model"]; present {
		t.Error("top-level model survived the reset")
	}
	if jsonconf.Lookup(out, "provider", ProviderLiteLLM) != nil {
		t.Error("gantry-litellm survived the reset")
	}
	if jsonconf.Lookup(out, "provider", ProviderBedrock) != nil {
		t.Error("gantry-bedrock survived the reset")
	}

	// Everything else stays - this is the point of a surgical reset.
	for _, key := range []string{"mcp", "mode", "agent", "theme", "keybinds"} {
		if !jsonconf.Equal(out[key], input[key]) {
			t.Errorf("key %q was altered by the reset", key)
		}
	}
	if got := jsonconf.Lookup(out, "provider", ProviderAmazonBedrock, "options", "region"); got != "eu-west-1" {
		t.Errorf("amazon-bedrock was altered: region = %v", got)
	}
	if got := jsonconf.Lookup(out, "provider", "openai", "models", "gpt-4", "name"); got != "GPT-4" {
		t.Errorf("unrelated provider was altered: %v", got)
	}
}

func TestResetGantryKeysDoesNotMutateInput(t *testing.T) {
	input := mustJSON(t, `{"model":"x","provider":{"gantry-litellm":{"npm":"pkg"}}}`)
	snapshot := jsonconf.Clone(input)

	ResetGantryKeys(input)

	if !jsonconf.Equal(map[string]interface{}(input), snapshot) {
		t.Errorf("input was mutated:\n got %#v\nwant %#v", input, snapshot)
	}
}

func TestResetGantryKeysDropsEmptyProviderObject(t *testing.T) {
	out := ResetGantryKeys(mustJSON(t, `{"provider":{"gantry-litellm":{"npm":"pkg"}}}`))
	if _, present := out["provider"]; present {
		t.Errorf("empty provider object was left behind: %#v", out["provider"])
	}
}

func TestResetGantryKeysOnConfigWithoutGantryKeys(t *testing.T) {
	input := mustJSON(t, `{"mcp":{"x":{"enabled":true}},"theme":"dark"}`)
	out := ResetGantryKeys(input)
	if !jsonconf.Equal(map[string]interface{}(out), map[string]interface{}(input)) {
		t.Errorf("reset was not a no-op:\n got %#v\nwant %#v", out, input)
	}
}

// The reset must leave the config in a state the builder then fills back in,
// which is how --resetconfig gets gantry's defaults back in one write.
func TestResetThenBuildRestoresDefaults(t *testing.T) {
	input := mustJSON(t, `{
		"model":"gantry-litellm/gpt-5.6-luna",
		"mcp":{"brave":{"enabled":true}},
		"provider":{"gantry-litellm":{"models":{
			"claude-opus-5":{"name":"Big Brain","options":{"reasoningEffort":"high"}},
			"my-extra":{"name":"Extra"}
		}}}
	}`)

	out, err := BuildLiteLLMConfig(ResetGantryKeys(input), testLiteLLM())
	if err != nil {
		t.Fatalf("build after reset: %v", err)
	}

	if out["model"] != DefaultLiteLLMModel {
		t.Errorf("model = %v, want the default restored", out["model"])
	}
	if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "claude-opus-5", "name"); got != "Claude Opus 5" {
		t.Errorf("name = %v, want the default restored", got)
	}
	if got := jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "claude-opus-5", "options", "reasoningEffort"); got != defaultReasoningEffort {
		t.Errorf("reasoningEffort = %v, want the default restored", got)
	}
	if jsonconf.Lookup(out, "provider", ProviderLiteLLM, "models", "my-extra") != nil {
		t.Error("a reset should clear the user's additions inside gantry's provider")
	}
	// But a reset is scoped to gantry's keys.
	if jsonconf.Lookup(out, "mcp", "brave", "enabled") != true {
		t.Error("reset discarded the user's MCP configuration")
	}
}

func TestLiteLLMCatalogShape(t *testing.T) {
	models := litellmModels()

	if len(models) != 7 {
		t.Errorf("catalog has %d models, want 7", len(models))
	}
	if _, present := models["claude-opus-5"]; !present {
		t.Error("claude-opus-5 missing from the catalog")
	}

	for key, entry := range models {
		obj, ok := entry.(map[string]interface{})
		if !ok {
			t.Errorf("%s: entry is not an object", key)
			continue
		}
		if obj["reasoning"] != true {
			t.Errorf("%s: reasoning = %v, want true", key, obj["reasoning"])
		}
		if name, _ := obj["name"].(string); name == "" {
			t.Errorf("%s: missing a display name", key)
		}
		if got := jsonconf.Lookup(obj, "options", "reasoningEffort"); got != defaultReasoningEffort {
			t.Errorf("%s: options.reasoningEffort = %v, want %q", key, got, defaultReasoningEffort)
		}

		variants, ok := obj["variants"].(map[string]interface{})
		if !ok {
			t.Errorf("%s: variants is not an object", key)
			continue
		}
		for _, want := range []string{"none", "low", "medium", "high", "xhigh"} {
			if got := jsonconf.Lookup(variants, want, "reasoningEffort"); got != want {
				t.Errorf("%s: variants.%s.reasoningEffort = %v, want %q", key, want, got, want)
			}
		}
		// "max" is Anthropic-API-only and has no representation in the
		// OpenAI-compatible request shape these models travel through.
		if _, present := variants["max"]; present {
			t.Errorf("%s: variant \"max\" is not expressible via reasoningEffort", key)
		}
	}
}

// The catalogs must be free of Go integer literals. A number written as an int
// here would come back from disk as a float64, so the write path would see a
// difference forever and rewrite the config on every launch. jsonconf.Equal
// defends against this, but keeping the catalogs all-string means the defence
// is never exercised in the hot path.
func TestCatalogsContainNoIntegerLiterals(t *testing.T) {
	for name, catalog := range map[string]map[string]interface{}{
		"litellm": litellmModels(),
		"bedrock": bedrockModels(),
	} {
		t.Run(name, func(t *testing.T) {
			assertNoInts(t, catalog, name)
		})
	}
}

func assertNoInts(t *testing.T, v interface{}, path string) {
	t.Helper()
	switch typed := v.(type) {
	case map[string]interface{}:
		for key, val := range typed {
			assertNoInts(t, val, path+"."+key)
		}
	case []interface{}:
		for i, val := range typed {
			assertNoInts(t, val, fmt.Sprintf("%s[%d]", path, i))
		}
	case string, bool, nil, float64:
		// Fine: these survive a JSON round trip unchanged.
	default:
		t.Errorf("%s: value %#v has type %T; use a string, bool or float64 so it round-trips", path, v, v)
	}
}

func TestBedrockCatalogShape(t *testing.T) {
	for key, entry := range bedrockModels() {
		obj, ok := entry.(map[string]interface{})
		if !ok {
			t.Errorf("%s: entry is not an object", key)
			continue
		}
		if name, _ := obj["name"].(string); name == "" {
			t.Errorf("%s: missing a display name", key)
		}
		if obj["reasoning"] != true {
			t.Errorf("%s: reasoning = %v, want true", key, obj["reasoning"])
		}
	}
}
