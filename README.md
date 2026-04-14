# specd

A local spec, task, and knowledge base management tool for any project or workspace, designed to be driven by AI agents via the AGENTS.md standard.

See [AGENTS.md](AGENTS.md) for the full specification.

## Prerequisites

- [Go](https://go.dev/dl/) 1.26+
- [just](https://github.com/casey/just) (`brew install just`)

## Quick Start

```sh
just build                # build the binary
./specd init              # initialize workspace in current directory
./specd new-spec --title "My Feature" --type technical --summary "Feature design"
./specd list specs
```

## Commands

```sh
just build                # build CLI binary
just test                 # run all tests
just clean                # remove build artifacts
```

## CLI Usage

```
specd init [path]           Initialize a new workspace
specd config <key> [value]  Get or set config (e.g. user.name)
specd new-spec              Create a spec (--title, --type, --summary, --body)
specd new-task              Create a task (--spec-id, --title, --summary, --body)
specd read <id>             Read a spec or task (--with-tasks, --with-criteria)
specd list specs            List specs (--type, --limit)
specd list tasks            List tasks (--spec-id, --status, --created-by, --limit)
```

All commands support `--json` for machine-readable output.

## Project Structure

```
cmd/specd/              CLI entry point
internal/
  cli/                  Cobra command definitions
  db/                   SQLite schema, migrations, meta
  frontmatter/          YAML frontmatter parser
  hash/                 Content hashing (SHA-256)
  lock/                 flock-based single-writer enforcement
  workspace/            Domain logic (init, specs, tasks, criteria)
justfile                Task runner
web/                    Astro frontend (future)
```
