# specd User Guide

specd is a specification-driven development CLI tool. It helps teams write, organize, and track specifications and their implementation tasks using markdown files as the source of truth.

## Quick Start

### Install

```sh
curl -fsSL https://stackific.com/specd/install.sh | bash
```

### Initialize a Project

```sh
cd your-project
specd init
```

This walks you through:

1. **Folder name** — where specd stores its files (default: `specd/`)
2. **Username** — your identity for authorship tracking
3. **Spec types** — categories for specifications (default: Business, Functional, Non-functional)
4. **Task stages** — workflow stages for tasks (default: Backlog, Todo, In progress, Done, plus optional stages)
5. **AI skills** — installs slash commands for Claude, Codex, or Gemini

After init, your project has:

```
your-project/
  .specd.json          # Project config (commit this)
  .specd.cache         # SQLite cache (gitignore this)
  specd/
    specs/             # Your specs and tasks go here
    kb/                # Knowledge base documents
```

### Create Your First Spec

Use your AI coding tool's slash command:

```
/specd-new-spec Implement user authentication with OAuth2
```

Or use the CLI directly:

```sh
specd new-spec \
  --title "User Authentication" \
  --summary "OAuth2 login with Google and GitHub" \
  --body "## Overview

Implement OAuth2 authentication.

## Acceptance Criteria

- The system must redirect to the provider's consent screen
- The system should support Google and GitHub providers
- The login flow will complete in under 3 seconds"
```

### Break a Spec into Tasks

```
/specd-new-tasks SPEC-1
```

The AI reads the spec's acceptance criteria, checks what tasks already exist, identifies gaps, and proposes new tasks to cover uncovered criteria.

### Track Progress

```
/specd-check-task TASK-1       # Toggle acceptance criteria
/specd-move-task TASK-1 done   # Change task stage
```

## Core Concepts

### Specs

A **spec** is a requirement document stored as markdown. Each spec lives in its own directory:

```
specd/specs/spec-1/spec.md
```

Specs have:
- A **title** (the `# Heading` in the body)
- A **type** (business, functional, non-functional, or custom)
- A **summary** (one-line description)
- **Acceptance criteria** (must/should/is/will claims as bullet items)
- **Links** to related specs and KB documents

### Tasks

A **task** is an implementation unit derived from a spec. Tasks live alongside their parent spec:

```
specd/specs/spec-1/TASK-1.md
specd/specs/spec-1/TASK-2.md
```

Tasks have:
- A **status** (backlog, todo, in_progress, done, etc.)
- **Acceptance criteria** with checkboxes (`- [ ]` / `- [x]`)
- **Dependencies** on other tasks (`depends_on`)
- **Links** to related tasks (`linked_tasks`)

### Ground Truth

Markdown files are always the source of truth. The `.specd.cache` SQLite database is a derived index that gets rebuilt automatically before every command. You can safely delete it — it will be recreated.

### Cache Sync

Before every command, specd walks your spec and task files, computes file hashes, and reconciles the database:
- **New files on disk** get inserted into the DB
- **Changed files** (hash mismatch) get updated
- **Deleted files** get removed from the DB
- FTS search indexes update automatically via triggers

This means you can edit spec and task files directly in your editor — specd picks up the changes.

## Writing Specs

### File Format

```markdown
---
id: SPEC-1
type: functional
summary: Implement OAuth2 login with Google and GitHub providers
position: 0
linked_specs:
  - SPEC-3
created_by: alice
created_at: 2025-01-15T10:30:00Z
updated_at: 2025-01-15T10:30:00Z
---

# User Authentication

## Overview

Users must be able to sign in using their Google or GitHub accounts.

## Requirements

- Redirect to provider's consent screen
- Exchange authorization code for access token

## Acceptance Criteria

- The system must redirect users to Google's OAuth2 consent screen
- The system should support GitHub as an alternative provider
- The login flow will complete in under 3 seconds
- The system must create new user records on first login
```

### Rules

- The title is the `# Heading` (H1) in the body — not in the frontmatter
- Only one H1 heading is allowed per file
- Use `##` for top-level sections, `###`-`######` within sections
- `## Acceptance Criteria` must be an H2 heading
- Spec criteria use plain bullets (`- `), not checkboxes
- Claims use **must**, **should**, **is**, or **will** language

## Writing Tasks

### File Format

```markdown
---
id: TASK-1
spec_id: SPEC-1
status: backlog
summary: Build the OAuth2 redirect handler
position: 0
depends_on:
  - TASK-3
created_by: alice
created_at: 2025-01-15T10:30:00Z
updated_at: 2025-01-15T10:30:00Z
---

# Implement OAuth Redirect

## Overview

Build the redirect handler that sends users to the consent screen.

## Acceptance Criteria

- [ ] The handler must build the authorization URL with correct client_id
- [x] The state parameter should be cryptographically random
- [ ] The scopes will be configurable via environment variables
```

