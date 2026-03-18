package app_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/markandeyan/trash-rm/internal/app"
	"github.com/markandeyan/trash-rm/internal/args"
	"github.com/markandeyan/trash-rm/internal/log"
	"github.com/markandeyan/trash-rm/internal/trash"
)

// recordingCommander records calls and delegates to the given exec.Cmd factory.
type call struct {
	name string
	args []string
}

func recordingCommander(calls *[]call, factory func(name string, a ...string) *exec.Cmd) app.Commander {
	return func(name string, a ...string) *exec.Cmd {
		*calls = append(*calls, call{name: name, args: a})
		return factory(name, a...)
	}
}

func successCommander(calls *[]call) app.Commander {
	return recordingCommander(calls, func(name string, a ...string) *exec.Cmd {
		return exec.Command("true")
	})
}

func successTrashCommander(calls *[]call) trash.Commander {
	return func(name string, a ...string) *exec.Cmd {
		*calls = append(*calls, call{name: name, args: a})
		return exec.Command("true")
	}
}

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func alwaysPickFirst(entries []log.LogEntry) (int, error) {
	return 0, nil
}

// TestRun_TrashFail_UserConfirms_RmRuns verifies that when trash fails but the user
// confirms the prompt, rm is still called and no log entry is written.
func TestRun_TrashFail_UserConfirms_RmRuns(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "history.json")

	var rmCalls []call
	// Commander: pass checkDependency ("trash --version") but fail Trash ("trash <file>").
	failTrash := trash.LinuxBackend{
		Commander: func(name string, a ...string) *exec.Cmd {
			if len(a) > 0 && a[0] == "--version" {
				return exec.Command("true") // checkDependency passes
			}
			return exec.Command("false") // actual trash call fails
		},
	}

	parsed := args.ParsedArgs{Files: []string{"foo.txt"}, RmFlags: []string{}}

	promptCalled := false
	cfg := app.Config{
		Parsed:  parsed,
		Backend: failTrash,
		Commander: func(name string, a ...string) *exec.Cmd {
			rmCalls = append(rmCalls, call{name: name, args: a})
			return exec.Command("true")
		},
		RmPath:  "/bin/true",
		LogPath: logPath,
		IsTTY:   true,
		PromptFn: func(msg string) (string, error) {
			promptCalled = true
			return "y", nil
		},
		SelectFn: alwaysPickFirst,
		CWD:      "/home/alice",
		Now:      fixedClock(time.Date(2026, 3, 17, 14, 5, 0, 0, time.UTC)),
		Out:      &bytes.Buffer{},
	}

	code := app.Run(cfg)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !promptCalled {
		t.Error("expected prompt to be called but it was not")
	}
	// rm must be called
	foundRm := false
	for _, c := range rmCalls {
		if c.name == "/bin/true" {
			foundRm = true
		}
	}
	if !foundRm {
		t.Errorf("expected rm to be called; calls: %v", rmCalls)
	}
	// No log entry must be written
	entries, err := log.ReadAll(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no log entry on trash failure, got %d entries", len(entries))
	}
}

// TestRun_TrashFail_UserDeclines_Aborts verifies that when trash fails and the user
// declines the prompt, rm is NOT called and a non-zero exit code is returned.
func TestRun_TrashFail_UserDeclines_Aborts(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "history.json")

	failTrash := trash.LinuxBackend{
		Commander: func(name string, a ...string) *exec.Cmd {
			if len(a) > 0 && a[0] == "--version" {
				return exec.Command("true")
			}
			return exec.Command("false")
		},
	}

	rmCalled := false
	parsed := args.ParsedArgs{Files: []string{"foo.txt"}, RmFlags: []string{}}
	cfg := app.Config{
		Parsed:  parsed,
		Backend: failTrash,
		Commander: func(name string, a ...string) *exec.Cmd {
			rmCalled = true
			return exec.Command("true")
		},
		RmPath:   "/bin/true",
		LogPath:  logPath,
		IsTTY:    true,
		PromptFn: func(msg string) (string, error) { return "n", nil },
		SelectFn: alwaysPickFirst,
		CWD:      "/home/alice",
		Now:      fixedClock(time.Date(2026, 3, 17, 14, 5, 0, 0, time.UTC)),
		Out:      &bytes.Buffer{},
	}

	code := app.Run(cfg)

	if code == 0 {
		t.Fatal("expected non-zero exit code, got 0")
	}
	if rmCalled {
		t.Error("expected rm NOT to be called when user declines")
	}
}

