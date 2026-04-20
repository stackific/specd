---
name: specd-init
description: Initialize specd in a project. Use when the user wants to set up specd in a new or existing project directory.
metadata:
  author: stackific
  version: "1.0"
---

# specd init

Initialize specd in the current project by creating a folder where specd stores its project-level files and configuring the username.

## When to use this skill

Use this skill when the user wants to:
- Set up specd in a new or existing project
- Initialize specd project files
- Configure their specd username

## Usage

```sh
specd init $ARGUMENTS
```

## Arguments

- `[project-path]` — path to the project directory (default: current directory)
- `--folder <name>` — folder name for specd's project files (default: "specd")
- `--username <name>` — your username (optional if already configured globally)

## Examples

```sh
# Interactive (prompts for folder name and username)
specd init

# Initialize in a specific project directory
specd init /path/to/project

# Fully non-interactive
specd init --folder specd --username an-unique-name-in-the-team

# Initialize in another directory with a custom folder name
specd init /path/to/project --folder specs
```

## Behavior

If the user provides arguments, pass them directly. If no arguments are given, run `specd init` and it will prompt interactively. The command detects the git username as a default if no username is configured.
