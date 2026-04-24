# specd CLI Reference

Complete reference for all specd commands, flags, and output formats.

## Global Behavior

- All commands except `init`, `version`, `skills`, `help`, and `logs` require an initialized project (`.specd.json` in the current directory) and a configured username.
- Before each command, the cache sync runs automatically to reconcile markdown files with the database.
- After each command, an update check runs silently (cached for 24 hours).

---

## specd init

Initialize specd in a project directory.

```
specd init [project-path] [flags]
```

### Arguments

| Argument | Required | Description |
|---|---|---|
| `project-path` | No | Path to the project directory (default: current directory) |

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--folder` | string | `specd` | Folder name for specd project files |
| `--username` | string | | Your username (prompts interactively if omitted) |
| `--skip-skills` | bool | `false` | Skip AI skills installation prompt |

### What It Creates

- `.specd.json` — project config marker (commit to VCS)
- `.specd.cache` — SQLite cache database (add to `.gitignore`)
- `<folder>/specs/` — directory for spec and task files
- `<folder>/kb/` — directory for knowledge base documents
- `~/.specd/config.json` — global username config (first run only)

### Interactive Prompts

When `--folder` and `--username` are both omitted, init runs interactively:

1. Folder name (default: `specd`)
2. Username (tries to detect from global config or `git config user.name`)
3. Spec types — select from defaults or add custom types
4. Task stages — select from required + optional stages
5. AI skills installation — choose providers and install level

When both flags are provided, init uses all defaults non-interactively.

### Default Spec Types

| Display Name | Slug |
|---|---|
| Business | `business` |
| Functional | `functional` |
| Non-functional | `nonfunctional` |

### Default Task Stages

**Required** (always included):

| Display Name | Slug |
|---|---|
| Backlog | `backlog` |
| Todo | `todo` |
| In progress | `in_progress` |
| Done | `done` |

**Optional** (included by default, can deselect):

| Display Name | Slug |
|---|---|
| Blocked | `blocked` |
| Pending Verification | `pending_verification` |
| Cancelled | `cancelled` |
| Wont Fix | `wont_fix` |

---

## specd new-spec

Create a new specification.

```
specd new-spec --title "..." --summary "..." --body "..."
```

### Flags

| Flag | Type | Required | Description |
|---|---|---|---|
| `--title` | string | Yes | Spec title (becomes the `# Heading` in the file) |
| `--summary` | string | Yes | One-line description |
| `--body` | string | Yes | Markdown body (see body format below) |

### Body Format

- Do NOT include a title — `--title` becomes the `# Heading` automatically
- Use `##` for top-level sections
- Include `## Acceptance Criteria` with bullet items using must/should/is/will language
- Do NOT use checkbox syntax (`- [ ]`) — that is for tasks only

### Output

```json
{
  "id": "SPEC-1",
  "path": "specd/specs/spec-1/spec.md",
  "default_type": "business",
  "available_types": ["business", "functional", "nonfunctional"],
  "related_specs": [],
  "related_kb_chunks": []
}
```

| Field | Description |
|---|---|
| `id` | Generated spec ID |
| `path` | Path to the created file |
| `default_type` | The type assigned (first configured type) |
| `available_types` | All configured spec types |
| `related_specs` | Search results for related specs (by title + summary) |
| `related_kb_chunks` | Search results for related KB content |

---

## specd get-spec

Retrieve a spec by ID with full details.

```
specd get-spec --id "SPEC-1"
```

### Flags

| Flag | Type | Required | Description |
|---|---|---|---|
| `--id` | string | Yes | Spec ID (e.g., `SPEC-1`) |

### Output

