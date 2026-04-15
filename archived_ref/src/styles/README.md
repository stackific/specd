# CSS architecture

How styles are organized in this project, what each layer does, and how new styles get added.

---

## TL;DR

- **BeerCSS** ships from `public/vendor/beer.min.css` as its own `<link>`.
- **Everything else** is bundled by Vite/LightningCSS into ONE custom CSS file served as a second `<link>`.
- Styles are organized into **4 layers**: foundation, layout chrome, components, pages.
- The foundation lives in `src/styles/`. Components and pages co-locate their `.css` next to their `.astro`.
- Class names follow a **namespaced flat convention** so the global bundle never collides with itself.

Two `<link rel="stylesheet">` tags ship on every page. That's it.

---

## The big picture

```
Browser loads ──┬── /vendor/beer.min.css        (vendored, separate cache)
                │
                └── /_astro/style.<hash>.css    (single bundled custom CSS,
                                                 contains everything else)
```

Everything our codebase writes ends up in the second file. BeerCSS is kept out of it on purpose so updating BeerCSS doesn't bust the cache for our styles, and updating our styles doesn't bust BeerCSS's cache.

---

## The four layers

CSS is organized into four logical layers. The cascade order matters for the foundation layer (later layers override earlier ones); component and page layers don't override each other because every class is namespaced.

```
┌─────────────────────────────────────────────────────────┐
│  LAYER 0:  BeerCSS (vendor)                             │
│            public/vendor/beer.min.css                   │
│            Material Design 3 components, color tokens,  │
│            grid, dialogs, fields, chips, articles.      │
│            Loaded as a separate <link>.                 │
├─────────────────────────────────────────────────────────┤
│  LAYER 1:  Foundation                                   │
│            src/styles/                                  │
│            ├── index.css       ← entrypoint (@imports)  │
│            ├── tokens.css      ← :root custom props     │
│            ├── base.css        ← element-level rules    │
│            ├── utilities.css   ← rolled atomic helpers  │
│            └── layout.css      ← page chrome globals    │
├─────────────────────────────────────────────────────────┤
│  LAYER 2:  Components                                   │
│            src/components/Showcase.css                  │
│            src/components/frames/Frame*.css             │
│            src/components/home/Card*.css                │
│            One .css file per .astro component.          │
│            Imported by the component's frontmatter.     │
├─────────────────────────────────────────────────────────┤
│  LAYER 3:  Pages                                        │
│            src/pages/index.css                          │
│            One .css file per page that needs styles.    │
│            Imported by the page's frontmatter.          │
└─────────────────────────────────────────────────────────┘
```

### Layer 0 — BeerCSS (vendor)

**Where:** `public/vendor/beer.min.css` (+ Material Symbols `.woff2` files)
**Loaded as:** a hardcoded `<link rel="stylesheet" href="/vendor/beer.min.css">` in `Layout.astro`'s `<head>`.
**Why separate:** so BeerCSS updates don't invalidate our bundle's cache, and so our LightningCSS pipeline doesn't have to re-process 88KB of vendor CSS on every build.

BeerCSS provides everything Material Design 3 expects: the surface system, color tokens (`--primary`, `--secondary`, `--surface-container`, etc.), the responsive grid (`s12 m6 l4`), components (`button`, `field`, `chip`, `article`, `dialog`, `nav`), spacing helpers, dark mode wiring (`body.dark`), and the Material Symbols font face.

When you write a frame or page, **reach for BeerCSS classes first** for layout, color, typography weights, and components. Only fall back to custom CSS for visual flourishes BeerCSS can't express.

To update BeerCSS, see `public/vendor/README.md`.

### Layer 1 — Foundation (`src/styles/`)

The foundation is a small set of files that establish design tokens, element defaults, and reusable atomic helpers we roll ourselves to extend BeerCSS.

```
src/styles/
├── index.css       Entrypoint. Imports the other files in cascade order.
│                   This is the file Layout.astro imports.
│
├── reset.css       Modern CSS reset, trimmed to coexist with BeerCSS.
│                   Box-sizing, margin reset, media defaults,
│                   reduced-motion accessibility, etc.
│
├── media.css       @custom-media definitions (--mobile / --tablet /
│                   --desktop / --motion-ok / --dark / --hover / …).
│                   Resolved at build time by LightningCSS.
│
├── tokens.css      :root CSS custom properties only. No selectors.
│                   Sizes, spaces, typography, ratios, containers,
│                   custom color palettes (--c-*), and the master
│                   --scale dial.
│
├── base.css        Element-level rules: body, h1/h2/h3, .visually-hidden.
│                   Targets HTML elements directly. No class atoms.
│
├── utilities.css   GENERATED — atomic helper classes.
│                   Source of truth: scripts/gen-utilities.mjs.
│                   Run `pnpm gen:utilities` to regenerate after a
│                   token-scale change.
│
└── layout.css      Page chrome globals owned by Layout.astro.
                    Currently: light/dark logo swap.
```

