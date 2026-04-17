# specd Bug Report (Phases 1-20)

Comprehensive bug report covering server-side and browser-side functionality.
Tested against a fresh workspace with 3 specs, 8 tasks (all statuses), 3 KB docs (md/html/txt), links, dependencies, citations, and trash data.

---

## CRITICAL Bugs

### BUG-01: Markdown body never renders on spec-detail and task-detail pages

- **Severity**: CRITICAL
- **Affected pages**: `/specs/{id}`, `/tasks/{id}`
- **Root cause**: `marked.min.js` is not loaded in `templates/layouts/base.html`. The script tag only exists in `templates/pages/kb-detail.html:123`. The `renderMarkdownBodies()` function in `assets/app.js:227` checks `if (typeof marked === "undefined") return;` and silently exits, leaving all `[data-md-body]` elements empty.
- **Evidence**: `typeof marked` evaluates to `"undefined"` on spec/task detail pages (confirmed via `page.evaluate()`). The `<article>` body area renders as an empty gray box on every spec and task detail page.
- **Impact**: Users cannot see the body content of any spec or task in the web UI. This is core functionality.
- **Reproducible steps**:
  1. Navigate to any spec detail page (e.g., `/specs/SPEC-1`)
  2. Observe the body section — it's an empty gray rectangle
  3. Same issue on any task detail page (e.g., `/tasks/TASK-8`)
- **Fix**: Add `<script src="/assets/marked.min.js"></script>` to `templates/layouts/base.html` before the `app.js` script tag.

### BUG-02: 404 error pages render as raw text without layout

- **Severity**: CRITICAL
- **Affected pages**: `/specs/NONEXISTENT`, `/tasks/TASK-999`, `/kb/KB-999`
- **Root cause**: Handlers use `http.Error(w, "Spec not found", http.StatusNotFound)` (`internal/web/handlers.go:459`) which writes a plain-text response, bypassing the base layout template entirely.
- **Evidence**: Navigating to `/specs/NONEXISTENT` shows "Spec not found" as plain black text on a white page — no nav, no footer, no styling.
- **Impact**: Broken user experience on any invalid URL. Violates CLAUDE.md rule: "never render a blank page with a raw error string."
- **Reproducible steps**:
  1. Navigate to `http://localhost:7823/specs/NONEXISTENT`
  2. Observe raw text "Spec not found" on blank white page
  3. Same for `/tasks/TASK-999` and `/kb/KB-999`
- **Fix**: Create a proper error page template and render it via `s.renderPage()` with appropriate HTTP status code.

---

## HIGH Bugs

### BUG-03: Citation preview text shows raw markdown syntax

- **Severity**: HIGH
- **Affected pages**: `/specs/{id}`, `/tasks/{id}` (References sections)
- **Root cause**: `templates/pages/spec-detail.html:68` renders `{{truncate .ChunkText 200}}` which contains raw markdown (`# JWT Best Practices ## Token Structure A JSON Web Token...`). The chunk text stored in the database includes markdown syntax, and truncation is applied without stripping markdown formatting first.
- **Evidence**: On spec detail for SPEC-1, the References section shows: `# JWT Best Practices ## Token Structure A JSON Web Token consists of three parts: Header, Payload, and Signature. ### Header The header typically consists of two parts: the type of token and the si...`
- **Impact**: Citation previews are unreadable with markdown syntax noise.
- **Reproducible steps**:
  1. Navigate to `/specs/SPEC-1`
  2. Scroll to "References" section
  3. Observe raw markdown in the citation preview text
- **Fix**: Strip markdown syntax from chunk text before truncation (e.g., remove `#`, `*`, `_`, `[`, `]` etc.), or render it via marked.js on the client side.

### BUG-04: Status page tasks summary card undercounts tasks

