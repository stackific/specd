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
- **CSS build:** [Lightning CSS](https://lightningcss.dev/) (bundle + minify) + [PurgeCSS](https://purgecss.com/) (tree-shake unused classes)
- **CSS linting:** [Stylelint](https://stylelint.io/) (standard config)
- **Package manager:** pnpm (for CSS build/lint deps in `static/`)
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
cmd/serve.go         # specd serve command (HTTP server with SPA + REST API)
cmd/schema.sql       # Embedded SQLite schema (dynamic CHECK constraints)
templates/           # Go HTML templates (layouts, partials, pages)
templates/layouts/   # Base layout with content block
templates/partials/  # Shared partials (nav, footer)
templates/pages/     # Page templates (override content block)
static/              # Static assets (embedded in Go binary)
static/vendor/       # Vendored JS (htmx, BeerCSS, Material Dynamic Colors)
static/fonts/        # Geist Variable font files (woff2)
static/images/       # Favicons, logos, manifest, robots.txt
static/css/src/      # CSS source (app.css with utilities)
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
- **Frontend conventions** are documented in the "Frontend (ui/)" section below.

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

- **`.specd.json`** — project config marker at the repo root. Committed to git. Contains dir name, spec types, task stages, search settings.
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
- Route `/` reads `default_route` from the `meta` table and issues a 307 redirect (default: `/docs/tutorial`).
- REST API routes live under `/api/` prefix (e.g. `/api/meta/default-route`).
- Page routes render Go templates with htmx partial support (HX-Request header → content block only).
- Static assets (CSS, JS, fonts, vendor libs) are served from the embedded `static/` filesystem.
- `--dev` flag enables live reload: re-parses templates on every request and injects an SSE-based reload script.

## Frontend (templates/ + static/)

- **Stack**: Go `html/template` + [htmx](https://htmx.org/) v2 + [BeerCSS](https://www.beercss.com/) (Material Design 3)
- **Font**: [Geist Variable](https://vercel.com/font) self-hosted in `static/fonts/`. Overrides BeerCSS's `--font`.
- **BeerCSS** is vendored in `static/vendor/` (`beer.min.css`, `beer.min.js`, `material-dynamic-colors.min.js`). Use BeerCSS components, grid, spacing, and typography classes natively — do not reinvent what BeerCSS already provides.
- **Material theme** generated from seed color `#1c4bea` via `ui("theme", "#1c4bea")` in `static/js/app.js` at page load.
- **Light/dark mode** toggle saved in `localStorage` under key `specd-theme`. Restored on load, falls back to system `prefers-color-scheme`.

### Template Layout System

- **Layouts**: Two layout templates in `templates/layouts/`:
  - `base.html` — full page shell (`<html>`, `<head>`, `<body>`). Renders `<title>`, meta tags, nav, footer, and the `{{block "content" .}}` slot. Used on initial page load.
  - `partial.html` — lightweight wrapper for htmx partial responses. Renders `<title>` (htmx natively processes `<title>` tags in responses to update `document.title`) followed by `{{block "content" .}}`. Used on htmx navigation so page metadata updates without a full reload.
- **Partials**: `templates/partials/nav.html` (desktop sidebar + mobile top bar + mobile drawer), `templates/partials/footer.html`.
- **Pages**: `templates/pages/*.html` — each defines `{{define "content"}}...{{end}}` to override the content block. Current pages: `welcome`, `tasks`, `specs`, `kb`, `search`, `settings`, `docs`, `tutorial`.
- **PageData struct** (`cmd/templates.go`): `Title`, `Active` (nav highlighting), `DevMode`, `Data` (page-specific payload).
- **htmx-aware rendering**: `renderPage()` checks the `HX-Request` header. Full page load → renders via `base.html`. htmx navigation → renders via `partial` (content + metadata). This keeps metadata in the layout layer — to add per-page metadata (e.g. description, OG tags), add it to `PageData` and render in both `base.html` and `partial.html`.
- **Template FuncMap**: `isActive` — used in nav partials for highlighting the active section.
- **Navigation**: All nav links use `hx-get`, `hx-target="#main-content"`, `hx-swap="innerHTML"`, `hx-push-url="true"` for SPA-like navigation with clean URLs. Nav link sub-templates (`nav-links`, `nav-links-mobile`) are shared between desktop and mobile to avoid duplication.
- **Nav structure**: Tasks (`task_alt`), Specs (`description`), KB (`menu_book`), Search (`search`) at top → `.max` spacer → Settings (`settings`), Docs (`article`), theme toggle at bottom.
- **Desktop sidebar toggle**: Hamburger toggles `max` class on the nav via `toggleSidebar()` in `static/js/app.js`. Persisted in `localStorage` under `specd-sidebar`.
- **Mobile drawer**: `<dialog class="left no-padding">` containing `<nav class="left max surface-container">`. BeerCSS handles the slide-in animation and overlay. Custom CSS (`#mobile-menu > nav`) overrides `position: static` and `block-size: 100%` so the inner nav fills the dialog.

### Styling

- **Use BeerCSS classes natively** for grid (`grid`, `s12`, `l6`), alignment (`top-align`, `center-align`), typography (`large-text`, `bold`), and components. Do not reinvent what BeerCSS ships.
- **All project-level styles live in `static/css/src/app.css`.** Plain CSS — no SCSS preprocessor.
- **Every tunable value is a CSS variable** on `:root`. Never hardcode sizes, spacing, or colors in rule bodies.
- **Logical properties by default** (`margin-block-start`, `padding-inline`).
- **Directional spacing uses utility classes** (`mb-s`, `px-l`, `m:mt-m`, …) in markup. All utilities are pre-generated in `app.css`.
- **Scope BeerCSS overrides to their containing element.** Keep footer-specific overrides under `footer ul`, `footer nav`, etc.

### CSS Build

- **Pipeline**: Lightning CSS (bundle + minify) → PurgeCSS (strip unused BeerCSS classes by scanning `templates/**/*.html`).
- **Source**: `static/css/src/app.css` — font-face declarations, design tokens, utility classes, overrides.
- **Output**: `static/css/dist/app.css` — single bundled, purged, minified CSS file.
- **Build command**: `task css:build` (or `cd static && pnpm run build:css`).
- **`static/css/dist/.gitkeep`**: Required so `go:embed` works before first CSS build. The `css/dist/` directory is gitignored, but `.gitkeep` is force-tracked.
- **Package manager**: pnpm for CSS build deps only (`lightningcss-cli`, `purgecss`).

### Build & Embedding

- **Embedding**: `templates/` and `static/` are embedded in the Go binary via `go:embed` in `main.go`. `fs.Sub` strips the prefixes. `cmd/frontend.go` holds `templateFS` and `staticFS` variables.
- **Dev workflow**: Use `task qa` — initializes a fresh `tmp/qa/` project, builds CSS, starts Air (Go live reload on port 8000 with `--dev` flag). Open `localhost:8000`. Templates are re-parsed on every request in dev mode. Air watches `.go`, `.html`, `.css`, `.js` files.
- **Live reload**: `--dev` flag on `specd serve` injects `livereload.js` which connects via SSE. When the server restarts (Air rebuild), the browser auto-reloads.
- **`specd serve --no-open`**: Suppresses browser auto-open and startup message. Used by Air in QA mode to avoid opening the browser on every Go rebuild.

### Code Standards

- **HTML**: Must be semantic, accessible, and SEO-friendly. Use `<article>`, `<nav>`, `<main>`, `<header>`, `<footer>`, proper heading hierarchy, `aria-label`, `role` attributes where needed.
- **Indentation**: 2 spaces for all frontend files (HTML, CSS, JS, JSON).

## Project Guard

- Most commands require an initialized project (`.specd.json` marker in cwd) and a globally configured username (`~/.specd/config.json`).
- `specd init` refuses to run in an already-initialized directory (checks for `.specd.json`).
- Exempt commands that work without initialization: `init`, `version`, `skills`, `help`.

## Skills Prerequisite

- **Every skill** must include a prerequisite section telling the AI to check for `.specd.json` and ask the user to run `specd init` in their terminal if missing. Skills must NOT run init themselves. The message must say exactly "Please run `specd init` in your terminal first" — do NOT suggest shell prefixes, prompt shortcuts (`!`), or any alternative execution method.
- Runtime-configurable values (e.g. `top_search_results`) must be read from `.specd.json` at runtime, not from build-time constants. Constants are only used as defaults during `specd init`.

