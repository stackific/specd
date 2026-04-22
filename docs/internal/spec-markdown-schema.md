# Spec Markdown Schema

This document defines the structure of spec markdown files stored in the specd project folder. These files are the **ground truth** — the SQLite cache database is derived from them and rebuilt on every command via the cache sync.

## File Location

```
<specd-folder>/specs/spec-<N>/spec.md
```

Each spec lives in its own numbered directory. The number matches the spec ID (e.g. `spec-1/spec.md` for `SPEC-1`).

## File Format

Each `spec.md` file consists of YAML frontmatter followed by a markdown body with exactly one H1 heading (the title) and one or more H2 sections.

```markdown
---
id: SPEC-1
slug: user-authentication
type: functional
summary: Implement OAuth2 login with Google and GitHub providers
position: 0
linked_specs:
  - SPEC-3
  - SPEC-5
created_by: alice
updated_by: bob
created_at: 2025-01-15T10:30:00Z
updated_at: 2025-02-01T14:00:00Z
---

# User Authentication

## Overview

Users must be able to sign in using their Google or GitHub accounts via OAuth2.

## Requirements

- Redirect to provider's consent screen
- Exchange authorization code for access token
- Create or update user record on successful login

## Acceptance Criteria

- The system must redirect users to Google's OAuth2 consent screen
- The system should support GitHub as an alternative provider
- Users may choose to stay logged in via remember-me
- The system must create new user records on first login
```

## Frontmatter Fields

The title is **NOT** in the frontmatter. It is the `# Heading` (H1) in the body.

| Field | Required | Description |
|---|---|---|
| `id` | Yes | Unique identifier, e.g. `SPEC-1`. Format: `SPEC-<number>`. |
| `slug` | Yes | Dash-separated identifier derived from the title, e.g. `user-authentication`. |
| `type` | Yes | Spec type slug from `.specd.json` `spec_types` (e.g. `business`, `functional`). |
| `summary` | Yes | One-line description. Used in search results and linking suggestions. |
| `position` | No | Integer for ordering in the specs list. Default `0`. |
| `linked_specs` | No | YAML list of spec IDs this spec is related to. Synced as bidirectional `spec_links` rows. |
| `created_by` | No | Username of the person who created the spec. |
| `updated_by` | No | Username of the person who last updated the spec. |
| `created_at` | No | RFC 3339 timestamp of when the spec was created. |
| `updated_at` | No | RFC 3339 timestamp of the last update. |

## Body Structure

The body MUST follow this structure:

- **Exactly one `# Title`** (H1) — this IS the spec title. The sync extracts it as the title field in the database. No other H1 headings are allowed.
- **`##` for top-level sections** — use H2 for major sections. H3 through H6 are fine within sections for sub-structure.
- **`## Acceptance Criteria`** (must be H2) — a special section containing claims as bullet items.

### Acceptance Criteria Claims

The `## Acceptance Criteria` section contains a bulleted list of claims. Each claim is a statement using requirement language:

- **must** — mandatory requirement, non-negotiable
- **should** — strongly recommended, expected unless justified
- **may** — optional, permitted but not required
- **might** — possible future consideration

Example:

```markdown
## Acceptance Criteria

- The system must generate a unique reset token with a 15-minute expiry
- The system should reject expired tokens with a clear error message
- Users may request a new token if the previous one expired
- The system might rate-limit reset requests to 3 per hour
```

These claims are extracted by the sync and stored in the `spec_claims` table with a dedicated FTS5 index (`spec_claims_fts`) for BM25 search. This allows searching across all acceptance criteria independently of the spec body.

**Do not** use checkbox syntax (`- [ ]`) for spec acceptance criteria. Checkboxes are reserved for task criteria only.

## Content Hash

The cache sync computes a SHA-256 hash of the **entire file** (frontmatter + body). This hash is stored in the database's `content_hash` column. Any edit triggers a sync update.

## How the Cache Sync Works

Before every non-exempt command, specd runs the cache sync:

1. Walk `<specd-folder>/specs/*/spec.md` on disk
2. Parse frontmatter, extract H1 title, extract claims from `## Acceptance Criteria`
3. Compute SHA-256 of the full file
4. Compare against the database:
   - **New on disk, missing in DB** → insert spec, sync links, sync claims
   - **Hash mismatch** → update spec, reconcile links, reconcile claims
   - **In DB but missing on disk** → delete (ON DELETE CASCADE handles links and claims)
5. FTS indexes are updated automatically via triggers

## What Gets Committed to Git

| File | Git | Description |
|---|---|---|
| `.specd.json` | Committed | Project config |
| `<specd-folder>/specs/*/spec.md` | Committed | Spec markdown files (ground truth) |
| `.specd.cache` | **Gitignored** | SQLite cache database (derived) |
| `~/.specd/` | N/A | User-level config |
