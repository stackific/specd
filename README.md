# specd

A local spec, task, and knowledge base management tool for any project or workspace, designed to be driven by AI agents via the AGENTS.md standard.

See [AGENTS.md](AGENTS.md) for the full specification.

## Prerequisites

- [Go](https://go.dev/dl/) 1.26+
- [Node.js](https://nodejs.org/) 22+ and [pnpm](https://pnpm.io/) (for CSS build)
- [just](https://github.com/casey/just) (`brew install just`)
- [air](https://github.com/air-verse/air) (dev only, `go install github.com/air-verse/air@latest`)

## Quick Start

```sh
just build                # build CSS + Go binary
./specd init              # initialize workspace in current directory
./specd new-spec --title "My Feature" --type technical --summary "Feature design"
./specd list specs
./specd serve             # start web UI on :7823
```

## Build Commands

```sh
just build                # build CSS (Vite) + CLI binary
just test                 # run all tests
just web                  # build CSS only (cd styles && pnpm build)
just dev                  # dev mode: Vite CSS watch + Air Go hot-reload
just clean                # remove build artifacts
```

## Installing Web Dependencies

The CSS build requires Node.js dependencies in the `styles/` directory:

```sh
cd styles && pnpm install
```

This is done automatically by `just build` and `just web`. You only need to run it manually if you want to use the generators (`pnpm gen:utilities`, `pnpm gen:colors`).

## CLI Usage

```
specd init [path]                   Initialize a new workspace
specd config <key> [value]          Get or set config (e.g. user.name)
specd new-spec                      Create a spec (--title, --type, --summary, --body)
specd new-task                      Create a task (--spec-id, --title, --summary, --body)
specd read <id>                     Read a spec or task (--with-tasks, --with-criteria)
specd list specs                    List specs (--type, --limit)
specd list tasks                    List tasks (--spec-id, --status, --created-by, --limit)
specd search "<query>"              Hybrid BM25 + trigram search (--kind, --limit)
specd link <from-id> <to-id>...     Link specs or tasks together
specd unlink <from-id> <to-id>...   Remove links
specd depend <task-id> --on <id>    Declare task dependencies
specd undepend <task-id> --on <id>  Remove dependencies
specd candidates <id>               Find link candidates
specd criteria list <task-id>       List acceptance criteria
specd criteria add <task-id> <text> Add a criterion
specd criteria check <task-id> <n>  Mark criterion as done
specd criteria uncheck <task-id> <n>
specd criteria remove <task-id> <n>
specd serve                         Start web UI (--port, default 7823)
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
  web/                  HTTP server, Go template rendering, htmx support
  workspace/            Domain logic (init, specs, tasks, criteria, search)
templates/              Go HTML templates (layouts, partials, pages)
assets/                 Vendored JS (htmx, BeerCSS runtime, app.js)
styles/                 CSS build pipeline (Vite + LightningCSS + PurgeCSS)
  src/styles/           Custom CSS framework (tokens, utilities, reset)
  public/vendor/        BeerCSS, Material Symbols fonts
  scripts/              Utility/color generators
  vite.config.js        Standalone CSS build config
ref/                    Reference Astro project (template)
embed.go                embed.FS directives for templates, dist, assets
```

## Web UI Stack

- **Go `html/template`** for server-rendered HTML
- **htmx** for partial page updates (no JS framework)
- **BeerCSS** (Material Design 3) + custom token-based CSS framework
- **Vite** (CSS-only build): LightningCSS for `@custom-media`, PurgeCSS for tree-shaking
- All assets embedded in the binary — zero network requests at runtime
