# STRICT CONSTRAINTS
- Must use 2 spaces as indentation
- Must use pnpm as the package manager
- Must implement Semantic HTML, Accessibility, W3C Valid markup and SEO-friendly structure when implement pages

## Stack
- Astro (static mode)
- BeerCSS (Material Design 3 CSS framework)
- material-dynamic-colors
- LightningCSS (minifier only — see CSS section)

## BeerCSS Reference

A full BeerCSS reference lives at `docs/BEERCSS.md` in the project root.

### BeerCSS breakpoints

BeerCSS uses three responsive size classes (`s`, `m`, `l`) that map to these viewport widths:

| Class | Name    | Viewport width |
|-------|---------|----------------|
| `s`   | mobile  | up to 600px    |
| `m`   | tablet  | at least 601px |
| `l`   | desktop | at least 993px |

Use them in markup with the standard BeerCSS responsive helpers (e.g. `s12 m6 l4` on a grid cell, or `s m` / `l` on `<nav>` to show different navs per breakpoint).

## CSS rules

**The full CSS architecture lives in `src/styles/README.md` — read it before touching any styles. It explains the layers, the cascade, the naming convention, and the build pipeline.** This section is the operational summary of what you must follow as an agent.

### Where styles live

```
public/vendor/beer.min.css                       BeerCSS, served as its own <link>
src/styles/index.css                             Custom CSS entrypoint (foundation)
src/styles/{tokens,base,utilities,layout}.css    Foundation layer
src/components/<Name>.astro + <Name>.css         Component CSS, co-located
src/pages/<page>.astro + <page>.css              Page CSS, co-located
```

### Hard rules

1. **NEVER write CSS inside a `<style>` block in an `.astro` file** for production code. Always extract to a co-located `.css` file and import it from the component/page frontmatter:
   ```astro
   ---
   import "./MyComponent.css";
   ---
   ```

2. **NEVER use `<style is:raw>`.** It does NOT make CSS global; it only skips template processing. Selectors are still scoped to the component, which silently breaks selector matching for child components. Use `is:global` if you absolutely must keep CSS in an `.astro` file (you almost never should), or extract to a `.css` file (preferred).

3. **NEVER write an unprefixed class name** in any custom `.css` file. Every selector must be namespaced. The bundle is global and unprefixed classes will collide.

4. **BeerCSS owns COMPONENTS only.** Use BeerCSS for buttons, fields, chips, dialogs, cards, articles, grid, navs, snackbars — anything that's a Material Design 3 UI component. Do NOT use BeerCSS for spacing, sizing, typography, margin/padding, or color outside of role tokens.

5. **Sizing, padding, margin, gap, typography, and custom colors come from our framework.** Use the utility classes generated from the token scales in `src/styles/utilities.css` — `.p-3`, `.m-3`, `.mt-2`, `.gap-3`, `.h-16`, `.w-full`, `.min-h-15`, `.text-3`, `.fw-5`, `.lh-4`, `.ls-2`, `.round-3`, `.ratio-16-9`, `.container-md`. Tokens live in `src/styles/tokens.css`. Never add a new utility by hand — edit `scripts/gen-utilities.mjs` and run `pnpm gen:utilities`.

6. **NEVER hard-code a `font-family` declaration in any production CSS.** The single global font is `--font` (set to Geist Variable in `src/styles/tokens.css`). Every text element inherits it from the body. The only exception is when an icon font face is required (e.g. `font-family: "Material Symbols Rounded"` for icon glyphs). Don't import IBM Plex, Inter, Georgia, JetBrains Mono, or any other font in production components — if you find yourself reaching for a different font for a stylistic accent, propose updating the global instead.

7. **NEVER `import "beercss"`** from any `.astro` script. The package's `index.js` re-imports `beer.min.css` as a side effect, which will double-bundle BeerCSS. Import the JS function directly from the compiled file:
   ```js
   import { ui } from "beercss/dist/cdn/beer.min.js";
   ```

### Naming convention (mandatory for all custom CSS)

#### Token / utility step naming

Tokens and the utility classes that consume them are paired 1:1 with the same step name.

- **Tokens** (CSS custom properties): `--<scale>-<step>` kebab-case
  Examples: `--space-3`, `--text-2`, `--lh-3`, `--ls--1`, `--c-primary-40`
- **Utility classes**: short verb + step
  Examples: `.p-3`, `.mt-2`, `.text-3`, `.lh-3`, `.ls-2`, `.round-4`, `.ratio-16-9`, `.container-md`

