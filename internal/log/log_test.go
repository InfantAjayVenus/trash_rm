package log_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/markandeyan/trash-rm/internal/log"
)

func TestAppend_AppendsWithoutOverwriting(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "history.json")

	first := log.LogEntry{
		Timestamp: "2026-03-17T10:00:00Z",
		Command:   "trash-rm foo.txt",
		CWD:       "/home/user",
		Files:     []string{"foo.txt"},
	}
	second := log.LogEntry{
		Timestamp: "2026-03-17T11:00:00Z",
		Command:   "trash-rm bar.txt",
		CWD:       "/home/user",
		Files:     []string{"bar.txt"},
	}

	if err := log.Append(logPath, first); err != nil {
		t.Fatalf("first Append returned error: %v", err)
	}
	if err := log.Append(logPath, second); err != nil {
		t.Fatalf("second Append returned error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	lines := splitLines(data)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var e1, e2 log.LogEntry
	if err := json.Unmarshal([]byte(lines[0]), &e1); err != nil {
		t.Fatalf("failed to unmarshal line 1: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &e2); err != nil {
		t.Fatalf("failed to unmarshal line 2: %v", err)
	}

	if e1.Command != first.Command {
		t.Errorf("line 1 Command: got %q, want %q", e1.Command, first.Command)
	}
	if e2.Command != second.Command {
		t.Errorf("line 2 Command: got %q, want %q", e2.Command, second.Command)
	}
}

// splitLines returns non-empty lines from raw NDJSON bytes.
func splitLines(data []byte) []string {
	var lines []string
	start := 0
	for i, b := range data {
		if b == '\n' {
			line := string(data[start:i])
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	return lines
}

func TestReadAll_ReturnsAllEntriesInOrder(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "history.json")

	entries := []log.LogEntry{
		{Timestamp: "2026-03-17T08:00:00Z", Command: "trash-rm a.txt", CWD: "/home/user", Files: []string{"a.txt"}},
		{Timestamp: "2026-03-17T09:00:00Z", Command: "trash-rm b.txt", CWD: "/home/user", Files: []string{"b.txt"}},
		{Timestamp: "2026-03-17T10:00:00Z", Command: "trash-rm c.txt", CWD: "/home/user", Files: []string{"c.txt"}},
	}

	for _, e := range entries {
		if err := log.Append(logPath, e); err != nil {
			t.Fatalf("Append returned error: %v", err)
		}
	}

	got, err := log.ReadAll(logPath)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	for i, want := range entries {
		if got[i].Command != want.Command {
			t.Errorf("entry %d Command: got %q, want %q", i, got[i].Command, want.Command)
		}
		if got[i].Files[0] != want.Files[0] {
			t.Errorf("entry %d Files[0]: got %q, want %q", i, got[i].Files[0], want.Files[0])
		}
	}
}

func TestReadAll_EmptyWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "nonexistent.json")

	got, err := log.ReadAll(logPath)
	if err != nil {
		t.Fatalf("ReadAll returned unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(got))
	}
}

func TestRewrite_RemovesTargetEntry(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "history.json")

	entryA := log.LogEntry{Timestamp: "2026-03-17T08:00:00Z", Command: "trash-rm a.txt", CWD: "/home/user", Files: []string{"a.txt"}}
	entryB := log.LogEntry{Timestamp: "2026-03-17T09:00:00Z", Command: "trash-rm b.txt", CWD: "/home/user", Files: []string{"b.txt"}}
	entryC := log.LogEntry{Timestamp: "2026-03-17T10:00:00Z", Command: "trash-rm c.txt", CWD: "/home/user", Files: []string{"c.txt"}}

	for _, e := range []log.LogEntry{entryA, entryB, entryC} {
		if err := log.Append(logPath, e); err != nil {
			t.Fatalf("Append returned error: %v", err)
		}
	}

	// Rewrite without B
	if err := log.Rewrite(logPath, []log.LogEntry{entryA, entryC}); err != nil {
		t.Fatalf("Rewrite returned error: %v", err)
	}

	got, err := log.ReadAll(logPath)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 entries after rewrite, got %d", len(got))
	}
	if got[0].Command != entryA.Command {
		t.Errorf("entry 0 Command: got %q, want %q", got[0].Command, entryA.Command)
	}
	if got[1].Command != entryC.Command {
		t.Errorf("entry 1 Command: got %q, want %q", got[1].Command, entryC.Command)
	}
}

func TestReadAll_ErrorOnMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "history.json")

	if err := os.WriteFile(logPath, []byte("not json\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := log.ReadAll(logPath)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestAppend_ParentDirAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "history.json")

	entry := log.LogEntry{Timestamp: "2026-03-17T08:00:00Z", Command: "trash-rm a.txt", CWD: "/home/user", Files: []string{"a.txt"}}

	if err := log.Append(logPath, entry); err != nil {
		t.Fatalf("first Append returned error: %v", err)
	}
	if err := log.Append(logPath, entry); err != nil {
		t.Fatalf("second Append returned error: %v", err)
	}
}

func TestRewrite_EmptySliceClearsFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "history.json")

	entryA := log.LogEntry{Timestamp: "2026-03-17T08:00:00Z", Command: "trash-rm a.txt", CWD: "/home/user", Files: []string{"a.txt"}}
	entryB := log.LogEntry{Timestamp: "2026-03-17T09:00:00Z", Command: "trash-rm b.txt", CWD: "/home/user", Files: []string{"b.txt"}}

	for _, e := range []log.LogEntry{entryA, entryB} {
		if err := log.Append(logPath, e); err != nil {
			t.Fatalf("Append returned error: %v", err)
		}
	}

	if err := log.Rewrite(logPath, []log.LogEntry{}); err != nil {
		t.Fatalf("Rewrite returned error: %v", err)
	}

	got, err := log.ReadAll(logPath)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty log after Rewrite with empty slice, got %d entries", len(got))
	}
}

func TestAppend_CreatesFileAndWritesNDJSON(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "subdir", "history.json")

	entry := log.LogEntry{
		Timestamp: "2026-03-17T14:05:00Z",
		Command:   "trash-rm -rf old-project/",
		CWD:       "/home/user/projects",
		Files:     []string{"old-project/"},
	}

	if err := log.Append(logPath, entry); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	var got log.LogEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal written line: %v", err)
	}

	if got.Timestamp != entry.Timestamp {
		t.Errorf("Timestamp: got %q, want %q", got.Timestamp, entry.Timestamp)
	}
	if got.Command != entry.Command {
		t.Errorf("Command: got %q, want %q", got.Command, entry.Command)
	}
	if got.CWD != entry.CWD {
		t.Errorf("CWD: got %q, want %q", got.CWD, entry.CWD)
	}
	if len(got.Files) != 1 || got.Files[0] != entry.Files[0] {
		t.Errorf("Files: got %v, want %v", got.Files, entry.Files)
	}
}
