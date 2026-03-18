package restore

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/markandeyan/trash-rm/internal/log"
)

// Commander constructs an exec.Cmd, used as a seam for testing.
type Commander func(name string, args ...string) *exec.Cmd

// SelectFunc allows the caller to inject a selection mechanism.
// It receives the alive log entries and returns the index of the chosen one.
type SelectFunc func(entries []log.LogEntry) (int, error)

// FilterAlive returns only those entries whose files appear in trashListOutput.
// This cross-references the sidecar log against `trash-list` to skip GC'd entries.
func FilterAlive(entries []log.LogEntry, trashListOutput string) []log.LogEntry {
	var alive []log.LogEntry
	for _, entry := range entries {
		if anyFileInTrashList(entry.Files, trashListOutput) {
			alive = append(alive, entry)
		}
	}
	return alive
}

// RestoreEntry calls `trash-restore` for each file in the entry.
// If trash-restore fails and the file is absent from trash-list, it returns
// an explicit "permanently deleted from trash" error message.
func RestoreEntry(entry log.LogEntry, commander Commander) error {
	for _, f := range entry.Files {
		cmd := commander("trash-restore", f)
		if err := cmd.Run(); err != nil {
			return permanentlyDeletedOrError(f, commander)
		}
	}
	return nil
}

func permanentlyDeletedOrError(file string, commander Commander) error {
	trashListOutput := trashListContents(commander)
	if !strings.Contains(trashListOutput, file) {
		return fmt.Errorf("Cannot restore %s: it has been permanently deleted from trash", file)
	}
	return fmt.Errorf("failed to restore %s from trash", file)
}

func trashListContents(commander Commander) string {
	cmd := commander("trash-list")
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()
	return out.String()
}

// Run orchestrates the --restore flow: reads the log, filters alive entries,
// calls selectFn for user selection, restores the chosen entry, and rewrites the log.
// Messages are written to out. Returns nil on success or if history is empty.
func Run(logPath string, selectFn SelectFunc, commander Commander, out io.Writer) error {
	entries, err := log.ReadAll(logPath)
	if err != nil {
		return fmt.Errorf("cannot read trash history: %w", err)
	}
	if len(entries) == 0 {
		fmt.Fprintln(out, "No trash history found.")
		return nil
	}

	trashList := trashListContents(commander)
	alive := FilterAlive(entries, trashList)
	if len(alive) == 0 {
		fmt.Fprintln(out, "No trash history found.")
		return nil
	}

	chosen, err := selectFn(alive)
	if err != nil {
		return err
	}
	if chosen < 0 {
		return nil
	}

	selectedEntry := alive[chosen]
	if err := RestoreEntry(selectedEntry, commander); err != nil {
		return err
	}

	return rewriteWithoutEntry(logPath, entries, selectedEntry)
}

// rewriteWithoutEntry rewrites the log excluding the entry that was just restored.
func rewriteWithoutEntry(logPath string, entries []log.LogEntry, restored log.LogEntry) error {
	remaining := make([]log.LogEntry, 0, len(entries)-1)
	for _, e := range entries {
		if e.Timestamp != restored.Timestamp || e.Command != restored.Command {
			remaining = append(remaining, e)
		}
	}
	return log.Rewrite(logPath, remaining)
}

func anyFileInTrashList(files []string, trashListOutput string) bool {
	for _, f := range files {
		if strings.Contains(trashListOutput, f) {
			return true
		}
	}
	return false
}
