# Guided Setup Wizard for Middle-School FTC Users

## Problem

`ftc-helper` targets middle-school FIRST Tech Challenge students setting up a
coding laptop for the first time. Today that means running several separate
commands in the right order (`download-git`, `download-studio`,
`download-rev`, `init`, `launch`), reading raw Go error text when something
fails, and tracking manually which step comes next. For a first-time,
non-technical user this is the single biggest source of confusion and
abandoned setups.

## Goals

- One command (`ftc-helper setup`) walks a kid through the entire laptop
  setup end to end: install Git, Android Studio, REV Hardware Client, create
  their project, and launch it.
- Skip any tool that's already installed — no redundant downloads/installs.
- Every failure shows a plain-English explanation (not a raw Go error) and
  offers to retry that step.
- Re-running `setup` after a failure or partial run picks up where it left
  off, for free, via idempotent detection — no separate state file.
- Existing commands (`list`, `init`, `launch`, `download-*`) keep working
  unchanged for advanced/repeat use; the wizard is additive, not a
  replacement.

## Non-goals

- No git-remote/GitHub setup in the wizard (kids without a team repo yet
  aren't blocked; `init --git` remains available for that later).
- No new persisted config/state file.
- No automated end-to-end test of the interactive wizard itself (manual
  smoke test only, same as today).

## Architecture

New `setup` command (Cobra), in a new `setup.go`, orchestrates existing logic
rather than reimplementing it:

```
ftc-helper setup
  Step 1/5: Git                  → detect on PATH, else download+install
  Step 2/5: Android Studio       → detect via findAndroidStudioExe, else download+install
  Step 3/5: REV Hardware Client  → detect common install path, else download+install
  Step 4/5: Project              → ask project name + version (default latest), reuse init logic
  Step 5/5: Launch                → reuse launch logic, opens Android Studio
```

Resumability is achieved purely through idempotent detection at each step
(tool already on disk → skip; project dir already exists → skip create, go
straight to launch) — re-running `setup` after any failure naturally
continues from where it stopped.

## Components

- **`setup.go`** — new file. `setupCmd` + `runSetupWizard()`. Thin
  orchestrator only; calls into existing helpers
  (`findLatestGitForWindows`, `findAndroidStudioExe`,
  `findRevHardwareClientURL`, `downloadURLToPath`, `runBinary`,
  `extractZip`, and the extracted `runInit`/`runLaunch` below).
- **`ui.go`** — new small file, shared UI helpers:
  - `stepHeader(n, total int, label string)` — colored `Step 2/5: Installing
    Android Studio...` header.
  - `ok(msg string)` / `fail(msg string)` — green `✓` / red `✗` lines.
  - `friendlyError(err error) string` — maps common failure modes (network
    timeout/DNS failure, GitHub API rate limit / 403, permission denied,
    disk full, Android Studio not found, corrupt/partial download zip) to
    plain-English text; unmapped errors fall through to
    `"Something went wrong: " + err.Error()` (never silently swallowed).
  - `confirmRetry(prompt string) bool` — y/n retry loop, reused per step.
- **Color dependency**: `fatih/color` for the ✓/✗ and step-header coloring.
  Works on Windows Terminal/PowerShell/modern cmd, macOS, Linux.
- **Refactor**: extract `initCmd`'s `Run` body into
  `runInit(version, projectName, gitURL string) error`, and `launchCmd`'s
  body into `runLaunch(projectName string) error`. Both existing commands
  and the new wizard call these — avoids duplicating download/extract/launch
  logic.

## Data flow

1. `setupCmd.Run` calls `runSetupWizard()`.
2. Each of steps 1–3: print header → detect → if present, `ok("Already
   installed")` and move on → else download → install → on error,
   `friendlyError` + `confirmRetry` loop → `ok()` once done.
3. Step 4: prompt (stdin) for project name; show recent versions from the
   existing `list` logic, defaulting to the latest if the kid just presses
   enter; call `runInit`.
4. Step 5: call `runLaunch` with the project name from step 4.

## Error handling

`friendlyError` covers the realistic failure set for this codebase: no
internet (timeout/DNS), GitHub API rate-limiting (403), disk
full/permission-denied on write, Android Studio executable not found
(reuse existing message), corrupt/partial zip download. Everything else
falls through to a generic but clearly-labeled message — nothing fails
silently.

## Testing

- `friendlyError` — table-driven unit test mapping known error types to
  expected output strings (same style as `androidversion_test.go`).
- `runInit`/`runLaunch` extraction — existing `TestExtractZip`-style
  temp-dir tests continue to cover download/extract; add one test asserting
  calling `runInit` twice on the same project directory is a no-op the
  second time (the resumability contract).
- Git-on-PATH detection (`detectGitInstalled() bool`) — thin wrapper around
  `exec.Command("git", "--version")`; a single happy-path test is enough.
- No automated test of the interactive wizard flow (stdin prompts, real
  installs) — manual smoke test only.
