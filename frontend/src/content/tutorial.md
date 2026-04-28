Welcome to **specd** — a spec-driven development tool that keeps your
requirements, tasks, and reference docs side-by-side with your code. This
tutorial walks you from an empty directory to a fully populated workspace
with specs, tasks, a kanban board, and search.

By the end, you will have:

- Initialized a specd workspace.
- Created your first spec.
- Broken that spec down into tasks.
- Browsed the web UI: the kanban, search, and settings.

## Prerequisites

- A recent build of the `specd` binary on your `PATH`.
- A working directory you control (specd writes markdown files in place).
- A modern browser for the web UI.

You can verify the install by running:

```sh
specd --version
```

## Step 1 — Initialize a workspace

From the directory you want to track, run:

```sh
specd init
```

This creates a `.specd/` directory containing the project database and
a `specd.toml` config. The config controls the spec types and task
stages that show up in the UI; the defaults are sensible, but you can
edit them later.

> **Note:** `specd init` is idempotent. Running it a second time does
> not clobber your config — it only fills in anything missing.

After init, your directory layout looks like:

```
my-project/
├── .specd/
│   └── specd.db
├── specd.toml
└── specs/
```

The `specs/` directory is where every spec — and the tasks under it —
will live as plain markdown files. Commit them to git like any other
source.

## Step 2 — Create your first spec

A *spec* describes a piece of behavior the system must, should, or will
have. Specs are markdown files with a small frontmatter header.

Create one with:

```sh
specd new-spec --type functional --title "Login via email link"
```

The CLI opens your `$EDITOR` with a starter template. Save and close,
and specd will:

1. Assign the next ID (e.g. `SPEC-1`).
2. Place the file at `specs/SPEC-1-login-via-email-link/SPEC-1.md`.
3. Index it in the SQLite database for the UI.

A minimal spec looks like:

```markdown
---
id: SPEC-1
type: functional
title: Login via email link
---

## Summary

Users can request a one-time login link sent to their email address.

## Claims

- The system **must** rate-limit login-link requests to one per minute.
- The system **should** expire links after 15 minutes.
- The link **is** single-use.
```

Note the **must / should / is / will** language — these are the words
specd recognizes as acceptance claims. Avoid weaker words like *may*
or *might* in claims; they tend to hide ambiguity.

## Step 3 — Break the spec into tasks

Specs describe *what*. Tasks describe *what to do next*. Create a
task under `SPEC-1` with:

```sh
specd new-task --spec SPEC-1 --title "Send login email"
```

The new task lands at `specs/SPEC-1-login-via-email-link/TASK-1.md`,
right next to its parent spec. A task file looks like:

```markdown
---
id: TASK-1
spec_id: SPEC-1
status: todo
title: Send login email
---

## Summary

Wire up the SMTP path that sends the one-time login link.

## Acceptance criteria

- [ ] SMTP credentials are read from the environment.
- [ ] A successful send is logged with a request ID.
- [ ] Failures bubble up as a 5xx response.
```

Each unchecked box is a criterion the kanban (and the CLI) tracks.
Tick them in the file, the database stays in sync; tick them in the
UI, the file stays in sync.

> **Note:** Tasks always live alongside their parent spec on disk.
> There is no separate `tasks/` directory — that keeps related work
> visible together when you `git diff` a feature branch.

## Step 4 — Open the web UI

Start the local server with:

```sh
specd serve
```

Open <http://localhost:3000> in your browser. By default you land on
the welcome page; the sidebar links you to:

- **Specs** — grouped by type, filterable, with full detail pages.
- **Tasks** — a list view and a kanban board.
- **Knowledge base** — reference docs you have ingested.
- **Search** — a single search bar across everything.
- **Settings** — pick which page is the default home.
- **Docs** — this tutorial and any other in-app guides.

### The kanban

The board groups tasks by `status`. Drag a card across columns to
move it; specd updates the database **and** rewrites the underlying
markdown file so your git history captures the change.

### Search

Search uses a hybrid BM25 + trigram index. Type a query and results
are grouped by kind (spec, task, KB doc). Click any hit to jump to
its detail page.

### Settings

The Settings page lets you change the *default route* — the page that
loads when you open the root URL. Pick from the configured choices
and click **Save**. A toast confirms the change, and the next visit
to `/` will redirect to the new page.

## Where to go next

- Run `specd --help` to see every subcommand the CLI exposes.
- Read the rest of the docs from the sidebar **Docs** entry.
- Open `specd.toml` to tweak spec types and task stages for your
  team's workflow.

Happy specing.