// TestRun_NonTTY_TrashFail_AutoAborts verifies that when trash fails in a non-TTY context
// the prompt is never called, rm is not called, and the exit code is non-zero.
func TestRun_NonTTY_TrashFail_AutoAborts(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "history.json")

	failTrash := trash.LinuxBackend{
		Commander: func(name string, a ...string) *exec.Cmd {
			if len(a) > 0 && a[0] == "--version" {
				return exec.Command("true")
			}
			return exec.Command("false")
		},
	}

	promptCalled := false
	rmCalled := false
	parsed := args.ParsedArgs{Files: []string{"foo.txt"}, RmFlags: []string{}}
	cfg := app.Config{
		Parsed:  parsed,
		Backend: failTrash,
		Commander: func(name string, a ...string) *exec.Cmd {
			rmCalled = true
			return exec.Command("true")
		},
		RmPath:  "/bin/true",
		LogPath: logPath,
		IsTTY:   false, // non-TTY
		PromptFn: func(msg string) (string, error) {
			promptCalled = true
			return "y", nil
		},
		SelectFn: alwaysPickFirst,
		CWD:      "/home/alice",
		Now:      fixedClock(time.Date(2026, 3, 17, 14, 5, 0, 0, time.UTC)),
		Out:      &bytes.Buffer{},
	}

	code := app.Run(cfg)

	if code == 0 {
		t.Fatal("expected non-zero exit code in non-TTY context, got 0")
	}
	if promptCalled {
		t.Error("expected prompt NOT to be called in non-TTY context")
	}
	if rmCalled {
		t.Error("expected rm NOT to be called in non-TTY context on trash failure")
	}
}

// TestRun_LogWriteFail_BlocksRm verifies that if the log write fails (e.g. unwritable dir),
// rm is NOT called and a non-zero exit code is returned.
func TestRun_LogWriteFail_BlocksRm(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "subdir", "history.json")

	// Make the parent directory unwritable so log.Append fails
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(logPath), 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(logPath), 0o755) //nolint:errcheck

	var calls []call
	backend := trash.LinuxBackend{Commander: successTrashCommander(&calls)}
	rmCalled := false
	parsed := args.ParsedArgs{Files: []string{"foo.txt"}, RmFlags: []string{}}

	cfg := app.Config{
		Parsed:  parsed,
		Backend: backend,
		Commander: func(name string, a ...string) *exec.Cmd {
			rmCalled = true
			return exec.Command("true")
		},
		RmPath:   "/bin/true",
		LogPath:  logPath,
		IsTTY:    true,
		PromptFn: func(msg string) (string, error) { return "y", nil },
		SelectFn: alwaysPickFirst,
		CWD:      "/home/alice",
		Now:      fixedClock(time.Date(2026, 3, 17, 14, 5, 0, 0, time.UTC)),
		Out:      &bytes.Buffer{},
	}

	code := app.Run(cfg)

	if code == 0 {
		t.Fatal("expected non-zero exit code when log write fails, got 0")
	}
	if rmCalled {
		t.Error("expected rm NOT to be called when log write fails")
	}
}

// TestRun_SkipTrash_CallsRmDirectly verifies that --be-brave-skip-trash causes rm
// to be called directly with no trash call and no log entry written.
func TestRun_SkipTrash_CallsRmDirectly(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "history.json")

	trashCalled := false
	backend := &spyTrasher{
		checkErr: nil,
		trashFn: func(files []string) error {
			trashCalled = true
			return nil
		},
	}

	var rmCalls []call
	parsed := args.ParsedArgs{
		Files:     []string{"foo.txt"},
		RmFlags:   []string{},
		SkipTrash: true,
	}
	cfg := app.Config{
		Parsed:  parsed,
		Backend: backend,
		Commander: func(name string, a ...string) *exec.Cmd {
			rmCalls = append(rmCalls, call{name: name, args: a})
			return exec.Command("true")
		},
		RmPath:   "/bin/true",
		LogPath:  logPath,
		IsTTY:    true,
		PromptFn: func(msg string) (string, error) { return "", nil },
		SelectFn: alwaysPickFirst,
		CWD:      "/home/alice",
		Now:      fixedClock(time.Date(2026, 3, 17, 14, 5, 0, 0, time.UTC)),
		Out:      &bytes.Buffer{},
	}

	code := app.Run(cfg)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if trashCalled {
		t.Error("expected trash NOT to be called with --be-brave-skip-trash")
	}

	// rm must be called
	if len(rmCalls) == 0 {
		t.Error("expected rm to be called with --be-brave-skip-trash")
	}

	// No log entry
	entries, err := log.ReadAll(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no log entry with --be-brave-skip-trash, got %d", len(entries))
	}
}

