package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

// TestStripJSONComments asserts that stripping comments yields the same JSON
// document, rather than asserting exact output bytes. How much whitespace is
// left behind where a comment used to be is incidental; what matters is that
// the result parses, and parses to the same thing.
//
// The URL cases are the regression test for a scanner-vs-regex bug: a plain
// `//.*$` match truncates every URL in the file. Since GANTRY writes baseURL
// into this very file, that turned any .jsonc config into invalid JSON and
// blocked the user from launching OpenCode at all.
func TestStripJSONComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // the JSON the stripped output must parse to
	}{
		{
			name:  "single line comment",
			input: `{"key": "value"} // comment`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "multi-line comment",
			input: `{"key": /* comment */ "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "no comments",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "url in a string value survives",
			input: `{"baseURL": "https://llm.example.com/v1"}`,
			want:  `{"baseURL": "https://llm.example.com/v1"}`,
		},
		{
			name:  "url survives alongside a real comment",
			input: "// header\n{\"baseURL\": \"https://llm.example.com\"} // trailing",
			want:  `{"baseURL": "https://llm.example.com"}`,
		},
		{
			name:  "block comment marker inside a string value survives",
			input: `{"note": "/* not a comment */"}`,
			want:  `{"note": "/* not a comment */"}`,
		},
		{
			name:  "line comment marker inside a string value survives",
			input: `{"note": "a // b"}`,
			want:  `{"note": "a // b"}`,
		},
		{
			name:  "escaped quote before a comment marker",
			input: `{"note": "say \"hi\" // not a comment"}`,
			want:  `{"note": "say \"hi\" // not a comment"}`,
		},
		{
			name:  "escaped backslash does not swallow the closing quote",
			input: `{"path": "C:\\dir\\", "next": "https://x.example.com"}`,
			want:  `{"path": "C:\\dir\\", "next": "https://x.example.com"}`,
		},
		{
			name:  "comment between object members",
			input: "{\n  \"a\": 1,\n  // explain b\n  \"b\": 2\n}",
			want:  `{"a": 1, "b": 2}`,
		},
		{
			name:  "block comment spanning lines",
			input: "{\n  /* one\n     two */\n  \"a\": 1\n}",
			want:  `{"a": 1}`,
		},
		{
			name:  "realistic gantry-written jsonc config",
			input: "{\n  // gantry manages this provider\n  \"provider\": {\n    \"gantry-litellm\": {\n      \"npm\": \"@ai-sdk/openai-compatible\",\n      \"options\": {\n        \"baseURL\": \"https://llm.vit-qa.example.com\", // the gateway\n        \"apiKey\": \"sk-abc123\"\n      }\n    }\n  }\n}",
			want:  `{"provider":{"gantry-litellm":{"npm":"@ai-sdk/openai-compatible","options":{"baseURL":"https://llm.vit-qa.example.com","apiKey":"sk-abc123"}}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stripped := stripJSONComments([]byte(tt.input))

			var got interface{}
			if err := json.Unmarshal(stripped, &got); err != nil {
				t.Fatalf("stripped output does not parse: %v\noutput: %s", err, stripped)
			}

			var want interface{}
			if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
				t.Fatalf("test's want value is not valid JSON: %v", err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("stripJSONComments() parsed to %#v, want %#v\noutput: %s", got, want, stripped)
			}
		})
	}
}

func TestStripJSONCommentsUnterminatedBlockComment(t *testing.T) {
	// An unterminated block comment consumes the rest of the input. The result
	// must not parse - better a reported syntax error than silently truncated
	// config.
	stripped := stripJSONComments([]byte(`{"a": 1 /* never closed`))
	var out interface{}
	if err := json.Unmarshal(stripped, &out); err == nil {
		t.Errorf("want a parse error for an unterminated block comment, got %#v", out)
	}
}

func TestConfigExists(t *testing.T) {
	// This test just verifies the function runs without panic
	// The actual result depends on the system state
	_ = ConfigExists()
}

// The provider-lookup helper this file used to test was superseded by
// jsonconf.Lookup, which is covered in internal/jsonconf/merge_test.go.
