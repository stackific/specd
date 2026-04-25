# CSS Conventions

All project-level styles live in `static/css/src/app.css`. [BeerCSS](https://www.beercss.com/) (Material Design 3) provides typography, grid, components, and a spacing scale. Custom styles add only what BeerCSS does not provide — directional margin/padding utilities, gap utilities, display/overflow/flex helpers, width constraints, a content container, logo swap, navigation drawer override, and screen-reader helpers.

The CSS build pipeline uses Lightning CSS (bundle + minify) → PurgeCSS (strip unused classes by scanning templates) → concatenation with vendored BeerCSS. See [CSS Build](#css-build) in Part 2.

This document has two parts:

- **[Part 1 — Building UI](#part-1--building-ui)** teaches you to lay out pages, align things, apply spacing, and toggle dark mode *using the existing system*. Reach for Part 2 only when Part 1 can't do what you need.
- **[Part 2 — Extending the system](#part-2--extending-the-system)** tells you how to add new utilities, variables, or overrides when a genuine gap exists. New additions are rare; BeerCSS + the utility classes cover almost every real need.

## Dev loop

```sh
task qa            # Fresh QA project, build CSS, start Air on :8000 with --dev
task css:build     # Build CSS (bundle + purge + minify)
task css:lint      # Run Stylelint on CSS source
task css:lint:fix  # Run Stylelint with auto-fix
```

In dev mode (`--dev`), Go templates are re-parsed on every request. CSS changes require re-running `task css:build` (or the QA task rebuilds on restart). Live reload via SSE auto-refreshes the browser when Air rebuilds.

---

# Part 1 — Building UI

## Decision tree

Ask the questions in order. Stop at the first "yes."

1. **Does BeerCSS already do it?** Use the BeerCSS class. Grid (`grid`, `s12 l6`), alignment (`top-align`, `center-align`), spacing (`padding`, `small-margin`, `no-padding`), typography (`large-text`, `bold`), and components (`button`, `chip`, `field`, `article`) all ship out of the box.
2. **Is it directional spacing on a scale we already have?** Use a utility class (`mb-s`, `px-l`, `m:mt-m`, …). See [Spacing](#spacing) below.
3. **Is it genuinely missing?** Head to [Part 2](#part-2--extending-the-system).

## Layout

### Page skeleton

`templates/layouts/base.html` defines the page shell. `templates/partials/nav.html` owns the nav rail, mobile top bar, mobile drawer, and exports `nav-links` / `nav-links-mobile` sub-templates. `templates/partials/footer.html` owns the footer. Pages define `{{define "content"}}...{{end}}` to fill the content block. Don't re-implement the shell — add pages in `templates/pages/`.

### Navigation

**Desktop** (≥993px): `<nav class="left l surface-container">` — BeerCSS nav rail. The hamburger toggles `max` class via `toggleSidebar()` in `static/js/app.js`, switching between collapsed rail (icon above label, 4rem) and expanded drawer (icon beside label, 12.75rem). State persists in `localStorage` under `specd-sidebar`.

**Mobile** (<993px): `<nav class="top s m">` top bar with hamburger. The hamburger opens `<dialog id="mobile-menu" class="left no-padding">` which contains `<nav class="left max surface-container">` — an expanded drawer that slides in from the left with a BeerCSS overlay.

Nav order: Tasks, Specs, KB, Search → `.max` spacer → Settings, Docs, theme toggle.

All nav links use `hx-get`, `hx-target="#main-content"`, `hx-swap="innerHTML"`, `hx-push-url="true"` for SPA-like navigation via htmx.

### Grid

```html
<div class="grid">
  <div class="s12 m6 l4">Card 1</div>
  <div class="s12 m6 l4">Card 2</div>
  <div class="s12 m6 l4">Card 3</div>
</div>
```

Breakpoints: `s` < 601px, `m` ≥ 601px, `l` ≥ 993px. Column counts sum to 12 at each breakpoint. `.grid` is forced `align-items: start` so cards top-align by default — opt back into center alignment with `center-align`.

### Container

```html
<section class="container">
  <h1>Content</h1>
</section>
```

Centers content with max-width `--container-max` (default 1200px) and responsive inline padding (`1rem` / `1.5rem` / `2rem` at s/m/l).

### Nav rail

BeerCSS `<nav class="left">` / `<nav class="right">` become sidebars. Width is set by `--rail-width` (default 4rem). Adding `max` class expands to 12.75rem with horizontal icon+label layout. To put a grid *inside* a nav, wrap it in a `<div class="grid">` — don't make the `<nav>` itself the grid container; BeerCSS sets `display: flex` on navs.

## Spacing

Every directional margin/padding is already a class. Format: `{bp}:{m|p}{dir}-{size}`.

| Part | Values |
|---|---|
| `{m\|p}` | `m` margin, `p` padding |
| `{dir}` | `t` top, `b` bottom, `l` left/start, `r` right/end, `x` inline (l+r), `y` block (t+b) |
| `{size}` | `t` tiny 0.25rem, `s` small 0.5rem, `m` medium 1rem, `l` large 1.5rem, `xl` extra-large 2rem |
| `{bp}` | `s:` base, `m:` ≥601px, `l:` ≥993px. Unprefixed == `s:`. |

```html
<h3 class="mb-s">Title</h3>                  <!-- 8px bottom -->
<nav class="px-m py-s">Links</nav>           <!-- 16px inline, 8px block -->
<div class="mb-t m:mb-s l:mb-m">…</div>      <!-- scales 4 → 8 → 16 px -->
<section class="l:px-l">…</section>          <!-- only ≥993px -->
```

The colon in `m:mb-s` is literal HTML. CSS escapes it as `.m\:mb-s`. Unprefixed and `s:` prefixes are equivalent (emitted in one rule).

For *uniform* (non-directional) spacing, BeerCSS's own `padding` / `margin` / `no-padding` / `no-margin` work directly.

## Gap

Control `gap` on flex/grid containers. Same scale and responsive prefixes as spacing.

```html
<div class="grid gap-m">…</div>              <!-- 1rem gap -->
<div class="flex gap-s l:gap-m">…</div>      <!-- 0.5rem, 1rem at ≥993px -->
```

BeerCSS `horizontal` hardcodes `gap: 1rem`. Use gap utilities when you need a different value or responsive scaling.

## Display

Toggle display mode. BeerCSS handles responsive show/hide via `s`/`m`/`l` classes; these are for non-responsive cases.

| Class | Effect |
|---|---|
| `hidden` | `display: none` |
| `block` | `display: block` |
| `flex` | `display: flex` |
| `inline-flex` | `display: inline-flex` |

## Overflow

```html
<div class="overflow-hidden">…</div>     <!-- clip overflow -->
<div class="overflow-auto">…</div>       <!-- scroll when needed -->
<div class="overflow-x-auto">…</div>     <!-- horizontal scroll only -->
<div class="overflow-y-auto">…</div>     <!-- vertical scroll only -->
```

## Flex children

Fine-grained flex child control. BeerCSS has `.max` (flex: max-content); these complement it.

| Class | Effect |
|---|---|
| `grow` | `flex-grow: 1` |
| `no-grow` | `flex-grow: 0` |
| `shrink-0` | `flex-shrink: 0` |

## Width constraints

Max-width utilities using CSS variables. BeerCSS has `small-width`/`medium-width`/`large-width` for fixed widths; these set `max-inline-size` for fluid-but-bounded layouts.

| Class | Max width |
|---|---|
| `max-w-dialog` | `min(360px, 92vw)` — mobile-safe dialog |
| `max-w-sm` | `24rem` (384px) |
| `max-w-md` | `36rem` (576px) |
| `max-w-lg` | `48rem` (768px) |
| `max-w-full` | `100%` |

```html
<dialog class="padding max-w-dialog">…</dialog>
<article class="max-w-md">Narrow reading column</article>
```

## Dark mode

Theme state lives in `localStorage` under `specd-theme`, falls back to `prefers-color-scheme` on first load. Toggling sets `body.dark` / `body.light`. Logic is in `static/js/app.js` (`initTheme`, `toggleTheme`, `syncThemeIcons`).

Use BeerCSS color tokens in templates — they recolor automatically. For project-specific dark variants (e.g. swapping a logo), target `body.dark .your-class`:

```html
<img class="logo-light" src="/logo.svg">
<img class="logo-dark" src="/logo-dark.svg">
```

Don't use `@media (prefers-color-scheme: dark)` — it ignores the user's explicit toggle.

## Accessibility helpers

- **`.sr-only`** — visually hidden, readable by screen readers. Use on labels that are implied visually but still need assistive text.
- **Semantic HTML** — `<article>`, `<nav>`, `<main>`, `<header>`, `<footer>`, proper heading hierarchy (`<h1>` once per page), `aria-label` / `role` where a visual cue isn't enough.

## Cheat sheet

```html
<main class="container py-l">
  <h1 class="mb-m">Dashboard</h1>

  <div class="grid gap-m">
    <article class="s12 m6 l4 padding primary-container">
      <h2 class="mb-s">Card</h2>
      <p>BeerCSS `padding` + our `mb-s`.</p>
    </article>
  </div>

  <div class="flex gap-s">
    <button class="grow">Save</button>
    <button class="shrink-0">Cancel</button>
  </div>

  <dialog class="padding max-w-dialog">
    <p>Mobile-safe dialog.</p>
  </dialog>

  <div class="overflow-x-auto">
    <table>…</table>
  </div>

  <section class="mt-xl">
    <h2 class="mb-s">Section</h2>
    <p>Extra-large spacing above.</p>
  </section>
</main>
```

---

# Part 2 — Extending the system

**Check first**: can Part 1 solve the problem? If yes, stop. Adding classes is easy; maintaining scope discipline isn't.

## File layout

Everything project-level lives in `static/css/src/app.css`, in this order:

```
font-face declarations   (Geist Variable)
  ↓
:root custom props       (+ responsive overrides)
  ↓
global rules             (body, container, nav rail, nav drawer, grid, logo)
  ↓
scoped overrides         (footer, …)
  ↓
helpers                  (.sr-only)
  ↓
static utilities         (display, overflow, flex, width constraints)
  ↓
spacing utility classes  (mt-t through py-xl, responsive)
  ↓
gap utility classes      (gap-t through gap-xl, responsive)
```

## CSS custom properties

Defined on `:root`. Every tunable value is a variable.

| Variable | Default | Responsive | Purpose |
|---|---|---|---|
| `--font` | `"Geist Variable", sans-serif` | No | Global font (overrides BeerCSS) |
| `--body-font-size` | `1rem` | No | Main content text size |
| `--letter-spacing` | `-0.01em` | No | Global letter spacing |
| `--container-max` | `1200px` | No | Max width for `.container` |
| `--container-padding` | `1rem` | Yes (1.5rem @ ≥601px, 2rem @ ≥993px) | `.container` inline padding |
| `--rail-width` | `4rem` | No | Nav rail width |
| `--space-t` | `0.25rem` | No | Tiny spacing |
| `--space-s` | `0.5rem` | No | Small spacing |
| `--space-m` | `1rem` | No | Medium spacing |
| `--space-l` | `1.5rem` | No | Large spacing |
| `--space-xl` | `2rem` | No | Extra-large spacing |
| `--max-w-dialog` | `min(360px, 92vw)` | No | Mobile-safe dialog width |
| `--max-w-sm` | `24rem` | No | Small max-width constraint |
| `--max-w-md` | `36rem` | No | Medium max-width constraint |
| `--max-w-lg` | `48rem` | No | Large max-width constraint |
| `--max-w-full` | `100%` | No | Full max-width constraint |

## Custom classes

- **`.container`** — centered, max-width, responsive inline padding.
- **`.logo-light` / `.logo-dark`** — visibility swap keyed on `body.dark`.
- **`.sr-only`** — visually hidden, screen-reader accessible.
- **Directional spacing utilities** — `{m|p}{dir}-{size}` with responsive `{bp}:` prefixes.
- **Gap utilities** (`gap-{t|s|m|l|xl}`, responsive) — same scale and prefixes.
- **Display toggles** — `.hidden`, `.block`, `.flex`, `.inline-flex`.
- **Overflow** — `.overflow-hidden`, `.overflow-auto`, `.overflow-x-auto`, `.overflow-y-auto`.
- **Flex children** — `.grow`, `.no-grow`, `.shrink-0`.
- **Width constraints** — `.max-w-dialog`, `.max-w-sm`, `.max-w-md`, `.max-w-lg`, `.max-w-full`.

## The spacing system

Utility classes are pre-generated in `app.css`. The pattern for each direction × size × breakpoint:

```css
/* Base (small) breakpoint — unprefixed and s: prefixed */
.mt-s, .s\:mt-s { margin-block-start: var(--space-s) !important; }

/* Medium breakpoint (601px+) */
@media (width >= 601px) {
  .m\:mt-s { margin-block-start: var(--space-s) !important; }
}

/* Large breakpoint (993px+) */
@media (width >= 993px) {
  .l\:mt-s { margin-block-start: var(--space-s) !important; }
}
```

**Sizes**: `t` (0.25rem), `s` (0.5rem), `m` (1rem), `l` (1.5rem), `xl` (2rem).

**Directions**: `mt` `mb` `ml` `mr` `mx` `my` `pt` `pb` `pl` `pr` `px` `py` — each maps to a logical CSS property (e.g. `mt` → `margin-block-start`, `px` → `padding-inline`).

Gap utilities follow the same pattern: `gap-{size}` with `s:`, `m:`, `l:` responsive prefixes.

## Adding things

### A new CSS variable

1. Declare on `:root` in `app.css`.
2. Use `var(--name)` in rules — never the literal.
3. If responsive, add to the existing `:root` media blocks (don't scatter new `@media` rules).
4. Update the variables table above.
5. If AI tooling or Go code needs to know, mirror in `AGENTS.md` § Frontend.

### A new spacing size

1. Add `--space-{key}: value` on `:root`.
2. Add all utility classes for the new key following the existing pattern (12 directions × 3 breakpoints).
3. Add `gap-{key}` classes in all 3 breakpoints.
4. Update the size table in [Part 1 — Spacing](#spacing).

### A new utility class

Add it to the appropriate section in `app.css`. Use `!important` to match the convention of other utilities. Update the tables in this document.

### A one-off rule

When no existing class expresses intent, add a rule to `app.css`. Keep the bar high: the file is small by design. Every new hand-written rule is maintenance cost.

## CSS Build

The build pipeline lives in `static/scripts/build-css.js`:

1. **Lightning CSS**: Bundle + minify `css/src/app.css` → `css/dist/custom.css`
2. **PurgeCSS**: Strip unused classes from `custom.css` by scanning `templates/**/*.html` (configured in `static/purgecss.config.cjs`)
3. **Font path rewrite**: Rewrite BeerCSS font URLs to absolute paths
4. **Concatenate**: Merge vendored BeerCSS + purged custom CSS → `css/dist/app.css`

```sh
task css:build     # runs: cd static && pnpm run build:css
```

**Important**: Classes created dynamically in JavaScript won't be found by PurgeCSS. Add them to the `safelist` in `static/purgecss.config.cjs`.

## Overriding BeerCSS

**Scope overrides to the element they're overriding.** This is the single most important rule in Part 2. BeerCSS targets element selectors (`nav`, `ul`, `li`, etc.); a naked override like `nav { … }` in `app.css` will leak into every component that uses `<nav>`.

**Do** keep overrides under a containing selector:

```css
/* Footer-specific list reset — only affects <ul>/<li> inside <footer>. */
footer ul { list-style: none !important; padding: 0 !important; margin: 0 !important; }
footer li { padding-left: 0 !important; margin-left: 0 !important; }
footer li::before { content: none !important; }
```

**Don't** publish your override globally:

```css
/* BAD — kills bullets in every <ul> in the app. */
ul { list-style: none !important; }
```

### Idioms

- **Lists**: Zero margin+padding with `!important` to beat BeerCSS specificity. Keep `list-style: none !important` explicit.
- **`<nav>`**: BeerCSS sets `display: flex`. To put a grid inside, wrap children in `<div class="grid">` or override to `display: block !important` inside a scoping selector.
- **Grid alignment**: we force `align-items: start` globally. For center alignment, opt in with `center-align` in markup.

Any override should target the narrowest selector that works. Use `!important` only to beat BeerCSS specificity, with a short comment when the reason isn't obvious.

## Pitfalls

- **Hardcoded values are a smell.** If you're typing `1rem`, `#1c4bea`, or `0.5rem`, there's a variable (or a place one should live).
- **`!important` on custom properties.** We use `--font: "Geist Variable", … !important;` to win the cascade against BeerCSS's own `--font` declaration. This is an intentional exception — most variables shouldn't need `!important`.
- **Global `.grid` override.** Pre-existing. Don't treat it as the norm; scope new overrides.
- **PurgeCSS strips JS-only classes.** If a CSS class is only referenced in JavaScript (not in templates), PurgeCSS will strip it. Add to the safelist in `static/purgecss.config.cjs`.

## Debugging

- **"My class isn't applying."** BeerCSS rules on elements (`nav`, `footer ul`) are specific and often need `!important` to override. Check escape: in CSS, `s:mb-s` is `.s\:mb-s`.
- **"CSS build failed."** Lightning CSS surfaces errors with file + line. Check `static/scripts/build-css.js` output.
- **"Output CSS is big."** BeerCSS + Material Symbols fonts dominate the bundle. Our custom styles are small. Check `static/css/dist/app.css` byte count.
- **"Rule applies in browser but not in headless."** `prefers-color-scheme` differs. Set `localStorage.specd-theme` explicitly in tests.

## Rules (strict)

- **Use BeerCSS classes natively** for grid, alignment, spacing, typography, and components before writing custom CSS.
- **Every tunable value is a CSS variable** on `:root`. Never hardcode sizes/spacing/colors in rule bodies.
- **Logical properties by default.** Physical only when mirroring a BeerCSS rule that uses physical.
- **Scope BeerCSS overrides.** Nest them under a containing selector (e.g. `footer ul`, not `ul`). Global overrides leak into unrelated components.
- **Keep `app.css` small.** Every new hand-written rule is a maintenance cost. Justify it against the decision tree.
