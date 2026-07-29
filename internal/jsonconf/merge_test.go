package jsonconf

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestObject(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		path     []string
		wantNil  bool
		wantKeys []string
	}{
		{
			name:  "creates missing levels",
			input: map[string]interface{}{},
			path:  []string{"provider", "gantry-litellm", "models"},
		},
		{
			name: "returns existing object",
			input: map[string]interface{}{
				"provider": map[string]interface{}{
					"gantry-litellm": map[string]interface{}{"npm": "pkg"},
				},
			},
			path:     []string{"provider", "gantry-litellm"},
			wantKeys: []string{"npm"},
		},
		{
			name:    "refuses to overwrite a scalar",
			input:   map[string]interface{}{"provider": "not-an-object"},
			path:    []string{"provider", "gantry-litellm"},
			wantNil: true,
		},
		{
			name:    "refuses to overwrite a nested scalar",
			input:   map[string]interface{}{"provider": map[string]interface{}{"gantry-litellm": 42}},
			path:    []string{"provider", "gantry-litellm", "models"},
			wantNil: true,
		},
		{
			name:  "treats explicit null as missing",
			input: map[string]interface{}{"provider": nil},
			path:  []string{"provider", "x"},
		},
		{
			name:     "empty path returns the map itself",
			input:    map[string]interface{}{"a": 1},
			path:     nil,
			wantKeys: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Object(tt.input, tt.path...)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("Object() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("Object() = nil, want non-nil")
			}
			for _, key := range tt.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("Object() result missing key %q", key)
				}
			}
			// The returned map must be live: writing to it writes to input.
			got["__probe"] = true
			if Lookup(tt.input, append(tt.path, "__probe")...) != true {
				t.Error("Object() returned a detached map; writes do not reach the input")
			}
		})
	}
}

func TestObjectDoesNotMutateOnRefusal(t *testing.T) {
	input := map[string]interface{}{"provider": "not-an-object"}
	if Object(input, "provider", "sub") != nil {
		t.Fatal("expected nil for a scalar level")
	}
	if input["provider"] != "not-an-object" {
		t.Errorf("input was mutated: provider = %v", input["provider"])
	}
}

func TestMergeMissing(t *testing.T) {
	tests := []struct {
		name string
		dst  map[string]interface{}
		src  map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "fills empty dst",
			dst:  map[string]interface{}{},
			src:  map[string]interface{}{"a": "1"},
			want: map[string]interface{}{"a": "1"},
		},
		{
			name: "dst wins on scalar collision",
			dst:  map[string]interface{}{"a": "mine"},
			src:  map[string]interface{}{"a": "theirs"},
			want: map[string]interface{}{"a": "mine"},
		},
		{
			name: "never deletes dst-only keys",
			dst:  map[string]interface{}{"mine": "keep"},
			src:  map[string]interface{}{"theirs": "add"},
			want: map[string]interface{}{"mine": "keep", "theirs": "add"},
		},
		{
			name: "recurses two levels and preserves the user's leaf",
			dst: map[string]interface{}{
				"claude-opus-5": map[string]interface{}{
					"options": map[string]interface{}{"reasoningEffort": "high"},
				},
			},
			src: map[string]interface{}{
				"claude-opus-5": map[string]interface{}{
					"name":    "Claude Opus 5",
					"options": map[string]interface{}{"reasoningEffort": "medium"},
					"variants": map[string]interface{}{
						"low": map[string]interface{}{"reasoningEffort": "low"},
					},
				},
			},
			want: map[string]interface{}{
				"claude-opus-5": map[string]interface{}{
					"name":    "Claude Opus 5",
					"options": map[string]interface{}{"reasoningEffort": "high"}, // user's value stands
					"variants": map[string]interface{}{
						"low": map[string]interface{}{"reasoningEffort": "low"},
					},
				},
			},
		},
		{
			name: "arrays are atomic - dst keeps its own",
			dst:  map[string]interface{}{"list": []interface{}{"mine"}},
			src:  map[string]interface{}{"list": []interface{}{"a", "b"}},
			want: map[string]interface{}{"list": []interface{}{"mine"}},
		},
		{
			name: "arrays are copied in when absent",
			dst:  map[string]interface{}{},
			src:  map[string]interface{}{"list": []interface{}{"a", "b"}},
			want: map[string]interface{}{"list": []interface{}{"a", "b"}},
		},
		{
			name: "explicit null in dst counts as absent",
			dst:  map[string]interface{}{"a": nil},
			src:  map[string]interface{}{"a": "filled"},
			want: map[string]interface{}{"a": "filled"},
		},
		{
			name: "dst scalar where src has object - dst wins, no panic",
			dst:  map[string]interface{}{"a": "scalar"},
			src:  map[string]interface{}{"a": map[string]interface{}{"deep": "val"}},
			want: map[string]interface{}{"a": "scalar"},
		},
		{
			name: "dst object where src has scalar - dst wins",
			dst:  map[string]interface{}{"a": map[string]interface{}{"deep": "val"}},
			src:  map[string]interface{}{"a": "scalar"},
			want: map[string]interface{}{"a": map[string]interface{}{"deep": "val"}},
		},
		{
			name: "nil src is a no-op",
			dst:  map[string]interface{}{"a": "1"},
			src:  nil,
			want: map[string]interface{}{"a": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MergeMissing(tt.dst, tt.src)
			if !reflect.DeepEqual(tt.dst, tt.want) {
				t.Errorf("MergeMissing() dst = %#v, want %#v", tt.dst, tt.want)
			}
		})
	}
}

func TestMergeMissingNilDstDoesNotPanic(t *testing.T) {
	MergeMissing(nil, map[string]interface{}{"a": "1"})
}

