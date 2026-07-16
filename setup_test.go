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
