package rm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/InfantAjayVenus/trash-rm/internal/rm"
)

func TestRealRmPath_NoneExist(t *testing.T) {
	_, err := rm.RealRmPathFrom([]string{"/tmp/no-rm-here-abc123", "/tmp/also-no-rm-xyz987"})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "real rm not found") {
		t.Errorf("error message %q does not contain %q", err.Error(), "real rm not found")
	}
}

func TestRealRmPath_ReturnsFirstExisting(t *testing.T) {
	dir := t.TempDir()
	fakeRm := filepath.Join(dir, "rm")

	if err := os.WriteFile(fakeRm, []byte("#!/bin/sh"), 0o755); err != nil {
		t.Fatalf("setup: create fake rm: %v", err)
	}

	got, err := rm.RealRmPathFrom([]string{filepath.Join(dir, "absent-rm"), fakeRm})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fakeRm {
		t.Errorf("got %q, want %q", got, fakeRm)
	}
}