// spyTrasher is a test double for trash.Trasher.
type spyTrasher struct {
	checkErr error
	trashFn  func(files []string) error
}

func (s *spyTrasher) CheckDependency() error { return s.checkErr }
func (s *spyTrasher) Trash(files []string) error {
	if s.trashFn != nil {
		return s.trashFn(files)
	}
	return nil
}

// TestRun_ExitCodeMirrorsRm verifies that when rm exits with a non-zero code,
// app.Run returns that same exit code.
func TestRun_ExitCodeMirrorsRm(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "history.json")

	backend := trash.LinuxBackend{Commander: func(name string, a ...string) *exec.Cmd {
		if len(a) > 0 && a[0] == "--version" {
			return exec.Command("true")
		}
		return exec.Command("true") // trash succeeds
	}}

	parsed := args.ParsedArgs{Files: []string{"foo.txt"}, RmFlags: []string{}}
	cfg := app.Config{
		Parsed:  parsed,
		Backend: backend,
		Commander: func(name string, a ...string) *exec.Cmd {
			// rm exits with code 2
			return exec.Command("sh", "-c", "exit 2")
		},
		RmPath:   "/bin/sh",
		LogPath:  logPath,
		IsTTY:    true,
		PromptFn: func(msg string) (string, error) { return "y", nil },
		SelectFn: alwaysPickFirst,
		CWD:      "/home/alice",
		Now:      fixedClock(time.Date(2026, 3, 17, 14, 5, 0, 0, time.UTC)),
		Out:      &bytes.Buffer{},
	}

	code := app.Run(cfg)

	if code != 2 {
		t.Fatalf("expected exit code 2 to mirror rm, got %d", code)
	}
}

// TestRun_HappyPath_TrashThenRm verifies that on a normal invocation:
// 1. Trash is called for each file
// 2. A log entry is written
// 3. rm is called with the file
func TestRun_HappyPath_TrashThenRm(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "history.json")

	var calls []call
	backend := trash.LinuxBackend{Commander: successTrashCommander(&calls)}

	parsed := args.ParsedArgs{
		Files:   []string{"foo.txt"},
		RmFlags: []string{},
	}

	rmPath := "/bin/true" // use /bin/true as a stand-in for rm so it actually runs
	cfg := app.Config{
		Parsed:    parsed,
		Backend:   backend,
		Commander: successCommander(&calls),
		RmPath:    rmPath,
		LogPath:   logPath,
		IsTTY:     true,
		PromptFn:  func(msg string) (string, error) { return "y", nil },
		SelectFn:  alwaysPickFirst,
		CWD:       "/home/alice",
		Now:       fixedClock(time.Date(2026, 3, 17, 14, 5, 0, 0, time.UTC)),
		Out:       &bytes.Buffer{},
	}

	code := app.Run(cfg)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// Verify trash was called with the file
	foundTrash := false
	for _, c := range calls {
		if c.name == "trash" && len(c.args) == 1 && c.args[0] == "foo.txt" {
			foundTrash = true
		}
	}
	if !foundTrash {
		t.Errorf("expected Commander to be called with (trash, foo.txt); calls: %v", calls)
	}

	// Verify log was written
	entries, err := log.ReadAll(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if !strings.Contains(strings.Join(entries[0].Files, ","), "foo.txt") {
		t.Errorf("log entry missing foo.txt; got files: %v", entries[0].Files)
	}
}