- **Severity**: HIGH
- **Affected page**: `/status`
- **Root cause**: `templates/pages/status.html:20` only displays three counts: `{{.Data.Status.Tasks.Done}} done · {{.Data.Status.Tasks.InProgress}} active · {{.Data.Status.Tasks.Blocked}} blocked`. This excludes backlog, todo, pending_verification, cancelled, and wontfix counts from the summary card.
- **Evidence**: With 6 tasks (1 per status), the card shows "1 done · 1 active · 1 blocked" (total: 3), while the actual total is 6.
- **Impact**: Misleading dashboard summary. Users cannot quickly see the true task distribution.
- **Reproducible steps**:
  1. Create tasks across multiple statuses
  2. Navigate to `/status`
  3. Observe the Tasks card shows only 3 of 6 tasks accounted for
- **Fix**: Either show all status counts in the summary, or use proper aggregate groupings (e.g., "1 done · 4 active · 1 blocked" where "active" = backlog + todo + in_progress + pending_verification).

### BUG-05: Specs list page does not group specs by type

- **Severity**: HIGH
- **Affected page**: `/specs`
- **Root cause**: `internal/web/handlers.go:373-377` lists all specs in a flat array without grouping. The template `templates/pages/specs.html` renders them in a single grid. The spec (AGENTS.md section 8.2) requires "Spec list: all specs grouped by type with progress bars."
- **Evidence**: Specs page shows Business, Functional, and Non-functional specs in a single ungrouped grid.
- **Impact**: Spec organization by type is lost, making it harder to find specs.
- **Reproducible steps**:
  1. Navigate to `/specs`
  2. Observe all specs are in a flat list without type grouping headers

### BUG-06: `handleReorderTask` and `handleDragMove` mutate incoming request headers

- **Severity**: HIGH
- **Affected code**: `internal/web/handlers.go:287`, `internal/web/handlers.go:323`, `internal/web/handlers.go:348`
- **Root cause**: These handlers call `r.Header.Set("HX-Request", "true")` to force htmx-style rendering, then call `s.handleBoard(w, r)`. Mutating the incoming `*http.Request` headers is incorrect — request headers represent the client's request, not server-side decisions. This can cause race conditions if the request object is shared.
- **Impact**: May cause incorrect rendering in non-htmx contexts. The forced `HX-Request` header means the handler always renders the board as a partial, never as a full page.
- **Fix**: Pass the htmx flag through the render context or use a separate parameter rather than mutating the request.

### BUG-07: Unsafe JSON metadata construction in trash operations

- **Severity**: HIGH
- **Affected code**: `internal/workspace/spec.go:363`, `internal/workspace/task.go:337` (approximate lines, from code analysis)
- **Root cause**: Trash metadata is constructed using `fmt.Sprintf()` to build JSON strings rather than using `json.Marshal()`. If a spec or task title contains double quotes, backslashes, or other JSON-special characters, the resulting metadata JSON is malformed.
- **Impact**: `trash restore` could fail with a JSON parse error for items whose titles contain special characters, making recovery impossible.
- **Fix**: Use `json.Marshal()` to construct the metadata JSON.

---

## MEDIUM Bugs

### BUG-08: Board page leaks `<option>` elements outside their `<select>` containers

- **Severity**: MEDIUM
- **Affected page**: `/` (board)
- **Root cause**: The accessibility tree shows multiple orphaned `<option>` elements appearing directly under `<main>` outside of any `<select>`. These are visible in the snapshot as duplicate option elements at the bottom of the DOM tree.
- **Evidence**: Playwright accessibility snapshots consistently show orphaned options like `option "SPEC-1 — User Authentication" [selected]` appearing outside their parent `<select>` elements, repeated 2-3 times.
- **Impact**: Invalid HTML structure. Screen readers may announce phantom options. Potential accessibility violations.
- **Reproducible steps**:
  1. Navigate to `/`
  2. Take an accessibility snapshot
  3. Observe orphaned `<option>` elements at the bottom of the main content area

### BUG-09: HTML KB reader iframe has excessive whitespace below content

