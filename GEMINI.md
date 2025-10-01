# FTC Helper (for Gemini)

This document provides a high-level overview of the FTC Helper project, its purpose, and how to get started using it.

## What is FTC Helper?

FTC Helper is a command-line tool that simplifies the development process for FIRST Tech Challenge (FTC) robotics teams and general development environment setup. It automates many of the common tasks involved in setting up and managing FTC projects, as well as downloading common development tools.

With FTC Helper, you can:

-   Quickly create new projects based on official FTC releases.
-   Easily manage different versions of the FTC Robot Controller SDK.
-   Seamlessly integrate your projects with Git for version control.
-   Launch your projects in Android Studio with a single command.
-   Download and install essential development tools like Git, Android Studio, REV Hardware Client, and Bambu Studio.
-   Check the version of the tool.

Our goal is to make the FTC and general development experience as smooth and efficient as possible, so you can spend less time on tedious setup and more time innovating.

## Getting Started

Ready to give it a try? Here's how to get started with FTC Helper:

1.  **Download the tool**: Head over to our [releases page](https://github.com/Harnish/ftc-helper/releases) and download the latest version of `ftc-helper.exe`.

2.  **Install it on your system**: For easy access, place the `ftc-helper.exe` file in a directory that is part of your system's PATH.

3.  **Explore the commands**: Open up your terminal or command prompt and type `ftc-helper --help` to see a list of all the available commands and what they do.

## Commands

Here is a list of the available commands and what they do:

-   `config`: Opens the configuration file in the default editor.
-   `download-all`: Downloads all the recommended software.
-   `download-bambu`: Downloads the latest version of Bambu Studio.
-   `download-git`: Downloads the latest version of Git.
-   `download-rev`: Downloads the latest version of the REV Hardware Client.
-   `download-studio`: Downloads the latest version of Android Studio.
-   `init`: Creates a new FTC project.
-   `launch`: Launches an FTC project in Android Studio.
-   `list`: Lists the available FTC Robot Controller SDK versions.
-   `projects`: Lists all the FTC projects in the current directory.
-   `pull`: Pulls the latest changes from the remote repository for an FTC project.
-   `push`: Pushes the latest changes to the remote repository for an FTC project.
-   `version`: Prints the version of the tool.

We hope you find FTC Helper to be a valuable addition to your FTC toolkit. If you have any questions, feedback, or suggestions, please don't hesitate to reach out to us on our [GitHub repository](https://github.com/Harnish/ftc-helper/issues).

