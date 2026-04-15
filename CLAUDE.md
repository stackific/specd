Read AGENTS.md

# Rules

- When reference files exist (e.g. `ref/`), copy them directly instead of rewriting from scratch. Do not "adapt" or "simplify" — use the original as-is and only make the specific changes requested.
- Before modifying any file, read how it is actually used in context (imports, consumers, layout). Do not assume.
- Do not empty, gut, or zero out config values (like `social: {}`) unless explicitly asked to. Preserve existing data.
- Do not drop CSS files, imports, or structural elements from the original. If unsure whether something is needed, check the ref or ask.
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
- `styles/vite.config.js` build pipeline
- `styles/public/vendor/beer.min.css` and the vendoring approach

# Go template conventions

- Templates in `templates/` (layouts, partials, pages) — embedded via `embed.go`
- `base.html` defines `{{block "content" .}}{{end}}` — each page overrides this block
- Partials: `{{define "name"}}...{{end}}`, included with `{{template "name" .}}`
- All templates receive `PageData` struct with `.Title` and `.CSSFile`
- htmx-aware handlers check `HX-Request` header: present → content partial only; absent → full page with base layout
