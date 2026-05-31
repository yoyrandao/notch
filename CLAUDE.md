# autotag

This file provides short guidance about the project creating. After some significant changes performed in project, Claude Code can edit this file to add some information.

## Termins

tool-specific project - is a repository or folder containing files of some specific project structure (like Helm chart, npm package, dotnet project, rust library, etc).

## Project

`autotag` - lightweight Golang CLI for automated semantic versioning of specific projects. It reads the history of conventional commits, computes next semantic version, writes changes to CHANGELOG.md, creates annotated git tag, pushes it and can run some automations specified by user via bash scripts. It automatically catches if the project has tool-specific project structure and alongside CHANGELOG.md it also patches the files containing version information.

The tool itself should help Dev and DevOps teams to get rid of manual versioning and maintainability issues and provide a functionality that performs versioining automatically.

Project packages as a single static binary, with minimal runtime dependencies except `git` on PATH.

## Commands

```sh
# build
go build ./...

# test
go test ./...
```

## Testing

CLI should be fully tested not only by unit tests, but also by end-to-end tests via creating temporary git repositories and running `autotag` CLI on them. Also it needs to cover tool-specific project repositories cases like patching files containing version information.

## Features

While performing each feature I requested, split it into stages and do it each stage.Each completed feature should bump the version of that CLI itself according what functionality was added (feature, breaking change or small fix).

## Dependencies

- `spf13/cobra` - CLI framework
- `knadh/koanf` - Config loading