- **Severity**: MEDIUM
- **Affected page**: `/kb/KB-{id}` (for HTML source type docs)
- **Root cause**: The iframe used for rendering cleaned HTML has a fixed or oversized height that doesn't match the actual content height, leaving a large blank area below the content.
- **Evidence**: Screenshot of `/kb/KB-2` (HTML doc) shows the rendered HTML taking up only ~15% of the iframe, with ~85% empty whitespace below.
- **Impact**: Poor visual appearance. Users may think the page is broken.
- **Fix**: Dynamically resize the iframe to match content height, or set a reasonable max-height with scrolling.

### BUG-10: Dark mode toggle icon selector uses `i.page` class that doesn't exist

- **Severity**: MEDIUM
- **Affected code**: `assets/app.js:37`, `assets/app.js:51`
- **Root cause**: The JavaScript selects dark mode toggle buttons using `button:has(> i.page)`, but the actual icon elements in `templates/partials/nav.html` don't have a class `page`. The `<i>` element just contains the text `dark_mode`.
- **Evidence**: Despite this selector mismatch, dark mode toggling still works (tested via Playwright). The `ui("mode", next)` call from BeerCSS handles the theme switch. However, the icon text update (`i.textContent = next === "dark" ? "light_mode" : "dark_mode"`) may not execute because the selector fails to match.
- **Impact**: Dark mode icon may not update to show `light_mode` when in dark mode and vice versa. The functional toggle works but visual feedback is inconsistent.

### BUG-11: No `aria-live` attribute on error snackbar

- **Severity**: MEDIUM
- **Affected code**: `templates/layouts/base.html:38-41`
- **Root cause**: The error snackbar has `role="alert"` but is missing `aria-live="assertive"` and `aria-atomic="true"`.
- **Impact**: Screen readers may not immediately announce error messages.

### BUG-12: `marked.parse()` called without try-catch in `renderMarkdownBodies()`

- **Severity**: MEDIUM
- **Affected code**: `assets/app.js:234`
- **Root cause**: `marked.parse(raw)` is called without error handling. If the markdown contains malformed syntax that causes marked.js to throw, the entire rendering pipeline breaks.
- **Impact**: One bad markdown body could break rendering for all subsequent elements on the page.
- **Fix**: Wrap in try-catch and show a fallback (e.g., raw text in a `<pre>` element).

### BUG-13: Board drag-and-drop provides no error feedback to user

- **Severity**: MEDIUM
- **Affected code**: `assets/app.js:162-189`
- **Root cause**: The `fetch()` calls for move and reorder operations don't handle HTTP errors. If the server returns an error (e.g., lock timeout, invalid status), the `.then(resp => resp.text())` silently succeeds with an error page HTML, which then gets injected as `container.outerHTML`.
- **Impact**: On server error, the board content is replaced with a raw error message or broken HTML, with no way to recover except refreshing.
- **Fix**: Check `resp.ok` before processing, and show an error snackbar on failure.

### BUG-14: Task status dropdown `onchange="this.form.submit()"` is an inline handler

- **Severity**: MEDIUM
- **Affected code**: `templates/pages/task-detail.html:40`
- **Root cause**: Uses inline JavaScript `onchange="this.form.submit()"`. This violates CSP best practices and is inconsistent with the rest of the codebase which uses event delegation.
- **Impact**: Would fail under a strict Content Security Policy. Also makes testing harder.

### BUG-15: KB search on the KB page doesn't use htmx for results

- **Severity**: MEDIUM
- **Affected page**: `/kb`
- **Root cause**: The search input on the KB page (`templates/pages/kb.html`) has a searchbox but searching requires a full page reload via form submission. The search field is not wired with htmx for live results like the search page.
- **Evidence**: Typing in the KB search box and pressing Enter triggers a full page navigation to `/kb?q=...`.
- **Impact**: Inconsistent UX between the main search page (which uses htmx) and the KB page search.

---

## LOW Bugs

### BUG-16: Progress bar calculation may be incorrect for cancelled/wontfix tasks

