package restore

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/InfantAjayVenus/trash-rm/internal/log"
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
		if anyFileInTrashList(entry, trashListOutput) {
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
		originalPath := originalPathFor(entry.CWD, f)
		cmd := commander("trash-restore", originalPath)
		cmd.Stdin = strings.NewReader("0\n")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			return permanentlyDeletedOrError(originalPath, commander, err, out.String())
		}
		if fileStillInTrash(originalPath, commander) {
			return fmt.Errorf("failed to restore %s from trash", originalPath)
		}
	}
	return nil
}

func permanentlyDeletedOrError(file string, commander Commander, runErr error, output string) error {
	trashListOutput := trashListContents(commander)
	if !strings.Contains(trashListOutput, file) {
		return fmt.Errorf("Cannot restore %s: it has been permanently deleted from trash", file)
	}
	output = strings.TrimSpace(output)
	if output != "" {
		return fmt.Errorf("failed to restore %s from trash: %s", file, output)
	}
	return fmt.Errorf("failed to restore %s from trash: %v", file, runErr)
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

func anyFileInTrashList(entry log.LogEntry, trashListOutput string) bool {
	for _, f := range entry.Files {
		if strings.Contains(trashListOutput, originalPathFor(entry.CWD, f)) {
			return true
		}
	}
	return false
}

func fileStillInTrash(file string, commander Commander) bool {
	return strings.Contains(trashListContents(commander), file)
}

func originalPathFor(cwd, file string) string {
	if filepath.IsAbs(file) {
		return filepath.Clean(file)
	}
	return filepath.Clean(filepath.Join(cwd, file))
}
