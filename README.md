# FTC Helper

FTC Helper is a command-line interface (CLI) tool designed to streamline the development process for FIRST Tech Challenge (FTC) robotics projects. It provides a set of commands to automate common tasks such as creating new projects, managing versions, and interacting with Git repositories.

## Quickstat guide 
[QUICKSTART.md](QUICKSTART.md)

## Features

- **List available FTC releases**: View a list of all available FTC Robot Controller versions from the official GitHub repository.
- **Initialize a new project**: Quickly set up a new FTC project based on a specific release version.
- **Launch projects**: Open your FTC projects in Android Studio with a single command.
- **Git integration**: Easily pull, commit, and push code to your Git repositories.
- **Project management**: List all your local FTC projects.

## Installation

1.  **Download the executable**: Grab the latest version of `ftc-helper.exe` from the [releases page](https://github.com/Harnish/ftc-helper/releases).
2.  **Place it in your PATH**: To use it from anywhere on your system, place the executable in a directory that is included in your system's PATH environment variable.

## Usage

### Commands

#### `list`

Lists all available FTC Robot Controller releases.

```bash
ftc-helper list
```

#### `init [version]`

Initializes a new FTC project with the specified version.

```bash
ftc-helper init <version> --project <project-name> --git <git-repository-url>
```

-   `<version>`: The FTC Robot Controller version to use (e.g., `v8.2`).
-   `--project <project-name>`: The name of the new project directory.
-   `--git <git-repository-url>`: (Optional) The URL of the Git repository to set up as a remote.

#### `launch [project_name]`

Launches a project in Android Studio.

```bash
ftc-helper launch <project-name>
```

#### `pull [project_name]`

Pulls the latest code from the Git repository into the project's `TeamCode` directory.

```bash
ftc-helper pull <project-name>
```

#### `push [project_name] [commit_message]`

Commits and pushes code changes to the remote Git repository.

```bash
ftc-helper push <project-name> "<commit-message>"
```

#### `projects`

Lists all active local projects.

```bash
ftc-helper projects
```

#### `download-studio`

Downloads the latest Android Studio installer for your OS. The command attempts to locate the correct installer for your platform and saves it to the current directory unless you provide `--out`.

```bash
ftc-helper download-studio
ftc-helper download-studio --out C:\Downloads\android-studio-installer.exe
```

Notes:
- On Windows the tool prefers `.msi` installers when available; fallbacks include `.exe` or `.zip`.
- You can override the Android Studio path used by `launch` with the `ANDROID_STUDIO_PATH` environment variable or by setting `android_studio_path` in `$HOME/.ftc-helper.yaml`.

#### `download-git`

Downloads the latest Git for Windows installer (prefers 64-bit) by querying the Git for Windows releases on GitHub. Use `--out` to control the output filename.

```bash
ftc-helper download-git
ftc-helper download-git --out C:\Downloads\Git-2.51.0-64-bit.exe
```

Notes:
- This command uses the GitHub Releases API and may be subject to rate limits for unauthenticated requests.
- If you need a specific variant, download directly from the Git for Windows releases page and use the `--out` flag to save it via the tool.

#### `download-rev`

Downloads the REV Hardware Client installer referenced from the REV docs install page. By default the file is saved with the filename taken from the link; use `--out` to change the destination.

```powershell
ftc-helper download-rev
ftc-helper download-rev --out C:\Downloads\REVHardwareClientInstaller.exe
```

#### `download-bambu`

Download the latest BambuLab Studio installer for your OS. This scrapes the BambuLab download page and attempts to pick an appropriate installer for Windows (.exe/.msi), macOS (.dmg/.pkg), or Linux (.AppImage/.deb/.tar.gz).

```powershell
ftc-helper download-bambu
ftc-helper download-bambu --out C:\Downloads\BambuStudioInstaller.exe
```

#### `download-all`

Downloads several tools (Git for Windows, REV Hardware Client, Android Studio) in sequence and offers to run each installer. Intended as a convenience for provisioning a Windows workstation.

```powershell
ftc-helper download-all
```


Notes:
- The command scrapes the REV docs page for links to installers. If REV changes the page structure it may need an update.

#### `config`

Prints the current runtime configuration (Viper settings) as YAML to stdout. This includes defaults, config file values (if loaded), environment variables, and flags bound to Viper.

```powershell
ftc-helper config > current-config.yaml
```


### Configuration

FTC Helper uses a configuration file located at `$HOME/.ftc-helper.yaml` to store settings. The following settings are available:

-   `work_dir`: The working directory where your FTC projects are stored.

You can also specify the working directory on the command line using the `--work-dir` or `-w` flag.

## Contributing

Contributions are welcome! If you have any ideas, suggestions, or bug reports, please open an issue on the [GitHub repository](https://github.com/Harnish/ftc-helper/issues).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Version command and automatic VERSION updates

This repository provides a `version` CLI command and a git hook to keep a `VERSION` file
in-sync with pushed tags.

- Run the CLI command to print the version resolved from build flags, a `VERSION` file, or the latest tag:
	- `ftc-helper version`
- To enable automatic updates when pushing tags, enable the repository hooks:
	- PowerShell: `pwsh -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-githooks.ps1`
	- This will set `core.hooksPath` to `.githooks` so the `pre-push` hook runs and updates `VERSION` when tags are pushed.

The hook will create a commit updating `VERSION` and attempt to push that commit before continuing the original push.
