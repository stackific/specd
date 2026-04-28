# specd

A specification-driven development CLI tool.

## Tech Stack

- **Language:** Go (no CGO — `CGO_ENABLED=0` always)
- **CLI framework:** [Cobra](https://cobra.dev/)
- **Task runner:** [Taskfile](https://taskfile.dev/) (`task` command)
- **Live reload:** [Air](https://github.com/air-verse/air)
- **Git hooks:** [Lefthook](https://github.com/evilmartians/lefthook) (parallel execution)
- **Linting:** [golangci-lint](https://golangci-lint.run/) (meta-linter, see `.golangci.yml`)
- **Formatting:** gofumpt + goimports + gci (auto-fix, never warn)
- **Security:** gosec (static), govulncheck (deps), gitleaks (secrets)
- **Commit linting:** [conform](https://github.com/siderolabs/conform) (conventional commits)
- **Frontend:** Go `html/template` + [htmx](https://htmx.org/) + [BeerCSS](https://www.beercss.com/) (Material Design 3), server-rendered, embedded in Go binary
- **CSS build:** [Sass](https://sass-lang.com/) (SCSS → CSS) + [Lightning CSS](https://lightningcss.dev/) (bundle + minify) + [PurgeCSS](https://purgecss.com/) (tree-shake unused classes)
- **CSS linting:** [Stylelint](https://stylelint.io/) (standard SCSS config)
- **Package manager:** pnpm (always use pnpm, never npm or yarn)
- You must write idiomatic Go and directory structure.
- You must use 2 spaces as indentation for Non-Go code file

## Project Structure

```
main.go              # Entrypoint (embeds skills/, templates/, static/ via go:embed)
cmd/                 # Cobra commands (root.go, subcommands)
cmd/frontend.go      # Embedded FS injection (SetTemplateFS, SetStaticFS)
cmd/templates.go     # Go template parsing, rendering (htmx-aware)
cmd/constants.go     # All magic strings and constants (single source of truth)
cmd/config.go        # Global (~/.specd/config.json) and project (.specd.json) config
cmd/database.go      # SQLite initialization, ID counters, project DB helpers
cmd/search.go        # Hybrid BM25 + trigram search across specs, tasks, KB
cmd/sync.go          # Cache refresher — reconciles spec and task markdown files with the DB
cmd/slug.go          # ToSlug (underscore — config values only), FromSlug (display)
cmd/providers.go     # AI provider definitions (Claude, Codex, Gemini)
cmd/new_spec.go      # specd new-spec command
cmd/new_task.go      # specd new-task command
cmd/update_spec.go   # specd update-spec command
cmd/update_task.go   # specd update-task command (status change, criteria toggle)
cmd/list_specs.go    # specd list-specs command (paginated)
cmd/list_tasks.go    # specd list-tasks command (paginated, filterable)
cmd/serve.go         # specd serve command (HTTP server with htmx + REST API)
cmd/schema.sql       # Embedded SQLite schema (dynamic CHECK constraints)
templates/           # Go HTML templates (layouts, partials, pages)
templates/layouts/   # Base layout with content block
templates/partials/  # Shared partials (nav, footer)
templates/pages/     # Page templates (override content block)
static/              # Static assets (embedded in Go binary)
static/vendor/       # Vendored JS (htmx, BeerCSS, Material Dynamic Colors)
static/fonts/        # Geist Variable font files (woff2)
static/images/       # Favicons, logos, manifest, robots.txt
static/css/src/      # SCSS source (app.scss with maps, mixins, utility generators)
static/css/dist/     # Built CSS output (gitignored, bundled + purged + minified)
static/js/           # Client-side JS (theme, nav, htmx config, livereload)
static/package.json  # CSS build deps (lightningcss-cli, purgecss)
skills/              # Embedded skills (Agent Skills Standard format)
scripts/             # Install/uninstall scripts
docs/internal/       # Internal setup guides
qa/specs/            # QA test resources (markdown specs + setup script)
Taskfile.yml         # Task definitions
lefthook.yml         # Git hook definitions
.golangci.yml        # Linter config
.conform.yaml        # Commit message policy
.gitleaks.toml       # Secret scanning config
.air.toml            # Live reload config
```

## Commands

```sh
task build           # Build binary to bin/
task run             # Build and run
task dev             # Live reload (uses air)
task test            # Run tests
task fmt             # Format all Go files (gofumpt + goimports + gci)
task fmt:check       # Check formatting without writing
task lint            # Run golangci-lint
task lint:fix        # Run golangci-lint with auto-fix
task sec             # Run all security checks
task sec:vulncheck   # Check deps for known vulnerabilities
task sec:gitleaks    # Scan for leaked secrets
task deadcode        # Find unreachable code from main
task check           # Run everything (fmt, lint, css:lint, test, security)
task build:all       # Cross-compile for linux/darwin/windows (amd64+arm64)
task hooks:install   # Install lefthook git hooks
task clean           # Remove bin/, tmp/, and static/css/dist/
task css:install      # Install CSS build/lint dependencies (pnpm)
task css:build        # Build CSS (bundle + purge + minify)
task css:lint         # Run Stylelint on CSS source
task css:lint:fix     # Run Stylelint with auto-fix
```

## Git Hooks (via lefthook)

- **pre-commit** (parallel): format (gofumpt + goimports + gci), golangci-lint --fix, gitleaks, stylelint (CSS)
- **commit-msg:** conform (conventional commit format required)
- **pre-push** (parallel): tests, govulncheck

Run `task hooks:install` after cloning.

## Commit Message Format

Conventional commits enforced by conform:

```
type(scope): description    # scope is optional
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

## Pushing Code

All commits must pass three hook stages before reaching the remote:

1. **pre-commit**: formatters, golangci-lint, gitleaks
2. **commit-msg**: conventional commit format (conform)
3. **pre-push**: `go test ./...`, `govulncheck ./...`

Additionally, all commits must be:
- **DCO signed off** (`Signed-off-by` line) — use `git commit -s` or set `git config format.signoff true`
- **Cryptographically signed** (SSH or GPG) — enforced by branch protection on `main`

If a push is rejected, fix the issue, amend or create a new commit, and push again. Do not use `--no-verify` to bypass hooks.

When committing via HEREDOC (`git commit -m "$(cat <<'EOF' ... EOF)"`), `format.signoff` does NOT apply — always pass `-s` explicitly.

## Rules

- **Never trust training data for external tool conventions, APIs, or directory structures.** Always search the web and read primary sources (official docs, actual repos) first. This is especially critical when the user explicitly asks you to search. Do not guess or rely on what you "know" — verify it.
- **No CGO.** All builds use `CGO_ENABLED=0`. Never add C dependencies.
- **Cross-compilation** targets: linux, darwin, windows × amd64, arm64. All built from macOS.
- **Cross-platform text parsing.** Always normalize `\r\n` → `\n` (via `normalizeLineEndings()`) before splitting or parsing file content. Windows writes CRLF; hardcoded `\n` splits will silently break. When hashing file content (e.g. `content_hash`), hash the **raw** bytes before normalizing so the hash matches what's on disk.
- **Cobra commands** go in `cmd/`. One file per command. Follow Cobra conventions.
- **All exported functions** must have a doc comment.
- **Unused function parameters** must be named `_`.
- **Always run `task lint` after writing or modifying Go code.** Do not declare work done until it passes with 0 issues. The pre-commit hook will block the commit otherwise.
- **Never finish a task without writing tests for all new or changed code.** If you added a function, command, or behavior, write tests for it before declaring done. Check for coverage gaps proactively — do not wait to be asked. Adequate tests means:
  - **Happy path**: the feature works as expected.
  - **Negative cases**: invalid input, missing data, nonexistent IDs return errors, not crashes.
  - **Edge cases**: empty inputs, boundary values, items that should NOT match.
  - **Side effects**: cascading deletes actually cascade, other records survive, counters don't reset.
  - **State transitions**: flags change (e.g. `needs_ai_processing` goes from 1 to 0), hashes update after file rewrites.
  - **Never call `rootCmd.Execute()` in tests for commands with blocking loops** (e.g. `logs` follow mode). Test the underlying functions directly or test preconditions only. Blocking tests hang the entire suite.
- **Do not start the dev server** — the user runs it themselves.
- **Do not add features, fallbacks, or logic beyond what was asked.** If the user says "use X as a fallback", only add X — do not invent additional fallbacks (e.g. OS username) on your own.
- **Never duplicate logic across functions.** If three functions share 80%+ of their code, extract the common logic into a single parameterized function. Use maps or config structs to handle per-case differences (e.g. `bm25Queries` map for per-kind SQL).
- **Never silently swallow errors.** SQL query errors, scan errors, and file I/O errors must be returned — not ignored with `if err == nil {` or `continue`. Silent failures hide bugs.
- **Filesystem is ground truth.** The database is a derived cache. When any command modifies a file on disk (e.g. `update-spec` rewrites frontmatter), it must recompute `content_hash` from the written file and update the database. Stale hashes break sync.
- **Never pass user input raw to FTS5 MATCH.** Always sanitize through `sanitizeBM25` or `sanitizeTrigram`. Never pass through FTS5 operators (AND, OR, NOT, NEAR) from user input — this is an injection vector.
- **All tunables and magic strings go in `cmd/constants.go`.** Never hardcode values (search weights, thresholds, directory names, file names) in function bodies or SQL strings. If a value might be tuned, it belongs in constants.
- **Spec acceptance criteria use plain list items** (`- criteria text`), never checkbox syntax. **Task acceptance criteria use checkbox syntax** (`- [ ] text` / `- [x] text`) — checked state is stored as an integer (`0`/`1`) in `task_criteria.checked` and synced bidirectionally between the markdown file and the database.
- **When the codebase outgrows `cmd/`** (~20+ files), extract domain logic into `internal/` packages. For now `cmd/` is fine for a Cobra CLI.
- **Frontend conventions** are documented in the "Frontend (templates/ + static/)" section below.

## Skills

- specd uses the **[Agent Skills Standard](https://agentskills.io/specification)** for all AI provider integrations.
- All three providers (Claude, OpenAI Codex, Gemini) support the same `<name>/SKILL.md` format. **Do NOT use legacy formats** (`.claude/commands/`, `.gemini/commands/*.toml`). Always use `<provider-dir>/skills/<name>/SKILL.md`.
- Canonical skills live in `skills/` at the repo root and are embedded into the binary via `go:embed`.
- Provider skill directories:
  - Claude: `.claude/skills/<name>/SKILL.md`
  - Codex: `.agents/skills/<name>/SKILL.md`
  - Gemini: `.gemini/skills/<name>/SKILL.md`
- **Always verify conventions against primary sources** before implementing. Do not rely on stale knowledge. Check the actual repos and official docs:
  - Agent Skills Standard: https://agentskills.io/specification
  - Claude Code: https://code.claude.com/docs/en/skills
  - Codex CLI: https://developers.openai.com/codex/skills
  - Gemini CLI: https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/creating-skills.md

## Search

- **Hybrid BM25 + trigram** strategy in `cmd/search.go`. Single `Search()` entry point, single `searchByKind()` implementation for all three kinds — no copy-pasted per-kind functions.
- **BM25** (primary): FTS5 with porter stemming. Tokens are individually quoted via `sanitizeBM25`. Results ranked by score (higher = better).
- **Trigram** (fallback): FTS5 trigram tokenizer for substring matching. Activated when BM25 returns < 3 hits or query has special characters. Trigram results have score=0, appended after BM25 results.
- **Deduplication**: `seen` map prevents the same ID appearing in both BM25 and trigram results. `excludeID` filters out the document being searched for.
- Per-kind SQL lives in the `bm25Queries` map — add new searchable kinds there, not by duplicating functions.
- **FTS indexes only contain searchable text** (title, summary, body). Enum fields like `type` and `status` are NOT indexed — they are short slugs that should be filtered via SQL WHERE, not full-text searched. The search queries JOIN to base tables for all metadata.
- `NewSpecResponse` JSON field for KB results is `related_kb_chunks` — do not rename this, deployed skills depend on it.
- See `SEARCH.md` at the repo root for a full prose explanation of the search strategy.

## Cache Sync

- **Markdown files are the ground truth.** The `.specd.cache` database is a derived cache, rebuilt by `SyncCache()` which runs in `PersistentPreRunE` before every non-exempt command.
- Sync walks `<specd-dir>/specs/*/spec.md` and `<specd-dir>/specs/*/TASK-*.md`, parses frontmatter, computes SHA-256 of the **full file** (frontmatter + body), and reconciles: insert new, update changed (by hash), delete removed, reconcile links and criteria.
- FTS and trigram indexes are maintained automatically via database triggers — sync only touches the base tables (`specs`, `spec_links`, `spec_claims`, `tasks`, `task_links`, `task_dependencies`, `task_criteria`).
- `update-spec` rewrites the entire spec.md from DB state via `rewriteSpecFile()` after any change, keeping the file as ground truth.
- **Task sync** walks `<specd-dir>/specs/*/TASK-*.md`, parses frontmatter (including `linked_tasks` and `depends_on` YAML lists), extracts H1 title, extracts checkbox criteria from `## Acceptance Criteria`, and reconciles `tasks`, `task_links`, `task_dependencies`, and `task_criteria` tables. Checked state is preserved when criteria text hasn't changed.
- `update-task` supports `--status`, `--check`, and `--uncheck` flags. It rewrites the task file via `rewriteTaskFile()` after changes, keeping checkboxes in sync with the DB.
- **Spec title** is the first `# Heading` in the body, NOT a frontmatter field. `extractH1Title()` parses it. `buildSpecMarkdown()` writes it as `# Title` after the `---` delimiter.
- **Acceptance criteria** (claims) are parsed from `## Acceptance Criteria` section bullets using must/should/is/will language. Stored in `spec_claims` table with a dedicated `spec_claims_fts` FTS5 index for searching claims independently.
- Spec frontmatter includes `id`, `type`, `summary`, `position`, `linked_specs`, `created_by`, `created_at`, `updated_at`. **No `title` field** — it's in the body.
- Spec body must have exactly one `# Title` (H1). No other H1 headings allowed. Use `##` for top-level sections, `###`–`######` freely within sections. `## Acceptance Criteria` must be H2.
- **Task files** live at `<specd-dir>/specs/spec-<N>/TASK-<N>.md` alongside their parent spec. Frontmatter includes `id`, `spec_id`, `status`, `summary`, `position`, `linked_tasks`, `depends_on`, `created_by`, `created_at`, `updated_at`. Title is in the body H1.
- **Validation**: `parseSpecMarkdown` validates required fields (`id`, `title` from H1, `type`, `summary`). Invalid specs are silently skipped.
- `update-spec` supports `--unlink-specs` and `--unlink-kb-chunks` to remove references. Response includes full summaries for linked specs and KB chunks, not just IDs.
- **Contradiction detection**: `specd search-claims --query "..." --exclude SPEC-X` searches `spec_claims_fts` for matching claims from other specs. Both `/specd-new-spec` and `/specd-update-spec` skills use this in step 3 to find and report conflicting acceptance criteria across specs. The AI evaluates matches — no automated resolution.
- Exempt commands (`init`, `version`, `skills`, `help`, `logs`) skip the sync.
- See `docs/internal/spec-markdown-schema.md` for the spec file format.

## File Conventions

- **`.specd.json`** — project config marker at the repo root. Committed to git. Contains directory name, spec types, task stages, search settings.
- **`<specd-dir>/`** (default: `specd/`) — contains `specs/` and `kb/` subdirectories. Committed to git.
- **`<specd-dir>/specs/spec-<N>/`** — each spec directory holds `spec.md` and its task files (`TASK-<N>.md`).
- **`<specd-dir>/kb/KB-<N>.md`** — KB document files.
- **`.specd.cache`** — SQLite cache database at the repo root. **Gitignored.** Rebuilt from spec/task markdown files. Never committed.
- **`~/.specd/`** — user-level config and skills. Never committed. Contains `config.json`, `skills/`, `update-check.json`, `specd.log`.

## Logging

- Uses `log/slog` (stdlib) with JSON lines output to `~/.specd/specd.log`.
- `InitLogger()` called in `PersistentPreRunE` — every command logs.
- Default level: `Info`. Set `SPECD_DEBUG=1` for `Debug` output.
- Log file auto-truncated at 10 MB.
- `specd logs` streams the log file (tail -f style). Exempt from project guard.
- Log operational events (command runs, sync inserts/updates/deletes, search queries). Never log user content (spec bodies, summaries).

## Slugs

- `ToSlug()` → underscore-separated (e.g. `pending_verification`). Used **only** for config values: spec types, task stages. These are stored in `.specd.json` and used in SQL CHECK constraints.
- `ToDashSlug()` → dash-separated. Currently unused — retained for future needs.
- No content type (specs, tasks, KB docs) uses slugs. They are identified by their ID only (e.g. `SPEC-1`, `TASK-1`, `KB-1`).

## Pagination

- `list-specs` and `list-tasks` support `--page` (1-based) and `--page-size` (default 20 from `DefaultPageSize`).
- `list-tasks` also supports `--spec-id` and `--status` filters.
- Responses include `page`, `page_size`, `total_count`, `total_pages` metadata.
- Results are ordered by `position`, then `id`.

## Web UI (Serve)

- `specd serve` starts an HTTP server rendering Go templates with htmx support and serving static assets.
- Port scanning starts at `DefaultServePort` (8000) and tries up to `MaxPortAttempts` (100) ports.
- Prints port scanning progress and the final URL to the terminal.
- Opens the user's default browser via `open`/`xdg-open`/`rundll32` depending on OS.
- Route `/` reads `default_route` from the `meta` table **server-side** and issues a 307 redirect (default: `/docs/tutorial`). Never replace this with a client-side fetch + redirect — keep the redirect in Go so the URL the user typed is the URL their browser navigates to.
- REST API routes live under `/api/` prefix (e.g. `/api/meta/default-route`). Settings-page actions that mutate state and return HTML fragments (not JSON) live under `/settings/` (e.g. `POST /settings/default-route`) — pick the prefix that matches the consumer.
- Page routes render Go templates with htmx partial support (HX-Request header → content block only).
- Static assets (CSS, JS, fonts, vendor libs) are served from the embedded `static/` filesystem.
- `--dev` flag enables live reload: re-parses templates on every request and injects an SSE-based reload script.
- **Form-handling discipline:** wrap `r.Body` with `http.MaxBytesReader(w, r.Body, MaxSettingsFormBytes)` before `r.ParseForm()` (gosec G120). When a form value is constrained to an allowlist (e.g. `StartpageChoices`), resolve it to the canonical entry from the list **before** logging or persisting — this both rejects unknown values and satisfies gosec G706 taint analysis on `slog` calls.
- **gocognit budget on `runServe`:** the function registers many routes; keep its complexity ≤ 20 by extracting per-page handlers (`makeSpecsHandler`, `makeSearchHandler`, `handleSettingsPage`) and shared helpers (`makeFreshPages`, `makePageHandler`) instead of inlining closures.
- **Persisting UI settings:** read with `ReadMeta(db, key)`, write with `WriteMeta(db, key, value)` (upserts via `ON CONFLICT(key) DO UPDATE`). Keys live in `cmd/constants.go` (e.g. `MetaDefaultRoute`).

## Frontend (templates/ + static/)

- **Stack**: Go `html/template` + [htmx](https://htmx.org/) v2 + [BeerCSS](https://www.beercss.com/) (Material Design 3)
- **Font**: [Geist Variable](https://vercel.com/font) via `@fontsource-variable/geist` (woff2 files copied to `static/fonts/` at build time). Overrides BeerCSS's `--font`.
- **Package manager**: Always use `pnpm`, never npm or yarn.
- **BeerCSS** is vendored in `static/vendor/` (`beer.min.css`, `beer.min.js`, `material-dynamic-colors.min.js`). Use BeerCSS components, grid, spacing, and typography classes natively — do not reinvent what BeerCSS already provides. When building layouts, study BeerCSS docs for correct class usage.
- **Material theme** generated from seed color `#1c4bea` via `ui("theme", "#1c4bea")` in `static/js/app.js` at page load.
- **Light/dark mode** toggle saved in `localStorage` under key `specd-theme`. Restored on load, falls back to system `prefers-color-scheme`.

### Template Layout System

- **Layouts**: Two layout templates in `templates/layouts/`:
  - `base.html` — full page shell (`<html>`, `<head>`, `<body>`). Renders `<title>`, meta tags, nav, footer, and the `{{block "content" .}}` slot. Used on initial page load.
  - `partial.html` — lightweight wrapper for htmx partial responses. Renders `<title>` (htmx natively processes `<title>` tags in responses to update `document.title`) followed by `{{block "content" .}}`. Used on htmx navigation so page metadata updates without a full reload.
- **Partials**: `templates/partials/nav.html` (desktop sidebar + mobile top bar + mobile drawer), `templates/partials/footer.html`.
- **Pages**: `templates/pages/*.html` — each defines `{{define "content"}}...{{end}}` to override the content block. Current pages: `welcome`, `tasks`, `task_detail`, `specs`, `spec_detail`, `kb`, `kb_detail`, `search`, `settings`, `docs`, `tutorial`.
- **Detail pages**: `/specs/{id}`, `/tasks/{id}`, `/kb/{id}` use Go 1.22+ `r.PathValue("id")` for routing. Each handler reuses the same loader as the corresponding CLI `get-*` command (`LoadSpecDetail` in `cmd/get_spec.go`, `LoadTaskDetail` in `cmd/handlers_task_detail.go`, `loadKBDetailPage` in `cmd/handlers_kb_detail.go`). When adding a new detail page, extract the load logic into an exported `Load<Kind>Detail(db, id)` function so the CLI and web stay in sync. Search-result links must point at these routes via `searchResultHref` (`cmd/templates.go`).
- **PageData struct** (`cmd/templates.go`): `Title`, `Active` (nav highlighting), `DevMode`, `CSSHash`, `JSHash`, `Data` (page-specific payload).
- **htmx-aware rendering**: `renderPage()` checks the `HX-Request` header. Full page load → renders via `base.html`. htmx navigation → renders via `partial` (content + metadata). This keeps metadata in the layout layer — to add per-page metadata (e.g. description, OG tags), add it to `PageData` and render in both `base.html` and `partial.html`.
- **Template FuncMap** (`cmd/templates.go`): `isActive` (active-section check), `searchResultHref` (search result links), `fromSlug` (turn `non_functional` → "Non Functional" for display). Add helpers here rather than precomputing in handlers.
- **Navigation**: All nav links use `hx-get`, `hx-target="#main-content"`, `hx-swap="innerHTML"`, `hx-push-url="true"` for SPA-like navigation with clean URLs. **Do NOT add `hx-select`** when targeting `#main-content` — the htmx partial response (`partial.html`) is just `<title>` + content block, with no `#main-content` wrapper, so `hx-select="#main-content"` matches nothing and swaps in empty. `hx-select` is only correct when scoping to a sub-region (e.g. search uses `hx-select="#search-results"`, which exists inside the content block). Nav link sub-templates (`nav-links`, `nav-links-mobile`) are shared between desktop and mobile to avoid duplication.
- **Desktop sidebar toggle**: Hamburger toggles `max` class on the nav via `toggleSidebar()` in `static/js/app.js`. Persisted in `localStorage` under `specd-sidebar`.
- **Mobile drawer**: `<dialog class="left no-padding">` containing `<nav class="left max surface-container">`. BeerCSS handles the slide-in animation and overlay. Custom CSS (`#mobile-menu > nav`) overrides `position: static` and `block-size: 100%` so the inner nav fills the dialog.
- **Main width per route**: `templates/layouts/base.html` picks `<main class="responsive max">` for `/tasks` (kanban needs the full viewport) and `<main class="responsive">` (75rem cap, BeerCSS-centered) for every other route, conditioned on `.Active`. For `/tasks`, `.tasks-page` uses `inline-size: max-content; max-inline-size: 100%; margin-inline: auto` so the page header lines up with the kanban's left edge on wide screens (4K) and the kanban still scrolls horizontally when it overflows.
- **htmx form-submit + partial swap**: When a form mutates a region of the current page, point `hx-get`/`hx-post` to the same route and add `hx-target="#region"`, `hx-select="#region"`, `hx-swap="outerHTML"`, `hx-push-url="true"`. The handler renders the **full** page; htmx extracts just the matching region from the response. One handler serves bookmarked URLs and in-page updates — no separate fragment endpoint. Canonical example: `cmd/search_page.go` + `templates/pages/search.html` (`#search-results`).
- **Template scope inside `{{with .Data}}`**: `{{with .Data}}` rebinds `.` to the wrapped value, but `$` still refers to the **root** template context (`*PageData`), **not** the wrapped data. To reference a sibling field of the wrapped data from inside a nested `{{range}}`, save it to a local variable (`{{$kind := .Kind}}`) before entering the range, then use `$kind`. Writing `$.Kind` will silently miss because PageData has no such field — and Go templates error at execute time, not parse time, so the page renders blank or shows a 500.
- **Grouped/paginated lists — count badges**: When grouping a paginated list by category (specs by type, tasks by stage), the per-group count badge must show the **project-wide total** for that category, not `len(.Items)` of the current page. Otherwise the chip shrinks as the user pages through. Pattern: query `SELECT category, COUNT(*) FROM table GROUP BY category` once and pass the totals map alongside the page items. See `loadSpecTypeTotals()` + `SpecsGroup.Total` in `cmd/handlers_specs_page.go`.
- **Composable filter form**: when a page has multiple combinable filters (e.g. `/specs` view+type, `/tasks` filter), wrap them in **one form** with `hx-trigger="change"`, one `<select>` per filter, and any preserved state (`page`, `page_size`) in `<input type="hidden">`. htmx serializes the form into the URL on change so filters compose automatically (`/specs?view=cards&type=business&page=1&page_size=20`). Pagination links and other in-page navigation must spell out **every** filter dimension in their `hx-get` URL — missing one drops that filter. Canonical example: `templates/pages/specs.html`.
- **Allowlist-driven query filters**: validate query-string filters against a runtime allowlist (e.g. `ProjectConfig.SpecTypes` for `?type=…`) before they reach SQL or the template. Use a sentinel like `"all"` for "no filter". The SQL pattern `WHERE ?1 = ?2 OR col = ?1` (with `?2 = "all"`) keeps one query covering both filtered and unfiltered cases without dynamic SQL string building. Canonical example: `loadSpecsPage()` + `isAllowedSpecType()` in `cmd/handlers_specs_page.go`.
- **Always verify routes after template changes** — `go build`, `go test`, and the linters do not catch Go template field-resolution errors. Hit the route with `curl` or Playwright before declaring a template change done.

### Styling

**All frontend styling work — writing markup for new pages, adding a rule, adjusting spacing, overriding a BeerCSS default — must follow [`docs/internal/css-conventions.md`](docs/internal/css-conventions.md).** That document is the single source of truth and is organized in two parts:

- **Part 1 — Building UI**: how to lay out pages, apply spacing, toggle dark mode, and write component styles *using the existing class and layout system* (BeerCSS components + our directional utility classes). Read this before writing any template.
- **Part 2 — Extending the system**: how to add a CSS variable, a new spacing size/direction/breakpoint, or a new class when Part 1 genuinely can't cover the case. New additions are rare by design.

Non-negotiables enforced by that document (summarised here so tooling picks them up):

- **Use BeerCSS classes natively** for grid (`grid`, `s12`, `l6`), alignment (`top-align`, `center-align`), typography (`large-text`, `bold`), and components. Do not reinvent what BeerCSS ships.
- **BeerCSS components already have a default look — don't gild them.** `<article>` ships with surface fill, elevation, padding, and `.75rem` rounded corners; adding `class="border round no-elevate"` strips the fill/shadow and produces a ghost outline that looks broken. `<header>` is `display: grid` with `min-block-size: 4rem` and `align-content: center` — using it for an inline section title makes a 64px-tall band with text floating in the middle. For inline titles use `<nav class="row">` or a plain `<h6>`.
- **Avoid double borders.** When wrapping a `<table>` inside an `<article class="round">`, drop `class="border"` from the table — the article already provides containment.
- **Confirmation feedback uses BeerCSS snackbars.** Place `<div id="…" class="snackbar">…</div>` in the page and trigger with the global `ui("#…")` helper (auto-fades). Wire from htmx via `hx-on::after-request="if (event.detail.successful) ui('#…')"` and let the form respond with `hx-swap="none"`. Don't invent visually-hidden classes (no `.hidden` exists in this project) — use `aria-labelledby` to link a fieldset/article to an existing heading instead.
- **Articles wrap long tokens.** A project-wide `article { overflow-wrap: anywhere; }` rule lives in `app.scss` so user-supplied IDs/titles/URLs break at the rounded card edge instead of overflowing. Don't strip it — and any new card-shaped container outside `<article>` needs the same rule, or it will visibly bleed.
- **Cards use the canonical `tile` recipe.** Every card in specd (kanban tasks, specs cards view, search results, linked-task lists on detail pages) uses the same anchor + `<article class="tile surface border no-margin">` + `nav.row.no-padding.no-margin.tile-meta` chip row + `<h6 class="small no-margin">` heading pattern. Grid responsive columns (`s12 m6 l4`) go on the anchor wrapper, not the article. Copy-paste from `templates/partials/board.html` or `templates/pages/search.html`; do not invent a new card shape. Full recipe with class glossary and don'ts: [`docs/internal/css-conventions.md`](docs/internal/css-conventions.md) → "Cards (the canonical `tile` recipe)".
- **All project-level styles live in `static/css/src/app.scss`.** SCSS with variables, maps, mixins, and `@each` loops.
- **Spacing utilities are generated** from `$breakpoints`, `$size-keys`, and `$dirs` maps via `@include spacing-utilities($bp)`. Never hand-write utility classes — add to the maps instead.
- **Use the `space()` and `no-space()` mixins** in component selectors (e.g. `footer nav { @include space(mt, m); }`) to apply the same values the utility classes provide.
- **Every tunable value is a CSS variable** on `:root`. Never hardcode sizes, spacing, or colors in rule bodies.
- **Logical properties by default** (`margin-block-start`, `padding-inline`). Physical only when exactly mirroring a BeerCSS rule that uses physical (see `no-space` mixin).
- **Directional spacing uses utility classes** (`mb-s`, `px-l`, `m:mt-m`, …) in markup, or `@include space($dir, $size)` in SCSS. Do not hand-write `margin-block-end: var(--space-m) !important`.
- **Scope BeerCSS overrides to their containing element.** Keep footer-specific overrides under `footer ul`, `footer nav`, etc. — never publish a naked `ul { … }` or `nav { … }` rule that leaks into every component.

### CSS Build

- **Pipeline**: Sass (compile SCSS) → Lightning CSS (bundle + minify) → PurgeCSS (strip unused custom classes by scanning `templates/**/*.html`) → concatenate with BeerCSS. **PurgeCSS only touches our compiled SCSS**; BeerCSS is appended after, so vendor classes like `snackbar`, `active`, `chip`, `round` are never purged and never need a safelist. Only custom classes added dynamically by JS need safelisting (see `static/purgecss.config.cjs`).
- **Source**: `static/css/src/app.scss` — design tokens, SCSS maps/mixins, utility class generators, overrides.
- **Output**: `static/css/dist/app.css` — single bundled, purged, minified CSS file (BeerCSS + custom).
- **Build command**: `task css:build` (or `cd static && pnpm run build:css`).
- **`static/css/dist/.gitkeep`**: Required so `go:embed` works before first CSS build. The `css/dist/` directory is gitignored, but `.gitkeep` is force-tracked.
- **Package manager**: pnpm for CSS build deps only (`sass`, `lightningcss-cli`, `purgecss`).
- **CSS linting**: Stylelint with `stylelint-config-standard-scss`. Run `task css:lint` or `cd static && pnpm run lint`.

### Build & Embedding

- **Embedding**: `templates/` and `static/` are embedded in the Go binary via `go:embed` in `main.go`. `fs.Sub` strips the prefixes. `cmd/frontend.go` holds `templateFS` and `staticFS` variables.
- **Dev workflow**: Use `task qa` — initializes `tmp/qa/` on first run, then **reuses existing data** on subsequent runs so seeded specs/tasks/KB survive restarts. Delete `tmp/qa/` to force re-init. Builds CSS, starts Air (Go live reload on port 8000 with `--dev` flag). Open `localhost:8000`. Templates are re-parsed on every request in dev mode. Air watches `.go`, `.html`, `.css`, `.js` files.
- **Air `clean_on_exit` is `false`** in `.air-qa.toml`. With `tmp_dir = "tmp"` and `clean_on_exit = true`, Air would delete the **entire** `tmp/` directory on shutdown — including `tmp/qa/`. Keep this `false`; if you ever change `tmp_dir`, audit again.
- **Seed data**: `qa/specs/setup.sh` builds a realistic project at `/tmp/specd-qa/` — currently 68 specs (24 business / 22 functional / 22 nonfunctional, all categories paginate at default page size 20), 65 tasks across stages, 6 KB docs. Copy `/tmp/specd-qa/specd/`, `/tmp/specd-qa/.specd.json`, and `/tmp/specd-qa/.specd.cache` into `tmp/qa/` to populate the QA project (KB sync from markdown is not yet implemented, so KB rows must be inserted via SQL — the script does this). The script uses `make_spec` (5th arg = type) and `make_task` shell helpers so adding more rows is a one-line append.
- **`new-spec` defaults the type to `business`.** It does **not** accept a `--type` flag. To create non-business specs (in tests or seed scripts), call `update-spec --id SPEC-N --type functional|nonfunctional` immediately after `new-spec`. The `make_spec` helper in `qa/specs/setup.sh` does this automatically when given a type argument.
- **Live reload**: `--dev` flag on `specd serve` injects `livereload.js` which connects via SSE. When the server restarts (Air rebuild), the browser auto-reloads.
- **`specd serve --no-open`**: Suppresses browser auto-open and startup message. Used by Air in QA mode to avoid opening the browser on every Go rebuild.

### Code Standards

- **HTML**: Must be semantic, accessible, and SEO-friendly. Use `<article>`, `<nav>`, `<main>`, `<header>`, `<footer>`, proper heading hierarchy, `aria-label`, `role` attributes where needed.
- **Indentation**: 2 spaces for all frontend files (HTML, CSS, JS, JSON).

### Server-rendered htmx partials with client state

Some pages mix server-rendered htmx swaps with client-side interactivity (drag/drop, collapse toggles, theme/sidebar). The kanban board at `/tasks` is the reference implementation. When you build similar features:

- **Render partials, swap into a stable root.** Define a partial template (e.g. `templates/partials/board.html`) and serve it from a JSON-free endpoint (e.g. `GET /api/tasks/board`). The page shell wraps a single `<div id="…-root" hx-get="…" hx-trigger="load" hx-swap="innerHTML">` so the first paint and every subsequent re-render go through the same partial.
- **State-changing endpoints return the same partial.** `POST /api/tasks/move` re-renders and returns the board partial; the client just swaps it in. The server stays authoritative.
- **Re-bind listeners on every htmx swap.** The DOM is replaced wholesale, so listeners attached to old nodes vanish. Subscribe once on `document.body` for `htmx:afterSwap`, filter by the swap's target id, then run a `bind()` that wires drag/drop, click handlers, etc.
- **Persist UI state in `localStorage`, restore on bind.** Examples: `specd-theme`, `specd-sidebar`, `specd-kanban-collapsed`. Keep a single key per concern; serialise as a slug list (`"todo,blocked"`) or a simple flag.
- **JS-applied classes must be safelisted.** PurgeCSS only sees template HTML. Any class added at runtime (`dragging`, `drop-target`, `collapsed`, `kanban-placeholder`) goes into `static/purgecss.config.cjs` `safelist`.
- **Templates need a per-request fresh map in dev.** Use the `makeFreshPages(devMode, cached)` helper in `cmd/serve.go` so partial endpoints pick up template edits via Air without a full rebuild.

### Frontmatter ↔ DB ↔ UI sync

Files are ground truth, the SQLite cache is derived. When a UI action mutates state (kanban drag, criterion toggle):

1. Wrap DB writes in a single transaction, renumbering any ordered set densely (`0..n-1`).
2. After commit, call `rewriteTaskFile()` (or the analogue) for **every** entity whose persisted fields changed — not just the directly-edited one. Renumbered siblings need their `position:` frontmatter updated too.
3. The rewriter recomputes `content_hash` from the just-written bytes so the next `SyncCache()` is a no-op.
4. Re-render the partial from fresh DB state for the client. Don't hand-craft the response from the request inputs; round-trip through the same loader the page uses.

### URL-driven view state vs. localStorage

Distinguish *where* state lives based on whether it's shareable / bookmarkable:

- **URL query string** — view mode, filter, sort, pagination, tab selection. Anything you'd reasonably want to bookmark, copy-paste to a colleague, or restore via browser back/forward. Source of truth is `r.URL.Query()`; the page handler reads it, threads it through `Data`, and the template renders the active state. Buttons use `hx-get="/page?param=value" hx-target="#main-content" hx-swap="innerHTML" hx-push-url="true"`. Reference: `/specs?view=` (`cmd/handlers_specs_page.go`) and `/tasks?filter=` (`cmd/handlers_tasks_page.go`). Embed the current value into nested resources via a `data-…` attribute so JS handlers (e.g. drag/drop POSTs) can read it back without re-parsing the URL.
- **`localStorage`** — client-private UI state that should *not* affect URLs: theme (`specd-theme`), sidebar collapsed/expanded (`specd-sidebar`), per-column kanban collapse (`specd-kanban-collapsed`). Keys are namespaced `specd-<concern>`; values are the smallest serialisation that works (a flag, a slug list).

When in doubt: if two users in the same project would expect to see the same thing after pasting the URL, it's URL state.

### Mandatory vs. optional task stages — no hard logic on optional slugs

`RequiredTaskStages` (`Backlog, Todo, In progress, Done`) are guaranteed present. `OptionalTaskStages` (`Blocked, Pending Verification, Cancelled, Wont Fix`) are opt-in at `specd init` and **may be absent** from any project. Code that needs to act on stage *meaning* (e.g. "is this completed?") must derive it from a mandatory anchor, not a hardcoded optional slug.

- **Bad:** `if status == "cancelled" || status == "wont_fix" { … }` — breaks the moment someone opts out, and assumes meanings the user might disagree with.
- **Good:** position-based — anything at or after `Done` in the kanban order is "completed". `Done` is mandatory. Reference: `completedStageSlugs(stages)` in `cmd/handlers_tasks_board.go`.

Positional layout *preferences* for known optional stages (e.g. "Pending Verification before Done") are fine to encode by name, since they're hints that gracefully fall through when the stage is absent.

## Project Guard

- Most commands require an initialized project (`.specd.json` marker in cwd) and a globally configured username (`~/.specd/config.json`).
- `specd init` refuses to run in an already-initialized directory (checks for `.specd.json`).
- Exempt commands that work without initialization: `init`, `version`, `skills`, `help`.

## Skills Prerequisite

- **Every skill** must include a prerequisite section telling the AI to check for `.specd.json` and ask the user to run `specd init` in their terminal if missing. Skills must NOT run init themselves. The message must say exactly "Please run `specd init` in your terminal first" — do NOT suggest shell prefixes, prompt shortcuts (`!`), or any alternative execution method.
- Runtime-configurable values (e.g. `top_search_results`) must be read from `.specd.json` at runtime, not from build-time constants. Constants are only used as defaults during `specd init`.