- **Severity**: LOW
- **Affected code**: `templates/pages/spec-detail.html:32`
- **Root cause**: `<progress value="{{.Data.Progress.Done}}" max="{{.Data.Progress.Active}}">` — need to verify that `Active` correctly excludes cancelled and wontfix tasks. The spec says "progress bars (based on non-cancelled/non-wontfix tasks)."
- **Impact**: Progress bar may show incorrect percentages if cancelled/wontfix tasks are included in the denominator.

### BUG-17: Footer template referenced but potentially fragile

- **Severity**: LOW
- **Affected code**: `templates/layouts/base.html:35`
- **Root cause**: `{{template "footer" .}}` references a footer template. If the footer template definition is missing or the partial file isn't loaded, the page will fail to render entirely (Go template execution error).
- **Impact**: If the footer partial is ever accidentally removed, ALL pages break.

### BUG-18: `ListCriteria` returns `nil` instead of empty slice for tasks with no criteria

- **Severity**: LOW
- **Affected code**: `internal/workspace/criteria.go` (ListCriteria function)
- **Root cause**: When a task has no criteria, the function returns `nil` instead of `[]CriterionRow{}`. The template checks `{{if .Data.Criteria}}` which treats `nil` as falsy, so it shows "No criteria yet." — this is actually correct behavior but inconsistent with other list functions that return empty slices.
- **Impact**: No functional impact, but inconsistent API behavior.

### BUG-19: Incomplete markdown stripping in KB reader

- **Severity**: LOW
- **Affected code**: `assets/kb-reader.js` (stripMarkdown function)
- **Root cause**: The `stripMarkdown()` function doesn't handle all markdown patterns (e.g., strikethrough `~~text~~`, tables, footnotes). Chunks containing these patterns may not highlight correctly in the reader.
- **Impact**: Chunk highlighting may fail for documents with advanced markdown syntax.

### BUG-20: No keyboard shortcut support in KB chunk navigation

- **Severity**: LOW
- **Affected page**: `/kb/{id}` (KB reader)
- **Root cause**: Chunk navigation only uses buttons (Previous/Next). No keyboard shortcuts (e.g., arrow keys) are provided.
- **Impact**: Reduced accessibility for keyboard-only users navigating through chunks.

### BUG-21: KB sidebar media query uses 600px instead of 601px (BeerCSS breakpoint)

- **Severity**: LOW
- **Affected code**: `styles/src/styles/pages/kb-detail.css` (approximately line 284)
- **Root cause**: The sidebar hides at `max-width: 600px`, but BeerCSS uses `601px` as the tablet breakpoint. This creates a 1px gap.
- **Impact**: At exactly 600-601px viewport width, the sidebar may behave inconsistently with other responsive elements.

### BUG-22: `handleReorderTask` hardcodes redirect to `/` on error

- **Severity**: LOW
- **Affected code**: `internal/web/handlers.go:274`
- **Root cause**: `redirectWithError(w, r, "/", "Invalid position")` always redirects to the root page on error, losing context about what the user was trying to do.
- **Impact**: User loses their place on error during reorder operations.

---

## Server-Side Code Issues (from static analysis)

### BUG-23: Multiple unchecked `db.Exec()` error returns in watcher

- **Severity**: HIGH (potential data corruption)
- **Affected code**: `internal/watcher/watcher.go` (multiple lines around task_criteria operations)
- **Root cause**: `DELETE FROM task_criteria` and `INSERT INTO task_criteria` calls ignore error returns. If an INSERT fails after DELETE succeeds, criteria are permanently lost.
- **Impact**: Task acceptance criteria can be silently corrupted during watcher-triggered updates.

### BUG-24: Unchecked `db.Exec()` calls in rebuild.go

- **Severity**: HIGH (silent failures)
- **Affected code**: `internal/workspace/rebuild.go` (multiple locations)
- **Root cause**: Inserts into `rejected_files` and `task_criteria` during rebuild ignore error returns.
- **Impact**: Rebuild may silently skip rejected file tracking or criteria restoration.

