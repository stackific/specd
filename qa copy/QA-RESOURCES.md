# QA Test Resources

This folder contains sample files and instructions for manually testing specd.

## Prerequisites

1. Go 1.26+ installed
2. specd binary built: `go build -o specd ./cmd/specd`
3. A clean test workspace (create one per test session)

## Setup a Fresh Test Workspace

```bash
# Create a temporary workspace for testing
mkdir /tmp/specd-qa && cd /tmp/specd-qa

# Initialize
specd init

# Set your name (used in created_by/updated_by)
specd config user.name "QA Tester"

# Start the server + watcher
specd serve --port 7823
```

Open http://localhost:7823 in your browser.

## Resource Files

| File | Type | Purpose | Used in Test Cases |
|------|------|---------|-------------------|
| `resources/sample-kb-article.md` | Markdown | Multi-section OAuth 2.0 article (~3000 chars). Produces multiple chunks. Shares vocabulary with `sample-kb-guide.md` for TF-IDF connection testing. | 5.4.2, 6.2.x, 24.1, 24.21, 36.2 |
| `resources/sample-kb-guide.md` | Markdown | JWT implementation guide (~2500 chars). Related vocabulary to OAuth article. Has code blocks for reader rendering tests. | 5.4.2, 6.2.x, 24.1, 24.21, 36.2 |
| `resources/sample-kb-notes.txt` | Plain text | Deployment runbook in plain text (~2000 chars). Tests .txt KB reader rendering and character-offset highlighting. | 5.4.5, 6.3.x, 24.2 |
| `resources/sample-kb-page.html` | HTML | API rate limiting docs with tables, code blocks, styled content, and an embedded `<script>` tag. Tests HTML sanitization and iframe rendering. | 5.4.3, 6.4.x, 24.3, 42.7 |
| `resources/sample-kb-xss-test.html` | HTML | Document packed with XSS payloads (script tags, event handlers, javascript: URIs, SVG, meta redirect). Tests bluemonday sanitization. | 42.1-42.8 |
| `resources/sample-kb-long.txt` | Plain text | ~5000-char architecture document. Produces many chunks. Tests chunking boundaries, sidebar navigation, and chunk count display. | 6.6.x, 24.22, 24.23 |
| `resources/sample-rejection-spec.md` | Markdown | A hand-created spec file without the canonical `SPEC-N-slug` naming pattern. Drop into `specd/specs/` to test watcher rejection. | 10.x, 32.x |
| `resources/sample-rejection-task.md` | Markdown | A hand-created task file without the canonical `TASK-N-slug` naming pattern. Drop into a spec directory to test watcher rejection. | 10.x, 32.x |
| `resources/seed-workspace.sh` | Shell | Seeds a full workspace with 5 specs, 11 tasks, spec links, task links, dependencies, KB docs, citations, and partial criteria. One command to set up everything. | All sections |

## Quick Start: Seed a Full Workspace

The fastest way to get a QA-ready workspace is the seed script. It creates specs, tasks, links, dependencies, KB docs, citations, and partial criteria in one shot:

```bash
mkdir /tmp/specd-qa && cd /tmp/specd-qa
specd init
bash /path/to/specd/qa/resources/seed-workspace.sh /path/to/specd/qa/resources
specd serve --port 7823
```

This gives you:

| Entity | Count | Details |
|--------|-------|---------|
| Specs | 5 | 1 business, 2 functional, 2 non-functional |
| Tasks | 11 | Across all specs, various statuses (backlog, todo, in_progress, done, blocked) |
| Spec links | 4 | SPEC-1<->SPEC-2, SPEC-1<->SPEC-3, SPEC-3<->SPEC-5, SPEC-4<->SPEC-5 |
| Task links | 4 | TASK-2<->TASK-3, TASK-3<->TASK-4, TASK-6<->TASK-7, TASK-10<->TASK-11 |
| Dependencies | 7 | Chains: TASK-1->2->3->4, TASK-3->5, TASK-6->7, TASK-8->9, TASK-10->11 |
| KB docs | 5 | 2 markdown, 2 plain text, 1 HTML |
| Citations | 8 | Spread across 2 specs and 2 tasks |
| Criteria | Partial | TASK-1 has 2/5 checked, TASK-6 has 2/4 checked, TASK-8 has 4/4 checked (done) |

