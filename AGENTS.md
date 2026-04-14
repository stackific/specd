# specd — Specification (v3, complete)

A local spec, task, and knowledge base management tool for any project or workspace, designed to be driven by AI agents (Claude Code, Claude Desktop, Codex, Cursor, and others) via the AGENTS.md standard and the Claude Skills system.

## 1. Scope and framing

specd is domain-agnostic. The software-development use case (specs + tasks + reference docs) is primary, but the same tool serves:

- Personal knowledge bases and research notes.
- Obsidian-style vaults managed by an AI agent.
- Product research, competitive analysis, due diligence.
- Long-form reading projects.
- Any use case where structured, incrementally-built knowledge is valuable.

## 2. Goals and non-goals

**Goals**
- Give AI agents a structured place to store, retrieve, link, and query specs, tasks, and reference documents.
- Store content in human-editable, git-friendly markdown where possible. Non-markdown references (PDF, HTML, plain text) are stored as-is and indexed.
- Fast, accurate retrieval via a derived SQLite cache with hybrid BM25 + trigram search.
- Statistical KB chunk-to-chunk connections via TF-IDF cosine similarity, no AI required.
- Minimal CLI driven by AI agents; zero embedded AI inside specd itself.
- Embedded read-write kanban web UI with a full KB reader supporting markdown, plain text, HTML, and PDF rendering with chunk highlighting.
- Ship as a single static Go binary with zero runtime network dependencies. All web UI assets (including PDF.js) embedded.
- Work from both Claude Code (project-scoped via AGENTS.md) and Claude Desktop (user-scoped via Claude Skills).

**Non-goals**
- No embedded AI. specd is a data layer and query tool; all reasoning is supplied by the calling agent.
- No server, multi-user, or remote sync. Per-workspace tool, git is the sync mechanism.
- No semantic/embedding search in v1.
- No time tracking, estimates, assignees, or due dates in v1.
- No automatic fixups. Lint reports; agent or user decides.
- No folder-level per-developer namespacing. One shared view of a project.
- No `adopt` command for importing hand-created files.
- No workspace templates in v1 (considered for v2).

## 3. Target users and usage modes

### Mode 1: inside a code project (Claude Code and similar)

The developer runs `specd init` inside their repo. specd creates `specd/` and `.specd/` directories and writes an AGENTS.md section describing how to use specd. The developer then interacts through slash commands (`/specd-new-spec`, `/specd-move`, `/specd-next`, etc.) or by asking natural questions that cause the agent to reach for specd subcommands based on AGENTS.md guidance.

### Mode 2: standalone workspace (Claude Desktop, mobile, web)

The user runs `specd init ~/my-vault` to create a workspace outside any code project. specd can additionally install a Claude Skill into the user's skills directory pointing at that vault. The user then talks to Claude Desktop about their project, research, or knowledge base, and Claude uses the skill to call specd subcommands.

Both modes use the same binary and the same data layer. The only difference is where the vault lives and which agent configuration is installed.

## 4. Architecture

Three layers:

1. **Markdown vault** (`specd/` under the workspace root). Source of truth for specs and tasks. KB reference docs stored as-is. Committed to git.
2. **SQLite cache** (`.specd/cache.db`). Derived. Holds FTS5 + trigram indexes, link graph, dependency graph, citations, chunk connections, positions, trash, counters. Rebuildable from the vault. Gitignored.
3. **CLI and embedded web UI** (single Go binary). Only writer. All mutations transactional against both markdown and SQLite. A file watcher catches out-of-band edits.

### 4.1 Workspace layout

```
<workspace-root>/
├── AGENTS.md                                    # agent instructions
├── specd/
│   ├── specs/
│   │   ├── index.md                             # auto-maintained catalog
│   │   ├── log.md                               # auto-maintained chronicle
│   │   ├── SPEC-1-user-authentication/
│   │   │   ├── spec.md
│   │   │   ├── TASK-1-design-auth-schema.md
│   │   │   └── TASK-4-implement-jwt.md
│   │   └── SPEC-2-oauth-with-github/
│   │       ├── spec.md
│   │       └── TASK-3-protection-middleware.md
│   └── kb/
│       ├── KB-1-oauth2-rfc6749.pdf
│       ├── KB-1-oauth2-rfc6749.clean.html       # only for HTML source
│       ├── KB-2-github-apps-docs.html
│       ├── KB-2-github-apps-docs.clean.html
│       └── KB-3-jwt-best-practices.md
└── .specd/
    ├── cache.db                                 # SQLite, gitignored
    ├── pdf-cache/                               # PDF.js-independent cache, gitignored
    └── lock                                     # flock lockfile
```

### 4.2 Markdown file format

Minimal frontmatter. Content is freeform.

**Spec** (`specd/specs/SPEC-2-oauth-with-github/spec.md`):

```markdown
---
title: OAuth with GitHub
type: technical
summary: OAuth flow using GitHub as an identity provider
linked_specs: [SPEC-1, SPEC-7]
cites:
  - kb: KB-4
    chunks: [12, 15]
  - kb: KB-9
    chunks: [3]
---

# OAuth with GitHub

Freeform body.
```

**Task** (`specd/specs/SPEC-2-oauth-with-github/TASK-3-protection-middleware.md`):

```markdown
---
title: Protection middleware
status: in_progress
summary: Add auth middleware to routes requiring authentication
linked_tasks: [TASK-2]
depends_on: [TASK-1]
cites:
  - kb: KB-4
    chunks: [12]
---

# Protection middleware

Freeform body.

## Acceptance criteria

- [x] Middleware rejects requests with invalid JWT (401)
- [ ] Middleware rejects requests with no Authorization header (401)
- [ ] Middleware attaches user context on valid JWT
- [ ] Unit tests cover all three cases
```

**Fields not stored in frontmatter**: `id`, `slug`, `spec_id` (derivable from path); `position` (UI state, SQLite only); `created_by`, `updated_by` (SQLite only).

**System-managed frontmatter fields** (written by CLI/watcher, humans should not edit): `linked_specs`, `linked_tasks`, `depends_on`, `cites`.

**User-editable frontmatter fields**: `title`, `type`, `status`, `summary`.

**User-editable body**: entire body including `## Acceptance criteria`. Checking a box (`- [ ]` → `- [x]`) is a valid human edit; the watcher syncs to SQLite.

### 4.3 KB file layout