Step numbers always go up = bigger / heavier / further. Step `0` is the baseline. Negative steps are written with a leading `-` on the step body and the scale-to-step separator dash stays in place — so step −1 of `space` is `--space--1` and the class is `.m--1`. Examples:

| Step | Token | Class | Computed |
|---|---|---|---|
| 2 | `--space-2` | `.m-2` / `.p-2` | `0.5rem` |
| 0 | `--space-0` | `.m-0` / `.p-0` | `0` |
| −1 | `--space--1` | `.m--1` | `-0.25rem` |
| −2 | `--space--2` | `.m--2` | `-0.5rem` |

#### Spacing utilities are Tailwind-style (m / p only)

Margin and padding use single-letter directional shortcuts (no dash between the verb and direction):

| Class | CSS | Class | CSS |
|---|---|---|---|
| `m-3` | `margin: var(--space-3)` | `p-3` | `padding: var(--space-3)` |
| `mt-3` | `margin-top` | `pt-3` | `padding-top` |
| `mr-3` | `margin-right` | `pr-3` | `padding-right` |
| `mb-3` | `margin-bottom` | `pb-3` | `padding-bottom` |
| `ml-3` | `margin-left` | `pl-3` | `padding-left` |
| `mx-3` | `margin-inline` | `px-3` | `padding-inline` |
| `my-3` | `margin-block` | `py-3` | `padding-block` |

Other utilities keep the dash separator: `gap-x-3`, `gap-y-3`, `text-3`, `lh-4`, `ls-2`, `h-16`, `w-full`, `min-h-10`, `max-w-15`.

#### Line height scale

