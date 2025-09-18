package main

import (
	"errors"
	"os/exec"
	"regexp"
	"strings"
)

// DetectGitVersion runs `git --version` and returns the parsed version string like "2.39.1".
func DetectGitVersion() (string, error) {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return "", err
	}
	return ParseGitVersion(string(out))
}

// ParseGitVersion extracts a semantic-like version from the output of `git --version`.
// Examples it handles:
// - git version 2.39.1.windows.1 -> 2.39.1
// - git version 2.25.1 -> 2.25.1
// - git version 2.34.1 (Apple Git-137) -> 2.34.1
func ParseGitVersion(output string) (string, error) {
	if output == "" {
		return "", errors.New("empty output")
	}
	// Typical output: "git version X.Y.Z..."
	output = strings.TrimSpace(output)
	// Find the first sequence that looks like a version number
	re := regexp.MustCompile(`\d+\.\d+(?:\.\d+)?`)
	m := re.FindString(output)
	if m == "" {
		return "", errors.New("could not parse git version")
	}
	return m, nil
}
