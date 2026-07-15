# Guided Setup Wizard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a single `ftc-helper setup` command that walks a middle-school FTC student through installing Git, Android Studio, and REV Hardware Client, then creates and launches their first project — skipping anything already installed and showing plain-English errors on failure.

**Architecture:** New `setup.go` orchestrates existing download/install/init/launch logic (no rewritten business logic). Two existing commands (`init`, `launch`) get their `Run` bodies extracted into standalone `runInit`/`runLaunch` functions so both the original commands and the new wizard call the same code. A new `ui.go` provides colored step headers, ✓/✗ status lines, a y/n retry prompt, and a `friendlyError` translator. Resumability comes from idempotent detection at each step — no new state file.

**Tech Stack:** Go, Cobra, `github.com/fatih/color` (new dependency for terminal color).

## Global Constraints

- No new persisted config/state file — resumability is achieved only via idempotent detection (spec: "Resumability").
- Existing commands (`list`, `init`, `launch`, `download-git`, `download-studio`, `download-rev`, `download-bambu`) must keep working unchanged for standalone use (spec: "Goals").
- The wizard does not prompt for or configure a git remote (spec: "Non-goals").
- Project layout convention is fixed: `<workDir>/<project>/TeamCode/src/main/java/org/firstinspires/ftc/teamcode` (existing convention, documented in CLAUDE.md).
- Color library: `github.com/fatih/color` (spec: "Components").
- `friendlyError(err error) string` never returns an empty string for a non-nil error, and unmapped errors fall through to `"Something went wrong: " + err.Error()` — never silently swallowed (spec: "Error handling").

---

## Task 1: Add color dependency

**Files:**
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Produces: `github.com/fatih/color` importable as `"github.com/fatih/color"` for later tasks.

- [ ] **Step 1: Fetch the dependency**

Run: `go get github.com/fatih/color@latest`
Expected: `go.mod` gains a `require github.com/fatih/color vX.Y.Z` line; `go.sum` gains matching entries.

- [ ] **Step 2: Tidy modules**

Run: `go mod tidy`
Expected: exits 0, no changes beyond what Step 1 already added (may add fatih/color's transitive deps like `mattn/go-colorable`, `mattn/go-isatty`).

- [ ] **Step 3: Verify it builds**

Run: `go build ./...`
Expected: exits 0 (nothing imports color yet, but the module must resolve cleanly).

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add fatih/color dependency for setup wizard UI"
```

---

## Task 2: UI helpers (`ui.go`)

**Files:**
- Create: `ui.go`
- Test: `ui_test.go`

**Interfaces:**
- Produces (used by Task 6 and by refactored `initCmd`/`launchCmd`/`listCmd` in Tasks 3–5):
  - `func stepHeader(n, total int, label string)`
  - `func ok(msg string)`
  - `func fail(msg string)`
  - `func friendlyError(err error) string`
  - `func confirmRetry(prompt string) bool`

- [ ] **Step 1: Write the failing test for `friendlyError`**

Create `ui_test.go`:

```go
package main

import (
	"errors"
	"strings"
	"testing"
)

func TestFriendlyError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"dns failure", errors.New("dial tcp: lookup github.com: no such host"), "Couldn't reach the internet. Check your wifi and try again."},
		{"github rate limit", errors.New("unexpected status code: 403"), "GitHub is temporarily limiting requests. Wait a few minutes and try again."},
		{"permission denied", errors.New("open C:\\out.exe: permission denied"), "Windows blocked writing that file. Try running as Administrator, or pick a different folder."},
		{"disk full", errors.New("write out.zip: no space left on device"), "Your disk is full. Free up some space and try again."},
		{"studio not found", errors.New("Could not find Android Studio executable. Set ANDROID_STUDIO_PATH or android_studio_path in config."), "Couldn't find Android Studio. Set ANDROID_STUDIO_PATH or install it first."},
		{"corrupt zip", errors.New("zip: not a valid zip file"), "The download got cut off. Let's try downloading it again."},
		{"unmapped", errors.New("something bizarre happened"), "Something went wrong: something bizarre happened"},
	}

	for _, c := range cases {
		got := friendlyError(c.err)
		if got != c.want {
			t.Errorf("%s: friendlyError(%q) = %q, want %q", c.name, c.err.Error(), got, c.want)
		}
	}
}

