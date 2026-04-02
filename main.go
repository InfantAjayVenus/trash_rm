package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/InfantAjayVenus/trash-rm/internal/app"
	"github.com/InfantAjayVenus/trash-rm/internal/args"
	"github.com/InfantAjayVenus/trash-rm/internal/platform"
	"github.com/InfantAjayVenus/trash-rm/internal/restore"
	"github.com/InfantAjayVenus/trash-rm/internal/rm"
	"github.com/InfantAjayVenus/trash-rm/internal/trash"
)

func main() {
	os.Exit(run())
}

func run() int {
	parsed, err := args.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine home directory:", err)
		return 1
	}

	logPath, err := platform.LogPathFor(runtime.GOOS, home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	backend := selectBackend(runtime.GOOS)

	rmPath, err := rm.RealRmPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine working directory:", err)
		return 1
	}

	isTTY := term.IsTerminal(int(os.Stdin.Fd()))

	cfg := app.Config{
		Parsed:    parsed,
		Backend:   backend,
		Commander: exec.Command,
		RmPath:    rmPath,
		LogPath:   logPath,
		IsTTY:     isTTY,
		PromptFn:  stdinPrompt,
		SelectFn:  restore.BubbleteaSelectFunc(os.Stdin, os.Stdout),
		CWD:       cwd,
		Now:       time.Now,
		Out:       os.Stdout,
	}

	return app.Run(cfg)
}

func selectBackend(goos string) trash.Trasher {
	switch goos {
	case "darwin":
		return trash.MacBackend{Commander: exec.Command}
	default:
		return trash.LinuxBackend{Commander: exec.Command}
	}
}

func stdinPrompt(msg string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s ", msg)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