You can then run individual test sections against this pre-populated workspace, or start fresh for specific scenarios.

## How to Use the Resources

### Spec Linking Testing (Sections 3.1.7, 21, 34.6)

The seed script creates 4 spec-to-spec links. To test manually or verify:

```bash
# View links on a spec
specd read SPEC-1 --with-links
# Expected: linked to SPEC-2 (Payment) and SPEC-3 (Rate Limiting)

specd read SPEC-5 --with-links
# Expected: linked to SPEC-3 (Rate Limiting) and SPEC-4 (Data Export)
```

**Test bidirectionality** -- links go both ways:

```bash
specd read SPEC-2 --with-links
# Expected: linked to SPEC-1 (even though the link command was `specd link SPEC-1 SPEC-2`)
```

**Test additional linking scenarios:**

```bash
# Link two more specs
specd link SPEC-2 SPEC-4
specd read SPEC-2 --with-links
# Expected: SPEC-1 and SPEC-4

# Self-link (should fail)
specd link SPEC-1 SPEC-1
# Expected: error

# Cross-kind link (should fail)
specd link SPEC-1 TASK-1
# Expected: error about both needing to be same kind

# Unlink
specd unlink SPEC-2 SPEC-4
specd read SPEC-2 --with-links
# Expected: only SPEC-1 (SPEC-4 removed)

# Idempotent link (should not error)
specd link SPEC-1 SPEC-2
specd link SPEC-1 SPEC-2
# Expected: no error on second call
```

**Verify in UI:**
- Open http://localhost:7823/specs/SPEC-1 -- "Related Specs" section should list SPEC-2 and SPEC-3
- Click a linked spec -- should navigate to its detail page
- Open a spec with no links -- "Related Specs" section should be hidden

**Verify frontmatter sync:**

```bash
# Check that linked_specs field was written to the markdown file
cat specd/specs/SPEC-1-*/spec.md | head -10
# Should include: linked_specs: [SPEC-2, SPEC-3]
```

### Task Linking Testing (Sections 4.5, 21, 35)

The seed script creates 4 task-to-task links. To test manually:

```bash
# View links on a task
specd read TASK-2 --with-links
# Expected: linked to TASK-3

specd read TASK-3 --with-links
# Expected: linked to TASK-2 and TASK-4 (bidirectional)
```

**Test additional linking scenarios:**

```bash
# Link tasks across different specs
specd link TASK-1 TASK-8
specd read TASK-1 --with-links
# Expected: TASK-8

# Unlink
specd unlink TASK-1 TASK-8

# Link nonexistent task
specd link TASK-1 TASK-999
# Expected: error
```

**Verify in UI:**
- Open http://localhost:7823/tasks/TASK-3 -- "Linked Tasks" section should show TASK-2 and TASK-4 with their statuses
- Open a task with no links -- section should be hidden

### Dependency Testing (Sections 4.4, 22, 26, 38)

The seed script creates 7 dependencies forming chains and a diamond:

```
TASK-1 (schema) ─blocks─> TASK-2 (JWT) ─blocks─> TASK-3 (login) ─blocks─> TASK-4 (OAuth)
                                                       └─blocks─> TASK-5 (password reset)
TASK-6 (Stripe setup) ─blocks─> TASK-7 (subscriptions)
TASK-8 (token bucket) ─blocks─> TASK-9 (middleware)
TASK-10 (export queue) ─blocks─> TASK-11 (export API)
```

**Verify dependency readiness:**

