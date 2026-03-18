package log

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LogEntry records one trash-rm invocation in the sidecar log.
type LogEntry struct {
	Timestamp string   `json:"timestamp"`
	Command   string   `json:"command"`
	CWD       string   `json:"cwd"`
	Files     []string `json:"files"`
}

// Append marshals entry as a single JSON line and appends it to the file at
// path. Parent directories are created if they do not exist.
func Append(path string, entry LogEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = f.Write(append(line, '\n'))
	return err
}

// ReadAll reads all NDJSON entries from path in file order.
// If the file does not exist, it returns an empty slice and no error.
func ReadAll(path string) ([]LogEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("log line %d: parse failed: %w", lineNum, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// Rewrite atomically replaces the file at path with the given entries.
// It writes to a temporary file in the same directory then renames it,
// ensuring the operation is atomic on POSIX systems.
func Rewrite(path string, entries []LogEntry) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "log-rewrite-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	writeErr := writeEntries(tmp, entries)
	closeErr := tmp.Close()

	if writeErr != nil {
		os.Remove(tmpPath)
		return writeErr
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return closeErr
	}

	return os.Rename(tmpPath, path)
}

func writeEntries(f *os.File, entries []LogEntry) error {
	for _, entry := range entries {
		line, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	return nil
}