```json
{
  "id": "SPEC-1",
  "title": "User Authentication",
  "type": "functional",
  "summary": "OAuth2 login with Google and GitHub",
  "body": "## Overview\n\n...",
  "path": "specd/specs/spec-1/spec.md",
  "position": 0,
  "linked_specs": ["SPEC-3"],
  "claims": [
    {"position": 1, "text": "The system must redirect to the consent screen"},
    {"position": 2, "text": "The system should support GitHub"}
  ],
  "tasks": [
    {
      "id": "TASK-1",
      "title": "Implement OAuth Redirect",
      "status": "backlog",
      "summary": "Build the redirect handler",
      "criteria": [
        {"position": 1, "text": "The handler must build the auth URL", "checked": 0},
        {"position": 2, "text": "The state parameter should be random", "checked": 1}
      ]
    }
  ],
  "created_by": "alice",
  "updated_by": "",
  "content_hash": "a1b2c3...",
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

| Field | Description |
|---|---|
| `claims` | Acceptance criteria from the spec's `## Acceptance Criteria` section |
| `tasks` | All child tasks with their criteria and checked state |
| `linked_specs` | IDs of related specs |
| `content_hash` | SHA-256 of the file for change detection |

---

## specd update-spec

Update a spec's type, linked specs, or KB citations.

```
specd update-spec --id "SPEC-1" [flags]
```

### Flags

| Flag | Type | Required | Description |
|---|---|---|---|
| `--id` | string | Yes | Spec ID to update |
| `--type` | string | No | New spec type (must be a configured type) |
| `--link-specs` | string | No | Comma-separated spec IDs to add as related |
| `--unlink-specs` | string | No | Comma-separated spec IDs to remove from related |
| `--link-kb-chunks` | string | No | Comma-separated KB chunk IDs to cite |
| `--unlink-kb-chunks` | string | No | Comma-separated KB chunk IDs to remove |

### Output

```json
{
  "id": "SPEC-1",
  "type": "functional",
  "linked_specs": [
    {"id": "SPEC-3", "title": "Session Management", "summary": "Token handling"}
  ],
  "linked_kb_chunks": [
    {"chunk_id": 1, "doc_id": "KB-1", "preview": "OAuth2 requires..."}
  ]
}
```

The spec file on disk is automatically rewritten to reflect changes.

---

## specd delete-spec

Delete a spec and all its tasks.

```
specd delete-spec --id "SPEC-1"
```

### Flags

| Flag | Type | Required | Description |
|---|---|---|---|
| `--id` | string | Yes | Spec ID to delete |

### Behavior

- Deletes the spec from the database (cascades to tasks, links, claims, citations)
- Removes the entire spec directory from disk (including all task files)

### Output

```json
{
  "id": "SPEC-1",
  "deleted": true,
  "path": "specd/specs/spec-1"
}
```

---

## specd list-specs

List all specs with pagination.

```
specd list-specs [--page N] [--page-size N]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--page` | int | `1` | Page number (1-based) |
| `--page-size` | int | `20` | Results per page |

### Output

```json
{
  "specs": [
    {
      "id": "SPEC-1",
      "title": "User Authentication",
      "type": "functional",
      "summary": "OAuth2 login",
      "position": 0,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total_count": 1,
  "total_pages": 1
}
```

---

## specd new-task

Create a new task for a spec.

```
specd new-task --spec-id "SPEC-1" --title "..." --summary "..." --body "..."
```

### Flags

| Flag | Type | Required | Description |
|---|---|---|---|
| `--spec-id` | string | Yes | Parent spec ID |
| `--title` | string | Yes | Task title |
| `--summary` | string | Yes | One-line summary |
| `--body` | string | Yes | Markdown body (see body format below) |

### Body Format

- Do NOT include a title — `--title` becomes the `# Heading` automatically
- Use `##` for top-level sections
- Include `## Acceptance Criteria` with **checkbox** items: `- [ ] criterion text`
- Criteria should use must/should/is/will language

### Output

```json
{
  "id": "TASK-1",
  "spec_id": "SPEC-1",
  "path": "specd/specs/spec-1/TASK-1.md",
  "status": "backlog"
}
```

The task file is created inside the parent spec's directory.

---

## specd get-task

Retrieve a task by ID with full details.

```
specd get-task --id "TASK-1"
```

### Flags

| Flag | Type | Required | Description |
|---|---|---|---|
| `--id` | string | Yes | Task ID (e.g., `TASK-1`) |

### Output

