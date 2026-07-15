# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`ftc-helper` is a single-binary Go CLI (Cobra + Viper) that automates setup for FIRST Tech Challenge robotics teams: scaffolding a new FTC project from an official release zip, launching it in Android Studio, git pull/push shortcuts scoped to the project's `TeamCode` dir, and scraper-based downloaders for Android Studio / Git for Windows / REV Hardware Client / BambuLab Studio installers.

## Commands

```bash
go build -o ftc-helper .        # build the binary
go test ./...                   # run all tests
go test -run TestParseGitVersion ./...   # run a single test
go vet ./...
go mod tidy                      # sync go.mod/go.sum after adding deps
```

Manual smoke test (no test harness drives the CLI end-to-end):
```bash
./ftc-helper list
./ftc-helper init v11.0 -p some-project -g git@github.com/User/Repo
./ftc-helper config
```

## Architecture

Flat `package main`, no internal packages — everything lives in root `.go` files:

- `main.go` — all Cobra command definitions and their `Run` functions live here in one file (list, init, launch, pull, push, projects, config, download-studio, download-git, download-rev, download-all), plus shared helpers (`extractZip`, `downloadURLToPath`, `runBinary`, `findAndroidStudioExe`). `rootCmd.PersistentPreRun` loads Viper config (`$HOME/.ftc-helper.yaml`, overridable via `--config`) and resolves `workDir` before every command runs.
- `bambulab.go` — `download-bambu` command and its page-scraping logic, split out separately from main.go's other download commands.
- `version.go` — `version` command; resolves version by priority: build-time ldflags `-X main.Version=...` → repo-root `VERSION` file → `git describe --tags` → short commit hash.
- `gitversion.go` / `androidversion.go` — standalone version-detection helpers (parse `git --version` output, parse Android Studio's `product-info.json`) used by the download/launch commands, not wired to a cobra command directly.

**Project layout convention**: every FTC project managed by this tool has code under `<project>/TeamCode/src/main/java/org/firstinspires/ftc/teamcode`. `pull`, `push`, and `projects` all hardcode this path and run git commands with `cmd.Dir` set there — an FTC project without this exact subpath is invisible to those commands.

**Downloader commands are HTML/API scrapers, not stable clients**: `download-studio`, `download-rev`, and `download-bambu` regex-scrape live vendor pages (`developer.android.com/studio`, REV docs, bambulab.com) for installer links. These are expected to break when vendor pages change structure — that's a scraper heuristic issue, not necessarily a logic bug. `download-git` uses the GitHub Releases API instead (subject to unauthenticated rate limits).

**VERSION file automation**: `.githooks/pre-push` updates the root `VERSION` file and commits it whenever a `v*` tag is pushed, then pushes that commit too. Enable via `pwsh -File .\scripts\install-githooks.ps1` (sets `core.hooksPath` to `.githooks`). `.githooks/post-commit` regenerates `CHANGELOG.md` via `scripts/update-changelog.ps1` on every commit. These hooks call PowerShell scripts (`scripts/*.ps1`) even from the bash hook wrappers — expect a `pwsh` dependency on non-Windows dev machines.

**Windows-first, cross-compiled**: distributed as a Windows-focused tool (see `.exe` naming, `ANDROID_STUDIO_PATH`/registry-style path probing in `findAndroidStudioExe`), but `go.yml` builds/tests on ubuntu/windows/macos and `release.yaml` cross-compiles linux/windows/darwin (amd64+arm64) binaries on tag push (`v*`), attaching them to a GitHub Release via `softprops/action-gh-release`.

## CI

- `.github/workflows/go.yml` — build matrix (ubuntu/windows/macos) on push/PR to `main`; only uploads binaries to a release if the head commit message starts with `Release` and branch is `main`.
- `.github/workflows/release.yaml` — on `v*` tag push: runs `go test ./...`, cross-compiles 5 platform binaries, creates a GitHub Release (marks prerelease if the tag contains `-`, e.g. `v1.0.0-rc1`).
