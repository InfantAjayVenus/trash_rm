# Spec: trash-rm CLI

## Overview

A Go CLI that wraps the native `rm` command with system trash integration: files are moved to the
system trash (via `trash-cli` on Linux, `osascript`/Finder on macOS) before deletion rather than
deleted immediately. A sidecar log records the invocation context so users can later review what
they trashed and from where, and restore an entire invocation's worth of files back to their
original locations. The binary is installed as `rm` (via symlink) so all `rm` invocations are
transparently protected without user or AI-agent awareness.

---

## Acceptance Criteria

### Core: rm passthrough

- [x] The binary is named `trash-rm`; the installer creates a symlink `rm → trash-rm` on `PATH`
      so that invoking `rm` transparently invokes `trash-rm`
- [x] When called as `rm` (via symlink), the binary detects this and resolves the real `rm` at a
      known absolute path (`/bin/rm` or `/usr/bin/rm`) to avoid infinite recursion — it does NOT
      use `exec.LookPath("rm")` in this case
<!-- -------- bee-comment -------- -->
> **@developer**: the cli command should be aliased with rm when installing, so that users(human / AI agents) don't have to be conscious about which tool they're using.
> **@bee**: Added two ACs: (1) the installer creates an `rm → trash-rm` symlink on PATH, and (2) when called as `rm`, the binary resolves the real `rm` via absolute path (not `LookPath`) to prevent infinite recursion. This is a subtle but critical detail — a process cannot safely use `LookPath("rm")` to find itself.
> - [x] mark as resolved
<!-- -------- /bee-comment -------- -->
- [x] Supports the same flags as the system `rm` (`-r`, `-f`, `-i`, `-v`, `-d`, and any other flags
      accepted by the system `rm`), forwarding them without modification
- [x] If no files are specified, exits with the same error code and message that native `rm` would
      produce
- [x] Exit code of `trash-rm` mirrors the exit code of the underlying `rm` invocation

### Core: trash-before-delete

- [x] Before invoking `rm`, each target file is sent to the system trash by calling
      the platform trash command for each file argument
- [x] If the trash backend is not found on `PATH`, `trash-rm` exits with a non-zero code and a
      human-readable error that includes platform-specific installation instructions:
      - Linux: "Install trash-cli: `sudo apt install trash-cli` (Debian/Ubuntu) or `sudo dnf install trash-cli` (Fedora)"
      - macOS: "Requires osascript (bundled with macOS) — should be available by default"
- [x] The installer script checks for the trash backend dependency at install time and prints a
      warning if it is missing, with the same installation instructions as above
<!-- -------- bee-comment -------- -->
> **@developer**: we can display a help message that shows how to install the dependency. If possible, make the dependency check part of installation.
> **@bee**: Updated the AC to include platform-specific install instructions in the error message (apt/dnf for Linux, osascript note for macOS). Added a separate AC requiring the installer to perform the same check at install time. This gives two layers: fail fast at runtime AND warn early at install time.
> - [x] mark as resolved
<!-- -------- /bee-comment -------- -->
- [x] If the trash operation fails for any file, `trash-rm` treats the failure as atomic: it aborts
      the entire trash step (no files are partially trashed), prints to stderr:
      "Unable to trash before delete. Proceed without trash protection? [y/N]", and waits for input
<!-- -------- bee-comment -------- -->
> **@developer**: the operation should be atomic, don't tell the user about any specific file, just tell unable to trash before delete, do you want to proceed?
> **@bee**: Reworked the AC to be atomic: on any failure, abort the whole trash step (undo any already-trashed files if possible), use the generic message "Unable to trash before delete. Proceed without trash protection? [y/N]" without naming individual files. Keeps the UX clean and treats the invocation as an indivisible unit.
> - [x] mark as resolved
<!-- -------- /bee-comment -------- -->
- [x] If the user confirms the prompt (enters `y` or `Y`), `trash-rm` skips the trash step and
      delegates all arguments directly to native `rm`; no sidecar log entry is written
- [x] If the user declines the prompt (enters anything other than `y`/`Y`, or presses Enter),
      `trash-rm` aborts with a non-zero exit code and does not call `rm`
- [x] If stdin is not a TTY (non-interactive context), the prompt is skipped and `trash-rm` aborts
      with a non-zero exit code rather than proceeding unprotected
- [x] A sidecar log entry is written to `~/.local/share/trash-rm/history.json` (Linux) or
      `~/Library/Application Support/trash-rm/history.json` (macOS) after a successful trash step,
      recording: the full command as typed, the working directory (CWD), the timestamp (RFC 3339),
      and the list of files that were trashed
