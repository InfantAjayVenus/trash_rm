package args

import (
	"errors"
	"path/filepath"
	"strings"
)

// ParsedArgs holds the result of parsing the CLI argv.
type ParsedArgs struct {
	Files     []string
	RmFlags   []string
	SkipTrash bool
	Restore   bool
}

// CalledAsRm reports whether the binary was invoked as "rm" (e.g. via symlink).
func CalledAsRm(argv0 string) bool {
	return filepath.Base(argv0) == "rm"
}

// Parse separates trash-rm-specific flags from rm flags and file operands.
// argv should be os.Args[1:] (everything after the binary name).
func Parse(argv []string) (ParsedArgs, error) {
	var result ParsedArgs

	for _, arg := range argv {
		switch arg {
		case "--be-brave-skip-trash":
			result.SkipTrash = true
		case "--restore":
			result.Restore = true
		default:
			if strings.HasPrefix(arg, "-") {
				result.RmFlags = append(result.RmFlags, arg)
			} else {
				result.Files = append(result.Files, arg)
			}
		}
	}

	if !result.Restore && len(result.Files) == 0 {
		return ParsedArgs{}, errors.New("missing file operand")
	}

	return result, nil
}
