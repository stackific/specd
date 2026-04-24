# Cache Sync Workflow

specd uses a cache sync strategy where **markdown files on disk are the ground truth** and the `.specd.cache` SQLite database is a derived cache. `SyncCache()` runs automatically in `PersistentPreRunE` before every non-exempt command.

## Trigger

- **When:** Before every command except `init`, `version`, `skills`, `help`, and `logs`.
- **Where:** `cmd/root.go` → `requireProjectInit()` → `SyncCache()`.
- **Who:** Runs as the configured `username` from `~/.specd/config.json`.

## Spec Sync

1. **Read from disk** — Walk `<specd-folder>/specs/*/spec.md`. For each file:
   - Parse YAML frontmatter (id, type, summary, position, linked_specs, created_by, timestamps).
   - Extract `# Title` (H1 heading) from body — title is NOT in frontmatter.
   - Extract `## Acceptance Criteria` bullet items as claims (must/should/is/will language).
   - Compute SHA-256 of the **raw** file content (before line-ending normalization).

2. **Read from database** — Load `id → content_hash` map from `specs` table.

3. **Reconcile** — Compare disk specs against DB specs:
   - **New** (ID not in DB): Insert spec row, sync links, sync claims.
   - **Changed** (hash mismatch): Update spec row, sync links, sync claims.
   - **Unchanged** (hash matches): Skip entirely — no link or claims sync.
   - **Deleted** (ID in DB but not on disk): Delete spec row. `ON DELETE CASCADE` cleans up `spec_links`, `spec_claims`. FTS triggers clean up indexes.

4. **Link sync** (`syncSpecLinks`): Delete all outbound links for the spec, then re-insert bidirectional links from `linked_specs` frontmatter.

5. **Claims sync** (`syncSpecClaims`): Delete all existing claims for the spec, then re-insert from parsed `## Acceptance Criteria` bullets with 1-based positions.

## Task Sync

1. **Read from disk** — Walk `<specd-folder>/specs/*/TASK-*.md`. Task files live alongside their parent spec. For each file:
   - Parse YAML frontmatter (id, spec_id, status, summary, position, linked_tasks, depends_on, created_by, timestamps).
   - Extract `# Title` (H1 heading) from body.
   - Extract `## Acceptance Criteria` checkbox items (`- [ ] text` / `- [x] text`).
   - Compute SHA-256 of the **raw** file content.

2. **Read from database** — Load `id → content_hash` map from `tasks` table.

3. **Reconcile** — Compare disk tasks against DB tasks:
   - **New**: Insert task row, sync links, sync dependencies, sync criteria.
   - **Changed**: Update task row, sync links, sync dependencies, sync criteria.
   - **Unchanged**: Skip entirely.
   - **Deleted**: Delete task row. `ON DELETE CASCADE` cleans up `task_links`, `task_dependencies`, `task_criteria`.

4. **Link sync** (`syncTaskLinks`): Delete all outbound links, re-insert bidirectional links from `linked_tasks` frontmatter.

5. **Dependency sync** (`syncTaskDependencies`): Delete all inbound dependencies (`blocked_task = ?`), re-insert from `depends_on` frontmatter.

6. **Criteria sync** (`syncTaskCriteria`): Preserves checked state for criteria whose text hasn't changed. Loads existing `text → (checked, checked_by)` map, deletes all, re-inserts with preserved state where text matches.

## FTS Index Maintenance

FTS indexes (`specs_fts`, `tasks_fts`, `kb_chunks_fts`, `spec_claims_fts`) and the `search_trigram` table are maintained automatically via SQLite triggers defined in `schema.sql`. The sync code only touches base tables — triggers handle index updates on INSERT, UPDATE, and DELETE.

## What Runs When

| Event | Spec sync | Task sync | Claims inserted |
|---|---|---|---|
| `specd new-spec` | No (inserts directly) | No | Yes (at creation) |
| `specd new-task` | No | No (inserts directly) | N/A (task criteria inserted at creation) |
| `specd update-spec` | No (updates directly) | No | No (not changed by update-spec) |
| `specd update-task` | No (updates directly) | No | N/A (criteria toggled directly in DB) |
| Any other non-exempt command | Yes | Yes | Yes (if spec is new/changed) |
| Manual file edit → next command | Yes (hash mismatch detected) | Yes | Yes |