KB documents live in `specd/kb/` with generated filenames following the pattern `KB-<id>-<slug>.<ext>`. Filenames are assigned by `specd kb add`; users cannot create these files by hand.

For HTML sources, a sanitized sidecar `KB-<id>-<slug>.clean.html` is written at ingest time using bluemonday (`UGCPolicy`). The reader serves the cleaned version. The original HTML is kept for reference but never served to the browser.

KB markdown files may have optional frontmatter (`title`, `note`) but it is not required. PDFs, TXT, and HTML have no frontmatter; their metadata lives in SQLite only.

### 4.4 Rejection of hand-created files

specd is the only path to create specs, tasks, and KB entries. The watcher enforces this:

1. On change, compute the content hash. If it matches the stored hash, skip (CLI's own write).
2. If the file has no matching SQLite row:
   - If its filename matches a canonical pattern (`SPEC-N-slug/spec.md`, `TASK-N-slug.md`, `KB-N-slug.ext`) and the workspace is in a rebuild-pending state, it is ingested.
   - Otherwise, a row is inserted into `rejected_files` and the file is left on disk untouched.
3. `specd status` and the web UI surface rejected files so the user can see what was skipped.

There is no `adopt` command. Users remove the hand-created file and re-add it through the CLI.

## 5. SQLite schema

```sql
-- Schema metadata, counters, timestamps
CREATE TABLE meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
-- Seeded: schema_version, next_spec_id, next_task_id, next_kb_id, last_tidy_at, user_name

-- Specs
CREATE TABLE specs (
  id           TEXT PRIMARY KEY,                -- "SPEC-42"
  slug         TEXT NOT NULL,
  title        TEXT NOT NULL,
  type         TEXT NOT NULL CHECK (type IN ('business','technical','non-technical')),
  summary      TEXT NOT NULL,
  body         TEXT NOT NULL,
  path         TEXT NOT NULL,                   -- relative to workspace root
  position     INTEGER NOT NULL DEFAULT 0,      -- global order for specs list
  created_by   TEXT,                            -- from `specd config user.name`
  updated_by   TEXT,
  content_hash TEXT NOT NULL,
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);

-- Tasks
CREATE TABLE tasks (
  id           TEXT PRIMARY KEY,                -- "TASK-134"
  slug         TEXT NOT NULL,
  spec_id      TEXT NOT NULL REFERENCES specs(id) ON DELETE CASCADE,
  title        TEXT NOT NULL,
  status       TEXT NOT NULL CHECK (status IN (
                 'backlog','todo','in_progress','blocked',
                 'pending_verification','done','cancelled','wontfix')),
  summary      TEXT NOT NULL,
  body         TEXT NOT NULL,
  path         TEXT NOT NULL,
  position     INTEGER NOT NULL DEFAULT 0,      -- per-status kanban order
  created_by   TEXT,
  updated_by   TEXT,
  content_hash TEXT NOT NULL,
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);

-- Acceptance criteria (parsed from "## Acceptance criteria" section)
CREATE TABLE task_criteria (
  task_id    TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  position   INTEGER NOT NULL,                  -- 1-based order in the list
  text       TEXT NOT NULL,
  checked    INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (task_id, position)
);

-- Undirected related links
CREATE TABLE spec_links (
  from_spec TEXT NOT NULL REFERENCES specs(id) ON DELETE CASCADE,
  to_spec   TEXT NOT NULL REFERENCES specs(id) ON DELETE CASCADE,
  PRIMARY KEY (from_spec, to_spec)
);

CREATE TABLE task_links (
  from_task TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  to_task   TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  PRIMARY KEY (from_task, to_task)
);

-- Directed task dependencies (blocker -> blocked)
CREATE TABLE task_dependencies (
  blocker_task TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  blocked_task TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  PRIMARY KEY (blocker_task, blocked_task)
);

-- KB documents
CREATE TABLE kb_docs (
  id           TEXT PRIMARY KEY,                -- "KB-17"
  slug         TEXT NOT NULL,
  title        TEXT NOT NULL,
  source_type  TEXT NOT NULL CHECK (source_type IN ('md','html','pdf','txt')),
  path         TEXT NOT NULL,                   -- original file path
  clean_path   TEXT,                            -- sanitized sidecar path for HTML
  note         TEXT,
  page_count   INTEGER,                         -- for PDFs only
  content_hash TEXT NOT NULL,
  added_at     TEXT NOT NULL,
  added_by     TEXT
);

-- KB chunks
CREATE TABLE kb_chunks (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_id     TEXT NOT NULL REFERENCES kb_docs(id) ON DELETE CASCADE,
  position   INTEGER NOT NULL,                  -- 0-based chunk index within doc
  text       TEXT NOT NULL,
  char_start INTEGER NOT NULL,
  char_end   INTEGER NOT NULL,
  page       INTEGER,                           -- nullable; PDF only
  UNIQUE (doc_id, position)
);

-- Citations (specs and tasks cite KB chunks)
CREATE TABLE citations (
  from_kind      TEXT NOT NULL CHECK (from_kind IN ('spec','task')),
  from_id        TEXT NOT NULL,
  kb_doc_id      TEXT NOT NULL REFERENCES kb_docs(id) ON DELETE CASCADE,
  chunk_position INTEGER NOT NULL,
  created_at     TEXT NOT NULL,
  PRIMARY KEY (from_kind, from_id, kb_doc_id, chunk_position)
);

-- Statistical chunk-to-chunk connections (TF-IDF cosine)
CREATE TABLE chunk_connections (
  from_chunk_id INTEGER NOT NULL REFERENCES kb_chunks(id) ON DELETE CASCADE,
  to_chunk_id   INTEGER NOT NULL REFERENCES kb_chunks(id) ON DELETE CASCADE,
  strength      REAL NOT NULL,                  -- cosine similarity 0..1
  method        TEXT NOT NULL DEFAULT 'tfidf_cosine',
  PRIMARY KEY (from_chunk_id, to_chunk_id)
);

-- Trash (soft delete with recovery)
CREATE TABLE trash (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  kind          TEXT NOT NULL CHECK (kind IN ('spec','task','kb')),
  original_id   TEXT NOT NULL,
  original_path TEXT NOT NULL,
  content       BLOB NOT NULL,
  metadata      TEXT NOT NULL,                  -- JSON snapshot of primary rows
  deleted_at    TEXT NOT NULL,
  deleted_by    TEXT NOT NULL CHECK (deleted_by IN ('cli','watcher'))
);

-- Rejected files (manually created, not registered)
CREATE TABLE rejected_files (
  path        TEXT PRIMARY KEY,
  detected_at TEXT NOT NULL,
  reason      TEXT NOT NULL
);

-- FTS5 with BM25, porter stemming, prefix indexes
CREATE VIRTUAL TABLE specs_fts USING fts5(
  id UNINDEXED, title, type, summary, body,
  content='specs', content_rowid='rowid',
  tokenize='porter unicode61',
  prefix='2 3 4'
);

CREATE VIRTUAL TABLE tasks_fts USING fts5(
  id UNINDEXED, title, status UNINDEXED, summary, body,
  content='tasks', content_rowid='rowid',
  tokenize='porter unicode61',
  prefix='2 3 4'
);

CREATE VIRTUAL TABLE kb_chunks_fts USING fts5(
  text,
  content='kb_chunks', content_rowid='id',
  tokenize='porter unicode61',
  prefix='2 3 4'
);

-- Trigram fallback for fuzzy / substring matching
CREATE VIRTUAL TABLE search_trigram USING fts5(
  kind UNINDEXED,                               -- 'spec' | 'task' | 'kb'
  ref_id UNINDEXED,
  text,
  tokenize='trigram'
);

-- Standard INSERT/UPDATE/DELETE triggers maintain FTS and trigram indexes
-- in sync with base tables. Triggers are defined per base table in migrations.
```

## 6. Retrieval

### 6.1 Hybrid search (specs, tasks, KB)

All free-text search uses a hybrid strategy:

1. **Primary: FTS5 BM25** with porter stemming and prefix indexes. Handles phrase queries, term queries, and prefix queries (`auth*`).
2. **Fallback: trigram index.** When BM25 returns fewer than 3 hits, specd re-runs against `search_trigram` and merges results below BM25 hits. Trigram hits are tagged with `match_type: "trigram"` in responses.
3. **Exact identifier lookup** is a phrase query on FTS5. No separate path.

`specd search`, `specd kb search`, `specd candidates`, and the candidates block of `new-spec`/`new-task` responses all use this hybrid strategy.

### 6.2 TF-IDF chunk-to-chunk connections

On `kb add` (after chunking and FTS5 indexing):

1. For each new chunk, compute a TF-IDF vector against the full KB corpus. Vectors are sparse (only terms present in the chunk), stored in-memory during the pass.
2. Prune candidate neighbors: only compare the new chunk to other chunks that share at least one term with IDF above a configurable threshold. This drastically reduces the O(n²) cost.
3. Compute cosine similarity with each candidate neighbor.
4. Insert rows into `chunk_connections` for pairs where similarity ≥ `threshold` (default 0.3).
5. Cap per-chunk connections at `top-k` (default 10) by strength; lower-scoring edges are dropped.
6. Connections are undirected; two rows are inserted per pair (A→B and B→A) for query simplicity.

`specd kb rebuild-connections [--threshold T] [--top-k K]` recomputes the entire graph. Use after tuning, after bulk-adding docs, or on rebuild.

**Known limitation**: TF-IDF catches lexical similarity. It will not connect chunks that discuss the same concept in different vocabulary. This is acceptable for v1. Optional semantic embeddings are a v2 consideration behind a feature flag.

## 7. CLI commands

All commands support `--json`. Human output is the default when stdout is a TTY. Every mutating command acquires `.specd/lock` (flock, 5s timeout) for the duration of its transaction.

### 7.1 Initialization and configuration

```
specd init [<path>] [--force] [--wire-legacy] [--skill]
specd config user.name "<name>"
specd config user.name                          # read current
specd workspace add <path> --name <nickname>
specd workspace list
specd workspace use <nickname>
specd workspace remove <nickname>
```

`init` creates `specd/specs/`, `specd/kb/`, `.specd/`, `specd/specs/index.md`, `specd/specs/log.md`; writes or appends the specd section in `AGENTS.md`; adds `.specd/` to `.gitignore`; seeds `meta.user_name` from `git config user.name` if available.

`--wire-legacy` detects `CLAUDE.md`, `.cursorrules`, `.clinerules`, `.github/copilot-instructions.md`. For each file found, it displays a diff showing the addition of the single line `See AGENTS.md for spec/task management instructions.` and asks per-file whether to apply. Never touches legacy files without explicit per-file confirmation.

`--skill` installs the Claude Skill folder into the platform-specific Claude skills directory (`~/Library/Application Support/Claude/skills/specd/` on macOS, corresponding paths on Linux and Windows) pointing at the current workspace. See section 12.

`specd config user.name` stores the current developer's display name in SQLite `meta.user_name`. This value populates `created_by` and `updated_by` columns on new specs, tasks, and KB entries. Defaults to `git config user.name` if present; otherwise empty.

`specd workspace` commands manage a per-machine registry at `~/.config/specd/workspaces.json` mapping nicknames to absolute workspace paths. `specd workspace use <nickname>` sets the active workspace for subsequent commands (via an env var exported to the current shell, or a local state file). This is primarily for Claude Desktop users running commands against one of several registered vaults.

### 7.2 Specs

```
specd new-spec --title "<title>" --type <business|technical|non-technical>
               --summary "<one-line>" --body "<markdown body>"
               [--link SPEC-N]... [--cite KB-N:position]... [--dry-run]
```

Allocates next spec ID, generates slug from title, writes `specd/specs/SPEC-N-<slug>/spec.md` and SQLite row in one transaction, updates `index.md`, appends to `log.md`. Returns JSON:

```json
{
  "id": "SPEC-42",
  "path": "specd/specs/SPEC-42-oauth-with-github/spec.md",
  "candidates": {
    "specs": [
      {"id": "SPEC-1", "title": "User authentication", "score": 0.89, "match_type": "bm25"}
    ],
    "tasks": [],
    "kb_chunks": [
      {
        "doc_id": "KB-4",
        "doc_title": "OAuth 2.0 RFC 6749",
        "chunk_position": 12,
        "text": "Authorization code grant is the most commonly used...",
        "score": 0.82,
        "match_type": "bm25"
      }
    ]
  },
  "tidy_reminder": null
}
```

The calling agent reviews all three candidate lists and proposes to the user which specs to link and which KB chunks to cite. On approval the agent calls `specd link SPEC-42 SPEC-1 SPEC-7` and `specd cite SPEC-42 KB-4:12 KB-9:3`.

```
specd read <spec-id> [--with-tasks] [--with-links] [--with-progress] [--with-citations]
specd list specs [--type <type>] [--linked-to SPEC-N] [--limit N] [--empty]
specd update <spec-id> [--title ...] [--type ...] [--summary ...] [--body ...]
specd rename <spec-id> --title "<new title>"    # updates slug + folder name
specd delete <spec-id>                           # soft delete to trash
specd reorder spec <id> --before <id> | --after <id> | --to <position>
```

`--with-citations` returns the cited KB chunks inline (chunk text, doc title, chunk position, page number for PDFs) so the agent has full grounding in one call.

### 7.3 Tasks

```
specd new-task --spec-id SPEC-N --title "<title>" --summary "<one-line>"
               --body "<markdown body>" [--status backlog]
               [--link TASK-N]... [--depends-on TASK-N]...
               [--cite KB-N:position]... [--dry-run]
```

Same pattern as `new-spec`. Candidates scan all tasks project-wide.

```
specd read <task-id> [--with-links] [--with-deps] [--with-criteria] [--with-citations]
specd list tasks [--spec-id SPEC-N] [--status <status>] [--linked-to TASK-N]
                 [--depends-on TASK-N] [--created-by <name>] [--limit N]
specd update <task-id> [--title ...] [--summary ...] [--body ...]
specd move <task-id> --status <status>
specd rename <task-id> --title "<new title>"
specd delete <task-id>
specd reorder task <id> --before <id> | --after <id> | --to <position>
```

### 7.4 Acceptance criteria

```
specd criteria list <task-id>
specd criteria add <task-id> "<text>"
specd criteria check <task-id> <position>       # writes [x] in markdown + SQLite
specd criteria uncheck <task-id> <position>     # writes [ ] in markdown + SQLite
specd criteria remove <task-id> <position>
```

All criteria commands round-trip through the markdown file. Humans editing the checkbox state directly in a text editor is equally valid; the watcher syncs either direction.

### 7.5 Links, dependencies, citations

```
specd link <from-id> <to-id>...                  # spec-to-spec or task-to-task
specd unlink <from-id> <to-id>...
specd depend <task-id> --on <task-id>...         # declare dependencies
specd undepend <task-id> --on <task-id>...
specd cite <spec-id|task-id> <KB-N:position>...  # add KB chunk citations
specd uncite <spec-id|task-id> <KB-N:position>...
specd candidates <spec-id|task-id> [--limit 20]
```

`candidates` runs hybrid search using the subject's title + summary + body against others of the same kind, excludes already-linked targets, returns top N ranked, and additionally returns a `kb_chunks` section with up to 20 relevant KB chunk candidates from the same search.

### 7.6 Sequencing

```
specd next [--limit 10] [--spec-id SPEC-N]
```

Returns up to `--limit` tasks in `todo` status, sorted for execution.

**Sort order** (each key breaks ties of the previous):
1. **Ready** (all dependencies in `done`, `cancelled`, or `wontfix`) before not-ready.
2. Within ready: **partially-done** (at least one criterion checked) before fresh.
3. Within partially-done: higher **criteria completion percentage** first.
4. Within equally-done: **kanban position** ascending.

Response:

```json
{
  "tasks": [
    {
      "id": "TASK-7",
      "title": "...",
      "ready": true,
      "partially_done": true,
      "criteria_progress": 0.8,
      "blocked_by": []
    },
    {
      "id": "TASK-12",
      "title": "...",
      "ready": false,
      "partially_done": false,
      "criteria_progress": 0.0,
      "blocked_by": ["TASK-5"]
    }
  ]
}
```

Dependency cycles halt the command with an error that includes the cycle path. Lint also reports cycles.

### 7.7 Knowledge base

```
specd kb add <path-or-url> [--title "..."] [--note "..."]
specd kb list [--source-type md|html|pdf|txt]
specd kb read <kb-id> [--chunk <n>]
specd kb search "<query>" [--limit 20]
specd kb connections <kb-id> [--chunk <n>] [--limit 20]
specd kb rebuild-connections [--threshold 0.3] [--top-k 10]
specd kb remove <kb-id>
```

`kb add` copies the source into `specd/kb/` with a generated filename, then processes by type:

- **Markdown (md), plain text (txt)**: read directly, chunk on paragraph boundaries, index.
- **HTML**: parse with `golang.org/x/net/html`, extract text, sanitize original with bluemonday (`UGCPolicy`) and write `<slug>.clean.html` sidecar, chunk extracted text, index.
- **PDF**: extract text per page using `github.com/gen2brain/go-fitz`, record page numbers with character offsets, chunk across pages but never break a page in the middle (chunk boundaries align to paragraph breaks within a page), index. Store `page_count` in `kb_docs`.

Chunking targets 500-1000 characters per chunk with a hard cap of 2000. Chunks respect paragraph boundaries; only a very long paragraph is split mid-sentence.

After indexing, compute TF-IDF cosine connections for the new chunks against existing chunks (see 6.2) and insert into `chunk_connections`.

URLs are fetched, saved locally under `specd/kb/`, and then processed as the appropriate type based on Content-Type or extension.

### 7.8 Unified search

```
specd search "<query>" [--kind spec|task|kb|all] [--limit 20] [--json]
```

Hybrid search across the selected kinds. Default `--kind all`. Results grouped by kind in the response, each with a relevance score and match type.

### 7.9 Maintenance

```
specd lint [--json]                              # read-only consistency checks
specd tidy                                       # runs lint, records last_tidy_at
specd rebuild [--force]                          # wipe cache.db, re-parse workspace
specd status [--detailed]                        # project summary
specd merge-fixup                                # repair ID collisions after git merge
```

`lint` reports:
- Dangling references in frontmatter (linked or cited IDs that do not exist).
- Orphan specs (no incoming links, no tasks).
- Orphan tasks (parent spec missing).
- System-field drift (frontmatter disagrees with SQLite for system-managed fields).
- Stale tidy (`last_tidy_at` older than threshold, default 7 days).
- Missing or trivial summaries (empty or single-word).
- Dependency cycles.
- Rejected files.
- KB docs where the source file is missing or hash mismatch.
- Citations pointing to nonexistent KB chunks.

`tidy` runs lint and updates `last_tidy_at`. Every CLI response includes a non-null `tidy_reminder` field when `last_tidy_at` is older than the threshold; agents surface this to the user.

`merge-fixup` scans for duplicate IDs, inconsistent link references, and other post-merge damage. Presents findings interactively and offers to renumber, updating folder names, frontmatter references across all affected files, and SQLite rows.

### 7.10 Trash

```
specd trash list [--kind spec|task|kb] [--older-than <duration>]
specd trash restore <trash-id>
specd trash purge [--older-than 30d]
specd trash purge-all
```

Restore recreates the original file and SQLite rows. If the original ID has been reused, restore allocates a new ID, notes this in the restored file's title, and warns.

### 7.11 Web UI

```
specd serve [--port 7823] [--open]
specd watch                                      # standalone watcher without server
```

Starts an HTTP server on localhost serving the embedded read-write web UI. Described in detail in section 8.

## 8. Web UI

### 8.1 General principles

- All assets embedded in the Go binary via `embed.FS`. **Zero network requests at runtime.** No CDNs, no external fonts, no tracking.
- Bundled client libraries: `marked.min.js` for markdown rendering, `pdf.js` and `pdf.worker.js` for PDF rendering. Approximate total bundle cost: 3-5 MB.
- CSS: determined by the implementer. Must be bundled locally, no CDN. User-configurable styling is out of scope for v1.
- Vanilla JS for UI logic; no React, Vue, or other SPA framework required. Small utility libraries inlined as needed.
- Single writer: UI actions that mutate state POST to the same internal functions as the CLI commands, going through the same lockfile.

### 8.2 Views

- **Kanban board**: tasks grouped by status across 8 columns. Filterable by spec and by `created_by`. Drag-and-drop between columns calls `move`. Drag-and-drop within a column calls `reorder`. Task cards show title, progress bar for criteria, dependency warning icon if not ready, citation icon if the task has references.
- **Spec list**: all specs grouped by type with progress bars (based on non-cancelled/non-wontfix tasks).
- **Spec detail**: body, linked specs, task list with criteria progress, citations section (see 8.3), incoming links.
- **Task detail**: body, acceptance criteria with interactive checkboxes, linked tasks, dependencies (with readiness indicator), citations section, parent spec link.
- **KB browser**: list of docs grouped by source type; click to read.
- **KB reader**: described in 8.4. Renders markdown, text, HTML, or PDF with chunk highlighting and navigation.
- **Search**: single input running hybrid search across all kinds; results grouped by kind.
- **Trash**: list, preview, restore.
- **Rejected files**: list with reason and detection timestamp.
- **Project status**: counts by kind and status, tidy state, lint summary.

### 8.3 Citations section (in spec and task detail views)

The citations section renders each citation as a card:

```
┌─ References ───────────────────────────────────────────────────┐
│                                                                │
│  [PDF]  OAuth 2.0 RFC 6749 · chunk 12 · page 4                │
│  "Authorization code grant is the most commonly used grant    │
│   type, optimized for confidential clients. Because this is   │
│   a redirection-based flow, the client must be capable of..." │
│                                              [View in source] │
│                                                                │
│  [MD]   JWT best practices · chunk 3                          │
│  "Always verify the signature before trusting any claims.    │
│   Libraries that skip signature verification when alg..."     │
│                                              [View in source] │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

Each card shows:
- Source type icon ([PDF], [MD], [HTML], [TXT]).
- KB doc title.
- Chunk position (and page number for PDFs).
- Preview: `kb_chunks.text` truncated to ~200 characters.
- "View in source" action opens the KB reader (8.4) anchored to this chunk.

Preview text comes from the response of `read <spec-id> --with-citations`; no additional fetch is needed.

### 8.4 KB reader

The reader opens when the user clicks "View in source" from a citation, or when navigating from the KB browser. First implementation: modal overlay covering most of the viewport. Split-pane alternative is a v2 upgrade.

Reader header shows:
- Doc title and source type.
- Current chunk position (`12 / 47`).
- Previous chunk, next chunk, close.
- Full chunk list toggle (side strip with every chunk's preview, clickable).

Reader body renders content per source type.

### 8.5 API endpoints for the reader

```
GET /api/kb/:doc_id                             # metadata
GET /api/kb/:doc_id/chunks                      # all chunks for the doc
GET /api/kb/:doc_id/chunk/:position             # single chunk with metadata
GET /api/kb/:doc_id/raw                         # raw or cleaned source bytes
```

`/raw` returns:
- Markdown/text: the file contents with `Content-Type: text/plain; charset=utf-8`.
- HTML: the sanitized clean sidecar (`.clean.html`) with `Content-Type: text/html; charset=utf-8`. Never the original HTML.
- PDF: the PDF bytes with `Content-Type: application/pdf`.

Path traversal protection: the handler resolves the path via SQLite lookup on `doc_id`, never from user input. Paths are constrained to live inside `specd/kb/`.

### 8.6 Rendering per source type

Chunk highlighting uses CSS:

```css
.chunk-highlight {
  background: #fff3a8;
  border-left: 3px solid #f5b700;
  padding: 2px 4px;
  scroll-margin: 100px;
}
```

(Exact styling left to the implementer; the key requirement is that the highlighted region is visually distinct and scrolls into view.)

**Markdown** (`source_type = 'md'`):

1. Fetch raw markdown from `/api/kb/:doc_id/raw`.
2. Render client-side with `marked.parse(raw)` (marked.js inlined from binary).
3. Walk the rendered DOM looking for the chunk's text. Use a prefix match (first 80 characters of `chunk.text`) rather than the full string, since formatting inside the chunk may split text across multiple DOM nodes.
4. Wrap the match in a `<mark class="chunk-highlight">`.
5. Scroll the match into view with `element.scrollIntoView({block: 'center'})`.

Pseudocode:

```javascript
async function renderMarkdown(docId, chunk) {
  const raw = await fetch(`/api/kb/${docId}/raw`).then(r => r.text());
  container.innerHTML = marked.parse(raw);
  const mark = highlightPrefixInDOM(container, chunk.text.slice(0, 80));
  if (mark) mark.scrollIntoView({block: 'center'});
}

function highlightPrefixInDOM(root, prefix) {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  while (walker.nextNode()) {
    const node = walker.currentNode;
    const idx = node.textContent.indexOf(prefix);
    if (idx < 0) continue;
    const range = document.createRange();
    range.setStart(node, idx);
    range.setEnd(node, Math.min(idx + prefix.length, node.textContent.length));
    const mark = document.createElement('mark');
    mark.className = 'chunk-highlight';
    range.surroundContents(mark);
    return mark;
  }
  return null;
}
```

**Plain text** (`source_type = 'txt'`):

1. Fetch raw text.
2. Slice on `chunk.char_start` and `chunk.char_end` (character offsets are exact for plain text).
3. Render as:
   ```html
   <pre>{before}<mark class="chunk-highlight">{target}</mark>{after}</pre>
   ```
4. Scroll the mark into view.

Escape HTML-special characters in all three slices before inserting.

**HTML** (`source_type = 'html'`):

1. Fetch the cleaned HTML from `/api/kb/:doc_id/raw` (bluemonday-sanitized at ingest, no scripts or event handlers).
2. Create an iframe with `sandbox="allow-same-origin"` (no `allow-scripts`), append to the reader body.
3. Write the HTML into the iframe's document: `iframe.contentDocument.open(); ... .write(raw); ... .close()`.
4. Walk the iframe's DOM using the same prefix-match as markdown.
5. Highlight and scroll inside the iframe.

The iframe isolates CSS from the parent reader and provides a security boundary. The `allow-same-origin` flag is needed so the parent can reach into `iframe.contentDocument` for highlighting; it does not enable script execution because `allow-scripts` is absent.

**PDF** (`source_type = 'pdf'`):

1. Fetch raw PDF bytes from `/api/kb/:doc_id/raw` as `ArrayBuffer`.
2. Load with PDF.js: `const pdf = await pdfjsLib.getDocument({data: pdfData}).promise;`.
3. Render `chunk.page` to a canvas at a readable scale (e.g., `scale: 1.5`).
4. Render PDF.js's built-in text layer over the canvas (invisible positioned divs per text run).
5. Walk the text layer DOM, find the prefix match for `chunk.text`, wrap matches in `<mark class="chunk-highlight">`.
6. Scroll the mark into view.
7. Previous/next chunk navigation: if the adjacent chunk is on a different page, render that page and repeat.

The PDF.js worker is loaded from `/assets/pdf.worker.js` (embedded path). Configure PDF.js at startup:

```javascript
pdfjsLib.GlobalWorkerOptions.workerSrc = '/assets/pdf.worker.js';
```

PDF.js disables embedded PDF JavaScript by default; verify in the bundled version and keep it disabled.

### 8.7 Chunk navigation inside the reader

Previous/next buttons in the header fetch the adjacent chunk's data (`/api/kb/:doc_id/chunk/:position±1`) and re-highlight. For markdown, text, and HTML, this moves the highlight within already-loaded content. For PDFs, it may render a new page.

The full chunk list side strip (togglable) shows every chunk as a clickable preview (first 60 characters). Clicking jumps directly.

### 8.8 Security summary for the reader

- HTML sanitization with bluemonday at ingest time. Only cleaned HTML is ever served to the browser.
- Iframe sandboxing (`sandbox="allow-same-origin"`) for HTML rendering. No script execution.
- PDF.js with JavaScript disabled.
- Path traversal protection via SQLite-resolved paths; never accept paths from client input.
- All `/api/kb/*` endpoints constrained to files under `specd/kb/` verified by prefix check.

## 9. File watcher

Started automatically by `specd serve`. Also available as `specd watch`.

Algorithm per file change event:

1. Compute content hash.
2. If the hash matches the stored `content_hash` for the file's row, skip (CLI's own write).
3. If no matching row exists:
   - If filename matches a canonical pattern and the file's directory is under `specd/` in a known location, it may be ingested (during a rebuild pass only; normal operation rejects).
   - Otherwise, insert into `rejected_files`.
4. If a row exists and the hash differs:
   - Re-parse frontmatter and body.
   - For system-managed fields, if markdown disagrees with SQLite, log drift warning. SQLite wins; markdown will be rewritten on next CLI touch.
   - For user-editable fields, markdown wins.
   - Re-parse `## Acceptance criteria` (if task) and sync `task_criteria`.
   - Update `content_hash`, `updated_at`, trigger FTS/trigram index updates.
5. Deletions: move data to `trash` with `deleted_by='watcher'`, remove primary row, cascades handle links/deps/citations.
6. Debounce: batch events within 200ms per path.

## 10. Single-writer enforcement

`.specd/lock` is a POSIX `flock`-style lockfile managed via `github.com/gofrs/flock`. Every mutation acquires an exclusive lock, holds it for the full transaction, and releases. Timeout: 5 seconds; on timeout the command exits with an error.

The web server acquires the lock only during writes, not for the server lifetime.

## 11. AGENTS.md section

`specd init` writes or appends this section:

```markdown
## Spec, task, and knowledge management via specd

This workspace uses `specd` for managing specifications, tasks, and a reference
knowledge base. specd is a local CLI backed by SQLite. It contains no AI;
you are the intelligence.

**When to use specd:**
- User asks about requirements, specs, tasks, project status, progress, dependencies, acceptance criteria, or sequencing.
- User asks to create, update, link, move, cite, or verify specs and tasks.
- User asks about references, sources, prior research, or mentions a doc they added.
- User asks about inconsistencies, orphans, or stale work.
- You need to understand what has been decided or researched about the project.

Do not read files under `specd/` directly. Always use `specd` commands with `--json`.
Do not create files under `specd/` manually — they will be rejected.

**Key commands** (always pass `--json`):

| Task | Command |
|------|---------|
| Create a spec | `specd new-spec --title "..." --type <business\|technical\|non-technical> --summary "..." --body "..."` |
| Create a task | `specd new-task --spec-id SPEC-N --title "..." --summary "..." --body "..."` |
| Add a reference doc | `specd kb add <path-or-url> --title "..." --note "..."` |
| Free-text search | `specd search "query" --kind all` |
| KB-only search | `specd kb search "query"` |
| Find link and citation candidates | `specd candidates SPEC-N` |
| Read a spec with tasks and citations | `specd read SPEC-N --with-tasks --with-links --with-progress --with-citations` |
| List in-progress tasks | `specd list tasks --status in_progress` |
| Move a task | `specd move TASK-N --status <status>` |
| Declare a dependency | `specd depend TASK-N --on TASK-M` |
| Get next ready tasks | `specd next --limit 10` |
| Check an acceptance criterion | `specd criteria check TASK-N <position>` |
| Cite a KB chunk | `specd cite SPEC-N KB-M:position` |
| Run consistency checks | `specd lint` |
| Project status | `specd status --detailed` |

**Creating a spec:**
1. Draft spec content in conversation with the user. Use `specd kb search` to ground the draft in reference material.
2. Call `specd new-spec` with title, type, summary, and full body.
3. Review the returned candidate specs, tasks, and KB chunks with the user.
4. On approval, call `specd link` and `specd cite` to persist the relationships.

**Creating a task:** same pattern, scoped under a parent spec. Consider declaring dependencies with `specd depend` where applicable.

**Verifying a task:**
1. Call `specd read TASK-N --with-criteria --with-citations`.
2. For each unchecked criterion, inspect code, run tests, or consult cited references.
3. Call `specd criteria check TASK-N <position>` for each verified item.
4. If all criteria pass, `specd move TASK-N --status done`. If partial, consider `pending_verification` or `blocked` and move on to other work.

**Picking next work:**
1. Call `specd next --limit 10`.
2. Present ready tasks to the user.
3. On selection, `specd move TASK-N --status in_progress`.

**Brownfield bootstrap:** when specd is newly installed in an existing project with code but no specs, read the codebase to populate initial specs and tasks. Identify major subsystems (technical specs), user-facing features (business specs), and TODOs/gaps (tasks). Use existing commands; there is no special bootstrap command.

**System-managed frontmatter fields** (`linked_specs`, `linked_tasks`, `depends_on`, `cites`) must never be edited by hand. Always use CLI commands.

**Tidy reminders:** when a specd response contains a non-null `tidy_reminder`, mention it and suggest running `/specd-tidyup`.
```

For agents that do not read AGENTS.md natively, `specd init --wire-legacy` offers to add the following single line to each detected legacy file, with a diff preview and per-file confirmation:

```
See AGENTS.md for spec/task management instructions.
```

Example: `CLAUDE.md` can be a one-line file containing only that sentence. The installer itself never touches any files; all project wiring happens on explicit `init`.

## 12. Claude Skill deliverable

`specd init --skill` installs a Claude Skill folder for use with Claude Desktop, mobile, and web:

```
specd-skill/
├── SKILL.md
└── resources/
    └── commands.md
```

Installed at the platform-appropriate path:
- macOS: `~/Library/Application Support/Claude/skills/specd/`
- Linux: `~/.config/Claude/skills/specd/`
- Windows: `%APPDATA%\Claude\skills\specd\`

`SKILL.md` contains the same guidance as the AGENTS.md section, adapted for user-scoped context. Since Claude Desktop does not automatically know the current workspace, `SKILL.md` explicitly instructs Claude to determine the workspace by checking the `SPECD_WORKSPACE` environment variable, the active workspace from `specd workspace list`, or by asking the user.

The skill works across all Claude surfaces where skills are supported, sharing the same specd binary and workspace data.

## 13. Slash commands

Example slash command definitions ship in `docs/slash-commands/` for users to copy into their agent's command directory. All prefixed `/specd-`:

- `/specd-new-spec`
- `/specd-new-task`
- `/specd-new-kb`
- `/specd-search`
- `/specd-kb-search`
- `/specd-link`
- `/specd-cite`
- `/specd-move`
- `/specd-next`
- `/specd-verify`
- `/specd-depend`
- `/specd-status`
- `/specd-tidyup`
- `/specd-bootstrap`
- `/specd-restore`

Each is a thin wrapper telling the agent to call the corresponding specd subcommand. Not required (AGENTS.md alone is sufficient for autonomous use); provided for discoverability and explicit user intents.

## 14. Attribution, not namespacing

specd does not namespace folders by developer. One shared `specd/` per workspace, committed to git, visible to all collaborators. All developers on a team see the same unified set of specs, tasks, and KB.

Attribution is handled per-row:

- `specd config user.name "Adam"` stores the developer's name in `meta.user_name`.
- `created_by` and `updated_by` columns on specs and tasks are populated from this value on every write.
- `specd list tasks --created-by adam` filters by author.
- The kanban UI can show author badges on task cards.

On first run, `specd init` seeds `meta.user_name` from `git config user.name` if available.

## 15. Multi-developer scenarios

### 15.1 ID collisions on merge

Two developers branch, both run `specd new-spec`, both allocate SPEC-43. After merge, two folders exist. `specd rebuild` detects duplicate IDs and halts. `specd merge-fixup` detects duplicates, presents them to the user, and renumbers one (updating folder name, rewriting all frontmatter references, updating SQLite).

### 15.2 Concurrent edits to the same file

Standard git merge conflict resolved by the user in their editor. Watcher picks up the resolved file.

### 15.3 Conflicting link declarations

Conflict on frontmatter lines. User resolves. Watcher syncs.

### 15.4 Orphaned links after branch merge

Lint reports dangling links on the next pass. User fixes.

### 15.5 Kanban position drift

Positions are stored only in SQLite, not in markdown. On a fresh clone, after `rebuild`, or after merge-fixup, positions reset to creation order. **Kanban ordering is local and does not survive branch merges or fresh clones in v1.** Documented limitation.

### 15.6 Trash isolation

Trash is per-machine (lives in SQLite). Deletes made on one machine are not visible on another. Acceptable because trash is an undo mechanism for local accidents, not a team recycling bin.

### 15.7 Counter desync

`specd rebuild` scans all existing files and sets `next_spec_id = max(existing) + 1` (same for tasks and KB), preventing collisions with existing IDs after any rebuild.

### 15.8 Recommended team workflow

- Prefer short-lived branches.
- Run `specd lint` before opening a PR.
- On merge conflicts involving specd files, run `specd merge-fixup` as part of resolution.
- Optionally adopt an author convention for ID ranges to reduce collision frequency.

## 16. Error handling and edge cases

- **Concurrent CLI invocations**: second process waits up to 5s on the lockfile, then errors.
- **Invalid frontmatter**: watcher logs error, leaves SQLite unchanged, lint reports unparseable file.
- **Missing referenced ID in a link/cite command**: CLI rejects before writing.
- **Cycle in task dependencies**: `depend` detects and rejects before writing. Lint reports cycles.
- **Large KB files**: chunking caps at 10,000 chunks per doc; larger files are rejected with a size warning. Configurable.
- **URL fetch failures in `kb add`**: command fails cleanly, no partial state.
- **PDF extraction failures**: command fails cleanly, logs the error, does not create a partial KB doc.
- **Rebuild when markdown has been manually added**: non-canonical files go to `rejected_files`; canonical-named files are ingested.
- **Symlinks in the workspace**: followed only if pointing inside the workspace; external symlinks ignored with warning.
- **Workspace path contains spaces or unusual characters**: supported; paths are quoted where necessary.
- **Trash restore when original ID has been reused**: allocates a new ID, annotates the restored file's title, warns.

## 17. Build, distribution, install

**Language**: Go. Single static binary, cross-compiled for darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64.

**Key dependencies:**
- `modernc.org/sqlite` (pure-Go SQLite for easy cross-compilation).
- `fsnotify/fsnotify` for the file watcher.
- `spf13/cobra` for CLI framework.
- `gofrs/flock` for cross-platform file locking.
- `github.com/gen2brain/go-fitz` for PDF text extraction with page awareness. **Note**: this is a CGO dependency wrapping MuPDF. If CGO is unacceptable, fall back to `github.com/ledongthuc/pdf` (pure Go) with limited PDF format support, or `github.com/unidoc/unipdf` (pure Go, commercial license considerations).
- `github.com/microcosm-cc/bluemonday` for HTML sanitization.
- `golang.org/x/net/html` for HTML parsing.
- Standard library `embed`, `net/http`, `html/template`, `archive/zip`.

**Embedded client assets** (via `embed.FS`):
- `marked.min.js` (~30 KB) for markdown rendering.
- `pdf.js` and `pdf.worker.js` (~3-5 MB total) for PDF rendering.
- UI HTML, CSS, and JS written by the implementer.
- SKILL.md template.
- AGENTS.md template.

**Installation paths:**
- GitHub Releases with pre-built binaries.
- `curl -sSL https://specd.dev/install.sh | sh` (installs binary only, touches no project files).
- Homebrew tap: `brew install specd`.
- Scoop for Windows.
- Optional thin wrappers for `npm install -g specd-cli` and `pipx install specd` that download the native binary on install.

The installer never edits project files or user config. All project and skill wiring happens on explicit `specd init`.

## 18. Suggested build order

1. **Data layer**: SQLite schema, migrations, frontmatter parser, file hashing, trash, counters.
2. **Core spec and task CLI**: `init`, `config`, `new-spec`, `new-task`, `read`, `list`. Happy path only.
3. **Acceptance criteria**: parser, commands, markdown round-trip.
4. **Linking and dependencies**: `link`, `unlink`, `depend`, `undepend`, `candidates` (specs/tasks only at this stage).
5. **Hybrid search infrastructure**: FTS5 + trigram setup, `search` command, triggers, index maintenance.
6. **Knowledge base (ingest)**: `kb add|list|read|remove`, chunking for md/txt, HTML parsing with bluemonday, PDF text extraction with go-fitz, `kb search`.
7. **KB connections**: TF-IDF cosine, `chunk_connections` table, `kb connections`, `kb rebuild-connections`.
8. **Citations**: `citations` table, `cite`/`uncite`, frontmatter `cites` field, `--with-citations` on `read`, candidates block includes `kb_chunks`.
9. **Mutation commands**: `update`, `move`, `rename`, `delete`, `reorder`.
10. **Sequencing**: `next` with topological sort, partially-done ordering.
11. **Maintenance**: `lint`, `tidy`, `rebuild`, `status`, `merge-fixup`, `trash` commands.
12. **File watcher**: `watch` standalone + integration with `serve`.
13. **Web UI scaffolding**: embedded assets, router, templates, base CSS.
14. **Kanban board**: columns, cards, drag-and-drop, `move`/`reorder` round-trip.
15. **Spec and task detail views**: body, links, deps, criteria, citations section.
16. **KB browser**: list, read, search.
17. **KB reader**: markdown and text renderers with chunk highlighting.
18. **KB reader HTML**: sandboxed iframe with chunk highlighting.
19. **KB reader PDF**: PDF.js integration, page rendering, text-layer highlighting, chunk navigation.
20. **Search, trash, rejected files, and status views**.
21. **Workspace registry** (`workspace add|list|use|remove`) and Claude Skill generator (`init --skill`).
22. **Legacy wiring** (`init --wire-legacy`) and AGENTS.md generator.
23. **Cross-platform builds, installers, release pipeline**.
24. **Example slash commands in `docs/slash-commands/`**.

## 19. Acceptance criteria for v1

- `specd init` in an empty directory produces a working workspace, SQLite cache, AGENTS.md section, and `.gitignore` entry.
- An AI agent, given only AGENTS.md and shell access, can create specs, tasks, and KB entries; link them; declare dependencies; cite KB chunks; search them; move tasks; verify criteria; pick next work; and report project status using `specd` commands with `--json`.
- A human editing a markdown file in any editor sees the change reflected in SQLite within one second of save (via the running watcher), and in the kanban UI on next refresh.
- `specd rebuild` fully reconstructs SQLite from the workspace with no data loss for canonical files, and reports non-canonical files via rejected files.
- The kanban web UI loads in under 500ms for a workspace with 100 specs, 1000 tasks, and 50 KB docs. Drag-and-drop between all 8 status columns works and persists.
- `specd lint` completes in under 1 second for the same workspace size.
- `specd search` returns ranked results in under 100ms.
- `specd next` returns ready tasks in topological order in under 50ms.
- The KB reader renders markdown, text, HTML, and PDF documents with chunk highlighting and scroll-to-chunk, entirely offline (no network requests).
- PDF documents of at least 100 pages render and navigate smoothly in the reader.
- TF-IDF chunk connections populate on `kb add` and are queryable via `specd kb connections`.
- `specd trash restore` successfully recovers soft-deleted items.
- `specd merge-fixup` detects and repairs simple ID collisions after a forced manual test merge.
- The binary runs on macOS (Intel and Apple Silicon), Linux (x86_64 and arm64), and Windows (x86_64) with no runtime network dependencies.
- The Claude Skill, installed via `specd init --skill`, works from Claude Desktop by calling `specd` commands against a user-specified workspace path.

## 20. Open implementation questions

- CGO vs pure-Go for PDF extraction. go-fitz (MuPDF, CGO) is the recommended default; document the fallback to a pure-Go library if CGO must be avoided.
- Exact trigram fallback threshold (starts at 3 hits; tune with usage).
- TF-IDF cosine threshold and top-k (start at 0.3 and 10; tune).
- Whether `specd next` should expose the sort keys in the response for debugging.
- Kanban reader split-pane upgrade (v2 consideration).
- Whether to show TF-IDF chunk connections proactively in the reader's sidebar ("related chunks from other docs"), or only on explicit command.
- Workspace templates (v2).
- Optional semantic embeddings behind a feature flag (v2).