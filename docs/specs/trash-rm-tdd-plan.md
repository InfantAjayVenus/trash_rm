# TDD Plan: trash-rm CLI — Full Feature

## Execution Instructions
Read this plan. Work on every slice in order.
Mark each checkbox done as you complete it ([ ] → [x]).
Each slice is independently testable — complete all steps in a slice before moving to the next. After the verifier passes a slice, create a git commit before starting the next slice.
<!-- -------- bee-comment -------- -->
> **@developer**: commit when a slice is successfully completed.
> **@bee**: Added "After the verifier passes a slice, create a git commit before starting the next slice." to the Execution Instructions. This gives you a clean commit-per-slice history that maps directly to the TDD plan, making it easy to bisect or revert individual slices.
> - [ ] mark as resolved
<!-- -------- /bee-comment -------- -->
If stuck after 3 attempts on a single step, mark it ⚠️ and move to the next independent step within the slice.

---

## Context
- **Source**: `docs/specs/trash-rm.md`
- **Slices**: 6 vertical slices — each maps to a named package in `internal/`
- **Risk level**: LOW — happy path + 1-2 edge cases per slice
- **Acceptance Criteria**: See spec sections: Core rm passthrough, Core trash-before-delete, Flag --be-brave-skip-trash, Flag --restore

---

## Codebase Analysis

### File Structure
This is a greenfield Go project. The layout to build toward:

```
main.go
internal/
  args/       # flag parsing + symlink detection
  trash/      # Trasher interface + LinuxBackend + MacBackend
  log/        # sidecar log append / read / rewrite
  rm/         # exec real rm at known absolute path
  restore/    # --restore flow + bubbletea TUI
  platform/   # OS detection, log path resolution
```

### Test Infrastructure
- Framework: Go's standard `testing` package
- Run command: `go test ./...`
- Test files live alongside the package they test (co-located): `internal/args/args_test.go`, etc.
- No test helpers exist yet — create as needed, keep them in `internal/<pkg>/testhelpers_test.go` if they grow large
- External-command calls (`trash`, `rm`, `osascript`) will be faked via a small `Commander` function type (a `func(name string, args ...string) *exec.Cmd` replacement) — this avoids hitting the real filesystem in unit tests

---

## Slice 1 — `internal/platform`: OS detection and log path resolution

### Acceptance Criteria Covered
- Sidecar log at `~/.local/share/trash-rm/history.json` (Linux)
- Sidecar log at `~/Library/Application Support/trash-rm/history.json` (macOS)

---

### Behavior 1.1: LogPath returns the Linux path when GOOS is linux

**Given** the current OS is Linux and `$HOME` is `/home/alice`
**When** `platform.LogPath()` is called
**Then** it returns `/home/alice/.local/share/trash-rm/history.json`

- [x] **RED**: Write failing test
  - Location: `internal/platform/platform_test.go`
  - Test name: `TestLogPath_Linux`
  - Strategy: the function should accept an injectable `os` (home dir + OS string) so tests don't depend on the real environment; simplest approach — export `LogPathFor(goos, home string) string`
  - Input: `goos="linux"`, `home="/home/alice"`
  - Expected: `"/home/alice/.local/share/trash-rm/history.json"`

