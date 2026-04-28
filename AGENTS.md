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
- **Frontend:** Vite + React 19 + [TanStack Router](https://tanstack.com/router) (file-based, CSR) + [shadcn/ui](https://ui.shadcn.com/) + [Tailwind v4](https://tailwindcss.com/) + [nanostores](https://github.com/nanostores/nanostores) + [lucide-react](https://lucide.dev/) + [react-markdown](https://github.com/remarkjs/react-markdown). Built to a static `frontend/dist/` and embedded in the Go binary.
- **Package manager:** pnpm (always use pnpm, never npm or yarn)
- You must write idiomatic Go and directory structure.
- You must use 2 spaces as indentation for non-Go code files.

## Project Structure

```
main.go              # Entrypoint (embeds skills/, frontend/dist/ via go:embed)
cmd/                 # Cobra commands (root.go, subcommands)
cmd/api.go           # RegisterAPI + JSON helpers (writeJSON/decodeJSON) — thin facade
cmd/api_meta.go      # GET /api/meta — project + startpage choices for SPA boot
cmd/api_specs.go     # GET /api/specs, /api/specs/{id} — read-only spec endpoints
cmd/api_tasks.go     # /api/tasks/* — list, board, detail, move, toggle, depends_on, delete
cmd/api_kb.go        # GET /api/kb, /api/kb/{id} — read-only KB endpoints
cmd/api_search.go    # GET /api/search — hybrid search wrapper
cmd/api_stats.go     # GET /api/stats — dashboard tile counts
cmd/api_settings.go  # POST /api/settings/* — UI-state mutations (default route, …)
cmd/spa_proxy.go     # Reverse proxy to Vite dev server in --spa-proxy mode
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
cmd/serve.go         # specd serve command (HTTP server: JSON API + embedded SPA, --spa-proxy in dev)
cmd/schema.sql       # Embedded SQLite schema (dynamic CHECK constraints)

frontend/                       # CSR SPA (Vite + React + TanStack Router + shadcn + Tailwind)
frontend/index.html             # static shell, mounts /src/main.tsx into #root
frontend/vite.config.ts         # vite + react + tailwind + tanstack-router plugins
frontend/src/main.tsx           # createRoot + RouterProvider entry
frontend/src/styles.css         # Tailwind v4 + shadcn tokens
frontend/src/routes/            # file-based routes (auto-generates routeTree.gen.ts)
frontend/src/components/ui/     # shadcn primitives (installed via shadcn CLI)
frontend/src/components/shell/  # AppShell, AppSidebar, PageHeader, RouteContextPane (Quick Search pane)
frontend/src/components/common/ # shared widgets
frontend/src/lib/api/           # one file per resource (specs, tasks, kb, search, meta, settings)
frontend/src/lib/api.ts         # fetchJSON helper
frontend/src/lib/stores/        # nanostores ($theme, $sidebarOpen, $searchOpen)
frontend/src/lib/nav.ts         # NAV_ITEMS, UTILITY_NAV_ITEMS, activeNavItem
frontend/src/content/           # static markdown (tutorial)
frontend/dist/                  # build output (gitignored except .gitkeep, embedded in Go binary)
frontend/public/                # favicons, manifest, robots

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
task build           # Build binary to bin/ (runs ui:build first so dist/ is fresh)
task run             # Build and run
task dev             # Live reload (uses air)
task qa              # Vite dev (5173) + Go --spa-proxy (8000) for full-stack QA
task test            # Run tests
task fmt             # Format all Go files (gofumpt + goimports + gci)
task fmt:check       # Check formatting without writing
task lint            # Run golangci-lint
task lint:fix        # Run golangci-lint with auto-fix
task sec             # Run all security checks
task sec:vulncheck   # Check deps for known vulnerabilities
task sec:gitleaks    # Scan for leaked secrets
task deadcode        # Find unreachable code from main
task check           # Run everything (fmt, lint, ui:lint, ui:typecheck, test, security)
task build:all       # Cross-compile for linux/darwin/windows (amd64+arm64)
task hooks:install   # Install lefthook git hooks
task clean           # Remove bin/, tmp/, and frontend/dist/

task ui:install      # pnpm install in frontend/
task ui:dev          # vite dev server (5173) only
task ui:build        # vite build → frontend/dist/
task ui:typecheck    # tsc --noEmit
task ui:lint         # eslint
```

## Git Hooks (via lefthook)

- **pre-commit** (parallel): format (gofumpt + goimports + gci), golangci-lint --fix, gitleaks. Frontend lint/typecheck is run via `task ui:lint` / `task ui:typecheck` (or `task check`), not from a hook.
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
- **Acceptance criteria language**: claims use must / should / is / will. Avoid may / might.
- **When the codebase outgrows `cmd/`** (~20+ files), extract domain logic into `internal/` packages. For now `cmd/` is fine for a Cobra CLI.
- **Frontend conventions** are documented in the "Frontend (frontend/)" section below.

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

- `specd serve` starts an HTTP server exposing a JSON API under `/api/*` and serving the embedded SPA from `frontend/dist/`.
- Port scanning starts at `DefaultServePort` (8000) and tries up to `MaxPortAttempts` (100) ports.
- Prints port scanning progress and the final URL to the terminal.
- Opens the user's default browser via `open`/`xdg-open`/`rundll32` depending on OS.
- In production (no `--spa-proxy` flag), `cmd/serve.go` serves the embedded SPA with an SPA-fallback rule: any non-`/api/*`, non-asset path returns `index.html` so TanStack Router takes over on the client.
- In dev (`--spa-proxy http://127.0.0.1:5173`), `cmd/spa_proxy.go` reverse-proxies all non-`/api/*` traffic to Vite. `task qa` wires this up.
- All page state mutations go through JSON endpoints under `/api/*` (e.g. `POST /api/tasks/move`, `POST /api/settings/default-route`). Mutations return the updated entity so the SPA can re-render off the response.
- **Form-handling discipline:** wrap `r.Body` with `http.MaxBytesReader(w, r.Body, MaxSettingsFormBytes)` before `r.ParseForm()` (gosec G120). When a value is constrained to an allowlist (e.g. `StartpageChoices`), resolve it to the canonical entry from the list **before** logging or persisting — this both rejects unknown values and satisfies gosec G706 taint analysis on `slog` calls.
- **Persisting UI settings:** read with `ReadMeta(db, key)`, write with `WriteMeta(db, key, value)` (upserts via `ON CONFLICT(key) DO UPDATE`). Keys live in `cmd/constants.go` (e.g. `MetaDefaultRoute`).
- **gocognit budget on `runServe`:** keep complexity ≤ 20 by extracting per-resource API handlers and shared helpers instead of inlining closures.

## Frontend (frontend/)

### Stack

- Vite + React 19 + TanStack Router (file-based routes) + shadcn/ui + Tailwind v4 + nanostores + lucide-react + react-markdown.
- **Pure CSR**, not SSR. TanStack Start was considered and rejected because Nitro emits a Node server bundle that conflicts with the embed-in-Go-binary goal.
- Package manager: pnpm only.

### Entry

- `frontend/index.html` is the static shell with `<div id="root">` and `<script type="module" src="/src/main.tsx">`.
- `main.tsx` calls `createRoot(rootEl).render(<RouterProvider router={router} />)`.
- **Do NOT add `shellComponent` to `__root.tsx`** — that pattern is TanStack Start SSR and is incompatible with this CSR setup; it caused a runtime hang. The root route uses `component: RootComponent` returning `<AppShell><Outlet/></AppShell>`. No `<html>`, `<head>`, or `<body>` inside React.

### Routing

- TanStack Router file-based routing in `src/routes/`. File names map to URL paths: `welcome.tsx` → `/welcome`, `specs.$id.tsx` → `/specs/:id`, `docs.tutorial.tsx` → `/docs/tutorial`.
- The `@tanstack/router-plugin/vite` plugin auto-generates `routeTree.gen.ts` on file events. **Do not edit that file**; it's regenerated on every build/dev cycle.
- `@/` absolute imports always — never relative paths in `frontend/`.

### App shell

- `src/components/shell/app-shell.tsx` mounts the global providers (TooltipProvider, SidebarProvider, Toaster) plus `AppSidebar` + `SidebarInset(PageHeader + Outlet)`. The Quick Search lives in pane 2 of the sidebar (`route-context-pane.tsx`); the binoculars button in `PageHeader` toggles the sidebar to expose it.
- **Sidebar layout** is based on the shadcn `sidebar-09` block (two panes — icon rail + context pane).
  - Rail body: Tasks, Specs, Knowledge.
  - Rail footer: Docs, Settings.
  - Pane 2 hosts a live search view (`route-context-pane.tsx`) that hits `/api/search`. Pane 2 is hidden on mobile (`md:flex` only).

### State

- nanostores by concern. `$theme` (`light|dark|system`), `$sidebarOpen`, `$searchOpen`. Persisted with the `specd-` localStorage prefix.
- **Theme application happens in a single `useEffect` in `app-shell.tsx`.** Do NOT also toggle the DOM class from the store subscription — duplicating it caused a render storm. The subscription only writes localStorage; the effect is the sole DOM toggler.
- Dark mode is applied by adding `class="dark"` to the `<html>` element from that effect.

### Components

- All shadcn primitives live in `src/components/ui/`. **Don't hand-roll anything shadcn already provides.** Run `pnpm dlx shadcn@latest add <name>` if a needed primitive isn't there yet.
- **`frontend/src/components/ui/` is off-limits for manual file editing.** Files in this directory are managed exclusively by the shadcn CLI (`pnpm dlx shadcn@latest add <name>` to add, `pnpm dlx shadcn@latest update` to upgrade). Do not hand-edit existing files there, do not hand-write new ones, do not “fix” a file by patching it. If a primitive is missing or buggy, fix it through the CLI; if a wrapper or composition is needed, build it in `src/components/` (outside `ui/`) and import the shadcn primitive into it.
- The list of installed components is whatever's currently in `src/components/ui/`.
- Icons come from `lucide-react` only. No Material Symbols. No inline SVGs unless Lucide genuinely lacks the icon.

### Theming

- shadcn CSS variables + Tailwind v4 in `src/styles.css`. No SCSS, no per-component CSS files.
- New design tokens go in `@theme` blocks in `styles.css`, not in ad-hoc stylesheets.

### Data fetching

- Plain `useEffect` + `AbortController` keyed on URL search params. **No TanStack Query, no SWR.**
- Resource wrappers in `src/lib/api/<resource>.ts` (specs, tasks, kb, search, meta, settings) use the central `fetchJSON<T>` helper from `src/lib/api.ts`.
- Every page reads from `/api/*`. JSON shapes are defined Go-side in `cmd/api_<resource>.go` and mirrored TS-side in `src/lib/api/<resource>.ts`. Keep them in lockstep.
- **One file per resource on the Go side.** `cmd/api.go` is the thin facade (RegisterAPI + JSON helpers); each resource lives in `cmd/api_<resource>.go`. When adding a new top-level path (e.g. `/api/foo/*`), create `cmd/api_foo.go` rather than appending to an existing file.
- Mutations always return the updated entity so the page can re-render off the response — never hand-craft state from request inputs.

### Optimistic updates

- Only for kanban DnD and criterion toggles. Read-only resources don't need optimism. Revert on error.

### Destructive confirmations

- Use shadcn `AlertDialog` for any irreversible action (delete spec, delete task, etc.). **Never use `window.confirm`** — the native dialog isn't themed and can't show context. Reference: `routes/specs.$id.tsx` Delete-task flow.
- The dialog state lives at the page level; the row's button only calls `onRequestDelete(item)` to open it. This keeps the click target a Button sibling of the row's Link (never nest a Button inside a Link — clicks fight).
- The Action button calls `event.preventDefault()` so the dialog stays open while the request is in flight; close it from the success branch and disable both buttons via a `deleting` flag.

### Slug-to-label formatting

Statuses and types are stored as lowercase underscore slugs (`in_progress`,
`wont_fix`, `nonfunctional`). For display, **never** rely on Tailwind's
`capitalize` alone — it only fixes the first letter and leaves the
underscore visible. Use:

- **Frontend:** `humanizeSlug(slug)` in `src/lib/format.ts` — replaces `_`
  and `-` with spaces and uppercases the first letter ("In progress",
  "Wont fix"). Mirrors the Go `FromSlug` helper.
- **Go:** `FromSlug(slug)` in `cmd/slug.go` — same semantics; use server-
  side when emitting a pre-rendered label like `status_label` on
  `apiTaskDetailResponse`.

`humanizeSlug` is preferred over CSS class transforms because it produces
correct output even when the slug contains multiple words and avoids
accessibility quirks (screen readers may pronounce underscores).

### Detail-route URL hygiene

Detail routes (`/specs/$id`, `/tasks/$id`) must declare
`validateSearch: () => ({})` so list-page filter state (`view=grouped&type=all&page=1&page_size=20`)
doesn't leak into the URL when the user navigates from a list. Without it,
TanStack Router preserves unknown search params on cross-route navigation.

Pair this with `search={{}}` on every `<Link to="/specs/$id">` /
`<Link to="/tasks/$id">` so click navigation produces a clean URL. The
"Back to list" links in detail pages keep the full search props because
they're navigating *to* the list and should preserve its state.

### In-place feedback for low-stakes actions

For frequent quiet actions (copying a hash, dismissing a row), prefer an
inline state swap over a global toast — toasts are noise. Reference:
`components/common/copyable-hash.tsx` toggles its own label between
`Hash: <truncated>` and `Copied` on click and resets after ~1.4 s. Use
toasts only for results that the user might miss visually (network errors,
async background save outcomes that don't have a nearby UI affordance).

### Markdown rendering

- Use the shared `Markdown` component (`components/common/markdown.tsx`), **not** raw `ReactMarkdown` — `@tailwindcss/typography` is not installed, so the `prose` class is dead. The component owns per-tag styling.
- For "one size smaller" body text (used on spec/task detail), wrap the call site with descendant utilities: `text-sm [&_h1]:text-2xl [&_h2]:text-xl [&_h3]:text-lg [&_h4]:text-sm [&_li]:leading-6 [&_p]:leading-6`. Don't bake size variants into the component until a third call site needs the same scaling.

### Timestamps

- Render every timestamp visible to a user with `formatRelativeTime(iso)` from `@/lib/format` ("3 minutes ago", "2 days ago", "in 4 hours"). It uses `Intl.RelativeTimeFormat` so plurals/locale work without manual branching.
- Always wrap the output in a `<time>` element: `<time dateTime={iso} title={formatDateTime(iso)}>{formatRelativeTime(iso)}</time>`. `dateTime` is for assistive tech and copy-paste; `title` exposes the absolute timestamp on hover.
- `formatRelativeTime` takes an optional `nowMs` second argument so tests can pin "now" without faking timers — see `src/lib/format.test.ts`.
- Don't add new "absolute date only" helpers (e.g. the old `formatDate`) — those are list-density artefacts that the relative form replaces.

### Search

- The sidebar's pane 2 (`route-context-pane.tsx`, "Quick Search") hosts a sticky live-search input that hits `/api/search`. The page-header binoculars button toggles the sidebar to expose it.
- The `/search` page is the full results view with filtering by kind and pagination. Search-related fetching uses the same `searchAll` helper in `lib/api/search.ts`.

### URL-driven view state vs. localStorage

State location is determined by whether it's shareable / bookmarkable:

- **URL search params** — view mode, filter, sort, pagination, tab selection. Anything you'd reasonably want to bookmark or share. Source of truth is TanStack Router's typed search via `Route.useSearch()`. Define schemas with `validateSearch` on the route. Reference: `routes/search.tsx` for `q`/`kind`/`page`.
- **`localStorage` (via nanostores)** — client-private UI state that should *not* affect URLs: `$theme` (`specd-theme`), `$sidebarOpen` (`specd-sidebar-open`), kanban column collapse state. Keys are namespaced `specd-<concern>`.

When in doubt: if two users in the same project would expect to see the same thing after pasting the URL, it's URL state.

### Frontmatter ↔ DB ↔ API sync

Files are ground truth, the SQLite cache is derived. When an API mutation changes state (kanban move, criterion toggle):

1. Wrap DB writes in a single transaction, renumbering any ordered set densely (`0..n-1`).
2. After commit, call `rewriteTaskFile()` (or the analogue) for **every** entity whose persisted fields changed — not just the directly-edited one. Renumbered siblings need their `position:` frontmatter updated too.
3. The rewriter recomputes `content_hash` from the just-written bytes so the next `SyncCache()` is a no-op.
4. Return the fresh entity from the loader (the same one the GET endpoint uses) — don't reconstruct response state from request inputs.

### Set-replacement endpoints for one-to-many collections

Mutating a many-to-many or one-to-many edge (e.g. `task.depends_on`,
`spec.linked_specs`) uses **PUT with the complete next set** rather than
add/remove deltas:

- `PUT /api/tasks/{id}/depends_on` with `{depends_on: ["TASK-2","TASK-3"]}`.
- `PUT /api/specs/{id}/linked_specs` with `{linked_specs: ["SPEC-2","SPEC-3"]}`.
- Empty array clears all rows.
- Each handler normalizes (uppercase, trim, dedupe), rejects self-references
  and unknown ids before any write, then mutates inside a single
  transaction. References: `apiSetTaskDependsOnHandler` in `cmd/api_tasks.go`
  (helpers: `normalizeDependsOnInput`, `verifyTasksExist`,
  `replaceTaskDependencies`); `apiSetLinkedSpecsHandler` in
  `cmd/api_specs.go` (helpers: `normalizeLinkedSpecsInput`,
  `verifySpecsExist`, `diffLinkedSpecs`, `applyLinkedSpecsDiff`).
- Always rewrite the affected markdown file (`rewriteTaskFile`,
  `rewriteSpecFile`) and return the freshly-loaded detail payload so the
  SPA can drop the response into state without an extra GET. The detail-
  payload assembly is shared with the GET handler via
  `buildTaskDetailResponse` / `buildSpecDetailResponse`.
- For **bidirectional** edges (e.g. `spec_links` is a from/to pair table)
  insert/delete BOTH directions and rewrite BOTH endpoints' markdown files
  so frontmatter stays consistent. Diff `current` vs `next` to minimise
  row churn rather than DELETE-all + INSERT-all.

The frontend mirrors this contract:

- One TS wrapper per resource (`setTaskDependsOn` in `lib/api/tasks.ts`,
  `setLinkedSpecs` in `lib/api/specs.ts`) accepts the *complete* list and
  returns the updated detail response.
- The same wrapper is reused by both the inline trash-icon "remove" (e.g.
  `saveDependsOn(refs.filter(…).map(r => r.id))`) and the multi-select
  picker (`onConfirm={saveDependsOn}`). One write path, one network call
  shape.
- For the picker UI, follow the `DependsOnPicker` /
  `SpecLinkPicker` pattern in `components/tasks/depends-on-picker.tsx` and
  `components/specs/spec-link-picker.tsx`: the parent owns the candidate
  list and the API call; the dialog is network-free and emits `onConfirm`
  with the full next set. Lazy-load candidates on first open
  (`openLinkPicker`) so the page stays a single GET on initial load.
- The Card containing the relationship list renders **unconditionally** so
  the `+` affordance is always reachable; an empty-state paragraph nudges
  the user toward the picker.

### Card "+" / action slot

When a card needs an action button right-aligned with the title (e.g. the
`+` for the depends-on / linked-specs picker), use the shadcn
`<CardAction>` primitive — NOT a flexbox override on `<CardHeader>`.
shadcn's `CardHeader` is a CSS grid that auto-switches its columns to
`[1fr_auto]` when a child has `data-slot="card-action"` (which `CardAction`
sets); reaching for `flex-row justify-between` on the header fights that
grid and produces stacked or misaligned children. Reference:
`<LinkedSpecs>` in `routes/specs.$id.tsx` and the depends-on header in
`routes/tasks.$id.tsx`.

### Resolving id-only references in detail responses

Internal models often store relationships as `[]string` IDs (`task.depends_on`, `task.linked_tasks`). The CLI is happy with that, but the SPA needs titles for human-readable UI. Pattern:

1. Keep the underlying model field as-is (`[]string`) so CLI consumers don't break.
2. Add a sibling `*_refs` field on the API detail response with `[]apiTaskRef{ID, Title}` (or analogous). Don't overload the existing field.
3. In the handler, fan out one batch query (`SELECT id, title FROM ... WHERE id IN (?,?,…)`) — never N+1. Use `loadTaskRefs` in `cmd/api_tasks.go` as the reference implementation; preserve input order, fall back to `{ID, ""}` for missing IDs so the UI can render a broken link instead of dropping it.
4. The frontend type adds the new field; the route reads `data.foo_refs.map(...)` and shows `{ref.id} — {ref.title}`.

### Mandatory vs. optional task stages — no hard logic on optional slugs

`RequiredTaskStages` (`Backlog, Todo, In progress, Done`) are guaranteed present. `OptionalTaskStages` (`Blocked, Pending Verification, Cancelled, Wont Fix`) are opt-in at `specd init` and **may be absent** from any project. Code that needs to act on stage *meaning* (e.g. "is this completed?") must derive it from a mandatory anchor, not a hardcoded optional slug.

- **Bad:** `if status == "cancelled" || status == "wont_fix" { … }` — breaks the moment someone opts out, and assumes meanings the user might disagree with.
- **Good:** position-based — anything at or after `Done` in the kanban order is "completed". `Done` is mandatory. Reference: `completedStageSlugs(stages)` in `cmd/handlers_tasks_board.go`.

Positional layout *preferences* for known optional stages (e.g. "Pending Verification before Done") are fine to encode by name, since they're hints that gracefully fall through when the stage is absent.

### Build

- `task ui:build` → `frontend/dist/` (~150 KB gz JS, code-split per route, ~15 KB gz CSS).
- `task build` runs `ui:build` first so the Go binary embeds a fresh `dist/`.
- `frontend/dist/.gitkeep` is force-tracked so `go:embed all:frontend/dist` works before the first build.

### Dev

- `task qa` runs `vite` (5173) + `specd serve --dev --spa-proxy http://127.0.0.1:5173` (8000).
- Vite handles HMR; the Go server proxies all non-`/api/*` traffic to Vite.
- Air watches Go files only. UI changes go through Vite HMR — no Go rebuild required.

### Embed

- `frontend/dist/` is embedded into the Go binary via `//go:embed all:frontend/dist` in `main.go`.
- In production (no `--spa-proxy` flag), `cmd/serve.go` serves the embedded SPA with an SPA-fallback rule: any non-asset, non-`/api/*` path returns `index.html` so TanStack Router takes over on the client.

### Code Standards

- **`@/` absolute imports always**, never relative paths in `frontend/`.
- **2-space indent** for non-Go files (TS, TSX, JSON, YAML, HTML, CSS).
- **Semantic HTML**: real `<header>`, `<main>`, `<section>`, `<nav>`, proper heading hierarchy, `aria-label` on icon-only buttons, `aria-current="page"` on active nav, `aria-hidden="true"` on decorative icons, `aria-busy` on saving forms, `aria-live="polite"` on result regions.
- **Responsive by default**: mobile-first; cards stack to one column under `md`; tables wrap in `overflow-x-auto`; pane 2 of the sidebar is `hidden md:flex`.
- **Tailwind utilities + shadcn variants only.** No BeerCSS class names, no SCSS, no custom CSS files. Layout via `grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3`, `flex items-center justify-between`, etc.

## Project Guard

- Most commands require an initialized project (`.specd.json` marker in cwd) and a globally configured username (`~/.specd/config.json`).
- `specd init` refuses to run in an already-initialized directory (checks for `.specd.json`).
- Exempt commands that work without initialization: `init`, `version`, `skills`, `help`.

## Skills Prerequisite

- **Every skill** must include a prerequisite section telling the AI to check for `.specd.json` and ask the user to run `specd init` in their terminal if missing. Skills must NOT run init themselves. The message must say exactly "Please run `specd init` in your terminal first" — do NOT suggest shell prefixes, prompt shortcuts (`!`), or any alternative execution method.
- Runtime-configurable values (e.g. `top_search_results`) must be read from `.specd.json` at runtime, not from build-time constants. Constants are only used as defaults during `specd init`.