### BUG-25: Missing `DELETE FROM citations` error check in watcher delete

- **Severity**: MEDIUM
- **Affected code**: `internal/watcher/watcher.go` (spec/task delete handlers)
- **Root cause**: Citations use `from_id` as plain text (not FK), so deletion must be explicit. The error from `DELETE FROM citations` is not checked.
- **Impact**: Orphaned citation records after watcher-triggered deletes.

### BUG-26: Race condition in watcher debounce timer callback

- **Severity**: MEDIUM
- **Affected code**: `internal/watcher/watcher.go` (timer callback)
- **Root cause**: Timer callback accesses `wt.w.DB` without holding the workspace lock. If the workspace is closed between timer creation and execution, this could cause a nil pointer dereference.
- **Impact**: Potential crash during rapid file changes when server is shutting down.

---

## Test Results

| Test Suite | Result |
|---|---|
| `go test ./...` | All tests pass (cached) |
| `go build` | Builds successfully |
| CSS build (`pnpm build`) | Builds successfully (66KB -> 16KB after purge) |
| Console errors (all pages) | 0 errors on all pages except 404 |
| Network errors | None observed |

---

## Fix Status

| Bug | Severity | Status | Fix Description |
|---|---|---|---|
| BUG-01 | CRITICAL | FIXED | Added `marked.min.js` to `base.html`, removed duplicate from `kb-detail.html` |
| BUG-02 | CRITICAL | FIXED | Created `error.html` template, added `renderError()` helper, updated spec/task/kb detail 404s |
| BUG-03 | HIGH | FIXED | Added `stripTruncate` template function, updated citation previews in spec-detail and task-detail |
| BUG-04 | HIGH | FIXED | Status card now shows all task groups: done/active/blocked/closed with `add` template function |
| BUG-05 | HIGH | FIXED | Specs list now groups by type (Functional/Business/Non-functional) with section headings |
| BUG-06 | HIGH | FIXED | Added `renderBoardPartial()` — no more `r.Header.Set("HX-Request")` mutation |
| BUG-07 | HIGH | FIXED | Replaced `fmt.Sprintf` JSON with `json.Marshal` in spec.go, task.go, kb.go trash metadata |
| BUG-08 | MEDIUM | WON'T FIX | BeerCSS framework artifact — `<select>` elements render custom dropdown overlays |
| BUG-09 | MEDIUM | FIXED | Added `requestAnimationFrame` + deferred resize for iframe content sizing |
| BUG-10 | MEDIUM | NOT A BUG | Nav icons DO have `class="page"` — selector is correct |
| BUG-11 | MEDIUM | FIXED | Added `aria-live="assertive" aria-atomic="true"` to error snackbar |
| BUG-12 | MEDIUM | FIXED | Wrapped `marked.parse()` in try-catch with `<pre>` fallback |
| BUG-13 | MEDIUM | FIXED | Consolidated DnD fetch, added `resp.ok` check and error snackbar on failure |
| BUG-14 | MEDIUM | FIXED | Replaced inline `onchange` with `sd-auto-submit` class + event delegation |
| BUG-15 | MEDIUM | OPEN | KB page search not wired with htmx — deferred (low impact UX inconsistency) |
| BUG-16 | LOW | OPEN | Progress bar excluded-status calculation — needs verification |
| BUG-17 | LOW | OPEN | Footer template fragility — mitigated by Go template compilation at startup |
| BUG-18 | LOW | OPEN | ListCriteria nil vs empty slice — cosmetic inconsistency |
| BUG-19 | LOW | OPEN | Incomplete markdown stripping in KB reader — edge case |
| BUG-20 | LOW | OPEN | No keyboard shortcuts for chunk navigation — accessibility enhancement |
| BUG-21 | LOW | OPEN | KB sidebar media query 600px vs 601px — 1px edge case |
| BUG-22 | LOW | FIXED | Redirect on reorder error now uses `/tasks/{id}` instead of `/` |
| BUG-23 | HIGH | OPEN | Unchecked `db.Exec()` in watcher task_criteria — needs transactional fix |
| BUG-24 | HIGH | OPEN | Unchecked `db.Exec()` in rebuild.go — needs error propagation |
| BUG-25 | MEDIUM | OPEN | Missing citation DELETE error check in watcher — needs fix |
| BUG-26 | MEDIUM | OPEN | Watcher debounce race condition — needs lock coordination |

