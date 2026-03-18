package rm

import (
	"errors"
	"os"
)

// RealRmPathFrom returns the first path in candidates that exists on the filesystem.
// It is used to locate the real rm binary at a known absolute path,
// avoiding exec.LookPath which would recurse through the trash-rm symlink.
func RealRmPathFrom(candidates []string) (string, error) {
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", errors.New("real rm not found at /bin/rm or /usr/bin/rm")
}

// RealRmPath returns the path to the real rm binary using the standard candidate list.
func RealRmPath() (string, error) {
	return RealRmPathFrom([]string{"/bin/rm", "/usr/bin/rm"})
}