- [x] If the sidecar log file or its parent directory does not exist, `trash-rm` creates them before
      writing
- [x] If writing the sidecar log fails, `trash-rm` aborts with a non-zero exit code and does NOT
      proceed with the `rm` invocation — files are in trash but the rm is blocked until log is
      confirmed written, preventing an unrecoverable state where files are trashed but unlisted
<!-- -------- bee-comment -------- -->
> **@developer**: if log fails how will the user restore?
> **@bee**: Changed log failure from non-fatal (warn and continue) to **fatal** (abort). Rationale: if the log write fails, the files have been moved to trash but won't appear in `--restore`, leaving the user with no way to find them. The safer contract is: trash succeeded → log succeeded → rm proceeds. If log fails, we block rm so the user knows something went wrong. They can retry or use `--be-brave-skip-trash` consciously.
> - [x] mark as resolved
<!-- -------- /bee-comment -------- -->
- [x] Directories are trashed and deleted only when `-r` (or `-R` or `--recursive`) is passed, the
      same as native `rm`

### Flag: --be-brave-skip-trash

- [x] When `--be-brave-skip-trash` is passed, the trash step is skipped entirely
- [x] With `--be-brave-skip-trash`, no sidecar log entry is written
- [x] With `--be-brave-skip-trash`, all remaining arguments are forwarded directly to native `rm`
      and its exit code is returned
- [x] `--be-brave-skip-trash` is consumed by `trash-rm` and is not forwarded to native `rm`

### Flag: --restore

- [x] When `--restore` is passed, `trash-rm` cross-references the sidecar log against `trash-list`
      output and displays only entries whose files still exist in the trash (not yet garbage
      collected), in reverse-chronological order (most recent first)
<!-- -------- bee-comment -------- -->
> **@developer**: the list should validate with the trash to display only the entries that aren't [garbage collected yet]
> **@bee**: Added the cross-reference step: before displaying, run `trash-list` and filter the sidecar log to only show entries where at least one file still appears in `trash-list`. Entries whose files have all been GC'd are silently omitted from the display. This keeps the list honest.
> - [x] mark as resolved
<!-- -------- /bee-comment -------- -->
- [x] The list is presented as an interactive TUI (using a Go terminal UI library) where the user
      can navigate entries with arrow keys (`↑`/`↓`) and vim motions (`j`/`k`); each entry shows:
      `[<timestamp>] <command>  (cwd: <working-directory>)`; pressing Enter selects; pressing `q`
      or Escape exits without restoring
<!-- -------- bee-comment -------- -->
> **@developer**: would be a better UX if the user can scroll through the list using arrow keys & vim motions.
> **@bee**: Replaced the numbered-list-with-typed-number approach with an interactive TUI using a Go terminal UI library (e.g., `charmbracelet/bubbletea` + `charmbracelet/bubbles/list`). Arrow keys + j/k navigation, Enter to select, q/Escape to quit. This also makes invalid input structurally impossible — the "invalid number" AC below has been removed.
> - [x] mark as resolved
<!-- -------- /bee-comment -------- -->
- [x] If the user selects an entry, `trash-rm` calls `trash-restore` for each file recorded in that
      sidecar log entry, restoring them to their original locations
- [x] If `trash-restore` succeeds for all files in the chosen entry, `trash-rm` exits with code 0
- [x] If `trash-restore` fails for any file, `trash-rm` checks whether the file is absent from
      `trash-list`; if so, it reports: "Cannot restore <file>: it has been permanently deleted from
      trash"; for other failures it reports the raw error; files that did succeed are left restored;
      exits with a non-zero code
<!-- -------- bee-comment -------- -->
> **@developer**: if the restoration failed because it was cleared from trash, explicitly tell the user so.
> **@bee**: Added logic to distinguish "file no longer in trash (GC'd)" from other errors. If `trash-list` doesn't show the file, the message is explicit: "Cannot restore <file>: it has been permanently deleted from trash." This gives the user a clear answer rather than a generic failure.
> - [x] mark as resolved
<!-- -------- /bee-comment -------- -->
- [x] If the user presses `q` or Escape in the TUI, `trash-rm` exits with code 0 without restoring
      anything
- [x] If the sidecar log is empty or does not exist, `--restore` prints "No trash history found."
      and exits with code 0
- [x] If the sidecar log cannot be parsed, `--restore` exits with a non-zero code and a
      human-readable error message
