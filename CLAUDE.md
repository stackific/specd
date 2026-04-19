Read AGENTS.md, first.

# Behaviour

- When the user tells you to stop, plan, or think — stop immediately. Do not continue writing code. Do not "finish up" what you're doing. Stop.
- Never declare something "fixed" or "done" without verifying it visually through the Playwright MCP server. Build, reload, interact, screenshot. If the screenshot shows a problem, fix it before responding. Save all verification screenshots to `qa/verify/` (not the project root).
- Do not give up after one attempt. When something doesn't work, diagnose the root cause (inspect computed styles, check which CSS file is loaded, read the framework docs), fix it, and verify again.
- Address every concern the user raises. Do not skip messages, selectively respond, or bury acknowledgments in a wall of text.

# Rules

- Write production-quality code. Validate all user inputs (trim whitespace, reject empty/whitespace-only values, collapse consecutive spaces). Handle all error paths properly — never render a blank page with a raw error string. Use the htmx native approach for dialog form errors: re-render the form partial with 422 status, preserved values, and inline error banner.
- Do not take shortcuts or do surface-level work. Think through edge cases, verify changes visually using the Playwright MCP server before declaring them done, and iterate until the result is actually correct.
- Work with the existing framework (BeerCSS, htmx) rather than fighting it. Read the framework docs first. Understand how framework classes and attributes work before writing custom CSS overrides. Prefer the simplest solution that works within the framework. Never use `!important` to fight framework rules. Never force `height`, `top`, or other sizing overrides on BeerCSS components — use the framework's own size classes (`small`, `large`, etc.) and let it position internal elements (icons, labels) itself. Use our custom CSS utilities only for spacing, alignment, typography, and layout — not for overriding component internals.
- When reference files exist (e.g. `archived_ref/`), copy them directly instead of rewriting from scratch. Do not "adapt" or "simplify" — use the original as-is and only make the specific changes requested.
- Before modifying any file, read how it is actually used in context (imports, consumers, layout). Do not assume.
- Do not empty, gut, or zero out config values (like `social: {}`) unless explicitly asked to. Preserve existing data.
- Do not drop CSS files, imports, or structural elements from the original. If unsure whether something is needed, check the archived_ref or ask.
- When something looks wrong, diff against the reference before guessing at a fix.
- Always add godoc comments to all Go source files (except tests): package-level doc comments, exported types, exported functions, and unexported helpers where the intent isn't obvious from the name. Do this as you write code, not as a separate pass.
- Must use 2 spaces as indentation for non-Go files. Go uses tabs (enforced by gofmt).
- Must use pnpm as the package manager for the CSS build (`styles/`).
- Must implement Semantic HTML, Accessibility, W3C Valid markup in Go templates.

# Stack

- Go `html/template` (server-rendered HTML in `templates/`)
- htmx (partial page updates, no JS framework)
- BeerCSS (Material Design 3 CSS framework)
- material-dynamic-colors (runtime Material 3 role token generation)
- Vite (CSS-only build in `styles/`: LightningCSS + PurgeCSS)

# Prerequisites

- Go 1.26+
- Node.js 22+ and pnpm (for CSS build in `styles/`)
- just (`brew install just`)
- air (dev only, `go install github.com/air-verse/air@latest`)

# CSS rules

**The full CSS architecture lives in `styles/src/styles/README.md`. BeerCSS reference at `styles/docs/BEERCSS.md`.** Below is the operational summary.

## Where styles live

```
styles/public/vendor/beer.min.css                  BeerCSS, served as its own <link>
styles/src/styles/index.css                        Custom CSS entrypoint (foundation)
styles/src/styles/{tokens,base,utilities,layout}.css  Foundation layer
styles/src/styles/components/<name>.css             Component CSS (imported in index.css)
styles/src/styles/pages/<page>.css                  Page CSS (imported in index.css)
```

## Hard rules