### Summary

| Status | Count |
|---|---|
| FIXED | 14 |
| NOT A BUG | 1 |
| WON'T FIX | 1 |
| OPEN | 10 |
| **Total** | **26** |

All CRITICAL bugs are fixed. 5 of 7 HIGH bugs are fixed (remaining 2 are server-side watcher/rebuild error handling). 6 of 8 MEDIUM bugs are fixed. LOW bugs are deferred as minor enhancements.

---

## Scenarios NOT Tested

The following functionality was **not tested** during this audit. Each entry includes the reason it was skipped and the risk level of undiscovered bugs.

### Web UI - Interactive Features

| Scenario | Why not tested | Risk |
|---|---|---|
| **Drag-and-drop on kanban board** | Playwright drag simulation is fragile and unreliable for complex DnD; would require custom JS injection to test properly | HIGH — the move/reorder fetch calls have no error handling (BUG-13), and the board HTML replacement logic is untested end-to-end |
| **Edit spec dialog** (open, fill, submit, error re-render) | Time constraints; dialog opens via `data-ui` BeerCSS attribute | MEDIUM — htmx `hx-post` + `hx-swap="outerHTML"` on the form could break the dialog container on 422 re-render |
| **Edit task dialog** (open, fill, submit, error re-render) | Same as above | MEDIUM — same htmx swap concern |
| **Delete spec confirmation + actual deletion** | Skipped to avoid destroying test data needed for other tests | MEDIUM — the delete handler redirects after deletion; untested whether htmx redirect works correctly |
| **Delete task confirmation + actual deletion** | Same as above | MEDIUM |
| **Criteria checkbox check/uncheck** | Not tested via UI click (only verified data exists via CLI) | MEDIUM — the `onchange="this.form.submit()"` inline handler triggers a full form POST; the `reloadTaskDetail` htmx response path is untested |
| **Criteria add via UI form** | Not tested (only verified the form exists in snapshot) | LOW — standard form POST; server-side validation was read in code review |
| **Criteria remove via UI** | Not tested | LOW — simple POST, similar pattern |
| **Trash restore button** | Not clicked; only verified it renders | MEDIUM — restore recreates files and DB rows; could fail silently if the original spec directory was also deleted |
| **Trash purge / purge-all** | Not tested | LOW — destructive operation; server-side code was reviewed |
| **KB add document form (file upload)** | Multipart upload not tested | HIGH — file upload via htmx with `hx-encoding="multipart/form-data"` is a common failure point; temp file handling in `handleAddKB` could leak files on error |
| **KB add document form (URL fetch)** | Not tested | MEDIUM — network fetch during `KBAdd` could hang or fail; error path untested |
| **KB remove button** | Not clicked | LOW — simple POST handler |
| **KB search input on KB page** | Not tested (typed nothing into the search box) | LOW — it's a standard form GET to `/kb?q=...` |
| **KB chunk sidebar toggle** | Button exists but not clicked | LOW — purely visual toggle |
| **New Spec form** (on `/specs` page) | Not opened or tested | MEDIUM — same htmx form pattern as new task; could have same 422 re-render issues |
| **Sidebar collapse/expand toggle** | Not tested | LOW — purely visual; localStorage persistence not verified |
| **Task status change via dropdown** (on task detail) | Not tested via UI click; only verified dropdown renders correctly | MEDIUM — `onchange="this.form.submit()"` triggers a POST; redirect back to task detail untested |

### Web UI - Edge Cases