func TestMergeMissingDoesNotMutateSrc(t *testing.T) {
	src := map[string]interface{}{
		"model": map[string]interface{}{"name": "N", "opts": map[string]interface{}{"e": "medium"}},
	}
	before := Clone(src)

	dst := map[string]interface{}{}
	MergeMissing(dst, src)

	// Mutating the merged result must not reach back into src.
	Object(dst, "model", "opts")["e"] = "tampered"
	dst["model"].(map[string]interface{})["name"] = "tampered"

	if !reflect.DeepEqual(src, before) {
		t.Errorf("src was mutated:\n got %#v\nwant %#v", src, before)
	}
}

func TestSetIfBlank(t *testing.T) {
	tests := []struct {
		name      string
		dst       map[string]interface{}
		wantWrote bool
		wantValue interface{}
	}{
		{"absent key", map[string]interface{}{}, true, "set"},
		{"explicit null", map[string]interface{}{"k": nil}, true, "set"},
		{"empty string", map[string]interface{}{"k": ""}, true, "set"},
		{"whitespace string", map[string]interface{}{"k": "   "}, true, "set"},
		{"non-empty string", map[string]interface{}{"k": "user"}, false, "user"},
		{"non-string value", map[string]interface{}{"k": false}, false, false},
		{"zero number", map[string]interface{}{"k": float64(0)}, false, float64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrote := SetIfBlank(tt.dst, "k", "set")
			if wrote != tt.wantWrote {
				t.Errorf("SetIfBlank() = %v, want %v", wrote, tt.wantWrote)
			}
			if tt.dst["k"] != tt.wantValue {
				t.Errorf("dst[k] = %v, want %v", tt.dst["k"], tt.wantValue)
			}
		})
	}
}

func TestSetIfBlankNilDst(t *testing.T) {
	if SetIfBlank(nil, "k", "v") {
		t.Error("SetIfBlank(nil, ...) reported a write")
	}
}

func TestClone(t *testing.T) {
	orig := map[string]interface{}{
		"nested": map[string]interface{}{"deep": map[string]interface{}{"k": "v"}},
		"list":   []interface{}{"a", map[string]interface{}{"k": "v"}},
	}
	clone := Clone(orig)

	Object(clone, "nested", "deep")["k"] = "changed"
	clone["list"].([]interface{})[1].(map[string]interface{})["k"] = "changed"
	clone["added"] = true

	if got := Lookup(orig, "nested", "deep", "k"); got != "v" {
		t.Errorf("original nested value changed to %v", got)
	}
	if got := orig["list"].([]interface{})[1].(map[string]interface{})["k"]; got != "v" {
		t.Errorf("original list element changed to %v", got)
	}
	if _, ok := orig["added"]; ok {
		t.Error("key added to clone appeared in the original")
	}
}

func TestCloneNil(t *testing.T) {
	got := Clone(nil)
	if got == nil {
		t.Fatal("Clone(nil) = nil, want an empty non-nil map")
	}
	if len(got) != 0 {
		t.Errorf("Clone(nil) = %v, want empty", got)
	}
}

// TestEqualIgnoresGoNumericTypes guards the whole idempotency scheme.
//
// A config read from disk holds numbers as float64; one built from Go literals
// holds int. If Equal treated those as different, every caller that writes on
// inequality would rewrite the config - and take a backup - on every single
// run. That was the original accumulating-backups bug in a subtler form.
func TestEqualIgnoresGoNumericTypes(t *testing.T) {
	tests := []struct {
		name string
		a, b interface{}
		want bool
	}{
		{"int vs float64", map[string]interface{}{"a": 1}, map[string]interface{}{"a": 1.0}, true},
		{"nested int vs float64",
			map[string]interface{}{"limit": map[string]interface{}{"context": 200000}},
			map[string]interface{}{"limit": map[string]interface{}{"context": float64(200000)}},
			true},
		{"[]string vs []interface{}",
			map[string]interface{}{"l": []string{"a", "b"}},
			map[string]interface{}{"l": []interface{}{"a", "b"}},
			true},
		{"typed struct vs map",
			map[string]interface{}{"s": struct {
				Name string `json:"name"`
			}{"n"}},
			map[string]interface{}{"s": map[string]interface{}{"name": "n"}},
			true},
		{"genuinely different values",
			map[string]interface{}{"a": 1}, map[string]interface{}{"a": 2}, false},
		{"genuinely different keys",
			map[string]interface{}{"a": 1}, map[string]interface{}{"b": 1}, false},
		{"extra key",
			map[string]interface{}{"a": 1, "b": 2}, map[string]interface{}{"a": 1}, false},
		{"key order is irrelevant",
			map[string]interface{}{"a": 1, "b": 2}, map[string]interface{}{"b": 2, "a": 1}, true},
		{"both empty", map[string]interface{}{}, map[string]interface{}{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Equal(tt.a, tt.b); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEqualUnmarshalableReportsUnequal(t *testing.T) {
	// A channel cannot be marshaled; Equal must not claim equality.
	bad := map[string]interface{}{"c": make(chan int)}
	if Equal(bad, map[string]interface{}{"c": nil}) {
		t.Error("Equal() reported equality for unmarshalable input")
	}
}

func TestNormalizeMatchesDiskRoundTrip(t *testing.T) {
	in := map[string]interface{}{"n": 7, "s": "x", "l": []string{"a"}}

	norm, err := Normalize(in)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	data, _ := json.Marshal(in)
	var viaDisk interface{}
	if err := json.Unmarshal(data, &viaDisk); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}
	if !reflect.DeepEqual(norm, viaDisk) {
		t.Errorf("Normalize() = %#v, want %#v", norm, viaDisk)
	}
}
