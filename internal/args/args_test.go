package args_test

import (
	"testing"

	"github.com/InfantAjayVenus/trash-rm/internal/args"
)

func TestCalledAsRm_False(t *testing.T) {
	if args.CalledAsRm("/usr/local/bin/trash-rm") {
		t.Errorf("expected CalledAsRm(\"/usr/local/bin/trash-rm\") to return false")
	}
}

func TestParse_NoFiles_ReturnsError(t *testing.T) {
	_, err := args.Parse([]string{"-rf"})
	if err == nil {
		t.Error("expected an error when no files are specified, got nil")
	}
}

func TestCalledAsRm_True(t *testing.T) {
	if !args.CalledAsRm("/usr/local/bin/rm") {
		t.Errorf("expected CalledAsRm(\"/usr/local/bin/rm\") to return true")
	}
	if !args.CalledAsRm("rm") {
		t.Errorf("expected CalledAsRm(\"rm\") to return true")
	}
}

func TestParse_RestoreFlag(t *testing.T) {
	got, err := args.Parse([]string{"--restore"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Restore {
		t.Errorf("expected Restore=true, got false")
	}
	if len(got.Files) != 0 {
		t.Errorf("expected Files to be empty, got %v", got.Files)
	}
	if len(got.RmFlags) != 0 {
		t.Errorf("expected RmFlags to be empty, got %v", got.RmFlags)
	}
}

func TestParse_SkipTrashFlag(t *testing.T) {
	got, err := args.Parse([]string{"--be-brave-skip-trash", "-f", "file.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.SkipTrash {
		t.Errorf("expected SkipTrash=true, got false")
	}

	wantFiles := []string{"file.txt"}
	if len(got.Files) != len(wantFiles) {
		t.Fatalf("expected Files %v, got %v", wantFiles, got.Files)
	}
	if got.Files[0] != "file.txt" {
		t.Errorf("Files[0]: want %q, got %q", "file.txt", got.Files[0])
	}

	wantFlags := []string{"-f"}
	if len(got.RmFlags) != len(wantFlags) {
		t.Fatalf("expected RmFlags %v, got %v", wantFlags, got.RmFlags)
	}
	if got.RmFlags[0] != "-f" {
		t.Errorf("RmFlags[0]: want %q, got %q", "-f", got.RmFlags[0])
	}

	for _, flag := range got.RmFlags {
		if flag == "--be-brave-skip-trash" {
			t.Errorf("--be-brave-skip-trash must not appear in RmFlags")
		}
	}
}

func TestParse_FilesAndFlags(t *testing.T) {
	got, err := args.Parse([]string{"-rf", "foo", "bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.SkipTrash {
		t.Errorf("expected SkipTrash=false, got true")
	}
	if got.Restore {
		t.Errorf("expected Restore=false, got true")
	}

	wantFiles := []string{"foo", "bar"}
	if len(got.Files) != len(wantFiles) {
		t.Fatalf("expected Files %v, got %v", wantFiles, got.Files)
	}
	for i, f := range wantFiles {
		if got.Files[i] != f {
			t.Errorf("Files[%d]: want %q, got %q", i, f, got.Files[i])
		}
	}

	wantFlags := []string{"-rf"}
	if len(got.RmFlags) != len(wantFlags) {
		t.Fatalf("expected RmFlags %v, got %v", wantFlags, got.RmFlags)
	}
	if got.RmFlags[0] != wantFlags[0] {
		t.Errorf("RmFlags[0]: want %q, got %q", wantFlags[0], got.RmFlags[0])
	}
}
