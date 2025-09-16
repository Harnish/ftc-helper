package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

var (
	cfgFile string
	workDir string
)

const asciiart = `+---------------------------+
|                           |
|      FTC HELPER           |
|                           |
|  [Image of a box of      |
|   macaroni and sauce]     |
|                           |
|  Delicious one-pan meal   |
|                           |
|     NET WT. 5.9 OZ.       |
|                           |
+---------------------------+`

// Main command
var rootCmd = &cobra.Command{
	Use:   "ftc-helper",
	Short: "A CLI tool for FTC robot development.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			home, _ := os.UserHomeDir()
			viper.AddConfigPath(home)
			viper.SetConfigName(".ftc-helper")
		}

		viper.AutomaticEnv()
		if err := viper.ReadInConfig(); err == nil {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
		workDir = viper.GetString("work_dir")
	},
}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	fmt.Println(asciiart)
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ftc-helper.yaml)")
	rootCmd.PersistentFlags().StringP("work-dir", "w", "", "working directory for projects")
	viper.BindPFlag("work_dir", rootCmd.PersistentFlags().Lookup("work-dir"))

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(launchCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(downloadStudioCmd)
	rootCmd.AddCommand(downloadGitCmd)
	rootCmd.AddCommand(downloadRevCmd)
	rootCmd.AddCommand(configCmd)

	initCmd.Flags().StringP("project", "p", "", "Name of the new project directory")
	initCmd.Flags().StringP("git", "g", "", "Git repository URL to set up as remote")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(home)
		viper.SetConfigName(".ftc-helper")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("No config file found. Using default values.")
		home, _ := os.UserHomeDir()
		viper.SetDefault("work_dir", home+"/StudioProjects")
	}
}

// Mode 1: List Releases
type Release struct {
	TagName string `json:"tag_name"`
}

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
		//Full hack removing the v from version
		subDir1 := strings.Replace(subDir, "v", "", 1)
		teamCodePath := filepath.Join(subDir1, "TeamCode", "src", "main", "java", "org", "firstinspires", "ftc", "teamcode")
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

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		_, err = io.Copy(outFile, rc)
		if err != nil {
			return err
		}
	}
	return nil
}

// findAndroidStudioExe tries to find the Android Studio launcher on Windows.
// Order: ANDROID_STUDIO_PATH env var, viper config "android_studio_path", common install paths.
func findAndroidStudioExe() (string, error) {
	// 1. env var override
	if p := os.Getenv("ANDROID_STUDIO_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 2. viper config (if user set it in config file)
	if v := viper.GetString("android_studio_path"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v, nil
		}
	}

	// 3. common install locations
	programFiles := []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), "C:\\Program Files", "C:\\Program Files (x86)"}
	candidates := []string{
		"Android\\Android Studio\\bin\\studio64.exe",
		"Android\\Android Studio\\bin\\studio.exe",
		"Android\\Android Studio\\bin\\launcher.exe",
		"JetBrains\\AndroidStudio\\bin\\studio64.exe",
		"JetBrains\\AndroidStudio\\bin\\studio.exe",
	}

	for _, base := range programFiles {
		if base == "" {
			continue
		}
		for _, c := range candidates {
			p := filepath.Join(base, c)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	return "", errors.New("Could not find Android Studio executable. Set ANDROID_STUDIO_PATH or android_studio_path in config.")
}

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

// Mode 4: Pull from Upstream
var pullCmd = &cobra.Command{
	Use:   "pull [project_name]",
	Short: "Pulls code into a project's TeamCode directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		teamCodePath := filepath.Join(workDir, projectName, "TeamCode", "src", "main", "java", "org", "firstinspires", "ftc", "teamcode")

		if _, err := os.Stat(teamCodePath); os.IsNotExist(err) {
			fmt.Println("Project not found or TeamCode directory does not exist.")
			return
		}

		fmt.Printf("Pulling code for project '%s'...\n", projectName)
		cmdGit := exec.Command("git", "pull")
		cmdGit.Dir = teamCodePath
		cmdGit.Stdout = os.Stdout
		cmdGit.Stderr = os.Stderr
		if err := cmdGit.Run(); err != nil {
			fmt.Println("Error pulling code:", err)
		}
	},
}