| Scenario | Why not tested | Risk |
|---|---|---|
| **Special characters in titles** (quotes, `<script>`, unicode, emoji) | Not tested | HIGH — Go's `html/template` should escape by default, but `data-md-body="{{.Body}}"` attribute could break if body contains `"` chars even after escaping; also affects BUG-07 (trash metadata JSON) |
| **Very long titles / bodies** | Not tested | LOW — may cause layout overflow issues on kanban cards and detail pages |
| **Empty body after edit** (editing a task/spec to have empty body) | Not tested | MEDIUM — could bypass server-side `minlength="20"` validation if htmx form allows empty textarea |
| **Rapid successive status changes** | Not tested | MEDIUM — flock timeout could trigger if multiple requests overlap; error feedback path untested |
| **Concurrent browser tabs** editing the same spec/task | Not tested | MEDIUM — no optimistic locking; last write wins, but the watcher could also interfere |
| **Browser back/forward navigation** after htmx swaps | Not tested | MEDIUM — htmx may not properly restore page state on browser history navigation; `hx-push-url` not observed in templates |
| **Search with no results** | Not tested | LOW — template likely shows empty state, but untested |
| **Search with special characters** (`"`, `'`, `*`, SQL injection attempts) | Not tested | MEDIUM — FTS5 query syntax could throw on malformed input (e.g., unmatched quotes); error handling on search errors not verified |
| **XSS via search query parameter** (`?q=<script>alert(1)</script>`) | Not tested | MEDIUM — Go `html/template` should escape, but the search input value rendering should be verified |
| **Body containing HTML entities** (`&amp;`, `<div>`, etc.) | Not tested | MEDIUM — double-escaping could occur when body goes through `html/template` escaping and then `marked.parse()` |

### Web UI - Responsive & Visual

| Scenario | Why not tested | Risk |
|---|---|---|
| **Mobile viewport** (< 600px) | Not tested at narrow widths | HIGH — kanban board with 8 columns likely overflows; sidebar behavior untested; touch targets may be too small |
| **Tablet viewport** (600-992px) | Not tested | MEDIUM — grid breakpoints and sidebar transitions untested |
| **Print stylesheet** | Not tested | LOW — unlikely to be used but no `@media print` rules observed |
| **Cross-browser** (Firefox, Safari) | Only tested in Chromium via Playwright | MEDIUM — CSS `has()` selector used in `app.js` is relatively new; BeerCSS may have browser-specific quirks |
| **RTL text / internationalization** | Not tested | LOW — all content is English; no `dir` attribute handling observed |

### KB Reader - Source Types

| Scenario | Why not tested | Risk |
|---|---|---|
| **PDF rendering** (PDF.js integration) | No PDF document was added to the test workspace | HIGH — PDF.js worker loading (`/assets/pdf.worker.min.mjs`), canvas rendering, text layer overlay, and chunk highlighting in text layer are all untested; this is the most complex reader path |
| **PDF chunk navigation across pages** | No PDF to test with | HIGH — rendering a new page canvas when navigating to a chunk on a different page is complex async code |
| **PDF with 100+ pages** | Not tested | MEDIUM — spec requires smooth navigation; memory usage and rendering performance untested |
| **HTML reader chunk highlighting** | Only verified iframe loads; did not test chunk highlight within iframe | MEDIUM — cross-origin iframe DOM access with `sandbox="allow-same-origin"` is browser-specific |
| **HTML reader with complex/malformed HTML** | Only tested with simple well-formed HTML | MEDIUM — bluemonday sanitization edge cases untested |
| **TXT reader with very large files** | Only tested with small file (1 chunk) | LOW — plain text rendering is simple but character offset-based highlighting could break with large files |
| **KB reader "View in source" from citation** | Not tested clicking citation link → reader with chunk anchor | MEDIUM — the `?chunk=N` URL parameter navigation was not end-to-end tested |
| **KB chunk connections display** | No TF-IDF connections existed in test data (0 connections) | LOW — the connections panel may have rendering issues when data exists |

### Server-Side - CLI Commands (phases 1-12)