1. **NEVER write an unprefixed class name** in any custom `.css` file. Every selector must be namespaced. The bundle is global and unprefixed classes will collide.
2. **BeerCSS owns COMPONENTS only.** Use BeerCSS for buttons, fields, chips, dialogs, cards, articles, grid, navs, snackbars. Do NOT use BeerCSS for spacing, sizing, typography, margin/padding, or color outside of role tokens.
3. **Sizing, padding, margin, gap, typography, and custom colors come from our framework.** Use utility classes from `styles/src/styles/utilities.css`. Tokens in `tokens.css`. Never add a utility by hand — edit `scripts/gen-utilities.mjs` and run `pnpm gen:utilities`.
4. **NEVER hard-code a `font-family`.** The global font is `--font` (Geist Variable, set in `tokens.css`). Only exception: icon font faces.
5. **BeerCSS JS is vendored as ES modules.** `assets/beer.min.js` and `assets/material-dynamic-colors.min.js` loaded as `<script type="module">` in `templates/layouts/base.html`. Theme logic in `assets/app.js`.

## Naming convention

- **Tokens**: `--<scale>-<step>` (e.g. `--space-3`, `--text-2`, `--c-primary-40`)
- **Utilities**: short verb + step (e.g. `.p-3`, `.mt-2`, `.text-3`, `.fw-5`)
- **Spacing**: `m-3`, `mt-3`, `mr-3`, `mb-3`, `ml-3`, `mx-3`, `my-3` / same for `p-`
- **Responsive**: `m:` (tablet ≥601px), `l:` (desktop ≥993px). Colon is literal in HTML, escaped as `\:` in CSS.
- **Components**: 2–3 letter prefix (`.c-event`, `.gfx-hero`). Internals: 2-letter prefix (`.ev-title`).
- **Pages**: 2–4 letter prefix (`.kb-grid`, `.board-col`).
- **State**: `.is-active`, `.is-hidden`, `.is-disabled`.
- **Colors**: `--c-*` palette (6 palettes × 18 tones, generated via `pnpm gen:colors`). BeerCSS role tokens (`--primary`, `--surface`) are separate — do not redefine them.

## CSS build pipeline

- `styles/vite.config.js`: `cssCodeSplit: false`, LightningCSS transformer (`@custom-media`), esbuild minifier
- PurgeCSS Vite plugin scans `templates/**/*.html` (Go templates) after build
- BeerCSS loaded via `<link>` in `base.html`, NOT in the CSS bundle
- Every page ships TWO `<link>` tags: `/vendor/beer.min.css` + `/assets/style.<hash>.css`
- **PurgeCSS safelist**: classes created only in JS (not in templates) will be purged. Add them to the safelist in `styles/vite.config.js` under `safelist.standard`. Current safelist includes `/^is-/`, `/^sd-drop-/`. Changes to `vite.config.js` require restarting the dev server (`just qa`).

## Adding new things

| Need | Where |
|---|---|
| New design token | `styles/src/styles/tokens.css` |
| New utility class | `styles/scripts/gen-utilities.mjs`, then `cd styles && pnpm gen:utilities` |
| New `@custom-media` | `styles/src/styles/media.css` |
| Regenerate colors | `cd styles && pnpm gen:colors '#hex'`, paste into `tokens.css` |
| New component CSS | `styles/src/styles/components/<name>.css`, add `@import` in `index.css` |
| New page CSS | `styles/src/styles/pages/<page>.css`, add `@import` in `index.css` |
| New Go template partial | `templates/partials/`, use `{{define "name"}}` |
| New page template | `templates/pages/`, define `{{define "content"}}`, add handler in `internal/web/handlers.go`, route in `internal/web/server.go` |
| Update BeerCSS | follow `styles/public/vendor/README.md` |

## Do not change

- `styles/src/styles/{index,reset,media,tokens,base,utilities,layout}.css`
- `styles/scripts/gen-utilities.mjs`, `styles/scripts/gen-colors.mjs`
- `styles/vite.config.js` build pipeline (safelist entries may be added but do not alter the plugin structure)
- `styles/public/vendor/beer.min.css` and the vendoring approach

## Page toolbar pattern

All list pages (board, specs, KB, trash) use the same responsive toolbar layout:

```html
<div class="sd-board-toolbar mb-5">
  <h1 class="text-4 m:text-5 fw-7 lh-1">Page Title</h1>
  <div class="sd-board-controls">
    <button class="round" data-ui="#dialog">...</button>
    <div class="small field round fill no-margin no-elevate sd-board-filter">
      <select>...</select>
    </div>
  </div>
</div>
```

