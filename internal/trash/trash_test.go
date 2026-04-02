package trash_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/InfantAjayVenus/trash-rm/internal/trash"
)

// call records a single Commander invocation.
type call struct {
	name string
	args []string
}

// recordingCommander captures Commander calls and returns a no-op command.
func recordingCommander(calls *[]call) trash.Commander {
	return func(name string, args ...string) *exec.Cmd {
		*calls = append(*calls, call{name: name, args: args})
		return exec.Command("true")
	}
}

// failCommander returns a Commander that always exits non-zero,
// simulating a missing or unavailable binary.
func failCommander(name string, args ...string) *exec.Cmd {
	return exec.Command("false")
}

func TestLinuxBackend_CheckDependency_MissingTrashCli(t *testing.T) {
	backend := trash.LinuxBackend{Commander: failCommander}

	err := backend.CheckDependency()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "Install trash-cli") {
		t.Errorf("error %q does not contain %q", err.Error(), "Install trash-cli")
	}
}

func TestMacBackend_CheckDependency_MissingOsascript(t *testing.T) {
	backend := trash.MacBackend{Commander: failCommander}

	err := backend.CheckDependency()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "Requires osascript (bundled with macOS)") {
		t.Errorf("error %q does not contain expected osascript message", err.Error())
	}
}

func TestMacBackend_Trash_CallsOsascript(t *testing.T) {
	var calls []call
	backend := trash.MacBackend{Commander: recordingCommander(&calls)}

	if err := backend.Trash([]string{"/Users/alice/foo.txt"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 Commander call, got %d", len(calls))
	}
	if calls[0].name != "osascript" {
		t.Errorf("expected command %q, got %q", "osascript", calls[0].name)
	}
	wantScript := `tell application "Finder" to delete POSIX file "/Users/alice/foo.txt"`
	if len(calls[0].args) != 2 || calls[0].args[0] != "-e" || calls[0].args[1] != wantScript {
		t.Errorf("expected args [-e %q], got %v", wantScript, calls[0].args)
	}
}

func TestLinuxBackend_Trash_CallsTrashCommand(t *testing.T) {
	var calls []call
	backend := trash.LinuxBackend{Commander: recordingCommander(&calls)}

	if err := backend.Trash([]string{"foo.txt"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 Commander call, got %d", len(calls))
	}
	if calls[0].name != "trash" {
		t.Errorf("expected command %q, got %q", "trash", calls[0].name)
	}
	if len(calls[0].args) != 1 || calls[0].args[0] != "foo.txt" {
		t.Errorf("expected args [%q], got %v", "foo.txt", calls[0].args)
	}
}
