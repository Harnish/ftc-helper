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
