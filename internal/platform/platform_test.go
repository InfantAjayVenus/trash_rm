package platform_test

import (
	"strings"
	"testing"

	"github.com/markandeyan/trash-rm/internal/platform"
)

func TestLogPath_Linux(t *testing.T) {
	got, err := platform.LogPathFor("linux", "/home/alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/home/alice/.local/share/trash-rm/history.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLogPath_Darwin(t *testing.T) {
	got, err := platform.LogPathFor("darwin", "/Users/alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/Users/alice/Library/Application Support/trash-rm/history.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLogPath_UnsupportedOS(t *testing.T) {
	got, err := platform.LogPathFor("windows", "/home/alice")
	if err == nil {
		t.Fatal("expected error for unsupported platform, got nil")
	}
	if got != "" {
		t.Errorf("expected empty path on error, got %q", got)
	}
	wantSubstring := "unsupported platform"
	if !strings.Contains(err.Error(), wantSubstring) {
		t.Errorf("error message %q does not contain %q", err.Error(), wantSubstring)
	}
}