```bash
# TASK-8 (token bucket) is "done" -- so TASK-9 should be ready
specd read TASK-9 --with-deps
# Expected: TASK-8 shown as ready (done)

# TASK-1 (schema) is "todo" -- so TASK-2 should NOT be ready
specd read TASK-2 --with-deps
# Expected: TASK-1 shown as not ready

# Check the full chain
specd read TASK-4 --with-deps
# Expected: TASK-3 shown as not ready (because TASK-3 depends on TASK-2 which depends on TASK-1)
```

**Verify `specd next` ordering:**

```bash
specd next --limit 10
# Expected order:
# 1. TASK-9 (rate limit middleware) -- ready, depends on done TASK-8
# 2. TASK-1 (design schema) -- ready (no deps), partially done (2/5 checked)
# 3. Other todo tasks that are ready but with 0% progress
# Blocked tasks should show blocked_by list
```

**Test cycle detection:**

```bash
# Try to create a cycle: TASK-1 already blocks TASK-2
specd depend TASK-1 --on TASK-2
# Expected: error with cycle path (TASK-1 -> TASK-2 -> ... -> TASK-1)

# Indirect cycle: TASK-1 blocks TASK-2 blocks TASK-3
specd depend TASK-1 --on TASK-3
# Expected: error with cycle path
```

**Test unblocking flow:**

```bash
# Complete the blocker
specd move TASK-1 --status done

# Now TASK-2 should become ready
specd read TASK-2 --with-deps
# Expected: TASK-1 shown as ready (done)

specd next --limit 5
# Expected: TASK-2 now appears as ready
```