### Rules

- Task criteria use **checkbox syntax** (`- [ ]` / `- [x]`), not plain bullets
- The `status` field must match one of the configured task stages
- `depends_on` lists task IDs that block this task
- `linked_tasks` lists related (non-blocking) task IDs

## Search

specd uses hybrid BM25 + trigram search across specs, tasks, and KB documents.

```sh
# Search everything
specd search --query "authentication"

# Search only specs
specd search --query "OAuth2 login" --kind spec

# Search only tasks
specd search --query "redirect handler" --kind task

# Limit results
specd search --query "authentication" --limit 3
```

### How Search Works

1. **BM25** (primary) — full-text search with porter stemming. Matches word variants (e.g., "authenticate" matches "authentication"). Results ranked by relevance.
2. **Trigram** (fallback) — substring matching for partial words, identifiers, and special characters. Activated when BM25 returns few results or the query has special characters.

### Search Weights

Configure how much each field matters in `.specd.json`:

```json
{
  "search_weights": {
    "title": 10.0,
    "summary": 5.0,
    "body": 1.0
  }
}
```

Higher weight = matches in that field rank higher. Default: title matches are 10x more important than body matches.

### Contradiction Detection

Search across acceptance criteria to find conflicting requirements:

```sh
specd search-claims --query "sessions should expire" --exclude SPEC-3
```

This searches all spec claims except SPEC-3's, helping you spot contradictions before they reach implementation.

## AI Skills

specd integrates with AI coding tools via the [Agent Skills Standard](https://agentskills.io/specification). Skills are slash commands that guide the AI through multi-step workflows.

### Available Skills

| Skill | Trigger | What it does |
|---|---|---|
| `/specd-new-spec` | "Add a spec for..." | Creates a spec, finds related content, checks for contradictions |
| `/specd-new-tasks` | "Create tasks for SPEC-1" | Decomposes a spec into tasks, does gap analysis first |
| `/specd-find-spec` | "Find the auth spec" | Searches by ID or keywords |
| `/specd-update-spec` | "Change SPEC-1 type to functional" | Updates type, links, KB citations |
| `/specd-delete-spec` | "Delete SPEC-3" | Removes spec and all its tasks |
| `/specd-check-task` | "Mark criterion 1 on TASK-2 as done" | Toggles acceptance criteria checkboxes |
| `/specd-move-task` | "Move TASK-1 to in progress" | Changes task stage |
| `/specd-delete-task` | "Delete TASK-5" | Removes a single task |

### Installing Skills

```sh
specd skills install
```

Select your AI providers (Claude, Codex, Gemini) and install level (user or repository).

### Supported Providers

| Provider | Skill Directory |
|---|---|
| Claude | `.claude/skills/<name>/SKILL.md` |
| OpenAI Codex | `.agents/skills/<name>/SKILL.md` |
| Gemini | `.gemini/skills/<name>/SKILL.md` |

## Configuration

### Project Config (`.specd.json`)

Created by `specd init` at the project root. Committed to version control.

```json
{
  "folder": "specd",
  "username": "",
  "spec_types": ["business", "functional", "nonfunctional"],
  "task_stages": ["backlog", "todo", "in_progress", "done", "blocked", "pending_verification", "cancelled", "wont_fix"],
  "top_search_results": 5,
  "search_weights": {
    "title": 10.0,
    "summary": 5.0,
    "body": 1.0
  }
}
```

| Field | Description |
|---|---|
| `folder` | Directory name for specd files |
| `username` | Project-specific username (overrides global) |
| `spec_types` | Allowed spec type values |
| `task_stages` | Allowed task status values |
| `top_search_results` | Max related items returned by search |
| `search_weights` | BM25 ranking weights per field |

### Global Config (`~/.specd/config.json`)

Your username, shared across all projects unless overridden.

```json
{
  "username": "alice"
}
```

## Logging

specd logs all operations to `~/.specd/specd.log` in JSON format.

```sh
# Stream logs (tail -f style)
specd logs

# Show last 50 lines
specd logs --lines 50
```

Set `SPECD_DEBUG=1` for verbose debug output.

## Web UI

```sh
specd serve
```

Opens a browser with the specd Web UI on `http://localhost:8000` (auto-finds an available port if 8000 is busy).

## Version & Updates

```sh
specd version
```

specd checks for updates automatically after each command (cached for 24 hours). If a newer version is available, it prints an upgrade notice.