- Desktop: heading left, controls right-aligned
- Tablet/mobile: heading wraps above, filter stretches to fill remaining width
- Button uses BeerCSS default size; field uses `small` — intentional mismatch accepted
- CSS in `styles/src/styles/pages/board.css` (`.sd-board-toolbar`, `.sd-board-controls`, `.sd-board-filter`)

# Validation rules

- Spec and task bodies must NOT contain H1 headings (`# Title`). Titles come from frontmatter only. Enforced at both workspace layer (`validateNoH1` in `slug.go`) and web handler layer (`bodyHasH1` in `handlers.go`). Existing H1s are stripped from edit forms (`stripH1`) and detail view rendering (`stripH1` template function).
- Search queries are trimmed, consecutive spaces collapsed, and special characters handled. FTS5 BM25 uses `sanitizeBM25` (extracts alphanumeric tokens); trigram uses `sanitizeTrigram` (quoted phrase). Both in `search.go`.

# Go project conventions

- The `citations` table uses `from_id` (plain text), not a foreign key to `specs` or `tasks`. Deleting a spec or task must explicitly `DELETE FROM citations WHERE from_kind = '...' AND from_id = '...'` — it will not cascade via FK.
- Every workspace mutation acquires the flock via `w.WithLock(func() error { ... })`. Read-only operations (candidates, search, read) do not need the lock.
- Tests use `setupWorkspace(t)` from `workspace_test.go` which calls `Init` in a `t.TempDir()`. Tests that need KB + spec + task use `setupWithKB(t)` from `cite_test.go`.
- Colocate tests with their source: `spec.go` → `workspace_test.go`, `kb.go` → `kb_test.go`, `cite.go` → `cite_test.go`, etc. Do not put tests in a single monolithic file.
- `NewSpecResult` and `NewTaskResult` include a `Candidates` field computed outside the lock after creation. This is intentional — candidates are read-only.
- PDF text extraction uses `ledongthuc/pdf` (pure-Go, no CGO). It has limited format support compared to go-fitz/MuPDF but keeps the build CGO-free to match `modernc.org/sqlite`.

# Go template conventions

- Templates in `templates/` (layouts, partials, pages) — embedded via `embed.go`
- `base.html` defines `{{block "content" .}}{{end}}` — each page overrides this block
- Partials: `{{define "name"}}...{{end}}`, included with `{{template "name" .}}`
- All templates receive `PageData` struct with `.Title` and `.CSSFile`
- htmx-aware handlers check `HX-Request` header: present → content partial only; absent → full page with base layout

# htmx rules

- **Every form that mutates state must use htmx attributes** (`hx-post`, `hx-target`, `hx-swap`). Never rely on plain `method="post"` with `this.form.submit()` — the handler returns an htmx content partial, and a plain form submit causes the browser to navigate to the POST URL and render raw HTML.
- For inline updates (criteria check/uncheck, add, remove): use `hx-post` pointing at the action URL, `hx-target` pointing at the nearest re-renderable container (e.g. `#task-detail`), and `hx-swap="outerHTML"`.
- For checkbox/select auto-submit: use `onchange="this.form.requestSubmit()"` (not `.submit()`) so htmx intercepts the submission.
- Dialog form handlers return `HX-Redirect` on success and re-render the form partial with HTTP 422 on validation error (htmx swaps in place, preserving user input).
- Non-dialog mutation handlers (criteria, move, delete) must check `r.Header.Get("HX-Request")` and fall back to `http.Redirect` for non-htmx requests as a safety net. Never force `r.Header.Set("HX-Request", "true")` — that masks bugs where forms lack htmx attributes.
- All navigation links use `hx-get`, `hx-target="main"`, `hx-swap="innerHTML"`, `hx-push-url="true"` for SPA-like transitions. The server detects `HX-Request` and returns only the content block.
- **htmx history cache is disabled** (`htmx.config.historyCacheSize = 0` in `base.html`). Do not re-enable — it bloats localStorage with full page snapshots.
- Use `hx-push-url="true"` on filter controls so the URL reflects current view state and is shareable/bookmarkable.

# localStorage conventions

Only four keys are used. Do not add new keys without justification.