// Mode 5: Commit and Push
var pushCmd = &cobra.Command{
	Use:   "push [project_name] [commit_message]",
	Short: "Commits and pushes code for a project",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		commitMessage := args[1]
		teamCodePath := filepath.Join(workDir, projectName, "TeamCode", "src", "main", "java", "org", "firstinspires", "ftc", "teamcode")

		if _, err := os.Stat(teamCodePath); os.IsNotExist(err) {
			fmt.Println("Project not found or TeamCode directory does not exist.")
			return
		}

		fmt.Println("Staging changes...")
		cmdAdd := exec.Command("git", "add", ".")
		cmdAdd.Dir = teamCodePath
		if err := cmdAdd.Run(); err != nil {
			fmt.Println("Error staging files:", err)
			return
		}

		fmt.Println("Committing changes...")
		cmdCommit := exec.Command("git", "commit", "-m", commitMessage)
		cmdCommit.Dir = teamCodePath
		if err := cmdCommit.Run(); err != nil {
			fmt.Println("Error committing changes:", err)
			return
		}

		fmt.Println("Pushing to remote...")
		cmdPush := exec.Command("git", "push")
		cmdPush.Dir = teamCodePath
		cmdPush.Stdout = os.Stdout
		cmdPush.Stderr = os.Stderr
		if err := cmdPush.Run(); err != nil {
			fmt.Println("Error pushing code:", err)
		}
	},
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Lists all active local projects",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Active projects in:", workDir)
		projects, err := ioutil.ReadDir(workDir)
		if err != nil {
			fmt.Println("Error reading working directory:", err)
			return
		}

		found := false
		for _, p := range projects {
			if p.IsDir() {
				projectPath := filepath.Join(workDir, p.Name())
				teamCodePath := filepath.Join(projectPath, "TeamCode", "src", "main", "java", "org", "firstinspires", "ftc", "teamcode")

				if _, err := os.Stat(teamCodePath); err == nil {
					fmt.Println("-", p.Name())
					found = true
				}
			}
		}

		if !found {
			fmt.Println("No active projects found.")
		}
	},
}

// config: print current configuration as YAML
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print the current configuration as YAML",
	Run: func(cmd *cobra.Command, args []string) {
		settings := viper.AllSettings()
		// Marshal to JSON first, then convert to YAML for pretty output
		b, err := json.Marshal(settings)
		if err != nil {
			fmt.Println("Error marshaling config to JSON:", err)
			return
		}
		y, err := yaml.JSONToYAML(b)
		if err != nil {
			fmt.Println("Error converting config to YAML:", err)
			return
		}
		fmt.Println(string(y))
	},
}

// download-git: download the latest Git for Windows installer (64-bit) from GitHub releases
var downloadGitCmd = &cobra.Command{
	Use:   "download-git",
	Short: "Download the latest Git for Windows installer (64-bit)",
	Run: func(cmd *cobra.Command, args []string) {
		out, _ := cmd.Flags().GetString("out")
		fmt.Println("Looking up latest Git for Windows release...")
		url, filename, err := findLatestGitForWindows()
		if err != nil {
			fmt.Println("Error finding Git for Windows release:", err)
			return
		}

		if out == "" {
			out = filename
		}

		fmt.Printf("Downloading %s to %s...\n", url, out)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Download error:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Download failed: status %d\n", resp.StatusCode)
			return
		}

		f, err := os.Create(out)
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

		fmt.Println("Download complete:", out)
	},
}

func init() {
	downloadGitCmd.Flags().StringP("out", "o", "", "Output path for the downloaded Git installer")
}

// findLatestGitForWindows queries the Git for Windows GitHub releases and finds the latest 64-bit installer URL and filename.
func findLatestGitForWindows() (string, string, error) {
	apiURL := "https://api.github.com/repos/git-for-windows/git/releases/latest"
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", err
	}

	// Prefer 64-bit installer (MinGit vs Portable vs 64-bit setup). Typical names: Git-*-64-bit.exe, Git-*-64-bit-portable.zip
	for _, a := range data.Assets {
		lower := strings.ToLower(a.Name)
		if strings.Contains(lower, "64-bit") && (strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".msi")) {
			return a.BrowserDownloadURL, a.Name, nil
		}
	}

	// Fallback: look for installer .exe
	for _, a := range data.Assets {
		if strings.HasSuffix(strings.ToLower(a.Name), ".exe") {
			return a.BrowserDownloadURL, a.Name, nil
		}
	}

	return "", "", errors.New("no suitable Git for Windows installer found in latest release assets")
}

