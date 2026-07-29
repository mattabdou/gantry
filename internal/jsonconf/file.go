package jsonconf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupSuffix is the extension GANTRY appends to timestamped config backups.
const BackupSuffix = "gantrybackup"

// ReadObject reads path and unmarshals it into a JSON object. A missing file
// yields an empty, non-nil map and no error, so callers can treat "no config
// yet" and "empty config" the same way.
func ReadObject(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}
	return UnmarshalObject(data)
}

// UnmarshalObject unmarshals data into a JSON object. Empty or whitespace-only
// input yields an empty map, matching the missing-file case in ReadObject.
func UnmarshalObject(data []byte) (map[string]interface{}, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	if obj == nil {
		// Valid JSON "null", or an empty document.
		obj = make(map[string]interface{})
	}
	return obj, nil
}

// WriteObject marshals m with two-space indent and writes it to path.
//
// The write goes to a temporary file in the same directory and is then renamed
// over the target, so a reader never observes a half-written config. That
// matters because `gantry exec` is spawned once per IDE request and configures
// the tool on each invocation, so concurrent writers exist in normal use.
func WriteObject(path string, m map[string]interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".gantrytmp*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename below succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	// CreateTemp makes the file 0600; config files are conventionally 0644.
	if err := os.Chmod(tmpName, 0644); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to replace config: %w", err)
	}
	return nil
}

// Backup copies path to path.<timestamp>.gantrybackup and returns the backup
// path. A missing source file is not an error and yields an empty path.
//
// The timestamp has second resolution, so two backups of the same file within
// one second would collide. Rather than let the second overwrite the first -
// silently discarding a state the user might need - a numeric suffix is added
// until the name is free. This happens in practice: enabling and then disabling
// powerline in a single run backs up twice in quick succession.
func Backup(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read config for backup: %w", err)
	}

	base := fmt.Sprintf("%s.%s", path, time.Now().Format("2006-01-02_15-04-05"))
	backupPath := fmt.Sprintf("%s.%s", base, BackupSuffix)
	for attempt := 2; ; attempt++ {
		// O_EXCL fails if the name is taken, which both detects the collision and
		// claims the name without a race.
		file, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			_, writeErr := file.Write(data)
			closeErr := file.Close()
			if writeErr != nil {
				return "", fmt.Errorf("failed to write backup: %w", writeErr)
			}
			if closeErr != nil {
				return "", fmt.Errorf("failed to write backup: %w", closeErr)
			}
			return backupPath, nil
		}
		if !os.IsExist(err) {
			return "", fmt.Errorf("failed to write backup: %w", err)
		}
		if attempt > 100 {
			return "", fmt.Errorf("failed to write backup: too many backups of %s in the same second", filepath.Base(path))
		}
		backupPath = fmt.Sprintf("%s-%d.%s", base, attempt, BackupSuffix)
	}
}