| Key | Purpose | Invalidation |
|-----|---------|-------------|
| `mode` | Dark/light theme preference | Never (user choice) |
| `nav-collapsed` | Sidebar collapsed state | Never (user choice) |
| `sd-board-spec` | Board spec filter preference | Auto-cleared by `initBoardFilter()` when the saved spec no longer exists in the dropdown |
| `specs-view` | Specs page view mode (`card` or `list`) | Never (user choice) |

- The URL is the source of truth for view state. localStorage only persists user preferences across visits.
- An early `<script>` in `<head>` restores `sd-board-spec` by redirecting `/` to `/?spec=SAVED` before first paint.
- Every localStorage preference must have an invalidation path — stale entries referencing deleted entities must be auto-cleared on next page load.
- Never store page content, HTML snapshots, or large data in localStorage.

# UI pattern catalogue

Reuse these patterns exactly. Do not invent new structural patterns.

## 1. Page toolbar

All list pages use `.sd-board-toolbar` (flexbox wrap). Heading left, controls right. On mobile the heading wraps above and the filter stretches to fill.

```html
<div class="sd-board-toolbar mb-5">
  <h1 class="text-4 m:text-5 fw-7 lh-1">Title</h1>
  <div class="sd-board-controls">
    <button class="round" data-ui="#dialog">
      <i aria-hidden="true">add</i>
      <span class="m l">New Item</span>
    </button>
    <div class="small field round fill no-margin no-elevate sd-board-filter">
      <select aria-label="Filter">...</select>
    </div>
  </div>
</div>
```

Used on: board, specs, KB, trash. CSS in `board.css`.

## 2. Data table with ID chip

First column: clickable chip with type icon + ID. Title column gets remaining space. Right columns shrink to content.

```html
<table class="border sd-kb-table">
  <thead><tr><th></th><th>Title</th><th class="m l">Col</th><th></th></tr></thead>
  <tbody><tr>
    <td class="sd-kb-id">
      <a href="/path" class="chip small no-margin no-line secondary" title="type">
        <i aria-hidden="true">icon</i><span>ID</span>
      </a>
    </td>
    <td class="fw-5"><a href="/path" class="no-line">Title</a></td>
    <td class="m l text--1">value</td>
    <td><!-- actions --></td>
  </tr></tbody>
</table>
```

Used on: KB list, specs list view, spec detail tasks. CSS: `.sd-kb-id` (width 1%, no-wrap, no right padding), `.sd-kb-table` last-child (width 1%, no-wrap).

## 3. Detail page header

Grid with title left, action buttons right-aligned.

```html
<div class="grid no-space mb-2">
  <div class="s8 middle-align">
    <h1 class="text-4 m:text-5 fw-7 lh-1">ID — Title</h1>
  </div>
  <div class="s4 right-align middle-align">
    <!-- status select, edit, delete buttons -->
  </div>
</div>
```

Task detail: status `<select>` + edit pencil + delete trash. Spec detail: add task circle + edit + delete.

## 4. Back navigation

```html
<nav class="mb-4">
  <a href="/parent" class="chip small no-margin">
    <i aria-hidden="true">arrow_back</i>
    <span>Parent Label</span>
  </a>
</nav>
```

## 5. Dialog form (create/edit)

htmx form inside a `<dialog>`. Returns 422 with re-rendered form on error, `HX-Redirect` on success.

```html
<dialog id="dialog-id" aria-label="Label" class="sd-dialog-form">
  <h5>Heading</h5>
  <form hx-post="/action" hx-target="this" hx-swap="outerHTML" class="sd-dialog-body">
    {{if .Error}}
    <div class="sd-form-error error-container round p-3" role="alert">
      <i aria-hidden="true">error</i><span>{{.Error}}</span>
    </div>
    {{end}}
    <!-- fields -->
    <nav class="right-align no-space">
      <button type="button" class="transparent link" data-ui="#dialog-id">Cancel</button>
      <button type="submit" class="ml-2">Save</button>
    </nav>
  </form>
</dialog>
```

## 6. Confirmation dialog (delete/remove)

No htmx — plain form POST inside dialog.

```html
<dialog id="confirm-dialog" aria-label="Confirm" style="max-height:none;">
  <h5>Delete Item?</h5>
  <p class="mt-3">Description of what happens.</p>
  <nav class="right-align no-space mt-4">
    <button type="button" class="transparent link" data-ui="#confirm-dialog">Cancel</button>
    <form method="post" action="/path/delete" class="inline">
      <button type="submit" class="error ml-2">Delete</button>
    </form>
  </nav>
</dialog>
```

