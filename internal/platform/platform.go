package platform

import (
	"fmt"
	"path/filepath"
)

// LogPathFor returns the sidecar log file path for the given OS and home directory.
// Supported values for goos: "linux", "darwin".
func LogPathFor(goos, home string) (string, error) {
	switch goos {
	case "linux":
		return filepath.Join(home, ".local", "share", "trash-rm", "history.json"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "trash-rm", "history.json"), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", goos)
	}
}
