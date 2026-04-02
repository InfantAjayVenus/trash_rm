// Package app is the testable entry point for trash-rm.
// main.go detects the environment (OS, TTY, invocation name) and calls Run
// with all dependencies injected, so the full flow is unit-testable.
package app

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/InfantAjayVenus/trash-rm/internal/args"
	"github.com/InfantAjayVenus/trash-rm/internal/log"
	"github.com/InfantAjayVenus/trash-rm/internal/restore"
	"github.com/InfantAjayVenus/trash-rm/internal/trash"
)

// Commander constructs an exec.Cmd. Matches the pattern used in trash and restore packages.
type Commander func(name string, a ...string) *exec.Cmd

// PromptFunc asks the user a yes/no question and returns the raw answer.
type PromptFunc func(msg string) (string, error)

// Config holds all dependencies for a single trash-rm invocation.
type Config struct {
	Parsed    args.ParsedArgs
	Backend   trash.Trasher
	Commander Commander
	RmPath    string
	LogPath   string
	IsTTY     bool
	PromptFn  PromptFunc
	SelectFn  restore.SelectFunc
	CWD       string
	Now       func() time.Time
	Out       io.Writer
}

// Run executes the full trash-rm flow and returns an exit code.
func Run(cfg Config) int {
	if cfg.Parsed.Restore {
		return runRestore(cfg)
	}
	if cfg.Parsed.SkipTrash {
		return runSkipTrash(cfg)
	}
	return runNormal(cfg)
}

func runRestore(cfg Config) int {
	if err := restore.Run(cfg.LogPath, cfg.SelectFn, restoreCommander(cfg.Commander), cfg.Out); err != nil {
		fmt.Fprintln(cfg.Out, err)
		return 1
	}
	return 0
}

func runSkipTrash(cfg Config) int {
	return execRm(cfg)
}

func runNormal(cfg Config) int {
	if err := cfg.Backend.CheckDependency(); err != nil {
		fmt.Fprintln(cfg.Out, err)
		return 1
	}

	if err := cfg.Backend.Trash(cfg.Parsed.Files); err != nil {
		return handleTrashFailure(cfg, err)
	}

	entry := log.LogEntry{
		Timestamp: cfg.Now().UTC().Format(time.RFC3339),
		Command:   buildCommand(cfg.Parsed),
		CWD:       cfg.CWD,
		Files:     cfg.Parsed.Files,
	}
	if err := log.Append(cfg.LogPath, entry); err != nil {
		fmt.Fprintln(cfg.Out, "fatal: could not write trash log:", err)
		return 1
	}

	return execRm(cfg)
}

func handleTrashFailure(cfg Config, _ error) int {
	if !cfg.IsTTY {
		fmt.Fprintln(cfg.Out, "Unable to trash before delete. Aborting (non-interactive context).")
		return 1
	}
	answer, err := cfg.PromptFn("Unable to trash before delete. Proceed without trash protection? [y/N]")
	if err != nil {
		return 1
	}
	if answer != "y" && answer != "Y" {
		return 1
	}
	return execRm(cfg)
}

func execRm(cfg Config) int {
	allArgs := append(cfg.Parsed.RmFlags, cfg.Parsed.Files...)
	cmd := cfg.Commander(cfg.RmPath, allArgs...)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

func buildCommand(parsed args.ParsedArgs) string {
	parts := append([]string{"trash-rm"}, parsed.RmFlags...)
	parts = append(parts, parsed.Files...)
	return strings.Join(parts, " ")
}

// restoreCommander adapts app.Commander to restore.Commander (same underlying type).
func restoreCommander(c Commander) restore.Commander {
	return restore.Commander(c)
}
