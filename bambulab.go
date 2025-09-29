package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var downloadBambuCmd = &cobra.Command{
	Use:   "download-bambu",
	Short: "Download the latest Bambu Studio installer for your OS",
	Run: func(cmd *cobra.Command, args []string) {
		out, _ := cmd.Flags().GetString("out")

		platform := detectBambuPlatform()
		if platform == "" {
			fmt.Println("Unsupported OS for automatic Bambu Studio download")
			return
		}

		fmt.Printf("Looking up latest Bambu Studio download for %s...\n", platform)
		downloadURL, err := findLatestBambuStudioURL(platform)
		if err != nil {
			fmt.Println("Error finding download URL:", err)
			return
		}

		fmt.Println("Found:", downloadURL)

		// Determine output path
		var outPath string
		if out != "" {
			outPath = out
		} else {
			u, _ := url.Parse(downloadURL)
			outPath = path.Base(u.Path)
		}

		fmt.Printf("Downloading to %s...\n", outPath)
		resp, err := http.Get(downloadURL)
		if err != nil {
			fmt.Println("Download error:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Download failed: status %d\n", resp.StatusCode)
			return
		}

		f, err := osCreate(outPath)
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer f.Close()

		_, err = io.Copy(f, resp.Body)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}

		fmt.Println("Download complete:", outPath)
	},
}

func init() {
	downloadBambuCmd.Flags().StringP("out", "o", "", "Output path for the downloaded installer")
}

// detectBambuPlatform maps runtime.GOOS to a simple platform string
func detectBambuPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "mac"
	case "linux":
		return "linux"
	default:
		return ""
	}
}

// findLatestBambuStudioURL scrapes the BambuLab Studio download page and returns a likely installer URL.
func findLatestBambuStudioURL(platform string) (string, error) {
	page := "https://bambulab.com/en-us/download/studio"
	resp, err := http.Get(page)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	body := string(bodyBytes)

	// Heuristic: find links ending with common installer extensions
	re := regexp.MustCompile(`https?://[\w\-./%?=&]+\.(exe|msi|dmg|pkg|AppImage|deb|tar.gz|zip)`)
	matches := re.FindAllString(body, -1)
	if len(matches) == 0 {
		return "", errors.New("no installer links found on BambuLab download page")
	}

	// prefer platform-specific matches
	lowerPlatform := strings.ToLower(platform)
	preferred := []string{}
	for _, m := range matches {
		lm := strings.ToLower(m)
		if lowerPlatform == "windows" && (strings.HasSuffix(lm, ".exe") || strings.HasSuffix(lm, ".msi") || strings.Contains(lm, "windows")) {
			preferred = append(preferred, m)
		}
		if lowerPlatform == "mac" && (strings.HasSuffix(lm, ".dmg") || strings.HasSuffix(lm, ".pkg") || strings.Contains(lm, "mac")) {
			preferred = append(preferred, m)
		}
		if lowerPlatform == "linux" && (strings.HasSuffix(lm, ".AppImage") || strings.HasSuffix(lm, ".deb") || strings.HasSuffix(lm, ".tar.gz") || strings.Contains(lm, "linux")) {
			preferred = append(preferred, m)
		}
	}

	if len(preferred) > 0 {
		return preferred[0], nil
	}

	// fallback to first match
	return matches[0], nil
}

// osCreate abstracts os.Create to keep imports tidy in this file
func osCreate(p string) (*os.File, error) {
	return os.Create(p)
}