// download-rev: download the REV Hardware Client installer from the REV docs install page
var downloadRevCmd = &cobra.Command{
	Use:   "download-rev",
	Short: "Download the REV Hardware Client installer from the REV docs page",
	Run: func(cmd *cobra.Command, args []string) {
		out, _ := cmd.Flags().GetString("out")
		fmt.Println("Looking up REV Hardware Client installer...")
		downloadURL, filename, err := findRevHardwareClientURL()
		if err != nil {
			fmt.Println("Error finding REV Hardware Client URL:", err)
			return
		}

		if out == "" {
			out = filename
		}

		fmt.Printf("Downloading %s to %s...\n", downloadURL, out)
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

		f, err := os.Create(out)
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

		fmt.Println("Download complete:", out)
	},
}

func init() {
	downloadRevCmd.Flags().StringP("out", "o", "", "Output path for the downloaded REV installer")
}

// findRevHardwareClientURL scrapes the REV docs install page and returns a likely installer URL and filename.
func findRevHardwareClientURL() (string, string, error) {
	page := "https://docs.revrobotics.com/rev-hardware-client/gs/install"
	resp, err := http.Get(page)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	// Heuristic: look for links ending with common installer extensions
	re := regexp.MustCompile(`https?://[\w\-./]+\.(exe|msi|dmg|zip|tar.gz)`)
	matches := re.FindAllString(string(body), -1)
	if len(matches) == 0 {
		return "", "", errors.New("no installer links found on REV install page")
	}

	// Prefer .exe or .msi for Windows
	for _, m := range matches {
		lower := strings.ToLower(m)
		if strings.HasSuffix(lower, ".msi") || strings.HasSuffix(lower, ".exe") {
			u, _ := url.Parse(m)
			return m, path.Base(u.Path), nil
		}
	}

	// fallback to first match
	u, _ := url.Parse(matches[0])
	return matches[0], path.Base(u.Path), nil
}

// download-studio: fetch the latest Android Studio installer for the current OS
var downloadStudioCmd = &cobra.Command{
	Use:   "download-studio",
	Short: "Download the latest Android Studio installer for your OS",
	Run: func(cmd *cobra.Command, args []string) {
		out, _ := cmd.Flags().GetString("out")

		platform := detectStudioPlatform()
		if platform == "" {
			fmt.Println("Unsupported OS for automatic Android Studio download")
			return
		}

		fmt.Printf("Looking up latest Android Studio download for %s...\n", platform)
		downloadURL, err := findLatestAndroidStudioURL(platform)
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
			// default to current dir with filename from URL
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

		f, err := os.Create(outPath)
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
	downloadStudioCmd.Flags().StringP("out", "o", "", "Output path for the downloaded installer")
}

// detectStudioPlatform returns the platform string used on developer.android.com
func detectStudioPlatform() string {
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

// findLatestAndroidStudioURL does a simple fetch of the Android Studio page and tries to extract a download URL for the platform.
func findLatestAndroidStudioURL(platform string) (string, error) {
	resp, err := http.Get("https://developer.android.com/studio")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Look for URLs that point to archives/installer files. This is a heuristic and may need updates.
	// Examples: https://redirector.gvt1.com/edgedl/android/studio/install/2023.1.1.15/android-studio-2023.1.1.15-windows.msi
	re := regexp.MustCompile(`https?://[\w\-./]+android-studio[\w\-.]*(?:windows|mac|mac-arm|linux)[\w\-./]*\.(exe|msi|dmg|tar\.gz)`)
	matches := re.FindAllString(string(body), -1)
	if len(matches) == 0 {
		// fallback: broader match for studio installer urls
		re2 := regexp.MustCompile(`https?://[\w\-./]+android/studio[\w\-./]+\.(exe|msi|dmg|tar.gz)`)
		matches = re2.FindAllString(string(body), -1)
	}

	if len(matches) == 0 {
		return "", errors.New("no download URLs found on the Android Studio page")
	}

	// Prefer a match containing the platform word.
	// On Windows prefer .msi installers when present, otherwise fallback to exe/zip.
	if platform == "windows" {
		// Look for MSI first
		for _, m := range matches {
			lower := strings.ToLower(m)
			if strings.HasSuffix(lower, ".msi") {
				return m, nil
			}
		}
		// Then prefer exe/zip or containing 'windows'
		for _, m := range matches {
			lower := strings.ToLower(m)
			if strings.Contains(lower, "windows") || strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".zip") {
				return m, nil
			}
		}
	} else {
		for _, m := range matches {
			lower := strings.ToLower(m)
			if platform == "mac" && (strings.Contains(lower, "mac") || strings.Contains(lower, "dmg")) {
				return m, nil
			}
			if platform == "linux" && (strings.Contains(lower, "linux") || strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".zip")) {
				return m, nil
			}
		}
	}

	// If no platform-specific match, return the first match
	return matches[0], nil
}
