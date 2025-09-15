# FTC Helper (for Gemini)

This document provides a high-level overview of the FTC Helper project, its purpose, and how to get started using it.

## What is FTC Helper?

FTC Helper is a command-line tool that simplifies the development process for FIRST Tech Challenge (FTC) robotics teams. It automates many of the common tasks involved in setting up and managing FTC projects, allowing teams to focus on what really matters: designing, building, and programming their robots.

With FTC Helper, you can:

-   Quickly create new projects based on official FTC releases.
-   Easily manage different versions of the FTC Robot Controller SDK.
-   Seamlessly integrate your projects with Git for version control.
-   Launch your projects in Android Studio with a single command.

Our goal is to make the FTC development experience as smooth and efficient as possible, so you can spend less time on tedious setup and more time innovating.

## Getting Started

Ready to give it a try? Here's how to get started with FTC Helper:

1.  **Download the tool**: Head over to our [releases page](https://github.com/your-username/ftc-helper-gemini/releases) and download the latest version of `ftc-helper.exe`.

2.  **Install it on your system**: For easy access, place the `ftc-helper.exe` file in a directory that is part of your system's PATH.

3.  **Explore the commands**: Open up your terminal or command prompt and type `ftc-helper --help` to see a list of all the available commands and what they do.

4.  **Create your first project**: Use the `init` command to create a new FTC project. You'll need to specify the FTC Robot Controller version you want to use and a name for your project.

    ```bash
    ftc-helper init v8.2 --project my-awesome-robot
    ```

5.  **Start coding!**: Launch your new project in Android Studio using the `launch` command and start bringing your robot to life.

    ```bash
    ftc-helper launch my-awesome-robot
    ```

We hope you find FTC Helper to be a valuable addition to your FTC toolkit. If you have any questions, feedback, or suggestions, please don't hesitate to reach out to us on our [GitHub repository](https://github.com/your-username/ftc-helper-gemini/issues).