**Verify in UI:**
- Open http://localhost:7823/tasks/TASK-2 -- "Dependencies" section shows TASK-1 with ready/blocked icon
- Ready deps show green check_circle icon
- Not-ready deps show red block icon
- Clicking a dependency link navigates to the blocker task
- On the board (http://localhost:7823/), tasks with unresolved deps show a warning icon on the card

### Combined Link + Dependency Verification

A task can have both links (related tasks) AND dependencies (blockers). These are different concepts:

```bash
# TASK-3 has links (TASK-2 and TASK-4) AND a dependency (blocks on TASK-2)
specd read TASK-3 --with-links --with-deps
# Expected:
#   Links: TASK-2, TASK-4
#   Dependencies: TASK-2 (not ready)
```

In the UI, both sections render independently on the task detail page.

### KB Document Testing (Sections 5, 6, 14, 24)

Add all KB resources to your test workspace:

```bash
# From the specd-qa workspace directory
specd kb add /path/to/specd/qa/resources/sample-kb-article.md --title "OAuth 2.0 RFC Guide"
specd kb add /path/to/specd/qa/resources/sample-kb-guide.md --title "JWT Best Practices"
specd kb add /path/to/specd/qa/resources/sample-kb-notes.txt --title "Deployment Runbook"
specd kb add /path/to/specd/qa/resources/sample-kb-page.html --title "API Rate Limiting"
specd kb add /path/to/specd/qa/resources/sample-kb-long.txt --title "Architecture Overview"
```

Or upload them via the web UI at http://localhost:7823/kb (click "Add Document").

Verify:
- `specd kb list` shows all 5 documents with correct types (md, txt, html)
- `specd kb connections KB-1` shows TF-IDF connections (OAuth and JWT docs should connect)
- Open each in the KB reader to test rendering per source type

### XSS / Security Testing (Section 42)

```bash
# Add the XSS test document
specd kb add /path/to/specd/qa/resources/sample-kb-xss-test.html --title "XSS Test"
```

Then open the KB reader in the browser. Verify:
- No alert dialogs appear (scripts stripped)
- Safe content (paragraphs, bold, italic, lists) is preserved
- The HTML renders inside a sandboxed iframe
- View the `.clean.html` sidecar to confirm script tags are removed

### Watcher Rejection Testing (Sections 10, 32)

With the server running (watcher active):

```bash
# Test non-canonical spec rejection
cp /path/to/specd/qa/resources/sample-rejection-spec.md /tmp/specd-qa/specd/specs/my-spec.md

# Test non-canonical task rejection (requires an existing spec directory)
cp /path/to/specd/qa/resources/sample-rejection-task.md /tmp/specd-qa/specd/specs/SPEC-1-*/my-task.md
```

Wait 1 second, then check:
- http://localhost:7823/rejected should show the rejected files
- `specd status` should report rejected file count
- The original files remain on disk (not deleted)

### TF-IDF Connection Testing (Sections 6.7, 24.21, 36.2)

After adding both `sample-kb-article.md` (OAuth) and `sample-kb-guide.md` (JWT):

```bash
specd kb connections KB-1
specd kb connections KB-2
```

Both should show connections to each other since they share authentication/security vocabulary. Open the KB reader for either document and check the "Related Chunks" section.

### Citation Testing (Sections 3.1.9, 4.6, 37)

After adding KB documents and creating specs/tasks:

```bash
# Find chunks to cite
specd kb read KB-1 --chunk 0

# Cite from a spec
specd cite SPEC-1 KB-1:0 KB-2:3

# Cite from a task
specd cite TASK-1 KB-1:2
```

Verify citation cards appear on spec/task detail pages with type icon, title, chunk preview, and "View in source" link.

### Search Testing (Sections 7, 25, 39)

After populating specs, tasks, and KB documents:

```bash
# Should find results across all kinds
specd search "authentication" --kind all

# Should trigger trigram fallback
specd search "authen"

# Phrase search
specd search '"access token"'
```

Also test via the web UI at http://localhost:7823/search.

### Criteria Testing (Sections 4.3, 20, 35.3)

The seed script creates tasks with criteria already embedded in the body. To test additional scenarios:

```bash
# Add a criterion to an existing task
specd criteria add TASK-9 "Verify middleware returns correct headers"
specd criteria list TASK-9
# Expected: new criterion at next position

# Check / uncheck
specd criteria check TASK-9 1
specd criteria uncheck TASK-9 1

# Remove middle criterion (should renumber remaining)
specd criteria add TASK-9 "Second criterion for removal test"
specd criteria add TASK-9 "Third criterion"
specd criteria remove TASK-9 2
specd criteria list TASK-9
# Expected: positions 1, 2 (renumbered from 1, 3)
```

**Verify in UI:**
- Open http://localhost:7823/tasks/TASK-1 -- criteria list shows 5 items, first 2 checked
- Click a checkbox -- criterion toggles, page refreshes
- Add a criterion via the form below the list
- Remove via the X button next to a criterion
- Check that the board card for TASK-1 shows progress indicator (2/5)

**Verify markdown round-trip:**

```bash
cat specd/specs/SPEC-1-*/TASK-1-*.md
# Should show: - [x] for checked, - [ ] for unchecked
```

### Trash and Restore Testing (Sections 9, 28)

```bash
# Delete a spec (cascades its tasks)
specd delete SPEC-4

# Verify cascade
specd list tasks --spec-id SPEC-4
# Expected: empty (tasks deleted)

specd trash list
# Expected: SPEC-4 + TASK-10 + TASK-11 in trash

# Restore
specd trash restore 1   # (use the trash ID from the list output)

# Test restore with ID conflict
specd delete SPEC-4
specd new-spec --title "Replacement Spec" --type functional --summary "Took SPEC-4 slot" --body "This spec may reuse the ID slot that was freed up."
specd trash restore <id>
# Expected: restored with NEW ID (e.g., SPEC-6), warning about conflict

# Purge
specd trash purge-all
specd trash list
# Expected: empty
```

### Candidates Testing (Sections 18.36-18.38)

After seeding the workspace:

```bash
# Candidates for a spec (finds related specs and KB chunks)
specd candidates SPEC-1 --limit 10
# Expected: SPEC-3 (Rate Limiting) ranked high (shares "authentication" vocabulary)
# SPEC-2 (Payment) may appear (linked to auth via business logic)
# Already-linked specs should be EXCLUDED from results
# KB chunks from OAuth/JWT docs should appear in kb_chunks section

# Candidates for a task
specd candidates TASK-2 --limit 10
# Expected: related tasks ranked by word overlap, self excluded

# Candidates with limit
specd candidates SPEC-1 --limit 2
# Expected: max 2 results per section
```

**Verify in practice:**
1. Create a new spec without linking it, then run candidates -- it should suggest related specs
2. Link it, re-run candidates -- linked spec should disappear from results

### Merge-Fixup Testing (Sections 27.5, 40.4)

To test merge-fixup, you need to simulate duplicate IDs (as would happen after a git merge conflict):

```bash
# Start fresh
mkdir /tmp/specd-fixup && cd /tmp/specd-fixup
specd init
specd new-spec --title "First Spec" --type functional --summary "First" --body "Body content that meets the minimum length requirement."

# Manually create a duplicate SPEC-1 directory (simulating a merge)
mkdir -p specd/specs/SPEC-1-duplicate-from-merge
cat > specd/specs/SPEC-1-duplicate-from-merge/spec.md << 'EOF'
---
title: Duplicate From Merge
type: business
summary: This simulates a merge collision
---

# Duplicate From Merge

This spec has the same ID number as the first one.
EOF

# Run merge-fixup
specd merge-fixup --json
# Expected:
# - duplicate_specs: [{id: "SPEC-1", paths: [..., ...]}]
# - renumbered: [{old_id: "SPEC-1", new_id: "SPEC-2", ...}]
# - Rebuild triggered automatically after fixup

# Verify
specd list specs
# Expected: two specs, SPEC-1 and SPEC-2, both accessible

# Same approach for tasks (duplicate TASK files) and KB (duplicate KB files)
```

### Lint Issue Creation Testing (Section 27.1)

Create specific issues to verify lint detects them:

```bash
# Start with a seeded workspace, then:

# 1. Orphan spec (no tasks, no links)
specd new-spec --title "Orphan" --type functional --summary "No links or tasks" --body "This spec has no connections to anything else in the system."

# 2. Missing summary
specd new-spec --title "Bad Summary" --type functional --summary "X" --body "This spec has a trivially short summary that lint should flag."

# 3. Dangling link: link two specs, delete one
specd new-spec --title "Link Target" --type functional --summary "Will be deleted" --body "This spec exists only to be linked then deleted for testing."
specd link SPEC-1 SPEC-7  # (adjust ID as needed)
# Now manually delete the spec directory (bypass trash)
rm -rf specd/specs/SPEC-7-*/
# Watcher will trash the DB row, but the link from SPEC-1 remains dangling

# 4. Stale tidy: set last_tidy_at to 10 days ago
# (This requires waiting or manually editing the DB -- skip for manual QA)

# 5. Rejected file
cp qa/resources/sample-rejection-spec.md specd/specs/bad-file.md

# Now run lint
specd lint --json
# Expected: issues for orphan_spec, missing_summary, rejected_files, and possibly dangling_link
```

### Rebuild Verification (Sections 27.3, 40.1)

After seeding a full workspace:

```bash
# Record current state
specd status --json > /tmp/before-rebuild.json
specd list specs --json > /tmp/specs-before.json
specd list tasks --json > /tmp/tasks-before.json

# Destroy the cache
rm .specd/cache.db

# Rebuild
specd rebuild --json

# Compare
specd status --json > /tmp/after-rebuild.json
diff /tmp/before-rebuild.json /tmp/after-rebuild.json

# Verify specific data survived:
specd read SPEC-1 --with-links --with-progress --with-citations --json
# Expected: links, progress, citations all present

specd read TASK-2 --with-deps --with-criteria --with-links --with-citations --json
# Expected: deps, criteria (with checked state), links, citations all present

specd search "authentication" --json
# Expected: FTS and trigram indexes functional

specd next --limit 10 --json
# Expected: correct ordering with dependencies
```

### Performance Testing (Section 41)

Generate a large workspace for performance benchmarks:

```bash
mkdir /tmp/specd-perf && cd /tmp/specd-perf
specd init

# Create 100 specs
for i in $(seq 1 100); do
  specd new-spec --title "Spec $i performance test" --type functional \
    --summary "Performance testing spec number $i" \
    --body "Body content for spec $i. This needs to be long enough to pass the validation requirement of twenty characters minimum."
done

# Create 10 tasks per spec (1000 total)
for spec in $(seq 1 100); do
  for task in $(seq 1 10); do
    specd new-task --spec-id "SPEC-$spec" --title "Task $task for spec $spec" \
      --summary "Task $task under spec $spec" --status backlog \
      --body "Task body for spec $spec task $task. This is long enough to pass validation. It includes meaningful content for search testing."
  done
done

# Add KB docs
for f in /path/to/specd/qa/resources/sample-kb-*.md /path/to/specd/qa/resources/sample-kb-*.txt /path/to/specd/qa/resources/sample-kb-page.html; do
  specd kb add "$f"
done

# Now benchmark:
time specd serve &  # Start server, measure load time
time curl -s http://localhost:7823/ > /dev/null        # Board: expect <500ms
time specd search "performance" --json > /dev/null      # Search: expect <100ms
time specd next --limit 10 --json > /dev/null           # Next: expect <50ms
time specd lint --json > /dev/null                      # Lint: expect <1s
kill %1
```

### Full Integration Walkthrough

For integration test cases (Sections 34-40), follow this sequence:

1. **Init** a fresh workspace
2. **Create specs**: Create 3+ specs (1 business, 1 functional, 1 non-functional) via CLI or UI
3. **Add KB docs**: Add all sample resources as KB documents
4. **Create tasks**: Create 5+ tasks under different specs with varying statuses
5. **Add criteria**: Add acceptance criteria to 2-3 tasks, check some items
6. **Link specs**: Link related specs together, verify bidirectionality
7. **Link tasks**: Link related tasks, verify bidirectionality and frontmatter sync
8. **Add dependencies**: Create a dependency chain (A blocks B blocks C), test cycle rejection
9. **Cite KB chunks**: Cite KB chunks from specs and tasks
10. **Search**: Search for terms that appear across specs, tasks, and KB
11. **Next**: Run `specd next` to verify ready/blocked ordering with deps and criteria progress
12. **Move tasks**: Complete blockers, verify dependents become ready
13. **Delete + Restore**: Delete a spec, check trash cascade, restore it
14. **Rebuild**: Delete `.specd/cache.db`, run `specd rebuild`, verify ALL data restored (links, deps, criteria, citations)
15. **Lint**: Run `specd lint` to check for any issues

Or simply run the seed script and start testing from step 10 onward.

## PDF Testing

For PDF-specific tests (Sections 6.5, 24.4), you need a real PDF file. Use any multi-page PDF you have available, or generate one:

```bash
# If you have pandoc installed:
pandoc /path/to/specd/qa/resources/sample-kb-long.txt -o /tmp/test-document.pdf

# Then add to KB:
specd kb add /tmp/test-document.pdf --title "Test PDF Document"
```

Verify:
- Page count shown in KB detail header
- PDF renders via PDF.js in the reader
- Text layer allows chunk highlighting
- Chunk navigation across pages works

## Tips

- Keep a terminal open with `specd serve` running while testing the UI
- Use browser DevTools Network tab for sections 14, 15, and 46 (API, assets, offline)
- Use browser DevTools responsive mode for section 45 (responsive layout)
- Use browser DevTools Accessibility panel for section 43
- To test the watcher (sections 30-33), edit files directly in `specd/specs/` with a text editor while the server is running
- For performance testing (section 41), create many specs/tasks via CLI in a loop:
  ```bash
  for i in $(seq 1 100); do
    specd new-spec --title "Spec $i" --type functional --summary "Test spec $i" --body "Body content for spec number $i that is long enough to pass validation."
  done
  ```