```json
{
  "id": "TASK-1",
  "spec_id": "SPEC-1",
  "title": "Implement OAuth Redirect",
  "status": "in_progress",
  "summary": "Build the redirect handler",
  "body": "## Overview\n\n...",
  "path": "specd/specs/spec-1/TASK-1.md",
  "position": 0,
  "linked_tasks": ["TASK-4"],
  "depends_on": ["TASK-2"],
  "criteria": [
    {"position": 1, "text": "The handler must build the auth URL", "checked": 0},
    {"position": 2, "text": "The state parameter should be random", "checked": 1}
  ],
  "created_by": "alice",
  "content_hash": "d4e5f6...",
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

| Field | Description |
|---|---|
| `linked_tasks` | IDs of related tasks (bidirectional, non-blocking) |
| `depends_on` | IDs of tasks that block this task |
| `criteria` | Acceptance criteria with checked state (0 = unchecked, 1 = checked) |

---

## specd update-task

Update a task's status or toggle acceptance criteria.

```
specd update-task --id "TASK-1" [flags]
```

### Flags

| Flag | Type | Required | Description |
|---|---|---|---|
| `--id` | string | Yes | Task ID to update |
| `--status` | string | No | New task stage (must match a configured stage slug) |
| `--check` | string | No | Comma-separated criterion positions to mark checked |
| `--uncheck` | string | No | Comma-separated criterion positions to mark unchecked |

### Examples

```sh
# Move task to in_progress
specd update-task --id "TASK-1" --status "in_progress"

# Check criteria 1 and 3
specd update-task --id "TASK-1" --check "1,3"

# Uncheck criterion 2
specd update-task --id "TASK-1" --uncheck "2"