- [x] After a successful restore of an entry, `trash-rm` removes that entry from the sidecar log
      (rewrites the log without the restored entry)
<!-- -------- bee-comment -------- -->
> **@developer**: we do have to remove the restored entry from the log, so log write will be done.
> **@bee**: Updated the AC: --restore DOES write to the log — after a successful restore it removes the restored entry (rewrites the log file without that entry). This keeps the log in sync with reality. Partial restores (some files failed) leave the entry intact.
> - [x] mark as resolved
<!-- -------- /bee-comment -------- -->

### Out of Scope

- Garbage-collection or expiry of sidecar log entries
- Integration with any trash backend other than `trash-cli` (Linux) and `osascript` (macOS)
- Windows support
- Pruning sidecar log entries when `trash-list` no longer shows a file (sync between log and trash)
- Restoring individual files within an entry (restore is all-or-nothing per invocation entry)
<!-- -------- bee-comment -------- -->
> **@developer**: I've added this to scope above. any rationale to not include it? (re: removing restored entries from log)
> **@bee**: Removed "Removing a sidecar log entry after a successful restore" from Out of Scope — it is now in scope per the updated --restore AC above. The original rationale ("append-only in this phase") was a conservative default to keep scope small, but there's no technical reason to exclude it, and without it the restore list grows stale. Good call.
> - [x] mark as resolved
<!-- -------- /bee-comment -------- -->
<!-- -------- bee-comment -------- -->
> **@developer**: I do need support for macOS as it is a Unix based system.
> **@bee**: Added macOS support. On macOS, `trash-cli` is unavailable; instead, `trash-rm` uses `osascript -e 'tell application "Finder" to delete POSIX file "<abs-path>"'` to move files to the macOS Trash. The sidecar log path changes to `~/Library/Application Support/trash-rm/history.json`. The `--restore` flow on macOS would use Finder/AppleScript to restore, or document a manual step (full restore automation on macOS is complex — this is flagged for further scoping if needed). "Windows support" remains out of scope.
> - [x] mark as resolved
<!-- -------- /bee-comment -------- -->

---

## Platform Support

| Platform | Trash command | Restore command | Sidecar log path |
|----------|---------------|-----------------|------------------|
| Linux    | `trash <file>` (trash-cli) | `trash-restore` | `~/.local/share/trash-rm/history.json` |
| macOS    | `osascript -e 'tell application "Finder" to delete POSIX file "<abs-path>"'` | TBD (manual via Finder or scripted) | `~/Library/Application Support/trash-rm/history.json` |

> **Note:** macOS `--restore` scope: moving files back from macOS Trash via `osascript` is possible
> but requires knowing the Finder trash item path. This may require additional scoping before
> implementation — flagged for discussion during TDD planning.

---

## Sidecar Log Format

File: `~/.local/share/trash-rm/history.json` (Linux) / `~/Library/Application Support/trash-rm/history.json` (macOS)

Each invocation appends one JSON object per line (newline-delimited JSON / NDJSON):

```json
{"timestamp":"2026-03-17T14:05:00Z","command":"trash-rm -rf old-project/","cwd":"/home/user/projects","files":["old-project/"]}
```

Fields:

| Field       | Type            | Description                                      |
|-------------|-----------------|--------------------------------------------------|
| `timestamp` | string (RFC3339)| When the deletion was made                       |
| `command`   | string          | Full command as typed (e.g. `trash-rm -rf foo/`) |
| `cwd`       | string          | Absolute working directory at time of invocation |
| `files`     | []string        | File/directory arguments that were trashed       |

---

## CLI Interface

```
rm [rm-flags] <files>...                              # (via symlink) trash then rm
trash-rm [rm-flags] <files>...                        # trash then rm
trash-rm --be-brave-skip-trash [rm-flags] <files>...  # rm only, no trash
trash-rm --restore                                    # interactive TUI restore from history
```

---

## Technical Context

- Language: Go (greenfield project)
- Trash backend (Linux): `trash` and `trash-restore` commands from `trash-cli`
- Trash backend (macOS): `osascript` (bundled with macOS)
- TUI library: `charmbracelet/bubbletea` + `charmbracelet/bubbles/list` (for `--restore` selector)
- Sidecar log: NDJSON, platform-specific path (see Platform Support table)
- The real `rm` binary: resolved at known absolute paths (`/bin/rm`, `/usr/bin/rm`) — NOT via `LookPath` to prevent symlink recursion
- TTY detection: needed for non-interactive abort on trash failure
- Risk level: LOW

[x] Reviewed
