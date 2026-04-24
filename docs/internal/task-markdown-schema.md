# Task Markdown Schema

This document defines the structure of task markdown files stored in the specd project folder. Like specs, these files are the **ground truth** — the SQLite cache database is derived from them and rebuilt on every command via the cache sync.

## File Location

```
<specd-folder>/tasks/task-<N>/task.md
```

Each task lives in its own numbered directory. The number matches the task ID (e.g. `task-1/task.md` for `TASK-1`).

## File Format

Each `task.md` file consists of YAML frontmatter followed by a markdown body with exactly one H1 heading (the title) and one or more H2 sections.

```markdown
---
id: TASK-1
slug: implement-oauth-redirect
spec_id: SPEC-1
status: backlog
summary: Implement the OAuth2 redirect flow for Google provider
position: 0
linked_tasks:
  - TASK-3
depends_on:
  - TASK-2
created_by: alice
updated_by: bob
created_at: 2025-01-15T10:30:00Z
updated_at: 2025-02-01T14:00:00Z
---

# Implement OAuth Redirect

## Overview

Build the redirect handler that sends users to Google's OAuth2 consent screen.

## Requirements

- Build the authorization URL with correct scopes
- Include a cryptographically random state parameter
- Store the state in session for CSRF verification

## Acceptance Criteria

- [ ] The redirect URL must include the correct client_id and redirect_uri
- [ ] The state parameter should be at least 32 bytes of cryptographic randomness
- [ ] The handler must store the state in the session before redirecting
```

## Frontmatter Fields

The title is **NOT** in the frontmatter. It is the `# Heading` (H1) in the body.

| Field | Required | Description |
|---|---|---|
| `id` | Yes | Unique identifier, e.g. `TASK-1`. Format: `TASK-<number>`. |
| `slug` | Yes | Dash-separated identifier derived from the title, e.g. `implement-oauth-redirect`. |
| `spec_id` | Yes | Parent spec ID that this task belongs to, e.g. `SPEC-1`. |
| `status` | Yes | Task stage slug from `.specd.json` `task_stages` (e.g. `backlog`, `todo`, `in_progress`). |
| `summary` | Yes | One-line description. Used in search results and task lists. |
| `position` | No | Integer for ordering within the same status. Default `0`. |
| `linked_tasks` | No | YAML list of task IDs this task is related to. Synced as bidirectional `task_links` rows. |
| `depends_on` | No | YAML list of task IDs that block this task. Synced as directed `task_dependencies` rows. |
| `created_by` | No | Username of the person who created the task. |
| `updated_by` | No | Username of the person who last updated the task. |
| `created_at` | No | RFC 3339 timestamp of when the task was created. |
| `updated_at` | No | RFC 3339 timestamp of the last update. |

## Body Structure

The body MUST follow this structure:

- **Exactly one `# Title`** (H1) — this IS the task title. The sync extracts it as the title field in the database. No other H1 headings are allowed.
- **`##` for top-level sections** — use H2 for major sections. H3 through H6 are fine within sections.
- **`## Acceptance Criteria`** (must be H2) — a special section containing checkbox items.

### Acceptance Criteria (Checkboxes)

The `## Acceptance Criteria` section contains a checklist of criteria. Unlike spec acceptance criteria (which use plain bullet items), task criteria use **checkbox syntax**:

- `- [ ] unchecked criterion`
- `- [x] checked criterion`

The checked state is stored as an integer (`0`/`1`) in the `task_criteria.checked` column and synced bidirectionally between the markdown file and the database.

Example:

```markdown
## Acceptance Criteria

- [ ] The handler must validate the redirect_uri against allowed origins
- [ ] The state parameter should be at least 32 bytes
- [x] The authorization URL must use HTTPS
```

**Do not** use plain bullet items (`- text`) for task criteria — always use checkbox syntax. Plain bullets are reserved for spec acceptance criteria (claims).

## Relationship to Specs

Every task belongs to exactly one spec via the `spec_id` frontmatter field. Deleting a spec cascades to all its tasks (via `ON DELETE CASCADE` in the database schema).

## Content Hash

The cache sync computes a SHA-256 hash of the **entire file** (frontmatter + body). This hash is stored in the database's `content_hash` column. Any edit triggers a sync update.

## What Gets Committed to Git

| File | Git | Description |
|---|---|---|
| `<specd-folder>/tasks/*/task.md` | Committed | Task markdown files (ground truth) |
| `.specd.cache` | **Gitignored** | SQLite cache database (derived) |
