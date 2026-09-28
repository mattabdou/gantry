package codex

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mattabdou/gantry/internal/config"
	"github.com/mattabdou/gantry/internal/jsonconf"
)

func testCatalog() []byte {
	return []byte(`{"models":[
		{"slug":"gpt-6-astra","visibility":"hide","supported_in_api":false,"base_instructions":"astra instructions"},
		{"slug":"gpt-5.6-sol","base_instructions":"sol instructions","model_messages":{"template":"native prompt"},"future_capability":true},
		{"slug":"gpt-5.6-luna","base_instructions":"luna instructions"},
		{"slug":"gpt-5.6-terra","base_instructions":"terra instructions"},
		{"slug":"other-model","visibility":"hide"}
	]}`)
}

func catalogModels(t *testing.T, data []byte) map[string]map[string]interface{} {
	t.Helper()
	var c map[string]interface{}
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]interface{}{}
	for _, value := range c["models"].([]interface{}) {
		m := value.(map[string]interface{})
		out[m["slug"].(string)] = m
	}
	return out
}

func TestBuildCatalogPreservesNativeMetadataAndAddsAliases(t *testing.T) {
	catalog, err := buildModelCatalog(testCatalog())
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(catalog)
	models := catalogModels(t, data)
	for _, id := range []string{"gpt-6-astra", "gpt-6-sol", "gpt-6-luna", "gpt-5.6-terra"} {
		if models[id]["visibility"] != "list" || models[id]["supported_in_api"] != true {
			t.Errorf("%s is not selectable through the API", id)
		}
	}
	if models["gpt-6-sol"]["base_instructions"] != "sol instructions" || models["gpt-6-sol"]["future_capability"] != true {
		t.Fatal("native prompt or unknown metadata lost")
	}
	if jsonconf.Lookup(models["gpt-6-sol"], "model_messages", "template") != "native prompt" {
		t.Fatal("native message template lost")
	}
	if models["gpt-5.6-sol"]["slug"] != "gpt-5.6-sol" || models["other-model"]["visibility"] != "hide" {
		t.Fatal("unrelated native models changed")
	}
	// A newer client already knows the model: its own metadata must win over
	// our fallback, and running the builder again must not duplicate entries.
	for _, value := range catalog["models"].([]interface{}) {
		m := value.(map[string]interface{})
		if m["slug"] == "gpt-6-sol" {
			m["base_instructions"] = "new native sol instructions"
		}
	}
	data, _ = json.Marshal(catalog)
	second, err := buildModelCatalog(data)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonconf.Equal(catalog, second) {
		t.Fatal("catalog is not a fixed point")
	}
}

func TestBuildCatalogRejectsInvalidOrOldCatalog(t *testing.T) {
	for _, input := range []string{`bad`, `null`, `{}`, `{"models":[]}`, `{"models":[42]}`, `{"models":[{}]}`, `{"models":[{"slug":"x"},{"slug":"x"}]}`, `{"models":[{"slug":"old-model"}]}`} {
		if _, err := buildModelCatalog([]byte(input)); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}

func fakeCodex(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "codex home")
	t.Setenv("CODEX_HOME", dir)
	original := readBundledCatalog
	readBundledCatalog = func() ([]byte, error) { return testCatalog(), nil }
	t.Cleanup(func() { readBundledCatalog = original })
	return dir
}

func TestConfigureProviderUpdatesAndRepairsCatalog(t *testing.T) {
	dir := fakeCodex(t)
	settings := &config.LiteLLMConfig{BaseURL: "https://gateway.example", AuthToken: "do-not-write-this"}
	otel := config.OTELConfig{Endpoint: "https://otel.example", Headers: "x-test=value"}
	// User config and custom catalogs must not be modified.
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	userConfig := []byte("model_catalog_json = \"my-models.json\"\n")
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), userConfig, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := ConfigureProvider("gpt-5.6-terra", settings, otel)
	if err != nil || !result.Updated {
		t.Fatalf("first configure: %v, %v", result, err)
	}
	profile, err := os.ReadFile(result.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(profile), "model_catalog_json = ") || !strings.Contains(string(profile), "gpt-5.6-terra") || strings.Contains(string(profile), settings.AuthToken) {
		t.Fatal("profile missing catalog/default or leaking the API key")
	}
	catalogPath := filepath.Join(dir, catalogFilename)
	oldTime := time.Unix(1000000000, 0)
	for _, path := range []string{result.ConfigPath, catalogPath} {
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
	}
	result, err = ConfigureProvider("gpt-5.6-terra", settings, otel)
	if err != nil || result.Updated {
		t.Fatalf("second configure: %v, %v", result, err)
	}
	for _, path := range []string{result.ConfigPath, catalogPath} {
		info, err := os.Stat(path)
		if err != nil || !info.ModTime().Equal(oldTime) {
			t.Fatalf("unchanged file rewritten: %s", path)
		}
	}
	for _, corrupt := range []bool{false, true} {
		if corrupt {
			err = os.WriteFile(catalogPath, []byte("invalid"), 0600)
		} else {
			err = os.Remove(catalogPath)
		}
		if err != nil {
			t.Fatal(err)
		}
		result, err = ConfigureProvider("gpt-5.6-terra", settings, otel)
		if err != nil || !result.Updated {
			t.Fatalf("repair: %v, %v", result, err)
		}
		data, err := os.ReadFile(catalogPath)
		if err != nil {
			t.Fatal(err)
		}
		if catalogModels(t, data)["gpt-6-sol"] == nil {
			t.Fatal("repair lost model")
		}
	}
	actual, _ := os.ReadFile(filepath.Join(dir, "config.toml"))
	if string(actual) != string(userConfig) {
		t.Fatal("user config changed")
	}
	entries, _ := filepath.Glob(filepath.Join(dir, ".gantry-*"))
	if len(entries) != 0 {
		t.Fatal("temporary files left behind")
	}
}

func TestConfigureProviderFailurePreservesExistingFiles(t *testing.T) {
	dir := fakeCodex(t)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "gantry.config.toml")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	readBundledCatalog = func() ([]byte, error) { return nil, errors.New("missing codex") }
	if _, err := ConfigureProvider("gpt-6-astra", &config.LiteLLMConfig{}, config.OTELConfig{}); err == nil {
		t.Fatal("expected error")
	}
	actual, _ := os.ReadFile(path)
	if string(actual) != "existing" {
		t.Fatal("profile changed on failure")
	}
	if _, err := ConfigureProvider("gpt-6-astra", nil, config.OTELConfig{}); err == nil {
		t.Fatal("nil settings accepted")
	}
}
