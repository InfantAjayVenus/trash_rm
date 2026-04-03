package restore_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/InfantAjayVenus/trash-rm/internal/log"
	"github.com/InfantAjayVenus/trash-rm/internal/restore"
)

// recordedCall captures one Commander invocation.
type recordedCall struct {
	name string
	args []string
}

// recordingCommander returns a Commander that records all calls and runs "true" (success).
func recordingCommander(calls *[]recordedCall) restore.Commander {
	return func(name string, args ...string) *exec.Cmd {
		*calls = append(*calls, recordedCall{name: name, args: args})
		return exec.Command("true")
	}
}

func TestRun_EmptyLog(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/history.json" // does not exist

	alwaysPickFirst := func(entries []log.LogEntry) (int, error) { return 0, nil }

	var out strings.Builder
	err := restore.Run(logPath, alwaysPickFirst, exec.Command, &out)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !strings.Contains(out.String(), "No trash history found.") {
		t.Errorf("expected 'No trash history found.' in output, got: %q", out.String())
	}
}

func TestFilterAlive_ReturnsOnlyLiveEntries(t *testing.T) {
	entries := []log.LogEntry{
		{Timestamp: "2026-01-01T00:00:00Z", Command: "trash-rm foo.txt", CWD: "/home/alice", Files: []string{"foo.txt"}},
		{Timestamp: "2026-01-02T00:00:00Z", Command: "trash-rm bar.txt", CWD: "/home/alice", Files: []string{"bar.txt"}},
	}

	trashListOutput := "/home/alice/foo.txt\n2026-01-01 00:00:00 /home/alice/foo.txt\n"

	alive := restore.FilterAlive(entries, trashListOutput)

	if len(alive) != 1 {
		t.Fatalf("expected 1 alive entry, got %d", len(alive))
	}
	if alive[0].Files[0] != "foo.txt" {
		t.Errorf("expected alive entry for foo.txt, got %s", alive[0].Files[0])
	}
}

func TestRestoreEntry_PermanentlyDeletedError(t *testing.T) {
	entry := log.LogEntry{
		Timestamp: "2026-01-01T00:00:00Z",
		Command:   "trash-rm gone.txt",
		CWD:       "/home/alice",
		Files:     []string{"gone.txt"},
	}

	// Commander: trash-restore fails; trash-list returns output without gone.txt
	commander := func(name string, args ...string) *exec.Cmd {
		if name == "trash-restore" {
			return exec.Command("false")
		}
		// trash-list — returns output that does NOT contain gone.txt
		return exec.Command("echo", "2026-01-01 00:00:00 /home/alice/other.txt")
	}

	err := restore.RestoreEntry(entry, commander)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "permanently deleted from trash") {
		t.Errorf("expected 'permanently deleted from trash' in error, got: %v", err)
	}
}

func TestRun_RemovesEntryAfterRestore(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/history.json"

	entryA := log.LogEntry{Timestamp: "2026-01-01T00:00:00Z", Command: "trash-rm a.txt", CWD: "/tmp", Files: []string{"a.txt"}}
	entryB := log.LogEntry{Timestamp: "2026-01-02T00:00:00Z", Command: "trash-rm b.txt", CWD: "/tmp", Files: []string{"b.txt"}}

	if err := log.Append(logPath, entryA); err != nil {
		t.Fatalf("setup: append entryA: %v", err)
	}
	if err := log.Append(logPath, entryB); err != nil {
		t.Fatalf("setup: append entryB: %v", err)
	}

	// SelectFunc always picks index 0 (entryA)
	alwaysPickFirst := func(entries []log.LogEntry) (int, error) { return 0, nil }

	trashListCalls := 0

	// Commander: both files are initially in trash; after restoring a.txt only b.txt remains.
	successCommander := func(name string, args ...string) *exec.Cmd {
		if name == "trash-list" {
			trashListCalls++
			switch trashListCalls {
			case 1:
				return exec.Command("echo", "/tmp/a.txt\n/tmp/b.txt")
			default:
				return exec.Command("echo", "/tmp/b.txt")
			}
		}
		return exec.Command("true")
	}

	var out strings.Builder
	err := restore.Run(logPath, alwaysPickFirst, successCommander, &out)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	remaining, err := log.ReadAll(logPath)
	if err != nil {
		t.Fatalf("ReadAll after Run: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining entry, got %d", len(remaining))
	}
	if remaining[0].Files[0] != "b.txt" {
		t.Errorf("expected remaining entry to be b.txt, got %s", remaining[0].Files[0])
	}
}