`--lh-0` is `1` (matches BeerCSS's old `no-line`), `--lh-4` is `1.5` (the body baseline). Steps `1`–`6` go `1.1, 1.25, 1.375, 1.5, 1.75, 2`. Step `--lh--1` is `0.95` for ultra-tight display headlines.

#### Responsive variants

Mobile-first, min-width:

| Prefix | When it applies | Underlying media |
|---|---|---|
| (none) | always | — |
| `m:` | tablet and up | `@media (--tablet)` ≥601px |
| `l:` | desktop and up | `@media (--desktop)` ≥993px |

The colon in the class name is a literal `:`. Selectors in `.css` files escape it as `\:` (CSS spec). HTML stays clean:

```html
<div class="p-3 m:p-5 l:p-7">…</div>
```
```css
.p-3      { padding: var(--space-3); }
.m\:p-5   { padding: var(--space-5); }    /* tablet+ */
.l\:p-7   { padding: var(--space-7); }    /* desktop+ */
```

#### Component / page class naming

- **Component containers**: 2–3 letter prefix — e.g. `.c-event`, `.gfx-hero`, `.sc-card`
- **Component internals**: 2-letter prefix tied to the parent component, used on every internal element — e.g. inside `.c-event` use `.ev-title`, `.ev-photo`, `.ev-status`
- **Page-specific classes**: 2–4 letter page prefix matching the page name — e.g. `.home-hero`, `.about-grid`
- **State modifiers**: separate flat class — `.is-active`, `.is-hidden`, `.is-disabled`

Pick a unique 2-letter prefix per component. Grep `src/components/**/*.css` before claiming a new prefix to make sure no other component already uses it.

#### Color tokens

The custom palette is namespaced under `--c-*` to avoid colliding with BeerCSS's role tokens (`--primary`, `--on-primary`, etc.). Six tonal palettes × 18 tones each, generated from a brand seed via `@material/material-color-utilities`:

```css
--c-primary-0   ...  --c-primary-100
--c-secondary-0 ...  --c-secondary-100
--c-tertiary-0  ...  --c-tertiary-100
--c-neutral-0   ...  --c-neutral-100
--c-neutral-variant-0 ... --c-neutral-variant-100
--c-error-0     ...  --c-error-100
```

Use these for illustrations, gradients, and non-component surfaces. For Material role colors (button backgrounds, surface containers, etc.), keep using BeerCSS's `--primary`, `--surface`, etc. Do NOT redefine BeerCSS's role tokens.

The seed for the current project lives in `src/styles/tokens.css` (search for `Generated from seed`). To regenerate after a brand change: `pnpm gen:colors '#NEWHEX'` and paste the output into `src/styles/tokens.css`.

### Scaling factor

`--scale` on `:root` is the master density dial. Bumping it scales every rem-based size, space, and gap at once. Pixel-snapped sizes (`--size-px-*`), content widths in ch (`--size-content-*`), breakpoint widths, and color tokens are NOT scaled.

```css
.compact { --scale: 0.875 }   /* tighter */
.cozy    { --scale: 1.125 }   /* looser */
```

### Adding new things

| Need | Where to put it |
|---|---|
| New design token | append to `src/styles/tokens.css` under `:root` |
| New utility class or new step | edit `scripts/gen-utilities.mjs`, then run `pnpm gen:utilities` |
| New `@custom-media` breakpoint | append to `src/styles/media.css` |
| Regenerate color palette from a new seed | `pnpm gen:colors '#hex'`, paste into `tokens.css` |
| New rule for an existing component | edit the component's co-located `.css` |
| New component | create `MyThing.astro` + `MyThing.css` side-by-side, frontmatter import |
| New page styles | create `<page>.css` next to `<page>.astro`, frontmatter import |
| Update BeerCSS | follow `public/vendor/README.md` |

### Build pipeline (do not change without reading the README)

- `astro.config.mjs` sets `vite.build.cssCodeSplit: false` so all custom CSS bundles into one file
- `vite.css.transformer: "lightningcss"` with `drafts.customMedia: true` so `@custom-media` rules in `media.css` resolve at build time
- `vite.build.cssMinify: "esbuild"` for minification (LightningCSS is the transformer, esbuild is the minifier — see the comment in `astro.config.mjs` for why)
- `build.inlineStylesheets: "never"` forces external `<link>` instead of inlined `<style>` in head
- A custom Astro integration in `astro.config.mjs` runs PurgeCSS against `dist/_astro/style.<hash>.css` after build with a Tailwind-style extractor that allows `:` in class names (so `m:p-3`/`l:p-3` survive)
- BeerCSS is loaded via a hand-written `<link rel="stylesheet" href="/vendor/beer.min.css">` in `Layout.astro`'s `<head>`. It is NOT in the bundle. Do not add `import "beercss/dist/cdn/beer.min.css"` anywhere.

Every page in `dist/` should ship exactly TWO `<link rel="stylesheet">` tags: one for `/vendor/beer.min.css` and one for `/_astro/style.<hash>.css`. If you see more or fewer, something is wrong.

## Templating this project

This codebase is designed to be reused as a starter for new sites. The CSS framework, build pipeline, naming conventions, and `src/styles/` foundation are project-agnostic and should be left alone. The pieces below are project-specific and need to be updated when adopting the template.

### Per-project find-and-replace checklist

| What | Where |
|---|---|
| Brand color seed | `pnpm gen:colors '#NEWHEX'` → paste output into the COLOR section of `src/styles/tokens.css`. Also update the seed comment in `tokens.css` and the default seed in `scripts/gen-colors.mjs`. |
| Site identity (name, url, description, og image, theme color, author, social handles) | `src/site.config.ts` — single source of truth for SEO and JSON-LD. |
| Production hostname | `astro.config.mjs` (`site:` field) and `public/robots.txt` (sitemap URL). |
| Logo files | `public/logo.svg`, `public/logo-dark.svg`, `public/favicon*`, `public/apple-touch-icon.png`, `public/og-image.png`. |
| Layout chrome (nav links, alt text, theme color passed to `ui("theme", …)`) | `src/layouts/Layout.astro`. |
| Pages | `src/pages/index.astro`, `about.astro`, `services.astro`, etc. — replace or remove. |
| Components | Anything under `src/components/` is project-specific. Pick new 2-letter prefixes per component when starting fresh. |
| Contact form (Turnstile + AWS SES) | `src/pages/api/contact.ts` and the contact dialog markup + JS in `Layout.astro`. Remove both if the new project doesn't need a contact form, otherwise update env vars per `README.md`. |
| `package.json` `name` field | rename to match the new project. |

### What to keep as-is

- `src/styles/{index,reset,media,tokens,base,utilities,layout}.css` — the framework foundation.
- `scripts/gen-utilities.mjs`, `scripts/gen-colors.mjs` — the generators.
- `astro.config.mjs` build pipeline (LightningCSS transformer + PurgeCSS integration).
- `public/vendor/beer.min.css` and the vendoring approach.
- The naming convention and hard rules in this file.
- `src/styles/README.md`, `docs/BEERCSS.md`, `docs/UTILITIES.md`.