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
./specd new-task --spec-id SPEC-1 --title "Implement it" --summary "Build the feature"
./specd list specs
./specd serve             # start web UI on :7823
```

## Build Commands

```sh
just build                # build CSS (Vite) + CLI binary
just test                 # run all tests (verbose, with count)
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

All commands support `--json` for machine-readable output.

### Initialization and Configuration

```
specd init [path]                         Initialize a new workspace
specd config <key> [value]                Get or set config (e.g. user.name)
```

### Specs

```
specd new-spec                            Create a spec
    --title, --type, --summary, --body
    --link SPEC-N, --cite KB-N:pos, --dry-run
specd read SPEC-N                         Read a spec
    --with-tasks, --with-links, --with-progress, --with-citations
specd list specs                          List specs
    --type, --linked-to, --empty, --limit
specd update SPEC-N                       Update a spec
    --title, --type, --summary, --body
specd rename SPEC-N --title "New Title"   Rename (updates slug + folder)
specd delete SPEC-N                       Soft-delete to trash
specd reorder spec SPEC-N                 Reposition
    --before SPEC-M, --after SPEC-M, --to N
```

### Tasks

```
specd new-task                            Create a task
    --spec-id, --title, --summary, --body, --status
    --link TASK-N, --depends-on TASK-N, --cite KB-N:pos, --dry-run
specd read TASK-N                         Read a task
    --with-criteria, --with-links, --with-deps, --with-citations
specd list tasks                          List tasks
    --spec-id, --status, --linked-to, --depends-on, --created-by, --limit
specd move TASK-N --status <status>       Change task status
specd update TASK-N                       Update a task
    --title, --summary, --body
specd rename TASK-N --title "New Title"   Rename (updates slug + filename)
specd delete TASK-N                       Soft-delete to trash
specd reorder task TASK-N                 Reposition within status column
    --before TASK-M, --after TASK-M, --to N
```

Task statuses: `backlog`, `todo`, `in_progress`, `blocked`, `pending_verification`, `done`, `cancelled`, `wontfix`.

### Acceptance Criteria

```
specd criteria list TASK-N                List criteria
specd criteria add TASK-N "text"          Add a criterion
specd criteria check TASK-N <position>    Mark as done
specd criteria uncheck TASK-N <position>  Mark as undone
specd criteria remove TASK-N <position>   Remove a criterion
```

### Links and Dependencies

```
specd link <from-id> <to-id>...           Link specs or tasks
specd unlink <from-id> <to-id>...         Remove links
specd depend <task-id> --on <task-id>...  Declare dependencies
specd undepend <task-id> --on <task-id>...
specd candidates <id>                     Find link/citation candidates
```

### Knowledge Base

```
specd kb add <path-or-url>                Add a document (md, txt, html, pdf)
    --title, --note
specd kb list                             List KB documents
    --source-type
specd kb read <kb-id>                     Read a document with chunks
    --chunk N
specd kb search "<query>"                 Search KB chunks
    --limit
specd kb remove <kb-id>                   Soft-delete to trash
specd kb connections <kb-id>              Show TF-IDF chunk connections
    --chunk N, --limit
specd kb rebuild-connections              Recompute connection graph
    --threshold, --top-k
```

### Citations

```
specd cite <id> KB-N:position...          Cite KB chunks from a spec or task
specd uncite <id> KB-N:position...        Remove citations
```

### Search

```
specd search "<query>"                    Hybrid BM25 + trigram search
    --kind (spec|task|kb|all), --limit
```

## Key Dependencies

- `modernc.org/sqlite` — pure-Go SQLite (no CGO)
- `spf13/cobra` — CLI framework
- `gofrs/flock` — cross-platform file locking
- `microcosm-cc/bluemonday` — HTML sanitization
- `golang.org/x/net/html` — HTML parsing
- `ledongthuc/pdf` — pure-Go PDF text extraction

## Project Structure

```
cmd/specd/              CLI entry point
internal/
  cli/                  Cobra command definitions
  db/                   SQLite schema, migrations, meta, FTS5/trigram triggers
  frontmatter/          YAML frontmatter parser (spec + task schemas)
  hash/                 Content hashing (SHA-256)
  lock/                 flock-based single-writer enforcement
  web/                  HTTP server, Go template rendering, htmx support
  workspace/            Domain logic
    workspace.go          Workspace open/close/lock
    init.go               Workspace initialization
    spec.go               Spec CRUD + rename + delete
    task.go               Task CRUD + move + rename + delete
    kb.go                 Knowledge base add/list/read/remove/search/connections
    chunk.go              Paragraph-aware text chunking (md, txt, html, pdf)
    tfidf.go              TF-IDF cosine similarity for chunk connections
    cite.go               Citation operations + frontmatter sync
    link.go               Undirected spec/task linking + frontmatter sync
    depend.go             Directed task dependencies + cycle detection
    criteria.go           Acceptance criteria CRUD + markdown round-trip
    candidates.go         Link/citation candidate scoring
    search.go             Hybrid BM25 + trigram search
    reorder.go            Position reordering (before/after/to)
    readopts.go           Read enrichment (links, progress, deps)
    slug.go               URL-safe slug generation
templates/              Go HTML templates (layouts, partials, pages)
assets/                 Vendored JS (htmx, BeerCSS runtime, app.js)
styles/                 CSS build pipeline (Vite + LightningCSS + PurgeCSS)
  src/styles/           Custom CSS framework (tokens, utilities, reset)
  public/vendor/        BeerCSS, Material Symbols fonts
  scripts/              Utility/color generators
  vite.config.js        Standalone CSS build config
embed.go                embed.FS directives for templates, dist, assets
```

## Web UI Stack

- **Go `html/template`** for server-rendered HTML
- **htmx** for partial page updates (no JS framework)
- **BeerCSS** (Material Design 3) + custom token-based CSS framework
- **Vite** (CSS-only build): LightningCSS for `@custom-media`, PurgeCSS for tree-shaking
- All assets embedded in the binary — zero network requests at runtime
