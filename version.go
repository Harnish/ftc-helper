package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Version can be set at build time with -ldflags "-X main.Version=..."
var Version = ""

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current ftc-helper version",
	Run: func(cmd *cobra.Command, args []string) {
		v, err := getVersion()
		if err != nil {
			fmt.Println("Version: unknown")
			return
		}
		fmt.Println(v)
	},
}

func getVersion() (string, error) {
	// 1) build-time version
	if Version != "" {
		return Version, nil
	}

	// 2) filesystem VERSION file at repo root
	rootOut, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		root := strings.TrimSpace(string(rootOut))
		verPath := filepath.Join(root, "VERSION")
		if b, err := ioutil.ReadFile(verPath); err == nil {
			s := strings.TrimSpace(string(b))
			if s != "" {
				return s, nil
			}
		}
	}

	// 3) try to get latest tag
	if out, err := exec.Command("git", "describe", "--tags", "--abbrev=0").Output(); err == nil {
		return strings.TrimSpace(string(out)), nil
	}

	// 4) fallback to commit
	if out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		return strings.TrimSpace(string(out)), nil
	}

	return "", fmt.Errorf("could not determine version")
}
