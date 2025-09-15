package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	workDir string
)

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
		viper.SetDefault("work_dir", home)
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
		if _, err := os.Stat(subDir); err == nil {
			files, _ := ioutil.ReadDir(subDir)
			for _, f := range files {
				os.Rename(filepath.Join(subDir, f.Name()), filepath.Join(projectPath, f.Name()))
			}
			os.Remove(subDir)
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
