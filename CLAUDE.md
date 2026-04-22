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
- **Frontend:** Custom CSS framework + HTML (planned, not yet implemented)
- You must write idiomatic Go.
- You must use 2 spaces as indentation for Non-Go code file

## Project Structure

```
main.go              # Entrypoint (embeds skills/ via go:embed)
cmd/                 # Cobra commands (root.go, subcommands)
cmd/constants.go     # All magic strings and constants (single source of truth)
cmd/config.go        # Global (~/.specd/config.json) and project (.specd.json) config
cmd/database.go      # SQLite initialization, ID counters, project DB helpers
cmd/search.go        # Hybrid BM25 + trigram search across specs, tasks, KB
cmd/sync.go          # Cache refresher — reconciles spec markdown files with the DB
cmd/slug.go          # ToSlug (underscore), ToDashSlug (dash), FromSlug (display)
cmd/providers.go     # AI provider definitions (Claude, Codex, Gemini)
cmd/new_spec.go      # specd new-spec command
cmd/update_spec.go   # specd update-spec command
cmd/schema.sql       # Embedded SQLite schema (dynamic CHECK constraints)
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
task check           # Run everything (fmt, lint, test, security)
task build:all       # Cross-compile for linux/darwin/windows (amd64+arm64)
task hooks:install   # Install lefthook git hooks
task clean           # Remove bin/ and tmp/
```

## Git Hooks (via lefthook)

- **pre-commit** (parallel): format (gofumpt + goimports + gci), golangci-lint --fix, gitleaks
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
- **Frontend work** (custom CSS framework, HTML, templates) comes later. Don't scaffold it yet.

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
- Sync walks `<specd-folder>/specs/*/spec.md`, parses frontmatter, computes SHA-256 of the **full file** (frontmatter + body), and reconciles: insert new, update changed (by hash), delete removed, reconcile `spec_links` from `linked_specs` frontmatter.
- FTS and trigram indexes are maintained automatically via database triggers — sync only touches the `specs`, `spec_links`, and `spec_claims` tables.
- `update-spec` rewrites the entire spec.md from DB state via `rewriteSpecFile()` after any change, keeping the file as ground truth.
- **Spec title** is the first `# Heading` in the body, NOT a frontmatter field. `extractH1Title()` parses it. `buildSpecMarkdown()` writes it as `# Title` after the `---` delimiter.
- **Acceptance criteria** (claims) are parsed from `## Acceptance Criteria` section bullets using must/should/may/might language. Stored in `spec_claims` table with a dedicated `spec_claims_fts` FTS5 index for searching claims independently.
- Spec frontmatter includes `id`, `slug`, `type`, `summary`, `position`, `linked_specs`, `created_by`, `created_at`, `updated_at`. **No `title` field** — it's in the body.
- Spec body must have exactly one `# Title` (H1). No other H1 headings allowed. Use `##` for top-level sections, `###`–`######` freely within sections. `## Acceptance Criteria` must be H2.
- **Validation**: `parseSpecMarkdown` validates required fields (`id`, `slug`, `title` from H1, `type`, `summary`). Invalid specs are silently skipped.
- `update-spec` supports `--unlink-specs` and `--unlink-kb-chunks` to remove references. Response includes full summaries for linked specs and KB chunks, not just IDs.
- **Contradiction detection**: `specd search-claims --query "..." --exclude SPEC-X` searches `spec_claims_fts` for matching claims from other specs. Both `/specd-new-spec` and `/specd-update-spec` skills use this in step 3 to find and report conflicting acceptance criteria across specs. The AI evaluates matches — no automated resolution.
- Exempt commands (`init`, `version`, `skills`, `help`, `logs`) skip the sync.
- See `docs/internal/spec-markdown-schema.md` for the spec file format.

## File Conventions

- **`.specd.json`** — project config marker at the repo root. Committed to git. Contains folder name, spec types, task stages, search settings.
- **`<specd-folder>/`** (default: `specd/`) — contains spec markdown files. Committed to git.
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

- **Two slug formats** — do not mix them:
  - `ToSlug()` → underscore-separated (e.g. `pending_verification`). Used for config values: spec types, task stages. These are stored in `.specd.json` and used in SQL CHECK constraints.
  - `ToDashSlug()` → dash-separated (e.g. `user-authentication`). Used for content identifiers: spec slugs, task slugs, KB slugs. These appear in filenames and URLs.
- **Never use `ToSlug()` for content identifiers** — always use `ToDashSlug()`.

## Project Guard

- Most commands require an initialized project (`.specd.json` marker in cwd) and a globally configured username (`~/.specd/config.json`).
- Exempt commands that work without initialization: `init`, `version`, `skills`, `help`.

## Skills Prerequisite

- **Every skill** must include a prerequisite section telling the AI to check for `.specd.json` and ask the user to run `specd init` in their terminal if missing. Skills must NOT run init themselves. The message must say exactly "Please run `specd init` in your terminal first" — do NOT suggest shell prefixes, prompt shortcuts (`!`), or any alternative execution method.
- Runtime-configurable values (e.g. `top_search_results`) must be read from `.specd.json` at runtime, not from build-time constants. Constants are only used as defaults during `specd init`.