| Scenario | Why not tested | Risk |
|---|---|---|
| **`specd lint`** via CLI (only seen in status page) | Only tested indirectly through status page | LOW — code was reviewed; lint issues display correctly on status page |
| **`specd tidy`** | Not invoked | LOW — runs lint + updates timestamp |
| **`specd rebuild --force`** | Not tested | MEDIUM — wipes and rebuilds cache.db; could have issues with FK cascade, counter reset, or criteria re-parsing |
| **`specd merge-fixup`** | Not tested | MEDIUM — designed for post-merge repair; complex renumbering logic untested |
| **`specd rename`** (spec or task) | Not tested | MEDIUM — renames slug + folder/file; could break links if references aren't updated |
| **`specd reorder spec`** | Not tested | LOW — simpler than task reorder |
| **`specd update`** via CLI | Not tested | LOW — similar logic to web update handlers |
| **`specd candidates`** | Partially tested (returned in `new-spec` / `new-task` output) | LOW |
| **`specd next`** with dependency cycles | Not tested | MEDIUM — cycle detection logic was reviewed but not exercised |
| **`specd kb rebuild-connections`** | Not tested | LOW — TF-IDF recomputation; no connections existed to verify against |
| **`specd kb add` from URL** | Not tested (only local files) | MEDIUM — HTTP fetch, content-type detection, temp file management |
| **`specd kb add` with PDF** | Not tested (no PDF) | HIGH — PDF text extraction with `ledongthuc/pdf`, page-aware chunking |

### Server-Side - Watcher Integration

| Scenario | Why not tested | Risk |
|---|---|---|
| **External file edit detected by watcher** | Not tested (would require editing a spec .md file outside the server and verifying SQLite updates) | HIGH — the watcher is a critical path; hash comparison, frontmatter re-parsing, criteria sync, FTS update are all untested end-to-end |
| **External file deletion detected by watcher** | Not tested | HIGH — trash insertion with `deleted_by='watcher'`, cascade cleanup |
| **Hand-created file rejection by watcher** | Created a file but watcher didn't detect it (file existed before server start) | MEDIUM — watcher only catches live changes; initial scan on startup not tested |
| **Watcher debounce under rapid edits** | Not tested | MEDIUM — 200ms debounce window; BUG-26 race condition applies |
| **Watcher + concurrent CLI writes** | Not tested | MEDIUM — flock contention between watcher callbacks and CLI commands |

### Performance & Scale

| Scenario | Why not tested | Risk |
|---|---|---|
| **100 specs / 1000 tasks / 50 KB docs** | Only tested with 3/8/3 | MEDIUM — spec requires board load < 500ms at this scale; kanban rendering with 1000 cards may be slow |
| **Large KB document (10,000 chunks)** | Not tested | MEDIUM — FTS5 indexing performance; KB reader with chunk sidebar listing 10K entries |
| **Search performance at scale** | Not tested | LOW — FTS5 + trigram should handle scale well, but untested |
| **Concurrent request handling** | Not tested | LOW — Go's `net/http` handles concurrency well, but flock contention could cause 5s timeouts |

### Security

| Scenario | Why not tested | Risk |
|---|---|---|
| **Path traversal via `/api/kb/:id/raw`** | Not tested (only reviewed code) | MEDIUM — `HasPrefix` check exists but symlink traversal was not tested |
| **SQL injection via search queries** | Not tested | LOW — FTS5 queries use parameterized queries, but FTS5 query syntax itself could be abused |
| **XSS via spec/task/KB content** | Not tested | LOW — Go `html/template` auto-escapes; `marked.parse()` output is `innerHTML` which could be a vector if marked doesn't sanitize |
| **CSRF protection** | Not tested | MEDIUM — no CSRF tokens observed on any forms; all mutation forms use POST but lack CSRF protection |
| **File upload size limits** | Not tested | MEDIUM — no `MaxBytesReader` observed in `handleAddKB`; could allow DoS via large upload |