- [x] **RUN**: Confirm test FAILS (package doesn't exist yet)

- [x] **GREEN**: Create `internal/platform/platform.go`, implement `LogPathFor(goos, home string) string` with a switch on `goos`

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: None needed yet

- [ ] **COMMIT**: `"feat: platform.LogPathFor returns linux log path"`

---

### Behavior 1.2: LogPath returns the macOS path when GOOS is darwin

**Given** the current OS is macOS and `$HOME` is `/Users/alice`
**When** `platform.LogPathFor("darwin", "/Users/alice")` is called
**Then** it returns `/Users/alice/Library/Application Support/trash-rm/history.json`

- [x] **RED**: Write failing test
  - Location: `internal/platform/platform_test.go`
  - Test name: `TestLogPath_Darwin`
  - Input: `goos="darwin"`, `home="/Users/alice"`
  - Expected: `"/Users/alice/Library/Application Support/trash-rm/history.json"`

- [x] **RUN**: Confirm test FAILS

- [x] **GREEN**: Add `darwin` case to the switch in `LogPathFor`

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: platform.LogPathFor returns darwin log path"`

---

### Edge Case 1.3: LogPath returns error for unsupported OS

**Given** an unrecognised GOOS string
**When** `LogPathFor("windows", "/home/alice")` is called
**Then** it returns `("", error)` with a message containing "unsupported platform"

- [x] **RED**: Write failing test
  - Test name: `TestLogPath_UnsupportedOS`
  - Adjust signature to `LogPathFor(goos, home string) (string, error)`

- [x] **GREEN**: Add default case returning an error; update callers in tests

- [x] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: platform.LogPathFor errors on unsupported OS"`

---

## Slice 2 — `internal/args`: Flag parsing and symlink detection

### Acceptance Criteria Covered
- Supports same flags as system `rm`, forwarding them without modification
- Detects when called as `rm` (via symlink)
- `--be-brave-skip-trash` consumed and not forwarded
- `--restore` consumed and not forwarded
- If no files specified, returns an error

---

### Behavior 2.1: Parse returns the file list and forwarded rm flags

**Given** `os.Args` equivalent is `["trash-rm", "-rf", "foo", "bar"]`
**When** `args.Parse([]string{"-rf", "foo", "bar"})` is called
**Then** it returns `ParsedArgs{Files: ["foo","bar"], RmFlags: ["-rf"], SkipTrash: false, Restore: false}`

- [x] **RED**: Write failing test
  - Location: `internal/args/args_test.go`
  - Test name: `TestParse_FilesAndFlags`
  - Define `ParsedArgs` struct in the test; it will drive the struct shape in production code

- [x] **RUN**: Confirm test FAILS

- [x] **GREEN**: Create `internal/args/args.go`, define `ParsedArgs`, implement `Parse(argv []string) (ParsedArgs, error)`
  - Walk `argv`, separate known trash-rm flags from everything else (flags + files pass through to `RmFlags` / `Files`)

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: args.Parse separates files from rm flags"`

---

### Behavior 2.2: Parse sets SkipTrash when --be-brave-skip-trash is present

**Given** argv `["--be-brave-skip-trash", "-f", "file.txt"]`
**When** `args.Parse(argv)` is called
**Then** `SkipTrash` is `true`, `"--be-brave-skip-trash"` is absent from `RmFlags`, file and `-f` are still present

- [x] **RED**: Write failing test
  - Test name: `TestParse_SkipTrashFlag`

- [x] **RUN**: Confirm test FAILS (or passes — behavior was already implemented in step 2.1 GREEN)

- [x] **GREEN**: Detect `--be-brave-skip-trash` in the walk loop; set `SkipTrash=true`; do not append to `RmFlags`

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: args.Parse consumes --be-brave-skip-trash"`

---

### Behavior 2.3: Parse sets Restore when --restore is present

**Given** argv `["--restore"]`
**When** `args.Parse(argv)` is called
**Then** `Restore` is `true` and `RmFlags` and `Files` are empty

- [x] **RED**: Write failing test
  - Test name: `TestParse_RestoreFlag`

- [x] **RUN**: Confirm test FAILS (or passes — behavior was already implemented in step 2.1 GREEN)

- [x] **GREEN**: Detect `--restore`; set `Restore=true`

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: args.Parse consumes --restore"`

---

### Behavior 2.4: DetectSymlink returns true when called as rm

**Given** `os.Args[0]` equivalent is `"/usr/local/bin/rm"` (basename is `rm`)
**When** `args.CalledAsRm(argv0 string) bool` is called
**Then** it returns `true`

- [x] **RED**: Write failing test
  - Test name: `TestCalledAsRm_True`
  - Input: `"/usr/local/bin/rm"` and `"rm"` (both)

- [x] **RUN**: Confirm test FAILS

- [x] **GREEN**: Implement `CalledAsRm(argv0 string) bool` — `filepath.Base(argv0) == "rm"`

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: args.CalledAsRm detects symlink invocation"`

---

### Edge Case 2.5: Parse returns error when no files are specified

**Given** argv `["-rf"]` (flags only, no files)
**When** `args.Parse(argv)` is called
**Then** it returns a non-nil error

- [x] **RED**: Write failing test
  - Test name: `TestParse_NoFiles_ReturnsError`

- [x] **GREEN**: After walking, if `Files` is empty (and `Restore` is false), return `errors.New("missing file operand")`

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: args.Parse errors when no files given"`

---

## Slice 3 — `internal/rm` + `internal/trash`: real-rm exec and trash backends

### Acceptance Criteria Covered
- Resolves real rm at `/bin/rm` or `/usr/bin/rm` (not LookPath)
- Trash backend: Linux = `trash <file>`, macOS = osascript command
- Exit code mirrors rm
- Trash backend not found → non-zero exit + human-readable instructions

---

### Behavior 3.1: RealRmPath returns /bin/rm when that file exists

**Given** `/bin/rm` is accessible (on Linux CI this is always true)
**When** `rm.RealRmPath()` is called
**Then** it returns `/bin/rm, nil`

- [x] **RED**: Write failing test
  - Location: `internal/rm/rm_test.go`
  - Test name: `TestRealRmPath_ReturnsFirstExisting`
  - Strategy: make the candidate list injectable — `RealRmPathFrom(candidates []string) (string, error)` — so tests can pass fake paths on a temp dir

- [x] **RUN**: Confirm test FAILS

- [x] **GREEN**: Create `internal/rm/rm.go`, implement `RealRmPathFrom(candidates []string) (string, error)` — iterate, return first that `os.Stat` says exists

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: Add `RealRmPath() (string, error)` that calls `RealRmPathFrom` with `["/bin/rm", "/usr/bin/rm"]`

- [ ] **COMMIT**: `"feat: rm.RealRmPathFrom finds rm at known absolute paths"`

---

### Behavior 3.2: RealRmPath errors when neither candidate exists

**Given** a temp directory with no `rm` binary
**When** `RealRmPathFrom([]string{"/tmp/no-rm-here"})` is called
**Then** it returns `("", error)` with message "real rm not found"

- [x] **RED**: Write failing test
  - Test name: `TestRealRmPath_NoneExist`

- [x] **GREEN**: Default case returns `("", errors.New("real rm not found at /bin/rm or /usr/bin/rm"))`

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: rm.RealRmPathFrom errors when rm is missing"`

---

### Behavior 3.3: LinuxBackend.Trash calls `trash <file>` for each file

**Given** a `LinuxBackend` with an injectable `Commander` (a `func(name string, args ...string) *exec.Cmd`)
**When** `backend.Trash([]string{"foo.txt"})` is called with a Commander that records calls
**Then** the Commander was called with `("trash", "foo.txt")`

- [x] **RED**: Write failing test
  - Location: `internal/trash/trash_test.go`
  - Test name: `TestLinuxBackend_Trash_CallsTrashCommand`
  - Define a small `recordingCommander` in the test file that captures `(name, args)` pairs and returns a no-op `exec.Cmd`

- [x] **RUN**: Confirm test FAILS

- [x] **GREEN**: Create `internal/trash/trash.go`; define `Trasher` interface with `Trash(files []string) error`; implement `LinuxBackend{Commander}` struct

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: trash.LinuxBackend calls trash-cli per file"`

---

### Behavior 3.4: MacBackend.Trash calls osascript with the Finder AppleScript

**Given** a `MacBackend` with an injectable `Commander`
**When** `backend.Trash([]string{"/Users/alice/foo.txt"})` is called
**Then** the Commander was called with `("osascript", "-e", "tell application \"Finder\" to delete POSIX file \"/Users/alice/foo.txt\"")`

- [x] **RED**: Write failing test
  - Test name: `TestMacBackend_Trash_CallsOsascript`

- [x] **GREEN**: Implement `MacBackend{Commander}` struct; build the AppleScript string with `fmt.Sprintf`

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: trash.MacBackend calls osascript for macOS trash"`

---

### Behavior 3.5: LinuxBackend.CheckDependency errors when `trash` is not on PATH

**Given** a Commander that returns `exec.Command("false")` (simulating missing binary)
**When** `backend.CheckDependency()` is called
**Then** it returns an error containing "Install trash-cli"

- [x] **RED**: Write failing test
  - Test name: `TestLinuxBackend_CheckDependency_MissingTrashCli`
  - Strategy: add `CheckDependency() error` to the `Trasher` interface (or as a separate `DependencyChecker` interface); inject a Commander that fails on `which trash` or use a simpler `os.LookPath`-like injection

- [x] **RUN**: Confirm test FAILS

- [x] **GREEN**: Implement `CheckDependency()` — look up the command; if not found, return an error with install instructions

- [x] **RUN**: Confirm test PASSES

- [x] **REFACTOR**: Confirm the MacBackend version returns "Requires osascript (bundled with macOS)" on the same path

- [ ] **COMMIT**: `"feat: trash backends report missing dependency with install instructions"`

---

## Slice 4 — `internal/log`: sidecar log append, read, and rewrite

### Acceptance Criteria Covered
- Append one NDJSON entry per invocation after successful trash
- Create file and parent dir if missing
- Log write failure is fatal (caller must abort)
- Read all entries for --restore
- Rewrite log without a restored entry after --restore

---

### Behavior 4.1: Append writes one NDJSON line to a new file

**Given** a temp directory path that does not yet exist
**When** `log.Append(path, entry)` is called with a valid `LogEntry`
**Then** the file is created (including parent dirs) and contains exactly one JSON line matching the entry fields

- [ ] **RED**: Write failing test
  - Location: `internal/log/log_test.go`
  - Test name: `TestAppend_CreatesFileAndWritesNDJSON`
  - Use `os.MkdirTemp` for the temp directory; construct `LogEntry{Timestamp, Command, CWD, Files}`
  - Assert `os.ReadFile` returns one JSON line; unmarshal and compare fields

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: Create `internal/log/log.go`; define `LogEntry` struct with json tags; implement `Append(path string, entry LogEntry) error` — `os.MkdirAll` for parent, open with `O_APPEND|O_CREATE|O_WRONLY`, marshal to JSON, write + newline

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: log.Append creates NDJSON sidecar log"`

---

### Behavior 4.2: Append adds a second line without overwriting the first

**Given** a log file already containing one entry
**When** `log.Append` is called a second time
**Then** the file contains two lines, each valid JSON

- [ ] **RED**: Write failing test
  - Test name: `TestAppend_AppendsWithoutOverwriting`

- [ ] **GREEN**: The `O_APPEND` flag already handles this — test should pass with current implementation; if not, fix the open flags

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"test: log.Append verified to append not overwrite"`

---

### Behavior 4.3: ReadAll returns all entries in order

**Given** a log file with three NDJSON lines
**When** `log.ReadAll(path string) ([]LogEntry, error)` is called
**Then** it returns a slice of 3 entries in file order (oldest first)

- [ ] **RED**: Write failing test
  - Test name: `TestReadAll_ReturnsAllEntriesInOrder`
  - Write 3 distinct entries via `Append`, then call `ReadAll` and check length + field values

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: Implement `ReadAll` — open file, `bufio.Scanner`, unmarshal each line, append to slice

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: log.ReadAll reads all NDJSON entries"`

---

### Behavior 4.4: ReadAll returns empty slice (not error) when file does not exist

**Given** a path to a non-existent file
**When** `log.ReadAll(path)` is called
**Then** it returns `([]LogEntry{}, nil)` — no error

- [ ] **RED**: Write failing test
  - Test name: `TestReadAll_EmptyWhenFileMissing`

- [ ] **GREEN**: Check `os.IsNotExist(err)` on open; return empty slice

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: log.ReadAll returns empty on missing file"`

---

### Behavior 4.5: Rewrite removes a specific entry and leaves the rest

**Given** a log file with entries A, B, C
**When** `log.Rewrite(path, []LogEntry{A, C})` is called (B omitted)
**Then** the file contains exactly entries A and C, in that order

- [ ] **RED**: Write failing test
  - Test name: `TestRewrite_RemovesTargetEntry`
  - Write A, B, C via `Append`; read them back; call `Rewrite` with `[A, C]`; `ReadAll` again and assert

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: Implement `Rewrite(path string, entries []LogEntry) error` — write to a temp file in same dir, then `os.Rename` (atomic on POSIX)

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: log.Rewrite atomically rewrites log without removed entry"`

---

### Edge Case 4.6: ReadAll returns error on malformed JSON line

**Given** a file where one line is not valid JSON
**When** `log.ReadAll(path)` is called
**Then** it returns a non-nil error with a message indicating parse failure

- [ ] **RED**: Write failing test
  - Test name: `TestReadAll_ErrorOnMalformedJSON`
  - Write `"not json\n"` directly to a temp file

- [ ] **GREEN**: Check `json.Unmarshal` error; return wrapped error

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: log.ReadAll surfaces JSON parse errors"`

---

## Slice 5 — `internal/restore`: --restore flow (log + trash-list cross-reference, entry removal)

### Acceptance Criteria Covered
- Cross-reference log against `trash-list` output; show only entries with surviving files
- After successful restore, remove entry from log (Rewrite)
- If `trash-restore` fails because file is gone from trash-list → specific error message
- Empty/missing log → "No trash history found."
- Malformed log → non-zero exit + human-readable error

This slice does NOT cover the bubbletea TUI selection — that is Slice 6.
Here we test the data layer: filtering, restoring, log rewriting.

---

### Behavior 5.1: FilterByTrashList returns only entries whose files appear in trash-list output

**Given** a `trash-list` output listing `foo.txt` but not `bar.txt`, and two log entries (one for `foo.txt`, one for `bar.txt`)
**When** `restore.FilterAlive(entries []log.LogEntry, trashListOutput string) []log.LogEntry` is called
**Then** it returns only the entry for `foo.txt`

- [ ] **RED**: Write failing test
  - Location: `internal/restore/restore_test.go`
  - Test name: `TestFilterAlive_ReturnsOnlyLiveEntries`
  - `trashListOutput` is a multi-line string as `trash-list` would produce

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: Create `internal/restore/restore.go`; implement `FilterAlive` — for each entry, check whether any `entry.Files` substring appears in `trashListOutput`

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: restore.FilterAlive cross-references log vs trash-list"`

---

### Behavior 5.2: RestoreEntry calls trash-restore for each file in the entry

**Given** a `LogEntry` with files `["foo.txt", "bar.txt"]` and a recording Commander
**When** `restore.RestoreEntry(entry, commander)` is called
**Then** the Commander was called twice: `("trash-restore", "foo.txt")` and `("trash-restore", "bar.txt")`

- [ ] **RED**: Write failing test
  - Test name: `TestRestoreEntry_CallsTrashRestorePerFile`

- [ ] **GREEN**: Implement `RestoreEntry(entry log.LogEntry, commander Commander) error` — iterate files, exec `trash-restore <file>`

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: restore.RestoreEntry calls trash-restore per file"`

---

### Behavior 5.3: RestoreEntry returns "permanently deleted from trash" error when file is absent from trash-list

**Given** a `trash-restore` Commander that fails, and `trash-list` output that does NOT contain the file
**When** `restore.RestoreEntry` is called
**Then** error message contains "permanently deleted from trash"

- [ ] **RED**: Write failing test
  - Test name: `TestRestoreEntry_PermanentlyDeletedError`
  - Inject two commanders: one for `trash-restore` (fails), one for `trash-list` (returns output without the file)

- [ ] **GREEN**: On `trash-restore` failure, run `trash-list` check; if file absent, return the specific message

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: restore.RestoreEntry distinguishes GC'd files from other errors"`

---

### Behavior 5.4: Run returns "No trash history found." when log is empty

**Given** an empty log path (file does not exist)
**When** `restore.Run(logPath, commander)` is called
**Then** it returns `(msg="No trash history found.", exitCode=0)` without error

- [ ] **RED**: Write failing test
  - Test name: `TestRun_EmptyLog`
  - `Run` returns `(message string, code int, err error)` or writes to an `io.Writer` — pick the shape that makes testing easy; prefer returning a string for the message so tests don't capture stdout

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: Implement `Run` skeleton; call `log.ReadAll`; if len==0, return the message

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: restore.Run reports empty history without error"`

---

### Behavior 5.5: Run removes the restored entry from the log after success

**Given** a log with two entries, and the user selects entry 0 (injected via a SelectFunc)
**When** `restore.Run` is called with a SelectFunc that always picks index 0 and a Commander that succeeds
**Then** the log file contains only the second entry afterward

- [ ] **RED**: Write failing test
  - Test name: `TestRun_RemovesEntryAfterRestore`
  - Add a `SelectFunc func(entries []log.LogEntry) (int, error)` parameter so the bubbletea TUI is pluggable — in tests pass a trivial picker; in production pass the TUI picker

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: After `RestoreEntry` succeeds, call `log.Rewrite` with the entry removed

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: restore.Run rewrites log after successful restore"`

---

## Slice 6 — `main.go`: wiring + atomic trash-before-delete flow

### Acceptance Criteria Covered
- Full happy path: parse → trash → log → rm
- Atomic trash failure prompt (TTY vs non-TTY)
- `--be-brave-skip-trash` skips trash and log, forwards to rm
- `--restore` delegates to restore.Run
- Exit code mirrors rm
- Log write failure is fatal (blocks rm)

This slice wires all packages together. Tests here are integration-style but still use the injectable Commander pattern established in earlier slices.

---

### Behavior 6.1: Happy path — trash, log, then rm are called in order

**Given** a recording Commander that succeeds for `trash` and `rm`, and a temp log path
**When** `app.Run(args, commander, logPath)` is called with `["foo.txt"]`
**Then** Commander received: `("trash", "foo.txt")` then `("/bin/rm", "foo.txt")`; log file contains one entry with `Files: ["foo.txt"]`

- [ ] **RED**: Write failing test
  - Location: `main_test.go` (or `internal/app/app_test.go` if you extract a top-level `app` package)
  - Test name: `TestRun_HappyPath_TrashThenRm`
  - Inject: Commander, logPath (temp), a fixed clock for timestamp, CWD

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: Wire `args.Parse` → `trash.LinuxBackend.Trash` → `log.Append` → `rm.Exec`; plumb Commander through

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: Confirm command order is enforced; check log entry fields

- [ ] **COMMIT**: `"feat: main wires trash → log → rm happy path"`

---

### Behavior 6.2: Trash failure with TTY — user confirms → rm runs, no log entry

**Given** a Commander where `trash` fails, and a fake `PromptFunc` that returns `"y"`
**When** `app.Run` is called
**Then** `rm` is still called; log file is empty (no entry written)

- [ ] **RED**: Write failing test
  - Test name: `TestRun_TrashFail_UserConfirms_RmRuns`
  - Add a `PromptFunc func(msg string) (string, error)` injectable to `app.Run`

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: On `Trash` error, call `PromptFunc`; if response is `"y"` or `"Y"`, skip log and call rm

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: app prompts user on trash failure; rm runs on confirm"`

---

### Behavior 6.3: Trash failure with TTY — user declines → abort, no rm, non-zero exit

**Given** a Commander where `trash` fails, and a `PromptFunc` that returns `"n"`
**When** `app.Run` is called
**Then** `rm` is NOT called; return value is a non-zero exit code

- [ ] **RED**: Write failing test
  - Test name: `TestRun_TrashFail_UserDeclines_Aborts`

- [ ] **GREEN**: If response is not `"y"`/`"Y"`, return non-zero exit code without calling rm

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: app aborts on user decline after trash failure"`

---

### Behavior 6.4: Non-TTY context — trash failure aborts automatically without prompt

**Given** a Commander where `trash` fails, and a `PromptFunc` that is never called (non-TTY)
**When** `app.Run` is called with `isTTY=false`
**Then** `rm` is NOT called; `PromptFunc` is never invoked; return is non-zero

- [ ] **RED**: Write failing test
  - Test name: `TestRun_NonTTY_TrashFail_AutoAborts`
  - Add `isTTY bool` to `app.Run` signature

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: Gate the prompt behind `isTTY`; if false, skip prompt and abort

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: app auto-aborts on trash failure in non-TTY context"`

---

### Behavior 6.5: Log write failure is fatal — rm is NOT called

**Given** a Commander where `trash` succeeds but the log path is unwritable (permissions set to 000 on parent dir)
**When** `app.Run` is called
**Then** `rm` is NOT called; return is non-zero

- [ ] **RED**: Write failing test
  - Test name: `TestRun_LogWriteFail_BlocksRm`
  - Use `os.Chmod(parentDir, 0o000)` in the test; defer `os.Chmod` back to restore

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: Check error from `log.Append`; if non-nil, return non-zero without calling rm

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: log write failure blocks rm (fatal)"`

---

### Behavior 6.6: --be-brave-skip-trash forwards directly to rm, no log entry

**Given** `--be-brave-skip-trash` flag is set, and a recording Commander
**When** `app.Run` is called with `["--be-brave-skip-trash", "foo.txt"]`
**Then** `trash` is never called; log is empty; `rm` is called with `["foo.txt"]`

- [ ] **RED**: Write failing test
  - Test name: `TestRun_SkipTrash_CallsRmDirectly`

- [ ] **GREEN**: Branch on `parsedArgs.SkipTrash`; if true, skip trash and log, call rm directly

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: --be-brave-skip-trash bypasses trash and log"`

---

### Behavior 6.7: Exit code mirrors rm exit code

**Given** a Commander where rm exits with code 2
**When** `app.Run` is called
**Then** the returned exit code is 2

- [ ] **RED**: Write failing test
  - Test name: `TestRun_ExitCodeMirrorsRm`
  - Inject a Commander that returns a Cmd with exit code 2 (use `exec.Command("sh", "-c", "exit 2")`)

- [ ] **RUN**: Confirm test FAILS

- [ ] **GREEN**: Capture exit code from rm via `cmd.ProcessState.ExitCode()`; return it

- [ ] **RUN**: Confirm test PASSES

- [ ] **REFACTOR**: None needed

- [ ] **COMMIT**: `"feat: app exit code mirrors rm exit code"`

---

## Edge Cases (All Slices)

These tests fill LOW-risk gaps not driven by a single slice.

### E1: CalledAsRm with full binary name returns false

- [x] **RED**: `TestCalledAsRm_False` — `args.CalledAsRm("/usr/local/bin/trash-rm")` → `false`
- [x] **GREEN → REFACTOR**
- [ ] **COMMIT**: `"test: CalledAsRm false for trash-rm binary name"`

### E2: Append to log is safe when parent dir already exists

- [ ] **RED**: `TestAppend_ParentDirAlreadyExists` — call Append twice; no error on second call
- [ ] **GREEN**: `os.MkdirAll` is idempotent — should pass without changes
- [ ] **COMMIT**: `"test: log.Append safe when parent dir pre-exists"`

### E3: Rewrite with empty slice clears the file

- [ ] **RED**: `TestRewrite_EmptySliceClearsFile` — write 2 entries, `Rewrite(path, []LogEntry{})`, `ReadAll` returns empty
- [ ] **GREEN**: Empty-slice write produces an empty file
- [ ] **COMMIT**: `"test: log.Rewrite with empty slice produces empty log"`

### E4: FilterAlive returns empty when no files survive

- [ ] **RED**: `TestFilterAlive_AllGarbageCollected` — all entries' files absent from trash-list → empty result
- [ ] **GREEN**: Loop exits with empty slice
- [ ] **COMMIT**: `"test: restore.FilterAlive returns empty when all GC'd"`

---

## Final Check

- [ ] **Run full test suite**: `go test ./...` — all tests pass
- [ ] **Review test names**: Read them top to bottom — do they describe trash-rm's behavior clearly?
- [ ] **Review implementation**: Any dead code, unused parameters, or logic that appeared before a test demanded it?
- [ ] **go vet**: `go vet ./...` — no warnings
- [ ] **Check that `--be-brave-skip-trash` does not appear in rm's argv**: Review Slice 2 behavior 2.2 test covers this

---

## Test Summary

| Slice | Package | # Tests | Description |
|-------|---------|---------|-------------|
| 1 | `internal/platform` | 3 | Log path resolution per OS |
| 2 | `internal/args` | 5 | Flag parsing + symlink detection |
| 3 | `internal/rm` + `internal/trash` | 5 | Real-rm path + trash backends |
| 4 | `internal/log` | 6 | NDJSON append / read / rewrite |
| 5 | `internal/restore` | 5 | Cross-reference, restore, log cleanup |
| 6 | `main` / `app` | 7 | Full wiring: all happy and failure paths |
| Edge | various | 4 | Gap-fill edge cases |
| **Total** | | **35** | |

---

## Notes for the Implementer

**Commander injection pattern** — used across slices 3, 5, 6:
```go
type Commander func(name string, args ...string) *exec.Cmd
```
In tests, supply a function that records calls and returns `exec.Command("true")` or `exec.Command("false")`.
In production, pass `exec.Command` directly.

**SelectFunc injection** — used in Slice 5 to decouple bubbletea TUI:
```go
type SelectFunc func(entries []log.LogEntry) (int, error)
```
In tests, pass `func(entries []log.LogEntry) (int, error) { return 0, nil }`.
In production, wire in the bubbletea list component.

**TTY detection** — use `golang.org/x/term`'s `term.IsTerminal(int(os.Stdin.Fd()))` in `main.go`; pass the result as `isTTY bool` into `app.Run` so it is testable without a real TTY.

**Build tags** — if platform-specific files are needed (e.g., `trash_linux.go`, `trash_darwin.go`), use Go build constraints (`//go:build linux`) rather than runtime switches where possible. Both patterns are acceptable; pick one and be consistent.

[x] Reviewed