For JS-driven confirmations (criterion remove), use `ui("#dialog-id")` to open and `htmx.ajax()` to submit.

## 7. Criterion row (checkbox + remove)

Flex row: checkbox form stretches, remove button shrinks.

```html
<div class="sd-criterion mb-2">
  <form hx-post="/check-or-uncheck" hx-target="#container" hx-swap="outerHTML"
        class="sd-criterion-check">
    <label class="checkbox">
      <input type="checkbox" onchange="this.form.requestSubmit()" />
      <span>Text</span>
    </label>
  </form>
  <button type="button" class="circle small transparent sd-criterion-remove"
    onclick="sdCriterionConfirmRemove(id, pos)">
    <i aria-hidden="true">close</i>
  </button>
</div>
```

CSS: `.sd-criterion` (flex, align-items center, gap), `.sd-criterion-check` (flex 1), `.sd-criterion-remove` (flex-shrink 0).

## 8. Inline add form

Flex row: input stretches, button fixed.

```html
<form hx-post="/add" hx-target="#container" hx-swap="outerHTML" class="sd-criterion-add mt-3">
  <div class="field border round no-margin">
    <input type="text" name="text" required placeholder="Add item" />
  </div>
  <button type="submit" class="large round">
    <i aria-hidden="true">add</i><span>Add</span>
  </button>
</form>
```

CSS: `.sd-criterion-add` (flex, align-items flex-end, gap), `.sd-criterion-add .field` (flex 1).

## 9. Citation/reference card

Clickable card linking to KB reader at the specific chunk.

```html
<a href="/kb/{{.KBDocID}}?chunk={{.ChunkPosition}}" class="no-line">
  <article class="surface-container-low round p-3 mb-2">
    <p class="text-1 fw-5 lh-2">
      <i class="small" aria-hidden="true">icon</i>
      Title · chunk N
    </p>
    <p class="text--1 lh-3 mt-1">Preview text</p>
    <p class="text--1 mt-1 primary-text">View in source</p>
  </article>
</a>
```

## 10. Kanban card

Draggable card with metadata row (progress, blocked icon, citation icon).

```html
<article class="sd-card surface-container-low round p-3 mb-2"
  draggable="true" data-task-id="ID" data-status="status">
  <a href="/tasks/ID" class="no-line sd-card-link">
    <p class="text-1 fw-5 lh-2">Title</p>
  </a>
  <p class="text--1 lh-3 mt-1">ID · Spec</p>
  <div class="sd-card-meta mt-2">
    <span class="sd-card-progress"><progress class="circle tiny" ...></progress><span class="text--2">N/M</span></span>
    <span class="sd-card-icon"><i class="text--1 error-text">warning</i></span>
  </div>
</article>
```

Drop placeholder: `<div class="sd-drop-placeholder"></div>` (created in JS, safelisted in PurgeCSS).

## 11. Markdown body rendering

Body passed via data attribute, rendered client-side by marked.js. Strip H1 and criteria before rendering.

```html
<article class="sd-body surface-container-low round p-4 m:p-5 mt-5">
  <div class="sd-body-md" data-md-body="{{stripH1 (stripCriteria .Body)}}"></div>
</article>
```

## 12. Empty state

```html
<article class="surface-container-low round p-5">
  <p class="text-2 lh-3">No items yet. Click <strong>Action</strong> to create one.</p>
</article>
```

## 13. Search input (autofocus)

```html
<form method="get" action="/search" class="mb-5">
  <div class="field label prefix border">
    <i aria-hidden="true">search</i>
    <input type="search" name="q" value="{{.Query}}" autocomplete="off" autofocus placeholder=" " />
    <label>Search specs, tasks, and KB...</label>
  </div>
</form>
```

## 14. Status select (auto-submit)

```html
<form method="post" action="/tasks/ID/move">
  <div class="field suffix border round small no-margin">
    <select name="status" class="sd-auto-submit">
      <option value="val" selected>Label</option>
    </select>
    <i aria-hidden="true">swap_vert</i>
  </div>
</form>
```

JS in `app.js` listens for `change` on `.sd-auto-submit` and calls `form.submit()`.