# Change status and check criteria in one call
specd update-task --id "TASK-1" --status "done" --check "1,2,3"
```

### Output

```json
{
  "id": "TASK-1",
  "spec_id": "SPEC-1",
  "status": "done",
  "criteria": [
    {"position": 1, "text": "The handler must build the auth URL", "checked": 1},
    {"position": 2, "text": "The state parameter should be random", "checked": 1},
    {"position": 3, "text": "The scopes will be configurable", "checked": 1}
  ]
}
```

The task file on disk is automatically rewritten with updated checkbox states and status.

---

## specd delete-task

Delete a single task.

```
specd delete-task --id "TASK-1"
```

### Flags

| Flag | Type | Required | Description |
|---|---|---|---|
| `--id` | string | Yes | Task ID to delete |

### Output

```json
{
  "id": "TASK-1",
  "spec_id": "SPEC-1",
  "deleted": true,
  "path": "specd/specs/spec-1/TASK-1.md"
}
```

---

## specd list-tasks

List tasks with pagination and optional filters.

```
specd list-tasks [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--page` | int | `1` | Page number (1-based) |
| `--page-size` | int | `20` | Results per page |
| `--spec-id` | string | | Filter by parent spec ID |
| `--status` | string | | Filter by task stage slug |

### Examples

```sh
# All tasks
specd list-tasks

# Tasks for a specific spec
specd list-tasks --spec-id "SPEC-1"

# Only in-progress tasks
specd list-tasks --status "in_progress"

# Combine filters
specd list-tasks --spec-id "SPEC-1" --status "todo" --page-size 10
```

### Output

```json
{
  "tasks": [
    {
      "id": "TASK-1",
      "spec_id": "SPEC-1",
      "title": "Implement OAuth Redirect",
      "status": "backlog",
      "summary": "Build the redirect handler",
      "position": 0,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total_count": 1,
  "total_pages": 1
}
```

---

## specd search

Search specs, tasks, and KB documents by text.

```
specd search --query "..." [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--query` | string | *required* | Search terms |
| `--kind` | string | `all` | What to search: `spec`, `task`, `kb`, or `all` |
| `--limit` | int | `0` | Max results per kind (0 = use project config) |

### Output

```json
{
  "specs": [
    {
      "kind": "spec",
      "id": "SPEC-2",
      "title": "Session Management",
      "summary": "Token handling and refresh",
      "score": 12.5,
      "match_type": "bm25"
    }
  ],
  "tasks": [],
  "kb": []
}
```

| Field | Description |
|---|---|
| `score` | Relevance score (higher = better match). 0 for trigram results. |
| `match_type` | `bm25` (full-text) or `trigram` (substring fallback) |

### Search Strategy

1. **BM25** — FTS5 full-text search with porter stemming. Handles word variants (e.g., "authenticate" matches "authentication").
2. **Trigram** — substring matching fallback. Activated when BM25 returns fewer than 3 results or the query contains special characters.
3. **Deduplication** — same result never appears in both BM25 and trigram.

---

## specd search-claims

Search acceptance criteria claims across all specs.

```
specd search-claims --query "..." [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--query` | string | *required* | Search terms |
| `--exclude` | string | | Spec ID to exclude from results |
| `--limit` | int | `0` | Max results (0 = use project config) |

### Output

```json
[
  {
    "spec_id": "SPEC-1",
    "spec_title": "User Authentication",
    "claim": "The system must invalidate sessions on password change",
    "match_type": "bm25"
  }
]
```

Use this to find contradictions between specs. The `--exclude` flag filters out the spec you're checking against.

---

## specd serve

Start the specd Web UI.

```
specd serve [--port N]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--port` | int | `8000` | Starting port number |

Automatically finds an available port (tries up to 100 ports) and opens the browser.

---

## specd skills install

Install AI provider skills.

```
specd skills install
```

Interactive prompt to select:
1. AI providers (Claude, OpenAI Codex, Gemini)
2. Install level (user-level or repository-level)

Skills are installed from the embedded canonical set to the provider's skill directory.

---

## specd logs

Stream the specd log file.

```
specd logs [--lines N]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--lines` | int | `0` | Number of recent lines to show (0 = all) |

Follows new output like `tail -f`. Press Ctrl+C to stop.

---

## specd version

Print the specd version.

```
specd version
```

---

## File Structure Reference

```
project-root/
  .specd.json                        # Project config (commit to VCS)
  .specd.cache                       # SQLite cache (gitignore)
  specd/                             # Project folder (configurable name)
    specs/
      spec-1/
        spec.md                      # Spec markdown
        TASK-1.md                    # Task files live with their spec
        TASK-2.md
      spec-2/
        spec.md
        TASK-3.md
    kb/
      KB-1.md                        # Knowledge base documents

~/.specd/                            # Global config (never committed)
  config.json                        # Username
  specd.log                          # Log file
  update-check.json                  # Cached version check
  skills/                            # Canonical skill files
    specd-new-spec/SKILL.md
    specd-new-tasks/SKILL.md
    ...
```

### Spec Frontmatter Fields

| Field | Required | Description |
|---|---|---|
| `id` | Yes | `SPEC-<number>` |
| `type` | Yes | Spec type slug (e.g., `business`, `functional`) |
| `summary` | Yes | One-line description |
| `position` | No | Sort order (default: `0`) |
| `linked_specs` | No | YAML list of related spec IDs |
| `created_by` | No | Author username |
| `updated_by` | No | Last editor username |
| `created_at` | No | RFC 3339 timestamp |
| `updated_at` | No | RFC 3339 timestamp |

### Task Frontmatter Fields

| Field | Required | Description |
|---|---|---|
| `id` | Yes | `TASK-<number>` |
| `spec_id` | Yes | Parent spec ID |
| `status` | Yes | Task stage slug (e.g., `backlog`, `in_progress`) |
| `summary` | Yes | One-line description |
| `position` | No | Sort order (default: `0`) |
| `linked_tasks` | No | YAML list of related task IDs (bidirectional) |
| `depends_on` | No | YAML list of blocking task IDs (directed) |
| `created_by` | No | Author username |
| `updated_by` | No | Last editor username |
| `created_at` | No | RFC 3339 timestamp |
| `updated_at` | No | RFC 3339 timestamp |

### Acceptance Criteria Syntax

**Specs** use plain bullets:
```markdown
## Acceptance Criteria

- The system must validate all input
- The response should include error details
```

**Tasks** use checkboxes:
```markdown
## Acceptance Criteria

- [ ] The handler must validate credentials
- [x] The response should return a JWT token
```

Claims should use **must**, **should**, **is**, or **will** language.
