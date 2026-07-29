package jsonconf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadObject(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name     string
		contents string // "" means do not create the file
		create   bool
		wantErr  bool
		wantLen  int
	}{
		{name: "missing file yields empty map", create: false, wantLen: 0},
		{name: "empty object", create: true, contents: `{}`, wantLen: 0},
		{name: "json null yields empty map", create: true, contents: `null`, wantLen: 0},
		{name: "populated object", create: true, contents: `{"a":1,"b":2}`, wantLen: 2},
		{name: "invalid json errors", create: true, contents: `{"a":`, wantErr: true},
		{name: "array is not an object", create: true, contents: `[1,2]`, wantErr: true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, "cfg", string(rune('a'+i))+".json")
			if tt.create {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(tt.contents), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got, err := ReadObject(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ReadObject() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadObject() error = %v", err)
			}
			if got == nil {
				t.Fatal("ReadObject() = nil, want a non-nil map")
			}
			if len(got) != tt.wantLen {
				t.Errorf("len(ReadObject()) = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestWriteObjectRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "opencode.json")
	want := map[string]interface{}{
		"model": "gantry-litellm/claude-opus-5",
		"provider": map[string]interface{}{
			"gantry-litellm": map[string]interface{}{"npm": "@ai-sdk/openai-compatible"},
		},
	}

	if err := WriteObject(path, want); err != nil {
		t.Fatalf("WriteObject() error = %v", err)
	}

	got, err := ReadObject(path)
	if err != nil {
		t.Fatalf("ReadObject() error = %v", err)
	}
	if !Equal(got, want) {
		t.Errorf("round trip lost data:\n got %#v\nwant %#v", got, want)
	}
}

func TestWriteObjectFormatting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.json")
	if err := WriteObject(path, map[string]interface{}{"a": map[string]interface{}{"b": "c"}}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)

	if !strings.Contains(text, "\n  \"a\"") {
		t.Errorf("want two-space indent, got:\n%s", text)
	}
	if !strings.HasSuffix(text, "\n") {
		t.Error("want a trailing newline")
	}
}

func TestWriteObjectLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := WriteObject(path, map[string]interface{}{"a": "b"}); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), "gantrytmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("want exactly 1 file in the directory, got %d", len(entries))
	}
}

func TestWriteObjectPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.json")
	if err := WriteObject(path, map[string]interface{}{"a": "b"}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// CreateTemp yields 0600; config files should end up readable.
	if perm := info.Mode().Perm(); perm != 0644 {
		t.Errorf("mode = %o, want 644", perm)
	}
}

func TestWriteObjectOverwritesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.json")
	if err := os.WriteFile(path, []byte(`{"old":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteObject(path, map[string]interface{}{"new": true}); err != nil {
		t.Fatal(err)
	}

	got, err := ReadObject(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, stale := got["old"]; stale {
		t.Error("old content survived the overwrite")
	}
	if got["new"] != true {
		t.Errorf("new content missing: %#v", got)
	}
}

func TestBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")
	contents := `{"mcp":{"brave":{"enabled":true}}}`
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}

	backupPath, err := Backup(path)
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}
	if backupPath == "" {
		t.Fatal("Backup() returned an empty path for an existing file")
	}
	if !strings.HasSuffix(backupPath, "."+BackupSuffix) {
		t.Errorf("backup path %q lacks the %q suffix", backupPath, BackupSuffix)
	}
	if !strings.HasPrefix(backupPath, path+".") {
		t.Errorf("backup path %q is not derived from %q", backupPath, path)
	}

	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(got) != contents {
		t.Errorf("backup contents = %q, want %q", got, contents)
	}

	// The original must still be there, untouched.
	orig, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading original: %v", err)
	}
	if string(orig) != contents {
		t.Errorf("original was modified: %q", orig)
	}
}

// Two backups of the same file inside one second must not collide. The
// timestamp only has second resolution, so without a uniquifying suffix the
// second backup would silently overwrite the first - discarding a state the user
// may need. Enabling then disabling powerline in a single run does exactly this.
func TestBackupDoesNotOverwriteAnEarlierBackupInTheSameSecond(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	if err := os.WriteFile(path, []byte(`{"state":"first"}`), 0644); err != nil {
		t.Fatal(err)
	}
	firstBackup, err := Backup(path)
	if err != nil {
		t.Fatal(err)
	}

	// Change the file and back it up again, immediately.
	if err := os.WriteFile(path, []byte(`{"state":"second"}`), 0644); err != nil {
		t.Fatal(err)
	}
	secondBackup, err := Backup(path)
	if err != nil {
		t.Fatal(err)
	}

	if firstBackup == secondBackup {
		t.Fatalf("both backups used the same path %q; the first state was lost", firstBackup)
	}

	first, err := os.ReadFile(firstBackup)
	if err != nil {
		t.Fatalf("the first backup is gone: %v", err)
	}
	if string(first) != `{"state":"first"}` {
		t.Errorf("first backup = %q, want the original state", first)
	}
	second, err := os.ReadFile(secondBackup)
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != `{"state":"second"}` {
		t.Errorf("second backup = %q", second)
	}

	// Both must still be on disk, and both must carry the backup suffix.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	backups := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "."+BackupSuffix) {
			backups++
		}
	}
	if backups != 2 {
		t.Errorf("%d backup files on disk, want 2", backups)
	}
}

func TestBackupManyInTheSameSecond(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	for i := 0; i < 12; i++ {
		backupPath, err := Backup(path)
		if err != nil {
			t.Fatalf("backup %d: %v", i, err)
		}
		if seen[backupPath] {
			t.Fatalf("backup %d reused path %q", i, backupPath)
		}
		seen[backupPath] = true
	}
}

func TestBackupMissingFileIsNotAnError(t *testing.T) {
	backupPath, err := Backup(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("Backup() error = %v, want nil", err)
	}
	if backupPath != "" {
		t.Errorf("Backup() = %q, want an empty path", backupPath)
	}
}

func TestUnmarshalObjectRejectsNonObjects(t *testing.T) {
	for _, input := range []string{`[1,2]`, `"str"`, `42`, `{"a":`} {
		if _, err := UnmarshalObject([]byte(input)); err == nil {
			t.Errorf("UnmarshalObject(%q) error = nil, want an error", input)
		}
	}
}

func TestUnmarshalObjectAcceptsNull(t *testing.T) {
	got, err := UnmarshalObject([]byte(`null`))
	if err != nil {
		t.Fatalf("UnmarshalObject() error = %v", err)
	}
	if got == nil {
		t.Fatal("UnmarshalObject(null) = nil, want an empty non-nil map")
	}
}

// Guards the marshal shape the write path depends on: Go sorts map keys, so a
// semantically unchanged config never gets reordered on disk.
func TestMarshalKeyOrderIsStable(t *testing.T) {
	m := map[string]interface{}{"z": 1, "a": 2, "m": 3}
	first, _ := json.Marshal(m)
	for i := 0; i < 5; i++ {
		next, _ := json.Marshal(m)
		if string(next) != string(first) {
			t.Fatalf("marshal output varies between calls:\n%s\n%s", first, next)
		}
	}
}