func TestFilterAlive_AllGarbageCollected(t *testing.T) {
	entries := []log.LogEntry{
		{Timestamp: "2026-01-01T00:00:00Z", Command: "trash-rm foo.txt", CWD: "/home/alice", Files: []string{"foo.txt"}},
		{Timestamp: "2026-01-02T00:00:00Z", Command: "trash-rm bar.txt", CWD: "/home/alice", Files: []string{"bar.txt"}},
	}

	// trash-list output contains neither foo.txt nor bar.txt
	trashListOutput := "2026-01-03 00:00:00 /home/alice/other.txt\n"

	alive := restore.FilterAlive(entries, trashListOutput)

	if len(alive) != 0 {
		t.Errorf("expected 0 alive entries, got %d", len(alive))
	}
}

func TestRestoreEntry_CallsTrashRestorePerFile(t *testing.T) {
	entry := log.LogEntry{
		Timestamp: "2026-01-01T00:00:00Z",
		Command:   "trash-rm foo.txt bar.txt",
		CWD:       "/home/alice",
		Files:     []string{"foo.txt", "bar.txt"},
	}

	var calls []recordedCall
	commander := recordingCommander(&calls)

	if err := restore.RestoreEntry(entry, commander); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(calls) != 4 {
		t.Fatalf("expected 4 commander calls, got %d", len(calls))
	}
	if calls[0].name != "trash-restore" || calls[0].args[0] != "/home/alice/foo.txt" {
		t.Errorf("first call: got (%s, %v), want (trash-restore, [/home/alice/foo.txt])", calls[0].name, calls[0].args)
	}
	if calls[1].name != "trash-list" {
		t.Errorf("second call: got (%s, %v), want (trash-list, [])", calls[1].name, calls[1].args)
	}
	if calls[2].name != "trash-restore" || calls[2].args[0] != "/home/alice/bar.txt" {
		t.Errorf("third call: got (%s, %v), want (trash-restore, [/home/alice/bar.txt])", calls[2].name, calls[2].args)
	}
	if calls[3].name != "trash-list" {
		t.Errorf("fourth call: got (%s, %v), want (trash-list, [])", calls[3].name, calls[3].args)
	}
}

func TestRun_UserQuitsWithoutRestoring(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/history.json"

	entry := log.LogEntry{Timestamp: "2026-03-17T10:00:00Z", Command: "rm foo.txt", CWD: "/home/user", Files: []string{"foo.txt"}}
	if err := log.Append(logPath, entry); err != nil {
		t.Fatal(err)
	}

	quitFn := func(entries []log.LogEntry) (int, error) { return -1, nil }
	trashListCommander := func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "/home/user/foo.txt") // file appears in trash-list so it passes FilterAlive
	}

	var out strings.Builder
	if err := restore.Run(logPath, quitFn, trashListCommander, &out); err != nil {
		t.Fatalf("expected nil error on quit, got: %v", err)
	}

	remaining, _ := log.ReadAll(logPath)
	if len(remaining) != 1 {
		t.Errorf("log should be unchanged after quit, got %d entries", len(remaining))
	}
}

func TestRestoreEntry_ReturnsErrorWhenFileStillInTrashAfterCommandSucceeds(t *testing.T) {
	entry := log.LogEntry{
		Timestamp: "2026-01-01T00:00:00Z",
		Command:   "trash-rm foo.txt",
		CWD:       "/home/alice",
		Files:     []string{"foo.txt"},
	}

	commander := func(name string, args ...string) *exec.Cmd {
		if name == "trash-restore" {
			return exec.Command("echo", "No files trashed from current dir")
		}
		return exec.Command("echo", "/home/alice/foo.txt")
	}

	err := restore.RestoreEntry(entry, commander)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to restore /home/alice/foo.txt from trash") {
		t.Fatalf("unexpected error: %v", err)
	}
}