**Cascade order inside the foundation** (enforced by `@import` order in `index.css`):

1. Vendor fonts (`@fontsource-variable/geist`)
2. `reset.css`
3. `media.css`
4. `tokens.css`
5. `base.css`
6. `utilities.css`
7. `layout.css`

This order matters because `base.css` rules use the tokens defined in `tokens.css`, `utilities.css` rules use both, and `layout.css` rules use everything below.

### Layer 2 — Components

Every reusable component that has visual styles ships its own co-located `.css` file.

```
src/components/
├── Footer.astro        ⟶ Footer.css         (no .css if BeerCSS is enough)
├── MyThing.astro       ⟶ MyThing.css        (.c-mything, .mt-*)
└── nested/
    └── Other.astro     ⟶ Other.css          (.c-other, .ot-*)
```

Each `.astro` file imports its `.css` from the frontmatter:

```astro
---
import "./MyThing.css";
---
```

Vite picks up the import, hands it to LightningCSS, and the rules end up in the single bundled output. You never `@import` component CSS from `index.css` — that's what the component import does for you. The cascade order between components is non-deterministic, which is fine because **every component class is namespaced** (see naming convention below).

### Layer 3 — Pages

Pages that need page-specific styles ship a co-located `.css` file the same way:

```
src/pages/
├── index.astro     ⟶ index.css   (.home-hero, …)
├── about.astro                   (no .css — BeerCSS does everything it needs)
└── services.astro  ⟶ services.css
```

A page that has no `.css` next to it is a deliberate signal that BeerCSS plus the framework utilities cover all its visual needs. Don't create an empty `.css` file just for symmetry.

---

## Naming convention

