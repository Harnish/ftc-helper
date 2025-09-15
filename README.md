# FTC Helper

FTC Helper is a command-line interface (CLI) tool designed to streamline the development process for FIRST Tech Challenge (FTC) robotics projects. It provides a set of commands to automate common tasks such as creating new projects, managing versions, and interacting with Git repositories.

## Features

- **List available FTC releases**: View a list of all available FTC Robot Controller versions from the official GitHub repository.
- **Initialize a new project**: Quickly set up a new FTC project based on a specific release version.
- **Launch projects**: Open your FTC projects in Android Studio with a single command.
- **Git integration**: Easily pull, commit, and push code to your Git repositories.
- **Project management**: List all your local FTC projects.

## Installation

1.  **Download the executable**: Grab the latest version of `ftc-helper.exe` from the [releases page](https://github.com/your-username/ftc-helper-gemini/releases).
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

### Configuration

FTC Helper uses a configuration file located at `$HOME/.ftc-helper.yaml` to store settings. The following settings are available:

-   `work_dir`: The working directory where your FTC projects are stored.

You can also specify the working directory on the command line using the `--work-dir` or `-w` flag.

## Contributing

Contributions are welcome! If you have any ideas, suggestions, or bug reports, please open an issue on the [GitHub repository](https://github.com/your-username/ftc-helper-gemini/issues).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
