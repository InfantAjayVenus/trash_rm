package trash

import (
	"errors"
	"fmt"
	"os/exec"
)

// Commander is a function that constructs an exec.Cmd, used as a seam
// so tests can inject a recording fake instead of shelling out to real processes.
type Commander func(name string, args ...string) *exec.Cmd

// Trasher moves files to the system trash.
type Trasher interface {
	Trash(files []string) error
	CheckDependency() error
}

// LinuxBackend implements Trasher using the trash-cli tool.
type LinuxBackend struct {
	Commander Commander
}

// MacBackend implements Trasher using osascript/Finder on macOS.
type MacBackend struct {
	Commander Commander
}

// Trash moves each file to the Linux trash by calling `trash <file>`.
func (b LinuxBackend) Trash(files []string) error {
	for _, f := range files {
		cmd := b.Commander("trash", f)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// CheckDependency verifies that the trash-cli tool is available.
// Returns an error with installation instructions if trash is not found.
func (b LinuxBackend) CheckDependency() error {
	cmd := b.Commander("trash", "--version")
	if err := cmd.Run(); err != nil {
		return errors.New("trash not found. Install trash-cli: `sudo apt install trash-cli` (Debian/Ubuntu) or `sudo dnf install trash-cli` (Fedora)")
	}
	return nil
}

// CheckDependency verifies that osascript is available (it is bundled with macOS).
func (b MacBackend) CheckDependency() error {
	cmd := b.Commander("osascript", "-e", "")
	if err := cmd.Run(); err != nil {
		return errors.New("osascript not found. Requires osascript (bundled with macOS) — should be available by default")
	}
	return nil
}

// Trash moves each file to the macOS Trash using osascript/Finder.
func (b MacBackend) Trash(files []string) error {
	for _, f := range files {
		script := fmt.Sprintf(`tell application "Finder" to delete POSIX file %q`, f)
		cmd := b.Commander("osascript", "-e", script)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