Every selector that lives in our bundled CSS is global (because Astro's auto-scoping is bypassed). The convention prevents collisions across the dozens of `.css` files that get merged.

### Tokens

CSS custom properties only. Naming: `--<scale>-<step>` in kebab-case. Step numbers always go up = bigger / heavier / further. Step `0` is the baseline. Negative steps put the sign on the step body, keeping the scale-to-step separator dash, so step −1 of `space` is `--space--1`.

```css
--scale            /* master density dial — every rem-based size is calc(* var(--scale)) */
--size-3           /* 1rem * --scale */
--size-px-3        /* 16px (NOT scaled — pixel-snapped) */
--size-fluid-2     /* clamp(1rem, 2vw, 1.5rem) */
--space-3          /* alias of --size-3 for margin/padding/gap */
--space--1         /* negative step — calc(-0.25rem * --scale) */
--text-3           /* font-size: 1.5rem */
--lh-3             /* line-height: 1.5 */
--ls--1            /* letter-spacing: -0.05em */
--fw-5             /* font-weight: 500 */
--ratio-16-9       /* aspect-ratio: 16 / 9 */
--container-md     /* max-width: 768px */
--c-primary-40     /* tonal palette tone — generated from a brand seed */
--c-neutral-90     /* …same shape for secondary, tertiary, neutral, neutral-variant, error */
```

The full set lives in `src/styles/tokens.css`, organized by scale.

### Utilities (atomic helpers)

`utilities.css` is **generated** from `scripts/gen-utilities.mjs`. Don't hand-edit it. To change which utilities ship, edit the generator and run `pnpm gen:utilities`.

#### Spacing utilities are Tailwind-style for margin and padding

Single-letter direction shortcut, no dash between verb and direction:

| Class | CSS |
|---|---|
| `m-3` / `p-3` | `margin: var(--space-3)` / `padding: …` |
| `mt-3` / `pt-3` | `margin-top` / `padding-top` |
| `mr-3` / `pr-3` | `margin-right` / `padding-right` |
| `mb-3` / `pb-3` | `margin-bottom` / `padding-bottom` |
| `ml-3` / `pl-3` | `margin-left` / `padding-left` |
| `mx-3` / `px-3` | `margin-inline` / `padding-inline` |
| `my-3` / `py-3` | `margin-block` / `padding-block` |
| `m--1` | negative margin step −1 (= −0.25rem) |

Other utilities keep the dash separator: `gap-x-3`, `gap-y-3`, `text-3`, `lh-3`, `ls-2`, `fw-5`, `round-4`, `ratio-16-9`, `container-md`.

#### Responsive variants

Mobile-first, min-width, with prefixes that match BeerCSS's `s`/`m`/`l` mental model:

| Prefix | When | Underlying media |
|---|---|---|
| (none) | always | — |
| `m:` | tablet+ | `@media (--tablet)` ≥601px |
| `l:` | desktop+ | `@media (--desktop)` ≥993px |

The colon in the class name is a literal `:` character. CSS selectors must escape it with `\:` (this is plain CSS, popularized by Tailwind). Markup stays clean — only the selector is escaped.

```html
<div class="p-3 m:p-5 l:p-7 text-3 m:text-4 l:text-5">…</div>
```

```css
.p-3      { padding: var(--space-3); }
@media (width >= 601px) { .m\:p-5 { padding: var(--space-5); } }
@media (width >= 993px) { .l\:p-7 { padding: var(--space-7); } }
```

Apply utilities directly in markup alongside BeerCSS classes:

```html
<button class="button primary p-3 mt-2">Save</button>
```

#### Custom @media

Use the `@custom-media` names from `media.css` inside any `.css` file:

```css
.my-component { padding: var(--space-3); }
@media (--desktop) { .my-component { padding: var(--space-7); } }
@media (--motion-ok) { .my-component { transition: transform var(--space-2) ease; } }
```

LightningCSS resolves them at build time.

### Color tokens

The `--c-*` tonal palette is **separate** from BeerCSS's role tokens. BeerCSS owns `--primary`, `--surface`, `--on-primary`, etc. — leave those alone. Our framework provides raw tones for the cases BeerCSS's roles don't cover (illustrations, gradients, custom surfaces):

```
--c-primary-0  -5  -10  -15  -20  -25  -30  -35  -40  -50  -60  -70  -80  -90  -95  -98  -99  -100
--c-secondary-…
--c-tertiary-…
--c-neutral-…
--c-neutral-variant-…
--c-error-…
```

Generated from a brand seed via `@material/material-color-utilities`. The current project's seed lives in the COLOR section of `src/styles/tokens.css` (search for `Generated from seed`). To regenerate after a brand change: `pnpm gen:colors '#hex'` and paste the output into the COLOR section of `tokens.css`.

### Component classes

Two-tier prefix: a 2–3 letter area prefix on the top-level container, then a 2-letter prefix derived from the component name on every internal element.

```
.c-event           ← container prefix
.ev-title          ← internal prefix (every child of .c-event)
.ev-status
.ev-photo
```

Container prefix conventions:

| Convention | Use for | Example |
|---|---|---|
| `c-<name>` | a self-contained card-like component | `.c-event`, `.c-chat`, `.c-pricing` |
| `gfx-<name>` | a decorative / graphic block | `.gfx-hero`, `.gfx-cycle` |
| `<2-letter>-` (no `c-`) | a shared / structural component | `.nv-bar`, `.tb-row` |

Internal prefix rules:

- Pick a unique 2-letter prefix per component derived from its name (`Event` → `ev-`, `Chat` → `ch-`, `Trust` → `tr-`).
- Use it on EVERY internal element of the component, even ones that look like they couldn't collide.
- Grep `src/components/**/*.css` before claiming a new prefix to make sure no other component already uses it.
- If a component gets two related families of internals (e.g. structural + visual effects), it can use two prefixes — but document them in the component's `.css` header comment.

### Page classes

2–4 letter page prefix matching the page name.

```css
.home-hero          /* in src/pages/index.css */
```

### State modifiers

Separate flat class, not BEM `__` or `--`.

```css
.is-active
.is-hidden
.is-disabled
```

JavaScript that toggles state classes on elements styled by our own bundle should use the `.is-*` pattern. (BeerCSS's own runtime classes — `.active`, `.dark`, field/dialog/chip states — are bare and are kept alive by the safelist in the PurgeCSS integration.)

### Why this convention

Once we left Astro's auto-scoping behind, **every selector became global**. Without the convention:

- `.row` in one component would override `.row` in another
- `.card` would collide with BeerCSS's `.card`
- `.button` would fight with BeerCSS's button

With the convention:

- `.c-event .ev-status` only ever matches what's inside the `.c-event` container
- No two components share the same internal prefix
- A new component is "drop in markup, write CSS that starts with `.c-yourname`, never collides with anything"

---

## How to add new styles

### Add a new design token

Edit `src/styles/tokens.css`. Append your new custom property under `:root`.

### Add a new utility class or extend a step

Edit `scripts/gen-utilities.mjs` (the source of truth — `utilities.css` is generated). Then run:

```sh
pnpm gen:utilities
```

If the new utility consumes a token, also add the token to `src/styles/tokens.css` first.

### Regenerate the color palette from a new seed

```sh
pnpm gen:colors '#NEWHEX'
```

Pipe or paste the printed output into the `COLOR` section of `src/styles/tokens.css` (manual paste — the rest of `tokens.css` is hand-organized).

### Add styles to an existing component

Edit the component's co-located `.css` file. Use the component's existing prefix.

### Add a new component

1. Create `src/components/MyThing.astro`
2. Create `src/components/MyThing.css` next to it
3. In `MyThing.astro` frontmatter: `import "./MyThing.css";`
4. Pick a unique 2-letter prefix for the component's internals (e.g. `mt-`)
5. Use `.<unique-name>` for the container and `.<prefix>-<element>` for internals

### Add styles to an existing page

Same pattern: co-located `.css`, frontmatter import, page prefix on every selector.

---

## Build pipeline

```
src/styles/index.css ───────┐
src/components/**/*.css ────┼──► Vite (LightningCSS transformer) ──► single bundle ──► PurgeCSS ──► dist/_astro/style.<hash>.css
src/pages/**/*.css ─────────┘    • resolves @custom-media                                                                  │
                                 • minifies                                                                                ▼
                                                                                                          <link href="/_astro/style.<hash>.css">

public/vendor/beer.min.css ──── (copied verbatim) ──► dist/vendor/beer.min.css ──── <link href="/vendor/beer.min.css">
```

Configuration in `astro.config.mjs`:

```js
build: {
  inlineStylesheets: "never",      // emit external <link>, never inline <style>
},
vite: {
  css: {
    transformer: "lightningcss",   // resolves @custom-media at build time
    lightningcss: {
      drafts: { customMedia: true },
    },
  },
  build: {
    cssCodeSplit: false,           // ONE bundled CSS file (not per-page chunks)
    cssMinify: "esbuild",          // see comment in astro.config.mjs for why not lightningcss
  },
}
```

LightningCSS is the CSS *transformer* (not just the minifier) because we depend on `@custom-media` resolution. The minifier is esbuild — when the Cloudflare adapter triggers an SSR build, Vite passes its SSR build target (`es2024`) to LightningCSS minify, which throws "Unsupported target es2024". esbuild minify avoids that path. The full explanation is in the comment in `astro.config.mjs`.

---

## What about purging unused CSS?

We purge **only our own bundled CSS** (`dist/_astro/style.<hash>.css`), never BeerCSS.

A small Astro integration in `astro.config.mjs` (`purgeOwnCss`) hooks `astro:build:done`, runs PurgeCSS against our bundle using every emitted `dist/**/*.html` as content, and rewrites the file in place. BeerCSS at `dist/vendor/beer.min.css` is never opened — its runtime-added classes (`active`, `dark`, field/dialog/chip states added by `ui()`) would be stripped if scanned, and we don't want that.

A custom extractor (`/[\w-/:]+(?<!:)/g`) lets PurgeCSS recognize class names that contain a literal `:`, so the responsive variants (`m:p-3`, `l:text-4`) survive purge.

The integration carries a small safelist for runtime-toggled classes that appear in our own bundle (e.g. `is-*`, `active`, `dark`, `hidden`, plus greedy keeps for `field*`, `chip*`, `dialog*`, `nav*`, `snackbar*`). If you add JS that toggles a class on an element from our own CSS at runtime, add that class to the safelist in `astro.config.mjs`.

---

## Common gotchas

| Gotcha | Why | Fix |
|---|---|---|
| `<style is:raw>` doesn't make CSS global | `is:raw` only skips Astro template processing; it does NOT bypass selector scoping. We learned this the hard way. | Use `is:global` if you really want a global block inside an `.astro` file — but **prefer extracting to a `.css` file** instead. |
| Importing `"beercss"` re-bundles BeerCSS | The package's index.js does `import "./dist/cdn/beer.min.css"` as a side effect. | Import from the JS file path directly: `import { ui } from "beercss/dist/cdn/beer.min.js";` |
| New class doesn't apply | It's probably outside the namespace and getting overridden, or it's defined in a `.css` file that nobody imports. | Check that the component/page actually has `import "./X.css";` in its frontmatter and that the class follows the prefix convention. |
| Two frames with the same internal prefix would collide | The convention prevents this only if you follow it. | Pick a 2-letter prefix that no other frame uses. Grep `src/components/frames/*.css` before deciding. |

---

## Files at a glance

```
public/vendor/
├── beer.min.css                            ← Layer 0 (vendor)
├── material-symbols-rounded.woff2
└── README.md                               ← how to update BeerCSS

src/styles/
├── README.md                               ← this file
├── index.css                               ← Layer 1 entrypoint
├── reset.css
├── media.css
├── tokens.css
├── base.css
├── utilities.css                           ← generated, do not hand-edit
└── layout.css

src/layouts/
└── Layout.astro                            ← imports src/styles/index.css

src/components/
└── <YourComponent>.astro + <YourComponent>.css     ← Layer 2 (co-located)

src/pages/
└── <page>.astro + <page>.css                       ← Layer 3 (co-located)
```