func TestFriendlyErrorNil(t *testing.T) {
	if got := friendlyError(nil); got != "" {
		t.Errorf("friendlyError(nil) = %q, want empty string", got)
	}
}

func TestFriendlyErrorNeverEmptyForNonNil(t *testing.T) {
	err := errors.New("x")
	got := friendlyError(err)
	if !strings.Contains(got, "x") {
		t.Errorf("expected fallback to contain original message, got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestFriendlyError -v ./...`
Expected: FAIL — `undefined: friendlyError`

- [ ] **Step 3: Implement `ui.go`**

Create `ui.go`:

```go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// stepHeader prints a colored progress header, e.g. "Step 2/5: Installing Android Studio".
func stepHeader(n, total int, label string) {
	color.New(color.FgCyan, color.Bold).Printf("\nStep %d/%d: %s\n", n, total, label)
}

// ok prints a green checkmark line.
func ok(msg string) {
	color.New(color.FgGreen).Println("  ✓ " + msg)
}

// fail prints a red X line.
func fail(msg string) {
	color.New(color.FgRed).Println("  ✗ " + msg)
}

// friendlyError translates common failure modes into plain-English text for
// non-technical users. Unmapped errors fall through to a generic but clearly
// labeled message so nothing fails silently.
func friendlyError(err error) string {
	if err == nil {
		return ""
	}

	lower := strings.ToLower(err.Error())

	switch {
	case strings.Contains(lower, "no such host"), strings.Contains(lower, "dial tcp"), strings.Contains(lower, "timeout"):
		return "Couldn't reach the internet. Check your wifi and try again."
	case strings.Contains(lower, "403"):
		return "GitHub is temporarily limiting requests. Wait a few minutes and try again."
	case strings.Contains(lower, "permission denied"):
		return "Windows blocked writing that file. Try running as Administrator, or pick a different folder."
	case strings.Contains(lower, "no space left"):
		return "Your disk is full. Free up some space and try again."
	case strings.Contains(lower, "could not find android studio executable"):
		return "Couldn't find Android Studio. Set ANDROID_STUDIO_PATH or install it first."
	case strings.Contains(lower, "not a valid zip file"), strings.Contains(lower, "unexpected eof"):
		return "The download got cut off. Let's try downloading it again."
	default:
		return "Something went wrong: " + err.Error()
	}
}

// confirmRetry prompts the user with a yes/no question and returns true for
// an affirmative answer. Any non-"y"/"yes" input (including read failure) is
// treated as "no".
func confirmRetry(prompt string) bool {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("%s (y/n): ", prompt)
	if !scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes"
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestFriendlyError -v ./...`
Expected: PASS (all three test functions)

- [ ] **Step 5: Commit**

```bash
git add ui.go ui_test.go
git commit -m "feat: add colored step UI and friendly error translation"
```

---

## Task 3: Extract `fetchReleases` (main.go)

**Files:**
- Modify: `main.go` (the `listCmd` block, originally lines 110–143)

**Interfaces:**
- Produces: `func fetchReleases() ([]Release, error)` — used by Task 6's `latestReleaseTag()`.
- Consumes: existing `Release` struct (`main.go`, `TagName string`).

- [ ] **Step 1: Replace the `listCmd` block**

In `main.go`, find:

```go
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists available FTC releases",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := http.Get("https://api.github.com/repos/FIRST-Tech-Challenge/FtcRobotController/releases")
		if err != nil {
			fmt.Println("Error fetching releases:", err)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		var releases []Release
		if err := json.Unmarshal(body, &releases); err != nil {
			fmt.Println("Error parsing JSON:", err)
			return
		}

		fmt.Println("Available FTC releases:")
		for _, release := range releases {
			fmt.Println("-", release.TagName)
		}
	},
}
```

Replace with:

```go
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists available FTC releases",
	Run: func(cmd *cobra.Command, args []string) {
		releases, err := fetchReleases()
		if err != nil {
			fmt.Println(friendlyError(err))
			return
		}

		fmt.Println("Available FTC releases:")
		for _, release := range releases {
			fmt.Println("-", release.TagName)
		}
	},
}

// fetchReleases fetches the list of FTC Robot Controller releases from GitHub.
func fetchReleases() ([]Release, error) {
	resp, err := http.Get("https://api.github.com/repos/FIRST-Tech-Challenge/FtcRobotController/releases")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}
	return releases, nil
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: exits 0 (no new test for this step — `fetchReleases` hits the real GitHub API, so it's exercised via the existing manual `./ftc-helper list` smoke test, not a unit test, per the plan's no-network-in-unit-tests constraint).

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "refactor: extract fetchReleases from listCmd for reuse in setup wizard"
```

---

## Task 4: Extract `runInit` with idempotency guard (main.go)

**Files:**
- Modify: `main.go` (the `initCmd` block, originally lines 145–234)
- Test: `main_test.go`

**Interfaces:**
- Produces: `func runInit(version, projectName, gitURL string) error` — used by Task 6's `runSetupWizard`.
- Consumes: existing `extractZip(src, dest string) error`.

- [ ] **Step 1: Write the failing idempotency test**

Add to `main_test.go`:

```go
func TestRunInit_SkipsIfProjectAlreadyExists(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "runinit-test")
	if err != nil {
		t.Fatalf("tmpdir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origWorkDir := workDir
	workDir = tmpDir
	defer func() { workDir = origWorkDir }()

	teamCodePath := filepath.Join(tmpDir, "myproject", "TeamCode", "src", "main", "java", "org", "firstinspires", "ftc", "teamcode")
	if err := os.MkdirAll(teamCodePath, 0755); err != nil {
		t.Fatalf("mkdir teamCodePath: %v", err)
	}

	// Should return nil immediately without attempting any network download,
	// since the project directory already exists.
	if err := runInit("v11.0", "myproject", ""); err != nil {
		t.Fatalf("expected no error for already-initialized project, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestRunInit_SkipsIfProjectAlreadyExists -v ./...`
Expected: FAIL — `undefined: runInit`

- [ ] **Step 3: Replace the `initCmd` block**

In `main.go`, find the full `initCmd` block (from `// Mode 2: Initialize Project` through the closing `}` before `func extractZip`):

```go
// Mode 2: Initialize Project
var initCmd = &cobra.Command{
	Use:   "init [version]",
	Short: "Initializes a new FTC project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version := args[0]
		projectName, _ := cmd.Flags().GetString("project")
		gitURL, _ := cmd.Flags().GetString("git")

		if projectName == "" {
			fmt.Println("Project name is required. Use --project flag.")
			return
		}

		projectPath := filepath.Join(workDir, projectName)
		zipURL := fmt.Sprintf("https://github.com/FIRST-Tech-Challenge/FtcRobotController/archive/refs/tags/%s.zip", version)

		fmt.Printf("Downloading %s to %s...\n", zipURL, projectPath)
		resp, err := http.Get(zipURL)
		if err != nil {
			fmt.Println("Error downloading file:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Error: Received status code %d\n", resp.StatusCode)
			return
		}

		// Save the zip file
		zipFile, err := ioutil.TempFile("", "ftc-*.zip")
		if err != nil {
			fmt.Println("Error creating temp file:", err)
			return
		}
		defer os.Remove(zipFile.Name())
		defer zipFile.Close()

		_, err = io.Copy(zipFile, resp.Body)
		if err != nil {
			fmt.Println("Error writing to temp file:", err)
			return
		}

		// Unzip the file
		fmt.Println("Extracting files...")
		if err := extractZip(zipFile.Name(), projectPath); err != nil {
			fmt.Println("Error extracting zip:", err)
			return
		}

		// Move contents up one level
		subDir := filepath.Join(projectPath, fmt.Sprintf("FtcRobotController-%s", version))
		subDirFixed := strings.Replace(subDir, "v", "", 1)
		if _, err := os.Stat(subDirFixed); err == nil {
			files, _ := os.ReadDir(subDirFixed)
			for _, f := range files {
				os.Rename(filepath.Join(subDirFixed, f.Name()), filepath.Join(projectPath, f.Name()))
			}
			os.Remove(subDirFixed)
		}

		// Git setup

		teamCodePath := filepath.Join(projectPath, "TeamCode", "src", "main", "java", "org", "firstinspires", "ftc", "teamcode")
		fmt.Println("Initializing git repository...")
		cmdGit := exec.Command("git", "init")
		cmdGit.Dir = teamCodePath
		if err := cmdGit.Run(); err != nil {
			fmt.Println("Error initializing git repo:", err)
		}

		if gitURL != "" {
			fmt.Printf("Setting up remote to %s...\n", gitURL)
			remoteURL := gitURL
			if !strings.HasPrefix(gitURL, "https://") && !strings.HasPrefix(gitURL, "git@") {
				remoteURL = "https://" + gitURL
			}
			cmdGitRemote := exec.Command("git", "remote", "add", "origin", remoteURL)
			cmdGitRemote.Dir = teamCodePath
			if err := cmdGitRemote.Run(); err != nil {
				fmt.Println("Error adding git remote:", err)
			}
		}

		fmt.Println("Project setup complete!")
	},
}
```

Replace with:

```go
// Mode 2: Initialize Project
var initCmd = &cobra.Command{
	Use:   "init [version]",
	Short: "Initializes a new FTC project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version := args[0]
		projectName, _ := cmd.Flags().GetString("project")
		gitURL, _ := cmd.Flags().GetString("git")

		if projectName == "" {
			fmt.Println("Project name is required. Use --project flag.")
			return
		}

		if err := runInit(version, projectName, gitURL); err != nil {
			fmt.Println(friendlyError(err))
		}
	},
}

// runInit downloads the given FTC Robot Controller release, extracts it into
// workDir/projectName, and initializes a git repo in the TeamCode directory.
// If the project's TeamCode directory already exists, it returns nil
// immediately without re-downloading anything (idempotent / resumable).
func runInit(version, projectName, gitURL string) error {
	projectPath := filepath.Join(workDir, projectName)
	teamCodePath := filepath.Join(projectPath, "TeamCode", "src", "main", "java", "org", "firstinspires", "ftc", "teamcode")

	if _, err := os.Stat(teamCodePath); err == nil {
		fmt.Println("Project already exists, skipping download.")
		return nil
	}

	zipURL := fmt.Sprintf("https://github.com/FIRST-Tech-Challenge/FtcRobotController/archive/refs/tags/%s.zip", version)

	fmt.Printf("Downloading %s to %s...\n", zipURL, projectPath)
	resp, err := http.Get(zipURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received status code %d", resp.StatusCode)
	}

	// Save the zip file
	zipFile, err := ioutil.TempFile("", "ftc-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(zipFile.Name())
	defer zipFile.Close()

	_, err = io.Copy(zipFile, resp.Body)
	if err != nil {
		return err
	}

	// Unzip the file
	fmt.Println("Extracting files...")
	if err := extractZip(zipFile.Name(), projectPath); err != nil {
		return err
	}

	// Move contents up one level
	subDir := filepath.Join(projectPath, fmt.Sprintf("FtcRobotController-%s", version))
	subDirFixed := strings.Replace(subDir, "v", "", 1)
	if _, err := os.Stat(subDirFixed); err == nil {
		files, _ := os.ReadDir(subDirFixed)
		for _, f := range files {
			os.Rename(filepath.Join(subDirFixed, f.Name()), filepath.Join(projectPath, f.Name()))
		}
		os.Remove(subDirFixed)
	}

	// Git setup
	fmt.Println("Initializing git repository...")
	cmdGit := exec.Command("git", "init")
	cmdGit.Dir = teamCodePath
	if err := cmdGit.Run(); err != nil {
		return fmt.Errorf("initializing git repo: %w", err)
	}

	if gitURL != "" {
		fmt.Printf("Setting up remote to %s...\n", gitURL)
		remoteURL := gitURL
		if !strings.HasPrefix(gitURL, "https://") && !strings.HasPrefix(gitURL, "git@") {
			remoteURL = "https://" + gitURL
		}
		cmdGitRemote := exec.Command("git", "remote", "add", "origin", remoteURL)
		cmdGitRemote.Dir = teamCodePath
		if err := cmdGitRemote.Run(); err != nil {
			return fmt.Errorf("adding git remote: %w", err)
		}
	}

	fmt.Println("Project setup complete!")
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestRunInit_SkipsIfProjectAlreadyExists -v ./...`
Expected: PASS

- [ ] **Step 5: Run full test suite to check nothing broke**

Run: `go test ./...`
Expected: PASS (all packages)

- [ ] **Step 6: Commit**

```bash
git add main.go main_test.go
git commit -m "refactor: extract runInit with idempotency guard for resumable setup"
```

---

## Task 5: Extract `runLaunch` (main.go)

**Files:**
- Modify: `main.go` (the `launchCmd` block, originally lines 316–371)
- Test: `main_test.go`

**Interfaces:**
- Produces: `func runLaunch(projectName string) error` — used by Task 6's `runSetupWizard`.
- Consumes: existing `findAndroidStudioExe() (string, error)`.

- [ ] **Step 1: Write the failing test**

Add to `main_test.go`:

```go
func TestRunLaunch_ProjectNotFound(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "runlaunch-test")
	if err != nil {
		t.Fatalf("tmpdir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origWorkDir := workDir
	workDir = tmpDir
	defer func() { workDir = origWorkDir }()

	err = runLaunch("does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent project, got nil")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("expected error to mention project name, got: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestRunLaunch_ProjectNotFound -v ./...`
Expected: FAIL — `undefined: runLaunch`

- [ ] **Step 3: Replace the `launchCmd` block**

In `main.go`, find:

```go
// Mode 3: Launch Project
var launchCmd = &cobra.Command{
	Use:   "launch [project_name]",
	Short: "Launches a local project in Android Studio",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		projectPath := filepath.Join(workDir, projectName)

		if _, err := os.Stat(projectPath); os.IsNotExist(err) {
			fmt.Println("Project not found:", projectName)
			return
		}

		var launchCmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin": // macOS
			launchCmd = exec.Command("open", "-a", "Android Studio.app", projectPath)
			launchCmd.Stdout = os.Stdout
			launchCmd.Stderr = os.Stderr
			if err := launchCmd.Start(); err != nil {
				fmt.Println("Error launching Android Studio on macOS:", err)
			} else {
				fmt.Printf("Launched '%s' in Android Studio (pid %d)\n", projectName, launchCmd.Process.Pid)
			}
			return
		case "linux":
			launchCmd = exec.Command("android-studio", projectPath)
			launchCmd.Stdout = os.Stdout
			launchCmd.Stderr = os.Stderr
			if err := launchCmd.Start(); err != nil {
				fmt.Println("Error launching Android Studio on Linux:", err)
			} else {
				fmt.Printf("Launched '%s' in Android Studio (pid %d)\n", projectName, launchCmd.Process.Pid)
			}
			return
		case "windows":
			exePath, err := findAndroidStudioExe()
			if err != nil {
				fmt.Println(err)
				return
			}
			launchCmd = exec.Command(exePath, projectPath)
			// Start non-blocking so this CLI can return immediately
			if err := launchCmd.Start(); err != nil {
				fmt.Println("Error launching Android Studio on Windows:", err)
			} else {
				fmt.Printf("Launched '%s' in Android Studio (pid %d) using '%s'\n", projectName, launchCmd.Process.Pid, exePath)
			}
			return
		default:
			fmt.Println("Unsupported operating system for launching Android Studio.")
			return
		}
	},
}
```

Replace with:

```go
// Mode 3: Launch Project
var launchCmd = &cobra.Command{
	Use:   "launch [project_name]",
	Short: "Launches a local project in Android Studio",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runLaunch(args[0]); err != nil {
			fmt.Println(friendlyError(err))
		}
	},
}

// runLaunch opens the given project in Android Studio for the current OS.
func runLaunch(projectName string) error {
	projectPath := filepath.Join(workDir, projectName)

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("project not found: %s", projectName)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", "-a", "Android Studio.app", projectPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("launching Android Studio on macOS: %w", err)
		}
		fmt.Printf("Launched '%s' in Android Studio (pid %d)\n", projectName, cmd.Process.Pid)
		return nil
	case "linux":
		cmd = exec.Command("android-studio", projectPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("launching Android Studio on Linux: %w", err)
		}
		fmt.Printf("Launched '%s' in Android Studio (pid %d)\n", projectName, cmd.Process.Pid)
		return nil
	case "windows":
		exePath, err := findAndroidStudioExe()
		if err != nil {
			return err
		}
		cmd = exec.Command(exePath, projectPath)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("launching Android Studio on Windows: %w", err)
		}
		fmt.Printf("Launched '%s' in Android Studio (pid %d) using '%s'\n", projectName, cmd.Process.Pid, exePath)
		return nil
	default:
		return errors.New("unsupported operating system for launching Android Studio")
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestRunLaunch_ProjectNotFound -v ./...`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add main.go main_test.go
git commit -m "refactor: extract runLaunch for reuse in setup wizard"
```

---

## Task 6: Setup wizard (`setup.go`)

**Files:**
- Create: `setup.go`
- Test: `setup_test.go`
- Modify: `main.go` (register the command)

**Interfaces:**
- Consumes: `stepHeader`, `ok`, `fail`, `friendlyError`, `confirmRetry` (Task 2); `fetchReleases` (Task 3); `runInit`, `runLaunch` (Tasks 4–5); existing `findLatestGitForWindows`, `downloadURLToPath`, `runBinary`, `findAndroidStudioExe`, `detectStudioPlatform`, `findLatestAndroidStudioURL`, `findRevHardwareClientURL`.
- Produces: `setupCmd *cobra.Command`, registered in `main.go`'s `init()`.

- [ ] **Step 1: Write failing tests for the pure detection helpers**

Create `setup_test.go`:

```go
package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDetectGitInstalled_MatchesLookPath(t *testing.T) {
	_, lookErr := exec.LookPath("git")
	want := lookErr == nil
	if got := detectGitInstalled(); got != want {
		t.Fatalf("detectGitInstalled() = %v, want %v (matching exec.LookPath)", got, want)
	}
}

func TestDetectRevInstalledIn(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "rev-detect-test")
	if err != nil {
		t.Fatalf("tmpdir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if detectRevInstalledIn([]string{tmpDir}) {
		t.Fatal("expected false when REV Hardware Client is not present")
	}

	revDir := filepath.Join(tmpDir, "REV Hardware Client")
	if err := os.MkdirAll(revDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	exePath := filepath.Join(revDir, "REV Hardware Client.exe")
	if err := os.WriteFile(exePath, []byte("dummy"), 0755); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	if !detectRevInstalledIn([]string{tmpDir}) {
		t.Fatal("expected true when REV Hardware Client.exe is present")
	}
}

func TestLatestReleaseTag_FallsBackOnError(t *testing.T) {
	// fetchReleases hits the network; this only checks the fallback path
	// works when passed an empty slice directly via the helper's shape.
	got := latestReleaseTagFrom(nil)
	if got != "v11.0" {
		t.Fatalf("expected fallback v11.0 for no releases, got %q", got)
	}
	got = latestReleaseTagFrom([]Release{{TagName: "v12.1"}})
	if got != "v12.1" {
		t.Fatalf("expected v12.1, got %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run 'TestDetectGitInstalled_MatchesLookPath|TestDetectRevInstalledIn|TestLatestReleaseTag_FallsBackOnError' -v ./...`
Expected: FAIL — `undefined: detectGitInstalled`, `undefined: detectRevInstalledIn`, `undefined: latestReleaseTagFrom`

- [ ] **Step 3: Implement `setup.go`**

Create `setup.go`:

```go
package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const setupTotalSteps = 5

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Guided setup: installs Git, Android Studio, REV Hardware Client, and creates your first FTC project",
	Run: func(cmd *cobra.Command, args []string) {
		runSetupWizard()
	},
}

func runSetupWizard() {
	reader := bufio.NewScanner(os.Stdin)

	stepHeader(1, setupTotalSteps, "Installing Git")
	setupStepWithRetry("Git", func() error {
		if detectGitInstalled() {
			ok("Git is already installed")
			return nil
		}
		gitURL, filename, err := findLatestGitForWindows()
		if err != nil {
			return err
		}
		if err := downloadURLToPath(gitURL, filename); err != nil {
			return err
		}
		runBinary(filename)
		ok("Git installed")
		return nil
	})

	stepHeader(2, setupTotalSteps, "Installing Android Studio")
	setupStepWithRetry("Android Studio", func() error {
		if _, err := findAndroidStudioExe(); err == nil {
			ok("Android Studio is already installed")
			return nil
		}
		platform := detectStudioPlatform()
		if platform == "" {
			return fmt.Errorf("unsupported OS for automatic Android Studio download")
		}
		downloadURL, err := findLatestAndroidStudioURL(platform)
		if err != nil {
			return err
		}
		u, _ := url.Parse(downloadURL)
		outPath := path.Base(u.Path)
		if err := downloadURLToPath(downloadURL, outPath); err != nil {
			return err
		}
		runBinary(outPath)
		ok("Android Studio installed")
		return nil
	})

	stepHeader(3, setupTotalSteps, "Installing REV Hardware Client")
	setupStepWithRetry("REV Hardware Client", func() error {
		if detectRevInstalled() {
			ok("REV Hardware Client is already installed")
			return nil
		}
		downloadURL, filename, err := findRevHardwareClientURL()
		if err != nil {
			return err
		}
		if err := downloadURLToPath(downloadURL, filename); err != nil {
			return err
		}
		runBinary(filename)
		ok("REV Hardware Client installed")
		return nil
	})

	stepHeader(4, setupTotalSteps, "Creating your project")
	fmt.Print("What do you want to name your project? ")
	reader.Scan()
	projectName := strings.TrimSpace(reader.Text())

	version := latestReleaseTag()
	fmt.Printf("Using FTC version %s. Press enter to accept, or type a different version: ", version)
	reader.Scan()
	if v := strings.TrimSpace(reader.Text()); v != "" {
		version = v
	}

	created := setupStepWithRetry("project creation", func() error {
		return runInit(version, projectName, "")
	})
	if !created {
		fmt.Println("Stopping setup. Run 'ftc-helper setup' again later to retry.")
		return
	}
	ok("Project created")

	stepHeader(5, setupTotalSteps, "Launching your project")
	if err := runLaunch(projectName); err != nil {
		fail(friendlyError(err))
		fmt.Println("You can launch it later with: ftc-helper launch " + projectName)
		return
	}
	ok("All done! Android Studio is opening your project.")
}

// setupStepWithRetry runs action, and on error shows a friendly message and
// asks whether to retry. Returns true once action succeeds, false if the
// user declines to retry.
func setupStepWithRetry(label string, action func() error) bool {
	for {
		err := action()
		if err == nil {
			return true
		}
		fail(friendlyError(err))
		if !confirmRetry("Try " + label + " again?") {
			fmt.Println("Skipping " + label + " for now. Run 'ftc-helper setup' again later to retry.")
			return false
		}
	}
}

// detectGitInstalled reports whether git is available on PATH.
func detectGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// detectRevInstalled reports whether REV Hardware Client appears to be
// installed in a common Windows Program Files location.
func detectRevInstalled() bool {
	return detectRevInstalledIn([]string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		"C:\\Program Files",
		"C:\\Program Files (x86)",
	})
}

// detectRevInstalledIn checks the given base directories for a REV Hardware
// Client executable. Extracted from detectRevInstalled for testability.
func detectRevInstalledIn(baseDirs []string) bool {
	candidates := []string{
		filepath.Join("REV Hardware Client", "REV Hardware Client.exe"),
		filepath.Join("REV Robotics", "REV Hardware Client", "REV Hardware Client.exe"),
	}
	for _, base := range baseDirs {
		if base == "" {
			continue
		}
		for _, c := range candidates {
			p := filepath.Join(base, c)
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	}
	return false
}

// latestReleaseTag fetches the newest FTC release tag, falling back to a
// known-good default if the network call fails.
func latestReleaseTag() string {
	releases, err := fetchReleases()
	if err != nil {
		return latestReleaseTagFrom(nil)
	}
	return latestReleaseTagFrom(releases)
}

// latestReleaseTagFrom picks the first release's tag, or a fallback if empty.
// Extracted from latestReleaseTag for testability without hitting the network.
func latestReleaseTagFrom(releases []Release) string {
	if len(releases) == 0 {
		return "v11.0"
	}
	return releases[0].TagName
}
```

- [ ] **Step 4: Register `setupCmd` in `main.go`**

In `main.go`, find (inside the first `func init()`):

```go
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(launchCmd)
```

Replace with:

```go
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(launchCmd)
	rootCmd.AddCommand(setupCmd)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -run 'TestDetectGitInstalled_MatchesLookPath|TestDetectRevInstalledIn|TestLatestReleaseTag_FallsBackOnError' -v ./...`
Expected: PASS

- [ ] **Step 6: Run full test suite and build**

Run: `go test ./... && go build -o ftc-helper .`
Expected: both exit 0

- [ ] **Step 7: Manual smoke test**

Run: `./ftc-helper setup`
Expected: prints colored `Step 1/5: Installing Git`, detects your local Git/PATH correctly, prompts for project name and version, and (network permitting) creates and launches a project. Interrupt with Ctrl+C is fine for this manual check — it's confirming the flow reads correctly end to end, not exercising every branch.

- [ ] **Step 8: Commit**

```bash
git add setup.go setup_test.go main.go
git commit -m "feat: add guided setup wizard command"
```

---

## Task 7: Docs update

**Files:**
- Modify: `QUICKSTART.md`
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Interfaces:**
- None (docs only).

- [ ] **Step 1: Rewrite `QUICKSTART.md` to lead with `setup`**

Replace the full contents of `QUICKSTART.md`:

```markdown
# Quickstart usage

* Download the ftc-helper for your Operating System
* Run this one command — it installs Git, Android Studio, and REV Hardware Client if you need them, then creates and opens your first project:
```
./ftc-helper.exe setup
```

Prefer to do it step by step instead? See the full command list in [README.md](README.md).
```

- [ ] **Step 2: Add a `setup` section to `README.md`**

In `README.md`, find:

```
### Commands

#### `list`
```

Replace with:

```
### Commands

#### `setup`

Guided first-time setup. Installs Git, Android Studio, and REV Hardware Client if they're not already on your computer (skipping anything already installed), then asks for a project name and FTC version and creates and launches your project. Recommended for first-time setup on a new laptop.

```bash
ftc-helper setup
```

#### `list`
```

- [ ] **Step 3: Add `setup` to the Commands table in `CLAUDE.md`**

In `CLAUDE.md`, find:

```
- `main.go` — all Cobra command definitions and their `Run` functions live here in one file (list, init, launch, pull, push, projects, config, download-studio, download-git, download-rev, download-all), plus shared helpers (`extractZip`, `downloadURLToPath`, `runBinary`, `findAndroidStudioExe`). `rootCmd.PersistentPreRun` loads Viper config (`$HOME/.ftc-helper.yaml`, overridable via `--config`) and resolves `workDir` before every command runs.
```

Replace with:

```
- `main.go` — all Cobra command definitions and their `Run` functions live here in one file (list, init, launch, pull, push, projects, config, download-studio, download-git, download-rev, download-all), plus shared helpers (`extractZip`, `downloadURLToPath`, `runBinary`, `findAndroidStudioExe`). `rootCmd.PersistentPreRun` loads Viper config (`$HOME/.ftc-helper.yaml`, overridable via `--config`) and resolves `workDir` before every command runs. `initCmd`/`launchCmd` are thin wrappers around `runInit`/`runLaunch`, which `setup.go`'s wizard also calls directly.
- `setup.go` — the `setup` command: a guided wizard for first-time users (aimed at middle-school FTC students) that installs Git/Android Studio/REV Hardware Client if missing, then creates and launches a project. Resumable by re-running — each step detects what's already done rather than tracking state in a file.
- `ui.go` — shared terminal UI helpers (`stepHeader`, `ok`, `fail`, `confirmRetry`) and `friendlyError`, which translates common Go errors into plain-English messages for non-technical users. Any command handling user-facing errors should route through `friendlyError` rather than printing `err.Error()` directly.
```

- [ ] **Step 4: Commit**

```bash
git add QUICKSTART.md README.md CLAUDE.md
git commit -m "docs: document the new setup wizard command"
```
