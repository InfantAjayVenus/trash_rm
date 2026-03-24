# trash-rm

`trash-rm` is a Go CLI that wraps `rm` with trash-first behavior.

Instead of deleting files immediately, it first sends them to the system trash, writes a sidecar history entry, and then calls the real system `rm`. It is intended to be installed as `trash-rm` plus an `rm -> trash-rm` symlink so normal `rm` usage is transparently protected.

## What It Does

- Moves file targets to the system trash before invoking the real `rm`
- Supports a bypass flag: `--be-brave-skip-trash`
- Records trash operations in a local history file
- Provides `--restore` to browse previous trash entries in an interactive TUI and restore them
- Resolves the real `rm` from `/bin/rm` or `/usr/bin/rm` to avoid symlink recursion

## Platform Support

### Linux

- Trash backend: `trash` from `trash-cli`
- Restore backend: `trash-list` and `trash-restore`
- History file: `~/.local/share/trash-rm/history.json`

### macOS

- Trash backend: `osascript` via Finder
- History file: `~/Library/Application Support/trash-rm/history.json`

Current implementation note: the restore path is written around `trash-list` and `trash-restore`, so trashing on macOS is implemented, but restore appears Linux-oriented in the current codebase.

## CLI

```bash
trash-rm [rm-flags] <files>...
trash-rm --be-brave-skip-trash [rm-flags] <files>...
trash-rm --restore

# after install.sh creates the symlink
rm [rm-flags] <files>...
```

## Behavior

### Normal delete flow

1. Parse CLI args.
2. Check the platform trash dependency.
3. Move each file to trash.
4. Append one NDJSON history entry.
5. Call the real `rm`.

If trashing fails:

- In an interactive terminal, the tool prompts:
  `Unable to trash before delete. Proceed without trash protection? [y/N]`
- In a non-interactive context, it aborts.
- If the user declines, it aborts.
- If the user confirms, it skips history logging and runs the real `rm`.

If writing the history log fails, it aborts and does not invoke `rm`.

### Skip-trash mode

`--be-brave-skip-trash` bypasses trash and logging entirely and forwards the rest of the arguments to the real `rm`.

### Restore mode

`--restore`:

- Reads the history log
- Cross-references entries against `trash-list`
- Shows remaining entries in a Bubble Tea TUI
- Restores the selected entry
- Rewrites the history file without the restored entry

If no usable history exists, it prints:

```text
No trash history found.
```

## History Format

History is stored as newline-delimited JSON:

```json
{"timestamp":"2026-03-17T14:05:00Z","command":"trash-rm -rf old-project","cwd":"/home/user/projects","files":["old-project"]}
```

Fields:

- `timestamp`: RFC 3339 UTC timestamp
- `command`: reconstructed command string
- `cwd`: working directory at deletion time
- `files`: file operands that were trashed

## Install

### Quick install

```bash
./install.sh
```

By default this:

- Builds `trash-rm`
- Installs it to `/usr/local/bin/trash-rm`
- Creates `/usr/local/bin/rm -> trash-rm`

You can override the install directory:

```bash
INSTALL_DIR="$HOME/.local/bin" ./install.sh
```

### Linux dependency

Install `trash-cli` before use:

```bash
sudo apt install trash-cli
# or
sudo dnf install trash-cli
```

The installer warns if the backend dependency is missing.

## Development

Repo layout:

- [`main.go`](/home/ajay_v/DevSpace/markandeyan/trash_rm/main.go): process setup and dependency wiring
- [`internal/app/app.go`](/home/ajay_v/DevSpace/markandeyan/trash_rm/internal/app/app.go): core execution flow
- [`internal/args/args.go`](/home/ajay_v/DevSpace/markandeyan/trash_rm/internal/args/args.go): CLI parsing
- [`internal/trash/trash.go`](/home/ajay_v/DevSpace/markandeyan/trash_rm/internal/trash/trash.go): Linux/macOS trash backends
- [`internal/restore/restore.go`](/home/ajay_v/DevSpace/markandeyan/trash_rm/internal/restore/restore.go): restore orchestration
- [`internal/restore/tui.go`](/home/ajay_v/DevSpace/markandeyan/trash_rm/internal/restore/tui.go): Bubble Tea selector UI
- [`internal/log/log.go`](/home/ajay_v/DevSpace/markandeyan/trash_rm/internal/log/log.go): NDJSON history persistence
- [`internal/rm/rm.go`](/home/ajay_v/DevSpace/markandeyan/trash_rm/internal/rm/rm.go): real `rm` lookup
- [`internal/platform/platform.go`](/home/ajay_v/DevSpace/markandeyan/trash_rm/internal/platform/platform.go): OS-specific history paths

## Current Caveats

- The Go module path in [`go.mod`](/home/ajay_v/DevSpace/markandeyan/trash_rm/go.mod) is `github.com/InfantAjayVenus/trash-rm`, but the source imports use `github.com/markandeyan/trash-rm`. The repo will not build or test until those match.
- `args.Parse` returns `missing file operand` itself instead of delegating that exact case to native `rm`.
- The parser treats any non-flag token as a file operand and any `-...` token as an `rm` flag; it is intentionally simple.
- The restore implementation is currently coupled to `trash-cli` commands, which limits practical restore support on macOS.

## Status

The repo includes unit tests across the internal packages, but local `go test ./...` currently fails because of the module/import-path mismatch described above.
