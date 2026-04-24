# CSS Conventions

All project-level styles live in `ui/src/app.scss`. [BeerCSS](https://www.beercss.com/) (Material Design 3) provides typography, grid, components, and a spacing scale. Custom styles add only what BeerCSS does not provide — directional margin/padding utilities, a content container, logo swap, and screen-reader helpers.

Vite compiles SCSS via its built-in Sass support. `sass` is a dev dependency; no Vite plugin is required. Importing `./app.scss` from `main.js` triggers compilation. `<style lang="scss">` blocks in Svelte components are handled by `vitePreprocess()` in `svelte.config.js`.

This document has two parts:

- **[Part 1 — Building UI](#part-1--building-ui)** teaches you to lay out pages, align things, apply spacing, and toggle dark mode *using the existing system*. Reach for Part 2 only when Part 1 can't do what you need.
- **[Part 2 — Extending the system](#part-2--extending-the-system)** tells you how to add new utilities, variables, or overrides when a genuine gap exists. New additions are rare; BeerCSS + the directional utilities cover almost every real need.

## Dev loop

```sh
task ui:dev      # Vite dev server with HMR on :5173
task ui:build    # Production build to ui/dist/
task ui:lint     # Biome (skips .scss silently — see "Pitfalls")
task qa          # Vite (5173) + Go (8000) together, fresh QA project
```

SCSS edits hot-reload without a full refresh. Svelte component styles HMR too.

---

# Part 1 — Building UI

## Decision tree

Ask the questions in order. Stop at the first "yes."

1. **Does BeerCSS already do it?** Use the BeerCSS class. Grid (`grid`, `s12 l6`), alignment (`top-align`, `center-align`), spacing (`padding`, `small-margin`, `no-padding`), typography (`large-text`, `bold`), and components (`button`, `chip`, `field`, `article`) all ship out of the box.
2. **Is it directional spacing on a scale we already have?** Use a utility class (`mb-s`, `px-l`, `m:mt-m`, …). See [Spacing](#spacing) below.
3. **Is it scoped to one Svelte component?** Put it in that component's `<style lang="scss">` block. Svelte auto-scopes selectors.
4. **Is it genuinely missing?** Head to [Part 2](#part-2--extending-the-system).

## Layout

### Page skeleton

`ui/src/layouts/Layout.svelte` owns the nav rail, mobile top bar, footer, and wraps `<main>`. Pages render via a `<slot>`. Don't re-implement the rail — use the layout.

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

BeerCSS `<nav class="left">` / `<nav class="right">` become sidebars. Width is set by `--rail-width` (default 4rem). To put a grid *inside* a nav, wrap it in a `<div class="grid">` — don't make the `<nav>` itself the grid container; BeerCSS sets `display: flex` on navs.

## Spacing

Every directional margin/padding is already a class. Format: `{bp}:{m|p}{dir}-{size}`.

| Part | Values |
|---|---|
| `{m\|p}` | `m` margin, `p` padding |
| `{dir}` | `t` top, `b` bottom, `l` left/start, `r` right/end, `x` inline (l+r), `y` block (t+b) |
| `{size}` | `t` tiny 0.25rem, `s` small 0.5rem, `m` medium 1rem, `l` large 1.5rem |
| `{bp}` | `s:` base, `m:` ≥601px, `l:` ≥993px. Unprefixed == `s:`. |

```html
<h3 class="mb-s">Title</h3>                  <!-- 8px bottom -->
<nav class="px-m py-s">Links</nav>           <!-- 16px inline, 8px block -->
<div class="mb-t m:mb-s l:mb-m">…</div>      <!-- scales 4 → 8 → 16 px -->
<section class="l:px-l">…</section>          <!-- only ≥993px -->
```

The colon in `m:mb-s` is literal HTML. CSS escapes it as `.m\:mb-s`. Unprefixed and `s:` prefixes are equivalent (emitted in one rule).

For *uniform* (non-directional) spacing, BeerCSS's own `padding` / `margin` / `no-padding` / `no-margin` work directly.

## Dark mode

Theme state lives in `localStorage` under `specd-theme`, falls back to `prefers-color-scheme` on first load. Toggling sets `body.dark` / `body.light`. Logic is in `ui/src/lib/theme.js`.

Use BeerCSS color tokens in templates — they recolor automatically. For project-specific dark variants (e.g. swapping a logo), target `body.dark .your-class`:

```html
<img class="logo-light" src="/logo.svg">
<img class="logo-dark" src="/logo-dark.svg">
```

Don't use `@media (prefers-color-scheme: dark)` — it ignores the user's explicit toggle.

## Svelte scoped styles

When a style is genuinely component-local:

```svelte
<style lang="scss">
  .card-banner {
    border-radius: var(--radius, 0.5rem);
  }
</style>
```

Svelte scopes `.card-banner` to the component. Still prefer BeerCSS classes and utility classes in markup — scoped blocks are for the rare case where no existing class expresses intent.

## Accessibility helpers

- **`.sr-only`** — visually hidden, readable by screen readers. Use on labels that are implied visually but still need assistive text.
- **Semantic HTML** — `<article>`, `<nav>`, `<main>`, `<header>`, `<footer>`, proper heading hierarchy (`<h1>` once per page), `aria-label` / `role` where a visual cue isn't enough.

## Cheat sheet

```html
<main class="container py-l">
  <h1 class="mb-m">Dashboard</h1>

  <div class="grid">
    <article class="s12 m6 l4 padding primary-container">
      <h2 class="mb-s">Card</h2>
      <p>BeerCSS `padding` + our `mb-s`.</p>
    </article>
  </div>

  <section class="mt-l">
    <h2 class="mb-s">Section</h2>
    <p>Responsive spacing scales with viewport.</p>
  </section>
</main>
```

---

# Part 2 — Extending the system

**Check first**: can Part 1 solve the problem? If yes, stop. Adding classes is easy; maintaining scope discipline isn't.

## File layout

Everything project-level lives in `ui/src/app.scss`, in this order:

```
@use imports            (sass:list, sass:map)
  ↓
maps                    ($breakpoints, $size-keys, $dirs)
  ↓
mixins                  (space, no-space, spacing-utilities)
  ↓
:root custom props      (+ responsive overrides)
  ↓
global rules            (body, container, nav rail, grid, logo)
  ↓
scoped overrides        (footer, …)
  ↓
utility-class generator (@each over $breakpoints)
  ↓
helpers                 (.sr-only)
```

Sass does not hoist. Maps and mixins must be declared **before** any rule that uses them. Keep the order above.

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

## Custom classes

- **`.container`** — centered, max-width, responsive inline padding.
- **`.logo-light` / `.logo-dark`** — visibility swap keyed on `body.dark`.
- **`.sr-only`** — visually hidden, screen-reader accessible.
- **Directional spacing utilities** — generated from the maps below.

## The spacing system, internals

Three maps + three mixins drive the utility classes *and* any ad-hoc spacing you need in SCSS.

### Maps

```scss
$breakpoints: (
  s: null,       // no media query
  m: 601px,
  l: 993px,
);

$size-keys: (t, s, m, l);   // each maps 1:1 to --space-{key}

$dirs: (
  mt: margin-block-start,
  mb: margin-block-end,
  ml: margin-inline-start,
  mr: margin-inline-end,
  mx: margin-inline,
  my: margin-block,
  pt: padding-block-start,
  pb: padding-block-end,
  pl: padding-inline-start,
  pr: padding-inline-end,
  px: padding-inline,
  py: padding-block,
);
```

### Mixins

- **`@include space($dir, $size)`** — emits one spacing declaration (same logical property + `var(--space-*)` + `!important` a utility class would). Use for ad-hoc selectors.

  ```scss
  footer nav { @include space(mt, m); }  // margin-block-start: var(--space-m) !important
  ```

- **`@include no-space($sides...)`** — zeros margin **and** padding. No args = all sides (bare `padding: 0; margin: 0`). Pass CSS side words (`top`, `left`, …) to zero specific sides. Uses **physical** properties on purpose, to exactly mirror BeerCSS's `padding-left` bullet indent.

  ```scss
  footer ul { @include no-space; }          // padding: 0; margin: 0;
  footer li { @include no-space(left); }    // padding-left: 0; margin-left: 0;
  ```

- **`@include spacing-utilities($bp)`** — internal: generates all `.d-sz` / `.bp\:d-sz` rules for one breakpoint. Called once per breakpoint by the loop at the bottom of `app.scss`.

## Adding things

### A new CSS variable

1. Declare on `:root` in `app.scss`.
2. Use `var(--name)` in rules — never the literal.
3. If responsive, add to the existing `:root` media blocks (don't scatter new `@media` rules).
4. Update the variables table above.
5. If AI tooling or Go code needs to know, mirror in `AGENTS.md` § Frontend.

### A new spacing size

1. Add `--space-xl: 2rem` on `:root`.
2. Add `xl` to `$size-keys`.
3. Every `{dir}-xl` / `{bp}:{dir}-xl` class now exists. `@include space($dir, xl)` works too.
4. Update the size table in [Part 1 — Spacing](#spacing).

### A new spacing direction

Add an entry to `$dirs`. Key = class prefix, value = logical CSS property (e.g. `margin-block-start`). Use logical properties, not physical.

### A new breakpoint

Add to `$breakpoints` with `min-width`. The key is also the class prefix (`xl:px-l`).

### A one-off rule

When no existing class expresses intent *and* it's too broad for component-scoped styles, add a rule to `app.scss`. Keep the bar high: the file is small by design. Every new hand-written rule is maintenance cost.

## Overriding BeerCSS

**Scope overrides to the element they're overriding.** This is the single most important rule in Part 2. BeerCSS targets element selectors (`nav`, `ul`, `li`, etc.); a naked override like `nav { … }` in `app.scss` will leak into every component that uses `<nav>`, including the nav rail and any future component you haven't thought of yet.

**Do** keep overrides nested under a containing selector:

```scss
// Footer-specific list reset — only affects <ul>/<li> inside <footer>.
footer ul { @include no-space; list-style: none !important; }
footer li { @include no-space(left); }
footer li::before { content: none !important; }

// Footer-specific nav behaviour — only the <nav> that lives in <footer>.
footer nav { display: block !important; @include space(mt, m); }
```

**Don't** publish your override globally:

```scss
// BAD — kills bullets in every <ul> in the app.
ul { list-style: none !important; }

// BAD — forces every <nav> to be a block container.
nav { display: block !important; }
```

If a rule genuinely must be project-wide (e.g. forcing `align-items: start` on every grid), note it in a comment and prefer targeting a narrower selector where possible. The existing global `.grid { align-items: start; }` is the exception, not the template — new rules should default to scoped.

### Idioms

- **Lists**: `@include no-space` or `no-space(side)` to null margin + padding. Keep `list-style: none !important` as an explicit inline declaration — it's not spacing.
- **`<nav>`**: BeerCSS sets `display: flex`. To put a grid inside, either wrap children in `<div class="grid">` or override the nav to `display: block !important` inside a scoping selector, then nest `<div class="grid">`.
- **Grid alignment**: we force `align-items: start` globally. For center alignment, opt in with `center-align` in markup.

Any override should target the narrowest selector that works. Use `!important` only to beat BeerCSS specificity, with a short comment when the reason isn't obvious.

## Sass specifics

- `@use "sass:list"`, `@use "sass:map"` — place with the existing `@use` lines at the top. Required for `list.length()`, `map.get()`, `map.keys()`.
- **No deprecated globals.** `length(...)` → `list.length(...)`. `map-get(...)` → `map.get(...)`. Same for `nth`, `keys`, etc.
- **Mixins and variables aren't hoisted.** Define at the top of the file.
- **Physical vs logical properties.** Logical by default. Physical only where exactly mirroring a BeerCSS rule that uses physical (see `no-space`).

## Pitfalls

- **Biome doesn't parse SCSS yet.** `.scss` files are silently skipped by `biome check` (status on Biome's matrix: ⌛ parsing/formatting, 🚫 linting). No lint safety net — be disciplined with 2-space indent and naming.
- **Mixin ordering.** Sass doesn't hoist. Define maps + mixins first.
- **Hardcoded values are a smell.** If you're typing `1rem`, `#1c4bea`, or `0.5rem`, there's a variable (or a place one should live).
- **Don't copy from `ui-poc` verbatim.** It's a reference for BeerCSS patterns, not a source of truth.
- **`!important` on custom properties.** We use `--font: "Geist Variable", … !important;` to win the cascade against BeerCSS's own `--font` declaration. This is an intentional exception — most variables shouldn't need `!important`.
- **Global `.grid` override.** Pre-existing. Don't treat it as the norm; scope new overrides.

## Debugging

- **"My class isn't applying."** BeerCSS rules on elements (`nav`, `footer ul`) are specific and often need `!important` to override. Check escape: in CSS, `s:mb-s` is `.s\:mb-s`.
- **"SCSS compile failed."** Vite surfaces Dart Sass errors with file + line. Most common: using a mixin/var before it's defined; missing `@use "sass:list"` / `@use "sass:map"`; deprecated global function.
- **"Output CSS is big."** BeerCSS + Material Symbols fonts dominate the bundle. Our custom styles are under 10 kB. Check `ui/dist/assets/index-*.css` byte count.
- **"HMR stopped."** Restart Vite. The Svelte plugin occasionally wedges when files are renamed.
- **"Rule applies in browser but not in headless."** `prefers-color-scheme` differs. Set `localStorage.specd-theme` explicitly in tests.

## Rules (strict)

- **Use BeerCSS classes natively** for grid, alignment, spacing, typography, and components before writing custom CSS.
- **Every tunable value is a CSS variable** on `:root`. Never hardcode sizes/spacing/colors in rule bodies.
- **Logical properties by default.** Physical only when mirroring a BeerCSS rule that uses physical.
- **Scope BeerCSS overrides.** Nest them under a containing selector (e.g. `footer ul`, not `ul`). Global overrides leak into unrelated components.
- **Ad-hoc spacing goes through `@include space(...)`.** Don't hand-write `margin-block-end: var(--space-m) !important`.
- **No copying from `ui-poc` verbatim.** Study patterns; re-derive through our scale.
- **Keep `app.scss` small.** Every new hand-written rule is a maintenance cost. Justify it against the decision tree.
