// Package jsonconf provides primitives for editing JSON configuration files
// that are owned by someone else.
//
// The files GANTRY touches (~/.config/opencode/opencode.json,
// ~/.claude/settings.json) belong to the user and to the tool, not to GANTRY.
// They contain keys GANTRY knows nothing about - MCP servers, agent
// definitions, keybinds, permissions, hooks - and those keys must survive
// every write. The functions here are deliberately narrow so that a caller
// spells out its merge policy key by key rather than assigning whole objects.
package jsonconf

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Object returns the nested JSON object at path, creating empty objects for any
// level that is missing. The returned map is the live one inside m, so writes
// to it are writes to m.
//
// It returns nil if some level along path exists but is not a JSON object. That
// lets a caller refuse to overwrite a value the user deliberately put there
// instead of silently replacing it.
func Object(m map[string]interface{}, path ...string) map[string]interface{} {
	cur := m
	for _, key := range path {
		if cur == nil {
			return nil
		}
		existing, ok := cur[key]
		if !ok || existing == nil {
			next := make(map[string]interface{})
			cur[key] = next
			cur = next
			continue
		}
		next, ok := existing.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = next
	}
	return cur
}

// Lookup returns the value at path, or nil if any level is missing or is not a
// JSON object. Unlike Object it never mutates m.
func Lookup(m map[string]interface{}, path ...string) interface{} {
	var cur interface{} = m
	for _, key := range path {
		obj, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur, ok = obj[key]
		if !ok {
			return nil
		}
	}
	return cur
}

// MergeMissing recursively copies keys from src into dst that dst does not
// already have. Where both sides hold a JSON object it recurses; otherwise
// dst's value stands. It never overwrites and never deletes, so a user's
// additions and edits survive.
//
// Values copied from src are deep-copied, so later mutation of src cannot
// reach into dst. Arrays are atomic: dst keeps its own array, or receives a
// copy of src's.
func MergeMissing(dst, src map[string]interface{}) {
	if dst == nil {
		return
	}
	for key, srcVal := range src {
		dstVal, present := dst[key]
		if !present || dstVal == nil {
			dst[key] = deepCopy(srcVal)
			continue
		}
		// Both sides objects: recurse so a user's edit to one nested field
		// does not block the rest of the subtree from being filled in.
		dstObj, dstIsObj := dstVal.(map[string]interface{})
		srcObj, srcIsObj := srcVal.(map[string]interface{})
		if dstIsObj && srcIsObj {
			MergeMissing(dstObj, srcObj)
		}
		// Otherwise dst wins: the user set it, we leave it.
	}
}

// SetIfBlank sets dst[key] to val when dst has no usable value there - the key
// is absent, or holds nil, or holds a string that is empty or only whitespace.
// It reports whether it wrote.
//
// This is a wider notion of "absent" than MergeMissing uses, for the case of a
// key the user may have deliberately blanked out rather than deleted.
func SetIfBlank(dst map[string]interface{}, key string, val interface{}) bool {
	if dst == nil {
		return false
	}
	existing, present := dst[key]
	if present && existing != nil {
		if s, isStr := existing.(string); !isStr || strings.TrimSpace(s) != "" {
			return false
		}
	}
	dst[key] = val
	return true
}

// Clone returns a deep copy of m, so a builder can take a caller's config,
// return a modified version, and leave the original untouched.
func Clone(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return make(map[string]interface{})
	}
	out, _ := deepCopy(m).(map[string]interface{})
	if out == nil {
		return make(map[string]interface{})
	}
	return out
}

func deepCopy(v interface{}) interface{} {
	switch typed := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, val := range typed {
			out[key] = deepCopy(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(typed))
		for i, val := range typed {
			out[i] = deepCopy(val)
		}
		return out
	default:
		// Scalars (string, bool, float64, nil) are immutable; share them.
		return v
	}
}

// Equal reports whether a and b represent the same JSON document.
//
// It compares after normalizing both sides through encoding/json rather than
// calling reflect.DeepEqual directly, and that is load-bearing. A config read
// from disk holds every number as float64, while one built from Go literals
// holds int. A direct DeepEqual would call those different forever, so callers
// that write on inequality would rewrite the file - and take a backup - on
// every single run. Normalizing also collapses typed structs to maps and
// []string to []interface{}.
func Equal(a, b interface{}) bool {
	normA, errA := Normalize(a)
	normB, errB := Normalize(b)
	if errA != nil || errB != nil {
		// Unmarshalable input: fall back to a direct comparison rather than
		// claiming equality, so a caller errs toward writing.
		return errA == nil && errB == nil && reflect.DeepEqual(a, b)
	}
	return reflect.DeepEqual(normA, normB)
}

// Normalize round-trips v through encoding/json, so that ints become float64,
// typed structs become map[string]interface{}, and []string becomes
// []interface{}. The result is what v would look like after being written to
// disk and read back.
func Normalize(v interface{}) (interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
